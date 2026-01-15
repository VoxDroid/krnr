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

// NewModel constructs the Bubble Tea TUI model used by cmd/tui. It accepts
// any implementation of Model (usually the framework-agnostic internal
// model) so tests can provide fakes.
func NewModel(ui Model) *TuiModel {
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
func NewProgram(ui Model) *tea.Program {
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
