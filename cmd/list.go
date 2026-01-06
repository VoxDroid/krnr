package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved command sets",
	Long:  "List saved command sets. Example:\n  krnr list",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbConn, err := db.InitDB()
		if err != nil {
			return err
		}
		defer dbConn.Close()

		r := registry.NewRepository(dbConn)
		sets, err := r.ListCommandSets()
		if err != nil {
			return err
		}
		for _, s := range sets {
			fmt.Printf("- %s\n", s.Name)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
