package async

import (
	"context"
	"testing"
	"time"
)

func TestAsync(t *testing.T) {
	ctx := context.Background()

	// Test basic async computation
	a := NewAsync(ctx, func(ctx context.Context) string {
		return "result"
	})

	result, err := a.Wait()
	if err != nil {
		t.Errorf("Wait returned error: %v", err)
	}
	if result != "result" {
		t.Errorf("Wait returned %v, expected %v", result, "result")
	}
}

func TestAsyncWithContext(t *testing.T) {
	// Test async computation with context cancellation
	ctx, cancel := context.WithCancel(context.Background())

	a := NewAsync(ctx, func(ctx context.Context) string {
		// Simulate some work
		time.Sleep(100 * time.Millisecond)
		return "result"
	})

	// Cancel the context before the computation completes
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := a.Wait()
	if err == nil {
		t.Error("Wait should return error when context is cancelled")
	}
}

func TestPoll(t *testing.T) {
	ctx := context.Background()

	a := NewAsync(ctx, func(ctx context.Context) string {
		time.Sleep(100 * time.Millisecond)
		return "result"
	})

	// Poll before the computation completes
	result, done, err := a.Poll()
	if done {
		t.Error("Poll should return done=false before computation completes")
	}
	if err != nil {
		t.Errorf("Poll returned error: %v", err)
	}

	// Wait for the computation to complete
	time.Sleep(150 * time.Millisecond)

	// Poll after the computation completes
	result, done, err = a.Poll()
	if !done {
		t.Error("Poll should return done=true after computation completes")
	}
	if err != nil {
		t.Errorf("Poll returned error: %v", err)
	}
	if result != "result" {
		t.Errorf("Poll returned %v, expected %v", result, "result")
	}
}

func TestCancel(t *testing.T) {
	ctx := context.Background()

	a := NewAsync(ctx, func(ctx context.Context) string {
		// Simulate some work
		time.Sleep(100 * time.Millisecond)
		return "result"
	})

	// Cancel the computation
	err := a.Cancel()
	if err != nil {
		t.Errorf("Cancel returned error: %v", err)
	}

	// Wait should return context cancelled error
	_, err = a.Wait()
	if err == nil {
		t.Error("Wait should return error after cancellation")
	}
}

func TestRace(t *testing.T) {
	ctx := context.Background()

	// Test race with two computations
	result, err := Race(ctx,
		func(ctx context.Context) string {
			time.Sleep(200 * time.Millisecond)
			return "slow"
		},
		func(ctx context.Context) int {
			time.Sleep(100 * time.Millisecond)
			return 42
		},
	)

	if err != nil {
		t.Errorf("Race returned error: %v", err)
	}

	// The int computation should complete first
	if result != 42 {
		t.Errorf("Race returned %v, expected %v", result, 42)
	}
}

func TestConcurrently(t *testing.T) {
	ctx := context.Background()

	// Test concurrently with two computations
	str, num, err := Concurrently(ctx,
		func(ctx context.Context) string {
			time.Sleep(100 * time.Millisecond)
			return "first"
		},
		func(ctx context.Context) int {
			time.Sleep(200 * time.Millisecond)
			return 42
		},
	)

	if err != nil {
		t.Errorf("Concurrently returned error: %v", err)
	}

	if str != "first" {
		t.Errorf("Concurrently returned %v, expected %v", str, "first")
	}

	if num != 42 {
		t.Errorf("Concurrently returned %v, expected %v", num, 42)
	}
}

func TestWaitAny(t *testing.T) {
	ctx := context.Background()

	// Create multiple async computations
	a1 := NewAsync(ctx, func(ctx context.Context) string {
		time.Sleep(300 * time.Millisecond)
		return "slow"
	})

	a2 := NewAsync(ctx, func(ctx context.Context) string {
		time.Sleep(100 * time.Millisecond)
		return "fast"
	})

	a3 := NewAsync(ctx, func(ctx context.Context) string {
		time.Sleep(200 * time.Millisecond)
		return "medium"
	})

	// WaitAny should return the result of the fastest computation
	result, err := WaitAny(a1, a2, a3)

	if err != nil {
		t.Errorf("WaitAny returned error: %v", err)
	}

	if result != "fast" {
		t.Errorf("WaitAny returned %v, expected %v", result, "fast")
	}
}

func TestWaitAll(t *testing.T) {
	ctx := context.Background()

	// Create multiple async computations
	a1 := NewAsync(ctx, func(ctx context.Context) string {
		time.Sleep(100 * time.Millisecond)
		return "first"
	})

	a2 := NewAsync(ctx, func(ctx context.Context) string {
		time.Sleep(200 * time.Millisecond)
		return "second"
	})

	a3 := NewAsync(ctx, func(ctx context.Context) string {
		time.Sleep(300 * time.Millisecond)
		return "third"
	})

	// WaitAll should return all results
	results, err := WaitAll(a1, a2, a3)

	if err != nil {
		t.Errorf("WaitAll returned error: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("WaitAll returned %d results, expected 3", len(results))
	}

	expected := []string{"first", "second", "third"}
	for i, result := range results {
		if result != expected[i] {
			t.Errorf("WaitAll[%d] returned %v, expected %v", i, result, expected[i])
		}
	}
}

func TestWaitBoth(t *testing.T) {
	ctx := context.Background()

	// Create two async computations
	a1 := NewAsync(ctx, func(ctx context.Context) string {
		time.Sleep(100 * time.Millisecond)
		return "first"
	})

	a2 := NewAsync(ctx, func(ctx context.Context) int {
		time.Sleep(200 * time.Millisecond)
		return 42
	})

	// WaitBoth should return both results
	str, num, err := WaitBoth(a1, a2)

	if err != nil {
		t.Errorf("WaitBoth returned error: %v", err)
	}

	if str != "first" {
		t.Errorf("WaitBoth returned %v, expected %v", str, "first")
	}

	if num != 42 {
		t.Errorf("WaitBoth returned %v, expected %v", num, 42)
	}
}

func TestErrorHandling(t *testing.T) {
	ctx := context.Background()

	// Test error handling
	a := NewAsync(ctx, func(ctx context.Context) string {
		panic("test panic")
	})

	_, err := a.Wait()
	if err == nil {
		t.Error("Wait should return error when computation panics")
	}
}

func TestEmptyWaitAny(t *testing.T) {
	// Test WaitAny with no computations
	_, err := WaitAny[string]()
	if err == nil {
		t.Error("WaitAny should return error when no computations are provided")
	}
}
