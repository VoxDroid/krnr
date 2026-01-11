package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
	"github.com/VoxDroid/krnr/internal/user"
)

var saveCmd = &cobra.Command{
	Use:   "save <name>",
	Short: "Save a named command set",
	Long:  "Save a named command set. Examples:\n  krnr save hello -d 'say hi' -c 'echo Hello' -c 'echo World'",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		desc, _ := cmd.Flags().GetString("description")
		cmds, _ := cmd.Flags().GetStringSlice("command")

		dbConn, err := db.InitDB()
		if err != nil {
			return err
		}
		defer func() { _ = dbConn.Close() }()

		r := registry.NewRepository(dbConn)
		// determine author (flag overrides stored whoami)
		authorFlag, _ := cmd.Flags().GetString("author")
		authorEmailFlag, _ := cmd.Flags().GetString("author-email")
		var authorNamePtr *string
		var authorEmailPtr *string
		if authorFlag != "" {
			authorNamePtr = &authorFlag
			if authorEmailFlag != "" {
				authorEmailPtr = &authorEmailFlag
			}
		} else {
			if p, ok, _ := user.GetProfile(); ok {
				if p.Name != "" {
					authorNamePtr = &p.Name
				}
				if p.Email != "" {
					authorEmailPtr = &p.Email
				}
			}
		}

		// Interactive duplicate name check (mirror record behavior)
		rdr := bufio.NewReader(cmd.InOrStdin())
		for {
			existing, err := r.GetCommandSetByName(name)
			if err != nil {
				return err
			}
			if existing == nil {
				break
			}
			cmd.Printf("name '%s' already exists; enter a new name: ", name)
			newNameRaw, err := rdr.ReadString('\n')
			if err != nil {
				return fmt.Errorf("read new name: %w", err)
			}
			newName := strings.TrimSpace(newNameRaw)
			if newName == "" {
				cmd.Println("name cannot be empty")
				name = ""
				continue
			}
			name = newName
		}

		id, err := r.CreateCommandSet(name, &desc, authorNamePtr, authorEmailPtr)
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
	saveCmd.Flags().StringP("author", "a", "", "Author name for this command set (overrides stored whoami)")
	saveCmd.Flags().StringP("author-email", "e", "", "Author email for this command set (optional)")
	rootCmd.AddCommand(saveCmd)
}
