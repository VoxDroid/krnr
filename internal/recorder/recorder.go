// Package recorder provides command recording functionality.
package recorder

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/VoxDroid/krnr/internal/registry"
)

// RecordCommands reads lines from r until EOF and returns non-empty, non-comment lines
// as a slice of commands. Lines starting with '#' are treated as comments and ignored.
// Special sentinel lines (single-line commands) `:end`, `:save`, and `:quit` stop
// recording immediately when a line consisting only of that token (after trimming)
// is encountered. This is a simple and robust mechanism that works across shells
// and Windows consoles without relying on raw terminal mode. Ctrl+Z (0x1A) and
// caret-Z sequences are still treated as immediate EOF when observed.
func RecordCommands(r io.Reader) ([]string, error) {
	br := bufio.NewReader(r)
	var out []string
	var lineBuf strings.Builder
	isSentinel := func(s string) bool {
		trim := strings.TrimSpace(s)
		switch trim {
		case ":end", ":save", ":quit":
			return true
		default:
			return false
		}
	}
	processLine := func(line string) (bool, error) {
		trim := strings.TrimSpace(line)
		if isSentinel(trim) {
			return true, nil
		}
		if trim != "" && !strings.HasPrefix(trim, "#") {
			out = append(out, trim)
		}
		return false, nil
	}

	for {
		b, err := br.ReadByte()
		if err != nil {
			if err == io.EOF {
				if lineBuf.Len() > 0 {
					if stop, err := processLine(lineBuf.String()); err != nil {
						return nil, err
					} else if stop {
						return out, nil
					}
				}
				return out, nil
			}
			return nil, fmt.Errorf("read commands: %w", err)
		}

		// ASCII SUB (Ctrl+Z) should stop reading immediately
		if b == 0x1A {
			if lineBuf.Len() > 0 {
				if stop, err := processLine(lineBuf.String()); err != nil {
					return nil, err
				} else if stop {
					return out, nil
				}
			}
			return out, nil
		}

		// Some Windows consoles echo a caret sequence '^Z' when Ctrl+Z is pressed
		// (two characters '^' and 'Z' or 'z'). Treat that sequence as EOF and
		// ignore it so the user doesn't have to press Enter after typing Ctrl+Z.
		if b == '^' {
			nb, err := br.ReadByte()
			if err != nil {
				if err == io.EOF {
					lineBuf.WriteByte('^')
					if lineBuf.Len() > 0 {
						if stop, err := processLine(lineBuf.String()); err != nil {
							return nil, err
						} else if stop {
							return out, nil
						}
					}
					return out, nil
				}
				return nil, fmt.Errorf("read commands: %w", err)
			}
			if nb == 'Z' || nb == 'z' {
				if lineBuf.Len() > 0 {
					if stop, err := processLine(lineBuf.String()); err != nil {
						return nil, err
					} else if stop {
						return out, nil
					}
				}
				return out, nil
			}
			lineBuf.WriteByte('^')
			lineBuf.WriteByte(nb)
			continue
		}

		if b == '\n' {
			if stop, err := processLine(lineBuf.String()); err != nil {
				return nil, err
			} else if stop {
				return out, nil
			}
			lineBuf.Reset()
			continue
		}

		// accumulate bytes into the current line
		lineBuf.WriteByte(b)
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
