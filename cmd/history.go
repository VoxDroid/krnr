package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
)

var historyCmd = &cobra.Command{
	Use:   "history <name>",
	Short: "Show version history for a named command set",
	Long:  "Show version history for a named command set (versions, timestamps, author, operation)",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		name := args[0]
		dbConn, err := db.InitDB()
		if err != nil {
			return err
		}
		defer func() { _ = dbConn.Close() }()

		r := registry.NewRepository(dbConn)
		vers, err := r.ListVersionsByName(name)
		if err != nil {
			return err
		}
		if len(vers) == 0 {
			fmt.Printf("no history for %s\n", name)
			return nil
		}
		for _, v := range vers {
			author := ""
			if v.AuthorName.Valid {
				author = v.AuthorName.String
			}
			fmt.Printf("v%d\t%s\t%s\t%s\n", v.Version, v.CreatedAt, v.Operation, author)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)
}
