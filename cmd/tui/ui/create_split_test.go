package ui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
)

func setupCreateModel() (*TuiModel, *saveFakeRegistry) {
	f := &saveFakeRegistry{}
	ui := modelpkg.New(f, nil, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m = initTestModel(m)
	return m, f
}

// saveEntryNameAndCommand opens the create modal, types `name`, navigates to
// commands field and adds the given `cmd`, then saves the entry.
func saveEntryNameAndCommand(t *testing.T, m *TuiModel, name, cmd string) *TuiModel {
	t.Helper()
	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = m1.(*TuiModel)
	for _, r := range name {
		m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = m2.(*TuiModel)
	}
	// tab until commands field (field 5)
	for i := 0; i < 10 && m.editor.field != 5; i++ {
		m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = m2.(*TuiModel)
	}
	mA, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	m = mA.(*TuiModel)
	for _, r := range cmd {
		mB, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = mB.(*TuiModel)
	}
	mC, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = mC.(*TuiModel)
	return m
}

func TestCreateEntry_OpenCreateModal(t *testing.T) {
	m, _ := setupCreateModel()
	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = m1.(*TuiModel)
	if !m.editingMeta || !m.editor.create {
		t.Fatalf("expected create editor to be open")
	}
	if shown, _ := m.IsDetailShown(); !shown {
		t.Fatalf("expected detail to be shown when creating an entry")
	}
	view := m.View()
	if !contains(view, "Create: New Entry") {
		t.Fatalf("expected Create modal title in view, got: %s", view)
	}
	if !contains(view, "Name: > ") {
		t.Fatalf("expected Name field prompt in view, got: %s", view)
	}
}

// fillCreateModal fills the create modal fields for name, description,
// author and email, and tags. It assumes the create modal is open.
func fillCreateModal(t *testing.T, m *TuiModel, name, desc, author, email, tags string) *TuiModel {
	t.Helper()
	// type name
	for _, r := range name {
		m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = m2.(*TuiModel)
	}
	// description
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = m3.(*TuiModel)
	for _, r := range desc {
		m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = m4.(*TuiModel)
	}
	// author
	m5, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = m5.(*TuiModel)
	for _, r := range author {
		m6, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = m6.(*TuiModel)
	}
	// email
	m7, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = m7.(*TuiModel)
	for _, r := range email {
		m8, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = m8.(*TuiModel)
	}
	// tags
	m9, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = m9.(*TuiModel)
	for _, r := range tags {
		m10, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = m10.(*TuiModel)
	}
	return m
}

func TestCreateEntry_FillAndSave(t *testing.T) {
	m, f := setupCreateModel()
	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = m1.(*TuiModel)
	m = fillCreateModal(t, m, "nset", "desc", "me", "me@example.com", "tag1")
	// tab until commands field (field 5)
	for i := 0; i < 10 && m.editor.field != 5; i++ {
		m11, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = m11.(*TuiModel)
	}
	m12, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	m = m12.(*TuiModel)
	for _, r := range []rune{'e', 'c', 'h', 'o', ' ', 'h', 'i'} {
		m13, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = m13.(*TuiModel)
	}
	m14, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	_ = m14
	if f.last.Name != "nset" {
		t.Fatalf("expected saved name nset got %s", f.last.Name)
	}
	if len(f.last.Commands) != 1 || f.last.Commands[0] != "echo hi" {
		t.Fatalf("expected saved commands to include 'echo hi', got %#v", f.last.Commands)
	}
	if f.last.AuthorName != "me" || f.last.AuthorEmail != "me@example.com" {
		t.Fatalf("expected author set, got %q <%q>", f.last.AuthorName, f.last.AuthorEmail)
	}
}

func TestCreateEntry_EmptyNameRejected(t *testing.T) {
	m, _ := setupCreateModel()
	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = m1.(*TuiModel)
	m.editor.name = "  "
	m.logs = nil
	var lastFail *TuiModel
	for i := 0; i < 5; i++ {
		mFail, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
		lastFail = mFail.(*TuiModel)
		if !lastFail.editingMeta {
			t.Fatalf("expected editor to still be open after failed save")
		}
	}
	attempts := 0
	for _, l := range lastFail.logs {
		if strings.Contains(l, "attempting save") {
			attempts++
		}
	}
	if attempts != 0 {
		t.Fatalf("expected 0 save attempts for empty-name spams, got %d (logs: %#v)", attempts, lastFail.logs)
	}
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
}

// attemptCreateDuplicate opens the create editor, sets name to `name`, adds
// a minimal command and attempts to save repeatedly to capture the final
// editor state after duplicate save attempts.
func attemptCreateDuplicate(t *testing.T, m *TuiModel, name string, tries int) *TuiModel {
	t.Helper()
	mEsc, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mEsc.(*TuiModel)
	mCreate2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = mCreate2.(*TuiModel)
	m.editor.name = name
	for i := 0; i < 6 && m.editor.field != 5; i++ {
		m20, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = m20.(*TuiModel)
	}
	mA2, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	m = mA2.(*TuiModel)
	for _, r := range []rune{'e', 'c', 'h', 'o'} {
		mB2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = mB2.(*TuiModel)
	}
	var mFailDup tea.Model
	var mFailDupM *TuiModel
	for i := 0; i < tries; i++ {
		mFailDup, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
		mFailDupM = mFailDup.(*TuiModel)
		if !mFailDupM.editingMeta {
			t.Fatalf("expected editor to still be open after duplicate-name save")
		}
	}
	return mFailDupM
}

func TestCreateEntry_DuplicateRejected(t *testing.T) {
	m, f := setupCreateModel()
	// first, save a baseline entry named 'nset'
	m = saveEntryNameAndCommand(t, m, "nset", "echo")
	if f.last.Name != "nset" {
		t.Fatalf("expected saved name nset got %s", f.last.Name)
	}
	mFailDupM := attemptCreateDuplicate(t, m, "nset", 5)
	// allow at most one 'attempting save' entry (guard against UI-level spamming)
	if countAttemptingSaveLogs(mFailDupM) > 1 {
		t.Fatalf("expected at most 1 save attempt for duplicate-name spams (UI-level prevented repeated attempts), got %d (logs: %#v)", countAttemptingSaveLogs(mFailDupM), mFailDupM.logs)
	}
	if f.last.Name != "nset" {
		t.Fatalf("expected saved name to remain 'nset', got %q", f.last.Name)
	}
	assertDuplicateNotification(t, mFailDupM)
}

func countAttemptingSaveLogs(m *TuiModel) int {
	c := 0
	for _, l := range m.logs {
		if strings.Contains(l, "attempting save") {
			c++
		}
	}
	return c
}

func assertDuplicateNotification(t *testing.T, m *TuiModel) {
	t.Helper()
	foundNotif := false
	for _, l := range m.logs {
		if strings.Contains(l, "notification:") {
			foundNotif = true
			break
		}
	}
	if !foundNotif {
		t.Fatalf("expected notification log, logs: %#v", m.logs)
	}
	if m.notification == "" {
		t.Fatalf("expected m.notification to be set, logs: %#v", m.logs)
	}
	view := m.View()
	if !contains(view, "already exists") {
		t.Fatalf("expected footer to show duplicate-name notification, got view:\n%s", view)
	}
}

func TestCreateEntry_DuplicateTypingClearsNotification(t *testing.T) {
	m, f := setupCreateModel()
	// create base entry
	m = saveEntryNameAndCommand(t, m, "nset", "echo")
	if f.last.Name != "nset" {
		t.Fatalf("expected saved name nset got %s", f.last.Name)
	}
	// attempt duplicate
	mEsc, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mEsc.(*TuiModel)
	mCreate2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = mCreate2.(*TuiModel)
	m.editor.name = "nset"
	for i := 0; i < 6 && m.editor.field != 5; i++ {
		m20, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = m20.(*TuiModel)
	}
	mA2, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	m = mA2.(*TuiModel)
	for _, r := range []rune{'e', 'c', 'h', 'o'} {
		mB2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = mB2.(*TuiModel)
	}
	var mFailDup tea.Model
	var mFailDupM *TuiModel
	for i := 0; i < 3; i++ {
		mFailDup, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
		mFailDupM = mFailDup.(*TuiModel)
		if !mFailDupM.editingMeta {
			t.Fatalf("expected editor to still be open after duplicate-name save")
		}
	}
	// now typing in the Name field should clear notification
	mFailDupM.editor.field = 0
	mX, _ := mFailDupM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	mXModel := mX.(*TuiModel)
	if mXModel.notification != "" {
		t.Fatalf("expected notification to be cleared after typing, got: %q", mXModel.notification)
	}
	view := mXModel.View()
	if contains(view, "already exists") {
		t.Fatalf("expected footer to not show duplicate-name notification after typing, got view:\n%s", view)
	}
}
