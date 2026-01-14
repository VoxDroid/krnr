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

	"github.com/VoxDroid/krnr/internal/executor"
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
	// live filter text while the list is in filtering state
	listFilter string
	// when true we are in our custom filter mode and typed characters should
	// immediately filter the left-hand list (we manage the prompt display
	// ourselves so the built-in list filtering UI can remain disabled)
	filterMode bool

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
	versionsPreviewContent string
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
	// We'll implement live filtering ourselves so disable the built-in filter UI
	l.SetFilteringEnabled(false)

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

	// Reconcile versions preview at end of Update to catch external selection
	// changes (mouse events, direct list.Select calls, or commands that alter
	// the list selection outside the normal key handlers). This uses the
	// deterministic setVersionsPreviewIndex helper which is idempotent.
	defer func() {
		if m.showDetail && m.focusRight && len(m.versions) > 0 {
			idx := m.versionsList.Index()
			if idx < 0 {
				idx = 0
			}
			_ = m.setVersionsPreviewIndex(idx)
		}
	}()

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
			case "up":
				// arrow/up should move the command selection when in commands field;
				// typed letters like 'k' must be accepted as rune input, so avoid
				// interpreting the 'k' rune as navigation here.
				if m.editor.field == 3 && m.editor.cmdIndex > 0 {
					m.editor.cmdIndex--
				}
				return m, nil
			case "down":
				// same rationale as above for 'j' vs down arrow
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
				raw := filterEmptyLines(m.editor.commands)
			clean := make([]string, 0, len(raw))
			for i, c := range raw {
				cSan := executor.Sanitize(c)
				if cSan != c {
					m.logs = append(m.logs, "sanitized command: \""+c+"\" -> \""+cSan+"\"")
					for j := range m.editor.commands {
						if strings.TrimSpace(m.editor.commands[j]) == strings.TrimSpace(c) {
							m.editor.commands[j] = cSan
							break
						}
					}
				}
				if err := executor.ValidateCommand(cSan); err != nil {
					m.logs = append(m.logs, "replace commands: "+err.Error())
					return m, nil
				}
				clean = append(clean, cSan)
				_ = i
			}
			if err := m.uiModel.ReplaceCommands(context.Background(), newCS.Name, clean); err != nil {
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
		// If we're in our custom filter mode, handle character input here so we can
		// perform live filtering and show a prompt. We do this instead of using the
		// built-in list filtering UI because we want immediate item updates.
		if m.filterMode {
			allowGlobalEnter := false
			switch msg.Type {
			case tea.KeyRunes:
				for _, ru := range msg.Runes {
					m.listFilter += string(ru)
				}
			case tea.KeyBackspace, tea.KeyDelete:
				if len(m.listFilter) > 0 {
					r := []rune(m.listFilter)
					m.listFilter = string(r[:len(r)-1])
				}
			case tea.KeyEsc:
				// cancel filter mode and restore items/title
				m.filterMode = false
				m.listFilter = ""
				items := make([]list.Item, 0, len(m.uiModel.ListCached()))
				for _, s := range m.uiModel.ListCached() {
					items = append(items, csItem{cs: s})
				}
				m.list.SetItems(items)
				m.list.Title = "krnr — command sets"
				return m, nil
			case tea.KeyEnter:
				// Exit filter mode and let the global Enter handler take over
				allowGlobalEnter = true
				m.filterMode = false
				m.listFilter = ""
				m.list.Title = "krnr — command sets"
			default:
				// let navigation keys be handled by the list so up/down still work
				m.list, cmd = m.list.Update(msg)
				return m, cmd
			}
			if !allowGlobalEnter {
				// apply live filtering against cached items
				q := strings.ToLower(strings.TrimSpace(m.listFilter))
				items := make([]list.Item, 0)
				if q == "" {
					for _, s := range m.uiModel.ListCached() {
						items = append(items, csItem{cs: s})
					}
				} else {
					for _, s := range m.uiModel.ListCached() {
						if strings.Contains(strings.ToLower(s.Name+" "+s.Description), q) {
							items = append(items, csItem{cs: s})
						}
					}
				}
				m.list.SetItems(items)
				// show the filter prompt in the title so list.View() contains it
				m.list.Title = "Filter: " + m.listFilter
				if len(items) > 0 {
					m.list.Select(0)
				}
				return m, nil
			}
			// else: allow handler to fall through to global key handling (e.g., Enter)
		}
		// If the list is in filtering state, let the list consume all keys
		// (including keys that would otherwise be global controls like q/esc).
		if m.list.FilterState() == list.Filtering {
			m.list, cmd = m.list.Update(msg)
			// Maintain our own live filter text so we can update the list items
			switch msg.Type {
			case tea.KeyRunes:
				for _, ru := range msg.Runes {
					m.listFilter += string(ru)
				}
			case tea.KeyBackspace, tea.KeyDelete:
				if len(m.listFilter) > 0 {
					r := []rune(m.listFilter)
					m.listFilter = string(r[:len(r)-1])
				}
			case tea.KeyEsc:
				// ESC cancels filter — clear our filter cache
				m.listFilter = ""
			}
			// apply live filtering against the cached list
			q := strings.ToLower(strings.TrimSpace(m.listFilter))
			if q == "" {
				// restore full items
				items := make([]list.Item, 0, len(m.uiModel.ListCached()))
				for _, s := range m.uiModel.ListCached() {
					items = append(items, csItem{cs: s})
				}
				m.list.SetItems(items)
			} else {
				items := make([]list.Item, 0)
				for _, s := range m.uiModel.ListCached() {
					if strings.Contains(strings.ToLower(s.Name+" "+s.Description), q) {
						items = append(items, csItem{cs: s})
					}
				}
				m.list.SetItems(items)
			}
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
				// when a version is highlighted, preview its metadata/commands on the
				// left-side pane so users can inspect historic versions without
				// permanently changing the detail view.
				if m.versionsSelected >= 0 && m.versionsSelected < len(m.versions) {
					m.setVersionsPreviewIndex(m.versionsSelected)
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
				// Ensure viewport has a reasonable size even if we haven't received a WindowSizeMsg
				headH := 1
				footerH := 1
				bodyH := m.height - headH - footerH - 2
				if bodyH < 3 {
					bodyH = 3
				}
				vpw := m.width - 8
				vph := bodyH - 4
				if vpw < 10 {
					vpw = 10
				}
				if vph < 3 {
					vph = 3
				}
				m.vp = viewport.New(vpw, vph)
				m.vp.SetContent(m.detail)
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
					// reserve one line for the versions pane shortcuts at the bottom
					indicatorH := 1
					available := innerBodyH - indicatorH - 2
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
			// if we were showing a historic version in the preview, restore the
			// current/latest command set details for the selected item
			if m.showDetail && m.detailName != "" {
				if cs, err := m.uiModel.GetCommandSet(context.Background(), m.detailName); err == nil {
					m.detail = formatCSFullScreen(cs, m.width, m.height)
					m.vp.SetContent(m.detail)
				} else if si := m.list.SelectedItem(); si != nil {
					if it, ok := si.(csItem); ok {
						m.vp.SetContent(formatCSDetails(it.cs, m.vp.Width))
					}
				}
			}
			return m, nil
		case "right":
			// Switch focus to right pane (versions)
			m.focusRight = true
			// If we have versions loaded, preview the currently-selected version
			if m.showDetail && len(m.versions) > 0 {
				idx := m.versionsList.Index()
				if idx < 0 {
					idx = 0
				}
				m.setVersionsPreviewIndex(idx)
			}
			return m, nil
		case "tab":
			// Toggle focus between panes; when toggling back to the left pane we
			// should restore the current/latest version preview if in detail view
			m.focusRight = !m.focusRight
			if !m.focusRight && m.showDetail && m.detailName != "" {
				if cs, err := m.uiModel.GetCommandSet(context.Background(), m.detailName); err == nil {
					m.detail = formatCSFullScreen(cs, m.width, m.height)
					m.vp.SetContent(m.detail)
				} else if si := m.list.SelectedItem(); si != nil {
					if it, ok := si.(csItem); ok {
						m.vp.SetContent(formatCSDetails(it.cs, m.vp.Width))
					}
				}
			} else if m.focusRight && m.showDetail && len(m.versions) > 0 {
				// toggled into versions pane — preview currently-selected version
				idx := m.versionsList.Index()
				if idx < 0 {
					idx = 0
				}
				m.setVersionsPreviewIndex(idx)
			}
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
					idx := m.versionsList.Index()
					if idx < 0 {
						idx = 0
					}
					m.setVersionsPreviewIndex(idx)
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
		// Left pane focused or other keys - when details are shown and focus is
		// on the left pane, make the detail content scrollable instead of
		// changing list selection. Arrow keys and paging should scroll the
		// details viewport. Enter is ignored (user must press 'b' to go back).
		if m.showDetail && !m.focusRight {
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
			case "enter":
				// do nothing — Enter shouldn't change detail while detail is open
				return m, nil
			}
		}

		m.list, cmd = m.list.Update(msg)

		if s == "/" {
			// enter our custom filter mode which shows a prompt and filters items
			m.filterMode = true
			m.listFilter = ""
			m.list.Title = "Filter: "
			items := make([]list.Item, 0, len(m.uiModel.ListCached()))
			for _, s := range m.uiModel.ListCached() {
				items = append(items, csItem{cs: s})
			}
			m.list.SetItems(items)
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

		// compute side/right widths with safe bounds to avoid overflow on narrow terminals
		sideW := int(float64(m.width) * 0.35)
		if sideW > 36 {
			sideW = 36
		}
		if sideW < 10 {
			sideW = 10
		}
		// ensure sideW leaves room for the right pane; adjust if necessary
		minRightW := 12
		if m.width-sideW-4 < minRightW {
			sideW = m.width - minRightW - 4
			if sideW < 6 {
				sideW = 6
			}
		}
		innerSideW := sideW - 2
		if innerSideW < 4 {
			innerSideW = 4
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
		// Configure the detail/preview viewport size based on whether we are
		// showing details (and whether the versions panel is present). This
		// ensures the viewport is anchored within the content box (top/bottom)
		// and avoids recreating it in View(), which previously caused truncation
		// and overlap when layout calculations drifted between renders.
		if m.showDetail && len(m.versions) > 0 {
			// when versions panel exists, left pane width is the remaining width
			rightW := 36
			if rightW > m.width/3 {
				rightW = m.width / 3
			}
			leftW := m.width - rightW - 6
			if leftW < 20 {
				leftW = 20
			}
			vpw := leftW - 4
			vph := innerBodyH - 2
			if vpw < 10 {
				vpw = 10
			}
			if vph < 3 {
				vph = 3
			}
			m.vp = viewport.New(vpw, vph)
		} else if m.showDetail {
			// full-screen detail (no versions)
			vpw := m.width - 8
			vph := bodyH - 4
			if vpw < 10 {
				vpw = 10
			}
			if vph < 3 {
				vph = 3
			}
			m.vp = viewport.New(vpw, vph)
		} else {
			// list preview area (default)
			m.vp = viewport.New(innerRightW, innerBodyH)
		}

		// configure versionsList sizing to match the inner area when present
		if len(m.versions) > 0 {
			// reserve one line for the versions pane shortcuts at the bottom
			indicatorH := 1
			available := innerBodyH - indicatorH - 2
			if available < 1 {
				available = 1
			}
			m.versionsList.SetSize(innerRightW, available)
		}

		// update content for selected
		if m.showDetail {
			// when in details mode, reformat the full-screen detail to match the
			// current terminal width/height so it remains responsive. Only update
			// the main detail when the left pane has focus; otherwise keep the
			// versions preview visible while focus is on the right pane.
			if !m.focusRight {
				if m.detailName != "" {
					if cs, err := m.uiModel.GetCommandSet(context.Background(), m.detailName); err == nil {
						m.detail = formatCSFullScreen(cs, m.width, m.height)
						m.vp.SetContent(m.detail)
					}
				}
			} else {
				// when focus is on the versions pane, ensure the preview matches
				// the currently-selected versions list index (if any)
				if len(m.versions) > 0 {
					idx := m.versionsList.Index()
					if idx < 0 {
						idx = 0
					}
					m.setVersionsPreviewIndex(idx)
				}
			}
		} else {
			if si := m.list.SelectedItem(); si != nil {
				if cs, ok := si.(csItem); ok {
					// construct a full metadata view
					full := formatCSDetails(cs.cs, m.vp.Width)
					m.vp.SetContent(full)
				}
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

		headH := 0 // removed top title box; inner detail contains its own title
		footerH := 1
		bottomH := 1
		// Account for footer and bottom bar when computing the available body height
		bodyH := m.height - headH - footerH - bottomH - 2
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
			rightW := 30
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
			// ensure viewport matches left content area so the detail becomes scrollable
			vpw := leftW - 4
			vph := innerBodyH - 2
			if vpw < 10 {
				vpw = 10
			}
			if vph < 3 {
				vph = 3
			}
			m.detail = strings.TrimRight(m.detail, "\n")
			// ensure viewport sized for current layout so detail becomes scrollable
			if m.vp.Width != vpw || m.vp.Height != vph {
				oldOff := m.vp.YOffset
				m.vp = viewport.New(vpw, vph)
				m.vp.YOffset = oldOff
			}
			// Only set the main detail content when left pane is focused. If the
			// right pane (versions) is focused, we should display the preview
			// content which is set by setVersionsPreviewIndex.
			if !m.focusRight {
				m.vp.SetContent(m.detail)
			}
			left := leftStyle.Render(m.vp.View())
			rightStyle := lipgloss.NewStyle().BorderStyle(detailRightBorderStyle).BorderForeground(lipgloss.Color(detailRightBorder)).Padding(1).Width(rightW).Height(bodyH)
			right := rightStyle.Render(m.renderVersions(innerRightW, innerBodyH))
			if m.width < 80 {
				body = lipgloss.JoinVertical(lipgloss.Left, left, right)
			} else {
				body = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
			}
		} else {
			// no versions pane; render detail inside a viewport so it can scroll
			// to match the behavior/responsiveness of the main page.
			m.detail = strings.TrimRight(m.detail, "\n")
			// ensure viewport sized for full-screen detail
			vpw := m.width - 8
			vph := bodyH - 4
			if vpw < 10 {
				vpw = 10
			}
			if vph < 3 {
				vph = 3
			}
			if m.vp.Width != vpw || m.vp.Height != vph {
				oldOff := m.vp.YOffset
				m.vp = viewport.New(vpw, vph)
				m.vp.YOffset = oldOff
			}
			m.vp.SetContent(m.detail)
			body = contentStyle.Render(m.vp.View())
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
			footer = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#94a3b8")).Render("(Tab) next - (Ctrl+A) add command - (Ctrl+D) del command - (Ctrl+S) save - (Esc) cancel")
		} else {
			base := "(e) Edit - (d) Delete - (s) Export - (r) Run - (T) Toggle Theme - (b) Back - (q) Quit"
			if len(m.versions) > 0 {
				base = base + " - (R) Rollback"
			}
			// When showing details, add a scroll hint and indicator so users can
			// discover navigation and their current position inside the detail.
			if m.showDetail {
				base = base + " - (↑/↓ scroll detail)"
				if ind := m.detailScrollIndicator(); ind != "" {
					base = base + " • " + ind
				}
			}
			footer = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#94a3b8")).Render(base)
		}
		return lipgloss.JoinVertical(lipgloss.Left, body, footer, bottom)
	}

	headH := 1 // reserve a 1-line top spacer so borders don't touch the terminal edge
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
	if m.filterMode || m.list.FilterState() == list.Filtering {
		status += " • FILTER MODE"
	}
	if m.runInProgress {
		status += " - RUNNING"
	}
	bottom := lipgloss.NewStyle().Background(lipgloss.Color(bottomBg)).Foreground(lipgloss.Color(bottomFg)).Padding(0, 1).Width(m.width).Render(" " + status + " ")

	var footerText string
	if m.filterMode || m.list.FilterState() == list.Filtering {
		// In filter mode keep the footer minimal and only show how to quit filter
		footerText = "(esc) quit filter"
	} else {
		// Use simple ASCII-friendly footer to ensure compatibility across
		// environments and avoid encoding issues with exotic characters.
		footerText = "(<-) / (->) / (Tab) switch focus - (Up)/(Down) scroll focused pane"
		footerText += " - (Enter) details - (r) run - (T) theme - (q) quit - (?) help"
	}
	footer := lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#94a3b8")).Render(footerText)

	// top spacer to ensure the main page has a minimum top height and borders don't touch the terminal edge
	topSpacer := lipgloss.NewStyle().Height(headH).Width(m.width).Render("")

	return lipgloss.JoinVertical(lipgloss.Left, topSpacer, body, footer, bottom)
}

// csItem adapts adapters.CommandSetSummary for the list component
type csItem struct{ cs adapters.CommandSetSummary }

func (c csItem) Title() string {
	return c.cs.Name
}

func (c csItem) Description() string {
	return c.cs.Description
}

func (c csItem) FilterValue() string {
	return c.cs.Name + " " + c.cs.Description
}

// verItem adapts adapters.Version for use with the bubbles list
type verItem struct{ v adapters.Version }

func (v verItem) Title() string {
	return fmt.Sprintf("v%d - %s", v.v.Version, v.v.Operation)
}
func (v verItem) Description() string { return v.v.CreatedAt }
func (v verItem) FilterValue() string {
	return fmt.Sprintf("v%d %s %s", v.v.Version, v.v.Operation, v.v.CreatedAt)
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

	return b.String()
}

// formatVersionPreview renders a short preview for a version (commands and dry-run)
func formatVersionPreview(name string, v adapters.Version, _ int, _ int) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("krnr — %s v%d - %s\n", name, v.Version, v.Operation))
	b.WriteString(strings.Repeat("-", 30) + "\n")
	for _, c := range v.Commands {
		b.WriteString("$ " + c + "\n")
	}
	return b.String()
}

// setVersionsPreviewIndex sets the versions preview to the given index and
// updates internal state; returns a tea.Cmd if needed (nil for now).
func (m *TuiModel) setVersionsPreviewIndex(idx int) tea.Cmd {
	if idx < 0 || idx >= len(m.versions) {
		return nil
	}
	// If no change to the selection and we already have a preview set, no work needed
	if idx == m.versionsSelected && m.versionsPreviewContent != "" {
		return nil
	}
	oldIdx := m.versionsSelected
	oldContent := m.versionsPreviewContent
	content := formatVersionDetails(m.detailName, m.versions[idx], m.vp.Width)
	changed := content != oldContent || idx != oldIdx

	// Update state
	m.versionsSelected = idx
	m.versionsPreviewContent = content
	// reset scroll and set new content if it actually changed
	if changed {
		m.vp.YOffset = 0
		m.vp.SetContent(content)
	}
	return nil
}

// formatVersionDetails renders a full metadata view for a historic version.
// It is similar to the full-screen command set rendering but for a specific
// version so users can inspect author, description, created date and commands.
func formatVersionDetails(name string, v adapters.Version, width int) string {
	// reuse visual styles from full-screen formatting
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0ea5a4")).Background(lipgloss.Color("#0b1226"))
	h := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0ea5a4"))
	k := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#94a3b8"))
	var b strings.Builder
	contentW := width - 6
	if contentW < 10 {
		contentW = 10
	}
	// Title header
	titleText := fmt.Sprintf("v%d %s — %s", v.Version, v.Operation, name)
	b.WriteString(titleStyle.Render(titleText) + "\n")
	sepLen := contentW
	if sepLen > len(titleText)+4 {
		sepLen = len(titleText) + 4
	}
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#0ea5a4")).Render(strings.Repeat("─", sepLen)) + "\n\n")

	// compute label widths
	labels := []string{"Version:", "Created:", "Author:", "Description:", "Commands:"}
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

	b.WriteString(h.Render("Version:") + " " + fmt.Sprintf("%d", v.Version) + "\n")
	if v.CreatedAt != "" {
		b.WriteString(h.Render("Created:") + " " + v.CreatedAt + "\n")
	}
	if v.AuthorName != "" {
		b.WriteString(h.Render("Author:") + " " + v.AuthorName + "\n")
	}

	if v.Description != "" {
		lines := wrapText(v.Description, valueW)
		b.WriteString("\n")
		b.WriteString(h.Render("Description:") + "\n")
		b.WriteString(renderTableBlockHeader("", strings.Join(lines, "\n"), labelW))
	}

	if len(v.Commands) > 0 {
		b.WriteString("\n")
		b.WriteString(h.Render("Commands:") + "\n")
		maxPrefix := 0
		for i := range v.Commands {
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
		for i, c := range v.Commands {
			p := fmt.Sprintf("%d) ", i+1)
			cb.WriteString(renderTwoCol(p, c, maxPrefix, innerTextW))
		}
		b.WriteString(renderTableBlockHeader("", strings.TrimSuffix(cb.String(), "\n"), labelW))
	}

	// metadata fields
	meta := []string{}
	if v.AuthorName != "" {
		meta = append(meta, "Author: "+v.AuthorName)
	}
	if v.AuthorEmail != "" {
		meta = append(meta, "Email: "+v.AuthorEmail)
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

// detailScrollIndicator builds a small "current/total" indicator for the
// detail pane based on the viewport's height and vertical offset. It helps
// users discover where they are inside long details.
func (m *TuiModel) detailScrollIndicator() string {
	if m.detail == "" || m.vp.Height <= 0 {
		return ""
	}
	lines := strings.Split(m.detail, "\n")
	total := len(lines)
	visible := m.vp.Height
	// attempt to read viewport offset; the viewport exposes YOffset so use it
	off := 0
	// guard against zero/nil viewport
	off = m.vp.YOffset
	if off < 0 {
		off = 0
	}
	if visible >= total {
		return fmt.Sprintf("%d/%d", 1, total)
	}
	if off > total-visible {
		off = total - visible
	}
	return fmt.Sprintf("%d/%d", off+1, total)
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
