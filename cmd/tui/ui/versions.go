package ui

import (
	"fmt"
	"strings"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// renderVersions renders the versions panel on the details view's right side.
// It uses the versionsList view so the list adapts automatically to the size
// (same behavior as the main list).
func (m *TuiModel) renderVersions(_ int, _ int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0ea5a4")).Render("Versions") + "\n\n")

	// list view will already be sized in WindowSizeMsg, so just render it
	b.WriteString(m.versionsList.View())

	return b.String()
}

// setVersionsPreviewIndex sets the versions preview to the given index and
// updates internal state; returns a tea.Cmd if needed (nil for now).
func (m *TuiModel) setVersionsPreviewIndex(idx int) tea.Cmd {
	if idx < 0 || idx >= len(m.versions) {
		return nil
	}
	// If no change to the selection and we already have a preview set, no work needed
	if idx == m.versionsSelected && m.versionsPreviewContent != "" {
		return nil
	}
	oldIdx := m.versionsSelected
	oldContent := m.versionsPreviewContent
	content := formatVersionDetails(m.detailName, m.versions[idx], m.vp.Width)
	changed := content != oldContent || idx != oldIdx

	// Update state
	m.versionsSelected = idx
	m.versionsPreviewContent = content
	// reset scroll and set new content if it actually changed
	if changed {
		m.vp.YOffset = 0
		m.vp.SetContent(content)
	}
	return nil
}

// formatVersionDetails renders a full metadata view for a historic version.
// It is similar to the full-screen command set rendering but for a specific
// version so users can inspect author, description, created date and commands.
func formatVersionDetails(name string, v adapters.Version, _ int) string {
	var b strings.Builder
	title := fmt.Sprintf("krnr — %s v%d Details", name, v.Version)
	b.WriteString(title + "\n")
	b.WriteString(strings.Repeat("=", 30) + "\n\n")

	if v.Description != "" {
		b.WriteString("Description:\n")
		b.WriteString(v.Description + "\n\n")
	}

	if len(v.Commands) > 0 {
		b.WriteString("Commands:\n")
		for i, c := range v.Commands {
			b.WriteString(fmt.Sprintf("%d) %s\n", i+1, c))
		}
		b.WriteString("\n")
	}

	if v.AuthorName != "" {
		b.WriteString(fmt.Sprintf("Author: %s\n", v.AuthorName))
	}
	if v.CreatedAt != "" {
		b.WriteString(fmt.Sprintf("Created: %s\n", v.CreatedAt))
	}
	return b.String()
}
