package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage tags for command sets",
	Long:  "Manage tags for command sets: add, remove, list",
}

var tagAddCmd = &cobra.Command{
	Use:   "add <set-name> <tag>",
	Short: "Add a tag to a command set",
	Args:  cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		name := args[0]
		tag := args[1]

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
		if err := r.AddTagToCommandSet(cs.ID, tag); err != nil {
			return err
		}
		fmt.Printf("added tag '%s' to '%s'\n", tag, name)
		return nil
	},
}

var tagRemoveCmd = &cobra.Command{
	Use:   "remove <set-name> <tag>",
	Short: "Remove a tag from a command set",
	Args:  cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		name := args[0]
		tag := args[1]

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
		if err := r.RemoveTagFromCommandSet(cs.ID, tag); err != nil {
			return err
		}
		fmt.Printf("removed tag '%s' from '%s'\n", tag, name)
		return nil
	},
}

var tagListCmd = &cobra.Command{
	Use:   "list <set-name>",
	Short: "List tags for a command set",
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
		tags, err := r.ListTagsForCommandSet(cs.ID)
		if err != nil {
			return err
		}
		for _, t := range tags {
			fmt.Printf("- %s\n", t)
		}
		return nil
	},
}

func init() {
	tagCmd.AddCommand(tagAddCmd)
	tagCmd.AddCommand(tagRemoveCmd)
	tagCmd.AddCommand(tagListCmd)
	rootCmd.AddCommand(tagCmd)
}
