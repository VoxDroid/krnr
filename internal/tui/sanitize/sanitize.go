// Package sanitize provides helpers for sanitizing streaming command output
// when presenting it in the TUI. The goal is to preserve color (SGR) codes
// while removing control sequences that can affect the global terminal state
// (alternate screen, clear-screen, cursor movement, OSC sequences, etc.).
package sanitize

import (
	"regexp"
	"strings"
)

// RunOutput removes non-SGR control sequences that can affect the terminal
// global state while preserving SGR ("m") color sequences. It also
// normalizes CR/LF sequences to LF for predictable rendering inside the
// TUI output viewport.
func RunOutput(in string) string {
	// Normalize CRLF and lone CR to LF
	out := strings.ReplaceAll(in, "\r\n", "\n")
	out = strings.ReplaceAll(out, "\r", "\n")

	// Remove OSC sequences (Operating System Command), e.g., ESC ] ... BEL
	oscRe := regexp.MustCompile(`\x1b\][^\x07]*\x07`)
	out = oscRe.ReplaceAllString(out, "")

	// Remove CSI sequences that are NOT SGR (i.e., do not end with 'm').
	// Keep SGR sequences (colors) intact.
	csiRe := regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)
	out = csiRe.ReplaceAllStringFunc(out, func(s string) string {
		if strings.HasSuffix(s, "m") {
			return s // preserve SGR color codes
		}
		return ""
	})

	return out
}
