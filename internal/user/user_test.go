package user

import (
	"testing"
)

func TestProfileSetGetClear(t *testing.T) {
	p := Profile{Name: "Alice", Email: "alice@example.com"}
	if err := SetProfile(p); err != nil {
		t.Fatalf("SetProfile: %v", err)
	}
	p2, ok, err := GetProfile()
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if !ok {
		t.Fatalf("expected profile to exist")
	}
	if p2.Name != p.Name || p2.Email != p.Email {
		t.Fatalf("unexpected profile: %+v", p2)
	}
	if err := ClearProfile(); err != nil {
		t.Fatalf("ClearProfile: %v", err)
	}
	_, ok, err = GetProfile()
	if err != nil {
		t.Fatalf("GetProfile after clear: %v", err)
	}
	if ok {
		t.Fatalf("expected profile to be cleared")
	}
}
