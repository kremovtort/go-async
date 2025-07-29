package async

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
)

// An asynchronous action spawned by NewAsync. Asynchronous actions are executed in a separate goroutine, and operations are provided for waiting for asynchronous actions to complete and obtaining their results (see e.g. Wait).
type Async[T any] interface {
	// Wait returns a channel that will receive the result of the async operation.
	Wait() <-chan T

	// WaitPanic returns a channel that will receive the panic of the async operation.
	WaitPanic() <-chan Panic

	// WaitCancel returns a channel that will receive a struct{} when the async operation is cancelled.
	WaitCancel() <-chan struct{}

	// Poll returns the result of the async operation and a boolean indicating if the operation is done.
	Poll() (T, bool)

	// PollState returns the state of the async operation and the result if it is done.
	PollState() State[T]

	// PollDone returns true if the async operation is successfully completed, e.g. not cancelled or panicked and result is available.
	PollDone() bool

	// PollCancelled returns true if the async operation is cancelled.
	PollCancelled() bool

	// PollPanicked returns true if the async operation panicked.
	PollPanicked() bool

	// PollPending returns true if the async operation is still in progress.
	PollPending() bool
}

type State[T any] struct {
	Status Status
	Result T
	Panic  any
}

type Panic struct {
	Panic      any
	StackTrace []byte
}

type Status int32

const (
	// Pending means the async operation is still in progress.
	Pending Status = iota
	// Done means the async operation is successfully completed and result is available.
	Done
	// Cancelled means the async operation is cancelled.
	Cancelled
	// Panicked means the async operation panicked.
	Panicked
)

type async[T any] struct {
	ctx    context.Context
	status int32
	mu     sync.RWMutex
	result atomic.Pointer[T]
	panic  atomic.Pointer[Panic]

	waitingForResult     []chan T
	waitingForResultChan chan chan T

	waitingForCancel     []chan struct{}
	waitingForCancelChan chan chan struct{}

	waitingForPanic     []chan Panic
	waitingForPanicChan chan chan Panic

	asyncCompleted chan struct{}
}

func NewAsync[T any](ctx context.Context, fn func(ctx context.Context) T) Async[T] {
	a := &async[T]{
		ctx:    ctx,
		status: int32(Pending),
		mu:     sync.RWMutex{},
		result: atomic.Pointer[T]{},
		panic:  atomic.Pointer[Panic]{},

		waitingForResultChan: make(chan chan T, 1),
		waitingForResult:     make([]chan T, 0, 8),

		waitingForCancelChan: make(chan chan struct{}, 1),
		waitingForCancel:     make([]chan struct{}, 0, 8),

		waitingForPanicChan: make(chan chan Panic, 1),
		waitingForPanic:     make([]chan Panic, 0, 8),

		asyncCompleted: make(chan struct{}, 1),
	}
	atomic.StoreInt32(&a.status, int32(Pending))

	go a.runAsync(fn)

	go a.processWaitingRequests()

	return a
}

func (a *async[T]) runAsync(fn func(ctx context.Context) T) {
	resultChannel := make(chan T, 1)
	panicChannel := make(chan Panic, 1)

	go func(a *async[T], resultChannel chan T, panicChannel chan Panic) {
		defer func() {
			if r := recover(); r != nil {
				panicChannel <- Panic{
					Panic:      r,
					StackTrace: debug.Stack(),
				}
			}
		}()
		result := fn(a.ctx)
		resultChannel <- result
	}(a, resultChannel, panicChannel)

	select {
	case <-a.ctx.Done():
		atomic.StoreInt32(&a.status, int32(Cancelled))
	case result := <-resultChannel:
		a.result.Store(&result)
		atomic.StoreInt32(&a.status, int32(Done))
	case panic := <-panicChannel:
		a.panic.Store(&panic)
		atomic.StoreInt32(&a.status, int32(Panicked))
	}

	a.asyncCompleted <- struct{}{}
}

func (a *async[T]) processWaitingRequests() {
	defer func() {
		close(a.waitingForResultChan)
		close(a.waitingForCancelChan)
		close(a.waitingForPanicChan)
		close(a.asyncCompleted)
	}()

loop:
	for {
		select {
		case ch, _ := <-a.waitingForResultChan:
			a.waitingForResult = append(a.waitingForResult, ch)
		case ch, _ := <-a.waitingForCancelChan:
			a.waitingForCancel = append(a.waitingForCancel, ch)
		case ch, _ := <-a.waitingForPanicChan:
			a.waitingForPanic = append(a.waitingForPanic, ch)
		case <-a.asyncCompleted:
			a.mu.Lock()
			defer a.mu.Unlock()
			for {
				select {
				case ch, _ := <-a.waitingForResultChan:
					a.waitingForResult = append(a.waitingForResult, ch)
				case ch, _ := <-a.waitingForCancelChan:
					a.waitingForCancel = append(a.waitingForCancel, ch)
				case ch, _ := <-a.waitingForPanicChan:
					a.waitingForPanic = append(a.waitingForPanic, ch)
				default:
					break loop
				}
			}
		}
	}

	a.sendResult()
}

func (a *async[T]) sendResult() {
	state := Status(atomic.LoadInt32(&a.status))
	switch state {
	case Done:
		result := a.result.Load()
		for _, ch := range a.waitingForResult {
			ch <- *result
		}
		for _, ch := range a.waitingForCancel {
			close(ch)
		}
		for _, ch := range a.waitingForPanic {
			close(ch)
		}
	case Cancelled:
		for _, ch := range a.waitingForResult {
			close(ch)
		}
		for _, ch := range a.waitingForCancel {
			ch <- struct{}{}
		}
		for _, ch := range a.waitingForPanic {
			close(ch)
		}
	case Panicked:
		panicValue := a.panic.Load()
		for _, ch := range a.waitingForResult {
			close(ch)
		}
		for _, ch := range a.waitingForCancel {
			close(ch)
		}
		for _, ch := range a.waitingForPanic {
			ch <- *panicValue
		}
	default:
		panic(fmt.Sprintf("unexpected state %d", state))
	}
}

func (a *async[T]) Wait() <-chan T {
again:
	state := Status(atomic.LoadInt32(&a.status))
	switch state {
	case Pending:
		a.mu.RLock()
		defer a.mu.RUnlock()
		if Status(atomic.LoadInt32(&a.status)) != Pending {
			goto again
		}
		ch := make(chan T, 1)
		a.waitingForResultChan <- ch
		return ch
	case Done:
		ch := make(chan T, 1)
		ch <- *a.result.Load()
		return ch
	default:
		ch := make(chan T)
		close(ch)
		return ch
	}
}

func (a *async[T]) WaitPanic() <-chan Panic {
again:
	state := Status(atomic.LoadInt32(&a.status))
	switch state {
	case Pending:
		a.mu.RLock()
		defer a.mu.RUnlock()
		if Status(atomic.LoadInt32(&a.status)) != Pending {
			goto again
		}
		ch := make(chan Panic, 1)
		a.waitingForPanicChan <- ch
		return ch
	case Panicked:
		ch := make(chan Panic, 1)
		ch <- *a.panic.Load()
		return ch
	default:
		ch := make(chan Panic)
		close(ch)
		return ch
	}
}

func (a *async[T]) WaitCancel() <-chan struct{} {
again:
	state := Status(atomic.LoadInt32(&a.status))
	switch state {
	case Pending:
		a.mu.RLock()
		defer a.mu.RUnlock()
		if Status(atomic.LoadInt32(&a.status)) != Pending {
			goto again
		}
		ch := make(chan struct{}, 1)
		a.waitingForCancelChan <- ch
		return ch
	case Cancelled:
		ch := make(chan struct{}, 1)
		ch <- struct{}{}
		return ch
	default:
		ch := make(chan struct{})
		close(ch)
		return ch
	}
}

func (a *async[T]) PollState() State[T] {
	state := Status(atomic.LoadInt32(&a.status))
	switch state {
	case Pending:
		return State[T]{Status: Pending}
	case Done:
		return State[T]{Status: Done, Result: *a.result.Load()}
	case Cancelled:
		return State[T]{Status: Cancelled}
	case Panicked:
		return State[T]{Status: Panicked, Panic: a.panic.Load()}
	default:
		panic(fmt.Sprintf("unexpected state %d", state))
	}
}

func (a *async[T]) Poll() (T, bool) {
	state := Status(atomic.LoadInt32(&a.status))
	if state == Done {
		return *a.result.Load(), true
	}
	var zero T
	return zero, false
}

func (a *async[T]) PollDone() bool {
	return atomic.LoadInt32(&a.status) == int32(Done)
}

func (a *async[T]) PollCancelled() bool {
	return atomic.LoadInt32(&a.status) == int32(Cancelled)
}

func (a *async[T]) PollPanicked() bool {
	return atomic.LoadInt32(&a.status) == int32(Panicked)
}

func (a *async[T]) PollPending() bool {
	return atomic.LoadInt32(&a.status) == int32(Pending)
}
