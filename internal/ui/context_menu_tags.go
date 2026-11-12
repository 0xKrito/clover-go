package ui

import (
	"sync"

	"github.com/bighaidao/clover-go/internal/app"
	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/bighaidao/clover-go/internal/macos"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

var (
	tagDotIcon     *mygo.MenuIcon
	tagDotIconOnce sync.Once
)

func getTagDotIcon() *mygo.MenuIcon {
	tagDotIconOnce.Do(func() {
		svgData := []byte(`<svg width="16" height="16" viewBox="0 0 16 16"><circle cx="8" cy="8" r="5" fill="currentColor"/></svg>`)
		if parsed, err := ui.ParseSVG(svgData); err == nil {
			if icon, err := mygo.NewMenuIconSVG(parsed, true); err == nil {
				tagDotIcon = icon
			}
		}
	})
	return tagDotIcon
}

func tagColor(t *ui.Theme, idx int) ui.Color {
	switch idx {
	case macos.TagRed:
		return TintRed.C(t)
	case macos.TagOrange:
		return TintOrange.C(t)
	case macos.TagYellow:
		return TintOrange.C(t).Mix(TintGreen.C(t), 0.5)
	case macos.TagGreen:
		return TintGreen.C(t)
	case macos.TagBlue:
		return TintBlue.C(t)
	case macos.TagPurple:
		return TintPurple.C(t)
	case macos.TagGray:
		return TintGray.C(t)
	default:
		return t.TextMuted
	}
}

// contextTagTrashActions renders Finder color tags submenu and trash entry.
func contextTagTrashActions(m *ui.Menu, c *ui.Context, coord *app.Coordinator, pane *app.PaneState, item domain.FileInfo) {
	m.Submenu("标记颜色", func(sub *ui.Menu) {
		tags := []struct {
			Idx  int
			Name string
		}{
			{macos.TagRed, "红色"},
			{macos.TagOrange, "橙色"},
			{macos.TagYellow, "黄色"},
			{macos.TagGreen, "绿色"},
			{macos.TagBlue, "蓝色"},
			{macos.TagPurple, "紫色"},
			{macos.TagGray, "灰色"},
		}
		currentTag := macos.ColorIndexFromTags(item.ColorTags)
		dotIcon := getTagDotIcon()
		th := c.Theme()
		for _, tg := range tags {
			tIdx := tg.Idx
			tName := tg.Name
			it := sub.Item(tName).Radio(currentTag == tIdx)
			if dotIcon != nil {
				it.Icon(dotIcon).IconColor(tagColor(th, tIdx))
			}
			if it.Chosen() {
				_ = macos.SetFileTag(item.Path, tIdx)
				applyTagToPaneItem(pane, item.Path, []string{tName})
				if coord.CurrentPreview != nil && coord.CurrentPreview.Path == item.Path {
					coord.CurrentPreview.ColorTag = tIdx
				}
				coord.Invalidate()
			}
		}
		sub.Separator()
		if sub.Item("清除颜色").Chosen() {
			_ = macos.RemoveFileTags(item.Path)
			applyTagToPaneItem(pane, item.Path, nil)
			if coord.CurrentPreview != nil && coord.CurrentPreview.Path == item.Path {
				coord.CurrentPreview.ColorTag = macos.TagNone
			}
			coord.Invalidate()
		}
	})

	if m.Item("移到废纸篓").Trailing("⌘⌫").Danger().Chosen() {
		coord.TrashSelected(pane)
	}
}

func applyTagToPaneItem(pane *app.PaneState, path string, tags []string) {
	if pane == nil {
		return
	}
	for i := range pane.Entries {
		if pane.Entries[i].Path == path {
			pane.Entries[i].ColorTags = tags
		}
	}
	for i := range pane.DisplayEntries {
		if pane.DisplayEntries[i].Path == path {
			pane.DisplayEntries[i].ColorTags = tags
		}
	}
}
