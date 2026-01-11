// Package importer provides tests for importing functionality.
package importer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
	"github.com/VoxDroid/krnr/internal/exporter"
	"github.com/VoxDroid/krnr/internal/registry"
)

func TestImportCommandSet(t *testing.T) {
	tmp := t.TempDir()
	_ = os.Setenv("HOME", tmp)
	_ = os.Setenv("USERPROFILE", tmp)

	// create source db and command set
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := registry.NewRepository(dbConn)
	desc := "imp"
	id, err := r.CreateCommandSet("imp-set", &desc, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}
	if _, err := r.AddCommand(id, 1, "echo X"); err != nil {
		t.Fatalf("AddCommand: %v", err)
	}

	// export the full DB to a file
	dst := filepath.Join(tmp, "imp.db")
	if err := exporter.ExportDatabase(dst); err != nil {
		t.Fatalf("ExportDatabase: %v", err)
	}

	// Prepare a fresh destination environment (new DB path)
	otherTmp := t.TempDir()
	_ = os.Setenv("HOME", otherTmp)
	_ = os.Setenv("USERPROFILE", otherTmp)
	// initialize empty DB
	dbConn2, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB() 2: %v", err)
	}
	_ = dbConn2.Close()

	// import command sets from exported file
	if err := ImportCommandSet(dst); err != nil {
		t.Fatalf("ImportCommandSet: %v", err)
	}

	// validate imported name exists
	dbConn3, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB() 3: %v", err)
	}
	defer func() { _ = dbConn3.Close() }()

	r2 := registry.NewRepository(dbConn3)
	cs, err := r2.GetCommandSetByName("imp-set")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected imported command set 'imp-set'")
	}
}
