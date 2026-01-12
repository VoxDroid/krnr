package ui

import (
	"context"
	"fmt"
	"strings"
	"testing"

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
	if len(m.list.Items()) != 2 { t.Fatalf("expected 2 items got %d", len(m.list.Items())) }
}

func TestInitPopulatesPreview(t *testing.T) {
	reg := &detailFakeRegistry{items: []adapters.CommandSetSummary{{Name: "one", Description: "First"}}, full: adapters.CommandSetSummary{Name: "one", Description: "First", Commands: []string{"echo hi"}}}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	m.Init()()
	if m.vp.View() == "" { t.Fatalf("expected preview content after Init") }
	if !strings.Contains(m.vp.View(), "echo hi") { t.Fatalf("expected command in preview after Init") }
}

func abs(x int) int { if x < 0 { return -x } ; return x }

func TestDescriptionIndentAndCommandAlignment(t *testing.T) {
	// create a set with a multi-line description and two commands
	full := adapters.CommandSetSummary{
		Name: "with-params",
		Description: "Param demo",
		Commands: []string{"echo User: {{user}}", "echo Token: {{token}}"},
		AuthorName: "VoxDroid",
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
	between := view[posDesc+len("Description:"):posParam]
	if strings.TrimSpace(between) != "" { t.Fatalf("expected only whitespace between Description: and Param demo, got %q", between) }

	// also assert description is indented by the same base offset used for commands
	maxPrefix := 0
	for i := range full.Commands {
		p := fmt.Sprintf("%d) ", i+1)
		if l := len(p); l > maxPrefix { maxPrefix = l }
	}
	expectedDelta := 2 + maxPrefix + 1 // the formatter indents by 2 + maxPrefix + 1 spaces relative to the header
	// find header line and the following non-empty line
	lines := strings.Split(view, "\n")
	headIdx := -1
	for i, ln := range lines {
		if strings.Contains(ln, "Description:") { headIdx = i; break }
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
	if delta < expectedDelta { t.Fatalf("expected param to be indented at least %d spaces relative to header, got %d (view:\n%s)", expectedDelta, delta, view) }

	// ensure both command lines show the prefix and a small gap before the echoed text
	if !strings.Contains(view, "1)  echo") || !strings.Contains(view, "2)  echo") {
		t.Fatalf("expected both command lines to include '1)  echo' and '2)  echo', got:\n%s", view)
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
	if !m.showDetail { t.Fatalf("expected showDetail true") }
	if !strings.Contains(m.detail, "Dry-run preview") { t.Fatalf("expected dry-run preview in detail") }
	if !strings.Contains(m.detail, "echo hi") { t.Fatalf("expected command in detail") }
}

// minimal fakes for testing
type fakeRegistry struct{ items []adapters.CommandSetSummary }
func (f *fakeRegistry) ListCommandSets(_ context.Context) ([]adapters.CommandSetSummary, error) { return f.items, nil }
func (f *fakeRegistry) GetCommandSet(_ context.Context, _ string) (adapters.CommandSetSummary, error) { return adapters.CommandSetSummary{}, nil }
func (f *fakeRegistry) SaveCommandSet(_ context.Context, _ adapters.CommandSetSummary) error { return nil }
func (f *fakeRegistry) DeleteCommandSet(_ context.Context, _ string) error { return nil }
func (f *fakeRegistry) GetCommands(_ context.Context, _ string) ([]string, error) { return []string{"echo hello"}, nil }

type fakeExec struct{}
func (f *fakeExec) Run(_ context.Context, _ string, _ []string) (adapters.RunHandle, error) { return nil, nil }

// detailFakeRegistry returns a full CommandSet via GetCommandSet so the UI can
// render full details and dry-run preview in tests.
type detailFakeRegistry struct{ items []adapters.CommandSetSummary; full adapters.CommandSetSummary }
func (f *detailFakeRegistry) ListCommandSets(_ context.Context) ([]adapters.CommandSetSummary, error) { return f.items, nil }
func (f *detailFakeRegistry) GetCommandSet(_ context.Context, _ string) (adapters.CommandSetSummary, error) { return f.full, nil }
func (f *detailFakeRegistry) SaveCommandSet(_ context.Context, _ adapters.CommandSetSummary) error { return nil }
func (f *detailFakeRegistry) DeleteCommandSet(_ context.Context, _ string) error { return nil }
func (f *detailFakeRegistry) GetCommands(_ context.Context, _ string) ([]string, error) { return []string{"echo hello"}, nil }
