package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
)

var describeCmd = &cobra.Command{
	Use:   "describe <name>",
	Short: "Show details for a command set",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		name := args[0]
		dbConn, err := db.InitDB()
		if err != nil {
			return err
		}
		defer func() { _ = dbConn.Close() }()

		r := registry.NewRepository(dbConn)
		cs, err := r.GetCommandSetByName(name)
		if err != nil {
			return err
		}
		if cs == nil {
			return fmt.Errorf("command set not found: %s", name)
		}
		fmt.Printf("Name: %s\n", cs.Name)
		if cs.Description.Valid {
			fmt.Printf("Description: %s\n", cs.Description.String)
		}
		fmt.Printf("Created: %s\n", cs.CreatedAt)
		fmt.Println("Commands:")
		for _, c := range cs.Commands {
			fmt.Printf("%d: %s\n", c.Position, c.Command)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(describeCmd)
}
