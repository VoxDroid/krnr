package ui

import (
	"context"
	"strings"
	"testing"

	adapters "github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestFilterTypingUpdatesList(t *testing.T) {
	reg := &fakeRegistry{items: []adapters.CommandSetSummary{{Name: "alpha", Description: "A"}, {Name: "beta", Description: "B"}, {Name: "bravo", Description: "B2"}}}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	// enter filter mode
	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = m1.(*TuiModel)
	if !m.filterMode {
		t.Fatalf("expected filter mode to be active")
	}
	// type 'b' to filter
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = m2.(*TuiModel)
	// debug: log cached list names
	for _, s := range m.uiModel.ListCached() {
		t.Logf("cached: %s", s.Name)
	}
	view := m.list.View()
	t.Logf("list view after b:\n%s", view)
	if !strings.Contains(view, "beta") || strings.Contains(view, "alpha") {
		t.Fatalf("expected filtered view to include only 'beta'/'bravo' but got:\n%s", view)
	}
	// type 'r' to refine to 'bravo'
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}.Type, Runes: []rune{'r'}})
	m = m3.(*TuiModel)
	view2 := m.list.View()
	if !strings.Contains(view2, "bravo") || strings.Contains(view2, "beta\n") {
		t.Fatalf("expected filtered view to include only 'bravo' but got:\n%s", view2)
	}
	// pressing Enter while filtered should open the selected item
	// debug selection state before Enter
	t.Logf("index=%d selected=%v", m.list.Index(), m.list.SelectedItem())
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m4.(*TuiModel)
	if !m.showDetail || m.detailName != "bravo" {
		t.Fatalf("expected Enter to open 'bravo', got showDetail=%v detailName=%q", m.showDetail, m.detailName)
	}
}
