package ui

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
	"github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
	tea "github.com/charmbracelet/bubbletea"
)

func createAndPopulateSrcDB(t *testing.T, src string) {
	srcConn, err := sql.Open("sqlite", src)
	if err != nil {
		t.Fatalf("open src db: %v", err)
	}
	defer func() { _ = srcConn.Close() }()
	if err := db.ApplyMigrations(srcConn); err != nil {
		t.Fatalf("apply migrations src: %v", err)
	}
	srcRepo := registry.NewRepository(srcConn)
	if _, err := srcRepo.CreateCommandSet("db-imported", nil, nil, nil, []string{"echo hi"}); err != nil {
		t.Fatalf("create command set src: %v", err)
	}
}

func performImportDBOverwriteRealDB(t *testing.T, m *TuiModel, src string) {
	m = selectMenuItem(t, m, "Import database")
	if !m.menuInputMode || m.menuAction != "import-db" {
		t.Fatalf("expected import-db prompt mode, got %v %q", m.menuInputMode, m.menuAction)
	}
	m.menuInput = src
	m5, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = m5
	if m.menuAction != "import-db-overwrite" {
		t.Fatalf("expected overwrite prompt stage, got %q", m.menuAction)
	}
	m.menuInput = "y"
	m6, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = m6
}

func TestMenuImportDatabaseOverwrite_RealDB(t *testing.T) {
	// sandbox DB path
	tmp := t.TempDir()
	dst := filepath.Join(tmp, "krnr.db")
	src := filepath.Join(tmp, "src.db")
	_ = os.Setenv("KRNR_DB", dst)
	defer func() { _ = os.Unsetenv("KRNR_DB") }()

	// create source DB and populate a command set
	createAndPopulateSrcDB(t, src)

	// ensure active DB exists and is open
	activeConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("init active db: %v", err)
	}
	defer func() { _ = activeConn.Close() }()

	// construct real adapters and model
	r := registry.NewRepository(activeConn)
	regAdapter := adapters.NewRegistryAdapter(r)
	exec := &fakeExec{}
	imp := adapters.NewImportExportAdapter(activeConn)
	ui := modelpkg.New(regAdapter, exec, imp, nil)
	_ = ui.RefreshList(context.Background())
	defer func() { _ = ui.Close() }()

	m := NewModel(ui)
	m = initTestModel(m)
	m1, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = m1.(*TuiModel)

	performImportDBOverwriteRealDB(t, m, src)

	found := false
	for _, it := range m.list.Items() {
		if ci, ok := it.(csItem); ok {
			if ci.cs.Name == "db-imported" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("expected list to include imported DB item 'db-imported', got %v", m.list.Items())
	}
}
