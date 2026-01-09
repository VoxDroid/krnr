package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/recorder"
	"github.com/VoxDroid/krnr/internal/registry"
)

var recordCmd = &cobra.Command{
	Use:   "record <name>",
	Short: "Record commands interactively into a new command set",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		descFlag, _ := cmd.Flags().GetString("description")
		var desc *string
		if descFlag != "" {
			desc = &descFlag
		}

		// Initialize DB and repository early so we can validate the provided name
		dbConn, err := db.InitDB()
		if err != nil {
			return err
		}
		defer func() { _ = dbConn.Close() }()

		r := registry.NewRepository(dbConn)

		// Use a single buffered reader for all interactive input so buffered data isn't lost
		rdr := bufio.NewReader(cmd.InOrStdin())

		// If the provided name already exists, warn and reprompt the user for a different name.
		// Read the replacement name from the same input stream so tests can script it.
		for {
			existing, err := r.GetCommandSetByName(name)
			if err != nil {
				return err
			}
			if existing == nil {
				break
			}
			fmt.Fprintf(cmd.OutOrStdout(), "name '%s' already exists; enter a new name: ", name)
			newNameRaw, err := rdr.ReadString('\n')
			if err != nil {
				return fmt.Errorf("read new name: %w", err)
			}
			newName := strings.TrimSpace(newNameRaw)
			if newName == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "name cannot be empty")
				name = ""
				continue
			}
			name = newName
		}

		fmt.Println("Enter commands, one per line. End with EOF (Ctrl-D on Unix, Ctrl-Z on Windows).")

		// read from the command input (stdin by default, overridable in tests)
		cmds, err := recorder.RecordCommands(rdr)
		if err != nil {
			return err
		}
		if len(cmds) == 0 {
			fmt.Println("no commands recorded; aborting")
			return nil
		}

		if _, err := recorder.SaveRecorded(r, name, desc, cmds); err != nil {
			return err
		}
		fmt.Printf("saved '%s' with %d commands\n", name, len(cmds))
		return nil
	},
}

func init() {
	recordCmd.Flags().StringP("description", "d", "", "Description for the recorded command set")
	rootCmd.AddCommand(recordCmd)
}
