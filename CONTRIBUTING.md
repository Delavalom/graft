# Contributing to Graft

Thank you for your interest in contributing to Graft!

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/<your-username>/graft.git`
3. Create a branch: `git checkout -b my-feature`
4. Make your changes
5. Run tests: `go test ./...`
6. Push and open a pull request

## Development

**Requirements:** Go 1.24+

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./provider/anthropic/

# Build
go build ./...

# Run an example
go run ./examples/basic/
```

## Guidelines

- Follow existing code patterns (functional options, type-safe tools, etc.)
- Add tests for new functionality
- Provider tests hit real APIs — set the appropriate environment variable (`OPENROUTER_API_KEY`, `ANTHROPIC_API_KEY`, `GOOGLE_API_KEY`)
- Keep dependencies minimal — Graft's only external dependency is OpenTelemetry
- Use `NewAgentError` with appropriate error types from `errors.go`

## Pull Request Process

1. Ensure `go test ./...` and `go vet ./...` pass
2. Update documentation if adding new features
3. Add an example in `examples/` for significant new functionality
4. Keep PRs focused — one feature or fix per PR

## Reporting Issues

Use [GitHub Issues](https://github.com/delavalom/graft/issues) with the provided templates for bug reports and feature requests.
