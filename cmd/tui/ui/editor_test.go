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
