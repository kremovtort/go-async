# go-async

A Go library for asynchronous computations with a clean and consistent API.

## Features

- Simple and intuitive API for asynchronous computations
- Context support for cancellation and timeouts
- Non-blocking polling for results
- Utilities for working with multiple async computations
- Type-safe with Go generics

## Installation

```bash
go get github.com/kremovtort/go-async
```

## Usage

### Basic Usage

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/kremovtort/go-async/pkg/async"
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

	"github.com/kremovtort/go-async/pkg/async"
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

	"github.com/kremovtort/go-async/pkg/async"
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

	"github.com/kremovtort/go-async/pkg/async"
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

## API

### NewAsync

```go
func NewAsync[T any](ctx context.Context, f func(context.Context) T) *Async[T]
```

Creates a new async computation that will execute the function `f` in a goroutine.

### Wait

```go
func (a *Async[T]) Wait() (T, error)
```

Waits for the async computation to complete and returns its result.

### Poll

```go
func (a *Async[T]) Poll() (T, bool, error)
```

Checks if the async computation has completed without blocking. Returns the result, a boolean indicating if the computation is done, and any error that occurred.

### Cancel

```go
func (a *Async[T]) Cancel() error
```

Cancels the async computation.

### Either

```go
func Either[T1, T2 any](ctx context.Context, f1 func(context.Context) T1, f2 func(context.Context) T2) (interface{}, error)
```

Runs two computations in parallel and returns the result of the first one to complete.

### Both

```go
func Both[T1, T2 any](ctx context.Context, f1 func(context.Context) T1, f2 func(context.Context) T2) (T1, T2, error)
```

Runs two computations in parallel and waits for both to complete.

### WaitAny

```go
func WaitAny[T any](asyncs ...*Async[T]) (T, error)
```

Waits for any of the async computations to complete and returns its result.

### WaitAll

```go
func WaitAll[T any](asyncs ...*Async[T]) ([]T, error)
```

Waits for all async computations to complete and returns their results.

### WaitBoth

```go
func WaitBoth[T1, T2 any](a1 *Async[T1], a2 *Async[T2]) (T1, T2, error)
```

Waits for both async computations to complete and returns their results.

### WaitEither

```go
func WaitEither[T1, T2 any](a1 *Async[T1], a2 *Async[T2]) (interface{}, error)
```

Waits for either of two async computations to complete and returns its result.

## License

MIT

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
- just (Task runner)

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. 