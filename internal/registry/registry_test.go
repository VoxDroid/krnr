package registry

import (
	"testing"

	"github.com/drei/krnr/internal/db"
)

func TestRepository_CRUD(t *testing.T) {
	// init DB
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer dbConn.Close()

	r := NewRepository(dbConn)

	// Create a command set
	desc := "demo"
	id, err := r.CreateCommandSet("demo-set", &desc)
	if err != nil {
		t.Fatalf("CreateCommandSet: %v", err)
	}
	if id == 0 {
		t.Fatalf("expected non-zero id")
	}

	// Add commands
	if _, err := r.AddCommand(id, 1, "echo hello"); err != nil {
		t.Fatalf("AddCommand: %v", err)
	}
	if _, err := r.AddCommand(id, 2, "echo world"); err != nil {
		t.Fatalf("AddCommand: %v", err)
	}

	// Retrieve
	cs, err := r.GetCommandSetByName("demo-set")
	if err != nil {
		t.Fatalf("GetCommandSetByName: %v", err)
	}
	if cs == nil {
		t.Fatalf("expected command set")
	}
	if len(cs.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cs.Commands))
	}

	// List
	sets, err := r.ListCommandSets()
	if err != nil {
		t.Fatalf("ListCommandSets: %v", err)
	}
	if len(sets) == 0 {
		t.Fatalf("expected at least one command set")
	}

	// Delete
	if err := r.DeleteCommandSet("demo-set"); err != nil {
		t.Fatalf("DeleteCommandSet: %v", err)
	}

	cs2, err := r.GetCommandSetByName("demo-set")
	if err != nil {
		t.Fatalf("GetCommandSetByName after delete: %v", err)
	}
	if cs2 != nil {
		t.Fatalf("expected nil after delete")
	}
}
