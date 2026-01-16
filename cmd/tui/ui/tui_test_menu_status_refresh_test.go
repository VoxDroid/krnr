package ui

import (
	"context"
	"strings"
	"testing"

	adapters "github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestStatusRefreshAfterInstall(t *testing.T) {
	reg := &replaceFakeRegistry{items: []adapters.CommandSetSummary{{Name: "one", Description: "First"}}}
	fi := &fakeInstaller{}
	ui := modelpkg.New(reg, &fakeExec{}, nil, fi)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	// First: open menu and select Status
	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = m1.(*TuiModel)
	for m.menuItems[m.menuIndex] != "Status" {
		m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = m2.(*TuiModel)
	}
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m3.(*TuiModel)
	firstView := m.vp.View()
	if !strings.Contains(firstView, "krnr status:") {
		t.Fatalf("expected initial status, got:\n%s", firstView)
	}
	// Now perform Install (user, add-to-path 'n') via menu
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = m4.(*TuiModel)
	for m.menuItems[m.menuIndex] != "Install" {
		m5, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = m5.(*TuiModel)
	}
	m6, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m6.(*TuiModel)
	// respond to scope prompt
	m.menuInput = "user"
	m7, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m7.(*TuiModel)
	// respond to add-to-path: yes
	m.menuInput = "y"
	m8, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m8.(*TuiModel)
	if !fi.called {
		t.Fatalf("expected installer to have been called during menu install")
	}
	// Now check Status again
	m9, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = m9.(*TuiModel)
	for m.menuItems[m.menuIndex] != "Status" {
		m10, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = m10.(*TuiModel)
	}
	m11, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m11.(*TuiModel)
	secondView := m.vp.View()
	if !strings.Contains(secondView, "User: installed=true") {
		t.Fatalf("expected status to reflect installed=true, got:\n%s", secondView)
	}
	// Ensure only one 'krnr status:' header (not stacked duplicates)
	headerCount := strings.Count(secondView, "krnr status:")
	if headerCount != 1 {
		t.Fatalf("expected a single status header, got %d occurrences; view:\n%s", headerCount, secondView)
	}
}
