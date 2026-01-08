package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/executor"
	"github.com/VoxDroid/krnr/internal/registry"
	"github.com/VoxDroid/krnr/internal/security"
	"github.com/VoxDroid/krnr/internal/utils"
)

var execFactory = func(dry, verbose bool) executor.Runner {
	return executor.New(dry, verbose)
}

var runCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Run a named command set",
	Long:  "Run a named command set. Examples:\n  krnr run hello --confirm\n  krnr run hello --show-stderr --suppress-command",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		dry, _ := cmd.Flags().GetBool("dry-run")
		confirmFlag, _ := cmd.Flags().GetBool("confirm")
		verbose, _ := cmd.Flags().GetBool("verbose")
		force, _ := cmd.Flags().GetBool("force")
		suppress, _ := cmd.Flags().GetBool("suppress-command")
		showStderr, _ := cmd.Flags().GetBool("show-stderr")

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

		// Create executor via factory so tests can inject a fake Runner.
		e := execFactory(dry, verbose)
		// Allow user to override the shell used to execute commands (e.g., pwsh, bash, cmd)
		shellFlag, _ := cmd.Flags().GetString("shell")
		if ex, ok := e.(*executor.Executor); ok {
			ex.Shell = shellFlag
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		for _, c := range cs.Commands {
			// Security: check if command is allowed
			if err := security.CheckAllowed(c.Command); err != nil && !force {
				return fmt.Errorf("refusing to run potentially dangerous command '%s': %v (use --force to override)", c.Command, err)
			}
			if !suppress {
				fmt.Printf("-> %s\n", c.Command)
			}
			stderr := io.Discard
			if showStderr {
				stderr = os.Stderr
			}
			if err := e.Execute(ctx, c.Command, "", os.Stdout, stderr); err != nil {
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
	runCmd.Flags().Bool("force", false, "Override safety checks and force execution")
	runCmd.Flags().Bool("suppress-command", false, "Suppress printing the written command before execution")
	runCmd.Flags().Bool("show-stderr", false, "Show command stderr output instead of omitting it")
	runCmd.Flags().String("shell", "", "Override shell to execute commands (e.g., pwsh, bash, cmd)")
	rootCmd.AddCommand(runCmd)
}
