package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
)

// dispatchKey routes KeyMsg to the appropriate handler based on current UI state.
// This is the entry point for Phase 4: we will iteratively move the key handling
// logic out of `Update()` into discrete, testable helpers.
func dispatchKey(m *TuiModel, msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	// Editor modal has its own handler
	if m.editingMeta {
		dm, cmd := m.handleEditorKey(msg)
		return dm, cmd, false
	}

	// Basic handling for our custom filter mode (kept lightweight here).
	if m.filterMode {
		dm, cmd, handled := handleDispatchFilterMode(m, msg)
		return dm, cmd, handled
	}

	// Fallback no-op — Update() will continue to handle global bindings and
	// list navigation when dispatchKey returns (this keeps migrations incremental).
	return m, nil, false
}

// applyListFilterKey applies a single KeyMsg to the list-filtering behavior
// without checking whether filtering is active. This makes the behavior easy
// to test in isolation.
func applyListFilterKey(m *TuiModel, msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	// capture previous selection so we can detect changes after navigation
	prevSel := ""
	if si := m.list.SelectedItem(); si != nil {
		if it, ok := si.(csItem); ok {
			prevSel = it.cs.Name
		}
	}

	m.list, _ = m.list.Update(msg)
	// update the live filter text and apply filtering only when the filter
	// text actually changed (typing/backspace/esc). Navigation keys should
	// not reapply the items and reset the selection.
	if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace || msg.Type == tea.KeyBackspace || msg.Type == tea.KeyDelete || msg.Type == tea.KeyEsc {
		updateListFilterText(m, msg)
		applyFilterItems(m)
	}
	// ensure preview updates if selection changed
	updatePreviewIfSelectionChanged(m, prevSel)

	return m, nil, true
}

func updateListFilterText(m *TuiModel, msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyRunes:
		for _, ru := range msg.Runes {
			m.listFilter += string(ru)
		}
	case tea.KeySpace:
		m.listFilter += " "
	case tea.KeyBackspace, tea.KeyDelete:
		if len(m.listFilter) > 0 {
			r := []rune(m.listFilter)
			m.listFilter = string(r[:len(r)-1])
		}
	case tea.KeyEsc:
		// ESC cancels filter — clear our filter cache
		m.listFilter = ""
	}
}

func updatePreviewIfSelectionChanged(m *TuiModel, prevSel string) {
	if si := m.list.SelectedItem(); si != nil {
		if it, ok := si.(csItem); ok {
			if it.cs.Name != prevSel {
				m.lastSelectedName = it.cs.Name
				if cs, err := m.uiModel.GetCommandSet(context.Background(), it.cs.Name); err == nil {
					m.detailName = cs.Name
					m.detail = formatCSDetails(cs, m.vp.Width)
					m.vp.SetContent(m.detail)
					m.logPreviewUpdate(cs.Name)
				}
			}
		}
	}
}

// handleListFiltering centralizes logic for the bubbles list internal filtering
// mode. Returns (model, cmd, handled).
func handleListFiltering(m *TuiModel, msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.list.FilterState() != list.Filtering {
		return m, nil, false
	}
	return applyListFilterKey(m, msg)
}

// applyFilterItems updates the list items and title based on the current
// `m.listFilter` string (live filter behavior used by the TUI).
func applyFilterItems(m *TuiModel) {
	q := strings.ToLower(strings.TrimSpace(m.listFilter))
	var items []list.Item
	if q == "" {
		items = listAllItems(m)
	} else {
		if strings.HasPrefix(m.listFilter, "#") {
			q = strings.TrimSpace(strings.TrimPrefix(q, "#"))
			items = listFilteredByTags(m, q)
		} else {
			items = listFilteredByHaystack(m, q)
		}
	}
	m.list.SetItems(items)
	m.list.Title = "Filter: " + m.listFilter
	if len(items) > 0 {
		m.list.Select(0)
		// Update the preview/content for the newly selected item so it stays
		// in sync while filtering. Do this regardless of whether the detail
		// pane is currently visible so behavior is consistent with non-filter
		// navigation (selection should always update the right-hand preview).
		if it, ok := items[0].(csItem); ok {
			if cs, err := m.uiModel.GetCommandSet(context.Background(), it.cs.Name); err == nil {
				m.detailName = cs.Name
				m.detail = formatCSDetails(cs, m.vp.Width)
				m.vp.SetContent(m.detail)
				m.logPreviewUpdate(cs.Name)
			}
		}
	}
}

func handleDispatchFilterMode(m *TuiModel, msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.Type {
	case tea.KeyRunes, tea.KeySpace:
		return handleFilterModeRunes(m, msg)
	case tea.KeyBackspace, tea.KeyDelete:
		return handleFilterModeBackspace(m, msg)
	case tea.KeyEsc:
		// cancel filter mode and restore items/title
		m.filterMode = false
		m.listFilter = ""
		m.list.SetItems(listAllItems(m))
		m.list.Title = "krnr — command sets"
		return m, nil, false
	case tea.KeyEnter:
		// Exit filter mode and let the global Enter handler take over
		m.filterMode = false
		m.listFilter = ""
		m.list.Title = "krnr — command sets"
		return m, nil, true
	default:
		return handleFilterModeNavigation(m, msg)
	}
}

func handleFilterModeRunes(m *TuiModel, msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if msg.Type == tea.KeySpace {
		m.listFilter += " "
	} else {
		for _, ru := range msg.Runes {
			m.listFilter += string(ru)
		}
	}
	applyFilterItems(m)
	return m, nil, false
}

func handleFilterModeBackspace(m *TuiModel, _ tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if len(m.listFilter) > 0 {
		r := []rune(m.listFilter)
		m.listFilter = string(r[:len(r)-1])
	}
	applyFilterItems(m)
	return m, nil, false
}

func handleFilterModeNavigation(m *TuiModel, msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	// Let list handle navigation keys
	var listCmd tea.Cmd
	// capture previous selection to detect changes
	prevName := ""
	if si := m.list.SelectedItem(); si != nil {
		if it, ok := si.(csItem); ok {
			prevName = it.cs.Name
		}
	}
	m.list, listCmd = m.list.Update(msg)
	// If the selection changed while filtering, update preview so users
	// see details while navigating in filter mode. Update preview
	// regardless of whether the detail pane is visible (consistent with
	// non-filter navigation behavior).
	if si := m.list.SelectedItem(); si != nil {
		if it, ok := si.(csItem); ok {
			if it.cs.Name != prevName {
				m.lastSelectedName = it.cs.Name
				if cs, err := m.uiModel.GetCommandSet(context.Background(), it.cs.Name); err == nil {
					m.detailName = cs.Name
					m.detail = formatCSDetails(cs, m.vp.Width)
					m.vp.SetContent(m.detail)
					m.logPreviewUpdate(cs.Name)
				}
			}
		}
	}
	return m, listCmd, false
}

func listAllItems(m *TuiModel) []list.Item {
	items := make([]list.Item, 0, len(m.uiModel.ListCached()))
	for _, s := range m.uiModel.ListCached() {
		items = append(items, csItem{cs: s})
	}
	return items
}

func listFilteredByTags(m *TuiModel, q string) []list.Item {
	items := make([]list.Item, 0)
	for _, s := range m.uiModel.ListCached() {
		matched := false
		for _, t := range s.Tags {
			if strings.Contains(strings.ToLower(t), q) {
				matched = true
				break
			}
		}
		if matched {
			items = append(items, csItem{cs: s})
		}
	}
	return items
}

func listFilteredByHaystack(m *TuiModel, q string) []list.Item {
	items := make([]list.Item, 0)
	for _, s := range m.uiModel.ListCached() {
		hay := strings.ToLower(s.Name + " " + s.Description)
		if len(s.Tags) > 0 {
			hay = hay + " " + strings.ToLower(strings.Join(s.Tags, " "))
		}
		if strings.Contains(hay, q) {
			items = append(items, csItem{cs: s})
		}
	}
	return items
}

// handleGlobalKeys handles global, top-level key bindings such as quit, help,
// Enter to view details, edit/delete/export/run, and pane focus toggles.
// It returns (possibly modified model, cmd, handled).
func handleGlobalKeys(m *TuiModel, s string, msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if dm, cmd, handled := handleGlobalQuitAndHelp(m, s); handled {
		return dm, cmd, true
	}
	if dm, cmd, handled := handleGlobalEnterBackEditCreateMenu(m, s, msg); handled {
		return dm, cmd, true
	}
	if dm, cmd, handled := handleGlobalConfirmAndActions(m, s); handled {
		return dm, cmd, true
	}
	if dm, cmd, handled := handleGlobalRunToggleThemeAndPane(m, s, msg); handled {
		return dm, cmd, true
	}

	// non-printable bindings
	if msg.Type == tea.KeyCtrlT {
		m.themeHighContrast = !m.themeHighContrast
		return m, nil, true
	}

	return m, nil, false
}

func handleGlobalQuitAndHelp(m *TuiModel, s string) (tea.Model, tea.Cmd, bool) {
	switch s {
	case "q", "esc":
		return m, tea.Quit, true
	case "?":
		return handleHelp(m)
	}
	return m, nil, false
}

func handleGlobalEnterBackEditCreateMenu(m *TuiModel, s string, msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch s {
	case "enter":
		return handleEnter(m, msg)
	case "b":
		return handleBack(m)
	case "e":
		return handleEdit(m)
	case "c", "C":
		return handleCreate(m)
	case "m":
		return handleMenu(m)
	}
	return m, nil, false
}

func handleGlobalConfirmAndActions(m *TuiModel, s string) (tea.Model, tea.Cmd, bool) {
	switch s {
	case "d":
		return handleDelete(m)
	case "s":
		return handleExport(m)
	case "y", "Y":
		return handleConfirmYes(m)
	case "n", "N":
		return handleConfirmNo(m)
	}
	return m, nil, false
}

func handleGlobalRunToggleThemeAndPane(m *TuiModel, s string, _ tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch s {
	case "r":
		return handleRun(m)
	case "T", "t":
		m.themeHighContrast = !m.themeHighContrast
		return m, nil, true
	case "left", "right", "tab":
		return handlePaneNavigation(m, s)
	}
	return m, nil, false
}

// helper: show help
func handleHelp(m *TuiModel) (tea.Model, tea.Cmd, bool) {
	m.setShowDetail(true)
	m.detail = "Help:\n\n? show help\nq or Esc to quit\nEnter to view details\n(C) Create new entry\n/ to filter\n← → or Tab to switch pane focus\n↑ ↓ to scroll focused pane"
	return m, nil, true
}

// helper: handle Enter key when pressed
func handleEnter(m *TuiModel, msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.showDetail && m.focusRight && len(m.versions) > 0 {
		m.versionsList, _ = m.versionsList.Update(msg)
		m.versionsSelected = m.versionsList.Index()
		if m.versionsSelected >= 0 && m.versionsSelected < len(m.versions) {
			m.setVersionsPreviewIndex(m.versionsSelected)
		}
		return m, nil, true
	}
	if i, ok := m.list.SelectedItem().(csItem); ok {
		m.setShowDetail(true)
		m.setDetailName(i.cs.Name)
		if cs, err := m.uiModel.GetCommandSet(context.Background(), i.cs.Name); err == nil {
			m.detail = formatCSFullScreen(cs, m.width, m.height)
			m.vp.SetContent(m.detail)
		} else {
			m.detail = formatCSDetails(i.cs, m.width/2)
			m.vp.SetContent(m.detail)
		}
		prepareDetailViewport(m)
		prepareVersionsList(m, i.cs.Name)
	}
	return m, nil, true
}

func prepareDetailViewport(m *TuiModel) {
	// compute viewport size conservatively
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
}

func prepareVersionsList(m *TuiModel, name string) {
	if vers, err := m.uiModel.ListVersions(context.Background(), name); err == nil {
		m.versions = vers
		m.versionsOffset = 0
		items := make([]list.Item, 0, len(vers))
		for _, v := range vers {
			items = append(items, verItem{v: v})
		}
		m.versionsList.SetItems(items)
		if len(vers) > 0 {
			m.versionsSelected = 0
			m.versionsList.Select(0)
		}
		setVersionsListSize(m)
	} else {
		m.versions = nil
		m.versionsSelected = 0
		m.versionsOffset = 0
		m.versionsList.SetItems([]list.Item{})
	}
}

func setVersionsListSize(m *TuiModel) {
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
	indicatorH := 1
	available := innerBodyH - indicatorH - 2
	if available < 1 {
		available = 1
	}
	m.versionsList.SetSize(innerRightW, available)
}

// helper: hide full-screen detail and reset focus
func handleBack(m *TuiModel) (tea.Model, tea.Cmd, bool) {
	m.setShowDetail(false)
	m.focusRight = false
	if si := m.list.SelectedItem(); si != nil {
		if it, ok := si.(csItem); ok {
			if cs, err := m.uiModel.GetCommandSet(context.Background(), it.cs.Name); err == nil {
				m.vp.SetContent(formatCSDetails(cs, m.vp.Width))
			} else {
				m.vp.SetContent(formatCSDetails(it.cs, m.vp.Width))
			}
		}
	}
	return m, nil, true
}

// helper: enter edit mode for selected item
func handleEdit(m *TuiModel) (tea.Model, tea.Cmd, bool) {
	if !m.showDetail {
		return m, nil, true
	}
	if i, ok := m.list.SelectedItem().(csItem); ok {
		name := i.cs.Name
		cs, err := m.uiModel.GetCommandSet(context.Background(), name)
		if err != nil {
			m.logs = append(m.logs, "edit: get: "+err.Error())
			return m, nil, true
		}
		m.editingMeta = true
		m.editor.create = false
		m.editor.field = 0
		m.editor.name = cs.Name
		m.editor.desc = cs.Description
		m.editor.author = cs.AuthorName
		m.editor.authorEmail = cs.AuthorEmail
		m.editor.tags = strings.Join(cs.Tags, ",")
		m.editor.commands = append([]string{}, cs.Commands...)
		if len(m.editor.commands) == 0 {
			m.editor.commands = []string{""}
		}
		m.editor.cmdIndex = 0
	}
	return m, nil, true
}

// helper: create new command set
func handleCreate(m *TuiModel) (tea.Model, tea.Cmd, bool) {
	m.editingMeta = true
	m.editor.create = true
	m.editor.field = 0
	m.editor.name = ""
	m.editor.desc = ""
	m.editor.author = ""
	m.editor.authorEmail = ""
	m.editor.tags = ""
	m.editor.commands = []string{""}
	m.editor.cmdIndex = 0
	m.setShowDetail(true)
	m.setDetailName("")
	return m, nil, true
}

// helper: open menu modal
func handleMenu(m *TuiModel) (tea.Model, tea.Cmd, bool) {
	m.showMenu = true
	m.menuIndex = 0
	if len(m.menuItems) == 0 {
		m.menuItems = []string{"Export database", "Import database", "Import set", "Install", "Uninstall", "Status", "Close"}
	}
	return m, nil, true
}

// helper: prepare delete confirmation
func handleDelete(m *TuiModel) (tea.Model, tea.Cmd, bool) {
	if !m.showDetail {
		return m, nil, true
	}
	var name string
	if i, ok := m.list.SelectedItem().(csItem); ok {
		name = i.cs.Name
	} else if m.detailName != "" {
		name = m.detailName
	}
	if name == "" {
		return m, nil, true
	}
	m.pendingDelete = true
	m.pendingDeleteName = name
	m.detail = fmt.Sprintf("Delete '%s' permanently? [y/N]\n\nPress (y) to confirm, (n) or (b) to cancel", name)
	m.vp.SetContent(m.detail)
	return m, nil, true
}

// helper: prepare export confirmation
func handleExport(m *TuiModel) (tea.Model, tea.Cmd, bool) {
	if !m.showDetail {
		return m, nil, true
	}
	var ename string
	if i, ok := m.list.SelectedItem().(csItem); ok {
		ename = i.cs.Name
	} else if m.detailName != "" {
		ename = m.detailName
	}
	if ename == "" {
		return m, nil, true
	}
	dflt := filepath.Join(os.TempDir(), ename+".db")
	dest := uniqueDestPath(dflt)
	m.pendingExport = true
	m.pendingExportName = ename
	m.pendingExportDest = dest
	m.detail = fmt.Sprintf("Export '%s' to:\n\n%s\n\nPress (y) to confirm, (n) to cancel", ename, dest)
	m.vp.SetContent(m.detail)
	return m, nil, true
}

// helper: confirm (y/Y)
func handleConfirmYes(m *TuiModel) (tea.Model, tea.Cmd, bool) {
	if m.pendingRollback {
		return confirmRollbackYes(m)
	}
	if m.pendingDelete {
		return confirmDeleteYes(m)
	}
	if m.pendingExport {
		return confirmExportYes(m)
	}
	m.logs = append(m.logs, "no pending action to confirm")
	return m, nil, true
}

func confirmRollbackYes(m *TuiModel) (tea.Model, tea.Cmd, bool) {
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
	return m, nil, true
}

func confirmDeleteYes(m *TuiModel) (tea.Model, tea.Cmd, bool) {
	name := m.pendingDeleteName
	if err := m.uiModel.Delete(context.Background(), name); err != nil {
		m.logs = append(m.logs, "delete error: "+err.Error())
		m.pendingDelete = false
		return m, nil, true
	}
	_ = m.uiModel.RefreshList(context.Background())
	items := make([]list.Item, 0, len(m.uiModel.ListCached()))
	for _, s := range m.uiModel.ListCached() {
		items = append(items, csItem{cs: s})
	}
	m.list.SetItems(items)
	m.logs = append(m.logs, fmt.Sprintf("deleted '%s'", name))
	m.pendingDelete = false
	m.pendingDeleteName = ""
	m.setShowDetail(false)
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
	m.detail = fmt.Sprintf("Deleted '%s'", name)
	m.vp.SetContent(m.detail)
	return m, nil, true
}

func confirmExportYes(m *TuiModel) (tea.Model, tea.Cmd, bool) {
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
	return m, nil, true
}

// helper: cancel confirm (n/N)
func handleConfirmNo(m *TuiModel) (tea.Model, tea.Cmd, bool) {
	if m.pendingRollback {
		return confirmRollbackNo(m)
	}
	if m.pendingDelete {
		return confirmDeleteNo(m)
	}
	if m.pendingExport {
		return confirmExportNo(m)
	}
	return m, nil, true
}

func confirmRollbackNo(m *TuiModel) (tea.Model, tea.Cmd, bool) {
	m.pendingRollback = false
	m.pendingRollbackName = ""
	m.pendingRollbackVersion = 0
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
	return m, nil, true
}

func confirmDeleteNo(m *TuiModel) (tea.Model, tea.Cmd, bool) {
	m.pendingDelete = false
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
	return m, nil, true
}

func confirmExportNo(m *TuiModel) (tea.Model, tea.Cmd, bool) {
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
	return m, nil, true
}

// helper: run a command set
func handleRun(m *TuiModel) (tea.Model, tea.Cmd, bool) {
	if m.runInProgress {
		return m, nil, true
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
		return m, nil, true
	}
	m.logs = nil
	m.runInProgress = true
	m.focusRight = true
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelRun = cancel
	h, err := m.uiModel.Run(ctx, name, nil)
	if err != nil {
		m.logs = append(m.logs, "run error: "+err.Error())
		m.runInProgress = false
		return m, nil, true
	}
	ch := make(chan adapters.RunEvent)
	m.runCh = ch
	go func() {
		for ev := range h.Events() {
			ch <- ev
		}
		close(ch)
	}()
	m.runInProgress = true
	return m, readLoop(m.runCh), true
}

// helper: left/right/tab pane navigation
func handlePaneNavigation(m *TuiModel, s string) (tea.Model, tea.Cmd, bool) {
	switch s {
	case "left":
		return handlePaneLeft(m)
	case "right":
		return handlePaneRight(m)
	case "tab":
		return handlePaneTab(m)
	}
	return m, nil, false
}

func handlePaneLeft(m *TuiModel) (tea.Model, tea.Cmd, bool) {
	m.focusRight = false
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
	return m, nil, true
}

func handlePaneRight(m *TuiModel) (tea.Model, tea.Cmd, bool) {
	m.focusRight = true
	if m.showDetail && len(m.versions) > 0 {
		idx := m.versionsList.Index()
		if idx < 0 {
			idx = 0
		}
		m.setVersionsPreviewIndex(idx)
	}
	return m, nil, true
}

func handlePaneTab(m *TuiModel) (tea.Model, tea.Cmd, bool) {
	m.focusRight = !m.focusRight
	if !m.focusRight {
		return handlePaneTabLeft(m)
	}
	return handlePaneTabRight(m)
}

func handlePaneTabLeft(m *TuiModel) (tea.Model, tea.Cmd, bool) {
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
	return m, nil, true
}

func handlePaneTabRight(m *TuiModel) (tea.Model, tea.Cmd, bool) {
	if m.showDetail && len(m.versions) > 0 {
		idx := m.versionsList.Index()
		if idx < 0 {
			idx = 0
		}
		m.setVersionsPreviewIndex(idx)
	}
	return m, nil, true
}

// handleFocusedNavigation handles scrolling and navigation when focus is on
// either pane and when detail view is shown. Returns (model, cmd, handled).
func handleFocusedNavigation(m *TuiModel, s string, msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.focusRight && m.showDetail && len(m.versions) > 0 {
		return handleVersionsNav(m, s, msg)
	}
	if m.focusRight {
		return handleRightPaneScroll(m, s)
	}
	if m.showDetail && !m.focusRight {
		return handleDetailPaneNavigation(m, s)
	}
	return m, nil, false
}

func handleVersionsNav(m *TuiModel, s string, msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch s {
	case "R":
		if m.versionsSelected >= 0 && m.versionsSelected < len(m.versions) {
			v := m.versions[m.versionsSelected]
			m.pendingRollback = true
			m.pendingRollbackVersion = v.Version
			m.pendingRollbackName = m.detailName
			m.detail = fmt.Sprintf("Rollback '%s' to version %d? [y/N]\n\nPress (y) to confirm, (n) to cancel", m.pendingRollbackName, m.pendingRollbackVersion)
			m.vp.SetContent(m.detail)
		}
		return m, nil, true
	default:
		var listCmd tea.Cmd
		m.versionsList, listCmd = m.versionsList.Update(msg)
		idx := m.versionsList.Index()
		if idx < 0 {
			idx = 0
		}
		m.setVersionsPreviewIndex(idx)
		return m, listCmd, true
	}
}

func handleRightPaneScroll(m *TuiModel, s string) (tea.Model, tea.Cmd, bool) {
	switch s {
	case "up", "k":
		m.vp.ScrollUp(1)
		return m, nil, true
	case "down", "j":
		m.vp.ScrollDown(1)
		return m, nil, true
	case "pgup":
		m.vp.HalfPageUp()
		return m, nil, true
	case "pgdown":
		m.vp.HalfPageDown()
		return m, nil, true
	case "home":
		m.vp.GotoTop()
		return m, nil, true
	case "end":
		m.vp.GotoBottom()
		return m, nil, true
	}
	return m, nil, false
}

func handleDetailPaneNavigation(m *TuiModel, s string) (tea.Model, tea.Cmd, bool) {
	switch s {
	case "up", "k":
		m.vp.ScrollUp(1)
		return m, nil, true
	case "down", "j":
		m.vp.ScrollDown(1)
		return m, nil, true
	case "pgup":
		m.vp.HalfPageUp()
		return m, nil, true
	case "pgdown":
		m.vp.HalfPageDown()
		return m, nil, true
	case "home":
		m.vp.GotoTop()
		return m, nil, true
	case "end":
		m.vp.GotoBottom()
		return m, nil, true
	case "enter":
		// do nothing — Enter shouldn't change detail while detail is open
		return m, nil, true
	}
	return m, nil, false
}
