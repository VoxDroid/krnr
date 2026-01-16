package ui

import (
	"context"
	"testing"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestMenuUninstallConfirmation(t *testing.T) {
	reg := &replaceFakeRegistry{items: []adapters.CommandSetSummary{{Name: "one"}}}
	fi := &fakeInstaller{}
	ui := modelpkg.New(reg, &fakeExec{}, nil, fi)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	m1, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = m1.(*TuiModel)

	// open menu and navigate to Uninstall
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = m2.(*TuiModel)
	for i := 0; i < len(m.menuItems) && m.menuItems[m.menuIndex] != "Uninstall"; i++ {
		m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = m3.(*TuiModel)
	}
	if m.menuItems[m.menuIndex] != "Uninstall" {
		t.Fatalf("Uninstall menu item not found: %v", m.menuItems)
	}
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m4.(*TuiModel)
	if !m.menuInputMode || m.menuAction != "uninstall-confirm" {
		t.Fatalf("expected uninstall-confirm prompt, got %v %q", m.menuInputMode, m.menuAction)
	}
	// answer no
	m.menuInput = "n"
	m5, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m5.(*TuiModel)
	if fi.called {
		t.Fatalf("expected uninstall not to be called on 'n'")
	}
	if !m.logsContains("uninstall aborted") {
		t.Fatalf("expected 'uninstall aborted' log, got %v", m.logs)
	}

	// now confirm yes
	m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = m2.(*TuiModel)
	for i := 0; i < len(m.menuItems) && m.menuItems[m.menuIndex] != "Uninstall"; i++ {
		m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = m3.(*TuiModel)
	}
	m4, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m4.(*TuiModel)
	m.menuInput = "y"
	m6, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m6.(*TuiModel)
	if !fi.called {
		t.Fatalf("expected uninstall to be called on 'y'")
	}
	if !m.logsContains("ok uninstalled") && !m.logsContains("uninstalled") {
		t.Fatalf("expected uninstall action in logs, got %v", m.logs)
	}
}
