package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOpenEditor_Success(t *testing.T) {
	d := t.TempDir()
	marker := filepath.Join(d, "marker.txt")
	// create a small script that writes 'ok' to marker and exits 0
	var script string
	if runtime.GOOS == "windows" {
		script = "@echo off\r\necho ok > \"" + marker + "\"\r\nexit /b 0\r\n"
		scriptPath := filepath.Join(d, "fake-editor.bat")
		if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
			t.Fatalf("write script: %v", err)
		}
		_ = os.Setenv("EDITOR", scriptPath)
		defer func() { _ = os.Unsetenv("EDITOR") }()
	} else {
		script = "#!/bin/sh\nprintf 'ok' > \"" + marker + "\"\nexit 0\n"
		scriptPath := filepath.Join(d, "fake-editor.sh")
		if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
			t.Fatalf("write script: %v", err)
		}
		if err := os.Chmod(scriptPath, 0o755); err != nil {
			t.Fatalf("chmod script: %v", err)
		}
		_ = os.Setenv("EDITOR", scriptPath)
		defer func() { _ = os.Unsetenv("EDITOR") }()
	}

	// call OpenEditor with a dummy file path
	if err := OpenEditor(filepath.Join(d, "dummy.txt")); err != nil {
		t.Fatalf("OpenEditor failed: %v", err)
	}

	b, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("marker not written: %v", err)
	}
	if strings.TrimSpace(string(b)) != "ok" {
		t.Fatalf("unexpected marker content: %q", string(b))
	}
}

func TestOpenEditor_Failure(t *testing.T) {
	d := t.TempDir()
	// create a script that exits non-zero
	var script string
	if runtime.GOOS == "windows" {
		script = "@echo off\r\nexit /b 1\r\n"
		scriptPath := filepath.Join(d, "fail-editor.bat")
		if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
			t.Fatalf("write script: %v", err)
		}
		_ = os.Setenv("EDITOR", scriptPath)
		defer func() { _ = os.Unsetenv("EDITOR") }()
	} else {
		script = "#!/bin/sh\nexit 1\n"
		scriptPath := filepath.Join(d, "fail-editor.sh")
		if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
			t.Fatalf("write script: %v", err)
		}
		if err := os.Chmod(scriptPath, 0o755); err != nil {
			t.Fatalf("chmod script: %v", err)
		}
		_ = os.Setenv("EDITOR", scriptPath)
		defer func() { _ = os.Unsetenv("EDITOR") }()
	}

	if err := OpenEditor(filepath.Join(d, "dummy.txt")); err == nil {
		t.Fatalf("expected error from failing editor, got nil")
	}
}
