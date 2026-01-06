package cmd

import (
	"fmt"

	"github.com/VoxDroid/krnr/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("krnr %s\n", version.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
