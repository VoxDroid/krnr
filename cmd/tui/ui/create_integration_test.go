package ui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
	"github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
)

func TestCreateEntryNoDuplicateVersions(t *testing.T) {
	// real DB-backed repository
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()
	r := registry.NewRepository(dbConn)
	regAdapter := adapters.NewRegistryAdapter(r)
	ui := modelpkg.New(regAdapter, nil, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()

	// open create modal
	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = m1.(*TuiModel)
	// fill Name: Test1
	for _, rch := range []rune{'T', 'e', 's', 't', '1'} {
		m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rch}})
		m = m2.(*TuiModel)
	}
	// tab to commands
	for i := 0; i < 10 && m.editor.field != 5; i++ {
		m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = m3.(*TuiModel)
	}
	// add a command and type 'echo hi'
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	m = m4.(*TuiModel)
	for _, rch := range []rune{'e', 'c', 'h', 'o', ' ', 'h', 'i'} {
		m5, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rch}})
		m = m5.(*TuiModel)
	}
	// save with Ctrl+S (this schedules the delayed save as well)
	m6, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = m6.(*TuiModel)

	// simulate delayed save tick that would normally arrive via tea.Tick
	m7, _ := m.Update(saveNowMsg{})
	m = m7.(*TuiModel)

	// ensure one version exists and it's a create
	vers, err := ui.ListVersions(context.Background(), "Test1")
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	if len(vers) != 1 {
		t.Fatalf("expected exactly 1 version after create, got %d", len(vers))
	}
	if vers[0].Operation != "create" {
		t.Fatalf("expected operation 'create', got %s", vers[0].Operation)
	}

	// cleanup
	_ = r.DeleteCommandSet("Test1")
}
