package main

import (
	"fmt"

	"github.com/kausys/openapi/cache"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show cache status",
	Long: `Status displays information about the current cache state.

It shows the number of cached files, schemas, routes, and parameters.

Example:
  openapi status`,
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	mgr := cache.NewManager(".")

	if err := mgr.Init(); err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	if err := mgr.Load(); err != nil {
		fmt.Println("📭 No cache found")
		return nil
	}

	stats := mgr.Stats()

	fmt.Println("📊 Cache Status")
	fmt.Println("───────────────")
	fmt.Printf("   Files:      %d\n", stats.FileCount)
	fmt.Printf("   Schemas:    %d\n", stats.SchemaCount)
	fmt.Printf("   Routes:     %d\n", stats.RouteCount)
	fmt.Printf("   Parameters: %d\n", stats.ParameterCount)

	return nil
}
