package ui

import (
	"context"
	"testing"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestRunStreamsSanitizesControlSequences(t *testing.T) {
	// Test sanitizer preserves colors and strips destructive control sequences
	fakeReg := &fakeRegistry{items: []adapters.CommandSetSummary{{Name: "one", Description: "First"}}}
	fakeExecWithANSI := &fakeExecAdapter{lines: []string{"\x1b[?1049h\x1b[2JHello\x1b[?1049l", "\x1b[1;32mGREEN\x1b[0m"}}
	ui2 := modelpkg.New(fakeReg, fakeExecWithANSI, nil, nil)
	_ = ui2.RefreshList(context.Background())
	m2 := NewModel(ui2)
	m2.Init()()

	h2, err := m2.uiModel.Run(context.Background(), m2.list.Items()[0].(csItem).cs.Name, nil)
	if err != nil {
		t.Fatalf("run start failed: %v", err)
	}
	ch2 := make(chan adapters.RunEvent)
	m2.runCh = ch2
	go func() {
		for ev := range h2.Events() {
			ch2 <- ev
		}
		close(ch2)
	}()
	cmd2 := readLoop(ch2)
	var m3 tea.Model
	for i := 0; i < 10; i++ {
		msg := cmd2()
		m3, cmd2 = m2.Update(msg)
		m2 = m3.(*TuiModel)
		if !m2.runInProgress && m2.runCh == nil {
			break
		}
	}
	// sanitized output should show 'Hello' and color SGR preserved for GREEN
	if len(m2.logs) != 2 {
		t.Fatalf("expected 2 log lines got %d", len(m2.logs))
	}
	if m2.logs[0] != "Hello" {
		t.Fatalf("expected first log to be Hello, got %q", m2.logs[0])
	}
	if m2.logs[1] != "\x1b[1;32mGREEN\x1b[0m" {
		t.Fatalf("expected color preserved, got %q", m2.logs[1])
	}
}

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
	if len(m.list.Items()) == 0 {
		t.Fatalf("no items")
	}

	// Start a run directly (avoid keybinding being swallowed by the list in tests).
	h, err := m.uiModel.Run(context.Background(), m.list.Items()[0].(csItem).cs.Name, nil)
	if err != nil {
		t.Fatalf("run start failed: %v", err)
	}
	ch := make(chan adapters.RunEvent)
	// ensure model runCh is set so Update continues reading
	m.runCh = ch
	go func() {
		for ev := range h.Events() {
			ch <- ev
		}
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
		if !m.runInProgress && m.runCh == nil {
			break
		}
	}

	if len(m.logs) != 3 {
		t.Fatalf("expected 3 log lines got %d", len(m.logs))
	}
	if m.logs[0] != "line1" || m.logs[2] != "line3" {
		t.Fatalf("unexpected logs: %v", m.logs)
	}
}

// fake executor adapter that returns a FakeRunHandle from the model package
type fakeExecAdapter struct{ lines []string }

func (f *fakeExecAdapter) Run(_ context.Context, _ string, _ []string) (adapters.RunHandle, error) {
	// use modelpkg FakeRunHandle to generate a handle
	return modelpkg.FakeRunHandle(f.lines, 0), nil
}
