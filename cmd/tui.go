package cmd

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Start interactive terminal UI (Bubble Tea prototype)",
	RunE: func(cmd *cobra.Command, _ []string) error {
		// Init DB
		dbConn, err := db.InitDB()
		if err != nil {
			return err
		}
		defer func() { _ = dbConn.Close() }()

		r := registry.NewRepository(dbConn)
		sets, err := r.ListCommandSets()
		if err != nil {
			return err
		}

		items := make([]list.Item, 0, len(sets))
		for _, s := range sets {
			items = append(items, csItem{cs: s})
		}

		l := list.New(items, list.NewDefaultDelegate(), 0, 0)
		l.Title = "krnr — command sets"
		l.SetShowStatusBar(false)
		l.SetFilteringEnabled(true)

	// viewport for right pane
	vp := viewport.New(0, 0)

	m := model{list: l, vp: vp}
	p := tea.NewProgram(m, tea.WithAltScreen())

		return p.Start()
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}


// csItem adapts registry.CommandSet to list.Item
type csItem struct{
	cs registry.CommandSet
}

func (c csItem) Title() string {
	return c.cs.Name
}
func (c csItem) Description() string {
	d := ""
	if c.cs.Description.Valid {
		d = c.cs.Description.String
	}
	return d
}
func (c csItem) FilterValue() string { return c.cs.Name + " " + c.Description() }

// model implements tea.Model
type model struct{
	list       list.Model
	vp         viewport.Model
	width      int
	height     int
	showDetail bool
	detail     string
	filterMode bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// First, hand the message to the list so filtering input will be processed
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		s := msg.String()

		// Enter filter-mode when user types '/'
		if s == "/" {
			m.filterMode = true
			return m, cmd
		}

		// If filter mode is active, don't intercept keys—let the list handle them
		if m.filterMode {
			if s == "esc" || s == "enter" {
				m.filterMode = false
			}
			return m, cmd
		}

		// Normal keyhandlers (only when not filtering)
		switch s {
		case "q", "esc":
			return m, tea.Quit
		case "?":
			m.showDetail = true
			m.detail = "Help:\n\n? show help\nq or Esc to quit\nEnter to view details\n/ to filter"
			return m, nil
		case "enter":
			if i, ok := m.list.SelectedItem().(csItem); ok {
				// build detail
				desc := ""
				if i.cs.Description.Valid {
					desc = i.cs.Description.String
				}
				last := ""
				if i.cs.LastRun.Valid {
					last = i.cs.LastRun.String
				}
				m.showDetail = true
				m.detail = fmt.Sprintf("Name: %s\n\nDescription:\n%s\n\nLast run:\n%s\n", i.cs.Name, desc, last)
				m.vp.SetContent(m.detail)
			}
			return m, nil
		case "b":
			m.showDetail = false
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// layout: header (1) + footer (1) + body padding
		headH := 1
		footerH := 1
		bodyH := m.height - headH - footerH - 2
		if bodyH < 3 { bodyH = 3 }

		// compute sidebar and right widths; account for borders so boxes don't overlap
		sideW := int(float64(m.width) * 0.35)
		if sideW > 36 { sideW = 36 }
		if sideW < 20 { sideW = 20 }
		innerSideW := sideW - 2
		if innerSideW < 10 { innerSideW = 10 }

		rightW := m.width - sideW - 4 // leave room for spacing and borders
		if rightW < 12 { rightW = 12 }
		innerRightW := rightW - 2
		if innerRightW < 10 { innerRightW = 10 }

		// inner heights for widgets (subtract border padding)
		innerBodyH := bodyH - 2
		if innerBodyH < 1 { innerBodyH = 1 }

		m.list.SetSize(innerSideW, innerBodyH)
		m.vp = viewport.New(innerRightW, innerBodyH)

		// update content to match selection
		if si := m.list.SelectedItem(); si != nil {
			if cs, ok := si.(csItem); ok {
				d := cs.Description()
				if d == "" { d = "(no description)" }
				m.vp.SetContent(fmt.Sprintf("Name: %s\n\nDescription:\n%s\n", cs.Title(), d))
			}
		}	}

	// keep viewport responsive: if selection changed, update content
	if si := m.list.SelectedItem(); si != nil {
		if cs, ok := si.(csItem); ok {
			d := cs.Description()
			if d == "" { d = "(no description)" }
			m.vp.SetContent(fmt.Sprintf("Name: %s\n\nDescription:\n%s\n", cs.Title(), d))
		}
	}

	return m, cmd
}

func (m model) View() string {
	// When showing a detail overlay, render it centered
	if m.showDetail {
		return lipgloss.NewStyle().Padding(1, 2).Render(m.detail + "\n\n(b) Back | (q) Quit")
	}

	// compute body height for consistent sizing (same logic as Update)
	headH := 1
	footerH := 1
	bodyH := m.height - headH - footerH - 2
	if bodyH < 3 { bodyH = 3 }

	// Header (topbar) — Neovim-like with full-width border
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff")).Background(lipgloss.Color("#0f766e")).Padding(0, 1)
	title := titleStyle.Render(fmt.Sprintf(" krnr — Command sets (%d) ", len(m.list.Items())))
	// center the title in the available width inside the border
	titleInner := lipgloss.Place(m.width-2, 1, lipgloss.Center, lipgloss.Center, title)
	titleBox := lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#0ea5a4")).Width(m.width).Render(titleInner)

	// Sidebar (list) using list.View for scroll behavior — fixed width box
	sidebarStyle := lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#7dd3fc")).Padding(0).Width(m.list.Width()).Height(bodyH)
	sidebar := sidebarStyle.Render(m.list.View())

	// Right pane (viewport)
	rightStyle := lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#c084fc")).Padding(1)
	right := rightStyle.Render(m.vp.View())

	// Combine side-by-side — if terminal too narrow, stack vertically
	var body string
	if m.width < 80 {
		body = lipgloss.JoinVertical(lipgloss.Left, sidebar, right)
	} else {
		body = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, right)
	}

	// Bottom bar/status — full width
	status := "Items: " + fmt.Sprintf("%d", len(m.list.Items()))
	if m.filterMode { status += " • FILTER MODE" }
	bottom := lipgloss.NewStyle().Background(lipgloss.Color("#0b1226")).Foreground(lipgloss.Color("#cbd5e1")).Padding(0, 1).Width(m.width).Render(" " + status + " ")

	// Footer/help
	footer := lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#94a3b8")).Render("/ filter • Enter view details • b back • q quit • ? help")

	// Full layout
	return lipgloss.JoinVertical(lipgloss.Left, titleBox, body, footer, bottom)
}
