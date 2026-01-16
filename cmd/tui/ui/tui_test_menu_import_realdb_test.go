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

func TestMenuImportDatabaseOverwrite_RealDB(t *testing.T) {
	// sandbox DB path
	tmp := t.TempDir()
	dst := filepath.Join(tmp, "krnr.db")
	src := filepath.Join(tmp, "src.db")
	_ = os.Setenv("KRNR_DB", dst)
	defer os.Unsetenv("KRNR_DB")

	// create source DB and populate a command set
	srcConn, err := sql.Open("sqlite", src)
	if err != nil {
		t.Fatalf("open src db: %v", err)
	}
	defer srcConn.Close()
	if err := db.ApplyMigrations(srcConn); err != nil {
		t.Fatalf("apply migrations src: %v", err)
	}
	srcRepo := registry.NewRepository(srcConn)
	if _, err := srcRepo.CreateCommandSet("db-imported", nil, nil, nil, []string{"echo hi"}); err != nil {
		t.Fatalf("create command set src: %v", err)
	}

	// ensure active DB exists and is open
	activeConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("init active db: %v", err)
	}
	defer activeConn.Close()

	// construct real adapters and model
	r := registry.NewRepository(activeConn)
	regAdapter := adapters.NewRegistryAdapter(r)
	exec := &fakeExec{}
	imp := adapters.NewImportExportAdapter(activeConn)
	ui := modelpkg.New(regAdapter, exec, imp, nil)
	_ = ui.RefreshList(context.Background())
	defer func() { _ = ui.Close() }()

	m := NewModel(ui)
	m.Init()()
	m1, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = m1.(*TuiModel)

	// open menu and select Import database (move down once)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = m2.(*TuiModel)
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = m3.(*TuiModel)
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m4.(*TuiModel)
	if !m.menuInputMode || m.menuAction != "import-db" {
		t.Fatalf("expected import-db prompt mode, got %v %q", m.menuInputMode, m.menuAction)
	}

	// set path and confirm -> advances to overwrite prompt
	m.menuInput = src
	m5, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m5.(*TuiModel)
	if m.menuAction != "import-db-overwrite" {
		t.Fatalf("expected overwrite prompt stage, got %q", m.menuAction)
	}
	// answer overwrite: yes
	m.menuInput = "y"
	m6, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m6.(*TuiModel)

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
