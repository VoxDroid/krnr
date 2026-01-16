package ui

import (
	"context"
	"os"
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

func TestFilterMatchesTags(t *testing.T) {
	reg := &fakeRegistry{items: []adapters.CommandSetSummary{{Name: "s1", Description: "", Tags: []string{"db", "alpha"}}, {Name: "s2", Description: "", Tags: []string{"net"}}}}
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
	// type 'db' to filter by tag (regular search includes tags)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = m2.(*TuiModel)
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = m3.(*TuiModel)
	view := m.list.View()
	if !strings.Contains(view, "s1") || strings.Contains(view, "s2") {
		t.Fatalf("expected filtered view to include only 's1' (tag db) but got:\n%s", view)
	}
}

func TestFilterHashTagOnly(t *testing.T) {
	reg := &fakeRegistry{items: []adapters.CommandSetSummary{{Name: "database", Description: "", Tags: []string{}}, {Name: "s2", Description: "", Tags: []string{"db"}}}}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	// show detail so preview updates are observable
	m.setShowDetail(true)
	m.setDetailName("")
	// enter filter mode
	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = m1.(*TuiModel)
	if !m.filterMode {
		t.Fatalf("expected filter mode to be active")
	}
	// type '#db' to filter by tag only
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'#'}})
	m = m2.(*TuiModel)
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = m3.(*TuiModel)
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = m4.(*TuiModel)
	view := m.list.View()
	if !strings.Contains(view, "s2") || strings.Contains(view, "database") {
		t.Fatalf("expected '#db' to match only s2 with tag 'db', got:\n%s", view)
	}
	// the preview (viewport) should now show details for s2
	vview := m.vp.View()
	if !strings.Contains(vview, "s2") {
		t.Fatalf("expected preview to show 's2' details, got:\n%s", vview)
	}
}

func TestFilterUpdatesPreview(t *testing.T) {
	reg := &previewFakeRegistry{items: []adapters.CommandSetSummary{{Name: "alpha", Description: "A"}, {Name: "beta", Description: "B"}}}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	// show detail so preview updates are observable
	m.setShowDetail(true)
	m.setDetailName("")
	// enter filter mode
	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = m1.(*TuiModel)
	if !m.filterMode {
		t.Fatalf("expected filter mode to be active")
	}
	// type 'b' to filter to beta
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = m2.(*TuiModel)
	// preview should show details for 'beta'
	if !strings.Contains(m.vp.View(), "beta") {
		t.Fatalf("expected preview to update to 'beta', got:\n%s", m.vp.View())
	}
}

func TestFilterScrollUpdatesPreview(t *testing.T) {
	reg := &previewFakeRegistry{items: []adapters.CommandSetSummary{{Name: "beta", Description: "B"}, {Name: "bravo", Description: "B2"}}}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	// show detail so preview updates are observable
	m.setShowDetail(true)
	m.setDetailName("")
	// enter filter mode
	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = m1.(*TuiModel)
	if !m.filterMode {
		t.Fatalf("expected filter mode to be active")
	}
	// type 'b' to filter to beta/bravo
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = m2.(*TuiModel)
	// preview should show details for 'beta'
	if !strings.Contains(m.vp.View(), "beta") {
		t.Fatalf("expected preview to update to 'beta', got:\n%s", m.vp.View())
	}
	// press Down to select next match
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = m3.(*TuiModel)
	// preview should now show 'bravo'
	if !strings.Contains(m.vp.View(), "bravo") {
		t.Fatalf("expected preview to update to 'bravo', got:\n%s", m.vp.View())
	}
}

func TestApplyListFilterKey_NavigationUpdatesPreview(t *testing.T) {
	reg := &previewFakeRegistry{items: []adapters.CommandSetSummary{{Name: "beta", Description: "B"}, {Name: "bravo", Description: "B2"}}}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	// show detail so preview updates are observable
	m.setShowDetail(true)
	m.setDetailName("")
	// prime the filter by typing 'b' using the list-filter helper
	m2, _, _ := applyListFilterKey(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = m2.(*TuiModel)
	if !strings.Contains(m.vp.View(), "beta") {
		t.Fatalf("expected preview to update to 'beta', got:\n%s", m.vp.View())
	}
	// now navigate down using the same helper
	m3, _, _ := applyListFilterKey(m, tea.KeyMsg{Type: tea.KeyDown})
	m = m3.(*TuiModel)
	if !strings.Contains(m.vp.View(), "bravo") {
		t.Fatalf("expected preview to update to 'bravo' after navigation, got:\n%s", m.vp.View())
	}
}

func TestFilterNavigationKeys_UpdatePreviewAcrossKeys(t *testing.T) {
	// create several items so Home/End behavior is meaningful
	items := []adapters.CommandSetSummary{{Name: "b0"}, {Name: "b1"}, {Name: "b2"}, {Name: "b3"}, {Name: "b4"}}
	reg := &previewFakeRegistry{items: items}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	m.setShowDetail(true)
	m.setDetailName("")
	// enter filter mode and type 'b' to match all
	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = m1.(*TuiModel)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = m2.(*TuiModel)
	// initial preview should be first item b0
	if !strings.Contains(m.vp.View(), "b0") {
		t.Fatalf("expected preview to show 'b0' initially, got:\n%s", m.vp.View())
	}
	// press Down -> b1
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = m3.(*TuiModel)
	if !strings.Contains(m.vp.View(), "b1") {
		t.Fatalf("expected preview to update to 'b1' after Down, got:\n%s", m.vp.View())
	}
	// press End -> b4
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m = m4.(*TuiModel)
	if !strings.Contains(m.vp.View(), "b4") {
		t.Fatalf("expected preview to update to 'b4' after End, got:\n%s", m.vp.View())
	}
	// press Home -> b0
	m5, _ := m.Update(tea.KeyMsg{Type: tea.KeyHome})
	m = m5.(*TuiModel)
	if !strings.Contains(m.vp.View(), "b0") {
		t.Fatalf("expected preview to update to 'b0' after Home, got:\n%s", m.vp.View())
	}
	// press Up at top should wrap or stay at b0; ensure preview still consistent
	m6, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = m6.(*TuiModel)
	if m.vp.View() == "" {
		t.Fatalf("expected preview to remain populated after Up, got empty view")
	}
}

func TestFilterUpdatesPreviewWhenDetailHidden(t *testing.T) {
	reg := &previewFakeRegistry{items: []adapters.CommandSetSummary{{Name: "beta", Description: "B"}, {Name: "bravo", Description: "B2"}}}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	// ensure detail pane is hidden
	m.setShowDetail(false)
	m.setDetailName("")
	// enter filter mode
	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = m1.(*TuiModel)
	// type 'b' to filter
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = m2.(*TuiModel)
	// preview should show details for 'beta' even though detail pane is hidden
	if !strings.Contains(m.vp.View(), "beta") {
		t.Fatalf("expected preview to update to 'beta' while filtering, got:\n%s", m.vp.View())
	}
	// press Down to select next match
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = m3.(*TuiModel)
	// preview should now show 'bravo'
	if !strings.Contains(m.vp.View(), "bravo") {
		t.Fatalf("expected preview to update to 'bravo', got:\n%s", m.vp.View())
	}
}

func TestPreviewLoggingWhenDebugEnabled(t *testing.T) {
	old := os.Getenv("KRNR_TUI_DEBUG_PREVIEW")
	_ = os.Setenv("KRNR_TUI_DEBUG_PREVIEW", "1")
	defer os.Setenv("KRNR_TUI_DEBUG_PREVIEW", old)

	reg := &previewFakeRegistry{items: []adapters.CommandSetSummary{{Name: "b0"}, {Name: "b1"}}}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	m.setShowDetail(true)
	m.setDetailName("")
	// type 'b' to filter
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = m2.(*TuiModel)
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = m3.(*TuiModel)
	if !m.logsContains("preview_update: b0") {
		t.Fatalf("expected logs to contain initial preview update, got: %+v", m.logs)
	}
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = m4.(*TuiModel)
	idx := m.list.Index()
	if idx != 1 {
		t.Fatalf("expected selection index to be 1 after Down, got=%d; logs=%+v", idx, m.logs)
	}
	if !m.logsContains("preview_update: b1") {
		t.Fatalf("expected logs to contain preview update to b1, got: %+v", m.logs)
	}
}

// previewFakeRegistry used to provide GetCommandSet that reflects requested name
// so preview updates in tests can assert the correct content.
type previewFakeRegistry struct{ items []adapters.CommandSetSummary }

func (p *previewFakeRegistry) ListCommandSets(_ context.Context) ([]adapters.CommandSetSummary, error) { return p.items, nil }
func (p *previewFakeRegistry) ReopenDB(ctx context.Context) error                                          { return nil }
func (p *previewFakeRegistry) GetCommandSet(_ context.Context, name string) (adapters.CommandSetSummary, error) {
	return adapters.CommandSetSummary{Name: name, Description: "desc", Commands: []string{"echo hi"}}, nil
}
func (p *previewFakeRegistry) SaveCommandSet(_ context.Context, _ adapters.CommandSetSummary) error { return nil }
func (p *previewFakeRegistry) DeleteCommandSet(_ context.Context, _ string) error                    { return nil }
func (p *previewFakeRegistry) GetCommands(_ context.Context, _ string) ([]string, error)             { return nil, adapters.ErrNotFound }
func (p *previewFakeRegistry) ReplaceCommands(_ context.Context, _ string, _ []string) error         { return nil }
func (p *previewFakeRegistry) UpdateCommandSet(_ context.Context, _ string, _ adapters.CommandSetSummary) error {
	return nil
}
func (p *previewFakeRegistry) UpdateCommandSetAndReplaceCommands(_ context.Context, _ string, _ adapters.CommandSetSummary) error {
	return nil
}
func (p *previewFakeRegistry) ListVersionsByName(_ context.Context, _ string) ([]adapters.Version, error) { return nil, nil }
func (p *previewFakeRegistry) ApplyVersionByName(_ context.Context, _ string, _ int) error                { return nil }
