package registry

import (
	"testing"
)

func TestApplyVersionFiltersEmptyCommands(t *testing.T) {
	r, id := setupVersionRepo(t)
	// initial set non-empty commands
	if err := r.ReplaceCommands(id, []string{"a", "b"}); err != nil {
		t.Fatalf("ReplaceCommands: %v", err)
	}
	// simulate a buggy version that somehow stored empty commands
	if err := r.RecordVersion(id, nil, nil, nil, []string{"", "   ", "c"}, "update"); err != nil {
		t.Fatalf("RecordVersion: %v", err)
	}
	// find the version number of the last inserted (should be 3)
	vers, err := r.ListVersionsByName("vtest")
	if err != nil {
		t.Fatalf("ListVersionsByName: %v", err)
	}
	if len(vers) == 0 {
		t.Fatalf("expected versions")
	}
	last := vers[0].Version
	// apply that version
	if err := r.ApplyVersionByName("vtest", last); err != nil {
		t.Fatalf("ApplyVersionByName: %v", err)
	}
	cs, err := r.GetCommandSetByName("vtest")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected command set after rollback")
	}
	// expect only the non-empty "c" to be present
	if len(cs.Commands) != 1 {
		t.Fatalf("expected 1 command after filtering, got %d", len(cs.Commands))
	}
	if cs.Commands[0].Command != "c" {
		t.Fatalf("expected command 'c', got %+v", cs.Commands)
	}
}
