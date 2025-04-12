package async

import (
	"context"
	"fmt"
	"sync"
)

// Async represents an asynchronous computation that will return a value of type T
type Async[T any] struct {
	ctx          context.Context
	cancel       context.CancelFunc
	result       chan T
	err          chan error
	done         bool
	mu           sync.Mutex
	storedResult T
	storedErr    error
	hasResult    bool
}

// NewAsync runs a function asynchronously and returns an Async handle to the computation
func NewAsync[T any](ctx context.Context, f func(context.Context) T) *Async[T] {
	ctx, cancel := context.WithCancel(ctx)

	a := &Async[T]{
		ctx:    ctx,
		cancel: cancel,
		result: make(chan T, 1),
		err:    make(chan error, 1),
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.err <- fmt.Errorf("panic in async computation: %v", r)
			}
		}()

		result := f(ctx)

		select {
		case a.result <- result:
		case <-ctx.Done():
		}
	}()

	return a
}

// Wait waits for the async computation to complete and returns its result
func (a *Async[T]) Wait() (T, error) {
	a.mu.Lock()
	if a.hasResult {
		a.mu.Unlock()
		return a.storedResult, a.storedErr
	}
	a.mu.Unlock()

	select {
	case result := <-a.result:
		a.mu.Lock()
		a.done = true
		a.hasResult = true
		a.storedResult = result
		a.storedErr = nil
		a.mu.Unlock()
		return result, nil
	case err := <-a.err:
		a.mu.Lock()
		a.done = true
		a.hasResult = true
		var zero T
		a.storedResult = zero
		a.storedErr = err
		a.mu.Unlock()
		return a.storedResult, err
	case <-a.ctx.Done():
		a.mu.Lock()
		a.done = true
		a.hasResult = true
		var zero T
		a.storedResult = zero
		a.storedErr = a.ctx.Err()
		a.mu.Unlock()
		return a.storedResult, a.ctx.Err()
	}
}

// Poll checks if the async computation has completed without blocking
func (a *Async[T]) Poll() (T, bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.hasResult {
		return a.storedResult, true, a.storedErr
	}

	select {
	case result := <-a.result:
		a.done = true
		a.hasResult = true
		a.storedResult = result
		a.storedErr = nil
		return result, true, nil
	case err := <-a.err:
		a.done = true
		a.hasResult = true
		var zero T
		a.storedResult = zero
		a.storedErr = err
		return a.storedResult, true, err
	case <-a.ctx.Done():
		a.done = true
		a.hasResult = true
		var zero T
		a.storedResult = zero
		a.storedErr = a.ctx.Err()
		return a.storedResult, true, a.ctx.Err()
	default:
		var zero T
		return zero, false, nil
	}
}

// Cancel cancels the async computation
func (a *Async[T]) Cancel() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.done {
		return
	}

	a.cancel()
	a.done = true
}

// Either runs two computations in parallel and returns the result of the first one to complete
func Either[T1, T2 any](ctx context.Context, f1 func(context.Context) T1, f2 func(context.Context) T2) (interface{}, error) {
	a1 := NewAsync(ctx, f1)
	a2 := NewAsync(ctx, f2)

	select {
	case result := <-a1.result:
		a2.Cancel()
		return result, nil
	case err := <-a1.err:
		a2.Cancel()
		return nil, err
	case result := <-a2.result:
		a1.Cancel()
		return result, nil
	case err := <-a2.err:
		a1.Cancel()
		return nil, err
	case <-ctx.Done():
		a1.Cancel()
		a2.Cancel()
		return nil, ctx.Err()
	}
}

// Both runs two computations in parallel and waits for both to complete
func Both[T1, T2 any](ctx context.Context, f1 func(context.Context) T1, f2 func(context.Context) T2) (T1, T2, error) {
	a1 := NewAsync(ctx, f1)
	a2 := NewAsync(ctx, f2)

	result1, err1 := a1.Wait()
	result2, err2 := a2.Wait()

	if err1 != nil {
		return result1, result2, err1
	}
	if err2 != nil {
		return result1, result2, err2
	}

	return result1, result2, nil
}

// WaitAny waits for any of the async computations to complete and returns its result
func WaitAny[T any](asyncs ...*Async[T]) (T, error) {
	if len(asyncs) == 0 {
		var zero T
		return zero, fmt.Errorf("no async computations provided")
	}

	type result struct {
		value T
		err   error
	}
	results := make(chan result, len(asyncs))

	for _, a := range asyncs {
		go func(a *Async[T]) {
			value, err := a.Wait()
			results <- result{value, err}
		}(a)
	}

	r := <-results

	for _, a := range asyncs {
		a.Cancel()
	}

	return r.value, r.err
}

// WaitAll waits for all async computations to complete and returns their results
func WaitAll[T any](asyncs ...*Async[T]) ([]T, error) {
	if len(asyncs) == 0 {
		return nil, nil
	}

	results := make([]T, len(asyncs))
	var firstErr error

	for i, a := range asyncs {
		result, err := a.Wait()
		if err != nil && firstErr == nil {
			firstErr = err
		}
		results[i] = result
	}

	return results, firstErr
}

// WaitBoth waits for both async computations to complete and returns their results
func WaitBoth[T1, T2 any](a1 *Async[T1], a2 *Async[T2]) (T1, T2, error) {
	result1, err1 := a1.Wait()
	result2, err2 := a2.Wait()

	if err1 != nil {
		return result1, result2, err1
	}
	if err2 != nil {
		return result1, result2, err2
	}

	return result1, result2, nil
}

// WaitEither waits for either of two async computations to complete and returns its result
func WaitEither[T1, T2 any](a1 *Async[T1], a2 *Async[T2]) (interface{}, error) {
	type result struct {
		value interface{}
		err   error
		index int
	}
	results := make(chan result, 2)

	go func() {
		value, err := a1.Wait()
		results <- result{value, err, 0}
	}()

	go func() {
		value, err := a2.Wait()
		results <- result{value, err, 1}
	}()

	r := <-results

	if r.err == nil {
		if r.index == 0 {
			a2.Cancel()
		} else {
			a1.Cancel()
		}
		return r.value, nil
	}

	r2 := <-results

	if r2.err == nil {
		if r2.index == 0 {
			a2.Cancel()
		} else {
			a1.Cancel()
		}
		return r2.value, nil
	}

	return nil, r.err
}

// All runs multiple computations in parallel and waits for all to complete
func All[T any](ctx context.Context, fs ...func(context.Context) T) ([]T, error) {
	if len(fs) == 0 {
		return nil, nil
	}

	asyncs := make([]*Async[T], len(fs))
	for i, f := range fs {
		asyncs[i] = NewAsync(ctx, f)
	}

	return WaitAll(asyncs...)
}

// Any runs multiple computations in parallel and returns the result of the first one to complete
func Any[T any](ctx context.Context, fs ...func(context.Context) T) (T, error) {
	if len(fs) == 0 {
		var zero T
		return zero, fmt.Errorf("no computations provided")
	}

	asyncs := make([]*Async[T], len(fs))
	for i, f := range fs {
		asyncs[i] = NewAsync(ctx, f)
	}

	return WaitAny(asyncs...)
}
