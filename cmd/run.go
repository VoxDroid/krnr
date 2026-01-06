package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/drei/krnr/internal/db"
	"github.com/drei/krnr/internal/executor"
	"github.com/drei/krnr/internal/registry"
	"github.com/drei/krnr/internal/utils"
)

var runCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Run a named command set",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		dry, _ := cmd.Flags().GetBool("dry-run")
		confirmFlag, _ := cmd.Flags().GetBool("confirm")
		verbose, _ := cmd.Flags().GetBool("verbose")

		dbConn, err := db.InitDB()
		if err != nil {
			return err
		}
		defer dbConn.Close()

		r := registry.NewRepository(dbConn)
		cs, err := r.GetCommandSetByName(name)
		if err != nil {
			return err
		}
		if cs == nil {
			return fmt.Errorf("command set not found: %s", name)
		}

		if confirmFlag {
			if !utils.Confirm(fmt.Sprintf("Run '%s' now?", name)) {
				fmt.Println("aborted")
				return nil
			}
		}

		e := &executor.Executor{DryRun: dry, Verbose: verbose}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		for _, c := range cs.Commands {
			fmt.Printf("-> %s\n", c.Command)
			if err := e.Execute(ctx, c.Command, "", os.Stdout, io.Discard); err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	runCmd.Flags().Bool("dry-run", false, "Do not actually execute commands")
	runCmd.Flags().Bool("confirm", false, "Ask for confirmation before running")
	runCmd.Flags().Bool("verbose", false, "Verbose output (prints dry-run messages)")
	rootCmd.AddCommand(runCmd)
}
