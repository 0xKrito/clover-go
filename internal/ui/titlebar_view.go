package ui

import (
	"github.com/egoist/mygo/ui"
)

// TitleBarView renders the standard macOS draggable window header matching fclover.
// The entire bar is marked DragWindow without any interactive elements blocking
// pointer events, allowing the user to click-drag from any point along the top.
func TitleBarView(c *ui.Context, title string) {
	t := c.Theme()
	pal := Pal(t)

	// Left inset reserving space for native traffic lights
	leftWidth := float32(76)

	ui.Row(c).Height(28).Background(pal.Sidebar).AlignItems(ui.Center).DragWindow().Children(func() {
		// Traffic lights safe area (no interactive flags, passes drags up)
		ui.Box(c).Width(leftWidth).Height(28)

		// Centered window title (fclover)
		ui.Box(c).Grow(1).Height(28).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			ui.Text(c, title).FontSize(12).FontWeight(500).TextColor(t.TextMuted.Alpha(0.85))
		})

		// Right symmetric spacer to keep center title precisely centered
		ui.Box(c).Width(leftWidth).Height(28)
	})
}
