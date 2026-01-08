package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/exporter"
	"github.com/VoxDroid/krnr/internal/importer"
	"github.com/VoxDroid/krnr/internal/registry"
)

// TestSaveRunExportImportRoundtrip exercises saving a command set via the CLI,
// running it, exporting the DB to a file, importing into a fresh environment,
// and verifying the command set exists and can be run after import.
func TestSaveRunExportImportRoundtrip(t *testing.T) {
	// Create a fresh KRNR_HOME
	tmp := setupTempDB(t)

	// Save a command set via the CLI
	rootCmd.SetArgs([]string{"save", "e2e-roundtrip", "-c", "echo E2E-RUN"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("save command failed: %v", err)
	}

	// Run it and check that output contains the expected text (platform-normalize)
	out, _ := captureOutput(func() {
		rootCmd.SetArgs([]string{"run", "e2e-roundtrip"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("run command failed: %v", err)
		}
	})
	normalized := strings.ReplaceAll(out, "\r\n", "\n")
	if !strings.Contains(normalized, "E2E-RUN") {
		t.Fatalf("expected E2E-RUN in output, got: %q", normalized)
	}

	// Export the database to a file
	dst := filepath.Join(tmp, "e2e-export.db")
	if err := exporter.ExportDatabase(dst); err != nil {
		t.Fatalf("ExportDatabase: %v", err)
	}
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("expected exported DB at %s, stat error: %v", dst, err)
	}

	// Create a fresh environment and ensure DB is empty
	other := t.TempDir()
	os.Setenv("KRNR_HOME", other)
	// initialize an empty DB at the new location
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB for fresh env: %v", err)
	}
	dbConn.Close()

	// Import the exported command set into the fresh DB
	if err := importer.ImportCommandSet(dst); err != nil {
		t.Fatalf("ImportCommandSet: %v", err)
	}

	// Verify the command set exists after import
	dbConn2, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB after import: %v", err)
	}
	defer dbConn2.Close()

	r := registry.NewRepository(dbConn2)
	cs, err := r.GetCommandSetByName("e2e-roundtrip")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected imported command set 'e2e-roundtrip' to exist")
	}

	// Run the imported command set to ensure it's functional
	out2, _ := captureOutput(func() {
		rootCmd.SetArgs([]string{"run", "e2e-roundtrip"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("run after import failed: %v", err)
		}
	})
	norm2 := strings.ReplaceAll(out2, "\r\n", "\n")
	if !strings.Contains(norm2, "E2E-RUN") {
		t.Fatalf("expected E2E-RUN in run-after-import output, got: %q", norm2)
	}
}
