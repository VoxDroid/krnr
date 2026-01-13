package ui

import (
	"testing"

	"context"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestThemeToggle(t *testing.T) {
	fakeReg := &fakeRegistry{items: []adapters.CommandSetSummary{{Name: "one", Description: "First"}}}
	ui := modelpkg.New(fakeReg, nil, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	// default is off
	if m.themeHighContrast {
		t.Fatalf("expected theme default off")
	}
	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	m = m1.(*TuiModel)
	if !m.themeHighContrast {
		t.Fatalf("expected theme toggled on")
	}
}
