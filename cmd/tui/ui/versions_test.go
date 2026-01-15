package ui

import (
	"testing"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
	"github.com/charmbracelet/bubbles/viewport"
)

func TestSetVersionsPreviewIndexUpdatesContentAndResetsOffset(t *testing.T) {
	m := &TuiModel{}
	m.vp = createTestViewport(50, 5)
	m.versions = []adapters.Version{
		{Version: 1, Commands: []string{"echo one"}, AuthorName: "a"},
		{Version: 2, Commands: []string{"echo two"}, AuthorName: "b"},
	}
	m.detailName = "one"
	// set width so content generation is deterministic
	m.vp.Width = 40
	// set initial selected and content
	m.versionsSelected = -1
	m.versionsPreviewContent = ""
	// set a non-zero offset to check reset
	m.vp.YOffset = 3
	m.setVersionsPreviewIndex(1)
	if m.versionsSelected != 1 {
		t.Fatalf("expected selected 1, got %d", m.versionsSelected)
	}
	if m.versionsPreviewContent == "" {
		t.Fatalf("expected preview content to be set")
	}
	if m.vp.YOffset != 0 {
		t.Fatalf("expected YOffset reset to 0, got %d", m.vp.YOffset)
	}
}

// helper to make a viewport for tests
func createTestViewport(w, h int) viewport.Model {
	vp := viewport.New(w, h)
	vp.YOffset = 0
	return vp
}
