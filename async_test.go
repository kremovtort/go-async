package async

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewAsync_Success(t *testing.T) {
	ctx := context.Background()
	a := NewAsync(ctx, func(ctx context.Context) string {
		return "success"
	})

	select {
	case result := <-a.Wait():
		if result != "success" {
			t.Errorf("expected 'success', got %v", result)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for result")
	}

	if !a.PollDone() {
		t.Error("expected PollDone to return true")
	}
}

func TestNewAsync_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	a := NewAsync(ctx, func(ctx context.Context) string {
		<-ctx.Done()
		return "cancelled"
	})

	// Cancel the context immediately
	cancel()

	select {
	case <-a.WaitCancel():
		// Expected
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for cancellation")
	}

	if !a.PollCancelled() {
		t.Error("expected PollCancelled to return true")
	}
}

func TestNewAsync_Panic(t *testing.T) {
	ctx := context.Background()
	a := NewAsync(ctx, func(ctx context.Context) string {
		panic("test panic")
	})

	select {
	case panicVal := <-a.WaitPanic():
		if panicVal.Panic != "test panic" {
			t.Errorf("expected 'test panic', got %v", panicVal)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for panic")
	}

	if !a.PollPanicked() {
		t.Error("expected PollPanicked to return true")
	}
}

func TestNewAsync_Poll(t *testing.T) {
	ctx := context.Background()
	a := NewAsync(ctx, func(ctx context.Context) int {
		return 42
	})

	// Initially should be pending
	if !a.PollPending() {
		t.Error("expected PollPending to return true initially")
	}

	// Wait for completion
	<-a.Wait()

	// After completion
	result, ok := a.Poll()
	if !ok {
		t.Error("expected Poll to return true")
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

func TestNewAsync_PollState(t *testing.T) {
	ctx := context.Background()
	a := NewAsync(ctx, func(ctx context.Context) string {
		return "test"
	})

	// Initially should be pending
	state := a.PollState()
	if state.Status != Pending {
		t.Errorf("expected Pending state, got %v", state.Status)
	}

	// Wait for completion
	<-a.Wait()

	// After completion
	state = a.PollState()
	if state.Status != Done {
		t.Errorf("expected Done state, got %v", state.Status)
	}
	if state.Result != "test" {
		t.Errorf("expected 'test', got %v", state.Result)
	}
}

func TestNewAsync_ConcurrentWaiters(t *testing.T) {
	ctx := context.Background()
	a := NewAsync(ctx, func(ctx context.Context) int {
		time.Sleep(100 * time.Millisecond)
		return 123
	})

	// Create multiple waiters
	const numWaiters = 10
	done := make(chan struct{})
	for range numWaiters {
		go func() {
			result := <-a.Wait()
			if result != 123 {
				t.Errorf("expected 123, got %d", result)
			}
			done <- struct{}{}
		}()
	}

	// Wait for all waiters to complete
	for range numWaiters {
		select {
		case <-done:
			// Expected
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for waiters")
		}
	}
}

func TestNewAsync_WithError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("test error")
	a := NewAsync(ctx, func(ctx context.Context) error {
		return expectedErr
	})

	select {
	case result := <-a.Wait():
		if result != expectedErr {
			t.Errorf("expected %v, got %v", expectedErr, result)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for result")
	}
}

func TestNewAsync_WithStruct(t *testing.T) {
	type TestStruct struct {
		ID   int
		Name string
	}

	ctx := context.Background()
	expected := TestStruct{ID: 1, Name: "test"}
	a := NewAsync(ctx, func(ctx context.Context) TestStruct {
		return expected
	})

	select {
	case result := <-a.Wait():
		if result != expected {
			t.Errorf("expected %v, got %v", expected, result)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for result")
	}
}

func TestNewAsync_WaitFirst(t *testing.T) {
	ctx := context.Background()

	async1 := NewAsync(ctx, func(ctx context.Context) int {
		time.Sleep(100 * time.Millisecond)
		return 1
	})

	async2 := NewAsync(ctx, func(ctx context.Context) int {
		time.Sleep(50 * time.Millisecond)
		return 2
	})

	async3 := NewAsync(ctx, func(ctx context.Context) int {
		time.Sleep(150 * time.Millisecond)
		return 3
	})

	var result int

	// Wait for the first result
	select {
	case result = <-async1.Wait():
	case result = <-async2.Wait():
	case result = <-async3.Wait():
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for result")
	}

	if result != 2 {
		t.Errorf("expected 2, got %d", result)
	}
}
