package adapters

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/VoxDroid/krnr/internal/tui/sanitize"
)

// fakeRunner writes provided lines to stdout and returns
type fakeRunner struct{ lines []string }

func (f *fakeRunner) Execute(_ context.Context, _ string, _ string, stdout io.Writer, _ io.Writer) error {
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
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	var got []string
	for ev := range h.Events() {
		if ev.Err != nil {
			t.Fatalf("event error: %v", ev.Err)
		}
		// skip the announce line that indicates the command
		if len(ev.Line) > 3 && ev.Line[:3] == "-> " {
			continue
		}
		got = append(got, ev.Line)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 lines got %d", len(got))
	}
}

func TestSanitizeRunOutput_PreservesSGR(t *testing.T) {
	in := "\x1b[1;32mGREEN\x1b[0m"
	out := sanitize.RunOutput(in)
	if out != in {
		t.Fatalf("expected SGR to be preserved, got %q", out)
	}
}

func TestSanitizeRunOutput_RemoveAltScreenAndClear(t *testing.T) {
	in := "\x1b[?1049h\x1b[2JHello\x1b[?1049l"
	out := sanitize.RunOutput(in)
	if out != "Hello" {
		t.Fatalf("expected alt/clear removed, got %q", out)
	}
}

func TestSanitizeRunOutput_NormalizesCR(t *testing.T) {
	in := "line1\rline2\r\nline3"
	out := sanitize.RunOutput(in)
	if out != "line1\nline2\nline3" {
		t.Fatalf("expected CR normalized to LF, got %q", out)
	}
}
