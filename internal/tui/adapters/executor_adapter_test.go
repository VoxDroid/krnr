package adapters

import (
	"context"
	"io"
	"testing"
	"time"
)

// fakeRunner writes provided lines to stdout and returns
type fakeRunner struct{ lines []string }

func (f *fakeRunner) Execute(ctx context.Context, command string, cwd string, stdout io.Writer, stderr io.Writer) error {
	for _, l := range f.lines {
		_, _ = io.WriteString(stdout, l+"\n")
		// slight delay to simulate streaming
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

func TestExecutorAdapter_RunStreamsOutput(t *testing.T) {
	r := &fakeRunner{lines: []string{"one", "two", "three"}}
	a := NewExecutorAdapter(r)
	// run a single command (fake runner will stream the 3 lines)
	h, err := a.Run(context.Background(), "set", []string{"run"})
	if err != nil { t.Fatalf("run failed: %v", err) }
	var got []string
	for ev := range h.Events() {
		if ev.Err != nil { t.Fatalf("event error: %v", ev.Err) }
		// skip the announce line that indicates the command
		if len(ev.Line) > 3 && ev.Line[:3] == "-> " { continue }
		got = append(got, ev.Line)
	}
	if len(got) != 3 { t.Fatalf("expected 3 lines got %d", len(got)) }
}
