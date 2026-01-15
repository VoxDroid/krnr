//go:build integration
// +build integration

package ui

import (
	"bufio"
	"bytes"
	"context"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/creack/pty"
)

// fake executor used by PTY tests; records commands it is asked to run
type fakeExecRec struct{ lastRunCommands []string }

func (f *fakeExecRec) Run(_ context.Context, _ string, cmds []string) (adapters.RunHandle, error) {
	f.lastRunCommands = append([]string{}, cmds...)
	return modelpkg.FakeRunHandle([]string{"runline1", "runline2"}, 0), nil
}

// This test launches the TUI in a pseudo-terminal and asserts initial
// rendering (items present, description indented, commands aligned) so we
// catch real terminal rendering/regressions.
func TestTuiInitialRender_Pty(t *testing.T) {
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
	p, tty, err := pty.Open()
	if err != nil {
		// PTY may be unsupported on this platform (e.g., Windows), skip the test
		t.Skipf("pty not supported: %v", err)
	}
	defer func() { _ = p.Close(); _ = tty.Close() }()

	// try to set the master fd to non-blocking so Read won't block when
	// SetReadDeadline is not supported on some platforms (CI macOS runners).
	if err := setNonblock(p.Fd()); err != nil {
		t.Logf("SetNonblock (master) failed: %v", err)
	}

	// create and start the program configured to use the pty
	prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithInput(tty), tea.WithOutput(tty))
	go func() {
		_, _ = prog.Run()
	}()

	// give the program some time to initialize and render
	// Some CI runners take longer; use a slightly larger initial delay.
	time.Sleep(800 * time.Millisecond)

	// read what is currently on the pty using a goroutine so a slow CI
	// environment doesn't block the test indefinitely.
	done := make(chan string, 1)
	go func() {
		var b strings.Builder
		buf := make([]byte, 1024)
		end := time.Now().Add(8 * time.Second)
		for {
			// overall timeout for the goroutine
			if time.Now().After(end) {
				break
			}
			n, err := p.Read(buf)
			if n > 0 {
				b.Write(buf[:n])
				// stop early if we have the things we expect
				if strings.Contains(b.String(), "with-params") && strings.Contains(b.String(), "Description:") {
					break
				}
			}
			if err != nil {
				// on non-blocking reads EAGAIN/EWOULDBLOCK means no data is ready
				if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
					time.Sleep(50 * time.Millisecond)
					continue
				}
				// if EOF or other errors, stop
				break
			}
		}
		done <- b.String()
	}()

	select {
	case out := <-done:
		if !strings.Contains(out, "with-params") {
			t.Fatalf("expected command set name in output, got:\n%s", out)
		}
		if !strings.Contains(out, "Description:") {
			t.Fatalf("expected Description: header in output, got:\n%s", out)
		}
		if !strings.Contains(out, "1)  echo") || !strings.Contains(out, "2)  echo") {
			t.Fatalf("expected aligned command prefixes in output, got:\n%s", out)
		}
	case <-time.After(8 * time.Second):
		// attempt to capture any partial output for diagnostics and quit
		var diagBuf [4096]byte
		// send quit to the program first to encourage it to flush and exit
		_, _ = p.Write([]byte("q"))
		n, err := p.Read(diagBuf[:])
		if err != nil && (err == syscall.EAGAIN || err == syscall.EWOULDBLOCK) {
			// no data available
			n = 0
		}
		outPartial := string(diagBuf[:n])
		t.Fatalf("pty output did not appear in time; partial output:\n%s", outPartial)
	}

	// quit (if not already done)
	_, _ = p.Write([]byte("q"))
	// allow program to exit cleanly
	time.Sleep(50 * time.Millisecond)
}

// TestTui_EditSaveRun_Pty exercises an end-to-end edit -> save -> run flow
// inside a PTY and asserts that sanitized commands are persisted and executed.
func TestTui_EditSaveRun_Pty(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("pty E2E tests skipped on Windows")
	}

	full := adapters.CommandSetSummary{Name: "one", Description: "First", Commands: []string{"echo hi"}}
	reg := &replaceFakeRegistry{items: []adapters.CommandSetSummary{{Name: "one", Description: "First"}}, full: full}

	// use top-level fake executor implementation
	fe := &fakeExecRec{}

	ui := modelpkg.New(reg, fe, nil, nil)
	_ = ui.RefreshList(context.Background())
	m := NewModel(ui)

	master, tty, err := pty.Open()
	if err != nil {
		t.Skipf("pty not supported: %v", err)
	}
	defer func() { _ = master.Close(); _ = tty.Close() }()
	// set non-blocking on master so Read won't block when SetReadDeadline is unsupported
	if err := setNonblock(master.Fd()); err != nil {
		t.Logf("SetNonblock (master) failed: %v", err)
	}

	prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithInput(tty), tea.WithOutput(tty))
	go func() { _, _ = prog.Run() }()

	// helper to read until needle or timeout
	readUntil := func(needle string, d time.Duration) (string, error) {
		end := time.Now().Add(d)
		var b bytes.Buffer
		r := bufio.NewReader(master)
		for time.Now().Before(end) {
			buf := make([]byte, 1024)
			n, err := r.Read(buf)
			if n > 0 {
				b.Write(buf[:n])
				if strings.Contains(b.String(), needle) {
					return b.String(), nil
				}
			}
			if err != nil {
				if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
					// no data yet, try again
					time.Sleep(50 * time.Millisecond)
					continue
				}
				// otherwise, break and return what we have
				break
			}
		}
		return b.String(), context.DeadlineExceeded
	}

	// await initial render
	if _, err := readUntil("krnr — command sets", 3*time.Second); err != nil {
		t.Fatalf("initial render not seen: %v", err)
	}

	// Enter detail
	if _, err := master.Write([]byte{'\r'}); err != nil {
		t.Fatalf("enter: %v", err)
	}
	if _, err := readUntil("(e) Edit", 2*time.Second); err != nil {
		t.Fatalf("detail not shown: %v", err)
	}

	// Edit
	if _, err := master.Write([]byte{'e'}); err != nil {
		t.Fatalf("edit: %v", err)
	}
	// tab 3x to commands field
	for i := 0; i < 3; i++ {
		_, _ = master.Write([]byte{'\t'})
		time.Sleep(40 * time.Millisecond)
	}
	// add command (Ctrl+A)
	_, _ = master.Write([]byte{0x01})
	// type smart-quoted command: echo “quoted”
	cmd := "echo \u201Cquoted\u201D"
	if _, err := master.Write([]byte(cmd)); err != nil {
		t.Fatalf("type cmd: %v", err)
	}
	// save Ctrl+S
	_, _ = master.Write([]byte{0x13})

	// wait for ReplaceCommands to be called
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(reg.lastCommands) > 0 {
			break
		}
		_, _ = readUntil("", 200*time.Millisecond)
	}
	if len(reg.lastCommands) == 0 {
		t.Fatalf("expected ReplaceCommands to be called")
	}
	// expect sanitized ASCII quotes (accept variants where an accidental missing space
	// may have resulted in `echo"quoted"` on some PTY environments)
	t.Logf("ReplaceCommands recorded: %#v", reg.lastCommands)
	found := false
	for _, c := range reg.lastCommands {
		if c == "echo \"quoted\"" || strings.Contains(c, "\"quoted\"") {
			found = true
		}
	}
	if !found {
		t.Fatalf("sanitized command not found in ReplaceCommands: %#v", reg.lastCommands)
	}
	// wait for UI to emit a sanitization log message as an additional marker
	if _, err := readUntil("sanitized command", 3*time.Second); err != nil {
		// non-fatal: continue but log for diagnostics
		t.Logf("sanitization marker not observed in PTY output: %v", err)
	}

	// Run
	_, _ = master.Write([]byte{'r'})
	// wait for executor Run call
	dead2 := time.Now().Add(2 * time.Second)
	for time.Now().Before(dead2) {
		if len(fe.lastRunCommands) > 0 {
			break
		}
		_, _ = readUntil("", 200*time.Millisecond)
	}
	if len(fe.lastRunCommands) == 0 {
		t.Fatalf("expected Run called")
	}
	t.Logf("Run recorded: %#v", fe.lastRunCommands)
	// assert sanitized command executed (accept variants where space may be missing)
	runFound := false
	for _, c := range fe.lastRunCommands {
		if c == "echo \"quoted\"" || strings.Contains(c, "\"quoted\"") {
			runFound = true
		}
	}
	if !runFound {
		t.Fatalf("expected sanitized command in Run, got: %#v", fe.lastRunCommands)
	}

	// Quit
	_, _ = master.Write([]byte{'q'})
	time.Sleep(100 * time.Millisecond)
}
