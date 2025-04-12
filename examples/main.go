package main

import (
	"context"
	"fmt"
	"time"

	"github.com/kremovtort/go-async/pkg/async"
)

func main() {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Example 1: Simple async computation
	fmt.Println("Example 1: Simple async computation")
	asyncTask := async.NewAsync(ctx, func(ctx context.Context) string {
		// Simulate some work
		time.Sleep(1 * time.Second)
		return "Hello, async!"
	})

	result, err := asyncTask.Wait()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %s\n", result)
	}

	// Example 2: Using Race to implement a timeout
	fmt.Println("\nExample 2: Using Race to implement a timeout")
	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer timeoutCancel()

	slowTask := func(ctx context.Context) string {
		time.Sleep(3 * time.Second)
		return "This should timeout"
	}

	timeoutTask := func(ctx context.Context) string {
		<-ctx.Done()
		return "Timeout occurred"
	}

	raceResult, err := async.Either(timeoutCtx, slowTask, timeoutTask)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Race result: %v\n", raceResult)
	}

	// Example 3: Using Concurrently to run multiple computations
	fmt.Println("\nExample 3: Using Concurrently to run multiple computations")
	ctx2 := context.Background()

	str, num, err := async.Both(ctx2,
		func(ctx context.Context) string {
			time.Sleep(1 * time.Second)
			return "First computation"
		},
		func(ctx context.Context) int {
			time.Sleep(2 * time.Second)
			return 42
		},
	)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Concurrent results: %s, %d\n", str, num)
	}

	// Example 4: Using WaitAny to get the first result from multiple computations
	fmt.Println("\nExample 4: Using WaitAny to get the first result from multiple computations")
	ctx3 := context.Background()

	a1 := async.NewAsync(ctx3, func(ctx context.Context) string {
		time.Sleep(3 * time.Second)
		return "Slow computation"
	})

	a2 := async.NewAsync(ctx3, func(ctx context.Context) string {
		time.Sleep(1 * time.Second)
		return "Fast computation"
	})

	a3 := async.NewAsync(ctx3, func(ctx context.Context) string {
		time.Sleep(2 * time.Second)
		return "Medium computation"
	})

	waitAnyResult, err := async.WaitAny(a1, a2, a3)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("WaitAny result: %s\n", waitAnyResult)
	}

	// Example 5: Using WaitAll to get all results from multiple computations
	fmt.Println("\nExample 5: Using WaitAll to get all results from multiple computations")
	ctx4 := context.Background()

	b1 := async.NewAsync(ctx4, func(ctx context.Context) string {
		time.Sleep(1 * time.Second)
		return "First"
	})

	b2 := async.NewAsync(ctx4, func(ctx context.Context) string {
		time.Sleep(2 * time.Second)
		return "Second"
	})

	b3 := async.NewAsync(ctx4, func(ctx context.Context) string {
		time.Sleep(3 * time.Second)
		return "Third"
	})

	waitAllResults, err := async.WaitAll(b1, b2, b3)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("WaitAll results: %v\n", waitAllResults)
	}
}
