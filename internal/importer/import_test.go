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

func exportTempDB(t *testing.T, setName string, cmds []string) string {
	t.Helper()
	tmp := t.TempDir()
	_ = os.Setenv("HOME", tmp)
	_ = os.Setenv("USERPROFILE", tmp)
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()
	r := registry.NewRepository(dbConn)
	desc := setName
	id, err := r.CreateCommandSet(setName, &desc, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}
	for i, c := range cmds {
		if _, err := r.AddCommand(id, i+1, c); err != nil {
			t.Fatalf("AddCommand: %v", err)
		}
	}
	dst := filepath.Join(tmp, setName+".db")
	if err := exporter.ExportDatabase(dst); err != nil {
		t.Fatalf("ExportDatabase: %v", err)
	}
	return dst
}

func prepareDestination(t *testing.T) string {
	t.Helper()
	otherTmp := t.TempDir()
	_ = os.Setenv("HOME", otherTmp)
	_ = os.Setenv("USERPROFILE", otherTmp)
	// initialize empty DB
	dbConn2, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB() 2: %v", err)
	}
	_ = dbConn2.Close()
	return otherTmp
}

func TestImportCommandSet(t *testing.T) {
	dst := exportTempDB(t, "imp-set", []string{"echo X"})
	prepareDestination(t)
	if err := ImportCommandSet(dst, ImportOptions{}); err != nil {
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

func createExistingSet(t *testing.T, name string, cmds []string) {
	t.Helper()
	dbConn3, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB() existing: %v", err)
	}
	r2 := registry.NewRepository(dbConn3)
	desc2 := name + "-existing"
	id2, err := r2.CreateCommandSet(name, &desc2, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet existing: %v", err)
	}
	for i, c := range cmds {
		if _, err := r2.AddCommand(id2, i+1, c); err != nil {
			t.Fatalf("AddCommand existing: %v", err)
		}
	}
	_ = dbConn3.Close()
}

func TestImportCommandSetMergeAndDedupe(t *testing.T) {
	dst := exportTempDB(t, "imp-merge", []string{"echo A", "echo B"})
	prepareDestination(t)
	createExistingSet(t, "imp-merge", []string{"echo B", "echo C"})

	// import command sets from exported file with merge + dedupe
	if err := ImportCommandSet(dst, ImportOptions{OnConflict: "merge", Dedupe: true}); err != nil {
		t.Fatalf("ImportCommandSet merge: %v", err)
	}

	// validate merged commands
	dbConn4, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB() 4: %v", err)
	}
	defer func() { _ = dbConn4.Close() }()

	r3 := registry.NewRepository(dbConn4)
	cs, err := r3.GetCommandSetByName("imp-merge")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected merged command set 'imp-merge'")
	}
	// existing commands were [echo B, echo C], incoming [echo A, echo B] => merged [echo B, echo C, echo A]
	if len(cs.Commands) != 3 || cs.Commands[0].Command != "echo B" || cs.Commands[1].Command != "echo C" || cs.Commands[2].Command != "echo A" {
		t.Fatalf("unexpected merged commands: %+v", cs.Commands)
	}
}
