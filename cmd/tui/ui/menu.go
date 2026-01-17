package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/VoxDroid/krnr/internal/config"
	"github.com/VoxDroid/krnr/internal/install"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// handleMenuKey processes KeyMsg events while the menu modal is active.
func (m *TuiModel) handleMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	// If we are accepting text input for the menu (e.g., import path),
	// handle input via a dedicated handler to keep this function small.
	if m.menuInputMode {
		return m.processMenuInput(msg)
	}

	switch s {
	case "up":
		if m.menuIndex > 0 {
			m.menuIndex--
		} else {
			m.menuIndex = len(m.menuItems) - 1
		}
	case "down":
		m.menuIndex = (m.menuIndex + 1) % len(m.menuItems)
	case "esc":
		m.showMenu = false
	case "enter":
		return m.activateMenuItem()
	}
	return m, nil
}

func (m *TuiModel) processMenuInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	// handle backspace/delete
	if msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete {
		m.menuInput = trimLastRune(m.menuInput)
		return m, nil
	}
	if msg.Type == tea.KeyRunes {
		for _, r := range msg.Runes {
			m.menuInput += string(r)
		}
		return m, nil
	}
	if s != "enter" {
		if s == "esc" {
			m.menuInputMode = false
			m.menuInput = ""
			m.menuAction = ""
			m.showMenu = false
		}
		return m, nil
	}

	// Enter pressed — validate/perform the current action
	var notifyCmd tea.Cmd
	actions := map[string]func() tea.Cmd{
		"import-db":           m.handleMenuInputImportDB,
		"import-db-overwrite": m.handleMenuInputImportDBOverwrite,
		"import-set":          m.handleMenuInputImportSet,
		"import-set-policy":   m.handleMenuInputImportSetPolicy,
		"import-set-dedupe":   m.handleMenuInputImportSetDedupe,
		"install-scope":       m.handleMenuInputInstallScope,
		"install-addpath":     m.handleMenuInputInstallAddPath,
		"uninstall-confirm":   m.handleMenuInputUninstallConfirm,
		"export-db":           m.handleMenuInputExportDB,
	}
	if f, ok := actions[m.menuAction]; ok {
		prior := m.menuAction
		notifyCmd = f()
		// Preserve menu state only if handler set a different follow-up action
		if m.menuAction != "" && m.menuAction != prior {
			m.menuInputMode = true
			m.showMenu = true
			return m, notifyCmd
		}
	}

	m.menuInputMode = false
	m.menuInput = ""
	m.menuAction = ""
	m.showMenu = false
	return m, notifyCmd
}

func (m *TuiModel) activateMenuItem() (tea.Model, tea.Cmd) {
	item := m.menuItems[m.menuIndex]
	switch item {
	case "Export database":
		cwd, _ := os.Getwd()
		defaultName := fmt.Sprintf("krnr-%s.db", time.Now().Format("2006-01-02"))
		pref := filepath.Join(cwd, defaultName)
		m.menuInputMode = true
		m.menuAction = "export-db"
		m.menuInput = pref
		return m, nil
	case "Import database":
		m.menuInputMode = true
		m.menuAction = "import-db"
		m.menuInput = ""

	case "Import set":
		m.menuInputMode = true
		m.menuAction = "import-set"
		m.menuInput = ""
	case "Install":
		m.menuInputMode = true
		m.menuAction = "install-scope"
		m.menuInput = "user" // default
		return m, nil
	case "Uninstall":
		m.menuInputMode = true
		m.menuAction = "uninstall-confirm"
		m.menuInput = "n" // default No
		return m, nil
	case "Status":
		statusLines := []string{"krnr status:"}
		if st, err := install.GetStatus(); err != nil {
			statusLines = append(statusLines, "status: error: "+err.Error())
		} else {
			statusLines = append(statusLines, fmt.Sprintf("  User: installed=%v, on_path=%v, path=%s", st.UserInstalled, st.UserOnPath, st.UserPath))
			statusLines = append(statusLines, fmt.Sprintf("  System: installed=%v, on_path=%v, path=%s", st.SystemInstalled, st.SystemOnPath, st.SystemPath))
			statusLines = append(statusLines, fmt.Sprintf("  Metadata found: %v", st.MetadataFound))
		}
		m.detail = strings.Join(statusLines, "\n")
		m.vp.SetContent(m.detail)
		m.setShowDetail(true)
		m.showMenu = false
	case "Close":
		m.showMenu = false
	}
	return m, nil
}

func (m *TuiModel) handleMenuInputImportDB() tea.Cmd {
	if _, err := os.Stat(m.menuInput); err != nil {
		m.logs = append(m.logs, "import error: source not found or inaccessible")
		m.setNotification("import error: source not found or inaccessible")
		return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
	}
	m.menuPendingSrc = m.menuInput
	m.menuAction = "import-db-overwrite"
	m.menuInput = "n"
	return nil
}

func (m *TuiModel) handleMenuInputImportDBOverwrite() tea.Cmd {
	ov := strings.ToLower(strings.TrimSpace(m.menuInput))
	overwrite := ov == "y" || ov == "yes"
	if overwrite {
		_ = m.uiModel.Close()
	}
	if err := m.uiModel.ImportDB(context.Background(), m.menuPendingSrc, overwrite); err != nil {
		if overwrite {
			_ = m.uiModel.ReopenDB(context.Background())
		}
		m.logs = append(m.logs, "import error: "+err.Error())
		m.setNotification("import error: " + err.Error())
		m.menuPendingSrc = ""
		return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
	}
	m.logs = append(m.logs, "imported database from "+m.menuPendingSrc)
	m.setNotification("imported database from " + m.menuPendingSrc)
	if err := m.uiModel.ReopenDB(context.Background()); err != nil {
		m.logs = append(m.logs, "warning: failed to reopen DB: "+err.Error())
		m.setNotification("warning: failed to reopen DB")
	}
	_ = m.uiModel.RefreshList(context.Background())
	m.updateListFromCache()
	m.menuPendingSrc = ""
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
}

func (m *TuiModel) handleMenuInputImportSet() tea.Cmd {
	if _, err := os.Stat(m.menuInput); err != nil {
		m.logs = append(m.logs, "import error: source not found or inaccessible")
		m.setNotification("import error: source not found or inaccessible")
		return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
	}
	m.menuPendingSrc = m.menuInput
	m.menuAction = "import-set-policy"
	m.menuInput = "rename"
	return nil
}

func (m *TuiModel) handleMenuInputImportSetPolicy() tea.Cmd {
	policy := strings.TrimSpace(m.menuInput)
	if policy == "" {
		policy = "rename"
	}
	if policy == "merge" {
		m.menuAction = "import-set-dedupe"
		m.menuInput = "n"
		return nil
	}
	if err := m.uiModel.ImportSet(context.Background(), m.menuPendingSrc, policy, false); err != nil {
		m.logs = append(m.logs, "import error: "+err.Error())
		m.setNotification("import error: " + err.Error())
		m.menuPendingSrc = ""
		return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
	}
	m.logs = append(m.logs, "imported command set(s) from "+m.menuPendingSrc)
	m.setNotification("imported command set(s) from " + m.menuPendingSrc)
	_ = m.uiModel.RefreshList(context.Background())
	m.updateListFromCache()
	m.menuPendingSrc = ""
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
}

func (m *TuiModel) handleMenuInputImportSetDedupe() tea.Cmd {
	ded := strings.ToLower(strings.TrimSpace(m.menuInput))
	dedupe := ded == "y" || ded == "yes"
	if err := m.uiModel.ImportSet(context.Background(), m.menuPendingSrc, "merge", dedupe); err != nil {
		m.logs = append(m.logs, "import error: "+err.Error())
		m.setNotification("import error: " + err.Error())
		m.menuPendingSrc = ""
		return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
	}
	m.logs = append(m.logs, "imported command set(s) from "+m.menuPendingSrc)
	m.setNotification("imported command set(s) from " + m.menuPendingSrc)
	_ = m.uiModel.RefreshList(context.Background())
	m.updateListFromCache()
	m.menuPendingSrc = ""
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
}

func (m *TuiModel) handleMenuInputInstallScope() tea.Cmd {
	scope := strings.ToLower(strings.TrimSpace(m.menuInput))
	if scope == "" {
		scope = "user"
	}
	m.menuAction = "install-addpath"
	m.menuInput = "n"
	m.menuPendingSrc = scope
	return nil
}

func (m *TuiModel) handleMenuInputInstallAddPath() tea.Cmd {
	add := strings.ToLower(strings.TrimSpace(m.menuInput))
	addToPath := add == "y" || add == "yes"
	scope := m.menuPendingSrc
	var opts = install.Options{AddToPath: addToPath}
	if scope == "system" {
		opts.System = true
	} else {
		opts.User = true
	}
	actions, err := m.uiModel.Install(context.Background(), opts)
	if err != nil {
		m.logs = append(m.logs, "install error: "+err.Error())
		m.setNotification("install error: " + err.Error())
		m.menuPendingSrc = ""
		return tea.Tick(5*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
	}
	m.logs = append(m.logs, actions...)
	if len(actions) > 0 {
		m.setNotification(actions[0])
	} else {
		m.setNotification("installed")
	}
	m.menuPendingSrc = ""
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
}

func (m *TuiModel) handleMenuInputUninstallConfirm() tea.Cmd {
	ov := strings.ToLower(strings.TrimSpace(m.menuInput))
	confirm := ov == "y" || ov == "yes"
	if !confirm {
		m.logs = append(m.logs, "uninstall aborted")
		m.setNotification("uninstall aborted")
		return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
	}
	actions, err := m.uiModel.Uninstall(context.Background())
	if err != nil {
		m.logs = append(m.logs, "uninstall error: "+err.Error())
		m.setNotification("uninstall error: " + err.Error())
		return tea.Tick(5*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
	}
	m.logs = append(m.logs, actions...)
	m.setNotification("uninstalled")
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
}

func (m *TuiModel) handleMenuInputExportDB() tea.Cmd {
	dst := m.menuInput
	parent := filepath.Dir(dst)
	if stat, err := os.Stat(parent); err != nil || !stat.IsDir() {
		if d, err2 := config.EnsureDataDir(); err2 == nil {
			dst = filepath.Join(d, filepath.Base(dst))
			m.logs = append(m.logs, "destination invalid; falling back to data dir")
			m.setNotification("destination invalid; falling back to data dir")
		} else {
			m.logs = append(m.logs, "export error: invalid destination and data dir not available")
			m.setNotification("export error: invalid destination and data dir not available")
		}
	}
	dst = uniqueDestPath(dst)
	if err := m.uiModel.Export(context.Background(), "", dst); err != nil {
		m.logs = append(m.logs, "export error: "+err.Error())
		m.setNotification("export error: " + err.Error())
		return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
	}
	m.logs = append(m.logs, "exported database to "+dst)
	m.setNotification("exported database to " + dst)
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
}

func (m *TuiModel) updateListFromCache() {
	// Rebuild list items from cached data and update preview
	newItems := make([]list.Item, 0, len(m.uiModel.ListCached()))
	for _, s := range m.uiModel.ListCached() {
		newItems = append(newItems, csItem{cs: s})
	}
	m.list.SetItems(newItems)
	// select the first item and update preview
	if len(newItems) > 0 {
		m.list.Select(0)
		if it, ok := newItems[0].(csItem); ok {
			if cs, err := m.uiModel.GetCommandSet(context.Background(), it.cs.Name); err == nil {
				m.vp.SetContent(formatCSDetails(cs, m.vp.Width))
			}
		}
	}
}

// renderMenu produces the modal content for the Menu overlay.
func (m *TuiModel) renderMenu() string {
	var b strings.Builder
	title := "Menu"
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0ea5a4")).Render(title))
	b.WriteString("\n\n")
	if m.width >= 80 {
		b.WriteString(m.renderMenuTwoCol())
	} else {
		b.WriteString(m.renderMenuSingleCol())
	}
	if m.menuInputMode {
		b.WriteString("\n")
		b.WriteString(m.renderMenuInputPrompt())
	}
	return b.String()
}

func (m *TuiModel) renderMenuTwoCol() string {
	leftLines, rightLines := m.menuSplitColumns()
	maxLeft := m.menuMaxLeftWidth(leftLines)
	return m.menuRenderPaired(leftLines, rightLines, maxLeft)
}

func (m *TuiModel) menuSplitColumns() ([]string, []string) {
	leftLines := []string{}
	rightLines := []string{}
	for i, it := range m.menuItems {
		line := it
		if i == m.menuIndex {
			line = "> " + line
		} else {
			line = "  " + line
		}
		if i%2 == 0 {
			leftLines = append(leftLines, line)
		} else {
			rightLines = append(rightLines, line)
		}
	}
	return leftLines, rightLines
}

func (m *TuiModel) menuMaxLeftWidth(leftLines []string) int {
	maxLeft := 0
	for _, l := range leftLines {
		if utf8.RuneCountInString(l) > maxLeft {
			maxLeft = utf8.RuneCountInString(l)
		}
	}
	return maxLeft
}

func (m *TuiModel) menuRenderPaired(leftLines, rightLines []string, maxLeft int) string {
	var b strings.Builder
	n := len(leftLines)
	if len(rightLines) > n {
		n = len(rightLines)
	}
	for i := 0; i < n; i++ {
		l := ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		r := ""
		if i < len(rightLines) {
			r = rightLines[i]
		}
		// pad left to maxLeft
		if utf8.RuneCountInString(l) < maxLeft {
			l = l + strings.Repeat(" ", maxLeft-utf8.RuneCountInString(l))
		}
		b.WriteString(l + "  " + r + "\n")
	}
	return b.String()
}

func (m *TuiModel) renderMenuSingleCol() string {
	var b strings.Builder
	for i, it := range m.menuItems {
		prefix := "  "
		if i == m.menuIndex {
			prefix = "> "
			b.WriteString(lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("#fde047")).Render(prefix + it))
		} else {
			b.WriteString(prefix + it)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (m *TuiModel) renderMenuInputPrompt() string {
	switch m.menuAction {
	case "import-db":
		return "Path: " + m.menuInput
	case "import-db-overwrite":
		return "Overwrite destination DB if it exists? [y/N]: " + m.menuInput
	case "import-set":
		return "Path: " + m.menuInput
	case "import-set-policy":
		return "On conflict (rename|skip|overwrite|merge) [rename]: " + m.menuInput
	case "import-set-dedupe":
		return "Dedupe when merging? [y/N]: " + m.menuInput
	case "install-scope":
		return "Install scope (system|user) [user]: " + m.menuInput
	case "install-addpath":
		return "Add to PATH? [y/N]: " + m.menuInput
	case "uninstall-confirm":
		return "Are you sure you want to uninstall? [y/N]: " + m.menuInput
	default:
		return "Path: " + m.menuInput
	}
}
