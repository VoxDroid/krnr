package executor

import (
	"bytes"
	"context"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestExecuteEcho(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var out bytes.Buffer
	var errb bytes.Buffer
	e := &Executor{}
	if err := e.Execute(ctx, "echo hello", "", nil, &out, &errb); err != nil {
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
	if err := e.Execute(ctx, "exit 1", "", nil, &out, &errb); err == nil {
		t.Fatalf("expected error for failing command")
	}
}

func TestDryRun(t *testing.T) {
	ctx := context.Background()
	var out bytes.Buffer
	var errb bytes.Buffer
	e := &Executor{DryRun: true, Verbose: true}
	if err := e.Execute(ctx, "echo hi", "", nil, &out, &errb); err != nil {
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

func TestUnescapeWriter_PreservesANSI(t *testing.T) {
	// Test that ANSI escape sequences pass through unmodified (not processed by unescapeWriter).
	// This is critical for tools like fastfetch, htop, etc. that use ANSI codes.
	var buf bytes.Buffer
	uw := &unescapeWriter{w: &buf}
	// Simulate colored output with ANSI escape code: ESC[1;32m GREEN ESC[0m
	ansiGreen := "\x1b[1;32mGREEN\x1b[0m\n"
	in := []byte(ansiGreen)
	if n, err := uw.Write(in); err != nil {
		t.Fatalf("write failed: %v", err)
	} else if n != len(in) {
		t.Fatalf("expected write len %d, got %d", len(in), n)
	}
	// Verify ANSI sequence passed through unchanged
	if buf.String() != ansiGreen {
		t.Fatalf("expected ANSI to pass through unchanged, got: %q", buf.String())
	}
}

func TestUnescapeWriter_ANSIWithQuotes(t *testing.T) {
	// Test that output containing both ANSI codes and quotes preserves both.
	var buf bytes.Buffer
	uw := &unescapeWriter{w: &buf}
	// Mix ANSI codes with quoted text: "message" with color codes
	mixedOutput := "\x1b[1;36m\"ANSI with quotes\"\x1b[0m\n"
	in := []byte(mixedOutput)
	if n, err := uw.Write(in); err != nil {
		t.Fatalf("write failed: %v", err)
	} else if n != len(in) {
		t.Fatalf("expected write len %d, got %d", len(in), n)
	}
	// ANSI-containing output must pass through unchanged (quotes should NOT be stripped)
	if buf.String() != mixedOutput {
		t.Fatalf("expected ANSI+quotes to pass through unchanged, got: %q", buf.String())
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

func TestExecute_Exit1ButHasStdout_ShouldSucceed(t *testing.T) {
	// Use a shell that supports `exit` and `echo`; if 'sh' isn't available on
	// Windows CI skip the test.
	e := &Executor{}
	if runtime.GOOS == "windows" {
		if p, err := exec.LookPath("sh"); err == nil {
			e.Shell = p
		} else {
			t.Skip("sh not available, skipping test")
		}
	}

	var out bytes.Buffer
	var errb bytes.Buffer
	if err := e.Execute(context.Background(), "echo hello; exit 1", "", nil, &out, &errb); err != nil {
		t.Fatalf("expected nil error for exit 1 with stdout, got: %v", err)
	}
	if !strings.Contains(out.String(), "hello") {
		t.Fatalf("expected 'hello' in stdout, got: %q", out.String())
	}
}

func TestExecute_Exit2WithStdout_ShouldReturnError(t *testing.T) {
	// Confirm that non-1 exit codes remain fatal even if stdout is present.
	e := &Executor{}
	if runtime.GOOS == "windows" {
		if p, err := exec.LookPath("sh"); err == nil {
			e.Shell = p
		} else {
			t.Skip("sh not available, skipping test")
		}
	}

	var out bytes.Buffer
	var errb bytes.Buffer
	if err := e.Execute(context.Background(), "echo bye; exit 2", "", nil, &out, &errb); err == nil {
		t.Fatalf("expected error for exit 2 even with stdout")
	}
}

func TestExecute_FindstrPipeline_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only test")
	}
	ctx := context.Background()
	e := &Executor{}
	var out, errb bytes.Buffer
	// left side runs two echoes inside a cmd /C so we can produce two lines
	command := "cmd /C \"echo OS Name & echo OS Version\" | findstr /C:\"OS Name\" /C:\"OS Version\""
	if err := e.Execute(ctx, command, "", nil, &out, &errb); err != nil {
		t.Fatalf("Execute pipeline failed: %v, stderr: %q", err, errb.String())
	}
	normalized := strings.ReplaceAll(strings.TrimSpace(out.String()), "\r\n", "\n")
	if !strings.Contains(normalized, "OS Name") || !strings.Contains(normalized, "OS Version") {
		t.Fatalf("expected findstr to match both lines, got: %q", normalized)
	}
}
