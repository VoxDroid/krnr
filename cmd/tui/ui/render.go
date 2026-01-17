package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/VoxDroid/krnr/internal/tui/adapters"
	"github.com/charmbracelet/lipgloss"
)

// simple word-wrap to produce lines no longer than width (approximate by rune count)
func wrapText(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	out := []string{}
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		cur := words[0]
		for _, w := range words[1:] {
			if utf8.RuneCountInString(cur)+1+utf8.RuneCountInString(w) > width {
				out = append(out, cur)
				cur = w
			} else {
				cur = cur + " " + w
			}
		}
		out = append(out, cur)
	}
	return out
}

// renderTwoCol renders a prefix in a fixed-width left column and wraps the
// text into the right column. Returns the joined lines.
func renderTwoCol(prefix, text string, prefixW, textW int) string {
	if prefixW < 0 {
		prefixW = 0
	}
	if textW < 0 {
		textW = 0
	}
	lines := wrapText(text, textW)
	var b strings.Builder
	for i, ln := range lines {
		var left string
		if i == 0 {
			// right-align prefix within prefixW
			padded := prefix
			if utf8.RuneCountInString(padded) < prefixW {
				padded = strings.Repeat(" ", prefixW-utf8.RuneCountInString(padded)) + padded
			}
			left = padded
		} else {
			left = strings.Repeat(" ", prefixW)
		}
		// ensure right text is not longer than textW (wrapText took care of it)
		b.WriteString(left + " " + ln + "\n")
	}
	return b.String()
}

// renderTableInline renders a label on the left and the value on the same line
// when possible. Values are wrapped to valueW and continuation lines are
// aligned under the value column.
func renderTableInline(label, value string, labelW, valueW int) string {
	if labelW < 0 {
		labelW = 0
	}
	if valueW < 0 {
		valueW = 0
	}
	lines := wrapText(value, valueW)
	var b strings.Builder
	for i, ln := range lines {
		if i == 0 {
			padded := label
			if utf8.RuneCountInString(padded) < labelW {
				padded = padded + strings.Repeat(" ", labelW-utf8.RuneCountInString(padded))
			}
			b.WriteString(padded + " " + ln + "\n")
		} else {
			b.WriteString(strings.Repeat(" ", labelW) + " " + ln + "\n")
		}
	}
	// if value is empty, still render the label alone
	if len(lines) == 0 {
		padded := label
		if utf8.RuneCountInString(padded) < labelW {
			padded = padded + strings.Repeat(" ", labelW-utf8.RuneCountInString(padded))
		}
		b.WriteString(padded + "\n")
	}
	return b.String()
}

// renderTableBlockHeader renders the label as a header line and places the
// (already wrapped) block lines underneath it, aligned to the value column.
func renderTableBlockHeader(label, block string, labelW int) string {
	if labelW < 0 {
		labelW = 0
	}
	lines := strings.Split(strings.TrimSuffix(block, "\n"), "\n")
	var b strings.Builder
	// render header
	padded := label
	if utf8.RuneCountInString(padded) < labelW {
		padded = padded + strings.Repeat(" ", labelW-utf8.RuneCountInString(padded))
	}
	b.WriteString(padded + "\n")
	// render block lines under the header
	for _, ln := range lines {
		b.WriteString(strings.Repeat(" ", labelW) + " " + ln + "\n")
	}
	return b.String()
}

// simulateOutput attempts to produce a simple dry-run output for well-known
// commands (e.g., `echo ...`). For unknown commands we fall back to the
// literal dry-run representation.
func simulateOutput(cmd string) string {
	trim := strings.TrimSpace(cmd)
	if strings.HasPrefix(trim, "echo ") {
		return strings.TrimSpace(strings.TrimPrefix(trim, "echo "))
	}
	return "$ " + cmd
}

func formatCSFullScreen(cs adapters.CommandSetSummary, width int, _ int) string {
	// Colored headings to match the main UI's visual style
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0ea5a4")).Background(lipgloss.Color("#0b1226"))
	h := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0ea5a4"))
	k := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#94a3b8"))
	dryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#94a3b8")).Italic(true)

	var b strings.Builder
	contentW := width - 6
	if contentW < 10 {
		contentW = 10
	}

	// Title header inside the container
	titleText := fmt.Sprintf("krnr — %s Details", cs.Name)
	b.WriteString(titleStyle.Render(titleText) + "\n")
	// Separator line
	sepLen := contentW
	if sepLen > len(titleText)+4 {
		sepLen = len(titleText) + 4
	}
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#0ea5a4")).Render(strings.Repeat("─", sepLen)) + "\n\n")

	// compute label column width (invisible border table)
	labels := []string{"Name:", "Description:", "Commands:", "Metadata:"}
	labelW := 0
	for _, l := range labels {
		if utf8.RuneCountInString(l) > labelW {
			labelW = utf8.RuneCountInString(l)
		}
	}
	valueW := contentW - labelW - 1
	if valueW < 10 {
		valueW = 10
	}

	// Name inline
	b.WriteString(h.Render("Name:") + " " + cs.Name + "\n")

	// Description
	appendDescription(&b, cs.Description, valueW, labelW, h)

	// Commands
	appendCommands(&b, cs.Commands, valueW, labelW, h)

	// Dry-run preview
	appendDryRunPreview(&b, cs.Commands, dryStyle)

	// Metadata
	appendMetadata(&b, cs, h, k)
	return b.String()
}

func appendDescription(b *strings.Builder, desc string, valueW, labelW int, h lipgloss.Style) {
	if desc == "" {
		return
	}
	lines := wrapText(desc, valueW)
	b.WriteString("\n")
	b.WriteString(h.Render("Description:") + "\n")
	b.WriteString(renderTableBlockHeader("", strings.Join(lines, "\n"), labelW))
}

func appendCommands(b *strings.Builder, commands []string, valueW, labelW int, h lipgloss.Style) {
	if len(commands) == 0 {
		return
	}
	b.WriteString("\n")
	b.WriteString(h.Render("Commands:") + "\n")
	maxPrefix := 0
	for i := range commands {
		p := fmt.Sprintf("%d) ", i+1)
		if l := utf8.RuneCountInString(p); l > maxPrefix {
			maxPrefix = l
		}
	}
	innerTextW := valueW - maxPrefix - 1
	if innerTextW < 10 {
		innerTextW = 10
	}
	var cb strings.Builder
	for i, c := range commands {
		p := fmt.Sprintf("%d) ", i+1)
		cb.WriteString(renderTwoCol(p, c, maxPrefix, innerTextW))
	}
	b.WriteString(renderTableBlockHeader("", strings.TrimSuffix(cb.String(), "\n"), labelW))
}

func appendDryRunPreview(b *strings.Builder, commands []string, dryStyle lipgloss.Style) {
	if len(commands) == 0 {
		return
	}
	b.WriteString("\n")
	b.WriteString(dryStyle.Render("Dry-run preview:") + "\n")
	for _, c := range commands {
		out := simulateOutput(c)
		b.WriteString(dryStyle.Render(out) + "\n")
	}
}

func appendMetadata(b *strings.Builder, cs adapters.CommandSetSummary, h, k lipgloss.Style) {
	meta := []string{}
	if cs.AuthorName != "" {
		meta = append(meta, "Author: "+cs.AuthorName)
	}
	if cs.AuthorEmail != "" {
		meta = append(meta, "Email: "+cs.AuthorEmail)
	}
	if cs.CreatedAt != "" {
		meta = append(meta, "Created: "+cs.CreatedAt)
	}
	if cs.LastRun != "" {
		meta = append(meta, "Last run: "+cs.LastRun)
	}
	if len(cs.Tags) > 0 {
		meta = append(meta, "Tags: "+strings.Join(cs.Tags, ", "))
	}
	if len(meta) > 0 {
		b.WriteString("\n")
		b.WriteString(h.Render("Metadata:") + "\n")
		for _, m := range meta {
			b.WriteString(k.Render("  "+m) + "\n")
		}
	}
}

func formatCSDetails(cs adapters.CommandSetSummary, width int) string {
	// invisible table layout — keep formatting simple and predictable for tests

	var b strings.Builder
	contentW := width - 4
	if contentW < 10 {
		contentW = 10
	}
	// label column width
	labels := []string{"Name:", "Description:", "Commands:", "Metadata:"}
	labelW := 0
	for _, l := range labels {
		if utf8.RuneCountInString(l) > labelW {
			labelW = utf8.RuneCountInString(l)
		}
	}
	valueW := contentW - labelW - 1
	if valueW < 10 {
		valueW = 10
	}

	// Name inline
	b.WriteString(renderTableInline("Name:", cs.Name, labelW, valueW))

	// Description
	if cs.Description != "" {
		appendDescription(&b, cs.Description, valueW, labelW, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0ea5a4")))
	}

	// Commands
	appendCommands(&b, cs.Commands, valueW, labelW, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0ea5a4")))

	// Metadata
	k := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#94a3b8"))
	h := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0ea5a4"))
	appendMetadata(&b, cs, h, k)
	return b.String()
}
