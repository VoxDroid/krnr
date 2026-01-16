package ui

import (
	"context"
	"strings"
	"testing"

	"os"
	"path/filepath"
	"runtime"

	"github.com/VoxDroid/krnr/internal/install"
	adapters "github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestStatusRefreshAfterInstall(t *testing.T) {
	tmp := t.TempDir()
	// isolate HOME so installs write to a temp directory during tests
	t.Setenv("HOME", tmp)
	// On Windows also set USERPROFILE so os.UserHomeDir picks up the temp dir
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmp)
	}
	// On Windows avoid running setx / PowerShell, but still allow metadata saving
	t.Setenv("KRNR_TEST_NO_SETX", "1")
	reg := &replaceFakeRegistry{items: []adapters.CommandSetSummary{{Name: "one", Description: "First"}}}
	inst := adapters.NewInstallerAdapter()
	ui := modelpkg.New(reg, &fakeExec{}, nil, inst)
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
	// check that the binary exists in the user bin (install performed)
	binName := "krnr"
	if runtime.GOOS == "windows" {
		binName = "krnr.exe"
	}
	targetPath := filepath.Join(install.DefaultUserBin(), binName)
	if _, err := os.Stat(targetPath); err != nil {
		t.Fatalf("expected installed binary at %s, stat failed: %v", targetPath, err)
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
	// Also assert the binary exists at the expected path; this ensures ExecuteInstall wrote files
	if _, err := os.Stat(targetPath); err != nil {
		t.Fatalf("expected installed binary still present at %s, err=%v", targetPath, err)
	}
}
