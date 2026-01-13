package ui

import (
	"bufio"
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
	time.Sleep(150 * time.Millisecond)

	// read what is currently on the pty
	r := bufio.NewReader(p)
	var b strings.Builder
	// read available bytes (non-blocking with small timeout)
	if err := p.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
		// ignore deadline set errors (platform specifics)
		t.Logf("SetReadDeadline: %v", err)
	}
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			break
		}
		b.WriteString(line)
	}
	out := b.String()

	if !strings.Contains(out, "with-params") {
		t.Fatalf("expected command set name in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Description:") {
		t.Fatalf("expected Description: header in output, got:\n%s", out)
	}
	if !strings.Contains(out, "1)  echo") || !strings.Contains(out, "2)  echo") {
		t.Fatalf("expected aligned command prefixes in output, got:\n%s", out)
	}

	// quit
	_, _ = p.Write([]byte("q"))
	// allow program to exit cleanly
	time.Sleep(50 * time.Millisecond)
}
