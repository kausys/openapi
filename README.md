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
- 🎨 **Swagger UI Integration** - Download and serve Swagger UI with multi-spec support
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

## 🎨 Swagger UI Integration

The `swagger` package provides everything you need to serve Swagger UI with your OpenAPI specs.

### Download Swagger UI

Use the CLI to download the latest Swagger UI release from GitHub:

```bash
# Check latest available version
openapi swagger version

# Download latest version with default customizations
openapi swagger download -o ./pkg/docs

# Download specific version
openapi swagger download -v 5.29.4 -o ./pkg/docs

# Download with simple single-spec initializer
openapi swagger download --simple -o ./pkg/docs

# Download without customizations
openapi swagger download --with-defaults=false -o ./pkg/docs
```

The download command automatically:
- Fetches the latest release from GitHub API
- Extracts only the `dist/` folder
- Removes source maps and ES module bundles
- Injects custom initializer for multi-spec support
- Adds custom CSS to hide Swagger branding

### Serve Swagger UI in Your Application

```go
package docs

import (
    _ "embed"
    "net/http"

    "github.com/kausys/openapi/swagger"
)

//go:embed swagger-ui.zip
var swaggerUIData []byte

//go:embed openapi.yaml
var specData []byte

func NewHandler() (*swagger.Handler, error) {
    return swagger.New(swaggerUIData, swagger.Config{
        BasePath:      "/swagger",      // Swagger UI served here
        SpecPath:      "/openapi/specs", // OpenAPI spec endpoint
        ResourcesPath: "/openapi/resources", // Multi-spec dropdown
        Specs: map[string][]byte{
            "api": specData,
        },
        DefaultSpec: "api",
    })
}
```

### Framework Integration

**Standard Library / Chi / Echo:**
```go
handler, _ := docs.NewHandler()
http.Handle("/swagger/", handler)
http.HandleFunc("/openapi/specs", handler.ServeHTTP)
http.HandleFunc("/openapi/resources", handler.ServeHTTP)

// Or register all routes at once:
mux := http.NewServeMux()
handler.Routes(mux)
```

**Gin:**
```go
handler, _ := docs.NewHandler()
router := gin.Default()
router.Any("/swagger/*any", gin.WrapH(handler))
router.GET("/openapi/specs", gin.WrapH(handler))
router.GET("/openapi/resources", gin.WrapH(handler))
```

**Fiber:**
```go
import "github.com/gofiber/adaptor/v2"

handler, _ := docs.NewHandler()
app := fiber.New()
app.Use("/swagger", adaptor.HTTPHandler(handler))
app.Get("/openapi/specs", adaptor.HTTPHandlerFunc(handler.ServeHTTP))
app.Get("/openapi/resources", adaptor.HTTPHandlerFunc(handler.ServeHTTP))
```

### Multi-Spec Support

Serve multiple API specs with a dropdown selector:

```go
handler, _ := swagger.New(swaggerUIData, swagger.Config{
    Specs: map[string][]byte{
        "public": publicSpecData,
        "admin":  adminSpecData,
        "mobile": mobileSpecData,
    },
    DefaultSpec: "public",
})
```

The Swagger UI will display a dropdown allowing users to switch between specs.

### Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `BasePath` | URL path for Swagger UI | `/swagger` |
| `SpecPath` | URL path for OpenAPI specs | `/openapi/specs` |
| `ResourcesPath` | URL path for spec list (multi-spec dropdown) | `/openapi/resources` |
| `Specs` | Map of spec name to YAML/JSON bytes | required |
| `DefaultSpec` | Default spec when no query param | first spec |

### go:generate Integration

Add to your docs package for automatic spec generation:

```go
//go:generate openapi generate -d ../.. -p ./... -o openapi.yaml --clean-unused
package docs
```

Then run:
```bash
go generate ./pkg/docs
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

