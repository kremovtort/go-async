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

	// State returns the state of the async operation.
	State() AsyncState

	// Done returns true if the async operation is successfully completed, e.g. not cancelled or paniced and result is available.
	Done() bool

	// Cancelled returns true if the async operation is cancelled.
	Cancelled() bool

	// Paniced returns true if the async operation paniced.
	Paniced() bool

	// Pending returns true if the async operation is still in progress.
	Pending() bool
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
	// Paniced means the async operation paniced.
	Paniced
)

type async[T any] struct {
	ctx                context.Context
	state              atomic.Int32
	result             atomic.Value
	panic              atomic.Value
	mu                 sync.RWMutex
	waitResultChannels chan (chan T)
	waitCancelChannels chan (chan struct{})
	waitPanicChannels  chan (chan any)
}

type Result[T any] struct {
	Value T
	Error error
}

type Result2[T1 any, T2 any] struct {
	Value1 T1
	Value2 T2
	Error  error
}

func NewAsync[T any](ctx context.Context, fn func(ctx context.Context) T) Async[T] {
	waitResultChannels := make(chan (chan T), 8)
	waitCancelChannels := make(chan (chan struct{}), 8)
	waitPanicChannels := make(chan (chan any), 8)

	a := &async[T]{
		ctx:                ctx,
		state:              atomic.Int32{},
		result:             atomic.Value{},
		panic:              atomic.Value{},
		mu:                 sync.RWMutex{},
		waitResultChannels: waitResultChannels,
		waitCancelChannels: waitCancelChannels,
		waitPanicChannels:  waitPanicChannels,
	}
	a.state.Store(int32(Pending))

	go func(a *async[T]) {
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
			a.state.Store(int32(Cancelled))
		case result := <-resultChannel:
			a.result.Store(result)
			a.state.Store(int32(Done))
		case panic := <-panicChannel:
			a.panic.Store(panic)
			a.state.Store(int32(Paniced))
		}

		a.sendResultAndCloseChannels()
	}(a)

	return a
}

func (a *async[T]) sendResultAndCloseChannels() {
	a.mu.Lock()
	defer a.mu.Unlock()
	defer close(a.waitResultChannels)
	defer close(a.waitCancelChannels)
	defer close(a.waitPanicChannels)

	state := AsyncState(a.state.Load())
	switch state {
	case Done:
		result := a.result.Load().(T)
		for {
			select {
			case ch, ok := <-a.waitResultChannels:
				if !ok {
					panic("waitResultChannels unexpectedly closed")
				}
				ch <- result
			default:
				goto Done
			}
		}
	case Cancelled:
		for {
			select {
			case ch, ok := <-a.waitCancelChannels:
				if !ok {
					panic("waitCancelChannels unexpectedly closed")
				}
				ch <- struct{}{}
			default:
				goto Done
			}
		}
	case Paniced:
		panicValue := a.panic.Load()
		for {
			select {
			case ch, ok := <-a.waitPanicChannels:
				if !ok {
					panic("waitPanicChannels unexpectedly closed")
				}
				ch <- panicValue
			default:
				goto Done
			}
		}
	default:
		panic(fmt.Sprintf("unexpected state %d", state))
	}
Done:
}

func (a *async[T]) Wait() <-chan T {
	for {
		state := AsyncState(a.state.Load())
		switch state {
		case Pending:
			locked := a.mu.TryRLock()
			if !locked {
				continue
			} else {
				defer a.mu.RUnlock()
				ch := make(chan T, 1)
				a.waitResultChannels <- ch
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
		state := AsyncState(a.state.Load())
		switch state {
		case Pending:
			locked := a.mu.TryRLock()
			if !locked {
				continue
			} else {
				defer a.mu.RUnlock()
				ch := make(chan any, 1)
				a.waitPanicChannels <- ch
				return ch
			}
		case Paniced:
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
		state := AsyncState(a.state.Load())
		switch state {
		case Pending:
			locked := a.mu.TryRLock()
			if !locked {
				continue
			} else {
				defer a.mu.RUnlock()
				ch := make(chan struct{}, 1)
				a.waitCancelChannels <- ch
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
	state := AsyncState(a.state.Load())
	switch state {
	case Pending:
		return AsyncPollResult[T]{State: Pending}
	case Done:
		return AsyncPollResult[T]{State: Done, Result: a.result.Load().(T)}
	case Cancelled:
		return AsyncPollResult[T]{State: Cancelled}
	case Paniced:
		return AsyncPollResult[T]{State: Paniced, Panic: a.panic.Load()}
	default:
		panic(fmt.Sprintf("unexpected state %d", state))
	}
}

func (a *async[T]) Poll() (T, bool) {
	state := AsyncState(a.state.Load())
	if state == Done {
		return a.result.Load().(T), true
	}
	var zero T
	return zero, false
}

func (a *async[T]) State() AsyncState {
	return AsyncState(a.state.Load())
}

func (a *async[T]) Done() bool {
	return a.State() == Done
}

func (a *async[T]) Cancelled() bool {
	return a.State() == Cancelled
}

func (a *async[T]) Paniced() bool {
	return a.State() == Paniced
}

func (a *async[T]) Pending() bool {
	return a.State() == Pending
}
