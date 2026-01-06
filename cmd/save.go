package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
)

var saveCmd = &cobra.Command{
	Use:   "save <name>",
	Short: "Save a named command set",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		desc, _ := cmd.Flags().GetString("description")
		cmds, _ := cmd.Flags().GetStringSlice("command")

		dbConn, err := db.InitDB()
		if err != nil {
			return err
		}
		defer dbConn.Close()

		r := registry.NewRepository(dbConn)
		id, err := r.CreateCommandSet(name, &desc)
		if err != nil {
			return err
		}

		for i, c := range cmds {
			if _, err := r.AddCommand(id, i+1, c); err != nil {
				return err
			}
		}

		fmt.Printf("saved '%s' with %d commands\n", name, len(cmds))
		return nil
	},
}

func init() {
	saveCmd.Flags().StringP("description", "d", "", "Description for the command set")
	saveCmd.Flags().StringSliceP("command", "c", []string{}, "Command to add to the set (can be repeated)")
	rootCmd.AddCommand(saveCmd)
}
