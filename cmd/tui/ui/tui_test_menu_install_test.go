package ui

import (
	"context"
	"testing"

	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
	"github.com/VoxDroid/krnr/internal/install"
	"github.com/VoxDroid/krnr/internal/tui/adapters"
	tea "github.com/charmbracelet/bubbletea"
)

// fakeInstaller for UI tests
type fakeInstaller struct{
	called bool
	opts   install.Options
}

func (f *fakeInstaller) Install(_ context.Context, opts install.Options) ([]string, error) {
	f.called = true
	f.opts = opts
	return []string{"ok installed"}, nil
}
func (f *fakeInstaller) Uninstall(_ context.Context) ([]string, error) {
	f.called = true
	return []string{"ok uninstalled"}, nil
}

func TestMenuInstallInteractive(t *testing.T) {
	// set up model with a fake registry and fake installer
	reg := &replaceFakeRegistry{items: []adapters.CommandSetSummary{{Name: "one"}}}
	fi := &fakeInstaller{}
	ui := modelpkg.New(reg, &fakeExec{}, nil, fi)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	m1, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = m1.(*TuiModel)

	// open menu and navigate to Install
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = m2.(*TuiModel)
	// move to Install item
	for i := 0; i < len(m.menuItems) && m.menuItems[m.menuIndex] != "Install"; i++ {
		m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = m3.(*TuiModel)
	}
	if m.menuItems[m.menuIndex] != "Install" {
		t.Fatalf("Install menu item not found: %v", m.menuItems)
	}
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m4.(*TuiModel)
	if !m.menuInputMode || m.menuAction != "install-scope" {
		t.Fatalf("expected install-scope prompt, got %v %q", m.menuInputMode, m.menuAction)
	}
	// answer scope user and advance
	m.menuInput = "user"
	m5, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m5.(*TuiModel)
	if m.menuAction != "install-addpath" {
		t.Fatalf("expected install-addpath prompt, got %q", m.menuAction)
	}
	// answer add to path: no
	m.menuInput = "n"
	m6, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m6.(*TuiModel)
	// verify installer was called and logs recorded
	if !fi.called {
		t.Fatalf("expected installer to be called")
	}
	found := false
	for _, l := range m.logs {
		if l == "ok installed" || l == "installed" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected install performed and logged, logs: %v", m.logs)
	}
}
