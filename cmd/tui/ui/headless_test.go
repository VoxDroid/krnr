package ui

import (
	"context"
	"testing"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestRunStreamsLinesHeadless(t *testing.T) {
	// fakes
	fakeReg := &fakeRegistry{items: []adapters.CommandSetSummary{{Name: "one", Description: "First"}}}
	fakeExec := &fakeExecAdapter{lines: []string{"line1", "line2", "line3"}}

	ui := modelpkg.New(fakeReg, fakeExec, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)
	// init
	m.Init()()
	// ensure selected item exists
	if len(m.list.Items()) == 0 { t.Fatalf("no items") }

	// Start a run directly (avoid keybinding being swallowed by the list in tests).
	h, err := m.uiModel.Run(context.Background(), m.list.Items()[0].(csItem).cs.Name, nil)
	if err != nil { t.Fatalf("run start failed: %v", err) }
	ch := make(chan adapters.RunEvent)
	// ensure model runCh is set so Update continues reading
	m.runCh = ch
	go func() {
		for ev := range h.Events() { ch <- ev }
		close(ch)
	}()

	// execute the returned cmd loop until runDoneMsg
	cmd := readLoop(ch)
	var m1 tea.Model
	for i := 0; i < 10; i++ {
		msg := cmd()
		m1, cmd = m.Update(msg)
		m = m1.(*TuiModel)
		// stop when run finished
		if !m.runInProgress && m.runCh == nil { break }
	}

	if len(m.logs) != 3 { t.Fatalf("expected 3 log lines got %d", len(m.logs)) }
	if m.logs[0] != "line1" || m.logs[2] != "line3" { t.Fatalf("unexpected logs: %v", m.logs) }
}

// fake executor adapter that returns a FakeRunHandle from the model package
type fakeExecAdapter struct{ lines []string }

func (f *fakeExecAdapter) Run(_ context.Context, _ string, _ []string) (adapters.RunHandle, error) {
	// use modelpkg FakeRunHandle to generate a handle
	return modelpkg.FakeRunHandle(f.lines, 0), nil
}
