package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	// open in-TUI editor
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = m3.(*TuiModel)
	if !m.editingMeta {
		t.Fatalf("expected editor to be open")
	}
	// cycle to commands field (tab 3 times)
	for i := 0; i < 3; i++ {
		m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = m4.(*TuiModel)
	}
	// add a new command (Ctrl+A) and type 'echo new1'
	m5, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	m = m5.(*TuiModel)
	for _, r := range []rune{'e', 'c', 'h', 'o', ' ', 'n', 'e', 'w', '1'} {
		m6, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = m6.(*TuiModel)
	}
	// save with Ctrl+S
	m7, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = m7.(*TuiModel)
	if reg.lastName != "one" {
		t.Fatalf("expected UpdateCommandSet called with name 'one', got %q", reg.lastName)
	}
	if len(reg.lastCommands) != 2 {
		t.Fatalf("expected two commands after edit, got %#v", reg.lastCommands)
	}
	if reg.lastCommands[1] != "echo new1" {
		t.Fatalf("expected second command to be 'echo new1', got %q", reg.lastCommands[1])
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
type fakeImpExp struct {
	lastName string
	lastDest string
}

func (f *fakeImpExp) Export(_ context.Context, name string, dest string) error {
	f.lastName = name
	f.lastDest = dest
	return nil
}
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
	}
}

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

func TestVersionsPanelAndRollback(t *testing.T) {
	// prepare fake registry with versions
	full := adapters.CommandSetSummary{Name: "vset", Description: "Versions set", Commands: []string{"echo current"}}
	vers := []adapters.Version{{Version: 2, CreatedAt: "2026-01-12T00:00:00Z", AuthorName: "alice", Description: "update", Commands: []string{"echo new2"}, Operation: "update"}, {Version: 1, CreatedAt: "2026-01-11T00:00:00Z", AuthorName: "bob", Description: "create", Commands: []string{"echo old1"}, Operation: "create"}}
	reg := &versionsFakeRegistry{items: []adapters.CommandSetSummary{{Name: "vset", Description: "Versions set"}}, full: full, versions: vers}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	m1, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = m1.(*TuiModel)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(*TuiModel)
	// versions should be loaded and visible
	if len(m.versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(m.versions))
	}
	view := m.View()
	if !strings.Contains(view, "Versions") {
		t.Fatalf("expected Versions header in view, got:\n%s", view)
	}
	// focus right and select second entry (older version)
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = m3.(*TuiModel)
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = m4.(*TuiModel)
	// initiate rollback (R)
	m5, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	m = m5.(*TuiModel)
	if !strings.Contains(m.detail, "Rollback 'vset'") {
		t.Fatalf("expected rollback prompt in detail, got:\n%s", m.detail)
	}
	// confirm
	m6, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = m6.(*TuiModel)
	if reg.lastAppliedVersion != 1 {
		t.Fatalf("expected applied version 1, got %d", reg.lastAppliedVersion)
	}
	if !strings.Contains(strings.Join(m.logs, "\n"), "rolled back") {
		t.Fatalf("expected rolled back log, got %v", m.logs)
	}
}

func TestEnterOnVersionsDoesNotChangeDetail(t *testing.T) {
	// prepare registry with two sets and versions only for the first
	full := adapters.CommandSetSummary{Name: "aset", Description: "A set", Commands: []string{"echo a"}}
	vers := []adapters.Version{{Version: 1, CreatedAt: "2026-01-11T00:00:00Z", AuthorName: "bob", Description: "create", Commands: []string{"echo old1"}, Operation: "create"}}
	reg := &versionsFakeRegistry{items: []adapters.CommandSetSummary{{Name: "aset", Description: "A set"}, {Name: "bset", Description: "B set"}}, full: full, versions: vers}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	m1, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = m1.(*TuiModel)
	// open detail for the first (aset)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(*TuiModel)
	if m.detailName != "aset" {
		t.Fatalf("expected detailName 'aset', got %q", m.detailName)
	}
	// move selection on the left pane down to bset (simulate user scrolling left pane)
	m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = m3.(*TuiModel)
	if si, ok := m.list.SelectedItem().(csItem); !ok || si.cs.Name != "bset" {
		t.Fatalf("expected left selection to be 'bset', got %v", si)
	}
	// focus right pane and interact with versions
	m4, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = m4.(*TuiModel)
	// ensure versions are present
	if len(m.versions) == 0 {
		t.Fatalf("expected versions to be loaded, got none")
	}
	// press Down inside versions to change selection and then press Enter
	m5, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = m5.(*TuiModel)
	m6, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m6.(*TuiModel)
	// detailName should remain the same (aset), not switch to bset
	if m.detailName != "aset" {
		t.Fatalf("expected detailName to remain 'aset' after Enter on versions, got %q", m.detailName)
	}
}

// versionsFakeRegistry supports listing and applying versions for tests
type versionsFakeRegistry struct {
	items              []adapters.CommandSetSummary
	full               adapters.CommandSetSummary
	versions           []adapters.Version
	lastAppliedVersion int
}

func (f *versionsFakeRegistry) ListCommandSets(_ context.Context) ([]adapters.CommandSetSummary, error) {
	return f.items, nil
}
func (f *versionsFakeRegistry) GetCommandSet(_ context.Context, _ string) (adapters.CommandSetSummary, error) {
	return f.full, nil
}
func (f *versionsFakeRegistry) SaveCommandSet(_ context.Context, _ adapters.CommandSetSummary) error {
	return nil
}
func (f *versionsFakeRegistry) DeleteCommandSet(_ context.Context, _ string) error { return nil }
func (f *versionsFakeRegistry) GetCommands(_ context.Context, _ string) ([]string, error) {
	return f.full.Commands, nil
}
func (f *versionsFakeRegistry) ReplaceCommands(_ context.Context, name string, cmds []string) error {
	f.full.Commands = append([]string{}, cmds...)
	return nil
}
func (f *versionsFakeRegistry) UpdateCommandSet(_ context.Context, oldName string, cs adapters.CommandSetSummary) error {
	f.full.Name = cs.Name
	f.full.Description = cs.Description
	return nil
}
func (f *versionsFakeRegistry) ListVersionsByName(_ context.Context, name string) ([]adapters.Version, error) {
	return append([]adapters.Version{}, f.versions...), nil
}
func (f *versionsFakeRegistry) ApplyVersionByName(_ context.Context, name string, versionNum int) error {
	f.lastAppliedVersion = versionNum
	// simulate applying by finding version and updating full.Commands
	for _, v := range f.versions {
		if v.Version == versionNum {
			f.full.Commands = append([]string{}, v.Commands...)
			// simulate recording a rollback version by prepending a new version entry
			f.versions = append([]adapters.Version{{Version: v.Version + 1, CreatedAt: "now", AuthorName: "system", Commands: f.full.Commands, Operation: "rollback"}}, f.versions...)
			return nil
		}
	}
	return fmt.Errorf("version not found")
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
func (f *fakeRegistry) UpdateCommandSet(_ context.Context, oldName string, cs adapters.CommandSetSummary) error {
	// update first matching item
	for i := range f.items {
		if f.items[i].Name == oldName {
			f.items[i].Name = cs.Name
			f.items[i].Description = cs.Description
			return nil
		}
	}
	return nil
}
func (f *fakeRegistry) ListVersionsByName(_ context.Context, _ string) ([]adapters.Version, error) {
	return nil, nil
}
func (f *fakeRegistry) ApplyVersionByName(_ context.Context, _ string, _ int) error { return nil }

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
func (f *detailFakeRegistry) ReplaceCommands(_ context.Context, _ string, _ []string) error {
	return nil
}
func (f *detailFakeRegistry) UpdateCommandSet(_ context.Context, oldName string, cs adapters.CommandSetSummary) error {
	if f.full.Name == oldName {
		f.full.Name = cs.Name
		f.full.Description = cs.Description
		f.full.Tags = append([]string{}, cs.Tags...)
	}
	return nil
}
func (f *detailFakeRegistry) ListVersionsByName(_ context.Context, _ string) ([]adapters.Version, error) {
	return nil, nil
}
func (f *detailFakeRegistry) ApplyVersionByName(_ context.Context, _ string, _ int) error { return nil }

// replaceFakeRegistry supports ReplaceCommands and records the last call
// for assertions in tests.
type replaceFakeRegistry struct {
	items        []adapters.CommandSetSummary
	full         adapters.CommandSetSummary
	lastName     string
	lastCommands []string
	lastDeleted  string
}

func (f *replaceFakeRegistry) ListCommandSets(_ context.Context) ([]adapters.CommandSetSummary, error) {
	return f.items, nil
}
func (f *replaceFakeRegistry) GetCommandSet(_ context.Context, _ string) (adapters.CommandSetSummary, error) {
	return f.full, nil
}
func (f *replaceFakeRegistry) SaveCommandSet(_ context.Context, _ adapters.CommandSetSummary) error {
	return nil
}
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
func (f *replaceFakeRegistry) GetCommands(_ context.Context, _ string) ([]string, error) {
	return f.full.Commands, nil
}
func (f *replaceFakeRegistry) ReplaceCommands(_ context.Context, name string, cmds []string) error {
	f.lastName = name
	f.lastCommands = append([]string{}, cmds...)
	f.full.Commands = append([]string{}, cmds...)
	return nil
}

func (f *replaceFakeRegistry) UpdateCommandSet(_ context.Context, oldName string, cs adapters.CommandSetSummary) error {
	f.lastName = cs.Name
	// update items list if name changed
	for i := range f.items {
		if f.items[i].Name == oldName {
			f.items[i].Name = cs.Name
			f.items[i].Description = cs.Description
		}
	}
	f.full.Name = cs.Name
	f.full.Description = cs.Description
	f.full.AuthorName = cs.AuthorName
	f.full.AuthorEmail = cs.AuthorEmail
	f.full.Tags = append([]string{}, cs.Tags...)
	return nil
}
func (f *replaceFakeRegistry) ListVersionsByName(_ context.Context, _ string) ([]adapters.Version, error) {
	return nil, nil
}
func (f *replaceFakeRegistry) ApplyVersionByName(_ context.Context, _ string, _ int) error {
	return nil
}
