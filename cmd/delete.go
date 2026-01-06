package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/drei/krnr/internal/db"
	"github.com/drei/krnr/internal/registry"
	"github.com/drei/krnr/internal/utils"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a command set",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		confirmFlag, _ := cmd.Flags().GetBool("confirm")

		dbConn, err := db.InitDB()
		if err != nil {
			return err
		}
		defer dbConn.Close()

		r := registry.NewRepository(dbConn)
		if confirmFlag {
			if !utils.Confirm(fmt.Sprintf("Delete '%s' permanently?", name)) {
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
	deleteCmd.Flags().Bool("confirm", false, "Ask for confirmation")
	rootCmd.AddCommand(deleteCmd)
}
