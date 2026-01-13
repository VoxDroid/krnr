package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
)

func TestNewModelInitializesList(t *testing.T) {
	fakeReg := &fakeRegistry{items: []adapters.CommandSetSummary{{Name: "one", Description: "First"}, {Name: "two", Description: "Second"}}}
	ui := modelpkg.New(fakeReg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	// Init should populate items via the Init cmd
	m.Init()()
	if len(m.list.Items()) != 2 {
		t.Fatalf("expected 2 items got %d", len(m.list.Items()))
	}
}

func TestInitPopulatesPreview(t *testing.T) {
	reg := &detailFakeRegistry{items: []adapters.CommandSetSummary{{Name: "one", Description: "First"}}, full: adapters.CommandSetSummary{Name: "one", Description: "First", Commands: []string{"echo hi"}}}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	if m.vp.View() == "" {
		t.Fatalf("expected preview content after Init")
	}
	if !strings.Contains(m.vp.View(), "echo hi") {
		t.Fatalf("expected command in preview after Init")
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func TestDescriptionIndentAndCommandAlignment(t *testing.T) {
	// create a set with a multi-line description and two commands
	full := adapters.CommandSetSummary{
		Name:        "with-params",
		Description: "Param demo",
		Commands:    []string{"echo User: {{user}}", "echo Token: {{token}}"},
		AuthorName:  "VoxDroid",
		AuthorEmail: "izeno.contact@gmail.com",
	}
	reg := &detailFakeRegistry{items: []adapters.CommandSetSummary{{Name: "with-params", Description: "Param demo"}}, full: full}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	view := m.vp.View()
	// description should appear on the line after the header and be whitespace-only prefixed
	posDesc := strings.Index(view, "Description:")
	posParam := strings.Index(view, "Param demo")
	t.Logf("posDesc=%d posParam=%d", posDesc, posParam)
	if posDesc == -1 || posParam == -1 || posParam <= posDesc {
		t.Fatalf("expected Param demo to appear after Description:, got:\n%s", view)
	}
	between := view[posDesc+len("Description:") : posParam]
	if strings.TrimSpace(between) != "" {
		t.Fatalf("expected only whitespace between Description: and Param demo, got %q", between)
	}

	// also assert description is indented by the same base offset used for commands
	maxPrefix := 0
	for i := range full.Commands {
		p := fmt.Sprintf("%d) ", i+1)
		if l := len(p); l > maxPrefix {
			maxPrefix = l
		}
	}
	expectedDelta := 2 + maxPrefix + 1 // the formatter indents by 2 + maxPrefix + 1 spaces relative to the header
	// find header line and the following non-empty line
	lines := strings.Split(view, "\n")
	headIdx := -1
	for i, ln := range lines {
		if strings.Contains(ln, "Description:") {
			headIdx = i
			break
		}
	}
	if headIdx == -1 || headIdx+1 >= len(lines) {
		t.Fatalf("couldn't locate Description header and line in view:\n%s", view)
	}
	// skip empty lines after header
	paramLine := lines[headIdx+1]
	if strings.TrimSpace(paramLine) == "" && headIdx+2 < len(lines) {
		paramLine = lines[headIdx+2]
	}
	leadingHeader := len(lines[headIdx]) - len(strings.TrimLeft(lines[headIdx], " "))
	leadingParam := len(paramLine) - len(strings.TrimLeft(paramLine, " "))
	delta := leadingParam - leadingHeader
	if delta < expectedDelta {
		t.Fatalf("expected param to be indented at least %d spaces relative to header, got %d (view:\n%s)", expectedDelta, delta, view)
	}

	// ensure both command lines show the prefix and a small gap before the echoed text
	if !strings.Contains(view, "1)  echo") || !strings.Contains(view, "2)  echo") {
		t.Fatalf("expected both command lines to include '1)  echo' and '2)  echo', got:\n%s", view)
	}
}

func TestFilterModeIgnoresControlsAndEscCancels(t *testing.T) {
	reg := &fakeRegistry{items: []adapters.CommandSetSummary{{Name: "a", Description: "A"}, {Name: "b", Description: "B"}}}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	// enter filter mode by sending '/'
	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = m1.(*TuiModel)
	if m.list.FilterState() != list.Filtering {
		t.Fatalf("expected list to be in filtering state")
	}
	// footer should show only the esc hint while filtering
	view := m.View()
	if !strings.Contains(view, "(esc) quit filter") {
		t.Fatalf("expected filter hint in footer, got:\n%s", view)
	}
	// ensure other shortcuts are hidden for cleanliness
	if strings.Contains(view, "(q) quit") || strings.Contains(view, "(r) run") || strings.Contains(view, "(Enter) details") {
		t.Fatalf("expected other shortcuts to be hidden while filtering, got footer:\n%s", view)
	}
	// pressing 'q' while filtering should NOT quit; the list should still be in filtering state
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = m2.(*TuiModel)
	if m.list.FilterState() != list.Filtering {
		t.Fatalf("expected filtering to remain active after pressing q")
	}
	// pressing ESC should cancel filter
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = m3.(*TuiModel)
	if m.list.FilterState() == list.Filtering {
		t.Fatalf("expected filtering to be cancelled after ESC")
	}
}
func TestEnterShowsFullScreenWithDryRun(t *testing.T) {
	reg := &detailFakeRegistry{items: []adapters.CommandSetSummary{{Name: "one", Description: "First"}}, full: adapters.CommandSetSummary{Name: "one", Description: "First", Commands: []string{"echo hi", "echo there is a long line that will wrap around the width for testing"}, AuthorName: "me", AuthorEmail: "me@example.com", Tags: []string{"tag1"}}}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	// ensure viewport has a reasonable width so wrapping applies
	m1, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m = m1.(*TuiModel)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(*TuiModel)
	if !m.showDetail {
		t.Fatalf("expected showDetail true")
	}
	if !strings.Contains(m.detail, "Dry-run preview") {
		t.Fatalf("expected dry-run preview in detail")
	}
	// dry-run should show simulated output (echo prints its argument)
	if !strings.Contains(m.detail, "hi") {
		t.Fatalf("expected dry-run output 'hi' in detail, got:\n%s", m.detail)
	}
	if strings.Contains(m.detail, "$ echo hi") {
		t.Fatalf("expected simulated output, not raw command: %s", m.detail)
	}
}

func TestDetailViewShowsTitle(t *testing.T) {
	reg := &detailFakeRegistry{items: []adapters.CommandSetSummary{{Name: "sysinf", Description: "Info"}}, full: adapters.CommandSetSummary{Name: "sysinf", Description: "Info", Commands: []string{"systeminfo"}, CreatedAt: "2026-01-12T14:05:35Z"}}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	// give the UI a large width so title has space
	m1, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = m1.(*TuiModel)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(*TuiModel)
	view := m.View()
	if !strings.Contains(view, "krnr — sysinf Details") {
		t.Fatalf("expected title 'krnr — sysinf Details' in View(), got:\n%s", view)
	}
	if !strings.Contains(view, "(e) Edit") {
		t.Fatalf("expected detail view to include '(e) Edit' hint, got:\n%s", view)
	}
	if !strings.Contains(view, "(d) Delete") {
		t.Fatalf("expected detail view to include '(d) Delete' hint, got:\n%s", view)
	}
	if !strings.Contains(view, "(s) Export") {
		t.Fatalf("expected detail view to include '(s) Export' hint, got:\n%s", view)
	}
}

func TestEditReplacesCommands(t *testing.T) {
	d := t.TempDir()
	// create a small script that overwrites the file with new commands
	var script string
	if runtime.GOOS == "windows" {
		script = "@echo off\r\necho echo new1 > %1\r\necho echo new2 >> %1\r\nexit /b 0\r\n"
		scriptPath := filepath.Join(d, "fake-editor.bat")
		if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
			t.Fatalf("write script: %v", err)
		}
		_ = os.Setenv("EDITOR", scriptPath)
		defer func() { _ = os.Unsetenv("EDITOR") }()
	} else {
		script = "#!/bin/sh\nprintf 'echo new1\necho new2\n' > \"$1\"\nexit 0\n"
		scriptPath := filepath.Join(d, "fake-editor.sh")
		if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
			t.Fatalf("write script: %v", err)
		}
		if err := os.Chmod(scriptPath, 0o755); err != nil {
			t.Fatalf("chmod script: %v", err)
		}
		_ = os.Setenv("EDITOR", scriptPath)
		defer func() { _ = os.Unsetenv("EDITOR") }()
	}

	full := adapters.CommandSetSummary{Name: "one", Description: "First", Commands: []string{"echo hi"}}
	reg := &replaceFakeRegistry{items: []adapters.CommandSetSummary{{Name: "one", Description: "First"}}, full: full}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	m1, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = m1.(*TuiModel)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(*TuiModel)
	// trigger edit (should call EDITOR and then ReplaceCommands)
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = m3.(*TuiModel)
	if reg.lastName != "one" {
		t.Fatalf("expected ReplaceCommands called with name 'one', got %q", reg.lastName)
	}
	if len(reg.lastCommands) != 2 || reg.lastCommands[0] != "echo new1" || reg.lastCommands[1] != "echo new2" {
		t.Fatalf("unexpected replaced commands: %#v", reg.lastCommands)
	}
	// the preview in detail should reflect updated commands
	if !strings.Contains(m.detail, "echo new1") {
		t.Fatalf("expected updated command shown in detail, got:\n%s", m.detail)
	}
}

func TestDeleteFromDetailPromptsAndDeletesWhenConfirmed(t *testing.T) {
	full := adapters.CommandSetSummary{Name: "one", Description: "First", Commands: []string{"echo hi"}}
	reg := &replaceFakeRegistry{items: []adapters.CommandSetSummary{{Name: "one", Description: "First"}}, full: full}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	m1, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = m1.(*TuiModel)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(*TuiModel)
	// trigger delete prompt
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = m3.(*TuiModel)
	if !strings.Contains(m.detail, "Delete 'one' permanently?") {
		t.Fatalf("expected delete prompt in detail, got:\n%s", m.detail)
	}
	// confirm deletion (lowercase)
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = m4.(*TuiModel)
	if reg.lastDeleted != "one" {
		t.Fatalf("expected DeleteCommandSet called for 'one', got %q", reg.lastDeleted)
	}
	if m.showDetail {
		t.Fatalf("expected showDetail false after deletion")
	}
	if len(m.list.Items()) != 0 {
		t.Fatalf("expected list to be empty after deletion, got %d", len(m.list.Items()))
	}
}

// fake import/export adapter for tests
type fakeImpExp struct{
	lastName string
	lastDest string
}
func (f *fakeImpExp) Export(_ context.Context, name string, dest string) error { f.lastName = name; f.lastDest = dest; return nil }
func (f *fakeImpExp) Import(_ context.Context, _ string, _ string) error { return nil }

func TestExportFromDetailPromptsAndExportsWhenConfirmed(t *testing.T) {
	full := adapters.CommandSetSummary{Name: "one", Description: "First", Commands: []string{"echo hi"}}
	reg := &replaceFakeRegistry{items: []adapters.CommandSetSummary{{Name: "one", Description: "First"}}, full: full}
	imp := &fakeImpExp{}
	ui := modelpkg.New(reg, &fakeExec{}, imp, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	m1, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = m1.(*TuiModel)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(*TuiModel)
	// create a file at the default destination so we simulate an existing file
	defaultDst := filepath.Join(os.TempDir(), "one.db")
	_ = os.WriteFile(defaultDst, []byte("x"), 0o644)
	defer func() { _ = os.Remove(defaultDst) }()
	// trigger export prompt
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = m3.(*TuiModel)
	if !strings.Contains(m.detail, "Export 'one' to") {
		t.Fatalf("expected export prompt in detail, got:\n%s", m.detail)
	}
	// confirm export
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = m4.(*TuiModel)
	if imp.lastName != "one" {
		t.Fatalf("expected Export called for 'one', got %q", imp.lastName)
	}
	if imp.lastDest == "" {
		t.Fatalf("expected destination to be set")
	}
	if imp.lastDest == defaultDst {
		t.Fatalf("expected destination to be different when default exists, got the same: %s", imp.lastDest)
	}
	if !strings.Contains(strings.Join(m.logs, "\n"), "exported") {
		t.Fatalf("expected exported log, got: %v", m.logs)
	}}

func TestDeleteConfirmUppercaseY(t *testing.T) {
	full := adapters.CommandSetSummary{Name: "upcase", Description: "Up", Commands: []string{"echo hi"}}
	reg := &replaceFakeRegistry{items: []adapters.CommandSetSummary{{Name: "upcase", Description: "Up"}}, full: full}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	m1, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = m1.(*TuiModel)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(*TuiModel)
	// prompt
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = m3.(*TuiModel)
	if !strings.Contains(m.detail, "Delete 'upcase' permanently?") {
		t.Fatalf("expected delete prompt in detail, got:\n%s", m.detail)
	}
	// confirm with uppercase 'Y'
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
	m = m4.(*TuiModel)
	if reg.lastDeleted != "upcase" {
		t.Fatalf("expected DeleteCommandSet called for 'upcase', got %q", reg.lastDeleted)
	}
}

func TestExportConfirmUppercaseY(t *testing.T) {
	full := adapters.CommandSetSummary{Name: "upcaseexp", Description: "Up", Commands: []string{"echo hi"}}
	reg := &replaceFakeRegistry{items: []adapters.CommandSetSummary{{Name: "upcaseexp", Description: "Up"}}, full: full}
	imp := &fakeImpExp{}
	ui := modelpkg.New(reg, &fakeExec{}, imp, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	m1, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = m1.(*TuiModel)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(*TuiModel)
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = m3.(*TuiModel)
	if !strings.Contains(m.detail, "Export 'upcaseexp' to") {
		t.Fatalf("expected export prompt in detail, got:\n%s", m.detail)
	}
	// confirm with uppercase Y
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
	m = m4.(*TuiModel)
	if imp.lastName != "upcaseexp" {
		t.Fatalf("expected Export called for 'upcaseexp', got %q", imp.lastName)
	}
} 
// minimal fakes for testing
type fakeRegistry struct{ items []adapters.CommandSetSummary }

func (f *fakeRegistry) ListCommandSets(_ context.Context) ([]adapters.CommandSetSummary, error) {
	return f.items, nil
}
func (f *fakeRegistry) GetCommandSet(_ context.Context, _ string) (adapters.CommandSetSummary, error) {
	return adapters.CommandSetSummary{}, nil
}
func (f *fakeRegistry) SaveCommandSet(_ context.Context, _ adapters.CommandSetSummary) error {
	return nil
}
func (f *fakeRegistry) DeleteCommandSet(_ context.Context, _ string) error { return nil }
func (f *fakeRegistry) GetCommands(_ context.Context, _ string) ([]string, error) {
	return []string{"echo hello"}, nil
}
func (f *fakeRegistry) ReplaceCommands(_ context.Context, _ string, _ []string) error { return nil }

type fakeExec struct{}

func (f *fakeExec) Run(_ context.Context, _ string, _ []string) (adapters.RunHandle, error) {
	return nil, nil
}

// detailFakeRegistry returns a full CommandSet via GetCommandSet so the UI can
// render full details and dry-run preview in tests.
type detailFakeRegistry struct {
	items []adapters.CommandSetSummary
	full  adapters.CommandSetSummary
}

func (f *detailFakeRegistry) ListCommandSets(_ context.Context) ([]adapters.CommandSetSummary, error) {
	return f.items, nil
}
func (f *detailFakeRegistry) GetCommandSet(_ context.Context, _ string) (adapters.CommandSetSummary, error) {
	return f.full, nil
}
func (f *detailFakeRegistry) SaveCommandSet(_ context.Context, _ adapters.CommandSetSummary) error {
	return nil
}
func (f *detailFakeRegistry) DeleteCommandSet(_ context.Context, _ string) error { return nil }
func (f *detailFakeRegistry) GetCommands(_ context.Context, _ string) ([]string, error) {
	return []string{"echo hello"}, nil
}
func (f *detailFakeRegistry) ReplaceCommands(_ context.Context, _ string, _ []string) error { return nil }

// replaceFakeRegistry supports ReplaceCommands and records the last call
// for assertions in tests.
type replaceFakeRegistry struct{
	items []adapters.CommandSetSummary
	full  adapters.CommandSetSummary
	lastName string
	lastCommands []string
	lastDeleted string
} 

func (f *replaceFakeRegistry) ListCommandSets(_ context.Context) ([]adapters.CommandSetSummary, error) { return f.items, nil }
func (f *replaceFakeRegistry) GetCommandSet(_ context.Context, _ string) (adapters.CommandSetSummary, error) { return f.full, nil }
func (f *replaceFakeRegistry) SaveCommandSet(_ context.Context, _ adapters.CommandSetSummary) error { return nil }
func (f *replaceFakeRegistry) DeleteCommandSet(_ context.Context, name string) error {
	f.lastDeleted = name
	// remove from items
	newItems := []adapters.CommandSetSummary{}
	for _, it := range f.items {
		if it.Name != name {
			newItems = append(newItems, it)
		}
	}
	f.items = newItems
	if f.full.Name == name {
		f.full = adapters.CommandSetSummary{}
	}
	return nil
} 
func (f *replaceFakeRegistry) GetCommands(_ context.Context, _ string) ([]string, error) { return f.full.Commands, nil }
func (f *replaceFakeRegistry) ReplaceCommands(_ context.Context, name string, cmds []string) error {
	f.lastName = name
	f.lastCommands = append([]string{}, cmds...)
	f.full.Commands = append([]string{}, cmds...)
	return nil
}
