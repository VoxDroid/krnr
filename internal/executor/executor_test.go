package executor

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
	"runtime"
)

func TestExecuteEcho(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var out bytes.Buffer
	var errb bytes.Buffer
	e := &Executor{}
	if err := e.Execute(ctx, "echo hello", "", &out, &errb); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(out.String(), "hello") {
		t.Fatalf("expected 'hello' in stdout, got: %q", out.String())
	}
}

func TestExecuteFail(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var out bytes.Buffer
	var errb bytes.Buffer
	e := &Executor{}
	// 'exit 1' should return non-zero from shell
	if err := e.Execute(ctx, "exit 1", "", &out, &errb); err == nil {
		t.Fatalf("expected error for failing command")
	}
}

func TestDryRun(t *testing.T) {
	ctx := context.Background()
	var out bytes.Buffer
	var errb bytes.Buffer
	e := &Executor{DryRun: true, Verbose: true}
	if err := e.Execute(ctx, "echo hi", "", &out, &errb); err != nil {
		t.Fatalf("dry-run should not error: %v", err)
	}
	if !strings.Contains(out.String(), "dry-run:") {
		t.Fatalf("expected dry-run message, got: %q", out.String())
	}
}

func TestUnescapeWriter(t *testing.T) {
	var buf bytes.Buffer
	uw := &unescapeWriter{w: &buf}
	in := []byte("\\\"HELLO\\\"\n")
	if n, err := uw.Write(in); err != nil {
		t.Fatalf("write failed: %v", err)
	} else if n != len(in) {
		t.Fatalf("expected write len %d, got %d", len(in), n)
	}
	// Expect outer quotes to be stripped: HELLO\n
	if buf.String() != "HELLO\n" {
		t.Fatalf("expected HELLO\n, got: %q", buf.String())
	}
}

func TestShellInvocationOverride(t *testing.T) {
	// pwsh should use -Command
	shell, args := shellInvocation("echo hi", "pwsh")
	if shell != "pwsh" {
		t.Fatalf("expected pwsh shell, got: %s", shell)
	}
	if len(args) < 1 || args[0] != "-Command" {
		t.Fatalf("expected -Command arg for pwsh, got: %v", args)
	}

	// generic overrides (bash) should use -c
	shell, args = shellInvocation("echo hi", "bash")
	if shell != "bash" {
		t.Fatalf("expected bash shell, got: %s", shell)
	}
	if len(args) < 1 || args[0] != "-c" {
		t.Fatalf("expected -c arg for bash, got: %v", args)
	}
}

func TestShellInvocationPowershellMapping(t *testing.T) {
	// 'powershell' should map to the appropriate executable depending on OS.
	shell, _ := shellInvocation("echo hi", "powershell")
	if runtime.GOOS == "windows" {
		// On Windows we expect either 'powershell' or a full path containing 'powershell'
		if !strings.Contains(strings.ToLower(shell), "powershell") {
			t.Fatalf("expected powershell on Windows, got: %q", shell)
		}
	} else {
		// Non-Windows prefer 'pwsh'
		if !strings.Contains(strings.ToLower(shell), "pwsh") {
			t.Fatalf("expected pwsh on non-Windows, got: %q", shell)
		}
	}
}