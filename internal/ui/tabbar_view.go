package ui

import (
	"github.com/bighaidao/clover-go/internal/app"
	"github.com/egoist/mygo/ui"
)

// TabBarView renders the Clover-style multi-tab bar with window dragging and context menus.
func TabBarView(c *ui.Context, coord *app.Coordinator, onTabSelect func(int), onTabClose func(int), onNewTab func()) {
	t := c.Theme()
	pal := Pal(t)
	tm := coord.Tabs

	ui.Row(c).Height(32).Background(pal.Sidebar).AlignItems(ui.End).DragWindow().Children(func() {
		// Left margin
		ui.Box(c).Width(8).Height(32)

		// 1. Tab Items
		for i, tab := range tm.Tabs {
			idx := i
			tabState := tab
			isActive := idx == tm.ActiveIndex

			renderTab(c, coord, tabState, idx, isActive, func() {
				if onTabSelect != nil {
					onTabSelect(idx)
				}
			}, func() {
				if onTabClose != nil {
					onTabClose(idx)
				}
			})
		}

		// 2. New Tab Button (+)
		btnNew := ui.ButtonBase(c).Label("新建标签 (⌘T)").Padding(6).Radius(5).AlignItems(ui.Center)
		if btnNew.Hovered() {
			btnNew.Background(t.Text.Alpha(0.08))
		}
		btnNew.Children(func() {
			ui.Icon(c, IcPlus).Size(12, 12).TextColor(t.TextMuted)
		})
		if btnNew.Clicked() && onNewTab != nil {
			onNewTab()
		}

		// 3. Fill remaining space as window drag area
		ui.Box(c).Grow(1).Height(32).DragWindow()
	})
}

func renderTab(c *ui.Context, coord *app.Coordinator, tab *app.TabState, idx int, isActive bool, onSelect, onClose func()) {
	t := c.Theme()
	pal := Pal(t)

	btn := ui.ButtonBase(c).Radius(6, 6, 0, 0).AlignItems(ui.Center).MinHeight(32)
	isOver := BindFileDropTarget(c, btn, tab.ActivePane().Path, coord)

	bg := ui.Transparent
	txtCol := t.TextMuted
	if isOver {
		btn.Border(1.5, t.Accent)
		bg = pal.Hover
		txtCol = t.Accent
	} else if isActive {
		bg = t.Surface // Seamless page connection to toolbar/content
		txtCol = t.Text
	} else if btn.Hovered() {
		bg = pal.Hover
		txtCol = t.Text
	}
	btn.Background(bg)

	// Context menu for Tab
	btn.ContextMenu(func(m *ui.Menu) {
		pinLabel := "固定标签"
		if tab.IsPinned {
			pinLabel = "取消固定标签"
		}
		if m.Item(pinLabel).Chosen() {
			coord.Tabs.TogglePin(idx)
		}
		if m.Item("复制标签页").Chosen() {
			coord.Tabs.DuplicateTab(idx)
			coord.LoadCurrentTab(nil)
		}
		if len(coord.Tabs.Tabs) > 1 && coord.OnDetach != nil {
			if m.Item("在独立窗口中打开").Chosen() {
				coord.OnDetach(idx)
			}
		}
		m.Separator()
		if len(coord.Tabs.Tabs) > 1 {
			if m.Item("关闭其他标签页").Chosen() {
				coord.Tabs.CloseOtherTabs(idx)
				coord.LoadCurrentTab(nil)
			}
		}
		if idx < len(coord.Tabs.Tabs)-1 {
			if m.Item("关闭右侧标签页").Chosen() {
				coord.Tabs.CloseTabsToTheRight(idx)
				coord.LoadCurrentTab(nil)
			}
		}
		if !tab.IsPinned {
			if m.Item("关闭标签页 (⌘W)").Chosen() && onClose != nil {
				onClose()
			}
		}
	})

	if tab.IsPinned {
		renderPinnedTab(c, btn, txtCol, onSelect)
		return
	}

	renderNormalTab(c, btn, tab, isActive, txtCol, onSelect, onClose)
}

func renderPinnedTab(c *ui.Context, btn ui.Element, txtCol ui.Color, onSelect func()) {
	btn.Padding(6, 10).Children(func() {
		ui.Icon(c, IcFolder).Size(14, 14).TextColor(txtCol)
	})
	if btn.Clicked() && onSelect != nil {
		onSelect()
	}
}

func renderNormalTab(c *ui.Context, btn ui.Element, tab *app.TabState, isActive bool, txtCol ui.Color, onSelect, onClose func()) {
	t := c.Theme()
	btn.Padding(6, 12).Gap(8).Children(func() {
		ui.Icon(c, IcFolder).Size(13, 13).TextColor(txtCol)

		ui.Column(c).Gap(1).Children(func() {
			ui.Text(c, tab.Title).FontSize(12).FontWeight(func() int {
				if isActive {
					return 500
				}
				return 400
			}()).TextColor(txtCol).SingleLine()

			if tab.Subtitle != "" {
				ui.Text(c, tab.Subtitle).FontSize(9).TextColor(t.TextMuted).SingleLine()
			}
		})

		// Close Tab Button (x)
		btnClose := ui.ButtonBase(c).Label("关闭标签").Padding(2).Radius(3).AlignItems(ui.Center)
		if btnClose.Hovered() {
			btnClose.Background(t.Text.Alpha(0.12))
		}
		btnClose.Children(func() {
			ui.Icon(c, IcClose).Size(10, 10).TextColor(t.TextMuted)
		})
		if btnClose.Clicked() && onClose != nil {
			onClose()
		}
	})

	if btn.Clicked() && onSelect != nil {
		onSelect()
	}
}
