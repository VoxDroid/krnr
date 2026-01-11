// Package recorder provides command recording functionality.
package recorder

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/VoxDroid/krnr/internal/registry"
)

// sentinel tokens that stop recording
var sentinelTokens = map[string]struct{}{
	":end":  {},
	":save": {},
	":quit": {},
}

func isSentinel(s string) bool {
	_, ok := sentinelTokens[strings.TrimSpace(s)]
	return ok
}

func shouldKeepLine(s string) bool {
	t := strings.TrimSpace(s)
	if t == "" {
		return false
	}
	if strings.HasPrefix(t, "#") {
		return false
	}
	return true
}

func processLineAppend(out *[]string, line string) (bool, error) {
	trim := strings.TrimSpace(line)
	if isSentinel(trim) {
		return true, nil
	}
	if shouldKeepLine(trim) {
		*out = append(*out, trim)
	}
	return false, nil
}

// truncateAtStop looks for Ctrl+Z (0x1A) or caret sequences '^Z'/'^z' and returns
// the prefix before the stop marker and a boolean indicating whether a stop was found.
func truncateAtStop(s string) (string, bool) {
	if idx := strings.IndexByte(s, 0x1A); idx >= 0 {
		return s[:idx], true
	}
	if idx := strings.Index(s, "^Z"); idx >= 0 {
		return s[:idx], true
	}
	if idx := strings.Index(s, "^z"); idx >= 0 {
		return s[:idx], true
	}
	return s, false
}

// RecordCommands reads lines from r until EOF and returns non-empty, non-comment lines
// as a slice of commands. Lines starting with '#' are treated as comments and ignored.
// Special sentinel lines (single-line commands) `:end`, `:save`, and `:quit` stop
// recording immediately when a line consisting only of that token (after trimming)
// is encountered. This implementation reads by line using bufio.Reader.ReadString and
// handles Ctrl+Z (0x1A) and '^Z' sequences by truncating input at the first occurrence
// so behavior is unchanged while simplifying control flow and reducing cyclomatic complexity.
func RecordCommands(r io.Reader) ([]string, error) {
	br := bufio.NewReader(r)
	var out []string
	for {
		s, err := br.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("read commands: %w", err)
		}

	
		// Truncate at any stop markers (Ctrl+Z or ^Z/^z) and handle the prefix.
		if prefix, stopped := truncateAtStop(s); stopped {
			if stop, err := processLineAppend(&out, prefix); err != nil {
				return nil, err
			} else if stop {
				return out, nil
			}
			return out, nil
		}

		// Process the line (without trailing newline)
		line := strings.TrimRight(s, "\r\n")
		if stop, err := processLineAppend(&out, line); err != nil {
			return nil, err
		} else if stop {
			return out, nil
		}

		if err == io.EOF {
			return out, nil
		}
	}
}

// SaveRecorded creates a command set with the given name/description and writes
// the provided commands into it using the repository. Returns the created command set ID.
func SaveRecorded(r *registry.Repository, name string, description *string, commands []string) (int64, error) {
	id, err := r.CreateCommandSet(name, description, nil, nil)
	if err != nil {
		return 0, err
	}
	for i, c := range commands {
		if _, err := r.AddCommand(id, i+1, c); err != nil {
			return 0, err
		}
	}
	return id, nil
}
