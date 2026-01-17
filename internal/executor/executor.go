// Package executor provides command execution functionality.
package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"

	"github.com/kballard/go-shellquote"
)

// Executor runs shell commands in an OS-aware way.
type Executor struct {
	DryRun  bool
	Verbose bool
	Shell   string // optional override (e.g., "pwsh")
}

// unescapeWriter wraps an io.Writer and normalizes output produced by some
// shells on Windows which can emit backslash-escaped quotes like \"HELLO\".
// It will:
//   - unescape `\"` -> `"`
//   - if the entire line is wrapped in quotes ("..."), strip the outer quotes
//     so `"HELLO"\n` becomes `HELLO\n` for a cleaner UX.
//
// This is conservative and applies only to simple cases where the whole output
// line is a quoted string.
type unescapeWriter struct {
	w io.Writer
}

func (u *unescapeWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	s := string(p)
	// Unescape backslash-escaped quotes
	s = strings.ReplaceAll(s, "\\\"", "\"")
	// Normalize newlines for inspection
	trimmed := strings.TrimRight(s, "\r\n")
	// If the whole trimmed line is wrapped in quotes, strip them
	if len(trimmed) >= 2 && strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"") {
		body := trimmed[1 : len(trimmed)-1]
		// re-append the original newline suffix
		suffix := s[len(trimmed):]
		s = body + suffix
	}
	if _, err := u.w.Write([]byte(s)); err != nil {
		return 0, err
	}
	return len(p), nil
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
// sanitizeCommand normalizes common unicode characters that often get
// inserted by editors (e.g., smart quotes, NBSP, zero-width spaces) and
// converts them to their ASCII equivalents where sensible.
func sanitizeCommand(s string) string {
	r := strings.NewReplacer(
		"\u2018", "'", // left single quote
		"\u2019", "'", // right single quote
		"\u201C", "\"", // left double quote
		"\u201D", "\"", // right double quote
		"\u00A0", " ", // NO-BREAK SPACE
		"\u200B", "", // zero width space
		"\u200E", "", // left-to-right mark
		"\u200F", "", // right-to-left mark
	)
	rp := r.Replace(s)
	// Remove embedded NUL (\x00) and other invisible control runes that are
	// commonly inserted by text editors in some environments. Keep tabs and
	// normal printable characters; drop NULs to be forgiving for TUI edits.
	return strings.Map(func(r rune) rune {
		if r == 0 {
			return -1
		}
		return r
	}, rp)
}

// Execute runs the provided command string using an OS-appropriate shell
// invocation (e.g., `bash -c` on Unix, `cmd /C` on Windows). It sanitizes
// the command, validates it for illegal characters or newlines, and then
// executes it writing stdout/stderr to the provided writers.
func (e *Executor) Execute(ctx context.Context, command string, cwd string, stdout io.Writer, stderr io.Writer) error {
	// validate and sanitize command
	var err error
	command, err = validateAndSanitize(command)
	if err != nil {
		return err
	}

	// Handle dry-run early
	if handled := e.handleDryRunIfNeeded(command, stdout); handled {
		return nil
	}

	// On Windows try a specialized handler for `| findstr ...` pipelines
	// to avoid cmd.exe's quoting pitfalls. If it succeeds, we're done.
	if runtime.GOOS == "windows" {
		if tryHandleWindowsFindstr(ctx, command, cwd, stdout, stderr) {
			return nil
		}
	}

	shell, args := shellInvocation(command, e.Shell)
	if err := validateShellAndArgs(shell, args); err != nil {
		return err
	}

	bout, berr, err := runShellCommand(ctx, shell, args, cwd)

	writeOutputs(bout, berr, stdout, stderr)

	if err != nil {
		return checkExecutionError(err, bout, berr, shell, args)
	}
	return nil
}

// shellInvocation returns the shell executable and arguments for the platform.
// Optional `override` lets callers request alternate shell (e.g., pwsh).
// splitArgs splits a command string into tokens respecting single and double
// quotes. It removes the surrounding quotes from quoted tokens (so
// `/C:\"OS Name\"` becomes `/C:OS Name` as a single token).
func splitArgs(s string) []string {
	// Prefer a robust third-party splitter that understands quoted tokens.
	if toks, err := shellquote.Split(s); err == nil {
		return toks
	}
	// Fall back to simple whitespace splitting if the splitter fails.
	return strings.Fields(s)
}

// handleWindowsFindstrPipeline detects simple pipelines of the form
// `<left> | findstr <args...>` and executes them without invoking the
// shell, piping stdout from the left command into the findstr process.
// This avoids cmd.exe's tricky quoting behavior for /C:"..." patterns.
func handleWindowsFindstrPipeline(ctx context.Context, command string, cwd string, stdout io.Writer, stderr io.Writer) error {
	leftTokens, findstrExe, findstrArgs, err := parseFindstrPipeline(command)
	if err != nil {
		return err
	}
	return runFindstrPipeline(ctx, leftTokens, findstrExe, findstrArgs, cwd, stdout, stderr)
}

func parseFindstrPipeline(command string) ([]string, string, []string, error) {
	parts := strings.SplitN(command, "|", 2)
	if len(parts) != 2 {
		return nil, "", nil, fmt.Errorf("not a pipeline")
	}
	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])
	if len(right) < 7 || strings.ToLower(right[:7]) != "findstr" {
		return nil, "", nil, fmt.Errorf("not a findstr pipeline")
	}
	leftTokens := splitArgs(left)
	rightTokens := splitArgs(right)
	if len(leftTokens) == 0 || len(rightTokens) == 0 {
		return nil, "", nil, fmt.Errorf("invalid pipeline tokens")
	}
	findstrArgs := normalizeFindstrArgs(rightTokens[1:])
	return leftTokens, rightTokens[0], findstrArgs, nil
}

func normalizeFindstrArgs(tokens []string) []string {
	var out []string
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		if strings.HasPrefix(strings.ToUpper(t), "/C:") {
			arg := t
			if arg == "/C:" && i+1 < len(tokens) {
				i++
				arg = arg + tokens[i]
			} else if !strings.Contains(arg, " ") && i+1 < len(tokens) && !strings.HasPrefix(strings.ToUpper(tokens[i+1]), "/C:") {
				i++
				arg = arg + " " + tokens[i]
			}
			out = append(out, arg)
			continue
		}
		out = append(out, t)
	}
	return out
}

func runFindstrPipeline(ctx context.Context, leftTokens []string, findstrExe string, findstrArgs []string, cwd string, stdout io.Writer, stderr io.Writer) error {
	leftCmd := exec.CommandContext(ctx, leftTokens[0], leftTokens[1:]...)
	if cwd != "" {
		leftCmd.Dir = cwd
	}
	leftStdout, err := leftCmd.StdoutPipe()
	if err != nil {
		return err
	}
	findCmd := exec.CommandContext(ctx, findstrExe, findstrArgs...)
	if cwd != "" {
		findCmd.Dir = cwd
	}
	findCmd.Stdin = leftStdout
	var bout, berr bytes.Buffer
	findCmd.Stdout = &bout
	findCmd.Stderr = &berr

	if err := leftCmd.Start(); err != nil {
		return err
	}
	if err := findCmd.Start(); err != nil {
		_ = leftCmd.Process.Kill()
		return err
	}

	leftErr := leftCmd.Wait()
	_ = leftStdout.Close()
	findErr := findCmd.Wait()

	_, _ = stdout.Write(bout.Bytes())
	_, _ = stderr.Write(berr.Bytes())

	if err := reportPipelineResult(&bout, &berr, findErr, strings.Join(append(leftTokens, findstrExe), " | ")); err != nil {
		return err
	}
	_ = leftErr
	return nil
}

func reportPipelineResult(bout *bytes.Buffer, berr *bytes.Buffer, findErr error, pipelineDesc string) error {
	if findErr != nil {
		if exitErr, ok := findErr.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 && bout.Len() > 0 {
				return nil
			}
		}
		outStr := strings.TrimSpace(bout.String())
		errStr := strings.TrimSpace(berr.String())
		if outStr != "" || errStr != "" {
			return fmt.Errorf("command failed: %w (pipeline=%q stdout=%q stderr=%q)", findErr, pipelineDesc, outStr, errStr)
		}
		return fmt.Errorf("command failed: %w (pipeline=%q)", findErr, pipelineDesc)
	}
	return nil
}

// runShellCommand executes a command by running the given executable and
// arguments, returning captured stdout/stderr buffers along with any error.
func runShellCommand(ctx context.Context, shell string, args []string, cwd string) (*bytes.Buffer, *bytes.Buffer, error) {
	cmd := exec.CommandContext(ctx, shell, args...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	var bout, berr bytes.Buffer
	cmd.Stdout = &bout
	cmd.Stderr = &berr
	if err := cmd.Run(); err != nil {
		return &bout, &berr, err
	}
	return &bout, &berr, nil
}

// tryHandleWindowsFindstr inspects the command to see if it looks like a
// `A | findstr ...` pipeline and, if so, tries to handle it. Returns true
// if the pipeline was handled successfully.
func tryHandleWindowsFindstr(ctx context.Context, command string, cwd string, stdout io.Writer, stderr io.Writer) bool {
	if !strings.Contains(command, "|") {
		return false
	}
	rhs := strings.TrimSpace(strings.SplitN(command, "|", 2)[1])
	if !strings.HasPrefix(strings.ToLower(rhs), "findstr") {
		return false
	}
	if err := handleWindowsFindstrPipeline(ctx, command, cwd, stdout, stderr); err == nil {
		return true
	}
	return false
}

func (e *Executor) handleDryRunIfNeeded(command string, stdout io.Writer) bool {
	if e.DryRun {
		if e.Verbose {
			_, _ = fmt.Fprintf(stdout, "dry-run: %s\n", command)
		}
		return true
	}
	return false
}

func writeOutputs(bout, berr *bytes.Buffer, stdout io.Writer, stderr io.Writer) {
	if runtime.GOOS == "windows" {
		_, _ = (&unescapeWriter{w: stdout}).Write(bout.Bytes())
		_, _ = (&unescapeWriter{w: stderr}).Write(berr.Bytes())
	} else {
		_, _ = stdout.Write(bout.Bytes())
		_, _ = stderr.Write(berr.Bytes())
	}
}

func checkExecutionError(err error, bout, berr *bytes.Buffer, shell string, args []string) error {
	// If the process exited with status 1 but produced stdout, treat
	// that as a non-fatal condition.
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 1 && bout.Len() > 0 {
			return nil
		}
	}
	outStr := strings.TrimSpace(bout.String())
	errStr := strings.TrimSpace(berr.String())
	if outStr != "" || errStr != "" {
		return fmt.Errorf("command failed: %w (shell=%s args=%q stdout=%q stderr=%q)", err, shell, args, outStr, errStr)
	}
	return fmt.Errorf("command failed: %w (shell=%s args=%q)", err, shell, args)
}

func shellInvocation(command string, overrideShell string) (string, []string) {
	if overrideShell != "" {
		// Handle PowerShell variants explicitly so users can request the
		// Windows-provided `powershell` (legacy Windows PowerShell) or the
		// cross-platform `pwsh` (PowerShell Core).
		switch overrideShell {
		case "pwsh":
			return "pwsh", []string{"-Command", command}
		case "powershell":
			// On Windows prefer the OS-provided 'powershell' if present, else
			// fall back to 'pwsh' if available. On non-Windows prefer 'pwsh'.
			if runtime.GOOS == "windows" {
				if p, err := exec.LookPath("powershell"); err == nil {
					return p, []string{"-Command", command}
				}
				if p, err := exec.LookPath("pwsh"); err == nil {
					return p, []string{"-Command", command}
				}
				// Fallback to the executable name; exec will surface a useful error
				return "powershell", []string{"-Command", command}
			}
			// Non-windows: prefer pwsh
			return "pwsh", []string{"-Command", command}
		default:
			// generic: use override as the shell command and -c to pass command
			return overrideShell, []string{"-c", command}
		}
	}

	if runtime.GOOS == "windows" {
		// Use cmd.exe by default on Windows
		return "cmd", []string{"/C", command}
	}
	// Unix-like
	return "bash", []string{"-c", command}
}

func validateShellAndArgs(shell string, args []string) error {
	// Validate shell + args for obvious issues to avoid opaque CreateProcess errors
	// Ensure shell is available on PATH; we only need to check that it exists.
	if _, err := exec.LookPath(shell); err != nil {
		return fmt.Errorf("shell not found in PATH: %s", shell)
	}
	// ensure args don't contain NUL or control chars that are likely to break CreateProcess
	for i, a := range args {
		if strings.IndexFunc(a, func(r rune) bool { return r == 0 || (r < 32 && r != '\t') || r == 0x7f }) != -1 {
			return fmt.Errorf("invalid shell arg[%d]: contains control characters", i)
		}
	}
	return nil
}

// Sanitize normalizes common unicode characters and removes embedded
// null and other invisible runes. Exported for use by callers (e.g., the
// TUI) that want to sanitize user-edited commands at save time.
func Sanitize(s string) string {
	return sanitizeCommand(s)
}
func validateAndSanitize(command string) (string, error) {
	// First sanitize common unicode punctuation and invisible characters
	// (e.g., smart quotes, NBSP, zero-width spaces, and NUL bytes) before
	// validation so the TUI editor becomes forgiving for accidental insertions.
	command = sanitizeCommand(command)

	// After sanitization, reject multiline commands (newlines are not allowed)
	if strings.Contains(command, "\n") {
		return "", fmt.Errorf("invalid command: contains newline characters; each command must be a single line")
	}

	// If any control characters remain (other than tab), fail with a helpful message
	if strings.IndexFunc(command, func(r rune) bool { return r == 0 || (r < 32 && r != '\t') || r == 0x7f }) != -1 {
		return "", fmt.Errorf("invalid command: contains control characters; remove non-printable characters")
	}
	return command, nil
}

// ValidateCommand checks for remaining problematic characters that will
// cause command execution to fail (e.g., newlines and control characters)
// and returns an error describing the problem if one is found.
func ValidateCommand(s string) error {
	if strings.Contains(s, "\n") {
		return fmt.Errorf("invalid command: contains newline characters; each command must be a single line")
	}
	if strings.IndexFunc(s, func(r rune) bool { return r == 0 || (r < 32 && r != '\t') || r == 0x7f }) != -1 {
		return fmt.Errorf("invalid command: contains control characters; remove non-printable characters")
	}
	return nil
}
