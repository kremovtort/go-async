# go-async

A Go library that implements functionality similar to Haskell's `async` and `async-pool` libraries, providing tools for asynchronous computation and parallel processing in Go.

## Features (Planned)

- Async computation management
- Parallel execution pools
- Cancellation support
- Error handling and propagation
- Resource management

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

MIT License (to be added)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. 