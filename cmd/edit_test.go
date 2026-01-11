package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
)

func TestEditCommand_Interactive(t *testing.T) {
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := registry.NewRepository(dbConn)
	_ = r.DeleteCommandSet("edit-test")
	id, err := r.CreateCommandSet("edit-test", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}
	if _, err := r.AddCommand(id, 1, "oldcmd"); err != nil {
		t.Fatalf("AddCommand: %v", err)
	}

	scriptPath := writeEditorScript(t)
	_ = os.Setenv("EDITOR", scriptPath)
	defer func() { _ = os.Unsetenv("EDITOR") }()

	// Run the edit command which should invoke the script, then read the temp file
	if err := editCmd.RunE(editCmd, []string{"edit-test"}); err != nil {
		t.Fatalf("editCmd failed: %v", err)
	}

	cs, err := r.GetCommandSetByName("edit-test")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected command set")
	}
	if len(cs.Commands) != 2 {
		t.Fatalf("expected 2 commands after edit, got %d", len(cs.Commands))
	}
	if cs.Commands[0].Command != "new1" || cs.Commands[1].Command != "new2" {
		t.Fatalf("unexpected commands: %+v", cs.Commands)
	}

	// cleanup
	_ = r.DeleteCommandSet("edit-test")
}

func writeEditorScript(t *testing.T) string {
	if runtime.GOOS == "windows" {
		return writeEditorScriptWindows(t)
	}
	return writeEditorScriptUnix(t)
}

func writeEditorScriptWindows(t *testing.T) string {
	d := t.TempDir()
	scriptPath := filepath.Join(d, "edit.bat")
	// write the new commands to the file passed as the first argument (%~1) so the CLI reads them back
	script := "@echo off\r\necho new1 > \"%~1\"\r\necho new2 >> \"%~1\"\r\nexit /b 0\r\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return scriptPath
}

func writeEditorScriptUnix(t *testing.T) string {
	d := t.TempDir()
	scriptPath := filepath.Join(d, "edit.sh")
	// write the new commands to the file passed as $1 so the CLI reads them back
	script := "#!/bin/sh\nprintf 'new1\nnew2\n' > \"$1\"\nexit 0\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	if err := os.Chmod(scriptPath, 0o755); err != nil {
		t.Fatalf("chmod script: %v", err)
	}
	return scriptPath
}
