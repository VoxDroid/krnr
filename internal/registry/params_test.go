package registry

import "testing"

func TestFindAndApplyParams(t *testing.T) {
	s := "echo Hello {{name}} and {{who}}"
	ps := FindParams(s)
	if len(ps) != 2 {
		t.Fatalf("expected 2 params, got %d", len(ps))
	}
	m := map[string]string{"name": "Alice", "who": "Bob"}
	r, err := ApplyParams(s, m)
	if err != nil {
		t.Fatalf("ApplyParams error: %v", err)
	}
	if r != "echo Hello Alice and Bob" {
		t.Fatalf("unexpected result: %s", r)
	}

	// missing param
	_, err = ApplyParams(s, map[string]string{"name": "Alice"})
	if err == nil {
		t.Fatalf("expected error for missing param")
	}
}
