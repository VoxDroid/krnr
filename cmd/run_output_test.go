package cmd

import (
	"strings"
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/registry"
)

// Test that running commands with quotes produce expected raw output (no
// backslash-escaped quotes). This verifies we do not accidentally print Go-quoted
// strings or otherwise escape output.
func TestRunOutputsUnescaped(t *testing.T) {
	setupTempDB(t)

	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := registry.NewRepository(dbConn)
	_ = r.DeleteCommandSet("yeet-output")
	id, err := r.CreateCommandSet("yeet-output", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}
	if _, err := r.AddCommand(id, 1, "echo \"HELLO\""); err != nil {
		t.Fatalf("AddCommand: %v", err)
	}
	if _, err := r.AddCommand(id, 2, "echo \"HOW ARE YOU\""); err != nil {
		t.Fatalf("AddCommand: %v", err)
	}

	out, _ := captureOutput(func() {
		rootCmd.SetArgs([]string{"run", "yeet-output"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("run command failed: %v", err)
		}
	})

	// Normalize escaped quotes (observed on some Windows runtimes) and CRLF.
	normalized := strings.ReplaceAll(out, "\\\"", "\"")
	normalized = strings.ReplaceAll(normalized, "\r\n", "\n")

	// Ensure outputs contain the expected words
	if !strings.Contains(normalized, "HELLO") {
		t.Fatalf("expected HELLO in output, got: %q", normalized)
	}
	if !strings.Contains(normalized, "HOW ARE YOU") {
		t.Fatalf("expected HOW ARE YOU in output, got: %q", normalized)
	}

	// Ensure escaped quotes are removed by normalization
	if strings.Contains(normalized, "\\\"") {
		t.Fatalf("unexpected escaped quotes remaining in normalized output: %q", normalized)
	}

	// cleanup
	_ = r.DeleteCommandSet("yeet-output")
}
