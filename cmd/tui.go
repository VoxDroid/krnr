package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/VoxDroid/krnr/cmd/tui/ui"
	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/executor"
	"github.com/VoxDroid/krnr/internal/registry"
	"github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Start interactive Terminal UI (initial release v1.2.0)",
	RunE: func(_ *cobra.Command, _ []string) error {
		// Init DB
		dbConn, err := db.InitDB()
		if err != nil {
			return err
		}
		defer func() { _ = dbConn.Close() }()

		ctx := context.Background()

		r := registry.NewRepository(dbConn)
		regAdapter := adapters.NewRegistryAdapter(r)
		runner := executor.New(false, false)
		execAdapter := adapters.NewExecutorAdapter(runner)
		impExpAdapter := adapters.NewImportExportAdapter(dbConn)
		installer := adapters.NewInstallerAdapter()

		uiModel := modelpkg.New(regAdapter, execAdapter, impExpAdapter, installer)
		if err := uiModel.RefreshList(ctx); err != nil {
			return err
		}

		p := ui.NewProgram(uiModel)
		_, err = p.Run()
		return err
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

// The Bubble Tea UI has been moved to `cmd/tui/ui` to keep UI
// implementation and tests centralized. See that package for the
// list, detail, and run modal implementation.
