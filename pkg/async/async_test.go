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
	_, done, err := a.Poll()
	if done {
		t.Error("Poll should return done=false before computation completes")
	}
	if err != nil {
		t.Errorf("Poll returned error: %v", err)
	}

	// Wait for the computation to complete
	time.Sleep(150 * time.Millisecond)

	// Poll after the computation completes
	result, done, err := a.Poll()
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

	a.Cancel()

	// Wait should return context cancelled error
	_, err := a.Wait()
	if err == nil {
		t.Error("Wait should return error after cancellation")
	}
}

func TestEither(t *testing.T) {
	ctx := context.Background()

	// Test either with two computations
	result, err := Either(ctx,
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
		t.Errorf("Either returned error: %v", err)
	}

	// The int computation should complete first
	if result != 42 {
		t.Errorf("Either returned %v, expected %v", result, 42)
	}
}

func TestBoth(t *testing.T) {
	ctx := context.Background()

	// Test both with two computations
	str, num, err := Both(ctx,
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
		t.Errorf("Both returned error: %v", err)
	}

	if str != "first" {
		t.Errorf("Both returned %v, expected %v", str, "first")
	}

	if num != 42 {
		t.Errorf("Both returned %v, expected %v", num, 42)
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

func TestMultipleWaitCalls(t *testing.T) {
	ctx := context.Background()

	// Create an async computation
	a := NewAsync(ctx, func(ctx context.Context) string {
		time.Sleep(100 * time.Millisecond)
		return "result"
	})

	// First call to Wait
	result1, err1 := a.Wait()
	if err1 != nil {
		t.Errorf("First Wait returned error: %v", err1)
	}
	if result1 != "result" {
		t.Errorf("First Wait returned %v, expected %v", result1, "result")
	}

	// Second call to Wait should return the same result
	result2, err2 := a.Wait()
	if err2 != nil {
		t.Errorf("Second Wait returned error: %v", err2)
	}
	if result2 != "result" {
		t.Errorf("Second Wait returned %v, expected %v", result2, "result")
	}

	// Third call to Wait should also return the same result
	result3, err3 := a.Wait()
	if err3 != nil {
		t.Errorf("Third Wait returned error: %v", err3)
	}
	if result3 != "result" {
		t.Errorf("Third Wait returned %v, expected %v", result3, "result")
	}
}

func TestMultiplePollCalls(t *testing.T) {
	ctx := context.Background()

	// Create an async computation
	a := NewAsync(ctx, func(ctx context.Context) string {
		time.Sleep(100 * time.Millisecond)
		return "result"
	})

	// Wait for the result to be available
	time.Sleep(200 * time.Millisecond)

	// First call to Poll
	result1, done1, err1 := a.Poll()
	if !done1 {
		t.Error("First Poll returned not done")
	}
	if err1 != nil {
		t.Errorf("First Poll returned error: %v", err1)
	}
	if result1 != "result" {
		t.Errorf("First Poll returned %v, expected %v", result1, "result")
	}

	// Second call to Poll should return the same result
	result2, done2, err2 := a.Poll()
	if !done2 {
		t.Error("Second Poll returned not done")
	}
	if err2 != nil {
		t.Errorf("Second Poll returned error: %v", err2)
	}
	if result2 != "result" {
		t.Errorf("Second Poll returned %v, expected %v", result2, "result")
	}

	// Third call to Poll should also return the same result
	result3, done3, err3 := a.Poll()
	if !done3 {
		t.Error("Third Poll returned not done")
	}
	if err3 != nil {
		t.Errorf("Third Poll returned error: %v", err3)
	}
	if result3 != "result" {
		t.Errorf("Third Poll returned %v, expected %v", result3, "result")
	}
}

func TestWaitEither(t *testing.T) {
	ctx := context.Background()

	// Create two async computations with different types
	a1 := NewAsync(ctx, func(ctx context.Context) string {
		time.Sleep(200 * time.Millisecond)
		return "slow string"
	})

	a2 := NewAsync(ctx, func(ctx context.Context) int {
		time.Sleep(100 * time.Millisecond)
		return 42
	})

	// WaitEither should return the result of the faster computation
	result, err := WaitEither(a1, a2)

	if err != nil {
		t.Errorf("WaitEither returned error: %v", err)
	}

	// The int computation should complete first
	if result != 42 {
		t.Errorf("WaitEither returned %v, expected %v", result, 42)
	}

	// Test with error handling
	a3 := NewAsync(ctx, func(ctx context.Context) string {
		panic("test panic")
	})

	a4 := NewAsync(ctx, func(ctx context.Context) int {
		time.Sleep(100 * time.Millisecond)
		return 42
	})

	// WaitEither should return the successful result from a4 even if a3 panics
	result, err = WaitEither(a3, a4)
	if err != nil {
		t.Errorf("WaitEither returned error: %v", err)
	}
	if result != 42 {
		t.Errorf("WaitEither returned %v, expected %v", result, 42)
	}
}

func TestAll(t *testing.T) {
	ctx := context.Background()

	// Test All with multiple computations
	results, err := All(ctx,
		func(ctx context.Context) string {
			time.Sleep(100 * time.Millisecond)
			return "first"
		},
		func(ctx context.Context) string {
			time.Sleep(200 * time.Millisecond)
			return "second"
		},
		func(ctx context.Context) string {
			time.Sleep(300 * time.Millisecond)
			return "third"
		},
	)

	if err != nil {
		t.Errorf("All returned error: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("All returned %d results, expected 3", len(results))
	}

	expected := []string{"first", "second", "third"}
	for i, result := range results {
		if result != expected[i] {
			t.Errorf("All[%d] returned %v, expected %v", i, result, expected[i])
		}
	}
}

func TestAny(t *testing.T) {
	ctx := context.Background()

	// Test Any with multiple computations
	result, err := Any(ctx,
		func(ctx context.Context) string {
			time.Sleep(300 * time.Millisecond)
			return "slow"
		},
		func(ctx context.Context) string {
			time.Sleep(100 * time.Millisecond)
			return "fast"
		},
		func(ctx context.Context) string {
			time.Sleep(200 * time.Millisecond)
			return "medium"
		},
	)

	if err != nil {
		t.Errorf("Any returned error: %v", err)
	}

	if result != "fast" {
		t.Errorf("Any returned %v, expected %v", result, "fast")
	}
}

func TestEither3(t *testing.T) {
	ctx := context.Background()

	// Test Either3 with three computations
	result, err := Either3(ctx,
		func(ctx context.Context) string {
			time.Sleep(300 * time.Millisecond)
			return "slow"
		},
		func(ctx context.Context) int {
			time.Sleep(200 * time.Millisecond)
			return 42
		},
		func(ctx context.Context) float64 {
			time.Sleep(100 * time.Millisecond)
			return 3.14
		},
	)

	if err != nil {
		t.Errorf("Either3 returned error: %v", err)
	}

	// The float64 computation should complete first
	if result != 3.14 {
		t.Errorf("Either3 returned %v, expected %v", result, 3.14)
	}
}

func TestBoth3(t *testing.T) {
	ctx := context.Background()

	// Test Both3 with three computations
	str, num, flt, err := Both3(ctx,
		func(ctx context.Context) string {
			time.Sleep(100 * time.Millisecond)
			return "first"
		},
		func(ctx context.Context) int {
			time.Sleep(200 * time.Millisecond)
			return 42
		},
		func(ctx context.Context) float64 {
			time.Sleep(300 * time.Millisecond)
			return 3.14
		},
	)

	if err != nil {
		t.Errorf("Both3 returned error: %v", err)
	}

	if str != "first" {
		t.Errorf("Both3 returned %v, expected %v", str, "first")
	}

	if num != 42 {
		t.Errorf("Both3 returned %v, expected %v", num, 42)
	}

	if flt != 3.14 {
		t.Errorf("Both3 returned %v, expected %v", flt, 3.14)
	}
}

func TestWaitEither3(t *testing.T) {
	ctx := context.Background()

	// Create three async computations with different types
	a1 := NewAsync(ctx, func(ctx context.Context) string {
		time.Sleep(300 * time.Millisecond)
		return "slow string"
	})

	a2 := NewAsync(ctx, func(ctx context.Context) int {
		time.Sleep(200 * time.Millisecond)
		return 42
	})

	a3 := NewAsync(ctx, func(ctx context.Context) float64 {
		time.Sleep(100 * time.Millisecond)
		return 3.14
	})

	// WaitEither3 should return the result of the faster computation
	result, err := WaitEither3(a1, a2, a3)

	if err != nil {
		t.Errorf("WaitEither3 returned error: %v", err)
	}

	// The float64 computation should complete first
	if result != 3.14 {
		t.Errorf("WaitEither3 returned %v, expected %v", result, 3.14)
	}

	// Test with error handling
	a4 := NewAsync(ctx, func(ctx context.Context) string {
		panic("test panic")
	})

	a5 := NewAsync(ctx, func(ctx context.Context) int {
		time.Sleep(200 * time.Millisecond)
		return 42
	})

	a6 := NewAsync(ctx, func(ctx context.Context) float64 {
		time.Sleep(100 * time.Millisecond)
		return 3.14
	})

	// WaitEither3 should return the successful result even if one computation panics
	result, err = WaitEither3(a4, a5, a6)
	if err != nil {
		t.Errorf("WaitEither3 returned error: %v", err)
	}
	if result != 3.14 {
		t.Errorf("WaitEither3 returned %v, expected %v", result, 3.14)
	}
}

func TestWaitBoth3(t *testing.T) {
	ctx := context.Background()

	// Create three async computations
	a1 := NewAsync(ctx, func(ctx context.Context) string {
		time.Sleep(100 * time.Millisecond)
		return "first"
	})

	a2 := NewAsync(ctx, func(ctx context.Context) int {
		time.Sleep(200 * time.Millisecond)
		return 42
	})

	a3 := NewAsync(ctx, func(ctx context.Context) float64 {
		time.Sleep(300 * time.Millisecond)
		return 3.14
	})

	// WaitBoth3 should return all three results
	str, num, flt, err := WaitBoth3(a1, a2, a3)

	if err != nil {
		t.Errorf("WaitBoth3 returned error: %v", err)
	}

	if str != "first" {
		t.Errorf("WaitBoth3 returned %v, expected %v", str, "first")
	}

	if num != 42 {
		t.Errorf("WaitBoth3 returned %v, expected %v", num, 42)
	}

	if flt != 3.14 {
		t.Errorf("WaitBoth3 returned %v, expected %v", flt, 3.14)
	}
}
