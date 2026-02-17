package adapters

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/VoxDroid/krnr/internal/executor"
	"github.com/VoxDroid/krnr/internal/tui/sanitize"
	"golang.org/x/term"
)

// executorAdapter implements ExecutorAdapter using an executor.Runner.
type executorAdapter struct{ runner executor.Runner }

// hostIsTerminal determines whether the provided fd refers to a terminal on
// the host. It is a package-level variable so unit tests can override it to
// simulate terminal conditions.
var hostIsTerminal = func(fd int) bool { return term.IsTerminal(fd) }

// NewExecutorAdapter constructs an ExecutorAdapter backed by the provided Runner.
func NewExecutorAdapter(r executor.Runner) ExecutorAdapter { return &executorAdapter{runner: r} }

// fdReader wraps an io.Reader and exposes a Fd() method so the executor's
// PTY detection recognises it as terminal-backed. The fd reports the host
// stdin file descriptor; the actual reads come from the wrapped pipe reader.
type fdReader struct {
	r  io.Reader
	fd uintptr
}

func (f *fdReader) Read(p []byte) (int, error) { return f.r.Read(p) }
func (f *fdReader) Fd() uintptr                { return f.fd }

func (e *executorAdapter) Run(ctx context.Context, _ string, commands []string) (RunHandle, error) {
	ctx, cancel := context.WithCancel(ctx)
	rchan := make(chan RunEvent)
	run := &runHandleImpl{ch: rchan, cancel: cancel}

	go func() {
		defer close(rchan)
		for _, cmdText := range commands {
			rchan <- RunEvent{Line: fmt.Sprintf("-> %s", cmdText)}
			if err := e.execAndStream(ctx, cmdText, rchan, run); err != nil {
				rchan <- RunEvent{Err: fmt.Errorf("exec: %w", err)}
				return
			}
		}
	}()

	return run, nil
}

// execAndStream launches a single command, streams its output to rchan, and
// returns the command error (if any). It wires up stdin/stdout pipes and
// the escape-sequence buffering loop.
func (e *executorAdapter) execAndStream(ctx context.Context, cmdText string, rchan chan<- RunEvent, run *runHandleImpl) error {
	rOut, wOut := io.Pipe()
	rIn, wIn := io.Pipe()
	run.stdin = wIn

	execErr := make(chan error, 1)
	go func() {
		// Wrap stdin so the executor detects a terminal fd and activates
		// hybrid PTY mode (stdin/ctty via PTY, stdout/stderr via pipes).
		// This lets interactive prompts (sudo password) work while keeping
		// stdout as a pipe so programs like fastfetch use simple output.
		stdinReader := prepareStdin(rIn)
		execErr <- e.runner.Execute(ctx, cmdText, "", stdinReader, wOut, wOut)
		_ = wOut.Close()
		_ = wIn.Close()
	}()

	streamOutput(ctx, rOut, rchan)

	err := <-execErr
	_ = rOut.Close()
	return err
}

// prepareStdin wraps rIn with an Fd() accessor when the host stdin is a
// terminal so the executor detects it and runs the child with hybrid PTY
// (stdin/ctty via PTY, stdout/stderr as pipes).
func prepareStdin(rIn io.Reader) io.Reader {
	if hostIsTerminal(int(os.Stdin.Fd())) {
		return &fdReader{r: rIn, fd: os.Stdin.Fd()}
	}
	return rIn
}

// streamOutput reads from rOut in chunks, buffers incomplete escape sequences
// across reads, and emits sanitized lines to rchan.
func streamOutput(ctx context.Context, rOut io.ReadCloser, rchan chan<- RunEvent) {
	buf := make([]byte, 4096)
	escBuf := ""
	for {
		select {
		case <-ctx.Done():
			_ = rOut.Close()
			flushEscBuf(escBuf, rchan)
			return
		default:
			n, err := rOut.Read(buf)
			if n > 0 {
				escBuf = emitChunkLines(escBuf+string(buf[:n]), rchan)
			}
			if err != nil {
				if err != io.EOF {
					rchan <- RunEvent{Err: err}
				}
				_ = rOut.Close()
				flushEscBuf(escBuf, rchan)
				return
			}
		}
	}
}

// flushEscBuf emits any remaining data held in the escape-sequence carry
// buffer. This is called when the stream ends (EOF or cancellation) to
// ensure no trailing output is silently dropped.
func flushEscBuf(escBuf string, rchan chan<- RunEvent) {
	if escBuf == "" {
		return
	}
	line := strings.TrimRight(escBuf, "\r\n")
	if line == "" {
		return
	}
	rchan <- RunEvent{Line: sanitize.RunOutput(line)}
}

// emitChunkLines splits a chunk into lines, sanitizes each, and sends them
// to rchan. It returns any trailing incomplete escape sequence that should
// be carried into the next read.
func emitChunkLines(chunk string, rchan chan<- RunEvent) string {
	tail := trailingIncompleteEscape(chunk)
	if tail != "" {
		chunk = chunk[:len(chunk)-len(tail)]
	}
	parts := strings.SplitAfter(chunk, "\n")
	var filtered []string
	for _, p := range parts {
		if p != "\n" {
			filtered = append(filtered, p)
		}
	}
	for _, p := range filtered {
		line := strings.TrimRight(p, "\r\n")
		if line == "" {
			continue
		}
		rchan <- RunEvent{Line: sanitize.RunOutput(line)}
	}
	return tail
}

type runHandleImpl struct {
	ch     <-chan RunEvent
	cancel context.CancelFunc
	stdin  io.WriteCloser
}

func (r *runHandleImpl) Events() <-chan RunEvent { return r.ch }
func (r *runHandleImpl) Cancel()                 { r.cancel() }

func (r *runHandleImpl) WriteInput(p []byte) (int, error) {
	if r.stdin == nil {
		return 0, fmt.Errorf("run does not accept input")
	}
	return r.stdin.Write(p)
}

// trailingIncompleteEscape inspects the provided string and returns a
// trailing substring that looks like the start of an escape/control
// sequence but is incomplete (split across reads). The returned tail
// should be kept and prepended to the next read before sanitization.
// If there is no trailing incomplete escape, an empty string is
// returned.
var (
	csiStartRe = regexp.MustCompile(`^\x1b\[[0-9;?]*[A-Za-z]`)
	oscStartRe = regexp.MustCompile(`^\x1b\][^\x07]*\x07`)
	dcsStartRe = regexp.MustCompile(`^\x1bP[\s\S]*?\x1b\\`)
)

func trailingIncompleteEscape(s string) string {
	idx := strings.LastIndex(s, "\x1b")
	if idx == -1 {
		return ""
	}
	// If ESC is the last byte, it's incomplete
	if idx == len(s)-1 {
		return s[idx:]
	}
	// Tail from last ESC
	tail := s[idx:]
	// If a complete CSI/OSC/DCS sequence exists at the start of the tail,
	// then there's no incomplete sequence spanning this read boundary.
	if csiStartRe.MatchString(tail) || oscStartRe.MatchString(tail) || dcsStartRe.MatchString(tail) {
		return ""
	}
	// Otherwise treat the tail from ESC as an incomplete escape sequence
	// that should be carried into the next read for proper detection.
	// Cap the carry size to avoid unbounded growth.
	if len(tail) > 1024 {
		return tail[len(tail)-1024:]
	}
	return tail
}
