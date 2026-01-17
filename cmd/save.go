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
	Long: `Save a named command set. Examples:
  krnr save hello -d 'say hi' -c 'echo Hello' -c 'echo World'

Note: When using Windows shells, be sure to properly quote embedded double-quotes; e.g., in PowerShell use -c 'systeminfo | findstr /C:"OS Name" /C:"OS Version"'`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		desc, _ := cmd.Flags().GetString("description")
		cmds, _ := cmd.Flags().GetStringSlice("command")
		// If the user accidentally allowed the shell to split a quoted
		// command into multiple positionals, join the remaining args into a
		// single command. If exactly one `-c` was provided, merge the
		// leftover tokens into that command (this covers the common case of
		// an embedded quoted substring getting split). If multiple `-c`
		// flags were provided, append the joined tokens as an additional
		// command.
		if len(args) > 1 {
			joined := strings.Join(args[1:], " ")
			if len(cmds) == 0 {
				cmd.PrintErrf("warning: detected unquoted command tokens; using joined command: %q\n", joined)
				cmds = append(cmds, joined)
			} else if len(cmds) == 1 {
				// Merge into the single provided command instead of creating a new
				// command so inputs like `-c "systeminfo | findstr /C:\"OS Name\" /C:\"OS Version\""`
				// that were split by the shell become a single preserved command.
				cmd.PrintErrf("warning: detected unquoted command tokens; merging into provided command: %q\n", joined)
				merged := strings.TrimSpace(cmds[0] + " " + joined)
				// Try to heuristically restore missing quotes for common patterns
				// like `findstr /C:OS Name /C:OS Version` -> `/C:"OS Name" /C:"OS Version"`.
				merged = normalizeFindstrCArgs(merged)
				cmds[0] = merged
			} else {
				cmd.PrintErrf("warning: detected unquoted command tokens; appending joined command: %q\n", joined)
				cmds = append(cmds, joined)
			}
		}

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

		if _, err := r.CreateCommandSet(name, &desc, authorNamePtr, authorEmailPtr, cmds); err != nil {
			return err
		}

		fmt.Printf("saved '%s' with %d commands\n", name, len(cmds))
		return nil
	},
}

func quoteIfNeededForFindstr(arg string) string {
	if arg == "" {
		return arg
	}
	if strings.HasPrefix(arg, "\"") || strings.HasPrefix(arg, "'") {
		return arg
	}
	if strings.ContainsAny(arg, " \t") {
		return "\"" + arg + "\""
	}
	return arg
}

func normalizeFindstrCArgs(s string) string {
	// Heuristic: only try this for strings that look like findstr usage
	low := strings.ToLower(s)
	if !strings.Contains(low, "findstr") || !strings.Contains(s, "/C:") {
		return s
	}
	parts := strings.Split(s, "/C:")
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		seg := parts[i]
		trimmed := strings.TrimSpace(seg)
		if trimmed == "" {
			// keep as-is (no argument)
			if len(out) > 0 && out[len(out)-1] != ' ' {
				out += " "
			}
			out += "/C:"
			continue
		}
		// if trimmed includes a space or is already quoted, quote appropriately
		arg := trimmed
		// If the trimmed segment includes further /C: it means the rest
		// contains additional /C: occurrences; only consider the portion
		// up to the next /C: (we already split, so it's safe).
		arg = quoteIfNeededForFindstr(arg)
		if len(out) > 0 && out[len(out)-1] != ' ' {
			out += " "
		}
		out += "/C:" + arg
	}
	return out
}

func init() {
	saveCmd.Flags().StringP("description", "d", "", "Description for the command set")
	saveCmd.Flags().StringSliceP("command", "c", []string{}, "Command to add to the set (can be repeated)")
	saveCmd.Flags().StringP("author", "a", "", "Author name for this command set (overrides stored whoami)")
	saveCmd.Flags().StringP("author-email", "e", "", "Author email for this command set (optional)")
	rootCmd.AddCommand(saveCmd)
}
