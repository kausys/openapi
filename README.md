# OpenAPI Generator for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/kausys/openapi.svg)](https://pkg.go.dev/github.com/kausys/openapi)
[![Go Report Card](https://goreportcard.com/badge/github.com/kausys/openapi)](https://goreportcard.com/report/github.com/kausys/openapi)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub Sponsors](https://img.shields.io/github/sponsors/kausys?style=social)](https://github.com/sponsors/kausys)

Generate **OpenAPI 3.0.4** specifications from Go source code using swagger-style comments. Features an extensible parser system, multi-spec generation, incremental caching, and a powerful CLI.

## ✨ Features

- 🚀 **OpenAPI 3.0.4 Compliant** - Full support for the latest OpenAPI specification
- 📝 **Swagger-style Comments** - Familiar `swagger:meta`, `swagger:route`, `swagger:model` directives
- 🔌 **Extensible Parser System** - Register custom parsers for new directives
- 📦 **Multi-Spec Generation** - Generate multiple API specs from a single codebase
- ⚡ **Incremental Caching** - Only re-process changed files
- 🧹 **Smart Schema Cleaning** - Only include schemas used by each spec
- 🛠️ **Powerful CLI** - Easy-to-use command-line interface

## 📦 Installation

```bash
go install github.com/kausys/openapi/cmd/openapi@latest
```

Or add to your project:

```bash
go get github.com/kausys/openapi
```

## 🚀 Quick Start

### 1. Add swagger comments to your code

```go
// swagger:meta
// Title: My Awesome API
// Version: 1.0.0
// Description: A sample API to demonstrate OpenAPI generation
// License: MIT
type Meta struct{}

// swagger:route GET /users users listUsers
// Summary: List all users
// Description: Returns a list of all users in the system
// Security: bearerAuth
// Responses:
//   200: usersResponse

// swagger:model
type User struct {
    // The unique identifier
    // Example: 123
    ID   int    `json:"id"`
    // The user's full name
    // Example: John Doe
    Name string `json:"name"`
}
```

### 2. Generate the OpenAPI spec

```bash
openapi generate -o api.yaml
```

### 3. Output

```yaml
openapi: "3.0.4"
info:
  title: My Awesome API
  version: 1.0.0
  description: A sample API to demonstrate OpenAPI generation
  license:
    name: MIT
paths:
  /users:
    get:
      operationId: listUsers
      summary: List all users
      tags:
        - users
      responses:
        "200":
          description: OK
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: integer
          example: 123
        name:
          type: string
          example: John Doe
```

## 📖 Documentation

### Directives

| Directive | Description |
|-----------|-------------|
| `swagger:meta` | API metadata (title, version, description, etc.) |
| `swagger:route` | Operation definitions |
| `swagger:model` | Schema/model definitions |
| `swagger:parameters` | Parameter definitions |
| `swagger:enum` | Enum definitions |
| `swagger:allOf` | Schema composition |

### Multi-Spec Generation

Generate multiple API specs from a single codebase using the `spec:` directive:

```go
// swagger:route GET /admin/users admin listAdminUsers
// spec: admin
type AdminListUsers struct{}

// swagger:route GET /users users listUsers
// spec: public mobile
type ListUsers struct{}
```

```bash
openapi generate --multi-specs -o ./specs/
# Generates: specs/admin.yaml, specs/public.yaml, specs/mobile.yaml
```

### CLI Options

```
Usage:
  openapi generate [flags]

Flags:
  -o, --output string    Output file path (default "openapi.yaml")
  -f, --format string    Output format: yaml or json (default "yaml")
  -p, --pattern string   Package pattern to scan (default "./...")
  -d, --dir string       Root directory to scan (default ".")
      --no-cache         Disable incremental caching
      --multi-specs      Generate multiple specs based on spec: directives
      --spec string      Generate only a specific spec by name
      --clean-unused     Remove unreferenced schemas
      --validate         Validate the generated spec
```

## 🔌 Extensible Parser System

Register custom parsers to extend the functionality:

```go
package main

import (
    "github.com/kausys/openapi/parser"
    _ "github.com/kausys/openapi/parser/tags" // Import built-in parsers
)

func init() {
    // Register a custom parser
    parser.Register(parser.DirectiveRoute, &MyCustomParser{})
}
```

## 💖 Support the Project

If you find this project useful, please consider supporting its development:

- ⭐ Star this repository
- 🐛 Report bugs and suggest features
- 🔀 Submit pull requests
- 💰 [Sponsor on GitHub](https://github.com/sponsors/kausys)
- ☕ [Buy me a coffee](https://ko-fi.com/reationio)

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

## 🙏 Acknowledgments

Inspired by [go-swagger](https://github.com/go-swagger/go-swagger) with a modern, extensible architecture.

