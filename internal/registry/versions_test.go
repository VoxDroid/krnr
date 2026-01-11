package registry

import (
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
)

func TestVersions_RecordAndListAndRollback(t *testing.T) {
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := NewRepository(dbConn)
	// ensure clean state
	_ = r.DeleteCommandSet("vtest")
	desc := "version test"
	id, err := r.CreateCommandSet("vtest", &desc, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}

	// initial replace -> create an update version
	if err := r.ReplaceCommands(id, []string{"echo one", "echo two"}); err != nil {
		t.Fatalf("ReplaceCommands: %v", err)
	}

	// second update
	if err := r.ReplaceCommands(id, []string{"echo three"}); err != nil {
		t.Fatalf("ReplaceCommands2: %v", err)
	}

	vers, err := r.ListVersionsByName("vtest")
	if err != nil {
		t.Fatalf("ListVersionsByName: %v", err)
	}
	if len(vers) < 3 {
		t.Fatalf("expected at least 3 versions (create + 2 updates), got %d", len(vers))
	}

	// newest should be the last update (version 3)
	if vers[0].Version != 3 {
		t.Fatalf("expected newest version 3, got %d", vers[0].Version)
	}
	if vers[0].Operation != "update" {
		t.Fatalf("expected operation 'update', got %s", vers[0].Operation)
	}

	// rollback to version 1 (initial empty commands)
	if err := r.ApplyVersionByName("vtest", 1); err != nil {
		t.Fatalf("ApplyVersionByName: %v", err)
	}

	cs, err := r.GetCommandSetByName("vtest")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected command set after rollback")
	}
	if len(cs.Commands) != 0 {
		t.Fatalf("expected 0 commands after rollback to version 1, got %d", len(cs.Commands))
	}
}
