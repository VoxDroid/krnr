package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VoxDroid/krnr/internal/install"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall krnr (remove binary and PATH modifications)",
	RunE: func(cmd *cobra.Command, _ []string) error {
		dry, _ := cmd.Flags().GetBool("dry-run")
		yes, _ := cmd.Flags().GetBool("yes")
		verbose, _ := cmd.Flags().GetBool("verbose")

		// Show plan
		actions, err := install.PlanUninstall()
		if err != nil {
			return err
		}
		fmt.Printf("Planned actions for uninstall:\n")
		for _, a := range actions {
			fmt.Printf("- %s\n", a)
		}
		if dry {
			return nil
		}

		if !yes {
			fmt.Print("Proceed with uninstall? [y/N]: ")
			r := bufio.NewReader(os.Stdin)
			resp, _ := r.ReadString('\n')
			resp = strings.TrimSpace(strings.ToLower(resp))
			if resp != "y" {
				fmt.Println("aborted by user (use --yes to skip confirmation)")
				return nil
			}
		}

		actions, err = install.Uninstall(verbose)
		if err != nil {
			return err
		}
		for _, a := range actions {
			fmt.Printf("- %s\n", a)
		}
		fmt.Println("uninstall completed")
		return nil
	},
}

func init() {
	uninstallCmd.Flags().BoolP("dry-run", "n", false, "Show actions but do not perform them")
	uninstallCmd.Flags().Bool("yes", false, "Assume yes for prompts")
	uninstallCmd.Flags().Bool("verbose", false, "Show verbose diagnostic information (before/after PATH when applicable)")
	rootCmd.AddCommand(uninstallCmd)
}
