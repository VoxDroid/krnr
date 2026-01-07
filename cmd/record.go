package cmd

import (
	"bufio"
	"fmt"

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

		fmt.Println("Enter commands, one per line. End with EOF (Ctrl-D on Unix, Ctrl-Z on Windows).")

		// read from the command input (stdin by default, overridable in tests)
		rdr := bufio.NewReader(cmd.InOrStdin())
		cmds, err := recorder.RecordCommands(rdr)
		if err != nil {
			return err
		}
		if len(cmds) == 0 {
			fmt.Println("no commands recorded; aborting")
			return nil
		}

		dbConn, err := db.InitDB()
		if err != nil {
			return err
		}
		defer dbConn.Close()

		r := registry.NewRepository(dbConn)
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
