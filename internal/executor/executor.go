package executor

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
)

// Executor runs shell commands in an OS-aware way.
type Executor struct {
	DryRun  bool
	Verbose bool
	Shell   string // optional override (e.g., "pwsh")
}

// Runner is an interface for executing commands. It allows tests to inject
// fake implementations without running real shell commands.
type Runner interface {
	Execute(ctx context.Context, command string, cwd string, stdout io.Writer, stderr io.Writer) error
}

// New returns a Runner backed by the real Executor implementation.
func New(dry, verbose bool) Runner {
	return &Executor{DryRun: dry, Verbose: verbose}
}

// Execute runs the given command string. stdout and stderr are written to the
// provided writers. If cwd is non-empty, the command runs in that directory.
func (e *Executor) Execute(ctx context.Context, command string, cwd string, stdout io.Writer, stderr io.Writer) error {
	if e.DryRun {
		if e.Verbose {
			fmt.Fprintf(stdout, "dry-run: %s\n", command)
		}
		return nil
	}

	shell, args := shellInvocation(command, e.Shell)
	cmd := exec.CommandContext(ctx, shell, args...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}
	return nil
}

// shellInvocation returns the shell executable and arguments for the platform.
// Optional `override` lets callers request alternate shell (e.g., pwsh).
func shellInvocation(command string, overrideShell string) (string, []string) {
	if overrideShell != "" {
		// If caller requests `pwsh`, use PowerShell CLI variant; otherwise pass through
		if overrideShell == "pwsh" || overrideShell == "powershell" {
			return "pwsh", []string{"-Command", command}
		}
		// generic: use override as the shell command and -c to pass command
		return overrideShell, []string{"-c", command}
	}

	if runtime.GOOS == "windows" {
		// Use cmd.exe by default on Windows
		return "cmd", []string{"/C", command}
	}
	// Unix-like
	return "bash", []string{"-c", command}
}
