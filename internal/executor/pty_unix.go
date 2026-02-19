//go:build !windows

package executor

import (
	"bytes"
	"io"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// isTerminal reports whether the given file descriptor refers to a terminal.
// It is a package-level variable so unit tests can override it to simulate
// terminal conditions without requiring a real TTY.
var isTerminal = func(fd uintptr) bool {
	return term.IsTerminal(int(fd))
}

// makeRaw/restoreTerminal wrap terminal mode changes so tests can override
// them. Default behavior: save the current terminal state and *disable local
// echo only* (do not flip other flags such as OPOST) so the host terminal
// does not display typed passwords but output post-processing stays intact.
// Tests can still override these hooks.
var makeRaw = func(fd int) (*term.State, error) {
	oldState, err := term.GetState(fd)
	if err != nil {
		return nil, err
	}
	// disable local echo only
	if err := setEcho(fd, false); err != nil {
		return nil, err
	}
	return oldState, nil
}
var restoreTerminal = func(fd int, state *term.State) error { return term.Restore(fd, state) }

// ptyStarter encapsulates starting a command with a hybrid PTY setup.
// The child's stdin and controlling terminal use a PTY so programs like
// sudo that open /dev/tty work correctly. Stdout and stderr remain as
// pipes (via io.MultiWriter) so programs like fastfetch detect pipe mode
// and produce simple, viewport-friendly output.
//
// It is a package-level variable so unit tests can override it.
var ptyStarter = func(cmd *exec.Cmd, stdin io.Reader, stdout, stderr io.Writer) (*bytes.Buffer, *bytes.Buffer, error) {
	ptmx, pts, err := pty.Open()
	if err != nil {
		return &bytes.Buffer{}, &bytes.Buffer{}, err
	}

	// Child's stdin is the PTY slave (terminal). Stdout/stderr are
	// streamed to caller's writers AND captured in buffers.
	cmd.Stdin = pts
	var bout, berr bytes.Buffer
	cmd.Stdout = io.MultiWriter(&bout, stdout)
	if stderr == stdout {
		cmd.Stderr = cmd.Stdout
	} else {
		cmd.Stderr = io.MultiWriter(&berr, stderr)
	}

	// Make the PTY slave the child's controlling terminal so /dev/tty
	// refers to our PTY, not the real host terminal.
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true
	cmd.SysProcAttr.Setctty = true

	if err := cmd.Start(); err != nil {
		_ = pts.Close()
		_ = ptmx.Close()
		return &bytes.Buffer{}, &bytes.Buffer{}, err
	}
	_ = pts.Close() // child has its own copy; close ours

	// If the caller's stdin is a terminal, put the *caller* terminal into
	// raw mode while the child runs. This prevents the host terminal from
	// locally echoing keystrokes (so password entry remains hidden) while
	// we forward input into the child's PTY master.
	if f, ok := stdin.(interface{ Fd() uintptr }); ok {
		if isTerminal(f.Fd()) {
			if oldState, err := makeRaw(int(f.Fd())); err == nil {
				defer func() { _ = restoreTerminal(int(f.Fd()), oldState) }()
			}
		}
	}

	// Forward user input from the caller's reader into the PTY master
	// so interactive prompts (sudo password, etc.) receive keystrokes.
	go func() { _, _ = io.Copy(ptmx, stdin) }()

	// Read any output the child writes directly to /dev/tty (e.g.,
	// sudo's password prompt) from the PTY master and forward it to
	// the caller's stdout so it appears in the TUI viewport.
	go func() { _, _ = io.Copy(stdout, ptmx) }()

	err = cmd.Wait()
	_ = ptmx.Close()
	return &bout, &berr, err
}
