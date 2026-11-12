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
			ui.Icon(c, icon).Size(13, 13).TextColor(col).Shrink(0)
		}
		ui.Text(c, label).FontSize(12).TextColor(col).FontWeight(500)
	})
}

// ColorDot renders an anti-aliased Finder tag color dot.
func ColorDot(c *ui.Context, color ui.Color, size float32) {
	ui.Box(c).Width(size).Height(size).Radius(size / 2).Background(color).Shrink(0).IgnoreInput(true)
}

var folderMenuIcon, _ = mygo.NewMenuIconSVG(IcFolder, true)

// compactBreadcrumbs limits breadcrumb depth for long paths with an ellipsis crumb.
func compactBreadcrumbs(crumbs []domain.PathCrumb) []domain.PathCrumb {
	if len(crumbs) <= 5 {
		return crumbs
	}
	res := make([]domain.PathCrumb, 0, 5)
	res = append(res, crumbs[0], domain.PathCrumb{Name: "…", FullPath: ""})
	return append(res, crumbs[len(crumbs)-3:]...)
}

// BreadcrumbBar renders clickable path breadcrumb pills.
func BreadcrumbBar(c *ui.Context, crumbs []domain.PathCrumb, coord *app.Coordinator, onNavigate func(string)) {
	t := c.Theme()
	if len(crumbs) == 0 {
		return
	}

	ui.ScrollHorizontal(c).Gap(4).AlignItems(ui.Center).Shrink(1).MinWidth(0).Scrollbars(false).Children(func() {
		if len(crumbs) <= 5 {
			for idx, crumb := range crumbs {
				isLast := idx == len(crumbs)-1
				renderBreadcrumbItem(c, crumb, isLast, t, coord, onNavigate)
				if !isLast {
					renderBreadcrumbSeparator(c, t)
				}
			}
			return
		}

		// First crumb (Home or root)
		renderBreadcrumbItem(c, crumbs[0], false, t, coord, onNavigate)
		renderBreadcrumbSeparator(c, t)

		// Folded middle crumbs in popover menu
		folded := crumbs[1 : len(crumbs)-3]
		renderFoldedBreadcrumbButton(c, folded, t, onNavigate)
		renderBreadcrumbSeparator(c, t)

		// Tail 3 crumbs
		tail := crumbs[len(crumbs)-3:]
		for idx, crumb := range tail {
			isLast := idx == len(tail)-1
			renderBreadcrumbItem(c, crumb, isLast, t, coord, onNavigate)
			if !isLast {
				renderBreadcrumbSeparator(c, t)
			}
		}
	})
}

func renderBreadcrumbSeparator(c *ui.Context, t *ui.Theme) {
	ui.Icon(c, IcChevronRight).Size(10, 10).TextColor(t.TextMuted.Alpha(0.5)).Shrink(0)
}

func renderFoldedBreadcrumbButton(c *ui.Context, folded []domain.PathCrumb, t *ui.Theme, onNavigate func(string)) {
	btn := ui.ButtonBase(c).Padding(2, 6).Radius(4).AlignItems(ui.Center).Tooltip("中间路径")
	if btn.Hovered() {
		btn.Background(t.Text.Alpha(0.08))
	}
	btn.Children(func() {
		ui.Text(c, "…").FontSize(12).TextColor(t.TextMuted)
	})
	btn.PopoverMenuButton(func(m *ui.Menu) {
		for _, f := range folded {
			fCopy := f
			it := m.Item(fCopy.Name)
			if folderMenuIcon != nil {
				it.Icon(folderMenuIcon)
			}
			if it.Chosen() && onNavigate != nil {
				onNavigate(fCopy.FullPath)
			}
		}
	}, ui.PopoverMenuButtonOptions{At: ui.AnchorBottomStart, Self: ui.AnchorTopStart})
}

func renderBreadcrumbItem(c *ui.Context, crumb domain.PathCrumb, isLast bool, t *ui.Theme, coord *app.Coordinator, onNavigate func(string)) {
	btn := ui.ButtonBase(c).Padding(2, 6).Radius(4).AlignItems(ui.Center)
	if crumb.FullPath != "" {
		btn.Tooltip(crumb.FullPath)
	}

	isOver := false
	if coord != nil && crumb.FullPath != "" {
		isOver = BindFileDropTarget(c, btn, crumb.FullPath, coord)
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

	maxW := float32(140)
	fw := 400
	if isLast {
		maxW = 200
		fw = 600
	}

	btn.Children(func() {
		if crumb.IsHome {
			ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, IcHome).Size(12, 12).TextColor(txtCol).Shrink(0)
				ui.Text(c, crumb.Name).FontSize(12).TextColor(txtCol).MaxWidth(maxW).SingleLine().FontWeight(fw)
			})
		} else {
			ui.Text(c, crumb.Name).FontSize(12).TextColor(txtCol).MaxWidth(maxW).SingleLine().FontWeight(fw)
		}
	})

	if btn.Clicked() && onNavigate != nil && crumb.FullPath != "" {
		onNavigate(crumb.FullPath)
	}
}

// EmptyCard renders an empty state illustration with title and recovery action.
func EmptyCard(c *ui.Context, icon *ui.SVG, title, subtitle string, actionBtn func()) {
	t := c.Theme()
	ui.Column(c).Grow(1).AlignItems(ui.Center).Justify(ui.Center).Gap(10).Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(48, 48).TextColor(t.TextMuted.Alpha(0.4))
		}
		ui.Text(c, title).FontSize(15).FontWeight(600).TextColor(t.Text).Balance()
		if subtitle != "" {
			ui.Text(c, subtitle).FontSize(12).TextColor(t.TextMuted).Pretty()
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
		Shrink(1).
		MinWidth(40).
		Padding(2, 6).
		Radius(4).
		Background(t.Background).
		Border(1.5, t.Accent).
		TextColor(t.Text).
		CaretWidth(1.5).
		CaretColor(t.Accent).
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
