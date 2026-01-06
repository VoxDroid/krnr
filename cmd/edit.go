package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
	"github.com/VoxDroid/krnr/internal/utils"
)

var editCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit a command set",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cmdsFlags, _ := cmd.Flags().GetStringSlice("command")

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

		// Non-interactive: if -c flags provided, replace commands directly
		if len(cmdsFlags) > 0 {
			if err := r.ReplaceCommands(cs.ID, cmdsFlags); err != nil {
				return err
			}
			fmt.Printf("updated '%s' with %d commands\n", name, len(cmdsFlags))
			return nil
		}

		// Interactive: write commands to temp file and open editor
		tmpf, err := ioutil.TempFile("", "krnr-edit-*.txt")
		if err != nil {
			return err
		}
		defer os.Remove(tmpf.Name())

		w := bufio.NewWriter(tmpf)
		for _, c := range cs.Commands {
			w.WriteString(c.Command + "\n")
		}
		w.Flush()
		tmpf.Close()

		if err := utils.OpenEditor(tmpf.Name()); err != nil {
			return err
		}

		// Read back file and parse non-empty lines
		b, err := ioutil.ReadFile(tmpf.Name())
		if err != nil {
			return err
		}
		lines := []string{}
		s := bufio.NewScanner(strings.NewReader(string(b)))
		for s.Scan() {
			line := strings.TrimSpace(s.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			lines = append(lines, line)
		}
		if err := s.Err(); err != nil {
			return err
		}
		if err := r.ReplaceCommands(cs.ID, lines); err != nil {
			return err
		}
		fmt.Printf("updated '%s' with %d commands\n", name, len(lines))
		return nil
	},
}

func init() {
	editCmd.Flags().StringSliceP("command", "c", []string{}, "Replace commands non-interactively (use multiple times)")
	rootCmd.AddCommand(editCmd)
}
