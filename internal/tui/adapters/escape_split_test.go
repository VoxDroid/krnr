package adapters

import (
	"context"
	"io"
	"testing"
)

// fakeRunnerSplit simulates writing a chunk that contains a split
// escape sequence across two writes.
type fakeRunnerSplit struct{}

func (f *fakeRunnerSplit) Execute(_ context.Context, _ string, _ string, _ io.Reader, stdout io.Writer, _ io.Writer) error {
	// Write 'Hello' and a partial escape (CSI prefix) without the final letter
	_, _ = stdout.Write([]byte("Hello\x1b[?104"))
	// then write the trailing bytes that complete the sequence and some text
	_, _ = stdout.Write([]byte("9hWorld\n"))
	return nil
}

func TestExecutorAdapter_SplitsEscapeAcrossChunks(t *testing.T) {
	ad := NewExecutorAdapter(&fakeRunnerSplit{})
	h, err := ad.Run(context.Background(), "", []string{"cmd"})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	var out []string
	for ev := range h.Events() {
		if ev.Err != nil {
			t.Fatalf("unexpected run error: %v", ev.Err)
		}
		if ev.Line == "" {
			continue
		}
		// skip the '-> cmd' announce line
		if len(ev.Line) >= 3 && ev.Line[:3] == "-> " {
			continue
		}
		out = append(out, ev.Line)
	}
	// Combined output should contain Hello and World and not include ESC
	joined := ""
	for _, l := range out {
		joined += l
	}
	if joined != "HelloWorld" {
		t.Fatalf("unexpected combined output: %q", joined)
	}
}

// fakeRunnerTrailingEsc simulates output that ends with an incomplete
// escape sequence, verifying the flush-on-EOF path emits the text
// portion instead of silently discarding it.
type fakeRunnerTrailingEsc struct{}

func (f *fakeRunnerTrailingEsc) Execute(_ context.Context, _ string, _ string, _ io.Reader, stdout io.Writer, _ io.Writer) error {
	// Write text followed by an incomplete escape at the very end of output
	_, _ = stdout.Write([]byte("Line1\nTrailingText\x1b[38"))
	return nil
}

func TestExecutorAdapter_FlushesEscBufOnEOF(t *testing.T) {
	ad := NewExecutorAdapter(&fakeRunnerTrailingEsc{})
	h, err := ad.Run(context.Background(), "", []string{"cmd"})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	var out []string
	for ev := range h.Events() {
		if ev.Err != nil {
			t.Fatalf("unexpected run error: %v", ev.Err)
		}
		if ev.Line == "" || (len(ev.Line) >= 3 && ev.Line[:3] == "-> ") {
			continue
		}
		out = append(out, ev.Line)
	}
	if len(out) < 2 {
		t.Fatalf("expected at least 2 lines, got %d: %v", len(out), out)
	}
	if out[0] != "Line1" {
		t.Fatalf("expected first line 'Line1', got %q", out[0])
	}
	// The trailing text must not be lost even though it was held in escBuf
	if out[1] != "TrailingText" {
		t.Fatalf("expected trailing text 'TrailingText' to be flushed, got %q", out[1])
	}
}
