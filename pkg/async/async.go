package async

import (
	"context"
	"fmt"
	"sync"
)

// Async represents an asynchronous computation that will return a value of type T
type Async[T any] struct {
	ctx    context.Context
	cancel context.CancelFunc
	result chan T
	err    chan error
	done   bool
	mu     sync.Mutex
}

// NewAsync runs a function asynchronously and returns an Async handle to the computation
func NewAsync[T any](ctx context.Context, f func(context.Context) T) *Async[T] {
	// Create a new context that we can cancel
	ctx, cancel := context.WithCancel(ctx)

	// Create the Async struct
	a := &Async[T]{
		ctx:    ctx,
		cancel: cancel,
		result: make(chan T, 1),
		err:    make(chan error, 1),
	}

	// Start the computation in a goroutine
	go func() {
		defer func() {
			// Recover from any panics
			if r := recover(); r != nil {
				a.err <- fmt.Errorf("panic in async computation: %v", r)
			}
		}()

		// Run the function
		result := f(ctx)

		// Send the result
		select {
		case a.result <- result:
		case <-ctx.Done():
			// Context was cancelled, don't send result
		}
	}()

	return a
}

// Wait waits for the async computation to complete and returns its result
func (a *Async[T]) Wait() (T, error) {
	a.mu.Lock()
	if a.done {
		a.mu.Unlock()
		return a.getResult()
	}
	a.mu.Unlock()

	// Wait for either a result or an error
	select {
	case result := <-a.result:
		a.mu.Lock()
		a.done = true
		a.mu.Unlock()
		return result, nil
	case err := <-a.err:
		a.mu.Lock()
		a.done = true
		a.mu.Unlock()
		var zero T
		return zero, err
	case <-a.ctx.Done():
		a.mu.Lock()
		a.done = true
		a.mu.Unlock()
		var zero T
		return zero, a.ctx.Err()
	}
}

// Poll checks if the async computation is done and returns its result if it is
func (a *Async[T]) Poll() (T, bool, error) {
	a.mu.Lock()
	if a.done {
		a.mu.Unlock()
		result, err := a.getResult()
		return result, true, err
	}
	a.mu.Unlock()

	// Check if there's a result or error available without blocking
	select {
	case result := <-a.result:
		a.mu.Lock()
		a.done = true
		a.mu.Unlock()
		return result, true, nil
	case err := <-a.err:
		a.mu.Lock()
		a.done = true
		a.mu.Unlock()
		var zero T
		return zero, true, err
	default:
		var zero T
		return zero, false, nil
	}
}

// Cancel cancels the async computation
func (a *Async[T]) Cancel() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.done {
		return nil
	}

	a.cancel()
	a.done = true
	return nil
}

// getResult is a helper function to get the result from the channels
func (a *Async[T]) getResult() (T, error) {
	select {
	case result := <-a.result:
		return result, nil
	case err := <-a.err:
		var zero T
		return zero, err
	default:
		var zero T
		return zero, fmt.Errorf("no result available")
	}
}

// Race runs two computations in parallel and returns the result of the first one to complete
func Race[T1, T2 any](ctx context.Context, f1 func(context.Context) T1, f2 func(context.Context) T2) (interface{}, error) {
	// Create two async computations
	a1 := NewAsync(ctx, f1)
	a2 := NewAsync(ctx, f2)

	// Wait for either to complete
	select {
	case result := <-a1.result:
		a2.Cancel() // Cancel the other computation
		return result, nil
	case err := <-a1.err:
		a2.Cancel() // Cancel the other computation
		return nil, err
	case result := <-a2.result:
		a1.Cancel() // Cancel the other computation
		return result, nil
	case err := <-a2.err:
		a1.Cancel() // Cancel the other computation
		return nil, err
	case <-ctx.Done():
		a1.Cancel() // Cancel both computations
		a2.Cancel()
		return nil, ctx.Err()
	}
}

// Concurrently runs two computations in parallel and waits for both to complete
func Concurrently[T1, T2 any](ctx context.Context, f1 func(context.Context) T1, f2 func(context.Context) T2) (T1, T2, error) {
	// Create two async computations
	a1 := NewAsync(ctx, f1)
	a2 := NewAsync(ctx, f2)

	// Wait for both to complete
	result1, err1 := a1.Wait()
	result2, err2 := a2.Wait()

	// Return the first error if any
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

	// Create a channel to receive results
	type result struct {
		value T
		err   error
	}
	results := make(chan result, len(asyncs))

	// Start a goroutine for each async computation
	for _, a := range asyncs {
		go func(a *Async[T]) {
			value, err := a.Wait()
			results <- result{value, err}
		}(a)
	}

	// Wait for the first result
	r := <-results

	// Cancel all other computations
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

	// Wait for all computations to complete
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

	// Return the first error if any
	if err1 != nil {
		return result1, result2, err1
	}
	if err2 != nil {
		return result1, result2, err2
	}

	return result1, result2, nil
}
