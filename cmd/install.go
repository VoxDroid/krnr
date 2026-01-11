package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VoxDroid/krnr/internal/install"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the krnr binary to your system or user path",
	Long:  "Install the current krnr binary to a per-user or system path. Use --dry-run to preview actions.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		user, _ := cmd.Flags().GetBool("user")
		system, _ := cmd.Flags().GetBool("system")
		path, _ := cmd.Flags().GetString("path")
		from, _ := cmd.Flags().GetString("from")
		dry, _ := cmd.Flags().GetBool("dry-run")
		check, _ := cmd.Flags().GetBool("check")
		yes, _ := cmd.Flags().GetBool("yes")
		opts := install.Options{User: user, System: system, Path: path, From: from, DryRun: dry, Check: check, Yes: yes}
		actions, target, err := install.PlanInstall(opts)
		if err != nil {
			return err
		}
		// Print plan
		fmt.Printf("Planned actions for install to %s:\n", target)
		for _, a := range actions {
			fmt.Printf("- %s\n", a)
		}

		if dry || check {
			return nil
		}
	// If the plan suggests adding to PATH, print an explicit visible prompt line so
	// tests and users will always see it even when stdin/stdout handling is odd.
	printedPathPrompt := false
	for _, a := range actions {
		al := strings.ToLower(a)
		if strings.Contains(al, "add") && strings.Contains(al, "path") {
			if system {
				fmt.Print("Target dir is not on PATH. System PATH modification may require admin privileges. Add it to PATH now? [y/N]: ")
			} else {
				fmt.Print("Target dir is not on PATH. Add it to PATH now? [y/N]: ")
			}
			printedPathPrompt = true
			break
		}
	}
		// contain the target dir (covers edge cases where plan suggests adding).
		wantPathPrompt := false
		for _, a := range actions {
			al := strings.ToLower(a)
			if strings.Contains(al, "add") && strings.Contains(al, "path") {
				wantPathPrompt = true
				break
			}
		}

		// Ask whether to add to PATH if missing or if plan suggested it
		pathEnv := os.Getenv("PATH")
		targetDir := filepath.Dir(target)
		if wantPathPrompt || !install.ContainsPath(pathEnv, targetDir) {
			// Use Flags().Changed to detect flags explicitly provided in this invocation.
			addFlagSet := cmd.Flags().Changed("add-to-path")
			yesFlagSet := cmd.Flags().Changed("yes")
			if !addFlagSet && !yesFlagSet && !opts.Yes {
				if !printedPathPrompt {
					if opts.System {
						fmt.Print("Target dir is not on PATH. System PATH modification may require admin privileges. Add it to PATH now? [y/N]: ")
						} else {
						fmt.Print("Target dir is not on PATH. Add it to PATH now? [y/N]: ")
					}
				}
				var resp string
				_, _ = fmt.Scanln(&resp)
				if resp == "y" || resp == "Y" {
					opts.AddToPath = true
				}
			} else if addFlagSet {
				opts.AddToPath = true
			}
		}
		if !opts.Yes {
			fmt.Print("Proceed? [y/N]: ")
			var resp string
			_, _ = fmt.Scanln(&resp)
			if resp != "y" && resp != "Y" {
				fmt.Println("aborted")
				return nil
			}
		}

		if _, err := install.ExecuteInstall(opts); err != nil {
			return err
		}
		fmt.Println("install completed")
		return nil
	},
}

func init() {
	installCmd.Flags().BoolP("user", "u", true, "Install into user-local bin (default)")
	installCmd.Flags().Bool("system", false, "Install system-wide (requires admin)")
	installCmd.Flags().String("path", "", "Custom target directory for the binary")
	installCmd.Flags().String("from", "", "Source binary path (default is the running executable)")
	installCmd.Flags().BoolP("dry-run", "n", false, "Show actions but do not perform them")
	installCmd.Flags().Bool("check", false, "Only check/installability (no changes)")
	installCmd.Flags().Bool("yes", false, "Assume yes for prompts (use with caution)")
	installCmd.Flags().Bool("add-to-path", false, "Automatically add target dir to PATH (with confirmation)")
	rootCmd.AddCommand(installCmd)
}
