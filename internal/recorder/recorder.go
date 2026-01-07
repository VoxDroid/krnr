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
func RecordCommands(r io.Reader) ([]string, error) {
	s := bufio.NewScanner(r)
	var out []string
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("read commands: %w", err)
	}
	return out, nil
}

// SaveRecorded creates a command set with the given name/description and writes
// the provided commands into it using the repository. Returns the created command set ID.
func SaveRecorded(r *registry.Repository, name string, description *string, commands []string) (int64, error) {
	id, err := r.CreateCommandSet(name, description)
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
