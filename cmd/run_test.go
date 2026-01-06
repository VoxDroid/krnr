package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/executor"
	"github.com/VoxDroid/krnr/internal/registry"
)

// fakeRunner implements the executor.Runner interface for tests.
type fakeRunner struct {
	lastCmd     string
	stderrWriter io.Writer
}

func (f *fakeRunner) Execute(ctx context.Context, command, cwd string, stdout io.Writer, stderr io.Writer) error {
	f.lastCmd = command
	f.stderrWriter = stderr
	if stdout != nil {
		fmt.Fprintln(stdout, "cmd output")
	}
	if stderr != nil {
		fmt.Fprintln(stderr, "cmd error")
	}
	return nil
}

func setupTempDB(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	os.Setenv("KRNR_HOME", d)
	return d
}

func captureOutput(f func()) (string, string) {
	oldOut := os.Stdout
	oldErr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	outC := make(chan string)
	errC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, rOut)
		outC <- buf.String()
	}()
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, rErr)
		errC <- buf.String()
	}()

	f()

	wOut.Close()
	wErr.Close()
	os.Stdout = oldOut
	os.Stderr = oldErr

	out := <-outC
	err := <-errC
	return out, err
}

func TestRunSuppressAndStderrFlags(t *testing.T) {
	setupTempDB(t)

dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer dbConn.Close()

	r := registry.NewRepository(dbConn)
	id, err := r.CreateCommandSet("hello", nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}
	if _, err := r.AddCommand(id, 1, "echo hello"); err != nil {
		t.Fatalf("AddCommand: %v", err)
	}

	origFactory := execFactory
	defer func() { execFactory = origFactory }()

	fake := &fakeRunner{}
	execFactory = func(dry, verbose bool) executor.Runner {
		return fake
	}

	// Case 1: default (do not suppress, do not show-stderr)
	out, errOut := captureOutput(func() {
		rootCmd.SetArgs([]string{"run", "hello"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("run command failed: %v", err)
		}
	})
	if !bytes.Contains([]byte(out), []byte("-> echo hello")) {
		t.Fatalf("expected command printed in stdout, got: %q", out)
	}
	if !bytes.Contains([]byte(out), []byte("cmd output")) {
		t.Fatalf("expected command output in stdout, got: %q", out)
	}
	if errOut != "" {
		t.Fatalf("expected no stderr output, got: %q", errOut)
	}
	if fake.stderrWriter != io.Discard {
		t.Fatalf("expected executor to receive io.Discard for stderr, got: %v", fake.stderrWriter)
	}

	// Case 2: suppress command printing
	out, errOut = captureOutput(func() {
		rootCmd.SetArgs([]string{"run", "hello", "--suppress-command"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("run command failed: %v", err)
		}
	})
	if bytes.Contains([]byte(out), []byte("-> echo hello")) {
		t.Fatalf("did not expect command printed in stdout, got: %q", out)
	}
	if !bytes.Contains([]byte(out), []byte("cmd output")) {
		t.Fatalf("expected command output in stdout, got: %q", out)
	}
	if errOut != "" {
		t.Fatalf("expected no stderr output, got: %q", errOut)
	}

	// Case 3: show stderr
	out, errOut = captureOutput(func() {
		rootCmd.SetArgs([]string{"run", "hello", "--show-stderr"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("run command failed: %v", err)
		}
	})
	if !bytes.Contains([]byte(errOut), []byte("cmd error")) {
		t.Fatalf("expected stderr output to include 'cmd error', got: %q", errOut)
	}
	// The fake runner should have received os.Stderr as writer
	if fake.stderrWriter != os.Stderr {
		t.Fatalf("expected executor to receive os.Stderr for stderr, got: %v", fake.stderrWriter)
	}
}
