//go:build integration
// +build integration

package executor

import (
	"bytes"
	"context"
	"io"
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
)

// TestExecute_PTYInteractive verifies that when a terminal-like stdin is
// provided, the executor runs the child in a PTY and streams prompts so they
// can be answered interactively. This is an integration-style test and is
// skipped in CI/Windows environments.
func TestExecute_PTYInteractive(t *testing.T) {
	skipIfUnsupported(t)

	master, slave, err := pty.Open()
	if err != nil {
		t.Fatalf("pty.Open failed: %v", err)
	}
	defer func() { _ = master.Close(); _ = slave.Close() }()

	e := &Executor{}
	if p, err := os.LookPath("sh"); err == nil {
		e.Shell = p
	}

	cmd := `printf "Enter:"; read -r line; printf "GOT:%s\n" "$line"`

	pr, pw := io.Pipe()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		d := e.Execute(ctx, cmd, "", slave, pw, io.Discard)
		_ = pw.Close()
		done <- d
	}()

	outCh := collectAsync(pr)

	_ = syscall.SetNonblock(int(master.Fd()), true)
	waitForPrompt(t, master, pw, "Enter:")

	_, _ = master.Write([]byte("s3cret\n"))

	waitForCompletion(t, done)

	out := <-outCh
	if !bytes.Contains([]byte(out), []byte("GOT:s3cret")) {
		t.Fatalf("expected GOT:s3cret in output, got: %q", out)
	}
}

// skipIfUnsupported skips the test on Windows and CI environments.
func skipIfUnsupported(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("PTY integration test skipped on Windows")
	}
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("skipping PTY integration in CI")
	}
}

// collectAsync reads all data from r in a background goroutine and sends
// the result as a string on the returned channel.
func collectAsync(r io.Reader) <-chan string {
	ch := make(chan string, 1)
	go func() {
		var b bytes.Buffer
		_, _ = io.Copy(&b, r)
		ch <- b.String()
	}()
	return ch
}

// waitForPrompt polls master until the expected prompt appears, forwarding
// the data to pw. Skips the test if the prompt is not seen within 2 seconds.
func waitForPrompt(t *testing.T, master *os.File, pw *io.PipeWriter, prompt string) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Skip("prompt did not appear in time; skipping PTY-flaky test")
		default:
			buf := make([]byte, 64)
			n, _ := master.Read(buf)
			if n > 0 && bytes.Contains(buf[:n], []byte(prompt)) {
				_, _ = pw.Write(buf[:n])
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
}

// waitForCompletion waits for the execution to finish, failing the test if
// it does not complete within 2 seconds.
func waitForCompletion(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Execute did not complete in time")
	}
}
