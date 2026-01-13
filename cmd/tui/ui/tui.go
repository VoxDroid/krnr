package ui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
	interactive "github.com/VoxDroid/krnr/internal/utils"
)

// TuiModel is the Bubble Tea model used by cmd/tui.
type TuiModel struct {
	uiModel *modelpkg.UIModel
	list    list.Model
	vp      viewport.Model

	width  int
	height int

	showDetail    bool
	detail        string
	detailName    string
	runInProgress bool
	logs          []string
	cancelRun     func()
	runCh         chan adapters.RunEvent
	// accessibility / theme
	themeHighContrast bool
	// track last selected name so we can detect changes and update preview
	lastSelectedName string
	// focus: false = left pane (list), true = right pane (viewport)
	focusRight bool
}

// Messages
type runEventMsg adapters.RunEvent
type runDoneMsg struct{}

func NewModel(ui *modelpkg.UIModel) *TuiModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "krnr — command sets"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	vp := viewport.New(0, 0)

	return &TuiModel{uiModel: ui, list: l, vp: vp}
}

// NewProgram constructs the tea.Program for the TUI.
func NewProgram(ui *modelpkg.UIModel) *tea.Program {
	m := NewModel(ui)
	p := tea.NewProgram(m, tea.WithAltScreen())
	return p
}

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

func (m *TuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		s := msg.String()
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
			// Edit commands for the selected command set when in detail view
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
				// create temp file with commands
				tmpf, err := os.CreateTemp("", "krnr-edit-*.txt")
				if err != nil {
					m.logs = append(m.logs, "edit: tmpfile: "+err.Error())
					return m, nil
				}
				defer func() { _ = os.Remove(tmpf.Name()) }()
				for _, c := range cs.Commands {
					_, _ = tmpf.WriteString(c + "\n")
				}
				_ = tmpf.Close()
				// open editor
				if err := interactive.OpenEditor(tmpf.Name()); err != nil {
					m.logs = append(m.logs, "edit: open editor: "+err.Error())
					return m, nil
				}
				// read back and parse
				b, err := os.ReadFile(tmpf.Name())
				if err != nil {
					m.logs = append(m.logs, "edit: read: "+err.Error())
					return m, nil
				}
				lines := []string{}
				sc := strings.Split(string(b), "\n")
				for _, ln := range sc {
					line := strings.TrimSpace(ln)
					if line == "" || strings.HasPrefix(line, "#") {
						continue
					}
					lines = append(lines, line)
				}
				if err := m.uiModel.ReplaceCommands(context.Background(), name, lines); err != nil {
					m.logs = append(m.logs, "edit: replace: "+err.Error())
					return m, nil
				}
				// refresh preview
				if updated, err := m.uiModel.GetCommandSet(context.Background(), name); err == nil {
					m.detail = formatCSFullScreen(updated, m.width, m.height)
					m.vp.SetContent(m.detail)
				}
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
			// Right pane focused - scroll viewport
			switch s {
			case "up", "k":
				m.vp.LineUp(1)
				return m, nil
			case "down", "j":
				m.vp.LineDown(1)
				return m, nil
			case "pgup":
				m.vp.HalfViewUp()
				return m, nil
			case "pgdown":
				m.vp.HalfViewDown()
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

func formatCSFullScreen(cs adapters.CommandSetSummary, width int, height int) string {
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

// add a small dry-run output test helper for the headless tests to assert
// the detail view renders simulated outputs for echo commands.
func dryRunOutputForTests(c adapters.CommandSetSummary, width int) string {
	return formatCSFullScreen(c, width, 20)
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
		body := contentStyle.Render(m.detail)

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

		footer := lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("#94a3b8")).
			Render("(e) Edit • (r) Run • (T) Toggle Theme • (b) Back • (q) Quit")
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
	var right string
	right = rightStyle.Render(m.vp.View())

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

	footerText := "← / → / Tab switch focus • ↑ / ↓ scroll focused pane • Enter details • r run • T theme • q quit • ? help"
	footer := lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#94a3b8")).Render(footerText)

	return lipgloss.JoinVertical(lipgloss.Left, titleBox, body, footer, bottom)
}

// csItem adapts adapters.CommandSetSummary for the list component
type csItem struct{ cs adapters.CommandSetSummary }

func (c csItem) Title() string       { return c.cs.Name }
func (c csItem) Description() string { return c.cs.Description }
func (c csItem) FilterValue() string { return c.cs.Name + " " + c.cs.Description }

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
