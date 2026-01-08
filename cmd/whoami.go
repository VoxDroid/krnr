package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/VoxDroid/krnr/internal/user"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Manage stored author identity",
	Long:  "Manage a persisted author identity used by `krnr save` as a default author.",
}

var whoamiSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set stored author identity",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		email, _ := cmd.Flags().GetString("email")
		if name == "" {
			return fmt.Errorf("--name is required")
		}
		if err := user.SetProfile(user.Profile{Name: name, Email: email}); err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "stored author as: %s <%s>\n", name, email)
		return nil
	},
}

var whoamiClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear stored author identity",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := user.ClearProfile(); err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		fmt.Fprintln(out, "cleared stored author identity")
		return nil
	},
}

var whoamiShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show stored author identity",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, ok, err := user.GetProfile()
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		if !ok {
			fmt.Fprintln(out, "no stored author identity")
			return nil
		}
		fmt.Fprintf(out, "%s <%s>\n", p.Name, p.Email)
		return nil
	},
}

func init() {
	whoamiSetCmd.Flags().StringP("name", "n", "", "Author name (required)")
	whoamiSetCmd.Flags().StringP("email", "e", "", "Author email (optional)")
	whoamiCmd.AddCommand(whoamiSetCmd)
	whoamiCmd.AddCommand(whoamiClearCmd)
	whoamiCmd.AddCommand(whoamiShowCmd)
	rootCmd.AddCommand(whoamiCmd)
}
