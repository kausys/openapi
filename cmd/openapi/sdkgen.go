package main

import (
	"fmt"

	"github.com/kausys/openapi/sdkgen"
	"github.com/spf13/cobra"
)

var (
	sdkgenOutputDir string
	sdkgenProvider  string
)

func init() {
	sdkgenCmd.Flags().StringVarP(&sdkgenOutputDir, "output", "o", "", "Output directory for generated SDK (required)")
	sdkgenCmd.Flags().StringVar(&sdkgenProvider, "provider", "", "Override provider name from config")
	_ = sdkgenCmd.MarkFlagRequired("output")
	rootCmd.AddCommand(sdkgenCmd)
}

var sdkgenCmd = &cobra.Command{
	Use:   "sdkgen <config.sdkgen.yaml>",
	Short: "Generate Go SDK package from OpenAPI spec",
	Long: `Generates a complete Go SDK package from an OpenAPI specification
and an SDK configuration file (.sdkgen.yaml).

The generated SDK follows the standard 4-layer architecture:
  - client/   HTTP client with middleware chain
  - config/   Configuration with gookit/config
  - models/   Request/response structs and enums
  - services/ Service methods per API tag

Example:
  openapi sdkgen pokemon.sdkgen.yaml -o ./pkg/sdk/pokemon
  openapi sdkgen pokemon.sdkgen.yaml -o ./pkg/sdk/pokemon --provider myProvider`,
	Args: cobra.ExactArgs(1),
	RunE: runSDKGen,
}

func runSDKGen(cmd *cobra.Command, args []string) error {
	opts := []sdkgen.Option{
		sdkgen.WithConfigPath(args[0]),
		sdkgen.WithOutputDir(sdkgenOutputDir),
	}

	if sdkgenProvider != "" {
		opts = append(opts, sdkgen.WithProvider(sdkgenProvider))
	}

	gen := sdkgen.New(opts...)

	if err := gen.Generate(); err != nil {
		return fmt.Errorf("SDK generation failed: %w", err)
	}

	fmt.Println("SDK generated successfully")
	return nil
}
