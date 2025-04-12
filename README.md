# go-async

A Go package that provides an elegant abstraction for asynchronous computations. The package introduces the concept of `Async[T]` - a type-safe wrapper around concurrent operations that will eventually produce a value of type `T`.

## Overview

This package offers a higher-level interface for managing concurrent computations in Go. Instead of directly dealing with goroutines and channels, it provides a clean abstraction where an `Async[T]` represents a computation that will eventually deliver a value of type `T`. The package includes utilities for:

- Creating asynchronous computations
- Waiting for their results
- Polling for completion
- Cancelling operations
- Combining multiple computations
- Error handling and panic recovery

## Key Features

- **Type Safety**: Leverages Go generics to ensure type safety across all operations
- **Context Support**: Full integration with Go's context package for cancellation and timeouts
- **Composable**: Functions to combine multiple async computations (`Either`, `Both`, `All`, `Any`)
- **Error Handling**: Built-in panic recovery and error propagation
- **Resource Management**: Automatic cleanup of resources through cancellation
- **Non-blocking Operations**: `Poll()` method for checking completion without blocking

## Example Usage

```go
package main

import (
	"context"
	"fmt"
	"time"

	async "github.com/kremovtort/go-async"
)

func main() {
	ctx := context.Background()

	// Create an async computation
	a := async.NewAsync(ctx, func(ctx context.Context) string {
		// Simulate some work
		time.Sleep(100 * time.Millisecond)
		return "Hello, World!"
	})

	// Wait for the result
	result, err := a.Wait()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println(result) // Output: Hello, World!
}
```

### Polling for Results

```go
package main

import (
	"context"
	"fmt"
	"time"

	async "github.com/kremovtort/go-async"
)

func main() {
	ctx := context.Background()

	a := async.NewAsync(ctx, func(ctx context.Context) string {
		time.Sleep(100 * time.Millisecond)
		return "Result"
	})

	// Poll for the result
	for {
		result, done, err := a.Poll()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		if done {
			fmt.Println(result) // Output: Result
			return
		}

		fmt.Println("Still working...")
		time.Sleep(10 * time.Millisecond)
	}
}
```

### Working with Multiple Async Computations

```go
package main

import (
	"context"
	"fmt"
	"time"

	async "github.com/kremovtort/go-async"
)

func main() {
	ctx := context.Background()

	// Run two computations in parallel and get the first result
	result, err := async.Either(ctx,
		func(ctx context.Context) string {
			time.Sleep(200 * time.Millisecond)
			return "Slow"
		},
		func(ctx context.Context) int {
			time.Sleep(100 * time.Millisecond)
			return 42
		},
	)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println(result) // Output: 42

	// Run two computations in parallel and wait for both
	str, num, err := async.Both(ctx,
		func(ctx context.Context) string {
			time.Sleep(100 * time.Millisecond)
			return "First"
		},
		func(ctx context.Context) int {
			time.Sleep(200 * time.Millisecond)
			return 42
		},
	)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("%s, %d\n", str, num) // Output: First, 42
}
```

### Cancellation

```go
package main

import (
	"context"
	"fmt"
	"time"

	async "github.com/kremovtort/go-async"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	a := async.NewAsync(ctx, func(ctx context.Context) string {
		// Simulate some work
		time.Sleep(100 * time.Millisecond)
		return "Result"
	})

	// Cancel the computation
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// Wait for the result
	_, err := a.Wait()
	if err != nil {
		fmt.Printf("Error: %v\n", err) // Output: Error: context canceled
	}
}
```

## Development

This project uses Nix flakes for development environment management. To get started:

1. Make sure you have Nix installed with flakes enabled
2. Clone the repository:
   ```bash
   git clone https://github.com/kremovtort/go-async.git
   cd go-async
   ```
3. Enter the development environment:
   ```bash
   nix develop
   ```

This will provide you with all necessary development tools:
- Go compiler
- gopls (Go language server)
- delve (Go debugger)
- golangci-lint (Go linter)
- pkgsite (Go documentation generator)
- just (Task runner)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. 