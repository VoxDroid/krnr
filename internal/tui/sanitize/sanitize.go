// Package sanitize provides helpers for sanitizing streaming command output
// when presenting it in the TUI. The goal is to preserve color (SGR) codes
// while removing control sequences that can affect the global terminal state
// (alternate screen, clear-screen, cursor movement, OSC sequences, etc.).
// Cursor-positioning sequences used for layout (e.g., by fastfetch) are
// converted to spaces so their text content remains visible.
package sanitize

import (
	"regexp"
	"strconv"
	"strings"
)

// Precompiled regexps used by RunOutput.
var (
	oscRe = regexp.MustCompile(`\x1b\][^\x07]*\x07`)
	csiRe = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)
)

// RunOutput removes non-SGR control sequences that can affect the terminal
// global state while preserving SGR ("m") color sequences. Cursor-positioning
// sequences (CUF "C" and CHA "G") are replaced with spaces so that programs
// which use side-by-side layout (e.g., fastfetch) remain readable in the TUI
// viewport. CR/LF sequences are normalized to LF.
func RunOutput(in string) string {
	// Normalize CRLF and lone CR to LF
	out := strings.ReplaceAll(in, "\r\n", "\n")
	out = strings.ReplaceAll(out, "\r", "\n")

	// Remove OSC sequences (Operating System Command), e.g., ESC ] ... BEL
	out = oscRe.ReplaceAllString(out, "")

	// Process CSI sequences: keep SGR (colors), convert cursor-positioning
	// to spaces, and remove everything else.
	out = csiRe.ReplaceAllStringFunc(out, replaceCsi)

	return out
}

// replaceCsi handles a single CSI sequence match. It preserves SGR codes,
// converts cursor-forward (C) and cursor-horizontal-absolute (G) to spaces,
// and strips everything else.
func replaceCsi(s string) string {
	suffix := s[len(s)-1]
	switch suffix {
	case 'm':
		return s // preserve SGR color codes
	case 'C':
		// Cursor Forward: \x1b[<n>C → n spaces (default 1)
		return strings.Repeat(" ", csiParam(s, 1))
	case 'G':
		// Cursor Horizontal Absolute: \x1b[<n>G
		// We cannot track the current column in a streaming sanitizer, so
		// emit a fixed separator to keep neighbouring text visible.
		return "  "
	default:
		return ""
	}
}

// csiParam extracts the first numeric parameter from a CSI sequence like
// \x1b[<n><letter>. Returns def if the parameter is absent or invalid.
func csiParam(s string, def int) int {
	// strip "\x1b[" prefix and trailing letter
	body := s[2 : len(s)-1]
	// drop any leading '?' for private-mode sequences
	body = strings.TrimLeft(body, "?")
	if body == "" {
		return def
	}
	// take first param before any ';'
	if idx := strings.IndexByte(body, ';'); idx >= 0 {
		body = body[:idx]
	}
	if n, err := strconv.Atoi(body); err == nil && n > 0 {
		return n
	}
	return def
}
