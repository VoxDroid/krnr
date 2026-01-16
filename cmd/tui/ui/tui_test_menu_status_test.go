package ui

import (
	"context"
	"strings"
	"testing"

	adapters "github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestMenuStatusShowsDiagnostics(t *testing.T) {
	reg := &replaceFakeRegistry{items: []adapters.CommandSetSummary{{Name: "one", Description: "First"}}}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	// open menu
	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = m1.(*TuiModel)
	if !m.showMenu {
		t.Fatalf("expected menu to be open")
	}
	// find Status item and select it
	for i := 0; i < len(m.menuItems); i++ {
		if m.menuItems[m.menuIndex] == "Status" {
			break
		}
		m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = m2.(*TuiModel)
	}
	// press Enter to invoke Status
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m3.(*TuiModel)
	// preview (viewport) should contain status heading
	if !strings.Contains(m.vp.View(), "krnr status:") {
		t.Fatalf("expected viewport to show status details, got:\n%s", m.vp.View())
	}
	// detail pane should be shown so user sees status
	if !m.showDetail {
		t.Fatalf("expected detail pane to be shown after Status selection")
	}
}
