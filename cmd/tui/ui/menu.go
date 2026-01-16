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
	"github.com/VoxDroid/krnr/internal/user"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// handleMenuKey processes KeyMsg events while the menu modal is active.
func (m *TuiModel) handleMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	// If we are accepting text input for the menu (e.g., import path),
	// handle rune and backspace specially.
	if m.menuInputMode {
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
		if s == "enter" {
			// Validate and perform action
			var notifyCmd tea.Cmd
			switch m.menuAction {
			case "import-db":
				// after entering the source path, validate and advance to overwrite prompt
				if _, err := os.Stat(m.menuInput); err != nil {
					m.logs = append(m.logs, "import error: source not found or inaccessible")
					m.setNotification("import error: source not found or inaccessible")
					notifyCmd = tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
				} else {
					m.menuPendingSrc = m.menuInput
					m.menuAction = "import-db-overwrite"
					m.menuInput = "n" // default to No
					return m, nil
				}
			case "import-db-overwrite":
				// interpret y/N
				ov := strings.ToLower(strings.TrimSpace(m.menuInput))
				overwrite := ov == "y" || ov == "yes"
				// If we are going to overwrite the on-disk DB, close the current
				// adapter/connection before copying so the on-disk replacement takes
				// effect reliably on platforms like Windows that hold file locks.
				if overwrite {
					_ = m.uiModel.Close()
				}
				if err := m.uiModel.ImportDB(context.Background(), m.menuPendingSrc, overwrite); err != nil {
					// If import failed and we closed the active DB, attempt to reopen it so
					// the UI remains usable.
					if overwrite {
						_ = m.uiModel.ReopenDB(context.Background())
					}
					m.logs = append(m.logs, "import error: "+err.Error())
					m.setNotification("import error: " + err.Error())
					notifyCmd = tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
				} else {
					m.logs = append(m.logs, "imported database from "+m.menuPendingSrc)
					m.setNotification("imported database from " + m.menuPendingSrc)
					// Reopen DB connection so the repository uses the freshly-written file
					if err := m.uiModel.ReopenDB(context.Background()); err != nil {
						// if reopen failed, still attempt to refresh list, but surface an error
						m.logs = append(m.logs, "warning: failed to reopen DB: "+err.Error())
						m.setNotification("warning: failed to reopen DB")
					}
					// Refresh list so new DB contents are reflected immediately
					_ = m.uiModel.RefreshList(context.Background())
					// rebuild list items from cached data
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
					notifyCmd = tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
				}
				m.menuPendingSrc = ""
			case "import-set":
				// after entering path, validate and advance to on-conflict prompt
				if _, err := os.Stat(m.menuInput); err != nil {
					m.logs = append(m.logs, "import error: source not found or inaccessible")
					m.setNotification("import error: source not found or inaccessible")
					notifyCmd = tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
				} else {
					m.menuPendingSrc = m.menuInput
					m.menuAction = "import-set-policy"
					m.menuInput = "rename"
					return m, nil
				}
			case "import-set-policy":
				policy := strings.TrimSpace(m.menuInput)
				if policy == "" {
					policy = "rename"
				}
				if policy == "merge" {
					// ask for dedupe option
					m.menuAction = "import-set-dedupe"
					m.menuInput = "n"
					m.menuPendingSrc = m.menuPendingSrc
					m.menuInput = "n"
					return m, nil
				}
				// else perform import with dedupe=false
				if err := m.uiModel.ImportSet(context.Background(), m.menuPendingSrc, policy, false); err != nil {
					m.logs = append(m.logs, "import error: "+err.Error())
					m.setNotification("import error: " + err.Error())
					notifyCmd = tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
				} else {
					m.logs = append(m.logs, "imported command set(s) from "+m.menuPendingSrc)
					m.setNotification("imported command set(s) from " + m.menuPendingSrc)
					// Refresh list so imported sets appear immediately
					_ = m.uiModel.RefreshList(context.Background())
					newItems := make([]list.Item, 0, len(m.uiModel.ListCached()))
					for _, s := range m.uiModel.ListCached() {
						newItems = append(newItems, csItem{cs: s})
					}
					m.list.SetItems(newItems)
					if len(newItems) > 0 {
						m.list.Select(0)
						if it, ok := newItems[0].(csItem); ok {
							if cs, err := m.uiModel.GetCommandSet(context.Background(), it.cs.Name); err == nil {
								m.vp.SetContent(formatCSDetails(cs, m.vp.Width))
							}
						}
					}
					notifyCmd = tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
				}
				m.menuPendingSrc = ""
			case "import-set-dedupe":
				ded := strings.ToLower(strings.TrimSpace(m.menuInput))
				dedupe := ded == "y" || ded == "yes"
				if err := m.uiModel.ImportSet(context.Background(), m.menuPendingSrc, "merge", dedupe); err != nil {
					m.logs = append(m.logs, "import error: "+err.Error())
					m.setNotification("import error: " + err.Error())
					notifyCmd = tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
				} else {
					m.logs = append(m.logs, "imported command set(s) from "+m.menuPendingSrc)
					m.setNotification("imported command set(s) from " + m.menuPendingSrc)
					// Refresh list so imported sets appear immediately
					_ = m.uiModel.RefreshList(context.Background())
					newItems := make([]list.Item, 0, len(m.uiModel.ListCached()))
					for _, s := range m.uiModel.ListCached() {
						newItems = append(newItems, csItem{cs: s})
					}
					m.list.SetItems(newItems)
					if len(newItems) > 0 {
						m.list.Select(0)
						if it, ok := newItems[0].(csItem); ok {
							if cs, err := m.uiModel.GetCommandSet(context.Background(), it.cs.Name); err == nil {
								m.vp.SetContent(formatCSDetails(cs, m.vp.Width))
							}
						}
					}
					notifyCmd = tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
				}
				m.menuPendingSrc = ""

			case "export-db":
				// Ensure directory exists and is writable. If invalid, fallback to data dir.
				dst := m.menuInput
				parent := filepath.Dir(dst)
				if stat, err := os.Stat(parent); err != nil || !stat.IsDir() {
					// fallback
					if d, err2 := config.EnsureDataDir(); err2 == nil {
						dst = filepath.Join(d, filepath.Base(dst))
						m.logs = append(m.logs, "destination invalid; falling back to data dir")
						m.setNotification("destination invalid; falling back to data dir")
						notifyCmd = tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
					} else {
						m.logs = append(m.logs, "export error: invalid destination and data dir not available")
						m.setNotification("export error: invalid destination and data dir not available")
						dst = dst // will likely fail
						notifyCmd = tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
					}
				}
				// if dest exists, pick unique path to avoid overwriting
				dst = uniqueDestPath(dst)
				if err := m.uiModel.Export(context.Background(), "", dst); err != nil {
					m.logs = append(m.logs, "export error: "+err.Error())
					m.setNotification("export error: " + err.Error())
					notifyCmd = tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
				} else {
					m.logs = append(m.logs, "exported database to "+dst)
					m.setNotification("exported database to " + dst)
					notifyCmd = tea.Tick(3*time.Second, func(time.Time) tea.Msg { return clearNotificationMsg{} })
				}
			}

			// clear menu state
			m.menuInputMode = false
			m.menuInput = ""
			m.menuAction = ""
			m.showMenu = false
			return m, notifyCmd
		}
		if s == "esc" {
			m.menuInputMode = false
			m.menuInput = ""
			m.menuAction = ""
			m.showMenu = false
			return m, nil
		}
		return m, nil
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
		item := m.menuItems[m.menuIndex]
		switch item {
		case "Export database":
			// Prompt for destination path (prefill with PWD default)
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
			// Installation UI is outside scope; log and close menu
			if si := m.list.SelectedItem(); si != nil {
				if it, ok := si.(csItem); ok {
					_ = m.uiModel.Install(context.Background(), it.cs.Name)
					m.logs = append(m.logs, "installed "+it.cs.Name)
				}
			}
			m.showMenu = false
		case "Uninstall":
			if si := m.list.SelectedItem(); si != nil {
				if it, ok := si.(csItem); ok {
					_ = m.uiModel.Uninstall(context.Background(), it.cs.Name)
					m.logs = append(m.logs, "uninstalled "+it.cs.Name)
				}
			}
			m.showMenu = false
		case "Status":
			// placeholder: show a status log entry
			m.logs = append(m.logs, "status: OK")
			m.showMenu = false
		case "Whoami":
			if p, ok, err := user.GetProfile(); err == nil && ok {
				m.logs = append(m.logs, "whoami: "+strings.TrimSpace(p.Name+" <"+p.Email+">"))
			} else if err != nil {
				m.logs = append(m.logs, "whoami: error: "+err.Error())
			} else {
				m.logs = append(m.logs, "whoami: (not set)")
			}
			m.showMenu = false
		case "Close":
			m.showMenu = false
		}
	}
	return m, nil
}

// renderMenu produces the modal content for the Menu overlay.
func (m *TuiModel) renderMenu() string {
	var b strings.Builder
	title := "Menu"
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0ea5a4")).Render(title))
	b.WriteString("\n\n")
	// render menu as two columns when wide enough for visual balance
	if m.width >= 80 {
		// two-column layout: split items into left/right columns and align
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
		// compute left column width
		maxLeft := 0
		for _, l := range leftLines {
			if utf8.RuneCountInString(l) > maxLeft {
				maxLeft = utf8.RuneCountInString(l)
			}
		}
		// render paired lines
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
	} else {
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
	}
	if m.menuInputMode {
		b.WriteString("\n")
		switch m.menuAction {
		case "import-db":
			b.WriteString("Path: ")
			b.WriteString(m.menuInput)
		case "import-db-overwrite":
			b.WriteString("Overwrite destination DB if it exists? [y/N]: ")
			b.WriteString(m.menuInput)
		case "import-set":
			b.WriteString("Path: ")
			b.WriteString(m.menuInput)
		case "import-set-policy":
			b.WriteString("On conflict (rename|skip|overwrite|merge) [rename]: ")
			b.WriteString(m.menuInput)
		case "import-set-dedupe":
			b.WriteString("Dedupe when merging? [y/N]: ")
			b.WriteString(m.menuInput)
		default:
			b.WriteString("Path: ")
			b.WriteString(m.menuInput)
		}
	}
	return b.String()
}
