// Package ui contains tests for the TUI package.
package ui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
)

func TestSaveViaModel(t *testing.T) {
	// fake registry that records saves
	f := &saveFakeRegistry{}
	ui := modelpkg.New(f, nil, nil, nil)
	cs := adapters.CommandSetSummary{Name: "nset", Description: "desc", Commands: []string{"echo hi"}, AuthorName: "me"}
	if err := ui.Save(context.Background(), cs); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if f.last.Name != "nset" {
		t.Fatalf("expected saved name nset got %s", f.last.Name)
	}
}

func TestCreateEntryViaKey(t *testing.T) {
	f := &saveFakeRegistry{}
	ui := modelpkg.New(f, nil, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	// Press 'c' to open create modal
	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = m1.(*TuiModel)
	if !m.editingMeta || !m.editor.create {
		t.Fatalf("expected create editor to be open")
	}
	// the editor should be rendered as the full-screen detail
	if shown, _ := m.IsDetailShown(); !shown {
		t.Fatalf("expected detail to be shown when creating an entry")
	}
	view := m.View()
	if !strings.Contains(view, "Create: New Entry") {
		t.Fatalf("expected Create modal title in view, got:\n%s", view)
	}
	// confirm focus hint via the presence of the focused field marker and ANSI
	if !strings.Contains(view, "Name: > ") {
		t.Fatalf("expected Name field prompt in view, got:\n%s", view)
	}
	// model should indicate the Name field is focused
	if m.editor.field != 0 {
		t.Fatalf("expected editor.field to be 0 (Name), got %d", m.editor.field)
	} // Press 'C' uppercase should also open create modal when starting fresh
	mFresh := NewModel(ui)
	mFresh.Init()()
	mU, _ := mFresh.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	mU2 := mU.(*TuiModel)
	if !mU2.editingMeta || !mU2.editor.create {
		t.Fatalf("expected create editor to be open when pressing uppercase 'C' on main screen")
	}
	if shown, _ := mU2.IsDetailShown(); !shown {
		t.Fatalf("expected detail to be shown when creating an entry via uppercase 'C')")
	}
	// fill Name
	for _, r := range []rune{'n', 's', 'e', 't'} {
		m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = m2.(*TuiModel)
	}
	// tab -> Description (field 1)
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = m3.(*TuiModel)
	for _, r := range []rune{'d', 'e', 's', 'c'} {
		m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = m4.(*TuiModel)
	}
	// tab -> Author (field 2)
	m5, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = m5.(*TuiModel)
	for _, r := range []rune{'m', 'e'} {
		m6, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = m6.(*TuiModel)
	}
	// tab -> Email (field 3)
	m7, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = m7.(*TuiModel)
	for _, r := range []rune{'m', 'e', '@', 'e', 'x', 'a', 'm', 'p', 'l', 'e', '.', 'c', 'o', 'm'} {
		m8, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = m8.(*TuiModel)
	}
	// tab -> Tags (field 4)
	m9, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = m9.(*TuiModel)
	for _, r := range []rune{'t', 'a', 'g', '1'} {
		m10, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = m10.(*TuiModel)
	}
	// tab until commands field (field 5)
	for i := 0; i < 10 && m.editor.field != 5; i++ {
		m11, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = m11.(*TuiModel)
	}
	// add a command and type 'echo hi'
	m12, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	m = m12.(*TuiModel)
	for _, r := range []rune{'e', 'c', 'h', 'o', ' ', 'h', 'i'} {
		m13, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = m13.(*TuiModel)
	}
	// save with Ctrl+S
	m14, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = m14.(*TuiModel)
	if f.last.Name != "nset" {
		t.Fatalf("expected saved name nset got %s", f.last.Name)
	}
	if len(f.last.Commands) != 1 || f.last.Commands[0] != "echo hi" {
		t.Fatalf("expected saved commands to include 'echo hi', got %#v", f.last.Commands)
	}
	if f.last.AuthorName != "me" || f.last.AuthorEmail != "me@example.com" {
		t.Fatalf("expected author set, got %q <%q>", f.last.AuthorName, f.last.AuthorEmail)
	}

	// create attempt with empty name should be rejected and not saved
	mCreate, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	mCreateModel := mCreate.(*TuiModel)
	// ensure name empty
	mCreateModel.editor.name = "  "
	// clear prior logs so we only inspect spam-related activity
	mCreateModel.logs = nil
	// press save several times to ensure repeated attempts don't create an empty entry
	var lastFail *TuiModel
	for i := 0; i < 5; i++ {
		mFail, _ := mCreateModel.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
		lastFail = mFail.(*TuiModel)
		if !lastFail.editingMeta {
			t.Fatalf("expected editor to still be open after failed save")
		}
	}
	// ensure there were no 'attempting save' logs for empty name
	attempts := 0
	for _, l := range lastFail.logs {
		if strings.Contains(l, "attempting save") {
			attempts++
		}
	}
	if attempts != 0 {
		t.Fatalf("expected 0 save attempts for empty-name spams, got %d (logs: %#v)", attempts, lastFail.logs)
	}
	// should still have the prior saved name
	if f.last.Name != "" && f.last.Name != "nset" {
		t.Fatalf("unexpected saved name after failed create: %q", f.last.Name)
	}
	// expect an error log mentioning invalid name (via replace commands wrapper)
	found := false
	for _, l := range lastFail.logs {
		if strings.Contains(l, "invalid name") || strings.Contains(l, "replace commands: invalid name") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected error log about invalid name, got logs: %#v", lastFail.logs)
	}

	// Now attempt to create another entry with the same name — should be rejected and produce a footer notification
	// Ensure any prior editor is canceled first
	mEsc, _ := lastFail.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mEscM := mEsc.(*TuiModel)
	mCreate2, _ := mEscM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	mCreate2M := mCreate2.(*TuiModel)
	mCreate2M.editor.name = "nset"
	// add a command so validation reaches Save
	for i := 0; i < 6 && mCreate2M.editor.field != 5; i++ {
		m20, _ := mCreate2M.Update(tea.KeyMsg{Type: tea.KeyTab})
		mCreate2M = m20.(*TuiModel)
	}
	mA, _ := mCreate2M.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	mCreate2M = mA.(*TuiModel)
	for _, r := range []rune{'e', 'c', 'h', 'o'} {
		mB, _ := mCreate2M.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		mCreate2M = mB.(*TuiModel)
	}
	var mFailDup tea.Model
	var mFailDupM *TuiModel
	for i := 0; i < 5; i++ {
		mFailDup, _ = mCreate2M.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
		mFailDupM = mFailDup.(*TuiModel)
		if !mFailDupM.editingMeta {
			t.Fatalf("expected editor to still be open after duplicate-name save")
		}
	}
	// ensure exactly one save attempt was made
	attempts = 0
	for _, l := range mFailDupM.logs {
		if strings.Contains(l, "attempting save") {
			attempts++
		}
	}
	if attempts != 0 {
		t.Fatalf("expected 0 save attempts for duplicate-name spams (UI-level prevented attempts), got %d (logs: %#v)", attempts, mFailDupM.logs)
	}
	// should not overwrite saved record
	if f.last.Name != "nset" {
		t.Fatalf("expected saved name to remain 'nset', got %q", f.last.Name)
	}
	// expect footer notification was set
	foundNotif := false
	for _, l := range mFailDupM.logs {
		if strings.Contains(l, "notification:") {
			foundNotif = true
			break
		}
	}
	if !foundNotif {
		t.Fatalf("expected notification log, logs: %#v", mFailDupM.logs)
	}
	if mFailDupM.notification == "" {
		t.Fatalf("expected m.notification to be set, logs: %#v", mFailDupM.logs)
	}
	view = mFailDupM.View()
	if !strings.Contains(view, "already exists") {
		t.Fatalf("expected footer to show duplicate-name notification, got view:\n%s", view)
	}
	// now simulate typing in the Name field — notification should clear
	mFailDupM.editor.field = 0
	mX, _ := mFailDupM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	mXModel := mX.(*TuiModel)
	if mXModel.notification != "" {
		t.Fatalf("expected notification to be cleared after typing, got: %q", mXModel.notification)
	}
	view = mXModel.View()
	if strings.Contains(view, "already exists") {
		t.Fatalf("expected footer to not show duplicate-name notification after typing, got view:\n%s", view)
	}
}

type saveFakeRegistry struct{ last adapters.CommandSetSummary }

func (s *saveFakeRegistry) ListCommandSets(_ context.Context) ([]adapters.CommandSetSummary, error) {
	if s.last.Name == "" {
		return nil, nil
	}
	return []adapters.CommandSetSummary{s.last}, nil
}
func (s *saveFakeRegistry) GetCommandSet(_ context.Context, name string) (adapters.CommandSetSummary, error) {
	// return saved item when name matches s.last
	if s.last.Name == name {
		return s.last, nil
	}
	return adapters.CommandSetSummary{}, adapters.ErrNotFound
}
func (s *saveFakeRegistry) GetCommands(_ context.Context, _ string) ([]string, error) {
	return nil, adapters.ErrNotFound
}
func (s *saveFakeRegistry) SaveCommandSet(_ context.Context, cs adapters.CommandSetSummary) error {
	s.last = cs
	return nil
}
func (s *saveFakeRegistry) DeleteCommandSet(_ context.Context, _ string) error { return nil }
func (s *saveFakeRegistry) ReplaceCommands(_ context.Context, _ string, _ []string) error {
	return nil
}
func (s *saveFakeRegistry) UpdateCommandSet(_ context.Context, _ string, cs adapters.CommandSetSummary) error {
	s.last = cs
	return nil
}
func (s *saveFakeRegistry) ListVersionsByName(_ context.Context, _ string) ([]adapters.Version, error) {
	return nil, nil
}
func (s *saveFakeRegistry) ApplyVersionByName(_ context.Context, _ string, _ int) error { return nil }
