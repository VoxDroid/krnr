package registry

import (
	"testing"
)

func setupVersionRepo(t *testing.T) (*Repository, int64) {
	r := setupTestDB(t)
	// ensure clean state
	_ = r.DeleteCommandSet("vtest")
	desc := "version test"
	id, err := r.CreateCommandSet("vtest", &desc, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}
	return r, id
}

func TestVersions_RecordAndList(t *testing.T) {
	r, id := setupVersionRepo(t)
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
}

func TestVersions_Rollback(t *testing.T) {
	r, id := setupVersionRepo(t)
	if err := r.ReplaceCommands(id, []string{"echo one", "echo two"}); err != nil {
		t.Fatalf("ReplaceCommands: %v", err)
	}
	if err := r.ReplaceCommands(id, []string{"echo three"}); err != nil {
		t.Fatalf("ReplaceCommands2: %v", err)
	}

	assertVersionCount(t, r, "vtest", 3)

	if err := r.ApplyVersionByName("vtest", 1); err != nil {
		t.Fatalf("ApplyVersionByName: %v", err)
	}

	cs := mustGetCommandSet(t, r, "vtest")
	if len(cs.Commands) != 0 {
		t.Fatalf("expected 0 commands after rollback to version 1, got %d", len(cs.Commands))
	}

	// Rollback must create exactly ONE new version (not two)
	vers := assertVersionCount(t, r, "vtest", 4)
	if vers[0].Operation != "rollback" {
		t.Fatalf("expected newest operation 'rollback', got %s", vers[0].Operation)
	}
	if vers[0].Version != 4 {
		t.Fatalf("expected newest version 4, got %d", vers[0].Version)
	}
}

func assertVersionCount(t *testing.T, r *Repository, name string, expected int) []Version {
	t.Helper()
	vers, err := r.ListVersionsByName(name)
	if err != nil {
		t.Fatalf("ListVersionsByName: %v", err)
	}
	if len(vers) != expected {
		t.Fatalf("expected %d versions, got %d", expected, len(vers))
	}
	return vers
}

func mustGetCommandSet(t *testing.T, r *Repository, name string) *CommandSet {
	t.Helper()
	cs, err := r.GetCommandSetByName(name)
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected command set %q to exist", name)
	}
	return cs
}
