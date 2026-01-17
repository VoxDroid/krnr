package ui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"sync"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
)

// TuiModel is the Bubble Tea model used by cmd/tui.
type TuiModel struct {
	uiModel Model
	list    list.Model
	vp      viewport.Model

	width  int
	height int
	mu     sync.RWMutex

	// menu modal state
	showMenu       bool
	menuItems      []string
	menuIndex      int
	menuInput      string // used when an item requires a path input (e.g., import db)
	menuPendingSrc string // holds the source path while we prompt for additional options
	menuInputMode  bool   // whether the menu is in input mode
	menuAction     string // action to perform on input confirmation

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

	// transient notification message shown in the footer for validation/errors
	notification string

	// editing metadata modal state
	editingMeta bool
	editor      struct {
		field            int // 0=name,1=desc,2=authorName,3=authorEmail,4=tags,5=commands
		name             string
		desc             string
		author           string
		authorEmail      string
		tags             string
		commands         []string
		cmdIndex         int
		create           bool   // true when creating a new entry (vs editing existing)
		saving           bool   // true while a save is in progress to prevent re-entry
		lastFailedName   string // last name that failed validation/save to short-circuit repeated spam
		lastFailedReason string // the notification/reason for why last save failed for that name
		lastEditAt       time.Time
		lastSavedAt      time.Time // timestamp of last successful save
		saveRetries      int
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

// saveNowMsg schedules an editor save after a short delay to allow PTY input
// to be processed before committing the edit. This helps make PTY end-to-end
// flows deterministic when typing then immediately saving.
type saveNowMsg struct{}

type clearNotificationMsg struct{}

// thread-safe helpers for detail view state used by integration tests
func (m *TuiModel) setShowDetail(v bool) {
	m.mu.Lock()
	m.showDetail = v
	m.mu.Unlock()
}

func (m *TuiModel) setDetailName(name string) {
	m.mu.Lock()
	m.detailName = name
	m.mu.Unlock()
}

// setNotification sets the transient footer notification
func (m *TuiModel) setNotification(msg string) {
	m.mu.Lock()
	m.notification = msg
	m.mu.Unlock()
}

// clearNotification clears any footer notification
func (m *TuiModel) clearNotification() {
	m.mu.Lock()
	m.notification = ""
	m.mu.Unlock()
}

// logsContains reports whether any log entry contains the provided substring.
func (m *TuiModel) logsContains(s string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, l := range m.logs {
		if strings.Contains(l, s) {
			return true
		}
	}
	return false
}

// IsDetailShown reports detail visibility and the currently shown name (thread-safe).
func (m *TuiModel) IsDetailShown() (bool, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.showDetail, m.detailName
}

// logPreviewUpdate appends a small trace to the tui logs when the
// KRNR_TUI_DEBUG_PREVIEW=1 environment variable is set. This helps
// interactive debugging without spamming logs in normal runs.
func (m *TuiModel) logPreviewUpdate(name string) {
	if os.Getenv("KRNR_TUI_DEBUG_PREVIEW") != "1" {
		return
	}
	m.mu.Lock()
	m.logs = append(m.logs, "preview_update: "+name)
	m.mu.Unlock()
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

	dm, cmd := m.handleUpdateMsg(msg)
	if newM, ok := dm.(*TuiModel); ok {
		m = newM
	}
	return m, cmd
}

func (m *TuiModel) handleUpdateMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case saveNowMsg:
		return m.handleSaveNowWrapped()
	case clearNotificationMsg:
		m.clearNotification()
		return m, nil
	case tea.KeyMsg:
		return m.handleKeyMsgWrapped(msg)
	case runEventMsg:
		return m.handleRunEventWrapped(adapters.RunEvent(msg))
	case runDoneMsg:
		m.runInProgress = false
		m.runCh = nil
		return m, nil
	case tea.WindowSizeMsg:
		return m.handleWindowSizeWrapped(msg)
	}
	return m, nil
}

func (m *TuiModel) handleSaveNowWrapped() (tea.Model, tea.Cmd) {
	dm, cmd := m.handleSaveNow()
	if newM, ok := dm.(*TuiModel); ok {
		m = newM
	}
	return m, cmd
}

func (m *TuiModel) handleKeyMsgWrapped(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if dm, cmd, handled := m.processKeyMsg(msg); handled {
		if newM, ok := dm.(*TuiModel); ok {
			m = newM
		}
		return m, cmd
	}
	return m, nil
}

func (m *TuiModel) handleRunEventWrapped(ev adapters.RunEvent) (tea.Model, tea.Cmd) {
	dm, cmd := m.handleRunEvent(ev)
	if newM, ok := dm.(*TuiModel); ok {
		m = newM
	}
	return m, cmd
}

func (m *TuiModel) handleWindowSizeWrapped(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	dm, cmd := m.handleWindowSize(msg)
	if newM, ok := dm.(*TuiModel); ok {
		m = newM
	}
	return m, cmd
}

func (m *TuiModel) handleRunEvent(ev adapters.RunEvent) (tea.Model, tea.Cmd) {
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
}

// processKeyMsg centralizes KeyMsg handling to keep Update() concise.
func (m *TuiModel) processKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	// handlers in prioritized order
	handlers := []func(tea.KeyMsg) (tea.Model, tea.Cmd, bool){
		m.handleEditorOrMenuKeys,
		m.handleFilterModeKeys,
		func(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
			if dm, cmd, handled := handleListFiltering(m, msg); handled {
				return dm, cmd, true
			}
			return m, nil, false
		},
		m.handleGlobalKeysWrapper,
		m.handleNonPrintableKey,
		m.handleListFilteringState,
		m.handleFocusedNavWrapper,
		m.handleListUpdateAndSelection,
	}
	for _, h := range handlers {
		if dm, cmd, handled := h(msg); handled {
			if newM, ok := dm.(*TuiModel); ok {
				m = newM
			}
			return m, cmd, true
		}
	}
	return m, nil, false
}

func (m *TuiModel) handleEditorOrMenuKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.editingMeta {
		dm, cmd := m.handleEditorKey(msg)
		return dm, cmd, true
	}
	if m.showMenu {
		dm, cmd := m.handleMenuKey(msg)
		return dm, cmd, true
	}
	return m, nil, false
}

func (m *TuiModel) handleFilterModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.filterMode {
		return m, nil, false
	}
	var dm tea.Model
	var cmd tea.Cmd
	var fall bool
	dm, cmd, fall = dispatchKey(m, msg)
	if newM, ok := dm.(*TuiModel); ok {
		m = newM
	}
	if !fall {
		if si := m.list.SelectedItem(); si != nil {
			if it, ok := si.(csItem); ok {
				if it.cs.Name != m.lastSelectedName {
					m.lastSelectedName = it.cs.Name
					if cs, err := m.uiModel.GetCommandSet(context.Background(), it.cs.Name); err == nil {
						m.vp.SetContent(formatCSDetails(cs, m.vp.Width))
						m.logPreviewUpdate(cs.Name)
					}
				}
			}
		}
		return m, cmd, true
	}
	return m, nil, false
}

func (m *TuiModel) handleGlobalKeysWrapper(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	s := msg.String()
	dm, cmd, handled := handleGlobalKeys(m, s, msg)
	if handled {
		return dm, cmd, true
	}
	return m, nil, false
}

func (m *TuiModel) handleNonPrintableKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if msg.Type == tea.KeyCtrlT {
		m.themeHighContrast = !m.themeHighContrast
		return m, nil, true
	}
	return m, nil, false
}

func (m *TuiModel) handleListFilteringState(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.list.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd, true
	}
	return m, nil, false
}

func (m *TuiModel) handleFocusedNavWrapper(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	s := msg.String()
	if dm, cmd, handled := handleFocusedNavigation(m, s, msg); handled {
		return dm, cmd, true
	}
	return m, nil, false
}

func (m *TuiModel) handleListUpdateAndSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	if msg.String() == "/" {
		m.filterMode = true
		m.listFilter = ""
		m.list.Title = "Filter: "
		items := make([]list.Item, 0, len(m.uiModel.ListCached()))
		for _, s := range m.uiModel.ListCached() {
			items = append(items, csItem{cs: s})
		}
		m.list.SetItems(items)
		return m, cmd, true
	}
	if si := m.list.SelectedItem(); si != nil {
		if it, ok := si.(csItem); ok {
			if it.cs.Name != m.lastSelectedName {
				if cs, err := m.uiModel.GetCommandSet(context.Background(), it.cs.Name); err == nil {
					m.vp.SetContent(formatCSDetails(cs, m.vp.Width))
				}
				m.lastSelectedName = it.cs.Name
			}
		}
	}
	return m, cmd, true
}

func (m *TuiModel) handleSaveNow() (tea.Model, tea.Cmd) {
	// scheduled save: if edits are still arriving, retry a few times before
	// performing the save. This avoids races where the delayed save fires
	// while the PTY is still delivering multi-byte rune sequences.
	const stabilityWindow = 50 * time.Millisecond
	const maxRetries = 5
	if time.Since(m.editor.lastEditAt) < stabilityWindow && m.editor.saveRetries < maxRetries {
		m.editor.saveRetries++
		return m, tea.Tick(stabilityWindow, func(_ time.Time) tea.Msg { return saveNowMsg{} })
	}
	// reset retry counter: short-circuit if a recent successful save already covered current edits
	m.editor.saveRetries = 0
	if !m.editor.lastSavedAt.IsZero() && m.editor.lastSavedAt.After(m.editor.lastEditAt) {
		// recent successful save supersedes this scheduled save; skip to avoid redundant updates
		m.editor.saving = false
		return m, nil
	}
	if err := m.editorSave(); err != nil {
		if !strings.Contains(err.Error(), "already in use") {
			m.logs = append(m.logs, "replace commands: "+err.Error())
		}
	}
	m.editor.saving = false
	return m, nil
}

func (m *TuiModel) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	innerSideW, innerRightW, innerBodyH := m.computeLayoutDimensions()
	m.list.SetSize(innerSideW, innerBodyH)
	m.updateViewportForLayout(innerRightW, innerBodyH)
	m.updateVersionsListSize(innerRightW, innerBodyH)
	m.updateSelectedContentForResize()
	return m, nil
}

func (m *TuiModel) computeLayoutDimensions() (int, int, int) {
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
	return innerSideW, innerRightW, innerBodyH
}

func (m *TuiModel) updateViewportForLayout(innerRightW, innerBodyH int) {
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
		vph := (innerBodyH + 1) - 4
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
}

func (m *TuiModel) updateVersionsListSize(innerRightW, innerBodyH int) {
	if len(m.versions) > 0 {
		// reserve one line for the versions pane shortcuts at the bottom
		indicatorH := 1
		available := innerBodyH - indicatorH - 2
		if available < 1 {
			available = 1
		}
		m.versionsList.SetSize(innerRightW, available)
	}
}

func (m *TuiModel) updateSelectedContentForResize() {
	if m.showDetail {
		if !m.focusRight {
			if m.detailName != "" {
				if cs, err := m.uiModel.GetCommandSet(context.Background(), m.detailName); err == nil {
					m.detail = formatCSFullScreen(cs, m.width, m.height)
					m.vp.SetContent(m.detail)
				}
			}
		} else {
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
				full := formatCSDetails(cs.cs, m.vp.Width)
				m.vp.SetContent(full)
			}
		}
	}
}

// Rendering helpers moved to cmd/tui/ui/render.go
// (wrapText, renderTwoCol, renderTableInline, renderTableBlockHeader,
// simulateOutput, formatCSFullScreen and formatCSDetails)
// This keeps the main TUI logic focused on state and input handling.

// View renders the current TUI view as a string.
func (m *TuiModel) View() string {
	if m.showMenu {
		return m.renderMenuOverlay()
	}
	if m.showDetail {
		return m.renderDetailView()
	}
	return m.renderMainView()
}

func (m *TuiModel) renderMenuOverlay() string {
	// compute an appropriate height for the menu overlay (leave space for Footer)
	h := m.height - 4
	if h < 3 {
		h = 3
	}
	contentStyle := lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#c084fc")).Padding(1).Width(m.width - 4).Height(h)
	return contentStyle.Render(m.renderMenu())
}

func (m *TuiModel) renderDetailView() string {
	rightBorder, bottomBg, bottomFg := m.detailColors()

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
	// choose appropriate body content
	if m.editingMeta {
		body = contentStyle.Render(m.renderEditor())
	} else if len(m.versions) > 0 {
		body = m.renderDetailWithVersions(bodyH)
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
		m.ensureViewportSize(vpw, vph)
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
	footer = m.detailFooter()
	// If there's a transient notification show it prominently in the footer
	m.mu.RLock()
	notif := m.notification
	m.mu.RUnlock()
	if notif != "" {
		footer = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f43f5e")).Render(notif)
	}
	return lipgloss.JoinVertical(lipgloss.Left, body, footer, bottom)
}

func (m *TuiModel) detailColors() (string, string, string) {
	if m.themeHighContrast {
		return "#ffffff", "#000000", "#ffffff"
	}
	return "#c084fc", "#0b1226", "#cbd5e1"
}

func (m *TuiModel) detailFooter() string {
	if m.editingMeta {
		return lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#94a3b8")).Render("(Tab) next - (Ctrl+A) add command - (Ctrl+D) del command - (Ctrl+S) save - (Esc) cancel")
	}
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
	return lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#94a3b8")).Render(base)
}

func (m *TuiModel) renderDetailWithVersions(bodyH int) string {
	// split the content region into main detail (left) and versions (right)
	rightW := 30
	if rightW > m.width/3 {
		rightW = m.width / 3
	}
	leftW := m.width - rightW - 6 // account for borders and padding
	if leftW < 20 {
		leftW = 20
	}
	leftStyle := lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#000000")).Padding(1)
	leftStyle = leftStyle.Width(leftW).Height(bodyH)
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
	detailRightBorder, detailRightBorderStyle, detailLeftBorder, detailLeftBorderStyle := computeDetailPaneStyles(m)
	// apply left pane border styling explicitly so focus state is clear
	leftStyle = leftStyle.BorderStyle(detailLeftBorderStyle).BorderForeground(lipgloss.Color(detailLeftBorder))

	// ensure viewport sized for current layout so detail becomes scrollable
	vpw := leftW - 4
	vph := innerBodyH - 2
	if vpw < 10 {
		vpw = 10
	}
	if vph < 3 {
		vph = 3
	}
	m.detail = strings.TrimRight(m.detail, "\n")
	m.ensureViewportSize(vpw, vph)
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
		return lipgloss.JoinVertical(lipgloss.Left, left, right)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func computeDetailPaneStyles(m *TuiModel) (string, lipgloss.Border, string, lipgloss.Border) {
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
	return detailRightBorder, detailRightBorderStyle, detailLeftBorder, detailLeftBorderStyle
}

func (m *TuiModel) renderMainView() string {
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

	bottom := lipgloss.NewStyle().Background(lipgloss.Color(bottomBg)).Foreground(lipgloss.Color(bottomFg)).Padding(0, 1).Width(m.width).Render(" " + m.mainStatus() + " ")

	footer := lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#94a3b8")).Render(m.mainFooterText())
	// If there's a transient notification show it prominently in the footer
	m.mu.RLock()
	notif := m.notification
	m.mu.RUnlock()
	if notif != "" {
		footer = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f43f5e")).Render(notif)
	}

	// top spacer to ensure the main page has a minimum top height and borders don't touch the terminal edge
	topSpacer := lipgloss.NewStyle().Height(headH).Width(m.width).Render("")

	return lipgloss.JoinVertical(lipgloss.Left, topSpacer, body, footer, bottom)
}

func (m *TuiModel) mainStatus() string {
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
	return status
}

func (m *TuiModel) mainFooterText() string {
	if m.filterMode || m.list.FilterState() == list.Filtering {
		// In filter mode keep the footer minimal and only show how to quit filter
		return "(esc) quit filter"
	}
	// Use simple ASCII-friendly footer to ensure compatibility across
	// environments and avoid encoding issues with exotic characters.
	footerText := "(<-) / (->) / (Tab) switch focus - (Up)/(Down) scroll focused pane"
	footerText += " - (Enter) details - (r) run - (T) theme - (C) New Entry - (m) Menu - (q) quit - (?) help"
	return footerText
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

// renderEditor moved into cmd/tui/ui/editor.go

// Versions rendering and helpers moved into cmd/tui/ui/versions.go
// (renderVersions, formatVersionPreview, setVersionsPreviewIndex,
// formatVersionDetails)

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
