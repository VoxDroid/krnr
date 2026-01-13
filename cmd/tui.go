package cmd

import (
	"context"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
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
	Short: "Start interactive terminal UI (Bubble Tea prototype)",
	RunE: func(cmd *cobra.Command, _ []string) error {
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

		uiModel := modelpkg.New(regAdapter, execAdapter, impExpAdapter, nil)
		if err := uiModel.RefreshList(ctx); err != nil { return err }

		p := ui.NewProgram(uiModel)
		return p.Start()
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}


// csItem adapts adapters.CommandSetSummary to list.Item
type csItem struct{ cs adapters.CommandSetSummary }

func (c csItem) Title() string { return c.cs.Name }
func (c csItem) Description() string { return c.cs.Description }
func (c csItem) FilterValue() string { return c.cs.Name + " " + c.cs.Description }

// model implements tea.Model
type model struct{
	list       list.Model
	vp         viewport.Model
	width      int
	height     int
	showDetail bool
	detail     string
	filterMode bool
}

// The Bubble Tea UI has been moved to `cmd/tui/ui` to keep UI
// implementation and tests centralized. See that package for the
// list, detail, and run modal implementation.
