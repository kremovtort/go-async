package async

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// An asynchronous action spawned by NewAsync. Asynchronous actions are executed in a separate goroutine, and operations are provided for waiting for asynchronous actions to complete and obtaining their results (see e.g. Wait).
type Async[T any] interface {
	// Wait returns a channel that will receive the result of the async operation.
	Wait() <-chan T

	// WaitPanic returns a channel that will receive the panic of the async operation.
	WaitPanic() <-chan any

	// WaitCancel returns a channel that will receive a struct{} when the async operation is cancelled.
	WaitCancel() <-chan struct{}

	// Poll returns the result of the async operation and a boolean indicating if the operation is done.
	Poll() (T, bool)

	// PollState returns the state of the async operation and the result if it is done.
	PollState() AsyncPollResult[T]

	// PollDone returns true if the async operation is successfully completed, e.g. not cancelled or panicked and result is available.
	PollDone() bool

	// PollCancelled returns true if the async operation is cancelled.
	PollCancelled() bool

	// PollPanicked returns true if the async operation panicked.
	PollPanicked() bool

	// PollPending returns true if the async operation is still in progress.
	PollPending() bool
}

type AsyncPollResult[T any] struct {
	State  AsyncState
	Result T
	Panic  any
}

type AsyncState int32

const (
	// Pending means the async operation is still in progress.
	Pending AsyncState = iota
	// Done means the async operation is successfully completed and result is available.
	Done
	// Cancelled means the async operation is cancelled.
	Cancelled
	// Panicked means the async operation panicked.
	Panicked
)

type async[T any] struct {
	ctx    context.Context
	state  int32
	mu     sync.RWMutex
	result atomic.Value
	panic  atomic.Value

	waitingForResult     []chan T
	waitingForResultChan chan chan T

	waitingForCancel     []chan struct{}
	waitingForCancelChan chan chan struct{}

	waitingForPanic     []chan any
	waitingForPanicChan chan chan any

	asyncCompleted chan struct{}
}

func NewAsync[T any](ctx context.Context, fn func(ctx context.Context) T) Async[T] {
	a := &async[T]{
		ctx:    ctx,
		state:  int32(Pending),
		mu:     sync.RWMutex{},
		result: atomic.Value{},
		panic:  atomic.Value{},

		waitingForResultChan: make(chan chan T, 1),
		waitingForResult:     make([]chan T, 0, 8),

		waitingForCancelChan: make(chan chan struct{}, 1),
		waitingForCancel:     make([]chan struct{}, 0, 8),

		waitingForPanicChan: make(chan chan any, 1),
		waitingForPanic:     make([]chan any, 0, 8),

		asyncCompleted: make(chan struct{}, 1),
	}
	atomic.StoreInt32(&a.state, int32(Pending))

	go a.processWaitingRequests()

	go a.runAsync(fn)

	return a
}

func (a *async[T]) runAsync(fn func(ctx context.Context) T) {
	resultChannel := make(chan T, 1)
	panicChannel := make(chan any, 1)

	go func(a *async[T], resultChannel chan T, panicChannel chan any) {
		defer func() {
			if r := recover(); r != nil {
				panicChannel <- r
			}
		}()
		result := fn(a.ctx)
		resultChannel <- result
	}(a, resultChannel, panicChannel)

	select {
	case <-a.ctx.Done():
		atomic.StoreInt32(&a.state, int32(Cancelled))
	case result := <-resultChannel:
		a.result.Store(result)
		atomic.StoreInt32(&a.state, int32(Done))
	case panic := <-panicChannel:
		a.panic.Store(panic)
		atomic.StoreInt32(&a.state, int32(Panicked))
	}

	a.asyncCompleted <- struct{}{}
}

func (a *async[T]) processWaitingRequests() {
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
	state := AsyncState(atomic.LoadInt32(&a.state))
	switch state {
	case Done:
		result := a.result.Load().(T)
		for _, ch := range a.waitingForResult {
			ch <- result
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
			ch <- panicValue
		}
	default:
		panic(fmt.Sprintf("unexpected state %d", state))
	}
}

func (a *async[T]) Wait() <-chan T {
	for {
		state := AsyncState(atomic.LoadInt32(&a.state))
		switch state {
		case Pending:
			locked := a.mu.TryRLock()
			if !locked {
				continue
			} else {
				defer a.mu.RUnlock()
				ch := make(chan T, 1)
				a.waitingForResultChan <- ch
				return ch
			}
		case Done:
			ch := make(chan T, 1)
			ch <- a.result.Load().(T)
			return ch
		default:
			ch := make(chan T)
			close(ch)
			return ch
		}
	}
}

func (a *async[T]) WaitPanic() <-chan any {
	for {
		state := AsyncState(atomic.LoadInt32(&a.state))
		switch state {
		case Pending:
			locked := a.mu.TryRLock()
			if !locked {
				continue
			} else {
				defer a.mu.RUnlock()
				ch := make(chan any, 1)
				a.waitingForPanicChan <- ch
				return ch
			}
		case Panicked:
			ch := make(chan any, 1)
			ch <- a.panic.Load()
			return ch
		default:
			ch := make(chan any)
			close(ch)
			return ch
		}
	}
}

func (a *async[T]) WaitCancel() <-chan struct{} {
	for {
		state := AsyncState(atomic.LoadInt32(&a.state))
		switch state {
		case Pending:
			locked := a.mu.TryRLock()
			if !locked {
				continue
			} else {
				defer a.mu.RUnlock()
				ch := make(chan struct{}, 1)
				a.waitingForCancelChan <- ch
				return ch
			}
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
}

func (a *async[T]) PollState() AsyncPollResult[T] {
	state := AsyncState(atomic.LoadInt32(&a.state))
	switch state {
	case Pending:
		return AsyncPollResult[T]{State: Pending}
	case Done:
		return AsyncPollResult[T]{State: Done, Result: a.result.Load().(T)}
	case Cancelled:
		return AsyncPollResult[T]{State: Cancelled}
	case Panicked:
		return AsyncPollResult[T]{State: Panicked, Panic: a.panic.Load()}
	default:
		panic(fmt.Sprintf("unexpected state %d", state))
	}
}

func (a *async[T]) Poll() (T, bool) {
	state := AsyncState(atomic.LoadInt32(&a.state))
	if state == Done {
		return a.result.Load().(T), true
	}
	var zero T
	return zero, false
}

func (a *async[T]) PollDone() bool {
	return atomic.LoadInt32(&a.state) == int32(Done)
}

func (a *async[T]) PollCancelled() bool {
	return atomic.LoadInt32(&a.state) == int32(Cancelled)
}

func (a *async[T]) PollPanicked() bool {
	return atomic.LoadInt32(&a.state) == int32(Panicked)
}

func (a *async[T]) PollPending() bool {
	return atomic.LoadInt32(&a.state) == int32(Pending)
}
