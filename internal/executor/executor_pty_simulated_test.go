//go:build !windows

package executor

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"testing"

	"golang.org/x/term"
)

// fakeReader simulates an io.Reader that also exposes a file descriptor.
// We do not require the fd to be valid for the OS; tests override the
// package-level isTerminal function to treat this fd as terminal-like.
type fakeReader struct{ fd uintptr }

func (f *fakeReader) Read(_ []byte) (int, error) { return 0, io.EOF }
func (f *fakeReader) Fd() uintptr                { return f.fd }

func TestExecute_PTYSimulated(t *testing.T) {
	// Save/restore global hooks
	origIsTerminal := isTerminal
	origPtyStarter := ptyStarter
	defer func() { isTerminal = origIsTerminal; ptyStarter = origPtyStarter }()

	// Simulate terminal detection for fd=0xdead
	isTerminal = func(fd uintptr) bool { return fd == 0xdead }

	// Simulate hybrid PTY starter that writes a prompt to the provided stdout.
	// This mirrors the real hybrid starter: stdout/stderr are the caller's
	// writers (pipes from the adapter), not the PTY.
	ptyStarter = func(_ *exec.Cmd, _ io.Reader, stdout, _ io.Writer) (*bytes.Buffer, *bytes.Buffer, error) {
		var bout bytes.Buffer
		_, _ = io.WriteString(stdout, "Enter:")
		bout.WriteString("Enter:")
		return &bout, &bytes.Buffer{}, nil
	}

	ctx := context.Background()
	e := &Executor{}
	var out bytes.Buffer
	var errb bytes.Buffer
	stdin := &fakeReader{fd: 0xdead}
	if err := e.Execute(ctx, "true", "", stdin, &out, &errb); err != nil {
		t.Fatalf("expected Execute to succeed under simulated PTY: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("Enter:")) {
		t.Fatalf("expected prompt streamed to stdout, got: %q", out.String())
	}
}

func TestExecute_SetsHostTerminalRaw(t *testing.T) {
	// Ensure we call makeRaw/restoreTerminal when stdin is a terminal-like FD.
	origIsTerminal := isTerminal
	origMakeRaw := makeRaw
	origRestore := restoreTerminal
	defer func() { isTerminal = origIsTerminal; makeRaw = origMakeRaw; restoreTerminal = origRestore }()

	isTerminal = func(fd uintptr) bool { return fd == 0xdead }

	calledMake := false
	calledRestore := false
	makeRaw = func(_ int) (*term.State, error) {
		calledMake = true
		return &term.State{}, nil
	}
	restoreTerminal = func(_ int, _ *term.State) error {
		calledRestore = true
		return nil
	}

	ctx := context.Background()
	e := &Executor{}
	var out bytes.Buffer
	var errb bytes.Buffer
	stdin := &fakeReader{fd: 0xdead}
	if err := e.Execute(ctx, "true", "", stdin, &out, &errb); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !calledMake || !calledRestore {
		t.Fatalf("expected makeRaw and restoreTerminal to be called; got make=%v restore=%v", calledMake, calledRestore)
	}
}
