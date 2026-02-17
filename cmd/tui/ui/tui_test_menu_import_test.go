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

func performImportDBOverwrite(t *testing.T, m *TuiModel, imp *fakeImpExp, src string) {
	m = selectMenuItem(t, m, "Import database")
	if !m.menuInputMode || m.menuAction != "import-db" {
		t.Fatalf("expected import-db prompt mode, got %v %q", m.menuInputMode, m.menuAction)
	}
	m.menuInput = src
	m5, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m5.(*TuiModel)
	if m.menuAction != "import-db-overwrite" {
		t.Fatalf("expected overwrite prompt stage, got %q", m.menuAction)
	}
	m.menuInput = "y"
	m6, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m6.(*TuiModel)
	if imp.lastSrc != src || !imp.lastOverwrite {
		t.Fatalf("unexpected import params: %q %v", imp.lastSrc, imp.lastOverwrite)
	}
	if !strings.Contains(m.notification, "imported database") {
		t.Fatalf("expected import notification, got %q", m.notification)
	}
}

func TestMenuImportDatabaseOverwrite(t *testing.T) {
	// setup
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.db")
	_ = os.WriteFile(src, []byte("x"), 0o644)
	reg := &replaceFakeRegistry{items: []adapters.CommandSetSummary{{Name: "one", Description: "First"}}}
	imp := &fakeImpExp{reg: reg}
	ui := modelpkg.New(reg, &fakeExec{}, imp, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m = initTestModel(m)
	m1, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = m1.(*TuiModel)

	performImportDBOverwrite(t, m, imp, src)

	// confirm list reflects DB import
	found := false
	for _, it := range m.list.Items() {
		if ci, ok := it.(csItem); ok {
			if ci.cs.Name == "db-imported" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("expected list to include imported DB item 'db-imported', got %v", m.list.Items())
	}
}

func performImportSetFlow(t *testing.T, m *TuiModel, imp *fakeImpExp, src string) {
	m.menuInput = src
	m6, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m6.(*TuiModel)
	if m.menuAction != "import-set-policy" {
		t.Fatalf("expected import-set-policy stage, got %q", m.menuAction)
	}
	// choose merge policy and dedupe
	m.menuInput = "merge"
	m7, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m7.(*TuiModel)
	if m.menuAction != "import-set-dedupe" {
		t.Fatalf("expected dedupe prompt stage, got %q", m.menuAction)
	}
	m.menuInput = "y"
	m8, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m8.(*TuiModel)
	if imp.lastSrc != src || imp.lastPolicy != "merge" || !imp.lastDedupe {
		t.Fatalf("unexpected import params: %q %q %v", imp.lastSrc, imp.lastPolicy, imp.lastDedupe)
	}
	if !strings.Contains(m.notification, "imported command set") {
		t.Fatalf("expected import notification, got %q", m.notification)
	}
}

func TestMenuImportSetPolicyAndDedupe(t *testing.T) {
	// setup
	tmp := t.TempDir()
	src := filepath.Join(tmp, "set.db")
	_ = os.WriteFile(src, []byte("x"), 0o644)
	reg := &replaceFakeRegistry{items: []adapters.CommandSetSummary{{Name: "one", Description: "First"}}}
	imp := &fakeImpExp{reg: reg}
	ui := modelpkg.New(reg, &fakeExec{}, imp, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m = initTestModel(m)
	m1, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = m1.(*TuiModel)
	// open menu and navigate to Import set
	m = selectMenuItem(t, m, "Import set")
	if !m.menuInputMode || m.menuAction != "import-set" {
		t.Fatalf("expected import-set prompt mode, got %v %q", m.menuInputMode, m.menuAction)
	}
	// perform the import flow (path -> policy -> dedupe)
	performImportSetFlow(t, m, imp, src)

	// confirm list includes imported set
	found := false
	for _, it := range m.list.Items() {
		if ci, ok := it.(csItem); ok {
			if ci.cs.Name == "imported-set" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("expected list to include imported set 'imported-set', got %v", m.list.Items())
	}
}
