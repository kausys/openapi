// Package main provides the CLI entry point for the OpenAPI generator.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time
	Version = "dev"
	// Commit is set at build time
	Commit = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "openapi",
	Short: "OpenAPI 3.1 specification generator for Go",
	Long: `OpenAPI is a tool that generates OpenAPI 3.1 specifications from Go source code.

It scans your Go code for swagger directives and generates a complete OpenAPI specification
in YAML or JSON format.

Example:
  openapi generate -o openapi.yaml
  openapi generate --pattern ./api/... -o api-spec.json --format json
  openapi clean`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}
