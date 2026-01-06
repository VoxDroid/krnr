package utils

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// OpenEditor opens the given file in the user's preferred editor.
// It respects the $EDITOR environment variable. On Windows if $EDITOR is not set,
// it falls back to notepad; on Unix it falls back to vi.
func OpenEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		if runtime.GOOS == "windows" {
			editor = "notepad"
		} else {
			editor = "vi"
		}
	}
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("open editor: %w", err)
	}
	return nil
}
