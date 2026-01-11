package registry

import (
	"testing"

	"github.com/VoxDroid/krnr/internal/db"
)

func TestFuzzyMatchBasics(t *testing.T) {
	cases := []struct {
		target string
		query  string
		expect bool
	}{
		{"alpha", "al", true},
		{"alpha", "ah", true},
		{"alpha", "ph", true},
		{"alpha", "xa", false},
		{"Hello World", "helloworld", true},
		{"Hello World", "hwd", true},
		{"Hello", "", true},
	}
	for _, c := range cases {
		got := FuzzyMatch(c.target, c.query)
		if got != c.expect {
			t.Fatalf("FuzzyMatch(%q, %q) = %v, want %v", c.target, c.query, got, c.expect)
		}
	}
}

func setupFuzzyRepo(t *testing.T) *Repository {
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	// ensure cleanup
	t.Cleanup(func() { _ = dbConn.Close() })
	r := NewRepository(dbConn)
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
	if err := r.AddTagToCommandSet(id1, "utils"); err != nil {
		t.Fatalf("AddTagToCommandSet alpha: %v", err)
	}

	d2 := "beta demo"
	id2, err := r.CreateCommandSet("beta", &d2, nil, nil)
	if err != nil {
		t.Fatalf("CreateCommandSet beta: %v", err)
	}
	if _, err := r.AddCommand(id2, 1, "echo beta"); err != nil {
		t.Fatalf("AddCommand beta: %v", err)
	}
	if err := r.AddTagToCommandSet(id2, "demo"); err != nil {
		t.Fatalf("AddTagToCommandSet beta: %v", err)
	}
	return r
}

func TestFuzzySearchCommandSets_AlphaMatches(t *testing.T) {
	r := setupFuzzyRepo(t)
	res, err := r.FuzzySearchCommandSets("alp")
	if err != nil {
		t.Fatalf("FuzzySearchCommandSets: %v", err)
	}
	if len(res) == 0 || res[0].Name != "alpha" {
		t.Fatalf("expected alpha in fuzzy search for 'alp', got: %+v", res)
	}
}

func TestFuzzySearchCommandSets_BetaMatches(t *testing.T) {
	r := setupFuzzyRepo(t)
	res, err := r.FuzzySearchCommandSets("dmo")
	if err != nil {
		t.Fatalf("FuzzySearchCommandSets: %v", err)
	}
	found := false
	for _, s := range res {
		if s.Name == "beta" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected beta in fuzzy search for 'dmo', got: %+v", res)
	}
}
