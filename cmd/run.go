package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/executor"
	"github.com/VoxDroid/krnr/internal/registry"
	"github.com/VoxDroid/krnr/internal/security"
	interactive "github.com/VoxDroid/krnr/internal/utils"
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
		defer func() { _ = dbConn.Close() }()

		r := registry.NewRepository(dbConn)
		cs, err := r.GetCommandSetByName(name)
		if err != nil {
			return err
		}
		if cs == nil {
			return fmt.Errorf("command set not found: %s", name)
		}

		if confirmFlag {
			if !interactive.Confirm(fmt.Sprintf("Run '%s' now?", name)) {
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

// collect parameters from flags
	paramVals, _ := cmd.Flags().GetStringArray("param")
	params := map[string]string{}
	for _, p := range paramVals {
		parts := strings.SplitN(p, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid --param value: %s (expected name=value)", p)
		}
		name := parts[0]
		val := parts[1]
		// env:NAME syntax reads from environment
	if strings.HasPrefix(val, "env:") {
			envKey := strings.TrimPrefix(val, "env:")
			params[name] = os.Getenv(envKey)
		} else {
			params[name] = val
		}
	}

	for _, c := range cs.Commands {
		cmdText := c.Command
		// Apply parameter substitution if present
		if len(registry.FindParams(cmdText)) > 0 {
			// gather missing params and prompt interactively if needed
			required := registry.FindParams(cmdText)
			for _, rname := range required {
				if _, ok := params[rname]; !ok {
					// interactive prompt
					val := interactive.Prompt(fmt.Sprintf("Value for parameter %s", rname))
					if val == "" {
						return fmt.Errorf("missing value for parameter %s", rname)
					}
					params[rname] = val
				}
			}
			sub, err := registry.ApplyParams(cmdText, params)
			if err != nil {
				return err
			}
			cmdText = sub
		}

		// Security: check if command is allowed
		if err := security.CheckAllowed(cmdText); err != nil && !force {
			return fmt.Errorf("refusing to run potentially dangerous command '%s': %v (use --force to override)", cmdText, err)
		}
		if !suppress {
			fmt.Printf("-> %s\n", cmdText)
		}
		stderr := io.Discard
		if showStderr {
			stderr = os.Stderr
		}
		if err := e.Execute(ctx, cmdText, "", os.Stdout, stderr); err != nil {
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
	runCmd.Flags().StringArray("param", []string{}, "Parameter values as name=value (repeatable). Use env:VAR to load from environment, e.g. --param user=env:USER")
	rootCmd.AddCommand(runCmd)
}
