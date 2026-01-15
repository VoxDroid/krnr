package ui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/VoxDroid/krnr/internal/executor"
	"github.com/VoxDroid/krnr/internal/tui/adapters"
)

// handleEditorKey processes KeyMsg events when the editor modal is active.
// It mirrors the previous handling logic that lived inside Update().
func (m *TuiModel) handleEditorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	// tab cycles fields
	switch s {
	case "tab":
		m.editor.field = (m.editor.field + 1) % 4
		return m, nil
	case "esc":
		// cancel editing and restore detail view
		m.editingMeta = false
		if cs, err := m.uiModel.GetCommandSet(context.Background(), m.detailName); err == nil {
			m.detail = formatCSFullScreen(cs, m.width, m.height)
			m.vp.SetContent(m.detail)
		}
		return m, nil
	case "up":
		if m.editor.field == 3 && m.editor.cmdIndex > 0 {
			m.editor.cmdIndex--
		}
		return m, nil
	case "down":
		if m.editor.field == 3 && m.editor.cmdIndex < len(m.editor.commands)-1 {
			m.editor.cmdIndex++
		}
		return m, nil
	case "enter":
		// Treat enter as newline for description field only
		if m.editor.field == 1 {
			m.editor.desc += "\n"
		}
		return m, nil
	}

	// Ctrl+A: add command (use Ctrl to avoid colliding with typed runes)
	if msg.Type == tea.KeyCtrlA && m.editor.field == 3 {
		m.editor.commands = append(m.editor.commands, "")
		m.editor.cmdIndex = len(m.editor.commands) - 1
		return m, nil
	}
	// Ctrl+D: delete current command
	if msg.Type == tea.KeyCtrlD && m.editor.field == 3 && len(m.editor.commands) > 0 {
		idx := m.editor.cmdIndex
		m.editor.commands = append(m.editor.commands[:idx], m.editor.commands[idx+1:]...)
		if m.editor.cmdIndex >= len(m.editor.commands) && m.editor.cmdIndex > 0 {
			m.editor.cmdIndex--
		}
		if len(m.editor.commands) == 0 {
			m.editor.commands = []string{""}
		}
		return m, nil
	}

	// Save on Ctrl+S
	if msg.Type == tea.KeyCtrlS {
		if err := m.editorSave(); err != nil {
			m.logs = append(m.logs, "replace commands: "+err.Error())
		}
		return m, nil
	}

	// Handle rune input and backspace depending on field
	if msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete {
		switch m.editor.field {
		case 0:
			m.editor.name = trimLastRune(m.editor.name)
		case 1:
			m.editor.desc = trimLastRune(m.editor.desc)
		case 2:
			m.editor.tags = trimLastRune(m.editor.tags)
		case 3:
			idx := m.editor.cmdIndex
			if idx >= 0 && idx < len(m.editor.commands) {
				m.editor.commands[idx] = trimLastRune(m.editor.commands[idx])
			}
		}
		return m, nil
	}
	if msg.Type == tea.KeyRunes {
		r := msg.Runes
		for _, ru := range r {
			switch m.editor.field {
			case 0:
				m.editor.name += string(ru)
			case 1:
				m.editor.desc += string(ru)
			case 2:
				m.editor.tags += string(ru)
			case 3:
				idx := m.editor.cmdIndex
				if idx >= 0 && idx < len(m.editor.commands) {
					m.editor.commands[idx] += string(ru)
				}
			}
		}
		return m, nil
	}

	return m, nil
}

// editorSave sanitizes and validates editor commands and persists them via the UI model.
func (m *TuiModel) editorSave() error {
	// build new summary and persist changes
	newCS := adapters.CommandSetSummary{
		Name:        m.editor.name,
		Description: m.editor.desc,
		Tags:        []string{},
	}
	if strings.TrimSpace(m.editor.tags) != "" {
		for _, t := range strings.Split(m.editor.tags, ",") {
			newCS.Tags = append(newCS.Tags, strings.TrimSpace(t))
		}
	}
	// call update for metadata
	if err := m.uiModel.UpdateCommandSet(context.Background(), m.detailName, newCS); err != nil {
		return fmt.Errorf("update: %w", err)
	}
	// replace commands: sanitize and validate before persisting so the user sees
	// corrected commands immediately rather than discovering issues later
	raw := filterEmptyLines(m.editor.commands)
	clean := make([]string, 0, len(raw))
	for _, c := range raw {
		cSan := executor.Sanitize(c)
		if cSan != c {
			m.logs = append(m.logs, "sanitized command: \""+c+"\" -> \""+cSan+"\"")
			// diagnostic: include quoted representations and raw bytes to help PTY tests
			m.logs = append(m.logs, fmt.Sprintf("sanitized debug: orig=%q san=%q bytes=%v", c, cSan, []byte(cSan)))
			for j := range m.editor.commands {
				if strings.TrimSpace(m.editor.commands[j]) == strings.TrimSpace(c) {
					m.editor.commands[j] = cSan
					break
				}
			}
		}
		if err := executor.ValidateCommand(cSan); err != nil {
			return err
		}
		clean = append(clean, cSan)
	}
	if err := m.uiModel.ReplaceCommands(context.Background(), newCS.Name, clean); err != nil {
		return fmt.Errorf("replace commands: %w", err)
	}
	// refresh detail
	if cs, err := m.uiModel.GetCommandSet(context.Background(), newCS.Name); err == nil {
		m.detailName = cs.Name
		m.detail = formatCSFullScreen(cs, m.width, m.height)
		m.vp.SetContent(m.detail)
	}
	m.editingMeta = false
	return nil
}

// renderEditor produces the editor modal content when editing metadata in-place.
func (m *TuiModel) renderEditor() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0ea5a4")).Render(fmt.Sprintf("Edit: %s", m.detailName)))
	b.WriteString("\n\n")
	// Name
	if m.editor.field == 0 {
		b.WriteString("Name: > " + m.editor.name + "\n")
	} else {
		b.WriteString("Name:   " + m.editor.name + "\n")
	}
	// Description
	if m.editor.field == 1 {
		b.WriteString("Description: > " + m.editor.desc + "\n")
	} else {
		b.WriteString("Description:   " + m.editor.desc + "\n")
	}
	// Tags
	if m.editor.field == 2 {
		b.WriteString("Tags: > " + m.editor.tags + "\n")
	} else {
		b.WriteString("Tags:   " + m.editor.tags + "\n")
	}
	// Commands
	b.WriteString("\nCommands:\n")
	for i, c := range m.editor.commands {
		prefix := fmt.Sprintf("%d) ", i+1)
		if m.editor.field == 3 && i == m.editor.cmdIndex {
			b.WriteString("* " + prefix + c + "\n")
		} else {
			b.WriteString("  " + prefix + c + "\n")
		}
	}
	return b.String()
}
