package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/exporter"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export database or command sets to portable files",
	RunE: func(cmd *cobra.Command, _ []string) error {
		// Interactive mode when invoked without subcommands
		rdr := bufio.NewReader(cmd.InOrStdin())
		cmd.Println("Select export type:\n  1) db\n  2) set")
		cmd.Print("Enter choice [1/2]: ")
		choiceRaw, err := rdr.ReadString('\n')
		if err != nil {
			return err
		}
		choice := strings.TrimSpace(choiceRaw)
		switch choice {
		case "1":
			cmd.Print("Destination path (leave empty for default): ")
			dstRaw, err := rdr.ReadString('\n')
			if err != nil {
				return err
			}
			dst := strings.TrimSpace(dstRaw)
			// default behavior handled by exportDbCmd when dst empty
			if dst == "" {
				date := time.Now().UTC().Format("2006-01-02")
				base := fmt.Sprintf("krnr-%s.db", date)
				dst = filepath.Join(".", base)
				si := 1
				for {
					if _, err := os.Stat(dst); os.IsNotExist(err) {
						break
					}
					dst = filepath.Join(".", fmt.Sprintf("krnr-%s-%d.db", date, si))
					si++
				}
			}
			// ensure DB is reachable
			dbConn, err := db.InitDB()
			if err != nil {
				return err
			}
			_ = dbConn.Close()
			if err := exporter.ExportDatabase(dst); err != nil {
				return err
			}
			cmd.Printf("exported database to %s\n", dst)
			return nil
		case "2":
			cmd.Print("Name of set to export: ")
			nameRaw, err := rdr.ReadString('\n')
			if err != nil {
				return err
			}
			name := strings.TrimSpace(nameRaw)
			if name == "" {
				return fmt.Errorf("set name cannot be empty")
			}
			cmd.Print("Destination path: ")
			dstRaw, err := rdr.ReadString('\n')
			if err != nil {
				return err
			}
			dst := strings.TrimSpace(dstRaw)
			if dst == "" {
				return fmt.Errorf("destination required")
			}
			dbConn, err := db.InitDB()
			if err != nil {
				return err
			}
			defer func() { _ = dbConn.Close() }()
			if err := exporter.ExportCommandSet(dbConn, name, dst); err != nil {
				return err
			}
			cmd.Printf("exported command set '%s' to %s\n", name, dst)
			return nil
		default:
			return fmt.Errorf("invalid choice: %s", choice)
		}
	},
}

var exportDbCmd = &cobra.Command{
	Use:   "db --dst <path>",
	Short: "Export the active database to a file",
	RunE: func(cmd *cobra.Command, _ []string) error {
		dst, _ := cmd.Flags().GetString("dst")
		// default destination when not provided: ./krnr-YYYY-MM-DD.db (avoid overwrite by suffixing)
		if dst == "" {
			date := time.Now().UTC().Format("2006-01-02")
			base := fmt.Sprintf("krnr-%s.db", date)
			dst = filepath.Join(".", base)
			// avoid overwriting an existing file by appending -N suffix
			si := 1
			for {
				if _, err := os.Stat(dst); os.IsNotExist(err) {
					break
				}
				dst = filepath.Join(".", fmt.Sprintf("krnr-%s-%d.db", date, si))
				si++
			}
		}
		// ensure DB is reachable (exporter will checkpoint itself)
		dbConn, err := db.InitDB()
		if err != nil {
			return err
		}
		_ = dbConn.Close()
		if err := exporter.ExportDatabase(dst); err != nil {
			return err
		}
		fmt.Printf("exported database to %s\n", dst)
		return nil
	},
}

var exportSetCmd = &cobra.Command{
	Use:   "set <name> --dst <path>",
	Short: "Export a single named command set to a standalone SQLite file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		dst, _ := cmd.Flags().GetString("dst")
		if dst == "" {
			return fmt.Errorf("--dst is required")
		}
		dbConn, err := db.InitDB()
		if err != nil {
			return err
		}
		defer func() { _ = dbConn.Close() }()
		if err := exporter.ExportCommandSet(dbConn, name, dst); err != nil {
			return err
		}
		fmt.Printf("exported command set '%s' to %s\n", name, dst)
		return nil
	},
}

func init() {
	exportDbCmd.Flags().String("dst", "", "Destination file path for exported DB (required)")
	exportSetCmd.Flags().String("dst", "", "Destination file path for exported command set (required)")
	exportCmd.AddCommand(exportDbCmd)
	exportCmd.AddCommand(exportSetCmd)
	rootCmd.AddCommand(exportCmd)
}
