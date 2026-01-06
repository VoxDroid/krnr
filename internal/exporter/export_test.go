package exporter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
)

func TestExportDatabase(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv("HOME", tmp)
	os.Setenv("USERPROFILE", tmp)

	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	dbConn.Close()

	dst := filepath.Join(tmp, "exported.db")
	if err := ExportDatabase(dst); err != nil {
		t.Fatalf("ExportDatabase: %v", err)
	}
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("exported file not found: %v", err)
	}
}
