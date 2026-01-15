package ui

import (
	"context"
	"strings"
	"testing"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestDispatchKey_DelegatesToEditor(t *testing.T) {
	m := NewModel(nil)
	m.editingMeta = true
	m.editor.field = 0
	msg := tea.KeyMsg{Type: tea.KeyTab}
	dm, _, _ := dispatchKey(m, msg)
	tm := dm.(*TuiModel)
	if tm.editor.field != 1 {
		t.Fatalf("expected editor.field to increment to 1; got %d", tm.editor.field)
	}
}

func TestHandleGlobalKeys_QuitAndHelp(t *testing.T) {
	m := NewModel(nil)
	// quit
	_, cmd, handled := handleGlobalKeys(m, "q", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled {
		t.Fatalf("expected quit to be handled")
	}
	if cmd == nil {
		t.Fatalf("expected non-nil cmd for quit handler")
	}
	// help
	m2, _, handled := handleGlobalKeys(m, "?", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if !handled {
		t.Fatalf("expected help to be handled")
	}
	mt := m2.(*TuiModel)
	if !mt.showDetail {
		t.Fatalf("expected showDetail true after help")
	}
	if !strings.Contains(mt.detail, "Help:") {
		t.Fatalf("expected detail to contain Help, got %q", mt.detail)
	}
}

func TestTabTogglesFocus(t *testing.T) {
	m := NewModel(nil)
	m.focusRight = false
	m1, _, handled := handleGlobalKeys(m, "tab", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\t'}})
	if !handled {
		t.Fatalf("expected tab to be handled")
	}
	mt := m1.(*TuiModel)
	if !mt.focusRight {
		t.Fatalf("expected focusRight to toggle to true")
	}
}

func TestRunViaKeyStartsAndStreams(t *testing.T) {
	// fakes
	fakeReg := &fakeRegistry{items: []adapters.CommandSetSummary{{Name: "one", Description: "First"}}}
	fakeExec := &fakeExecAdapter{lines: []string{"r1", "r2"}}
	ui := modelpkg.New(fakeReg, fakeExec, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	// press 'r' to run
	m1, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = m1.(*TuiModel)
	// consume returned cmd until run completes
	for i := 0; i < 10; i++ {
		if cmd == nil {
			break
		}
		msg := cmd()
		m1, cmd = m.Update(msg)
		m = m1.(*TuiModel)
		if !m.runInProgress && m.runCh == nil {
			break
		}
	}
	if len(m.logs) != 2 {
		t.Fatalf("expected 2 log lines got %d", len(m.logs))
	}
	if m.logs[0] != "r1" || m.logs[1] != "r2" {
		t.Fatalf("unexpected logs: %v", m.logs)
	}
}

func TestHandleListFiltering_DelegatedAndUpdatesFilter(t *testing.T) {
	fakeReg := &fakeRegistry{items: []adapters.CommandSetSummary{{Name: "alpha", Description: "A"}, {Name: "beta", Description: "B"}, {Name: "bravo", Description: "B2"}}}
	ui := modelpkg.New(fakeReg, &fakeExecAdapter{lines: []string{"ok"}}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	// Directly apply the filtering key logic (test helper) — this avoids
	// having to manipulate the internal list state to reach Filtering.
	m2, _, handled := applyListFilterKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	if !handled {
		t.Fatalf("expected applyListFilterKey to handle rune input")
	}
	m = m2.(*TuiModel)
	if m.listFilter != "b" {
		t.Fatalf("expected listFilter to be 'b', got %q", m.listFilter)
	}
	if len(m.list.Items()) == 0 {
		t.Fatalf("expected filtered list to have items")
	}
	// Backspace should remove the filter char
	m3, _, handled := applyListFilterKey(m, tea.KeyMsg{Type: tea.KeyBackspace})
	if !handled {
		t.Fatalf("expected backspace to be handled")
	}
	m = m3.(*TuiModel)
	if m.listFilter != "" {
		t.Fatalf("expected listFilter to be empty after backspace, got %q", m.listFilter)
	}
}
