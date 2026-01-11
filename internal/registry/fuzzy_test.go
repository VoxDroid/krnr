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

func TestFuzzySearchCommandSets(t *testing.T) {
	// init DB
	dbConn, err := db.InitDB()
	if err != nil {
		t.Fatalf("InitDB(): %v", err)
	}
	defer func() { _ = dbConn.Close() }()

	r := NewRepository(dbConn)

	// clean state names
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

	res1, err := r.FuzzySearchCommandSets("alp")
	if err != nil {
		t.Fatalf("FuzzySearchCommandSets: %v", err)
	}
	if len(res1) == 0 || res1[0].Name != "alpha" {
		t.Fatalf("expected alpha in fuzzy search for 'alp', got: %+v", res1)
	}

	res2, err := r.FuzzySearchCommandSets("dmo")
	if err != nil {
		t.Fatalf("FuzzySearchCommandSets: %v", err)
	}
	// 'dmo' should fuzzy-match 'demo' tag or description
	found := false
	for _, s := range res2 {
		if s.Name == "beta" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected beta in fuzzy search for 'dmo', got: %+v", res2)
	}
}
