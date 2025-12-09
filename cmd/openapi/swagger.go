package main

import (
	"fmt"

	"github.com/kausys/openapi/swagger"
	"github.com/spf13/cobra"
)

var (
	swaggerOutputDir   string
	swaggerVersion     string
	swaggerUseDefaults bool
	swaggerSimple      bool
)

func init() {
	swaggerCmd.AddCommand(swaggerDownloadCmd)
	swaggerCmd.AddCommand(swaggerVersionCmd)

	swaggerDownloadCmd.Flags().StringVarP(&swaggerOutputDir, "output", "o", ".", "Output directory for swagger-ui.zip")
	swaggerDownloadCmd.Flags().StringVarP(&swaggerVersion, "version", "v", "", "Specific version to download (default: latest)")
	swaggerDownloadCmd.Flags().BoolVar(&swaggerUseDefaults, "with-defaults", true, "Include default initializer and CSS customizations")
	swaggerDownloadCmd.Flags().BoolVar(&swaggerSimple, "simple", false, "Use simple initializer for single-spec mode")

	rootCmd.AddCommand(swaggerCmd)
}

var swaggerCmd = &cobra.Command{
	Use:   "swagger",
	Short: "Swagger UI management commands",
	Long: `Commands for managing Swagger UI assets.

The swagger command group provides utilities for downloading and configuring
Swagger UI for use with your OpenAPI specifications.`,
}

var swaggerDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download the latest Swagger UI release",
	Long: `Download the latest (or specific) Swagger UI release from GitHub.

This command fetches the Swagger UI release, extracts the dist folder,
applies customizations (custom initializer, CSS), removes unnecessary files
(source maps, ES modules), and packages it as swagger-ui.zip ready for embedding.

Examples:
  # Download latest version with default customizations
  openapi swagger download -o ./pkg/openapi

  # Download specific version
  openapi swagger download -v 5.29.4 -o ./pkg/openapi

  # Download without customizations
  openapi swagger download --with-defaults=false -o ./pkg/openapi

  # Download with simple single-spec initializer
  openapi swagger download --simple -o ./pkg/openapi`,
	RunE: runSwaggerDownload,
}

var swaggerVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the latest available Swagger UI version",
	Long:  `Queries the GitHub API to find the latest Swagger UI release version.`,
	RunE:  runSwaggerVersion,
}

func runSwaggerDownload(cmd *cobra.Command, args []string) error {
	opts := swagger.DownloadOptions{
		OutputDir: swaggerOutputDir,
		Version:   swaggerVersion,
	}

	if swaggerUseDefaults {
		opts.CustomCSS = swagger.DefaultCSS
		if swaggerSimple {
			opts.CustomInitializer = swagger.SimpleInitializer
		} else {
			opts.CustomInitializer = swagger.DefaultInitializer
		}
	}

	version, err := swagger.Download(opts)
	if err != nil {
		return fmt.Errorf("failed to download Swagger UI: %w", err)
	}

	fmt.Printf("✅ Successfully downloaded Swagger UI %s\n", version)
	fmt.Printf("\nTo use in your project:\n")
	fmt.Printf("  1. Add to your Go file:\n")
	fmt.Printf("     //go:embed swagger-ui.zip\n")
	fmt.Printf("     var swaggerUIData []byte\n\n")
	fmt.Printf("  2. Create the handler:\n")
	fmt.Printf("     handler, err := swagger.New(swaggerUIData, swagger.Config{\n")
	fmt.Printf("         Specs: map[string][]byte{\"api\": specData},\n")
	fmt.Printf("     })\n")

	return nil
}

func runSwaggerVersion(cmd *cobra.Command, args []string) error {
	version, err := swagger.GetLatestVersion()
	if err != nil {
		return fmt.Errorf("failed to get latest version: %w", err)
	}

	fmt.Printf("Latest Swagger UI version: %s\n", version)
	fmt.Printf("Release URL: https://github.com/swagger-api/swagger-ui/releases/tag/%s\n", version)

	return nil
}

