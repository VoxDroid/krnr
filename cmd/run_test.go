package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/executor"
	"github.com/VoxDroid/krnr/internal/registry"
)

// fakeRunner implements the executor.Runner interface for tests.
type fakeRunner struct {
	lastCmd      string
	stderrWriter io.Writer
}

func (f *fakeRunner) Execute(_ context.Context, command, _ string, _ io.Reader, stdout io.Writer, stderr io.Writer) error {
	f.lastCmd = command
	f.stderrWriter = stderr
	if stdout != nil {
		_, _ = fmt.Fprintln(stdout, "cmd output")
	}
	if stderr != nil {
		_, _ = fmt.Fprintln(stderr, "cmd error")
	}
	return nil
}

func setupTempDB(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	_ = os.Setenv("KRNR_HOME", d)
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
		_, _ = io.Copy(&buf, rOut)
		outC <- buf.String()
	}()
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rErr)
		errC <- buf.String()
	}()

	f()

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = oldOut
	os.Stderr = oldErr

	out := <-outC
	err := <-errC
	return out, err
}

func TestRun_DefaultPrints(t *testing.T) {
	setupTempDB(t)

	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := registry.NewRepository(dbConn)
	id, err := r.CreateCommandSet("hello", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}
	if _, err := r.AddCommand(id, 1, "echo hello"); err != nil {
		t.Fatalf("AddCommand: %v", err)
	}

	origFactory := execFactory
	defer func() { execFactory = origFactory }()

	fake := &fakeRunner{}
	execFactory = func(_, _ bool) executor.Runner {
		return fake
	}

	// default (do not suppress, do not show-stderr)
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
}

func TestRun_SuppressCommand(t *testing.T) {
	setupTempDB(t)

	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := registry.NewRepository(dbConn)
	id, err := r.CreateCommandSet("hello", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}
	if _, err := r.AddCommand(id, 1, "echo hello"); err != nil {
		t.Fatalf("AddCommand: %v", err)
	}

	origFactory := execFactory
	defer func() { execFactory = origFactory }()

	fake := &fakeRunner{}
	execFactory = func(_, _ bool) executor.Runner {
		return fake
	}

	// suppress command printing
	out, errOut := captureOutput(func() {
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
}

func TestRun_ShowStderr(t *testing.T) {
	setupTempDB(t)

	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := registry.NewRepository(dbConn)
	id, err := r.CreateCommandSet("hello", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}
	if _, err := r.AddCommand(id, 1, "echo hello"); err != nil {
		t.Fatalf("AddCommand: %v", err)
	}

	origFactory := execFactory
	defer func() { execFactory = origFactory }()

	fake := &fakeRunner{}
	execFactory = func(_, _ bool) executor.Runner {
		return fake
	}

	// show stderr
	// Ensure flag state is clean
	_ = runCmd.Flags().Set("suppress-command", "false")
	_ = runCmd.Flags().Set("show-stderr", "true")
	_, errOut := captureOutput(func() {
		rootCmd.SetArgs([]string{"run", "hello", "--show-stderr"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("run command failed: %v", err)
		}
	})
	if !bytes.Contains([]byte(errOut), []byte("cmd error")) {
		t.Fatalf("expected stderr output to include 'cmd error', got: %q", errOut)
	}
	// The fake runner should have received a non-discard stderr writer
	if fake.stderrWriter == io.Discard || fake.stderrWriter == nil {
		t.Fatalf("expected executor to receive a real stderr writer, got: %v", fake.stderrWriter)
	}
}

func TestRunDryRunAndVerbose(t *testing.T) {
	setupTempDB(t)

	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := registry.NewRepository(dbConn)
	id, err := r.CreateCommandSet("dry", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}
	if _, err := r.AddCommand(id, 1, "echo dry"); err != nil {
		t.Fatalf("AddCommand: %v", err)
	}

	origFactory := execFactory
	defer func() { execFactory = origFactory }()

	// Use real executor for dry-run behavior (it will print dry-run messages)
	execFactory = func(_, verbose bool) executor.Runner {
		return executor.New(true, verbose) // dry is always true here
	}

	// Ensure flags do not carry over from other tests
	_ = runCmd.Flags().Set("dry-run", "false")
	_ = runCmd.Flags().Set("verbose", "false")
	out, errOut := captureOutput(func() {
		rootCmd.SetArgs([]string{"run", "dry", "--dry-run", "--verbose"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("run command failed: %v", err)
		}
	})

	// Dry-run should produce a dry-run message and should not produce actual command output
	if !strings.Contains(out, "dry-run: echo dry") {
		t.Fatalf("expected dry-run message in stdout, got: %q", out)
	}
	if strings.Contains(out, "cmd output") {
		t.Fatalf("did not expect real command output during dry-run, got: %q", out)
	}
	if errOut != "" {
		t.Fatalf("expected no stderr for dry-run, got: %q", errOut)
	}
}

func TestRunConfirmBehavior(t *testing.T) {
	setupTempDB(t)

	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := registry.NewRepository(dbConn)
	id, err := r.CreateCommandSet("confirm", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}
	if _, err := r.AddCommand(id, 1, "echo confirm"); err != nil {
		t.Fatalf("AddCommand: %v", err)
	}

	origFactory := execFactory
	defer func() { execFactory = origFactory }()

	fake := &fakeRunner{}
	execFactory = func(_, _ bool) executor.Runner {
		return fake
	}

	// Case: user declines
	oldStdin := os.Stdin
	rR, rW, _ := os.Pipe()
	_, _ = rW.Write([]byte("n\n"))
	_ = rW.Close()
	os.Stdin = rR
	out, _ := captureOutput(func() {
		rootCmd.SetArgs([]string{"run", "confirm", "--confirm"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("run command failed: %v", err)
		}
	})
	_ = rR.Close()
	os.Stdin = oldStdin
	if !strings.Contains(out, "aborted") {
		t.Fatalf("expected 'aborted' when user replies n, got: %q", out)
	}
	if fake.lastCmd != "" {
		t.Fatalf("expected fake runner not to run when aborted, got lastCmd=%q", fake.lastCmd)
	}

	// Case: user accepts
	rR, rW, _ = os.Pipe()
	_, _ = rW.Write([]byte("y\n"))
	_ = rW.Close()
	os.Stdin = rR
	_, _ = captureOutput(func() {
		rootCmd.SetArgs([]string{"run", "confirm", "--confirm"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("run command failed: %v", err)
		}
	})
	_ = rR.Close()
	os.Stdin = oldStdin
	if fake.lastCmd != "echo confirm" {
		t.Fatalf("expected fake runner to be invoked with command, got: %q", fake.lastCmd)
	}
}

func TestRunForceBehavior(t *testing.T) {
	setupTempDB(t)

	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := registry.NewRepository(dbConn)
	id, err := r.CreateCommandSet("unsafe", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}
	if _, err := r.AddCommand(id, 1, "rm -rf /"); err != nil {
		t.Fatalf("AddCommand: %v", err)
	}

	origFactory := execFactory
	defer func() { execFactory = origFactory }()

	fake := &fakeRunner{}
	execFactory = func(_, _ bool) executor.Runner {
		return fake
	}

	// Without --force we expect an error
	_ = runCmd.Flags().Set("confirm", "false")
	rootCmd.SetArgs([]string{"run", "unsafe"})
	if err := rootCmd.Execute(); err == nil {
		t.Fatalf("expected error when running unsafe command without --force")
	} else if !strings.Contains(err.Error(), "refusing to run") {
		t.Fatalf("unexpected error message: %v", err)
	}

	// With --force it should run
	_ = runCmd.Flags().Set("confirm", "false")
	rootCmd.SetArgs([]string{"run", "unsafe", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected run to succeed with --force, got: %v", err)
	}
	if fake.lastCmd != "rm -rf /" {
		t.Fatalf("expected fake runner to be invoked with unsafe command, got: %q", fake.lastCmd)
	}
}

func TestRunShellFlagWiresExecutor(t *testing.T) {
	setupTempDB(t)

	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := registry.NewRepository(dbConn)
	id, err := r.CreateCommandSet("shell-test", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}
	if _, err := r.AddCommand(id, 1, "echo shell"); err != nil {
		t.Fatalf("AddCommand: %v", err)
	}

	origFactory := execFactory
	defer func() { execFactory = origFactory }()

	var captured *executor.Executor
	execFactory = func(dry, verbose bool) executor.Runner {
		e := &executor.Executor{DryRun: dry, Verbose: verbose}
		captured = e
		return e
	}

	// Ensure flags do not carry over
	_ = runCmd.Flags().Set("dry-run", "false")
	// Run with shell override and dry-run to avoid side effects
	captureOutput(func() {
		rootCmd.SetArgs([]string{"run", "shell-test", "--shell", "pwsh", "--dry-run"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("run command failed: %v", err)
		}
	})

	if captured == nil {
		t.Fatalf("expected executor to be captured")
	}
	if captured.Shell != "pwsh" {
		t.Fatalf("expected shell to be pwsh, got: %q", captured.Shell)
	}
}
