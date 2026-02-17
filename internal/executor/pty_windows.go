//go:build windows

package executor

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
)

// isTerminal always returns false on Windows since PTY-based interactive
// execution is not supported. The non-interactive streaming path is used
// instead.
var isTerminal = func(_ uintptr) bool {
	return false
}

// ptyStarter is not supported on Windows. It returns an error if called.
var ptyStarter = func(_ *exec.Cmd, _ io.Reader, _, _ io.Writer) (*bytes.Buffer, *bytes.Buffer, error) {
	return &bytes.Buffer{}, &bytes.Buffer{}, fmt.Errorf("PTY not supported on Windows")
}
