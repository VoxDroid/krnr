package nameutil

import "testing"

func TestValidateName(t *testing.T) {
	if err := ValidateName("  "); err == nil {
		t.Fatalf("expected error for empty name")
	}
	if err := ValidateName("ok name"); err != nil {
		t.Fatalf("unexpected error for valid name: %v", err)
	}
	// control char
	if err := ValidateName("bad\x00name"); err == nil {
		t.Fatalf("expected error for control bytes")
	}
	// invalid utf8 sequence
	if err := ValidateName(string([]byte{0xff, 0xff})); err == nil {
		t.Fatalf("expected error for invalid utf8")
	}
}

func TestSanitizeName(t *testing.T) {
	if s, changed := SanitizeName("hello\x00world"); s != "helloworld" || !changed {
		t.Fatalf("expected NUL removed: got %q changed=%v", s, changed)
	}
	if s, changed := SanitizeName(" a \u200B b "); s != "a  b" || !changed {
		t.Fatalf("expected zero-width removed and trimmed: got %q changed=%v", s, changed)
	}
}
