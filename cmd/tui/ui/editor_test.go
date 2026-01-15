package ui

import (
	"context"
	"strings"
	"testing"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestEditorSaveRejectsControlCharacters_Newline(t *testing.T) {
	full := adapters.CommandSetSummary{Name: "one", Description: "First", Commands: []string{"echo hi"}}
	reg := &replaceFakeRegistry{items: []adapters.CommandSetSummary{{Name: "one", Description: "First"}}, full: full}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	m1, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = m1.(*TuiModel)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(*TuiModel)
	// open in-TUI editor
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = m3.(*TuiModel)
	if !m.editingMeta {
		t.Fatalf("expected editor to be open")
	}
	// cycle to commands field (tab until commands)
	for i := 0; i < 10 && m.editor.field != 5; i++ {
		m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = m4.(*TuiModel)
	}
	// replace the current command with a command containing a newline
	idx := m.editor.cmdIndex
	m.editor.commands[idx] = "echo broken\nnext"
	// save with Ctrl+S
	m7, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = m7.(*TuiModel)
	// ReplaceCommands should not have been called due to invalid input
	if len(reg.lastCommands) != 0 {
		t.Fatalf("expected ReplaceCommands not to be called, got %#v", reg.lastCommands)
	}
	// expect an error log
	found := false
	for _, l := range m.logs {
		if strings.Contains(l, "replace commands: invalid command") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected error log about invalid command, got logs: %#v", m.logs)
	}
}

func TestCtrlAFromOtherFieldsAddsCommand(t *testing.T) {
	full := adapters.CommandSetSummary{Name: "one", Description: "First", Commands: []string{"echo hi"}}
	reg := &replaceFakeRegistry{items: []adapters.CommandSetSummary{{Name: "one", Description: "First"}}, full: full}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	m1, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = m1.(*TuiModel)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(*TuiModel)
	// open in-TUI editor
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = m3.(*TuiModel)
	if !m.editingMeta {
		t.Fatalf("expected editor to be open")
	}
	// ensure we are not on commands field
	if m.editor.field == 5 {
		t.Fatalf("expected initial field not to be commands, got 5")
	}
	// press Ctrl+A from current field - should switch to commands and add one
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	m = m4.(*TuiModel)
	if m.editor.field != 5 {
		t.Fatalf("expected field to be commands (5) after Ctrl+A, got %d", m.editor.field)
	}
	if len(m.editor.commands) == 0 {
		t.Fatalf("expected commands list to have at least one entry")
	}
	idx := m.editor.cmdIndex
	// type into the new command
	m5, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e', 'c', 'h', 'o', ' ', 'x'}})
	m = m5.(*TuiModel)
	if m.editor.commands[idx] != "echo x" {
		t.Fatalf("expected typed command 'echo x', got %q", m.editor.commands[idx])
	}
	// Save with Ctrl+S and ensure ReplaceCommands is invoked with the updated command
	m6, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = m6.(*TuiModel)
	if len(reg.lastCommands) == 0 {
		t.Fatalf("expected ReplaceCommands to be called, got none")
	}
	found := false
	for _, c := range reg.lastCommands {
		if strings.Contains(c, "echo x") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected updated command in ReplaceCommands, got %#v", reg.lastCommands)
	}
}
