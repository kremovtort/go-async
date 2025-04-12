# go-async

A Go library that implements functionality similar to Haskell's `async` library, providing tools for asynchronous computation and parallel processing in Go.

## Features

- **Async Computation**: Run functions asynchronously and handle their results
- **Cancellation Support**: Cancel async computations using context.Context
- **Error Handling**: Proper error propagation and panic recovery
- **Parallel Execution**: Run multiple computations in parallel with various waiting strategies
- **Type Safety**: Full support for Go generics

## Installation

```bash
go get github.com/kremovtort/go-async
```

## Usage

### Basic Async Computation

```go
import (
    "context"
    "github.com/kremovtort/go-async/pkg/async"
)

// Create a context
ctx := context.Background()

// Run a function asynchronously
asyncTask := async.NewAsync(ctx, func(ctx context.Context) string {
    // Do some work
    return "result"
})

// Wait for the result
result, err := asyncTask.Wait()
if err != nil {
    // Handle error
} else {
    // Use result
}
```

### Cancellation

```go
// Create a context with cancellation
ctx, cancel := context.WithCancel(context.Background())

// Run a function asynchronously
asyncTask := async.NewAsync(ctx, func(ctx context.Context) string {
    // Check for cancellation
    select {
    case <-ctx.Done():
        return ""
    default:
        // Do some work
        return "result"
    }
})

// Cancel the computation
cancel()

// Wait for the result (will return context cancelled error)
_, err := asyncTask.Wait()
```

### Race

```go
// Run two computations in parallel and get the first result
result, err := async.Race(ctx, 
    func(ctx context.Context) string { 
        time.Sleep(2 * time.Second)
        return "first"
    },
    func(ctx context.Context) int { 
        time.Sleep(1 * time.Second)
        return 42
    },
)
// result will be either "first" or 42, depending on which completes first
```

### Concurrently

```go
// Run two computations in parallel and wait for both
str, num, err := async.Concurrently(ctx,
    func(ctx context.Context) string { 
        time.Sleep(1 * time.Second)
        return "first"
    },
    func(ctx context.Context) int { 
        time.Sleep(2 * time.Second)
        return 42
    },
)
// str will be "first" and num will be 42
```

### WaitAny

```go
// Create multiple async computations
a1 := async.NewAsync(ctx, func(ctx context.Context) string {
    time.Sleep(3 * time.Second)
    return "slow"
})

a2 := async.NewAsync(ctx, func(ctx context.Context) string {
    time.Sleep(1 * time.Second)
    return "fast"
})

// Wait for any to complete
result, err := async.WaitAny(a1, a2)
// result will be "fast"
```

### WaitAll

```go
// Create multiple async computations
a1 := async.NewAsync(ctx, func(ctx context.Context) string {
    time.Sleep(1 * time.Second)
    return "first"
})

a2 := async.NewAsync(ctx, func(ctx context.Context) string {
    time.Sleep(2 * time.Second)
    return "second"
})

// Wait for all to complete
results, err := async.WaitAll(a1, a2)
// results will be []string{"first", "second"}
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

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. 