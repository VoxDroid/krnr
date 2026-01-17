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

func openStatusView(t *testing.T, m *TuiModel) string {
	m = selectMenuItem(t, m, "Status")
	view := m.vp.View()
	if !strings.Contains(view, "krnr status:") {
		t.Fatalf("expected status view to contain header, got:\n%s", view)
	}
	return view
}

func performInstallFromMenu(t *testing.T, m *TuiModel, scope, addPath string) *TuiModel {
	m = selectMenuItem(t, m, "Install")
	m.menuInput = scope
	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m1.(*TuiModel)
	m.menuInput = addPath
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(*TuiModel)
	return m
}

func assertBinaryExists(t *testing.T) {
	binName := "krnr"
	if runtime.GOOS == "windows" {
		binName = "krnr.exe"
	}
	targetPath := filepath.Join(install.DefaultUserBin(), binName)
	if _, err := os.Stat(targetPath); err != nil {
		t.Fatalf("expected installed binary at %s, stat failed: %v", targetPath, err)
	}
}

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

	// ensure status initially renders
	_ = openStatusView(t, m)

	// perform Install (user, add-to-path 'y') via menu and verify binary exists
	m = performInstallFromMenu(t, m, "user", "y")
	assertBinaryExists(t)

	// status should reflect installed=true and not duplicate header
	secondView := openStatusView(t, m)
	if !strings.Contains(secondView, "User: installed=true") {
		t.Fatalf("expected status to reflect installed=true, got:\n%s", secondView)
	}
	headerCount := strings.Count(secondView, "krnr status:")
	if headerCount != 1 {
		t.Fatalf("expected a single status header, got %d occurrences; view:\n%s", headerCount, secondView)
	}
	// Also assert the binary exists at the expected path; this ensures ExecuteInstall wrote files
	assertBinaryExists(t)
}
