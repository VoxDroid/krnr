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
	RunE: func(cmd *cobra.Command, _ []string) error {
		dbConn, err := db.InitDB()
		if err != nil {
			return err
		}
		defer func() { _ = dbConn.Close() }()

		r := registry.NewRepository(dbConn)
		// check flags
		tagFilter, _ := cmd.Flags().GetString("tag")
		textFilter, _ := cmd.Flags().GetString("filter")
		fuzzyFlag, _ := cmd.Flags().GetBool("fuzzy")
		var sets []registry.CommandSet
		if tagFilter != "" {
			sets, err = r.ListCommandSetsByTag(tagFilter)
			if err != nil {
				return err
			}
		} else if textFilter != "" {
			if fuzzyFlag {
				sets, err = r.FuzzySearchCommandSets(textFilter)
				if err != nil {
					return err
				}
			} else {
				sets, err = r.SearchCommandSets(textFilter)
				if err != nil {
					return err
				}
			}
		} else {
			sets, err = r.ListCommandSets()
			if err != nil {
				return err
			}
		}

		for _, s := range sets {
			fmt.Printf("- %s\n", s.Name)
		}
		return nil
	},
}

func init() {
	listCmd.Flags().String("tag", "", "Filter by tag name")
	listCmd.Flags().String("filter", "", "Filter by text search")
	listCmd.Flags().Bool("fuzzy", false, "Enable fuzzy matching for text filter")
	rootCmd.AddCommand(listCmd)
}
