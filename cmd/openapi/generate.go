package main

import (
	"fmt"

	"github.com/kausys/openapi/generator"
	"github.com/spf13/cobra"
)

var (
	outputFile   string
	outputFormat string
	pattern      string
	dir          string
	noCache      bool
	flatten      bool
	validate     bool
	ignorePaths  []string
	cleanUnused  bool
	multiSpec    bool
	specName     string
	noDefault    bool
	enumRefs     bool
)

func init() {
	generateCmd.Flags().StringVarP(&outputFile, "output", "o", "openapi.yaml", "Output file path")
	generateCmd.Flags().StringVarP(&outputFormat, "format", "f", "yaml", "Output format: yaml or json")
	generateCmd.Flags().StringVarP(&pattern, "pattern", "p", "./...", "Package pattern to scan")
	generateCmd.Flags().StringVarP(&dir, "dir", "d", ".", "Root directory to scan from")
	generateCmd.Flags().BoolVar(&noCache, "no-cache", false, "Disable incremental caching")
	generateCmd.Flags().BoolVar(&flatten, "flatten", false, "Inline $ref schemas instead of using references")
	generateCmd.Flags().BoolVar(&validate, "validate", false, "Validate the generated spec")
	generateCmd.Flags().StringSliceVar(&ignorePaths, "ignore", nil, "Path patterns to ignore")
	generateCmd.Flags().BoolVar(&cleanUnused, "clean-unused", false, "Remove unreferenced schemas")
	generateCmd.Flags().BoolVar(&multiSpec, "multi-specs", false, "Generate multiple specs based on spec: directives")
	generateCmd.Flags().StringVar(&specName, "spec", "", "Generate only a specific spec by name")
	generateCmd.Flags().BoolVar(&noDefault, "no-default", false, "Skip generating the default spec for routes without spec: directives")
	generateCmd.Flags().BoolVar(&enumRefs, "enum-refs", false, "Generate enums as $ref references instead of inline")
	rootCmd.AddCommand(generateCmd)
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate OpenAPI specification from Go source code",
	Long: `Generate scans Go source code for swagger directives and generates
an OpenAPI 3.1 specification.

Supported directives:
  swagger:meta       - API metadata (title, version, description)
  swagger:model      - Schema definitions
  swagger:route      - Operation definitions
  swagger:parameters - Parameter definitions
  swagger:enum       - Enum definitions

Example:
  openapi generate
  openapi generate -o api.yaml -p ./api/...
  openapi generate -o api.json -f json --no-cache`,
	RunE: runGenerate,
}

func runGenerate(cmd *cobra.Command, args []string) error {
	// Load custom types from config file
	if err := generator.LoadConfigFile(dir); err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	gen := generator.New(
		generator.WithDir(dir),
		generator.WithPattern(pattern),
		generator.WithOutput(outputFile, outputFormat),
		generator.WithCache(!noCache),
		generator.WithFlatten(flatten),
		generator.WithValidation(validate),
		generator.WithIgnorePaths(ignorePaths...),
		generator.WithCleanUnused(cleanUnused),
		generator.WithNoDefault(noDefault),
		generator.WithEnumRefs(enumRefs),
	)

	if multiSpec {
		_, err := gen.GenerateMulti()
		if err != nil {
			return fmt.Errorf("multi-spec generation failed: %w", err)
		}
		return nil
	}

	if specName != "" {
		_, err := gen.GenerateSpec(specName)
		if err != nil {
			return fmt.Errorf("spec generation failed: %w", err)
		}
		return nil
	}

	_, err := gen.Generate()
	if err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	return nil
}
