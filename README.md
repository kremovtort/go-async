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
- gotools
- gopls (Go language server)
- delve (Go debugger)
- golangci-lint (Go linter)
- just (Task runner)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. 