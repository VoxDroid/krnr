package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback <name> --version <n>",
	Short: "Rollback a command set to a previous version",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		vnum, _ := cmd.Flags().GetInt("version")
		if vnum <= 0 {
			return fmt.Errorf("--version must be a positive integer")
		}
		dbConn, err := db.InitDB()
		if err != nil {
			return err
		}
		defer func() { _ = dbConn.Close() }()

		r := registry.NewRepository(dbConn)
		if err := r.ApplyVersionByName(name, vnum); err != nil {
			return err
		}
		fmt.Printf("rolled back %s to v%d\n", name, vnum)
		return nil
	},
}

func init() {
	rollbackCmd.Flags().Int("version", 0, "Version number to rollback to")
	rootCmd.AddCommand(rollbackCmd)
}
