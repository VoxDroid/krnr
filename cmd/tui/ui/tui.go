package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
)

// TuiModel is the Bubble Tea model used by cmd/tui.
type TuiModel struct {
	uiModel *modelpkg.UIModel
	list    list.Model
	vp      viewport.Model

	width  int
	height int

	showDetail bool
	detail     string
	detailName string
	// delete confirmation state
	pendingDelete     bool
	pendingDeleteName string
	// export confirmation state
	pendingExport     bool
	pendingExportName string
	pendingExportDest string
	runInProgress     bool
	logs              []string
	cancelRun         func()
	runCh             chan adapters.RunEvent
	// accessibility / theme
	themeHighContrast bool
	// track last selected name so we can detect changes and update preview
	lastSelectedName string
	// focus: false = left pane (list), true = right pane (viewport)
	focusRight bool

	// editing metadata modal state
	editingMeta bool
	editor      struct {
		field    int // 0=name,1=desc,2=tags,3=commands
		name     string
		desc     string
		tags     string
		commands []string
		cmdIndex int
	}

	// versions / rollback state
	versions               []adapters.Version
	versionsSelected       int
	versionsList           list.Model
	versionsOffset         int
	pendingRollback        bool
	pendingRollbackVersion int
	pendingRollbackName    string
}

// Messages
type runEventMsg adapters.RunEvent
type runDoneMsg struct{}

// NewModel constructs the Bubble Tea TUI model used by cmd/tui.
func NewModel(ui *modelpkg.UIModel) *TuiModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "krnr — command sets"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	vp := viewport.New(0, 0)
	vlist := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	vlist.SetShowStatusBar(false)
	vlist.SetFilteringEnabled(false)

	return &TuiModel{uiModel: ui, list: l, vp: vp, versionsList: vlist}
}

// NewProgram constructs the tea.Program for the TUI.
func NewProgram(ui *modelpkg.UIModel) *tea.Program {
	m := NewModel(ui)
	p := tea.NewProgram(m, tea.WithAltScreen())
	return p
}

// Init initializes the model by refreshing the list and populating the preview.
func (m *TuiModel) Init() tea.Cmd {
	// load initial list
	return func() tea.Msg {
		_ = m.uiModel.RefreshList(context.Background())
		items := make([]list.Item, 0, len(m.uiModel.ListCached()))
		for _, s := range m.uiModel.ListCached() {
			items = append(items, csItem{cs: s})
		}

		// Ensure the list and viewport have reasonable defaults so the UI shows
		// content on first render (before a WindowSizeMsg arrives).
		if m.list.Height() == 0 {
			m.list.SetSize(30, 10)
		}
		if m.vp.Width == 0 || m.vp.Height == 0 {
			m.vp = viewport.New(40, 12)
		}

		// Set items after sizing so the list delegate can render immediately
		m.list.SetItems(items)

		// If there are items, select the first and populate the preview
		if len(items) > 0 {
			m.list.Select(0)
			if it, ok := items[0].(csItem); ok {
				m.lastSelectedName = it.cs.Name
				// attempt to fetch full details, fall back to summary
				if cs, err := m.uiModel.GetCommandSet(context.Background(), it.cs.Name); err == nil {
					m.vp.SetContent(formatCSDetails(cs, m.vp.Width))
				} else {
					m.vp.SetContent(formatCSDetails(it.cs, m.vp.Width))
				}
			}
		}
		return nil
	}
}

// Update handles incoming events and updates model state.
func (m *TuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		s := msg.String()
		// If we're editing metadata, consume keys for the editor first
		if m.editingMeta {
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
			case "up", "k":
				if m.editor.field == 3 && m.editor.cmdIndex > 0 {
					m.editor.cmdIndex--
				}
				return m, nil
			case "down", "j":
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
					m.logs = append(m.logs, "update: "+err.Error())
					return m, nil
				}
				// replace commands
				if err := m.uiModel.ReplaceCommands(context.Background(), newCS.Name, filterEmptyLines(m.editor.commands)); err != nil {
					m.logs = append(m.logs, "replace commands: "+err.Error())
					return m, nil
				}
				// refresh detail
				if cs, err := m.uiModel.GetCommandSet(context.Background(), newCS.Name); err == nil {
					m.detailName = cs.Name
					m.detail = formatCSFullScreen(cs, m.width, m.height)
					m.vp.SetContent(m.detail)
				}
				m.editingMeta = false
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
		}
		// If the list is in filtering state, let the list consume all keys
		// (including keys that would otherwise be global controls like q/esc).
		if m.list.FilterState() == list.Filtering {
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}
		// global keybindings handled BEFORE passing to the list so they are
		// not swallowed when filtering is enabled.
		switch s {
		case "q", "esc":
			return m, tea.Quit
		case "?":
			m.showDetail = true
			m.detail = "Help:\n\n? show help\nq or Esc to quit\nEnter to view details\n/ to filter\n← → or Tab to switch pane focus\n↑ ↓ to scroll focused pane"
			return m, nil
		case "enter":
			// If we are showing details and the right pane (versions) is focused,
			// forward Enter to the versions list instead of reopening a new detail
			if m.showDetail && m.focusRight && len(m.versions) > 0 {
				m.versionsList, cmd = m.versionsList.Update(msg)
				m.versionsSelected = m.versionsList.Index()
				if si := m.versionsList.SelectedItem(); si != nil {
					if vi, ok := si.(verItem); ok {
						m.detail = formatVersionPreview(m.detailName, vi.v, m.width-40, m.height)
						m.vp.SetContent(m.detail)
					}
				}
				return m, cmd
			}
			if i, ok := m.list.SelectedItem().(csItem); ok {
				m.showDetail = true
				m.detailName = i.cs.Name
				// fetch the full set (including commands) if possible and render
				if cs, err := m.uiModel.GetCommandSet(context.Background(), i.cs.Name); err == nil {
					m.detail = formatCSFullScreen(cs, m.width, m.height)
					m.vp.SetContent(m.detail)
				} else {
					m.detail = formatCSDetails(i.cs, m.width/2)
					m.vp.SetContent(m.detail)
				}
				// fetch versions for right-side panel
				if vers, err := m.uiModel.ListVersions(context.Background(), i.cs.Name); err == nil {
					m.versions = vers
					m.versionsOffset = 0
					// populate versions list items
					items := make([]list.Item, 0, len(vers))
					for _, v := range vers {
						items = append(items, verItem{v: v})
					}
					m.versionsList.SetItems(items)
					if len(vers) > 0 {
						m.versionsSelected = 0
						m.versionsList.Select(0)
					}
					// compute inner sizes and set list size so it adapts even without a recent WS
					headH := 1
					footerH := 1
					bodyH := m.height - headH - footerH - 2
					if bodyH < 3 {
						bodyH = 3
					}
					sideW := int(float64(m.width) * 0.35)
					if sideW > 36 {
						sideW = 36
					}
					if sideW < 20 {
						sideW = 20
					}
					rightW := m.width - sideW - 4
					if rightW < 12 {
						rightW = 12
					}
					innerRightW := rightW - 2
					if innerRightW < 10 {
						innerRightW = 10
					}
					innerBodyH := bodyH - 2
					if innerBodyH < 1 {
						innerBodyH = 1
					}
					previewH := 8
					if previewH > innerBodyH/2 {
						previewH = innerBodyH / 2
					}
					available := innerBodyH - previewH - 4
					if available < 1 {
						available = 1
					}
					m.versionsList.SetSize(innerRightW, available)
				} else {
					m.versions = nil
					m.versionsSelected = 0
					m.versionsOffset = 0
					m.versionsList.SetItems([]list.Item{})
				}
			}
			return m, nil
		case "b":
			m.showDetail = false
			m.focusRight = false // reset focus to left pane when going back
			// Restore the viewport content to show the preview of the selected item
			if si := m.list.SelectedItem(); si != nil {
				if it, ok := si.(csItem); ok {
					// Fetch full details for the preview pane
					if cs, err := m.uiModel.GetCommandSet(context.Background(), it.cs.Name); err == nil {
						m.vp.SetContent(formatCSDetails(cs, m.vp.Width))
					} else {
						m.vp.SetContent(formatCSDetails(it.cs, m.vp.Width))
					}
				}
			}
			return m, nil
		case "e":
			// Open in-TUI metadata editor for the selected command set when in detail view
			if !m.showDetail {
				return m, nil
			}
			if i, ok := m.list.SelectedItem().(csItem); ok {
				name := i.cs.Name
				cs, err := m.uiModel.GetCommandSet(context.Background(), name)
				if err != nil {
					m.logs = append(m.logs, "edit: get: "+err.Error())
					return m, nil
				}
				// populate modal editor state
				m.editingMeta = true
				m.editor.field = 0
				m.editor.name = cs.Name
				m.editor.desc = cs.Description
				m.editor.tags = strings.Join(cs.Tags, ",")
				m.editor.commands = append([]string{}, cs.Commands...)
				if len(m.editor.commands) == 0 {
					m.editor.commands = []string{""}
				}
				m.editor.cmdIndex = 0
			}
			return m, nil
		case "d":
			// initiate delete confirmation (only valid in detail view)
			if !m.showDetail {
				return m, nil
			}
			var name string
			if i, ok := m.list.SelectedItem().(csItem); ok {
				name = i.cs.Name
			} else if m.detailName != "" {
				name = m.detailName
			}
			if name == "" {
				return m, nil
			}
			m.pendingDelete = true
			m.pendingDeleteName = name
			m.detail = fmt.Sprintf("Delete '%s' permanently? [y/N]\n\nPress (y) to confirm, (n) or (b) to cancel", name)
			m.vp.SetContent(m.detail)
			return m, nil
		case "s":
			// export selected set to a temp file (confirm)
			if !m.showDetail {
				return m, nil
			}
			var ename string
			if i, ok := m.list.SelectedItem().(csItem); ok {
				ename = i.cs.Name
			} else if m.detailName != "" {
				ename = m.detailName
			}
			if ename == "" {
				return m, nil
			}
			// default dest in temp dir
			dflt := filepath.Join(os.TempDir(), ename+".db")
			// ensure we pick a path that doesn't already exist to avoid constraint errors
			dest := uniqueDestPath(dflt)
			m.pendingExport = true
			m.pendingExportName = ename
			m.pendingExportDest = dest
			m.detail = fmt.Sprintf("Export '%s' to:\n\n%s\n\nPress (y) to confirm, (n) to cancel", ename, dest)
			m.vp.SetContent(m.detail)
			return m, nil
		case "y", "Y":
			if m.pendingRollback {
				name := m.pendingRollbackName
				ver := m.pendingRollbackVersion
				if err := m.uiModel.ApplyVersion(context.Background(), name, ver); err != nil {
					m.logs = append(m.logs, "rollback error: "+err.Error())
					m.detail = fmt.Sprintf("Rollback failed: %s", err.Error())
					m.vp.SetContent(m.detail)
				} else {
					m.logs = append(m.logs, fmt.Sprintf("rolled back '%s' to v%d", name, ver))
					m.detail = fmt.Sprintf("Rolled back '%s' to version %d", name, ver)
					m.vp.SetContent(m.detail)
					// refresh versions list and detail content
					if vers, err := m.uiModel.ListVersions(context.Background(), name); err == nil {
						m.versions = vers
						m.versionsSelected = 0
					}
					if cs, err := m.uiModel.GetCommandSet(context.Background(), name); err == nil {
						m.detail = formatCSFullScreen(cs, m.width, m.height)
						m.vp.SetContent(m.detail)
					}
				}
				m.pendingRollback = false
				m.pendingRollbackName = ""
				m.pendingRollbackVersion = 0
				return m, nil
			}
			if m.pendingDelete {
				name := m.pendingDeleteName
				if err := m.uiModel.Delete(context.Background(), name); err != nil {
					m.logs = append(m.logs, "delete error: "+err.Error())
					m.pendingDelete = false
					return m, nil
				}
				// refresh list and preview
				_ = m.uiModel.RefreshList(context.Background())
				items := make([]list.Item, 0, len(m.uiModel.ListCached()))
				for _, s := range m.uiModel.ListCached() {
					items = append(items, csItem{cs: s})
				}
				m.list.SetItems(items)
				m.logs = append(m.logs, fmt.Sprintf("deleted '%s'", name))
				m.pendingDelete = false
				m.pendingDeleteName = ""
				m.showDetail = false
				// select first item and refresh preview if exists
				if len(items) > 0 {
					m.list.Select(0)
					if it, ok := items[0].(csItem); ok {
						if cs, err := m.uiModel.GetCommandSet(context.Background(), it.cs.Name); err == nil {
							m.vp.SetContent(formatCSDetails(cs, m.vp.Width))
						} else {
							m.vp.SetContent(formatCSDetails(it.cs, m.vp.Width))
						}
					}
				} else {
					m.vp.SetContent("")
				}
				// show an explicit confirmation in the detail pane
				m.detail = fmt.Sprintf("Deleted '%s'", name)
				m.vp.SetContent(m.detail)
				return m, nil
			}
			if m.pendingExport {
				name := m.pendingExportName
				dest := m.pendingExportDest
				if err := m.uiModel.Export(context.Background(), name, dest); err != nil {
					m.logs = append(m.logs, "export error: "+err.Error())
					m.detail = fmt.Sprintf("Export failed: %s", err.Error())
					m.vp.SetContent(m.detail)
				} else {
					m.logs = append(m.logs, fmt.Sprintf("exported '%s' to %s", name, dest))
					m.detail = fmt.Sprintf("Exported '%s' to %s", name, dest)
					m.vp.SetContent(m.detail)
				}
				m.pendingExport = false
				m.pendingExportName = ""
				m.pendingExportDest = ""
				return m, nil
			}
			m.logs = append(m.logs, "no pending action to confirm")
			return m, nil
		case "n", "N":
			if m.pendingRollback {
				m.pendingRollback = false
				m.pendingRollbackName = ""
				m.pendingRollbackVersion = 0
				// restore detail
				name := m.detailName
				if name == "" {
					if si := m.list.SelectedItem(); si != nil {
						if it, ok := si.(csItem); ok {
							name = it.cs.Name
						}
					}
				}
				if name != "" {
					if cs, err := m.uiModel.GetCommandSet(context.Background(), name); err == nil {
						m.detail = formatCSFullScreen(cs, m.width, m.height)
						m.vp.SetContent(m.detail)
					}
				}
				return m, nil
			}
			if m.pendingDelete {
				m.pendingDelete = false
				// restore detail of the selected (or the cached name)
				name := m.detailName
				if name == "" {
					if si := m.list.SelectedItem(); si != nil {
						if it, ok := si.(csItem); ok {
							name = it.cs.Name
						}
					}
				}
				if name != "" {
					if cs, err := m.uiModel.GetCommandSet(context.Background(), name); err == nil {
						m.detail = formatCSFullScreen(cs, m.width, m.height)
						m.vp.SetContent(m.detail)
					}
				}
				return m, nil
			}
			if m.pendingExport {
				m.pendingExport = false
				name := m.detailName
				if name == "" {
					if si := m.list.SelectedItem(); si != nil {
						if it, ok := si.(csItem); ok {
							name = it.cs.Name
						}
					}
				}
				if name != "" {
					if cs, err := m.uiModel.GetCommandSet(context.Background(), name); err == nil {
						m.detail = formatCSFullScreen(cs, m.width, m.height)
						m.vp.SetContent(m.detail)
					}
				}
				return m, nil
			}
			return m, nil
		case "r":
			// run selected (fall back to first item if none selected)
			if m.runInProgress {
				return m, nil
			}
			var name string
			if i, ok := m.list.SelectedItem().(csItem); ok {
				name = i.cs.Name
			} else if len(m.list.Items()) > 0 {
				if it, ok := m.list.Items()[0].(csItem); ok {
					name = it.cs.Name
				}
			}
			if name == "" {
				return m, nil
			}
			m.logs = nil
			m.runInProgress = true
			m.focusRight = true
			// start run via model
			ctx, cancel := context.WithCancel(context.Background())
			m.cancelRun = cancel
			h, err := m.uiModel.Run(ctx, name, nil)
			if err != nil {
				m.logs = append(m.logs, "run error: "+err.Error())
				m.runInProgress = false
				return m, nil
			}
			// stream events into a channel and return a command that reads from it
			ch := make(chan adapters.RunEvent)
			m.runCh = ch
			go func() {
				for ev := range h.Events() {
					ch <- ev
				}
				close(ch)
			}()
			m.runInProgress = true
			return m, readLoop(m.runCh)
		case "T", "t":
			m.themeHighContrast = !m.themeHighContrast
			return m, nil
		case "left":
			// Switch focus to left pane (list)
			m.focusRight = false
			return m, nil
		case "right":
			// Switch focus to right pane (viewport)
			m.focusRight = true
			return m, nil
		case "tab":
			// Toggle focus between panes
			m.focusRight = !m.focusRight
			return m, nil
		}
		// non-printable bindings
		if msg.Type == tea.KeyCtrlT {
			m.themeHighContrast = !m.themeHighContrast
			return m, nil
		}

		// Hand message to the list so filtering input will be processed
		// When filtering, let list handle keys
		if m.list.FilterState() == list.Filtering {
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}
		// Handle scrolling based on which pane has focus
		if m.focusRight {
			// If we're in the detail view and versions are present, use the right
			// pane's list for navigation and selection instead of manually handling keys.
			if m.showDetail && len(m.versions) > 0 {
				switch s {
				case "R":
					// initiate rollback confirmation for selected version
					if m.versionsSelected >= 0 && m.versionsSelected < len(m.versions) {
						v := m.versions[m.versionsSelected]
						m.pendingRollback = true
						m.pendingRollbackVersion = v.Version
						m.pendingRollbackName = m.detailName
						m.detail = fmt.Sprintf("Rollback '%s' to version %d? [y/N]\n\nPress (y) to confirm, (n) to cancel", m.pendingRollbackName, m.pendingRollbackVersion)
						m.vp.SetContent(m.detail)
					}
					return m, nil
				default:
					// Forward navigation keys to the versions list so it handles paging
					var listCmd tea.Cmd
					m.versionsList, listCmd = m.versionsList.Update(msg)
					m.versionsSelected = m.versionsList.Index()
					if si := m.versionsList.SelectedItem(); si != nil {
						if vi, ok := si.(verItem); ok {
							m.detail = formatVersionPreview(m.detailName, vi.v, m.width-40, m.height)
							m.vp.SetContent(m.detail)
						}
					}
					return m, listCmd
				}
			}
			// Right pane focused - scroll viewport (fallback)
			switch s {
			case "up", "k":
				m.vp.ScrollUp(1)
				return m, nil
			case "down", "j":
				m.vp.ScrollDown(1)
				return m, nil
			case "pgup":
				m.vp.HalfPageUp()
				return m, nil
			case "pgdown":
				m.vp.HalfPageDown()
				return m, nil
			case "home":
				m.vp.GotoTop()
				return m, nil
			case "end":
				m.vp.GotoBottom()
				return m, nil
			}
		}
		// Left pane focused or other keys - pass to list

		m.list, cmd = m.list.Update(msg)

		if s == "/" {
			// filter mode entry is already handled by the list; just return
			return m, cmd
		}

		// If the selection changed, update the preview pane by fetching the
		// full CommandSet (including commands) from the model and rendering it.
		if si := m.list.SelectedItem(); si != nil {
			if it, ok := si.(csItem); ok {
				if it.cs.Name != m.lastSelectedName {
					// attempt to fetch full details (may include commands)
					if cs, err := m.uiModel.GetCommandSet(context.Background(), it.cs.Name); err == nil {
						m.vp.SetContent(formatCSDetails(cs, m.vp.Width))
					}
					m.lastSelectedName = it.cs.Name
				}
			}
		}

	case runEventMsg:
		ev := adapters.RunEvent(msg)
		if ev.Err != nil {
			m.logs = append(m.logs, "err: "+ev.Err.Error())
			m.runInProgress = false
			m.runCh = nil
			return m, nil
		}
		m.logs = append(m.logs, ev.Line)
		// keep viewport scrolled to bottom
		m.vp.SetContent(strings.Join(m.logs, "\n"))
		m.vp.GotoBottom()
		// continue reading
		if m.runCh != nil {
			return m, readLoop(m.runCh)
		}
		return m, nil
	case runDoneMsg:
		m.runInProgress = false
		m.runCh = nil
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headH := 1
		footerH := 1
		bodyH := m.height - headH - footerH - 2
		if bodyH < 3 {
			bodyH = 3
		}

		sideW := int(float64(m.width) * 0.35)
		if sideW > 36 {
			sideW = 36
		}
		if sideW < 20 {
			sideW = 20
		}
		innerSideW := sideW - 2
		if innerSideW < 10 {
			innerSideW = 10
		}

		rightW := m.width - sideW - 4
		if rightW < 12 {
			rightW = 12
		}
		innerRightW := rightW - 2
		if innerRightW < 10 {
			innerRightW = 10
		}

		innerBodyH := bodyH - 2
		if innerBodyH < 1 {
			innerBodyH = 1
		}

		m.list.SetSize(innerSideW, innerBodyH)
		m.vp = viewport.New(innerRightW, innerBodyH)

		// configure versionsList sizing to match the inner area when present
		if len(m.versions) > 0 {
			previewH := 8
			if previewH > innerBodyH/2 {
				previewH = innerBodyH / 2
			}
			available := innerBodyH - previewH - 4
			if available < 1 {
				available = 1
			}
			m.versionsList.SetSize(innerRightW, available)
		}

		// update content for selected
		if si := m.list.SelectedItem(); si != nil {
			if cs, ok := si.(csItem); ok {
				// construct a full metadata view
				full := formatCSDetails(cs.cs, m.vp.Width)
				m.vp.SetContent(full)
			}
		}
	}

	return m, cmd
}

// readLoop returns a command that reads one event from the channel and
// returns it as a tea.Msg. The caller should return the readLoop command
// again from Update to continue the stream.
func readLoop(ch <-chan adapters.RunEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return runDoneMsg{}
		}
		return runEventMsg(ev)
	}
}

// simple word-wrap to produce lines no longer than width (approximate by rune count)
func wrapText(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	out := []string{}
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		cur := words[0]
		for _, w := range words[1:] {
			if utf8.RuneCountInString(cur)+1+utf8.RuneCountInString(w) > width {
				out = append(out, cur)
				cur = w
			} else {
				cur = cur + " " + w
			}
		}
		out = append(out, cur)
	}
	return out
}

// renderTwoCol renders a prefix in a fixed-width left column and wraps the
// text into the right column. Returns the joined lines.
func renderTwoCol(prefix, text string, prefixW, textW int) string {
	if prefixW < 0 {
		prefixW = 0
	}
	if textW < 0 {
		textW = 0
	}
	lines := wrapText(text, textW)
	var b strings.Builder
	for i, ln := range lines {
		var left string
		if i == 0 {
			// right-align prefix within prefixW
			padded := prefix
			if utf8.RuneCountInString(padded) < prefixW {
				padded = strings.Repeat(" ", prefixW-utf8.RuneCountInString(padded)) + padded
			}
			left = padded
		} else {
			left = strings.Repeat(" ", prefixW)
		}
		// ensure right text is not longer than textW (wrapText took care of it)
		b.WriteString(left + " " + ln + "\n")
	}
	return b.String()
}

// renderTableInline renders a label on the left and the value on the same line
// when possible. Values are wrapped to valueW and continuation lines are
// aligned under the value column.
func renderTableInline(label, value string, labelW, valueW int) string {
	if labelW < 0 {
		labelW = 0
	}
	if valueW < 0 {
		valueW = 0
	}
	lines := wrapText(value, valueW)
	var b strings.Builder
	for i, ln := range lines {
		if i == 0 {
			padded := label
			if utf8.RuneCountInString(padded) < labelW {
				padded = padded + strings.Repeat(" ", labelW-utf8.RuneCountInString(padded))
			}
			b.WriteString(padded + " " + ln + "\n")
		} else {
			b.WriteString(strings.Repeat(" ", labelW) + " " + ln + "\n")
		}
	}
	// if value is empty, still render the label alone
	if len(lines) == 0 {
		padded := label
		if utf8.RuneCountInString(padded) < labelW {
			padded = padded + strings.Repeat(" ", labelW-utf8.RuneCountInString(padded))
		}
		b.WriteString(padded + "\n")
	}
	return b.String()
}

// renderTableBlockHeader renders the label as a header line and places the
// (already wrapped) block lines underneath it, aligned to the value column.
func renderTableBlockHeader(label, block string, labelW int) string {
	if labelW < 0 {
		labelW = 0
	}
	lines := strings.Split(strings.TrimSuffix(block, "\n"), "\n")
	var b strings.Builder
	// render header
	padded := label
	if utf8.RuneCountInString(padded) < labelW {
		padded = padded + strings.Repeat(" ", labelW-utf8.RuneCountInString(padded))
	}
	b.WriteString(padded + "\n")
	// render block lines under the header
	for _, ln := range lines {
		b.WriteString(strings.Repeat(" ", labelW) + " " + ln + "\n")
	}
	return b.String()
}

// simulateOutput attempts to produce a simple dry-run output for well-known
// commands (e.g., `echo ...`). For unknown commands we fall back to the
// literal dry-run representation.
func simulateOutput(cmd string) string {
	trim := strings.TrimSpace(cmd)
	if strings.HasPrefix(trim, "echo ") {
		return strings.TrimSpace(strings.TrimPrefix(trim, "echo "))
	}
	return "$ " + cmd
}

func formatCSFullScreen(cs adapters.CommandSetSummary, width int, _ int) string {
	// Colored headings to match the main UI's visual style
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0ea5a4")).Background(lipgloss.Color("#0b1226"))
	h := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0ea5a4"))
	k := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#94a3b8"))
	dryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#94a3b8")).Italic(true)

	var b strings.Builder
	contentW := width - 6
	if contentW < 10 {
		contentW = 10
	}

	// Title header inside the container
	titleText := fmt.Sprintf("krnr — %s Details", cs.Name)
	b.WriteString(titleStyle.Render(titleText) + "\n")
	// Separator line
	sepLen := contentW
	if sepLen > len(titleText)+4 {
		sepLen = len(titleText) + 4
	}
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#0ea5a4")).Render(strings.Repeat("─", sepLen)) + "\n\n")

	// compute label column width (invisible border table)
	labels := []string{"Name:", "Description:", "Commands:", "Metadata:"}
	labelW := 0
	for _, l := range labels {
		if utf8.RuneCountInString(l) > labelW {
			labelW = utf8.RuneCountInString(l)
		}
	}
	valueW := contentW - labelW - 1
	if valueW < 10 {
		valueW = 10
	}

	// Name inline
	b.WriteString(h.Render("Name:") + " " + cs.Name + "\n")

	// Description header + wrapped lines
	if cs.Description != "" {
		lines := wrapText(cs.Description, valueW)
		b.WriteString("\n")
		b.WriteString(h.Render("Description:") + "\n")
		b.WriteString(renderTableBlockHeader("", strings.Join(lines, "\n"), labelW))
	}

	// Commands: render inner two-column block then place under Commands: header
	if len(cs.Commands) > 0 {
		b.WriteString("\n")
		b.WriteString(h.Render("Commands:") + "\n")
		maxPrefix := 0
		for i := range cs.Commands {
			p := fmt.Sprintf("%d) ", i+1)
			if l := utf8.RuneCountInString(p); l > maxPrefix {
				maxPrefix = l
			}
		}
		// inner text width for commands inside value column
		innerTextW := valueW - maxPrefix - 1
		if innerTextW < 10 {
			innerTextW = 10
		}
		var cb strings.Builder
		for i, c := range cs.Commands {
			p := fmt.Sprintf("%d) ", i+1)
			cb.WriteString(renderTwoCol(p, c, maxPrefix, innerTextW))
		}
		b.WriteString(renderTableBlockHeader("", strings.TrimSuffix(cb.String(), "\n"), labelW))
	}

	// Dry-run preview — show simulated output where possible
	if len(cs.Commands) > 0 {
		b.WriteString("\n")
		b.WriteString(h.Render("Dry-run preview:") + "\n")
		for _, c := range cs.Commands {
			out := simulateOutput(c)
			b.WriteString(dryStyle.Render(out) + "\n")
		}
	}

	// Metadata
	meta := []string{}
	if cs.AuthorName != "" {
		meta = append(meta, "Author: "+cs.AuthorName)
	}
	if cs.AuthorEmail != "" {
		meta = append(meta, "Email: "+cs.AuthorEmail)
	}
	if cs.CreatedAt != "" {
		meta = append(meta, "Created: "+cs.CreatedAt)
	}
	if cs.LastRun != "" {
		meta = append(meta, "Last run: "+cs.LastRun)
	}
	if len(cs.Tags) > 0 {
		meta = append(meta, "Tags: "+strings.Join(cs.Tags, ", "))
	}
	if len(meta) > 0 {
		b.WriteString("\n")
		b.WriteString(h.Render("Metadata:") + "\n")
		for _, m := range meta {
			b.WriteString(k.Render("  "+m) + "\n")
		}
	}
	return b.String()
}

func formatCSDetails(cs adapters.CommandSetSummary, width int) string {
	// invisible table layout — keep formatting simple and predictable for tests

	var b strings.Builder
	contentW := width - 4
	if contentW < 10 {
		contentW = 10
	}
	// label column width
	labels := []string{"Name:", "Description:", "Commands:", "Metadata:"}
	labelW := 0
	for _, l := range labels {
		if utf8.RuneCountInString(l) > labelW {
			labelW = utf8.RuneCountInString(l)
		}
	}
	valueW := contentW - labelW - 1
	if valueW < 10 {
		valueW = 10
	}

	// Name inline
	b.WriteString(renderTableInline("Name:", cs.Name, labelW, valueW))

	// Description as header + lines
	if cs.Description != "" {
		lines := wrapText(cs.Description, valueW)
		b.WriteString("\n")
		b.WriteString(renderTableBlockHeader("Description:", strings.Join(lines, "\n"), labelW))
	}

	// Commands
	if len(cs.Commands) > 0 {
		b.WriteString("\n")
		maxPrefix := 0
		for i := range cs.Commands {
			p := fmt.Sprintf("%d) ", i+1)
			if l := utf8.RuneCountInString(p); l > maxPrefix {
				maxPrefix = l
			}
		}
		innerTextW := valueW - maxPrefix - 1
		if innerTextW < 10 {
			innerTextW = 10
		}
		var cb strings.Builder
		for i, c := range cs.Commands {
			p := fmt.Sprintf("%d) ", i+1)
			cb.WriteString(renderTwoCol(p, c, maxPrefix, innerTextW))
		}
		b.WriteString(renderTableBlockHeader("Commands:", strings.TrimSuffix(cb.String(), "\n"), labelW))
	}

	// Metadata
	meta := []string{}
	if cs.AuthorName != "" {
		meta = append(meta, "Author: "+cs.AuthorName)
	}
	if cs.AuthorEmail != "" {
		meta = append(meta, "Email: "+cs.AuthorEmail)
	}
	if cs.CreatedAt != "" {
		meta = append(meta, "Created: "+cs.CreatedAt)
	}
	if cs.LastRun != "" {
		meta = append(meta, "Last run: "+cs.LastRun)
	}
	if len(cs.Tags) > 0 {
		meta = append(meta, "Tags: "+strings.Join(cs.Tags, ", "))
	}
	if len(meta) > 0 {
		b.WriteString("\n")
		b.WriteString(renderTableBlockHeader("Metadata:", strings.TrimSuffix(strings.Join(meta, "\n"), "\n"), labelW))
	}
	return b.String()
}

// uniqueDestPath returns a non-existing destination path by appending
// a numeric suffix before the extension if needed (e.g., name-1.db).
func uniqueDestPath(base string) string {
	if _, err := os.Stat(base); err == nil {
		// file exists — try appended counters
		root := strings.TrimSuffix(base, filepath.Ext(base))
		ext := filepath.Ext(base)
		for i := 1; ; i++ {
			cand := fmt.Sprintf("%s-%d%s", root, i, ext)
			if _, err := os.Stat(cand); err != nil {
				return cand
			}
		}
	}
	return base
}

// View renders the current TUI view as a string.
func (m *TuiModel) View() string {
	if m.showDetail {
		// Use the same top title bar and content container approach as the main
		// page so borders and colors remain consistent across views.
		var rightBorder, bottomBg, bottomFg string
		if m.themeHighContrast {
			rightBorder = "#ffffff"
			bottomBg, bottomFg = "#000000", "#ffffff"
		} else {
			rightBorder = "#c084fc"
			bottomBg, bottomFg = "#0b1226", "#cbd5e1"
		}

		footerH := 1
		bottomH := 1
		bodyH := m.height - footerH - bottomH - 2
		if bodyH < 3 {
			bodyH = 3
		}

		contentStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(rightBorder)).
			Padding(1).
			Width(m.width - 2).
			Height(bodyH)
		var body string
		if m.editingMeta {
			body = contentStyle.Render(m.renderEditor())
		} else if len(m.versions) > 0 {
			// split the content region into main detail (left) and versions (right)
			rightW := 36
			if rightW > m.width/3 {
				rightW = m.width / 3
			}
			leftW := m.width - rightW - 6 // account for borders and padding
			if leftW < 20 {
				leftW = 20
			}
			leftStyle := contentStyle
			leftStyle = leftStyle.Width(leftW)
			// compute inner sizes for the right pane and pass them to the renderer
			innerRightW := rightW - 2
			if innerRightW < 10 {
				innerRightW = 10
			}
			innerBodyH := bodyH - 2
			if innerBodyH < 1 {
				innerBodyH = 1
			}
			// Determine border color and thickness to match main page focus styling
			var detailRightBorder string
			var detailRightBorderStyle lipgloss.Border
			var detailLeftBorder string
			var detailLeftBorderStyle lipgloss.Border
			if m.themeHighContrast {
				if m.focusRight {
					detailRightBorder = "#ffffff"
					detailRightBorderStyle = lipgloss.ThickBorder()
					detailLeftBorder = "#444444"
					detailLeftBorderStyle = lipgloss.NormalBorder()
				} else {
					detailRightBorder = "#444444"
					detailRightBorderStyle = lipgloss.NormalBorder()
					detailLeftBorder = "#ffffff"
					detailLeftBorderStyle = lipgloss.ThickBorder()
				}
			} else {
				if m.focusRight {
					detailRightBorder = "#c084fc"
					detailRightBorderStyle = lipgloss.ThickBorder()
					detailLeftBorder = "#334155"
					detailLeftBorderStyle = lipgloss.NormalBorder()
				} else {
					detailRightBorder = "#334155"
					detailRightBorderStyle = lipgloss.NormalBorder()
					detailLeftBorder = "#7dd3fc"
					detailLeftBorderStyle = lipgloss.ThickBorder()
				}
			}
			// apply left pane border styling explicitly so focus state is clear
			leftStyle = leftStyle.BorderStyle(detailLeftBorderStyle).BorderForeground(lipgloss.Color(detailLeftBorder))
			left := leftStyle.Render(m.detail)
			rightStyle := lipgloss.NewStyle().BorderStyle(detailRightBorderStyle).BorderForeground(lipgloss.Color(detailRightBorder)).Padding(1).Width(rightW).Height(bodyH)
			right := rightStyle.Render(m.renderVersions(innerRightW, innerBodyH))
			if m.width < 80 {
				body = lipgloss.JoinVertical(lipgloss.Left, left, right)
			} else {
				body = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
			}
		} else {
			body = contentStyle.Render(m.detail)
		}

		status := fmt.Sprintf("Viewing: %s", m.detailName)
		if m.runInProgress {
			status += " • RUNNING"
		}
		bottom := lipgloss.NewStyle().
			Background(lipgloss.Color(bottomBg)).
			Foreground(lipgloss.Color(bottomFg)).
			Padding(0, 1).
			Width(m.width).
			Render(" " + status + " ")

		var footer string
		if m.editingMeta {
			footer = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#94a3b8")).Render("(Tab) next • (Ctrl+A) add command • (Ctrl+D) del command • (Ctrl+S) save • (Esc) cancel")
		} else {
			base := "(e) Edit • (d) Delete • (s) Export • (r) Run • (T) Toggle Theme • (b) Back • (q) Quit"
			if len(m.versions) > 0 {
				base = base + " • (R) Rollback"
			}
			footer = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#94a3b8")).Render(base)
		}
		return lipgloss.JoinVertical(lipgloss.Left, body, footer, bottom)
	}

	headH := 1
	footerH := 1
	bodyH := m.height - headH - footerH - 2
	if bodyH < 3 {
		bodyH = 3
	}

	// colors adjust for high-contrast theme
	var sideBorder, rightBorder, bottomBg, bottomFg string
	var sideBorderStyle, rightBorderStyle lipgloss.Border
	sideBorderStyle = lipgloss.NormalBorder()
	rightBorderStyle = lipgloss.NormalBorder()

	if m.themeHighContrast {
		bottomBg, bottomFg = "#000000", "#ffffff"
		if m.focusRight {
			sideBorder = "#444444"
			rightBorder = "#ffffff"
			rightBorderStyle = lipgloss.ThickBorder()
		} else {
			sideBorder = "#ffffff"
			sideBorderStyle = lipgloss.ThickBorder()
			rightBorder = "#444444"
		}
	} else {
		bottomBg, bottomFg = "#0b1226", "#cbd5e1"
		if m.focusRight {
			// Right focused
			sideBorder = "#334155"  // dimmed slate
			rightBorder = "#c084fc" // active purple
			rightBorderStyle = lipgloss.ThickBorder()
		} else {
			// Left focused
			sideBorder = "#7dd3fc" // active sky
			sideBorderStyle = lipgloss.ThickBorder()
			rightBorder = "#334155" // dimmed slate
		}
	}

	titleBox := m.renderTitleBox(fmt.Sprintf(" krnr — Command sets (%d) ", len(m.list.Items())))

	sidebarStyle := lipgloss.NewStyle().BorderStyle(sideBorderStyle).BorderForeground(lipgloss.Color(sideBorder)).Padding(0).Width(m.list.Width()).Height(bodyH)
	sidebar := sidebarStyle.Render(m.list.View())

	// compute right pane width to align with outer layout (same logic used in WindowSizeMsg)
	rightW := m.width - m.list.Width() - 4
	if rightW < 12 {
		rightW = 12
	}
	rightStyle := lipgloss.NewStyle().BorderStyle(rightBorderStyle).BorderForeground(lipgloss.Color(rightBorder)).Padding(1).Width(rightW).Height(bodyH)
	right := rightStyle.Render(m.vp.View())

	var body string
	if m.width < 80 {
		body = lipgloss.JoinVertical(lipgloss.Left, sidebar, right)
	} else {
		body = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, right)
	}

	status := "Items: " + fmt.Sprintf("%d", len(m.list.Items()))
	if m.focusRight {
		status += " • FOCUS: PREVIEW/LOGS"
	} else {
		status += " • FOCUS: COMMAND LIST"
	}
	if m.list.FilterState() == list.Filtering {
		status += " • FILTER MODE"
	}
	if m.runInProgress {
		status += " • RUNNING"
	}
	bottom := lipgloss.NewStyle().Background(lipgloss.Color(bottomBg)).Foreground(lipgloss.Color(bottomFg)).Padding(0, 1).Width(m.width).Render(" " + status + " ")

	var footerText string
	if m.list.FilterState() == list.Filtering {
		// In filter mode keep the footer minimal and only show how to quit filter
		footerText = "(esc) quit filter"
	} else {
		footerText = "(←) / (→) / (Tab) switch focus • (↑) / (↓) scroll focused pane"
		footerText += " • (Enter) details • (r) run • (T) theme • (q) quit • (?) help"
	}
	footer := lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#94a3b8")).Render(footerText)

	return lipgloss.JoinVertical(lipgloss.Left, titleBox, body, footer, bottom)
}

// csItem adapts adapters.CommandSetSummary for the list component
type csItem struct{ cs adapters.CommandSetSummary }

func (c csItem) Title() string       { return c.cs.Name }
func (c csItem) Description() string { return c.cs.Description }
func (c csItem) FilterValue() string { return c.cs.Name + " " + c.cs.Description }

// verItem adapts adapters.Version for use with the bubbles list
type verItem struct{ v adapters.Version }

func (v verItem) Title() string {
	return fmt.Sprintf("v%d • %s", v.v.Version, v.v.Operation)
}
func (v verItem) Description() string { return v.v.CreatedAt }
func (v verItem) FilterValue() string {
	return fmt.Sprintf("v%d %s %s", v.v.Version, v.v.Operation, v.v.CreatedAt)
}

// renderTitleBox produces a consistent title bar (with border) matching the
// main page. Use this to keep title styling identical across views.
func (m *TuiModel) renderTitleBox(text string) string {
	var titleFg, titleBg, titleBorder string
	if m.themeHighContrast {
		titleFg, titleBg = "#000000", "#ffff00"
		titleBorder = "#ffff00"
	} else {
		titleFg, titleBg = "#ffffff", "#0f766e"
		titleBorder = "#0ea5a4"
	}
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(titleFg)).Background(lipgloss.Color(titleBg)).Padding(0, 1)
	title := titleStyle.Render(text)
	titleInner := lipgloss.Place(m.width-2, 1, lipgloss.Center, lipgloss.Center, title)
	return lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color(titleBorder)).Width(m.width).Render(titleInner)
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

// renderVersions renders the versions panel on the details view's right side.
// It used the versionsList view so the list adapts automatically to the size
// (same behavior as the main list).
func (m *TuiModel) renderVersions(width, height int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0ea5a4")).Render("Versions") + "\n\n")

	// list view will already be sized in WindowSizeMsg, so just render it
	b.WriteString(m.versionsList.View())

	// preview area
	previewH := 8
	if previewH > height/2 {
		previewH = height / 2
	}
	if m.versionsSelected >= 0 && m.versionsSelected < len(m.versions) {
		b.WriteString("\nPreview:\n")
		b.WriteString(formatVersionPreview(m.detailName, m.versions[m.versionsSelected], width-2, previewH))
	}
	return b.String()
}

// formatVersionPreview renders a short preview for a version (commands and dry-run)
func formatVersionPreview(name string, v adapters.Version, _ int, _ int) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("krnr — %s v%d • %s\n", name, v.Version, v.Operation))
	b.WriteString(strings.Repeat("-", 30) + "\n")
	for _, c := range v.Commands {
		b.WriteString("$ " + c + "\n")
	}
	return b.String()
}

// trimLastRune removes the last rune from a string if present
func trimLastRune(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if len(r) == 0 {
		return ""
	}
	return string(r[:len(r)-1])
}

// filterEmptyLines returns a copy with empty or whitespace-only lines removed
func filterEmptyLines(in []string) []string {
	out := []string{}
	for _, l := range in {
		if strings.TrimSpace(l) != "" {
			out = append(out, strings.TrimSpace(l))
		}
	}
	return out
}
