package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/importer"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import a database file or exported command set into the active environment",
	RunE: func(cmd *cobra.Command, _ []string) error {
		// Interactive mode: prompt for type and options when invoked without subcommands
		rdr := bufio.NewReader(cmd.InOrStdin())
		cmd.Println("Select import type:\n  1) db\n  2) set")
		cmd.Print("Enter choice [1/2]: ")
		choiceRaw, err := rdr.ReadString('\n')
		if err != nil {
			return err
		}
		choice := strings.TrimSpace(choiceRaw)
		switch choice {
		case "1":
			// DB import
			cmd.Print("Path to source DB file: ")
			srcRaw, err := rdr.ReadString('\n')
			if err != nil {
				return err
			}
			src := strings.TrimSpace(srcRaw)
			if src == "" {
				return fmt.Errorf("source path cannot be empty")
			}
			cmd.Print("Overwrite destination DB if it exists? [y/N]: ")
			overRaw, err := rdr.ReadString('\n')
			if err != nil {
				return err
			}
			over := strings.ToLower(strings.TrimSpace(overRaw))
			overwrite := over == "y" || over == "yes"
			if err := importer.ImportDatabase(src, overwrite, importer.ImportOptions{}); err != nil {
				return err
			}
			cmd.Printf("imported database from %s\n", src)
			return nil
		case "2":
			// set import
			cmd.Print("Path to exported set file: ")
			srcRaw, err := rdr.ReadString('\n')
			if err != nil {
				return err
			}
			src := strings.TrimSpace(srcRaw)
			if src == "" {
				return fmt.Errorf("source path cannot be empty")
			}
			// Ensure destination DB exists and close to avoid file locks
			dbConn, err := db.InitDB()
			if err != nil {
				return err
			}
			_ = dbConn.Close()
			cmd.Print("On conflict (rename|skip|overwrite|merge) [rename]: ")
			ocRaw, err := rdr.ReadString('\n')
			if err != nil {
				return err
			}
			oc := strings.TrimSpace(ocRaw)
			if oc == "" {
				oc = "rename"
			}
			cmd.Print("Dedupe when merging? [y/N]: ")
			dedRaw, err := rdr.ReadString('\n')
			if err != nil {
				return err
			}
			ded := strings.ToLower(strings.TrimSpace(dedRaw))
			dedupe := ded == "y" || ded == "yes"
			if err := importer.ImportCommandSet(src, importer.ImportOptions{OnConflict: oc, Dedupe: dedupe}); err != nil {
				return err
			}
			cmd.Printf("imported command set(s) from %s\n", src)
			return nil
		default:
			return fmt.Errorf("invalid choice: %s", choice)
		}
	},
}

var importDbCmd = &cobra.Command{
	Use:   "db <file> [--overwrite]",
	Short: "Import an entire DB file as the active database (dangerous)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		src := args[0]
		overwrite, _ := cmd.Flags().GetBool("overwrite")
		onConflict, _ := cmd.Flags().GetString("on-conflict")
		dedupe, _ := cmd.Flags().GetBool("dedupe")
		// validate file exists
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("source DB not found: %w", err)
		}
		if err := importer.ImportDatabase(src, overwrite, importer.ImportOptions{OnConflict: onConflict, Dedupe: dedupe}); err != nil {
			return err
		}
		fmt.Printf("imported database from %s\n", src)
		return nil
	},
}

var importSetCmd = &cobra.Command{
	Use:   "set <file>",
	Short: "Import a exported command set file into the active DB (name collisions are detected and suffixed)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		src := args[0]
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("source file not found: %w", err)
		}
		// Ensure destination DB exists and close the connection immediately to avoid file locks
		dbConn, err := db.InitDB()
		if err != nil {
			return err
		}
		_ = dbConn.Close()
		onConflict, _ := cmd.Flags().GetString("on-conflict")
		dedupe, _ := cmd.Flags().GetBool("dedupe")
		if err := importer.ImportCommandSet(src, importer.ImportOptions{OnConflict: onConflict, Dedupe: dedupe}); err != nil {
			return err
		}
		fmt.Printf("imported command set(s) from %s\n", src)
		return nil
	},
}

func init() {
	importDbCmd.Flags().Bool("overwrite", false, "Overwrite the active database file if it exists")
	importDbCmd.Flags().String("on-conflict", "rename", "Conflict policy when importing into an existing DB: rename|skip|overwrite|merge")
	importDbCmd.Flags().Bool("dedupe", false, "When merging, dedupe identical commands")

	importSetCmd.Flags().String("on-conflict", "rename", "Conflict policy for importing sets: rename|skip|overwrite|merge")
	importSetCmd.Flags().Bool("dedupe", false, "When merging, dedupe identical commands")

	importCmd.AddCommand(importDbCmd)
	importCmd.AddCommand(importSetCmd)
	rootCmd.AddCommand(importCmd)
}
