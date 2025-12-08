package main

import (
	"fmt"

	"github.com/kausys/openapi/cache"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cleanCmd)
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean the cache directory",
	Long: `Clean removes the .openapi cache directory.

This forces a full rebuild on the next generate command.

Example:
  openapi clean`,
	RunE: runClean,
}

func runClean(cmd *cobra.Command, args []string) error {
	mgr := cache.NewManager(".")

	if err := mgr.Clean(); err != nil {
		return fmt.Errorf("failed to clean cache: %w", err)
	}

	fmt.Println("✅ Cache cleaned successfully")
	return nil
}
