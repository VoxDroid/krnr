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
	if dm, cmd, handled := m.handleEditorNavKey(msg); handled {
		if newM, ok := dm.(*TuiModel); ok {
			m = newM
		}
		return m, cmd
	}

	// control sequences and editing actions delegated to helpers
	if msg.Type == tea.KeyCtrlA {
		return m.handleEditorCtrlA()
	}
	if msg.Type == tea.KeyCtrlD {
		return m.handleEditorCtrlD()
	}
	if msg.Type == tea.KeyCtrlS {
		return m.handleEditorCtrlS()
	}
	if msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete {
		return m.handleEditorBackspace()
	}
	if msg.Type == tea.KeyRunes {
		return m.handleEditorRunes(msg)
	}
	return m, nil
}

func (m *TuiModel) handleEditorNavKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	s := msg.String()
	switch s {
	case "tab":
		m.editor.field = (m.editor.field + 1) % 6
		return m, nil, true
	case "esc":
		dm, cmd := m.handleEditorCancel()
		return dm, cmd, true
	case "up":
		m.adjustEditorCmdIndex(-1)
		return m, nil, true
	case "down":
		m.adjustEditorCmdIndex(1)
		return m, nil, true
	case "enter":
		if m.editor.field == 1 {
			m.editor.desc += "\n"
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m *TuiModel) adjustEditorCmdIndex(delta int) {
	if m.editor.field != 5 {
		return
	}
	if delta < 0 {
		if m.editor.cmdIndex > 0 {
			m.editor.cmdIndex--
		}
		return
	}
	if delta > 0 {
		if m.editor.cmdIndex < len(m.editor.commands)-1 {
			m.editor.cmdIndex++
		}
	}
}

func (m *TuiModel) handleEditorCancel() (tea.Model, tea.Cmd) {
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
}

func (m *TuiModel) handleEditorCtrlA() (tea.Model, tea.Cmd) {
	m.editor.field = 5
	m.editor.commands = append(m.editor.commands, "")
	m.editor.cmdIndex = len(m.editor.commands) - 1
	m.editor.lastEditAt = time.Now()
	m.editor.saveRetries = 0
	return m, nil
}

func (m *TuiModel) handleEditorCtrlD() (tea.Model, tea.Cmd) {
	if m.editor.field == 5 && len(m.editor.commands) > 0 {
		idx := m.editor.cmdIndex
		m.editor.commands = append(m.editor.commands[:idx], m.editor.commands[idx+1:]...)
		if m.editor.cmdIndex >= len(m.editor.commands) && m.editor.cmdIndex > 0 {
			m.editor.cmdIndex--
		}
		if len(m.editor.commands) == 0 {
			m.editor.commands = []string{""}
		}
	}
	return m, nil
}

func (m *TuiModel) handleEditorCtrlS() (tea.Model, tea.Cmd) {
	if m.editor.saving {
		m.setNotification("save in progress")
		m.logs = append(m.logs, "notification: save in progress")
		return m, nil
	}
	m.editor.saving = true
	if err := m.editorSave(); err != nil {
		m.logs = append(m.logs, "replace commands: "+err.Error())
	}
	m.editor.saving = false
	return m, tea.Tick(getSaveDelay(), func(_ time.Time) tea.Msg { return saveNowMsg{} })
}

func (m *TuiModel) handleEditorBackspace() (tea.Model, tea.Cmd) {
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
	m.editor.lastEditAt = time.Now()
	m.editor.saveRetries = 0
	return m, nil
}

func (m *TuiModel) handleEditorRunes(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	r := msg.Runes
	for _, ru := range r {
		switch m.editor.field {
		case 0:
			m.editor.name += string(ru)
			m.editor.lastFailedName = ""
			m.editor.lastFailedReason = ""
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
	m.editor.lastEditAt = time.Now()
	m.editor.saveRetries = 0
	return m, nil
}

// editorSave sanitizes and validates editor commands and persists them via the UI model.
func (m *TuiModel) editorSave() error {
	name := strings.TrimSpace(m.editor.name)
	if err := m.validateEditorName(name); err != nil {
		return err
	}
	if err := m.checkEditorDuplicate(name); err != nil {
		return err
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
	clean, err := m.sanitizeAndValidateCommands()
	if err != nil {
		return err
	}
	newCS.Commands = clean
	if m.editor.create {
		if err := m.createCommandSet(newCS); err != nil {
			return err
		}
	} else {
		if err := m.updateCommandSet(newCS); err != nil {
			return err
		}
	}
	m.editingMeta = false
	m.editor.create = false
	return nil
}

func (m *TuiModel) validateEditorName(name string) error {
	if m.editor.lastFailedName != "" && m.editor.lastFailedName == name {
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
	if sanitized, changed := nameutil.SanitizeName(name); changed {
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
		var ctrlHex []string
		for _, r := range name {
			if unicode.IsControl(r) {
				ctrlHex = append(ctrlHex, fmt.Sprintf("U+%04X", r))
			}
		}
		m.logs = append(m.logs, fmt.Sprintf("name-diagnostics: %q bytes=%v ctrl=%v", name, []byte(name), ctrlHex))
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func (m *TuiModel) checkEditorDuplicate(name string) error {
	_ = m.uiModel.RefreshList(context.Background())
	cached := m.uiModel.ListCached()
	if m.editor.create {
		for _, cs := range cached {
			if cs.Name == name {
				msg := "invalid name: name already exists"
				m.editor.lastFailedName = name
				m.editor.lastFailedReason = msg
				m.setNotification(msg)
				m.logs = append(m.logs, "notification: "+msg)
				return fmt.Errorf("%s", msg)
			}
		}
	} else {
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
	return nil
}

func (m *TuiModel) sanitizeAndValidateCommands() ([]string, error) {
	raw := filterEmptyLines(m.editor.commands)
	clean := make([]string, 0, len(raw))
	for _, c := range raw {
		cSan := executor.Sanitize(c)
		if cSan != c {
			m.logs = append(m.logs, "sanitized command: \""+c+"\" -> \""+cSan+"\"")
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
			return nil, err
		}
		clean = append(clean, cSan)
	}
	return clean, nil
}

func (m *TuiModel) createCommandSet(newCS adapters.CommandSetSummary) error {
	m.logs = append(m.logs, "attempting save: "+newCS.Name)
	if err := m.uiModel.Save(context.Background(), newCS); err != nil {
		m.editor.lastFailedName = newCS.Name
		m.editor.lastFailedReason = err.Error()
		m.setNotification(err.Error())
		m.logs = append(m.logs, "notification: "+err.Error())
		return fmt.Errorf("save: %w", err)
	}
	_ = m.uiModel.RefreshList(context.Background())
	items := make([]list.Item, 0, len(m.uiModel.ListCached()))
	for _, s := range m.uiModel.ListCached() {
		items = append(items, csItem{cs: s})
	}
	m.list.SetItems(items)
	for i, it := range items {
		if csi, ok := it.(csItem); ok && csi.cs.Name == newCS.Name {
			m.list.Select(i)
			break
		}
	}
	m.detailName = newCS.Name
	m.detail = formatCSFullScreen(newCS, m.width, m.height)
	m.vp.SetContent(m.detail)
	m.clearNotification()
	m.editor.lastSavedAt = time.Now()
	return nil
}

func (m *TuiModel) updateCommandSet(newCS adapters.CommandSetSummary) error {
	if err := m.uiModel.UpdateCommandSetAndReplaceCommands(context.Background(), m.detailName, newCS); err != nil {
		m.setNotification(err.Error())
		m.logs = append(m.logs, "notification: "+err.Error())
		return fmt.Errorf("update: %w", err)
	}
	if cs, err := m.uiModel.GetCommandSet(context.Background(), newCS.Name); err == nil {
		m.detailName = cs.Name
		m.detail = formatCSFullScreen(cs, m.width, m.height)
		m.vp.SetContent(m.detail)
	}
	m.editor.lastSavedAt = time.Now()
	m.clearNotification()
	return nil
}

// renderEditor produces the editor modal content when editing metadata in-place.
func (m *TuiModel) renderEditor() string {
	var b strings.Builder
	b.WriteString(m.renderEditorTitle())
	b.WriteString("\n\n")
	b.WriteString(m.renderEditorFields())
	b.WriteString("\n")
	b.WriteString(m.renderEditorCommands())
	return b.String()
}

func (m *TuiModel) renderEditorTitle() string {
	title := "Edit: " + m.detailName
	if m.editor.create {
		title = "Create: New Entry"
	}
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0ea5a4")).Render(title)
}

func (m *TuiModel) renderEditorFields() string {
	// styles for focused field highlighting
	focusStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#000000")).Background(lipgloss.Color("#fde047"))
	// inline error style for name validation
	errorStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#dc2626"))
	var b strings.Builder
	b.WriteString(m.renderEditorFieldName(focusStyle, errorStyle))
	b.WriteString(m.renderEditorFieldGeneric("Description", m.editor.desc, 1, focusStyle))
	b.WriteString(m.renderEditorFieldGeneric("Author", m.editor.author, 2, focusStyle))
	b.WriteString(m.renderEditorFieldGeneric("Email", m.editor.authorEmail, 3, focusStyle))
	b.WriteString(m.renderEditorFieldGeneric("Tags", m.editor.tags, 4, focusStyle))
	return b.String()
}

func (m *TuiModel) renderEditorFieldName(focusStyle, errorStyle lipgloss.Style) string {
	var line string
	if m.editor.field == 0 {
		line = focusStyle.Render("Name: > " + m.editor.name)
	} else {
		line = "Name:   " + m.editor.name
	}
	// show inline error if lastFailedName matches current name or name empty
	if strings.TrimSpace(m.editor.name) == "" || (m.editor.lastFailedName != "" && strings.TrimSpace(m.editor.lastFailedName) == strings.TrimSpace(m.editor.name)) {
		line = line + " " + errorStyle.Render("⚠ invalid")
	}
	return line + "\n"
}

func (m *TuiModel) renderEditorFieldGeneric(label, value string, index int, focusStyle lipgloss.Style) string {
	if m.editor.field == index {
		return focusStyle.Render(label+": > "+value) + "\n"
	}
	return label + ":   " + value + "\n"
}

func (m *TuiModel) renderEditorCommands() string {
	var b strings.Builder
	b.WriteString("Commands:\n")
	for i, c := range m.editor.commands {
		prefix := fmt.Sprintf("%d) ", i+1)
		if m.editor.field == 5 && i == m.editor.cmdIndex {
			focusStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#000000")).Background(lipgloss.Color("#fde047"))
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
