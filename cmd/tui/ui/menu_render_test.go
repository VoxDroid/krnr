package ui

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	adapters "github.com/VoxDroid/krnr/internal/tui/adapters"
	modelpkg "github.com/VoxDroid/krnr/internal/tui/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestMenuSingleColumnHighlightAndPadding(t *testing.T) {
	reg := &replaceFakeRegistry{items: []adapters.CommandSetSummary{{Name: "one", Description: "First"}}}
	ui := modelpkg.New(reg, &fakeExec{}, nil, nil)
	_ = ui.RefreshList(context.TODO())
	m := NewModel(ui)
	m = initTestModel(m)
	// open menu
	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = m1.(*TuiModel)
	if !m.showMenu {
		t.Fatalf("expected menu to be open")
	}
	// Render menu and validate per-item padding and selection absence of arrow markers.
	re := regexp.MustCompile("\x1b\\[[0-9;]*m")
	var widths []int
	// Fix width to a reasonable value so layout doesn't wrap unpredictably in tests.
	m.width = 80
	for i := range m.menuItems {
		m.menuIndex = i
		raw := m.renderMenu()
		plain := re.ReplaceAllString(raw, "")
		// ensure no arrow markers anywhere
		if strings.Contains(plain, "> ") {
			t.Fatalf("unexpected arrow marker in menu plain output: %q", plain)
		}
		// find the line that contains the current menu item
		lines := strings.Split(plain, "\n")
		found := false
		for _, ln := range lines {
			if strings.Contains(ln, m.menuItems[i]) {
				// remove a single leading and trailing space (we add one on both sides in render)
				trim := strings.TrimSuffix(strings.TrimPrefix(ln, " "), " ")
				widths = append(widths, utf8.RuneCountInString(trim))
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("menu item %q not found in render output: %q", m.menuItems[i], plain)
		}
	}
	// ensure all widths are equal
	for j := 1; j < len(widths); j++ {
		if widths[j] != widths[0] {
			t.Fatalf("menu lines not padded equally: %d vs %d; widths: %+v", widths[j], widths[0], widths)
		}
	}
}
