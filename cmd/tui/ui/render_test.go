package ui

import (
	"testing"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
)

func TestWrapTextAndRenderSimple(t *testing.T) {
	in := "one two three four five"
	lines := wrapText(in, 10)
	if len(lines) < 2 {
		t.Fatalf("expected wrap to create multiple lines, got %d", len(lines))
	}
}

func TestFormatCSDetailsContainsFields(t *testing.T) {
	cs := adapters.CommandSetSummary{Name: "foo", Description: "a description", Commands: []string{"echo hi"}, AuthorName: "me", Tags: []string{"t"}}
	out := formatCSDetails(cs, 60)
	if out == "" {
		t.Fatalf("expected non-empty output")
	}
	if !contains(out, "Name:") || !contains(out, "Commands:") {
		t.Fatalf("expected Name and Commands in output, got:\n%s", out)
	}
}

func TestFormatCSFullScreenTitle(t *testing.T) {
	cs := adapters.CommandSetSummary{Name: "bar", Description: "desc"}
	out := formatCSFullScreen(cs, 80, 24)
	if !contains(out, "krnr — bar Details") {
		t.Fatalf("expected title in full screen output, got:\n%s", out)
	}
}

// small helper to avoid importing strings twice in tests
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (len(s) == len(sub) && s == sub || (len(s) > len(sub) && (indexOf(s, sub) != -1)))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
