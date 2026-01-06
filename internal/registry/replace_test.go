package registry

import (
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
)

func TestReplaceCommands(t *testing.T) {
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer dbConn.Close()

	r := NewRepository(dbConn)
	desc := "replace"
	id, err := r.CreateCommandSet("rep-set", &desc)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}
	if _, err := r.AddCommand(id, 1, "one"); err != nil {
		t.Fatalf("AddCommand: %v", err)
	}
	if err := r.ReplaceCommands(id, []string{"alpha", "beta", "gamma"}); err != nil {
		t.Fatalf("ReplaceCommands: %v", err)
	}
	cs, err := r.GetCommandSetByName("rep-set")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if len(cs.Commands) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(cs.Commands))
	}
	if cs.Commands[0].Command != "alpha" {
		t.Fatalf("expected alpha at pos 1, got %s", cs.Commands[0].Command)
	}
}
