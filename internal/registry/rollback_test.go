package registry

import (
	"testing"
)

func TestApplyVersionToNonEmpty(t *testing.T) {
	r, id := setupVersionRepo(t)
	if err := r.ReplaceCommands(id, []string{"a", "b"}); err != nil {
		t.Fatalf("ReplaceCommands: %v", err)
	}
	if err := r.ReplaceCommands(id, []string{"c"}); err != nil {
		t.Fatalf("ReplaceCommands2: %v", err)
	}

	if err := r.ApplyVersionByName("vtest", 2); err != nil {
		t.Fatalf("ApplyVersionByName: %v", err)
	}

	cs := mustGetCommandSet(t, r, "vtest")
	if len(cs.Commands) != 2 {
		t.Fatalf("expected 2 commands after rollback to version 2, got %d", len(cs.Commands))
	}
	if cs.Commands[0].Command != "a" || cs.Commands[1].Command != "b" {
		t.Fatalf("unexpected commands after rollback: %+v", cs.Commands)
	}

	// Verify exactly one new version was created (rollback), not two
	vers := assertVersionCount(t, r, "vtest", 4)
	if vers[0].Operation != "rollback" {
		t.Fatalf("expected newest operation 'rollback', got %s", vers[0].Operation)
	}
}
