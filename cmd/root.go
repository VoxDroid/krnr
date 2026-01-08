// Package cmd provides the CLI commands for krnr.
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
	PersistentPreRun: func(cmd *cobra.Command, _ []string) {
		// Easter egg flag: --whoami
		who, _ := cmd.Flags().GetBool("whoami")
		if who {
			fmt.Println("Hello there, you found something interesting, isn't it?")
			fmt.Println("I'm @VoxDroid — https://github.com/VoxDroid")
			os.Exit(0)
		}
	},
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println("krnr: run 'krnr --help' to see available commands")
	},
}

func init() {
	// hidden easter-egg flag
	rootCmd.PersistentFlags().Bool("whoami", false, "Easter egg: show author handle")
	_ = rootCmd.PersistentFlags().MarkHidden("whoami")
}

// Execute executes the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
