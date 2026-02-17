package adapters

import (
	"context"
	"io"
	"os"
	"testing"
)

// fakeRunnerFD captures the stdin passed to Execute for inspection in tests.
type fakeRunnerFD struct {
	called bool
	stdin  interface {
		Fd() uintptr
	}
}

func (f *fakeRunnerFD) Execute(_ context.Context, _ string, _ string, stdin io.Reader, _ io.Writer, _ io.Writer) error {
	f.called = true
	if rf, ok := stdin.(interface{ Fd() uintptr }); ok {
		f.stdin = rf
	}
	return nil
}

func TestExecutorAdapter_PassesHostFDToRunner(t *testing.T) {
	orig := hostIsTerminal
	defer func() { hostIsTerminal = orig }()
	// simulate that the host stdin is a terminal
	hostIsTerminal = func(_ int) bool { return true }

	fr := &fakeRunnerFD{}
	ad := NewExecutorAdapter(fr)
	h, err := ad.Run(context.Background(), "", []string{"echo hi"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for ev := range h.Events() {
		_ = ev
	}
	if !fr.called {
		t.Fatalf("expected fake runner to be called")
	}
	if fr.stdin == nil {
		t.Fatalf("expected stdin to implement Fd(), got nil")
	}
	if fr.stdin.Fd() != os.Stdin.Fd() {
		t.Fatalf("expected stdin Fd to be os.Stdin.Fd(), got %d vs %d", fr.stdin.Fd(), os.Stdin.Fd())
	}
}

func TestExecutorAdapter_NoPTYWhenNotTerminal(t *testing.T) {
	orig := hostIsTerminal
	defer func() { hostIsTerminal = orig }()
	// simulate non-terminal host
	hostIsTerminal = func(_ int) bool { return false }

	fr := &fakeRunnerFD{}
	ad := NewExecutorAdapter(fr)
	h, err := ad.Run(context.Background(), "", []string{"echo hi"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for ev := range h.Events() {
		_ = ev
	}
	if !fr.called {
		t.Fatalf("expected runner to be called")
	}
	if fr.stdin != nil {
		t.Fatalf("expected stdin to NOT have Fd() when host is not terminal")
	}
}
