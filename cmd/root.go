package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "krnr",
	Short: "krnr is a global, SQLite-backed command runner",
	Long:  "krnr provides a global registry of named terminal workflows",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("krnr: run 'krnr --help' to see available commands")
	},
}

// Execute executes the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
