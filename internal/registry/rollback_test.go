package registry

import (
	"testing"
)

func TestApplyVersionToNonEmpty(t *testing.T) {
	r, id := setupVersionRepo(t)
	// first update -> version 2
	if err := r.ReplaceCommands(id, []string{"a", "b"}); err != nil {
		t.Fatalf("ReplaceCommands: %v", err)
	}
	// second update -> version 3
	if err := r.ReplaceCommands(id, []string{"c"}); err != nil {
		t.Fatalf("ReplaceCommands2: %v", err)
	}

	// apply version 2
	if err := r.ApplyVersionByName("vtest", 2); err != nil {
		t.Fatalf("ApplyVersionByName: %v", err)
	}

	cs, err := r.GetCommandSetByName("vtest")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected command set after rollback")
	}
	if len(cs.Commands) != 2 {
		t.Fatalf("expected 2 commands after rollback to version 2, got %d", len(cs.Commands))
	}
	if cs.Commands[0].Command != "a" || cs.Commands[1].Command != "b" {
		t.Fatalf("unexpected commands after rollback: %+v", cs.Commands)
	}
}
