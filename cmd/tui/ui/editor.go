package ui

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/VoxDroid/krnr/internal/executor"
	"github.com/VoxDroid/krnr/internal/nameutil"
	"github.com/VoxDroid/krnr/internal/tui/adapters"
)

// handleEditorKey processes KeyMsg events when the editor modal is active.
// It mirrors the previous handling logic that lived inside Update().
func (m *TuiModel) handleEditorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	// tab cycles fields
	switch s {
	case "tab":
		m.editor.field = (m.editor.field + 1) % 6
		return m, nil
	case "esc":
		// cancel editing and restore prior view. If we were creating a new entry
		// hide the full-screen detail entirely; otherwise refresh the detail.
		wasCreate := m.editor.create
		m.editingMeta = false
		m.editor.create = false
		if wasCreate {
			m.setShowDetail(false)
			m.detail = ""
			m.vp.SetContent("")
			return m, nil
		}
		if cs, err := m.uiModel.GetCommandSet(context.Background(), m.detailName); err == nil {
			m.detail = formatCSFullScreen(cs, m.width, m.height)
			m.vp.SetContent(m.detail)
		}
		return m, nil
	case "up":
		if m.editor.field == 5 && m.editor.cmdIndex > 0 {
			m.editor.cmdIndex--
		}
		return m, nil
	case "down":
		if m.editor.field == 5 && m.editor.cmdIndex < len(m.editor.commands)-1 {
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
	if msg.Type == tea.KeyCtrlA && m.editor.field == 5 {
		m.editor.commands = append(m.editor.commands, "")
		m.editor.cmdIndex = len(m.editor.commands) - 1
		return m, nil
	}
	// Ctrl+D: delete current command
	if msg.Type == tea.KeyCtrlD && m.editor.field == 5 && len(m.editor.commands) > 0 {
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

	// Save on Ctrl+S: schedule a short delayed save so that recently-typed
	// runes arriving just before Ctrl+S have a small window to be processed.
	if msg.Type == tea.KeyCtrlS {
		// Prevent re-entry: set saving; perform an immediate save for fast
		// non-PTY flows but also schedule a delayed save to catch any runes that
		// arrive just after Ctrl+S in PTY environments.
		if m.editor.saving {
			m.setNotification("save in progress")
			m.logs = append(m.logs, "notification: save in progress")
			return m, nil
		}
		m.editor.saving = true
		// immediate attempt
		if err := m.editorSave(); err != nil {
			m.logs = append(m.logs, "replace commands: "+err.Error())
		}
		// clear immediate saving guard so subsequent Ctrl+S presses behave normally
		m.editor.saving = false
		// schedule a delayed save as a second attempt to capture late keystrokes.
		// Delay is configurable via KRNR_SAVE_DELAY_MS (milliseconds) to adjust for
		// slower CI/PTY environments; default is 120ms.
		return m, tea.Tick(getSaveDelay(), func(_ time.Time) tea.Msg { return saveNowMsg{} })
	}

	// Handle rune input and backspace depending on field
	if msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete {
		switch m.editor.field {
		case 0:
			m.editor.name = trimLastRune(m.editor.name)
			m.editor.lastFailedName = ""
			m.editor.lastFailedReason = ""
			m.clearNotification()
		case 1:
			m.editor.desc = trimLastRune(m.editor.desc)
		case 2:
			m.editor.author = trimLastRune(m.editor.author)
		case 3:
			m.editor.authorEmail = trimLastRune(m.editor.authorEmail)
		case 4:
			m.editor.tags = trimLastRune(m.editor.tags)
		case 5:
			idx := m.editor.cmdIndex
			if idx >= 0 && idx < len(m.editor.commands) {
				m.editor.commands[idx] = trimLastRune(m.editor.commands[idx])
			}
		}
		// mark last edit time so scheduled save can wait for stability
		m.editor.lastEditAt = time.Now()
		m.editor.saveRetries = 0
		return m, nil
	}
	if msg.Type == tea.KeyRunes {
		r := msg.Runes
		for _, ru := range r {
			switch m.editor.field {
			case 0:
				m.editor.name += string(ru)
				m.editor.lastFailedName = ""
				m.editor.lastFailedReason = ""
				// user started editing: clear previous notifications
				m.clearNotification()
			case 1:
				m.editor.desc += string(ru)
			case 2:
				m.editor.author += string(ru)
			case 3:
				m.editor.authorEmail += string(ru)
			case 4:
				m.editor.tags += string(ru)
			case 5:
				idx := m.editor.cmdIndex
				if idx >= 0 && idx < len(m.editor.commands) {
					m.editor.commands[idx] += string(ru)
				}
			}
		}
		// mark last edit time so scheduled save can wait for stability
		m.editor.lastEditAt = time.Now()
		m.editor.saveRetries = 0
		return m, nil
	}

	return m, nil
}

// editorSave sanitizes and validates editor commands and persists them via the UI model.
func (m *TuiModel) editorSave() error {
	// build new summary and persist changes
	name := strings.TrimSpace(m.editor.name)
	// short-circuit repeat failures for the same name
	if m.editor.lastFailedName != "" && m.editor.lastFailedName == name {
		// Re-display the original failure reason for better UX and deterministic tests
		msg := m.editor.lastFailedReason
		if msg == "" {
			msg = "previous save attempt failed for this name"
		}
		m.setNotification(msg)
		m.logs = append(m.logs, "notification: "+msg)
		return fmt.Errorf("%s", msg)
	}
	if name == "" {
		msg := "invalid name: name cannot be empty"
		m.editor.lastFailedName = name
		m.editor.lastFailedReason = msg
		m.setNotification(msg)
		m.logs = append(m.logs, "notification: "+msg)
		return fmt.Errorf("%s", msg)
	}
	// reject names with invalid bytes or control characters (defensive)
	// sanitize name first to remove harmless invisible/control characters
	if sanitized, changed := nameutil.SanitizeName(name); changed {
		// replace displayed input so user sees the cleaned name
		m.editor.name = sanitized
		m.setNotification("invalid characters removed from name")
		m.logs = append(m.logs, "notification: invalid characters removed from name")
		name = sanitized
	}
	if err := nameutil.ValidateName(name); err != nil {
		msg := err.Error()
		m.editor.lastFailedName = name
		m.editor.lastFailedReason = msg
		m.setNotification(msg)
		m.logs = append(m.logs, "notification: "+msg)
		// Diagnostics: log the name bytes and any control runes (hex)
		var ctrlHex []string
		for _, r := range name {
			if unicode.IsControl(r) {
				ctrlHex = append(ctrlHex, fmt.Sprintf("U+%04X", r))
			}
		}
		m.logs = append(m.logs, fmt.Sprintf("name-diagnostics: %q bytes=%v ctrl=%v", name, []byte(name), ctrlHex))
		return fmt.Errorf("%s", msg)
	}
	// UI-level duplicate check: refresh and verify name doesn't already exist
	_ = m.uiModel.RefreshList(context.Background())
	// check cached list for duplicates to provide immediate feedback at UI level
	cached := m.uiModel.ListCached()
	if m.editor.create {
		for _, cs := range cached {
			if cs.Name == name {
				msg := "invalid name: name already exists"
				m.editor.lastFailedName = name
				m.editor.lastFailedReason = msg
				m.editor.lastFailedReason = msg
				m.setNotification(msg)
				m.logs = append(m.logs, "notification: "+msg)
				return fmt.Errorf("%s", msg)
			}
		}
	} else {
		// updating existing set; if name changed, ensure it doesn't collide with another set
		if name != m.detailName {
			for _, cs := range cached {
				if cs.Name == name {
					msg := "invalid name: name already exists"
					m.editor.lastFailedName = name
					m.setNotification(msg)
					m.logs = append(m.logs, "notification: "+msg)
					return fmt.Errorf("%s", msg)
				}
			}
		}
	}

	newCS := adapters.CommandSetSummary{
		Name:        name,
		Description: m.editor.desc,
		Tags:        []string{},
		Commands:    []string{},
		AuthorName:  m.editor.author,
		AuthorEmail: m.editor.authorEmail,
	}
	if strings.TrimSpace(m.editor.tags) != "" {
		for _, t := range strings.Split(m.editor.tags, ",") {
			newCS.Tags = append(newCS.Tags, strings.TrimSpace(t))
		}
	}
	// sanitize and validate commands before persisting
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
			m.setNotification(err.Error())
			m.logs = append(m.logs, "notification: "+err.Error())
			return err
		}
		clean = append(clean, cSan)
	}
	newCS.Commands = clean
	if m.editor.create {
		// create new set
		m.logs = append(m.logs, "attempting save: "+newCS.Name)
		if err := m.uiModel.Save(context.Background(), newCS); err != nil {
			m.editor.lastFailedName = name
			m.editor.lastFailedReason = err.Error()
			m.setNotification(err.Error())
			m.logs = append(m.logs, "notification: "+err.Error())
			return fmt.Errorf("save: %w", err)
		}
		// refresh list and select new item
		_ = m.uiModel.RefreshList(context.Background())
		items := make([]list.Item, 0, len(m.uiModel.ListCached()))
		for _, s := range m.uiModel.ListCached() {
			items = append(items, csItem{cs: s})
		}
		m.list.SetItems(items)
		// select newly created item if present
		for i, it := range items {
			if csi, ok := it.(csItem); ok && csi.cs.Name == newCS.Name {
				m.list.Select(i)
				break
			}
		}
		m.detailName = newCS.Name
		m.detail = formatCSFullScreen(newCS, m.width, m.height)
		m.vp.SetContent(m.detail)
		// clear any prior notification if save succeeded
		m.clearNotification()
	} else {
		// call update for metadata
		if err := m.uiModel.UpdateCommandSet(context.Background(), m.detailName, newCS); err != nil {
			m.setNotification(err.Error())
			m.logs = append(m.logs, "notification: "+err.Error())
			return fmt.Errorf("update: %w", err)
		}
		// replace commands
		if err := m.uiModel.ReplaceCommands(context.Background(), newCS.Name, clean); err != nil {
			m.setNotification(err.Error())
			return fmt.Errorf("replace commands: %w", err)
		}
		// refresh detail
		if cs, err := m.uiModel.GetCommandSet(context.Background(), newCS.Name); err == nil {
			m.detailName = cs.Name
			m.detail = formatCSFullScreen(cs, m.width, m.height)
			m.vp.SetContent(m.detail)
		}
		m.clearNotification()
	}
	m.editingMeta = false
	m.editor.create = false
	return nil
}

// renderEditor produces the editor modal content when editing metadata in-place.
func (m *TuiModel) renderEditor() string {
	var b strings.Builder
	// explicit title: include which field is active to make it obvious
	title := "Edit: " + m.detailName
	if m.editor.create {
		title = "Create: New Entry"
	}

	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0ea5a4")).Render(title))
	b.WriteString("\n\n")
	// styles for focused field highlighting
	focusStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#000000")).Background(lipgloss.Color("#fde047"))
	// inline error style for name validation
	errorStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#dc2626"))
	// Name
	if m.editor.field == 0 {
		line := focusStyle.Render("Name: > " + m.editor.name)
		// show inline error if lastFailedName matches current name or name empty
		if strings.TrimSpace(m.editor.name) == "" || (m.editor.lastFailedName != "" && strings.TrimSpace(m.editor.lastFailedName) == strings.TrimSpace(m.editor.name)) {
			line = line + " " + errorStyle.Render("⚠ invalid")
		}
		b.WriteString(line + "\n")
	} else {
		line := "Name:   " + m.editor.name
		if strings.TrimSpace(m.editor.name) == "" || (m.editor.lastFailedName != "" && strings.TrimSpace(m.editor.lastFailedName) == strings.TrimSpace(m.editor.name)) {
			line = line + " " + errorStyle.Render("⚠ invalid")
		}
		b.WriteString(line + "\n")
	}
	// Description
	if m.editor.field == 1 {
		b.WriteString(focusStyle.Render("Description: > "+m.editor.desc) + "\n")
	} else {
		b.WriteString("Description:   " + m.editor.desc + "\n")
	}
	// Author name
	if m.editor.field == 2 {
		b.WriteString(focusStyle.Render("Author: > "+m.editor.author) + "\n")
	} else {
		b.WriteString("Author:   " + m.editor.author + "\n")
	}
	// Author email
	if m.editor.field == 3 {
		b.WriteString(focusStyle.Render("Email: > "+m.editor.authorEmail) + "\n")
	} else {
		b.WriteString("Email:   " + m.editor.authorEmail + "\n")
	}
	// Tags
	if m.editor.field == 4 {
		b.WriteString(focusStyle.Render("Tags: > "+m.editor.tags) + "\n")
	} else {
		b.WriteString("Tags:   " + m.editor.tags + "\n")
	}
	// Commands
	b.WriteString("\nCommands:\n")
	for i, c := range m.editor.commands {
		prefix := fmt.Sprintf("%d) ", i+1)
		if m.editor.field == 5 && i == m.editor.cmdIndex {
			b.WriteString(focusStyle.Render("* "+prefix+c) + "\n")
		} else {
			b.WriteString("  " + prefix + c + "\n")
		}
	}
	return b.String()
}

// getSaveDelay returns the configured save delay duration.
func getSaveDelay() time.Duration {
	if v := os.Getenv("KRNR_SAVE_DELAY_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Millisecond
		}
	}
	return 120 * time.Millisecond
}
