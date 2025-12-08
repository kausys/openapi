# Contributing to OpenAPI Generator for Go

First off, thank you for considering contributing to this project! 🎉

## 📋 Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Pull Request Process](#pull-request-process)
- [Coding Guidelines](#coding-guidelines)
- [Adding New Parsers](#adding-new-parsers)

## Code of Conduct

This project adheres to a [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/openapi.git`
3. Add upstream remote: `git remote add upstream https://github.com/kausys/openapi.git`
4. Create a branch: `git checkout -b feature/amazing-feature`

## Development Setup

### Prerequisites

- Go 1.25 or later
- Make (optional, for convenience commands)

### Building

```bash
# Build the CLI
go build -o openapi ./cmd/openapi

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run linter
golangci-lint run
```

### Project Structure

```
openapi/
├── cache/          # Incremental caching system
├── cmd/openapi/    # CLI application
├── generator/      # OpenAPI spec generation
├── parser/         # Extensible parser system
│   └── tags/       # Built-in tag parsers
├── scanner/        # Go source code scanner
├── spec/           # OpenAPI 3.0.4 type definitions
└── examples/       # Usage examples
```

## How to Contribute

### Reporting Bugs

1. Check if the bug has already been reported in [Issues](https://github.com/kausys/openapi/issues)
2. If not, create a new issue using the Bug Report template
3. Include:
   - Go version (`go version`)
   - Operating system
   - Steps to reproduce
   - Expected vs actual behavior
   - Minimal code example if possible

### Suggesting Features

1. Check if the feature has already been requested
2. Create a new issue using the Feature Request template
3. Explain the use case and benefits

### Submitting Code

1. Ensure your code follows the [Coding Guidelines](#coding-guidelines)
2. Add tests for new functionality
3. Update documentation if needed
4. Run all tests: `go test ./...`
5. Submit a pull request

## Pull Request Process

1. Update the README.md if needed
2. Add or update tests
3. Ensure CI passes
4. Request review from maintainers
5. Squash commits before merging (if requested)

### Commit Message Format

```
type(scope): brief description

Longer description if needed.

Fixes #123
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

## Coding Guidelines

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting
- Export only what's necessary
- Add godoc comments for exported types/functions

### Testing

- Write table-driven tests when appropriate
- Aim for meaningful test coverage
- Use descriptive test names

### Error Handling

- Return errors, don't panic
- Wrap errors with context: `fmt.Errorf("doing X: %w", err)`
- Use custom error types for typed error handling

## Adding New Parsers

The parser system is designed to be extensible. Here's how to add a new parser:

### 1. Create a new parser file

```go
// parser/tags/myparser.go
package tags

import (
    "github.com/kausys/openapi/parser"
    "github.com/kausys/openapi/spec"
)

func NewMyParser() *SingleLineParser {
    return NewSingleLineParser(
        "myparser",           // Parser name
        "mydirective:",       // Directive prefix
        []parser.Context{parser.ContextRoute}, // Supported contexts
        parser.SetterMap{
            parser.ContextRoute: func(target any, value any) error {
                if op, ok := target.(*spec.Operation); ok {
                    // Apply the value to the operation
                    return nil
                }
                return &parser.ErrInvalidTarget{...}
            },
        },
    )
}
```

### 2. Register in init.go

```go
// parser/tags/init.go
func registerRouteParsers() {
    // ... existing parsers ...
    parser.Register(parser.DirectiveRoute, NewMyParser())
}
```

### 3. Add tests

```go
// parser/tags/myparser_test.go
func TestMyParser(t *testing.T) {
    // Test your parser
}
```

## Questions?

Feel free to open an issue or reach out to the maintainers.

Thank you for contributing! 💖

