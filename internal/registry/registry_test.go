package registry

import (
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
)

func TestRepository_CRUD(t *testing.T) {
	// init DB
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := NewRepository(dbConn)

	// Create a command set
	// ensure clean state from previous runs
	_ = r.DeleteCommandSet("demo")
	desc := "demo"
	id, err := r.CreateCommandSet("demo", &desc, nil, nil)
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
	cs, err := r.GetCommandSetByName("demo")
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
	if err := r.DeleteCommandSet("demo"); err != nil {
		t.Fatalf("DeleteCommandSet: %v", err)
	}

	cs2, err := r.GetCommandSetByName("demo")
	if err != nil {
		t.Fatalf("GetCommandSetByName after delete: %v", err)
	}
	if cs2 != nil {
		t.Fatalf("expected nil after delete")
	}
}

func TestRepository_Tags(t *testing.T) {
	// init DB
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := NewRepository(dbConn)

	// Create sets (ensure clean state from previous runs)
	_ = r.DeleteCommandSet("alpha")
	_ = r.DeleteCommandSet("beta")
	d1 := "alpha description"
	id1, err := r.CreateCommandSet("alpha", &d1, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet alpha: %v", err)
	}
	if _, err := r.AddCommand(id1, 1, "echo alpha"); err != nil {
		t.Fatalf("AddCommand alpha: %v", err)
	}

	d2 := "beta demo"
	id2, err := r.CreateCommandSet("beta", &d2, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet beta: %v", err)
	}
	if _, err := r.AddCommand(id2, 1, "echo beta"); err != nil {
		t.Fatalf("AddCommand beta: %v", err)
	}

	// Tags
	if err := r.AddTagToCommandSet(id1, "utils"); err != nil {
		t.Fatalf("AddTagToCommandSet: %v", err)
	}
	if err := r.AddTagToCommandSet(id2, "demo"); err != nil {
		t.Fatalf("AddTagToCommandSet: %v", err)
	}

	// List tags
	tags1, err := r.ListTagsForCommandSet(id1)
	if err != nil {
		t.Fatalf("ListTagsForCommandSet: %v", err)
	}
	found := false
	for _, tg := range tags1 {
		if tg == "utils" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected tag 'utils' for alpha")
	}

	// List by tag
	setsWithUtils, err := r.ListCommandSetsByTag("utils")
	if err != nil {
		t.Fatalf("ListCommandSetsByTag: %v", err)
	}
	if len(setsWithUtils) != 1 || setsWithUtils[0].Name != "alpha" {
		t.Fatalf("expected only alpha for tag 'utils', got %+v", setsWithUtils)
	}

	// GetCommandSetByName includes tags
	csAlpha, err := r.GetCommandSetByName("alpha")
	if err != nil {
		t.Fatalf("GetCommandSetByName alpha: %v", err)
	}
	if csAlpha == nil {
		t.Fatalf("expected alpha command set")
	}
	if len(csAlpha.Tags) == 0 {
		t.Fatalf("expected tags on alpha, got none")
	}

	// Remove tag
	if err := r.RemoveTagFromCommandSet(id1, "utils"); err != nil {
		t.Fatalf("RemoveTagFromCommandSet: %v", err)
	}
	tagsAfter, err := r.ListTagsForCommandSet(id1)
	if err != nil {
		t.Fatalf("ListTagsForCommandSet after remove: %v", err)
	}
	for _, tg := range tagsAfter {
		if tg == "utils" {
			t.Fatalf("expected tag 'utils' to be removed")
		}
	}
}

func TestRepository_Search(t *testing.T) {
	// init DB
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := NewRepository(dbConn)

	// ensure clean state
	_ = r.DeleteCommandSet("alpha")
	_ = r.DeleteCommandSet("beta")
	d1 := "alpha description"
	id1, err := r.CreateCommandSet("alpha", &d1, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet alpha: %v", err)
	}
	if _, err := r.AddCommand(id1, 1, "echo alpha"); err != nil {
		t.Fatalf("AddCommand alpha: %v", err)
	}

	d2 := "beta demo"
	id2, err := r.CreateCommandSet("beta", &d2, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet beta: %v", err)
	}
	if _, err := r.AddCommand(id2, 1, "echo beta"); err != nil {
		t.Fatalf("AddCommand beta: %v", err)
	}

	// Search
	results, err := r.SearchCommandSets("demo")
	if err != nil {
		t.Fatalf("SearchCommandSets: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected search results for 'demo'")
	}
	foundBeta := false
	for _, s := range results {
		if s.Name == "beta" {
			foundBeta = true
			break
		}
	}
	if !foundBeta {
		t.Fatalf("expected 'beta' in search results")
	}
}
