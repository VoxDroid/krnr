package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
	interactive "github.com/VoxDroid/krnr/internal/utils"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a command set",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		yesFlag, _ := cmd.Flags().GetBool("yes")

		dbConn, err := db.InitDB()
		if err != nil {
			return err
		}
		defer func() { _ = dbConn.Close() }()

		r := registry.NewRepository(dbConn)
		if !yesFlag {
			// prompt interactively; read from the command's input so tests can script it
			if !interactive.ConfirmReader(fmt.Sprintf("Delete '%s' permanently?", name), cmd.InOrStdin()) {
				fmt.Println("aborted")
				return nil
			}
		}
		if err := r.DeleteCommandSet(name); err != nil {
			return err
		}
		fmt.Printf("deleted '%s'\n", name)
		return nil
	},
}

func init() {
	deleteCmd.Flags().Bool("yes", false, "Assume yes for prompts")
	rootCmd.AddCommand(deleteCmd)
}
