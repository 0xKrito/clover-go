package ui

import (
	"github.com/bighaidao/clover-go/internal/app"
	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/egoist/mygo/ui"
)

// IconGridViewForPane renders the virtualized grid view of items for any specified pane.
func IconGridViewForPane(c *ui.Context, coord *app.Coordinator, pane *app.PaneState) {
	n := len(pane.DisplayEntries)
	if n == 0 {
		return
	}

	container := ui.Column(c).Grow(1).FillHeight()
	BindFileDropTarget(c, container, pane.Path, coord)
	container.ContextMenu(func(m *ui.Menu) {
		ShowEmptyAreaContextMenu(m, coord, pane)
	})

	iconSize := float32(coord.GridIconSize)
	cellW := iconSize + 28
	cellH := iconSize + 44

	container.Children(func() {
		grid := ui.GridView(c, &pane.GridState, n, cellW, cellH, func(i int) {
			if i < 0 || i >= n {
				return
			}
			item := pane.DisplayEntries[i]
			isSelected := pane.Chosen.Has(item.Path) || pane.SelectedIdx == i

			renderGridCell(c, coord, pane, item, i, isSelected, iconSize)
		}).Grow(1)

		if grid.Changed() && pane.SelectedIdx >= 0 && pane.SelectedIdx < n {
			coord.LoadPreview(pane.DisplayEntries[pane.SelectedIdx], nil)
		}
		if grid.Submitted() && pane.SelectedIdx >= 0 && pane.SelectedIdx < n {
			if coord.RenamingIdx < 0 {
				coord.StartRename(pane, pane.SelectedIdx)
			}
		}
		if grid.Shortcut(0, ui.KeySpace) {
			TriggerQuickLook(coord, pane)
		}
	})

	if container.Shortcut(0, ui.KeySpace) {
		TriggerQuickLook(coord, pane)
	}
}

func renderGridCell(c *ui.Context, coord *app.Coordinator, pane *app.PaneState, item domain.FileInfo, i int, isSelected bool, iconSize float32) {
	t := c.Theme()
	isRenaming := coord.RenamingPane == pane && coord.RenamingIdx == i
	// 同 file_list_view.go：拖拽源让 cell 成为 press target，
	// 必须内嵌 ButtonBase hit 层承接单击/双击。
	cell := ui.Column(c).Padding(6).Radius(6).AlignItems(ui.Center).Gap(6)
	BindFileDragSource(c, cell, coord, pane, item, i)

	isFolderDrop := false
	if item.IsDir && !item.IsAppBundle {
		isFolderDrop = BindFileDropTarget(c, cell, item.Path, coord)
	}

	if isItemCut(coord, item.Path) {
		// ⌘X 待移动的条目半透明。
		cell.Opacity(0.55)
	}

	if isFolderDrop {
		cell.Border(1.5, t.Accent).Background(Pal(t).Hover)
	} else if isRenaming {
		// 重命名中：柔和浅底，避免图标和输入框落在选中色上。
		cell.Background(t.Surface)
	} else if isSelected {
		cell.Background(t.Selection)
	} else if cell.Hovered() {
		cell.Background(Pal(t).Hover)
	}

	cell.Children(func() {
		hit := ui.ButtonBase(c).Grow(1).FillHeight()
		hit.Children(func() {
			ui.Column(c).AlignItems(ui.Center).Gap(6).Grow(1).Children(func() {
				renderGridIcon(c, coord, item, isSelected, iconSize)
				renderGridLabel(c, coord, pane, item, i, isSelected)
			})
		})
		if hit.Clicked() {
			// 点击其他格子时先提交正在进行的重命名。
			if coord.RenamingPane != nil && (coord.RenamingPane != pane || coord.RenamingIdx != i) {
				_, _ = coord.CommitRename(coord.RenamingPane)
			}
			pane.ClickSelectRow(i, hit.ClickModifiers())
			coord.LoadPreview(item, nil)
			coord.Invalidate()
		}
		if hit.DoubleClicked() {
			openItem(coord, pane, item)
		}
		if hit.Shortcut(0, ui.KeySpace) {
			TriggerQuickLook(coord, pane)
		}
		if coord.RenamingIdx < 0 && hit.Shortcut(0, ui.KeyEnter) {
			coord.StartRename(pane, i)
		}
	})

	cell.ContextMenu(func(m *ui.Menu) {
		if !pane.Chosen.Has(item.Path) {
			pane.Chosen.Clear()
			pane.Chosen.Add(item.Path)
			pane.SelectedIdx = i
			coord.LoadPreview(item, nil)
		}
		ShowFileContextMenu(m, c, coord, pane, item, i)
	})
}

func renderGridIcon(c *ui.Context, coord *app.Coordinator, item domain.FileInfo, isSelected bool, iconSize float32) {
	t := c.Theme()
	box := ui.Box(c).Size(iconSize, iconSize).AlignItems(ui.Center).Justify(ui.Center)

	box.Children(func() {
		px := int(iconSize * 2)
		var bmp *ui.Bitmap
		if coord.Icons != nil {
			bmp = coord.Icons.GetOrQueue(item.Path, item.ModTime.Unix(), item.Size, px, func() {
				coord.Invalidate()
			})
		}

		if bmp != nil {
			// 图标穿透鼠标，点击图标等同点击整格。
			ui.Image(c, bmp).Size(iconSize, iconSize).Fit(ui.Contain).IgnoreInput(true)
		} else {
			ic := IcFile
			if item.IsDir && !item.IsAppBundle {
				ic = IcFolder
			} else if item.IsAppBundle {
				ic = IcApp
			}
			col := t.TextMuted
			if isSelected {
				col = t.Accent
			}
			ui.Icon(c, ic).Size(iconSize*0.65, iconSize*0.65).TextColor(col).IgnoreInput(true)
		}
	})
}

func renderGridLabel(c *ui.Context, coord *app.Coordinator, pane *app.PaneState, item domain.FileInfo, i int, isSelected bool) {
	if coord.RenamingPane == pane && coord.RenamingIdx == i {
		// 与列表视图共用同一套内联重命名（主干选中 / Esc 取消 / 失焦提交）。
		renderInlineRenameInput(c, coord, pane, item, 11)
		return
	}
	// GridView 的选中格自身绘制 Accent 淡底与描边（不是实心 Accent 块），
	// 因此文字仍用主题文本色，才能保证浅色/深色两套主题下的对比度。
	t := c.Theme()
	txtCol := t.Text
	if isSelected {
		txtCol = t.Accent
	}
	ui.Text(c, item.Name).FontSize(11).TextColor(txtCol).Lines(1, 2).TextAlign(ui.Center)
}
