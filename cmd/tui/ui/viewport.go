package ui

import "github.com/charmbracelet/bubbles/viewport"

// ensureViewportSize resizes the main detail viewport preserving YOffset.
// It avoids a full reset of scrolling when size hasn't changed.
func (m *TuiModel) ensureViewportSize(width, height int) {
	if m.vp.Width != width || m.vp.Height != height {
		oldOff := m.vp.YOffset
		m.vp = viewport.New(width, height)
		m.vp.YOffset = oldOff
	}
}
