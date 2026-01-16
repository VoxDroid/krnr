package ui

import (
	"context"
	"strings"
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
	_, _ = m.Update(saveNowMsg{})

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

	// now test tag-based search against DB-backed cache
	// find the created command set id and add tag 'whathe' to it
	cs, err := r.GetCommandSetByName("Test1")
	if err != nil || cs == nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if err := r.AddTagToCommandSet(cs.ID, "whathe"); err != nil {
		t.Fatalf("AddTagToCommandSet: %v", err)
	}
	// refresh UI cache and initialize list state
	_ = ui.RefreshList(context.Background())
	m2 := NewModel(ui)
	m2.Init()()
	// enter filter mode and type '#whathe'
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m2 = m3.(*TuiModel)
	for _, ch := range []rune{'#', 'w', 'h', 'a', 't', 'h', 'e'} {
		m4, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m2 = m4.(*TuiModel)
	}
	view := m2.list.View()
	if !strings.Contains(view, "Test1") {
		t.Fatalf("expected '#whathe' to match 'Test1' from DB, got:\n%s", view)
	}

	// cleanup
	_ = r.DeleteCommandSet("Test1")
}
