package ui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestMenuExportFallbackInvalidPath(t *testing.T) {
	reg := &replaceFakeRegistry{items: []adapters.CommandSetSummary{{Name: "one", Description: "First"}}, full: adapters.CommandSetSummary{Name: "one", Commands: []string{"echo hi"}}}
	imp := &fakeImpExp{}
	ui := modelpkg.New(reg, &fakeExec{}, imp, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	m1, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = m1.(*TuiModel)
	// open menu
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = m2.(*TuiModel)
	if !m.showMenu {
		t.Fatalf("expected menu to be open")
	}
	// select Export database to enter input mode
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m3.(*TuiModel)
	if !m.menuInputMode || m.menuAction != "export-db" {
		t.Fatalf("expected export prompt mode after selecting Export database, got menuInputMode=%v action=%q", m.menuInputMode, m.menuAction)
	}
	// set an invalid path (non-existent parent)
	invalid := filepath.Join(os.TempDir(), "no-such-dir-hopefully-not-exists", "out.db")
	m.menuInput = invalid
	// confirm
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m4.(*TuiModel)
	if imp.lastDest == "" {
		t.Fatalf("expected destination to be set after fallback, got empty")
	}
	if !strings.Contains(imp.lastDest, filepath.Base(invalid)) {
		t.Fatalf("expected fallback to include base filename, got %q", imp.lastDest)
	}
}
