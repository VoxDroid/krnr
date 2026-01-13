package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/creack/pty"
)

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

	// create and start the program configured to use the pty
	prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithInput(tty), tea.WithOutput(tty))
	go func() {
		_, _ = prog.Run()
	}()

	// give the program some time to initialize and render
	// Some CI runners take longer; use a slightly larger initial delay.
	time.Sleep(500 * time.Millisecond)

	// read what is currently on the pty using a goroutine so a slow CI
	// environment doesn't block the test indefinitely.
	done := make(chan string, 1)
	go func() {
		var b strings.Builder
		buf := make([]byte, 1024)
		end := time.Now().Add(5 * time.Second)
		for {
			// overall timeout for the goroutine
			if time.Now().After(end) {
				break
			}
			// set a short read deadline so Read doesn't block forever
			if err := p.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
				// log and continue; deadline may be unsupported on some platforms
				t.Logf("SetReadDeadline: %v", err)
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
				// if timeout, try again; otherwise stop
				if ne, ok := err.(interface{ Timeout() bool }); ok && ne.Timeout() {
					continue
				}
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
	case <-time.After(5 * time.Second):
		// attempt to capture any partial output for diagnostics and quit
		var diagBuf [4096]byte
		// send quit to the program first to encourage it to flush and exit
		_, _ = p.Write([]byte("q"))
		// set a short read deadline so this diagnostic read cannot block indefinitely
		if err := p.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
			t.Logf("SetReadDeadline (diag): %v", err)
		}
		n, _ := p.Read(diagBuf[:])
		outPartial := string(diagBuf[:n])
		t.Fatalf("pty output did not appear in time; partial output:\n%s", outPartial)
	}

	// quit (if not already done)
	_, _ = p.Write([]byte("q"))
	// allow program to exit cleanly
	time.Sleep(50 * time.Millisecond)
}
