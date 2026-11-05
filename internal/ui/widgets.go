package ui

import (
	"strings"

	"github.com/bighaidao/clover-go/internal/app"
	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/bighaidao/clover-go/internal/fs"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// IconButton creates a compact toolbar icon button with smooth hover feedback.
func IconButton(c *ui.Context, icon *ui.SVG, tooltip string, disabled bool) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Label(tooltip).Disabled(disabled).Padding(4).Radius(5).AlignItems(ui.Center)
	bg := ui.Transparent
	col := t.TextMuted
	switch {
	case disabled:
		col = t.TextMuted.Alpha(0.4)
	case b.Pressed():
		bg = t.Text.Alpha(0.15)
		col = t.Text
	case b.Hovered():
		bg = t.Text.Alpha(0.08)
		col = t.Text
	}
	b.Background(bg)
	return b.Children(func() {
		ui.Icon(c, icon).Size(14, 14).TextColor(col)
	})
}

// PillButton renders a macOS capsule button with icon and text.
func PillButton(c *ui.Context, icon *ui.SVG, label string, disabled bool) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Label(label).Disabled(disabled).Padding(4, 10).Gap(6).Radius(6).AlignItems(ui.Center)
	bg := t.Text.Alpha(0.06)
	col := t.Text
	switch {
	case disabled:
		col = t.TextMuted
	case b.Pressed():
		bg = t.Text.Alpha(0.14)
	case b.Hovered():
		bg = t.Text.Alpha(0.10)
	}
	b.Background(bg)
	return b.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(13, 13).TextColor(col)
		}
		ui.Text(c, label).FontSize(12).TextColor(col).FontWeight(500)
	})
}

// ColorDot renders an anti-aliased Finder tag color dot.
func ColorDot(c *ui.Context, color ui.Color, size float32) {
	ui.Box(c).Width(size).Height(size).Radius(size / 2).Background(color).IgnoreInput(true)
}

// BreadcrumbBar renders clickable path breadcrumb pills.
func BreadcrumbBar(c *ui.Context, crumbs []domain.PathCrumb, coord *app.Coordinator, onNavigate func(string)) {
	t := c.Theme()
	displayCrumbs := crumbs
	if len(crumbs) > 4 {
		displayCrumbs = make([]domain.PathCrumb, 0, 4)
		displayCrumbs = append(displayCrumbs, crumbs[0])
		displayCrumbs = append(displayCrumbs, domain.PathCrumb{
			Name:     "…",
			FullPath: crumbs[len(crumbs)-2].FullPath,
		})
		displayCrumbs = append(displayCrumbs, crumbs[len(crumbs)-2])
		displayCrumbs = append(displayCrumbs, crumbs[len(crumbs)-1])
	}

	ui.ScrollHorizontal(c).Gap(4).AlignItems(ui.Center).Shrink(1).MinWidth(0).Scrollbars(false).Children(func() {
		for idx, crumb := range displayCrumbs {
			cCopy := crumb
			isLast := idx == len(displayCrumbs)-1
			btn := ui.ButtonBase(c).Padding(2, 6).Radius(4).AlignItems(ui.Center)

			isOver := false
			if coord != nil && !isLast && cCopy.Name != "…" {
				isOver = BindFileDropTarget(c, btn, cCopy.FullPath, coord)
			}

			txtCol := t.TextMuted
			if isOver {
				btn.Border(1.5, t.Accent).Background(Pal(t).Hover)
				txtCol = t.Accent
			} else if isLast {
				txtCol = t.Text
			} else if btn.Hovered() {
				btn.Background(t.Text.Alpha(0.08))
				txtCol = t.Text
			}
			btn.Children(func() {
				if cCopy.IsHome {
					ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, IcHome).Size(12, 12).TextColor(txtCol)
						ui.Text(c, cCopy.Name).FontSize(12).TextColor(txtCol).MaxWidth(110).SingleLine().FontWeight(func() int {
							if isLast {
								return 600
							}
							return 400
						}())
					})
				} else {
					ui.Text(c, cCopy.Name).FontSize(12).TextColor(txtCol).MaxWidth(110).SingleLine().FontWeight(func() int {
						if isLast {
							return 600
						}
						return 400
					}())
				}
			})
			if btn.Clicked() && onNavigate != nil {
				onNavigate(cCopy.FullPath)
			}

			if !isLast {
				ui.Icon(c, IcChevronRight).Size(10, 10).TextColor(t.TextMuted.Alpha(0.5))
			}
		}
	})
}

// EmptyCard renders an empty state illustration with title and recovery action.
func EmptyCard(c *ui.Context, icon *ui.SVG, title, subtitle string, actionBtn func()) {
	t := c.Theme()
	ui.Column(c).Grow(1).AlignItems(ui.Center).Justify(ui.Center).Gap(10).Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(48, 48).TextColor(t.TextMuted.Alpha(0.4))
		}
		ui.Text(c, title).FontSize(15).FontWeight(600).TextColor(t.Text)
		if subtitle != "" {
			ui.Text(c, subtitle).FontSize(12).TextColor(t.TextMuted)
		}
		if actionBtn != nil {
			ui.Box(c).Height(8)
			actionBtn()
		}
	})
}

// renderInlineRenameInput renders the inline rename field shared by the list and
// the grid: the stem is preselected, Esc cancels, Enter commits, and losing the
// focus commits as Finder does.
func renderInlineRenameInput(c *ui.Context, coord *app.Coordinator, pane *app.PaneState, item domain.FileInfo, fontSize float32) {
	t := c.Theme()
	input := ui.TextInput(c, &coord.RenameDraft).
		AutoFocus().
		Grow(1).
		MinWidth(80).
		Padding(2, 6).
		Radius(4).
		Background(t.Background).
		Border(1.5, t.Accent).
		TextColor(t.Text).
		CaretWidth(1.5).
		FontSize(fontSize).
		SingleLine()

	if coord.RenameSelectPending {
		coord.RenameSelectPending = false
		stemEnd := app.FilenameStemRunes(coord.RenameDraft, item.IsDir)
		input.SetTextSelection(0, stemEnd)
	}

	input.OnTextKey(func(k ui.TextKey) bool {
		if k.Key == ui.KeyEscape {
			coord.CancelRename()
			return true
		}
		return false
	})

	if input.Focused() {
		coord.RenameFocusedOnce = true
	} else if coord.RenameFocusedOnce {
		_, _ = coord.CommitRename(pane)
		return
	}

	if input.Submitted() {
		_, _ = coord.CommitRename(pane)
	}
}

// isItemCut reports whether path is staged on the internal clipboard by ⌘X, so
// its row can show the translucent "pending move" look Finder uses.
func isItemCut(coord *app.Coordinator, path string) bool {
	if coord == nil || coord.Clipboard.Op != app.ClipCut {
		return false
	}
	for _, p := range coord.Clipboard.Paths {
		if p == path {
			return true
		}
	}
	return false
}

// openItem opens the item: directories navigate the pane the item lives in
// (double-click in the right pane must open in the right pane); archives drill
// down in-place into virtual VFS; virtual files extract to temp for launching.
func openItem(coord *app.Coordinator, pane *app.PaneState, item domain.FileInfo) {
	if item.IsAppBundle {
		_ = mygo.Shell.OpenPath(item.Path)
		return
	}
	if fs.IsArchiveFile(item.Path) && !fs.IsArchiveVirtualPath(item.Path) {
		vPath := fs.FormatArchiveVirtualPath(item.Path, "")
		coord.Tabs.NavigatePane(coord.Tabs.ActiveIndex, pane, vPath)
		coord.LoadCurrentTab(nil)
		return
	}
	if fs.IsArchiveVirtualPath(item.Path) {
		if item.IsDir {
			coord.Tabs.NavigatePane(coord.Tabs.ActiveIndex, pane, item.Path)
			coord.LoadCurrentTab(nil)
			return
		}
		tmpPath, err := fs.ExtractArchiveFileToTemp(item.Path)
		if err == nil {
			_ = mygo.Shell.OpenPath(tmpPath)
		}
		return
	}
	if item.IsDir {
		coord.Tabs.NavigatePane(coord.Tabs.ActiveIndex, pane, item.Path)
		coord.LoadCurrentTab(nil)
		return
	}
	_ = mygo.Shell.OpenPath(item.Path)
}

func tagColorFromSlice(c *ui.Context, tags []string) ui.Color {
	t := c.Theme()
	if len(tags) == 0 {
		return TintGray.C(t)
	}
	switch strings.ToLower(tags[0]) {
	case "red", "红色":
		return TintRed.C(t)
	case "orange", "橙色":
		return TintOrange.C(t)
	case "yellow", "黄色":
		return ui.Hex("#FFCC00")
	case "green", "绿色":
		return TintGreen.C(t)
	case "blue", "蓝色":
		return TintBlue.C(t)
	case "purple", "紫色":
		return TintPurple.C(t)
	default:
		return TintGray.C(t)
	}
}
