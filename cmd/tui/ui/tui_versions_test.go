package ui

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestVersionsPreviewRemoved(_ *testing.T) {
	// Preview removed by design — this test prevents regressions.
}

func TestVersionsSelectionPreviewsAndRestore(t *testing.T) {
	full := adapters.CommandSetSummary{Name: "vset", Description: "Versions set", Commands: []string{"echo current"}}
	vers := []adapters.Version{{Version: 2, CreatedAt: "2026-01-12T00:00:00Z", AuthorName: "alice", Description: "update", Commands: []string{"echo new2"}, Operation: "update"}, {Version: 1, CreatedAt: "2026-01-11T00:00:00Z", AuthorName: "bob", Description: "create", Commands: []string{"echo old1"}, Operation: "create"}}
	reg := &versionsFakeRegistry{items: []adapters.CommandSetSummary{{Name: "vset", Description: "Versions set"}}, full: full, versions: vers}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m = initTestModel(m)
	m1, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = m1.(*TuiModel)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(*TuiModel)

	// initial preview shows current commands
	if !strings.Contains(m.vp.View(), "echo current") {
		t.Fatalf("expected initial preview to include current command, got:\n%s", m.vp.View())
	}

	// focus right and select second entry (older version)
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = m3.(*TuiModel)
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = m4.(*TuiModel)

	// left preview should now show the older version commands
	if !strings.Contains(m.vp.View(), "echo old1") {
		t.Fatalf("expected preview to show older version commands, got:\n%s", m.vp.View())
	}

	// switch focus back to left - preview should return to current
	m5, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = m5.(*TuiModel)
	if !strings.Contains(m.vp.View(), "echo current") {
		t.Fatalf("expected preview to restore to current commands after switching focus back, got:\n%s", m.vp.View())
	}
}

func TestVersionsDeepScrollUpdatesPreview(t *testing.T) {
	// prepare many versions to ensure pagination and deep scroll
	full := adapters.CommandSetSummary{Name: "bigset", Description: "Many versions", Commands: []string{"echo current"}}
	var vers []adapters.Version
	for i := 80; i >= 1; i-- {
		vers = append(vers, adapters.Version{Version: i, CreatedAt: "2026-01-01T00:00:00Z", AuthorName: "user", Description: "v", Commands: []string{fmt.Sprintf("echo old%d", i)}, Operation: "update"})
	}
	reg := &versionsFakeRegistry{items: []adapters.CommandSetSummary{{Name: "bigset", Description: "Many versions"}}, full: full, versions: vers}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m = initTestModel(m)
	m1, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = m1.(*TuiModel)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(*TuiModel)
	// focus right
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = m3.(*TuiModel)
	// move down 60 times to reach v60 (starting at index 0 -> v80)
	for i := 0; i < 60; i++ {
		mN, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = mN.(*TuiModel)
	}
	// expected version at that point is Version 20 (since list sorted newest first)
	// but content should match the corresponding version's command
	if !strings.Contains(m.vp.View(), "echo old20") {
		t.Fatalf("expected preview after deep scroll to show 'echo old20', got:\n%s", m.vp.View())
	}
}

func TestVersionsSelectionShowsHistoricCommands(t *testing.T) {
	// latest has five commands, historic version has two
	full := adapters.CommandSetSummary{Name: "longhista", Description: "long history test", Commands: []string{"echo 1", "echo 2", "echo 3", "echo 4", "echo 5"}}
	vers := []adapters.Version{{Version: 3, CreatedAt: "2026-01-14T10:57:16Z", AuthorName: "alice", Description: "update", Commands: []string{"echo 60", "echo 61"}, Operation: "update"}, {Version: 1, CreatedAt: "2026-01-14T10:28:41Z", AuthorName: "bob", Description: "create", Commands: []string{"echo 1", "echo 2"}, Operation: "create"}}
	reg := &versionsFakeRegistry{items: []adapters.CommandSetSummary{{Name: "longhista", Description: "long history test"}}, full: full, versions: vers}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m = initTestModel(m)
	m1, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m = m1.(*TuiModel)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(*TuiModel)

	// sanity: should show latest (echo 5)
	if !strings.Contains(m.vp.View(), "echo 5") {
		t.Fatalf("expected initial preview to include latest command, got:\n%s", m.vp.View())
	}

	// focus right: immediately preview the currently-selected version (index 0 => v3)
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = m3.(*TuiModel)
	view0 := m.vp.View()
	if !strings.Contains(view0, "echo 61") {
		t.Fatalf("expected preview on focus to show v3's command 'echo 61', got:\n%s", view0)
	}

	// simulate an external selection change to versions list (e.g., via mouse)
	// directly select the next item on the versions list without sending a key
	m.versionsList.Select(1)
	// send an unrelated message (WindowSizeMsg) so deferred reconciliation runs
	mN, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m = mN.(*TuiModel)

	// left preview should now reflect the selected historic version (v1)
	view := m.vp.View()
	if !strings.Contains(view, "echo 2") || strings.Contains(view, "echo 5") {
		t.Fatalf("expected preview to show historic commands (v1) and not latest after external selection change; got:\n%s", view)
	}
}
