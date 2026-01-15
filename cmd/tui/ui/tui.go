package ui

import (
	"context"
	"fmt"
	"strings"

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

// NewModel, NewProgram and Init were moved to `core.go` as part of
// Phase 5 modularization (core orchestration and interfaces).

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

func (m *TuiModel) IsDetailShown() (bool, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.showDetail, m.detailName
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
		// If we're editing metadata, delegate key handling to editor.go
		if m.editingMeta {
			return m.handleEditorKey(msg)
		}

		// If we're in our custom filter mode, use the centralized dispatch helper
		// which will handle editing/filter behaviors. This keeps `Update()` small
		// and makes it easier to unit test the input rules in isolation.
		if m.filterMode {
			var dm tea.Model
			var fall bool
			dm, cmd, fall = dispatchKey(m, msg)
			if newM, ok := dm.(*TuiModel); ok {
				m = newM
			}
			if !fall {
				return m, cmd
			}
		}
		// If the list is in filtering state, delegate to helper
		if dm, cmd, handled := handleListFiltering(m, msg); handled {
			if newM, ok := dm.(*TuiModel); ok {
				m = newM
			}
			return m, cmd
		}
		// Global keybindings handled BEFORE passing to the list so they are
		// not swallowed when filtering is enabled. Delegate to handler.
		dm, cmd, handled := handleGlobalKeys(m, s, msg)
		if handled {
			if newM, ok := dm.(*TuiModel); ok {
				m = newM
			}
			return m, cmd
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

		// Delegate focused navigation (right/left pane scrolling and versions list)
		if dm, cmd, handled := handleFocusedNavigation(m, s, msg); handled {
			if newM, ok := dm.(*TuiModel); ok {
				m = newM
			}
			return m, cmd
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

// Rendering helpers moved to cmd/tui/ui/render.go
// (wrapText, renderTwoCol, renderTableInline, renderTableBlockHeader,
// simulateOutput, formatCSFullScreen and formatCSDetails)
// This keeps the main TUI logic focused on state and input handling.

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
		// If there's a transient notification show it prominently in the footer
		m.mu.RLock()
		notif := m.notification
		m.mu.RUnlock()
		if notif != "" {
			footer = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f43f5e")).Render(notif)
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
		footerText += " - (Enter) details - (r) run - (T) theme - (C) New Entry - (q) quit - (?) help"
	}
	footer := lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#94a3b8")).Render(footerText)
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
