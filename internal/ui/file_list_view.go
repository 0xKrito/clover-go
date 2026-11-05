package ui

import (
	"time"

	"github.com/bighaidao/clover-go/internal/app"
	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/egoist/mygo/ui"
)

// FileListViewForPane renders the 4-column Finder-style table view with native selection for any specified pane.
func FileListViewForPane(c *ui.Context, coord *app.Coordinator, pane *app.PaneState) {
	t := c.Theme()
	now := time.Now()

	// clickedRow records a press taken by a row this frame, so a press on the
	// empty area around the rows can commit a pending rename and clear the
	// selection without ever stealing a row's click.
	clickedRow := false
	// clickedHead keeps header presses (sorting, resizing) from clearing the
	// selection, as in Finder.
	clickedHead := false

	container := ui.Column(c).Grow(1).FillHeight()
	BindFileDropTarget(c, container, pane.Path, coord)
	container.ContextMenu(func(m *ui.Menu) {
		ShowEmptyAreaContextMenu(m, coord, pane)
	})

	container.Children(func() {
		// Table Column Headers
		head := ui.Row(c).Height(26).Padding(0, 8).AlignItems(ui.Center).Clip()
		head.ContextMenu(func(m *ui.Menu) {
			if m.Item("创建时间").Checked(!coord.HideColCreateTime).Chosen() {
				coord.ToggleColumnVisible("create_time")
			}
			if m.Item("修改时间").Checked(!coord.HideColModTime).Chosen() {
				coord.ToggleColumnVisible("mod_time")
			}
			if m.Item("大小").Checked(!coord.HideColSize).Chosen() {
				coord.ToggleColumnVisible("size")
			}
			if m.Item("类型").Checked(!coord.HideColKind).Chosen() {
				coord.ToggleColumnVisible("kind")
			}
			m.Separator()
			if m.Item("重置所有列宽").Chosen() {
				coord.ResetAllColumns()
			}
		})
		head.Children(func() {
			renderTableHeaders(c, coord, pane)
		})
		if head.Clicked() {
			clickedHead = true
		}
		ui.Box(c).Height(1).Background(Pal(t).Line)

		minTotalW := columnMinTotalWidth(coord)

		// Table Body
		list := ui.List(c, &pane.TableState, len(pane.DisplayEntries), func(i int) {
			if i < 0 || i >= len(pane.DisplayEntries) {
				return
			}
			if renderTableRow(c, coord, pane, i, now) {
				clickedRow = true
			}
		}).Grow(1).RowWidth(minTotalW)

		head.ScrollWith(list)

		if list.Changed() && pane.SelectedIdx >= 0 && pane.SelectedIdx < len(pane.DisplayEntries) {
			coord.LoadPreview(pane.DisplayEntries[pane.SelectedIdx], nil)
		}
		if list.Submitted() && pane.SelectedIdx >= 0 && pane.SelectedIdx < len(pane.DisplayEntries) {
			if coord.RenamingIdx < 0 {
				coord.StartRename(pane, pane.SelectedIdx)
			}
		}
		if list.Shortcut(0, ui.KeySpace) {
			TriggerQuickLook(coord, pane)
		}
	})

	if container.Clicked() && !clickedRow && !clickedHead {
		// A press on the empty area below the rows: commit what was being
		// renamed, then clear the selection (Finder semantics).
		if coord.RenamingPane != nil {
			_, _ = coord.CommitRename(coord.RenamingPane)
		}
		pane.Chosen.Clear()
		pane.SelectedIdx = -1
		coord.Invalidate()
	}

	if container.Shortcut(0, ui.KeySpace) {
		TriggerQuickLook(coord, pane)
	}
}

// columnMinTotalWidth sums the name minimum and the visible metadata columns,
// so the header and the rows scroll sideways together.
func columnMinTotalWidth(coord *app.Coordinator) float32 {
	w := float32(120)
	visible := 1
	if !coord.HideColCreateTime {
		w += coord.ColWidthCreateTime
		visible++
	}
	if !coord.HideColModTime {
		w += coord.ColWidthModTime
		visible++
	}
	if !coord.HideColSize {
		w += coord.ColWidthSize
		visible++
	}
	if !coord.HideColKind {
		w += coord.ColWidthKind
		visible++
	}
	// One 8 DIP gutter after each visible column.
	return w + float32(visible)*8
}

func renderTableHeaders(c *ui.Context, coord *app.Coordinator, pane *app.PaneState) {
	nameW := float32(0)
	if coord.NameColManual {
		nameW = coord.ColWidthName
	}
	renderHeaderColumn(c, "名称", domain.SortByName, pane, coord, nameW)
	renderColDivider(c, func(dx float32) { coord.UpdateColWidth("name", dx) }, func() { coord.ResetColWidth("name") })

	if !coord.HideColCreateTime {
		renderHeaderColumn(c, "创建时间", domain.SortByCreateTime, pane, coord, coord.ColWidthCreateTime)
		renderColDivider(c, func(dx float32) { coord.UpdateColWidth("create_time", dx) }, func() { coord.ResetColWidth("create_time") })
	}
	if !coord.HideColModTime {
		renderHeaderColumn(c, "修改时间", domain.SortByModTime, pane, coord, coord.ColWidthModTime)
		renderColDivider(c, func(dx float32) { coord.UpdateColWidth("mod_time", dx) }, func() { coord.ResetColWidth("mod_time") })
	}
	if !coord.HideColSize {
		renderHeaderColumn(c, "大小", domain.SortBySize, pane, coord, coord.ColWidthSize)
		renderColDivider(c, func(dx float32) { coord.UpdateColWidth("size", dx) }, func() { coord.ResetColWidth("size") })
	}
	if !coord.HideColKind {
		renderHeaderColumn(c, "类型", domain.SortByKind, pane, coord, coord.ColWidthKind)
		renderColDivider(c, func(dx float32) { coord.UpdateColWidth("kind", dx) }, func() { coord.ResetColWidth("kind") })
	}
}

func renderColDivider(c *ui.Context, onDrag func(dx float32), onReset func()) {
	t := c.Theme()
	div := ui.Box(c).Width(8).FillHeight().Cursor(ui.CursorResizeEW).AlignItems(ui.Center).Justify(ui.Center)
	lineColor := Pal(t).Line
	if div.Hovered() {
		lineColor = t.Accent
	}
	div.Children(func() {
		ui.Box(c).Width(1).Height(14).Background(lineColor)
	})
	if dx, _, ok := div.Dragged(); ok && dx != 0 {
		onDrag(dx)
	}
	if div.DoubleClicked() && onReset != nil {
		onReset()
	}
}

func renderHeaderColumn(c *ui.Context, label string, field domain.SortField, pane *app.PaneState, coord *app.Coordinator, fixedW float32) {
	t := c.Theme()
	isActive := pane.Sort.Field == field
	arrow := ""
	if isActive {
		if pane.Sort.Direction == domain.SortAsc {
			arrow = " ▲"
		} else {
			arrow = " ▼"
		}
	}

	btn := ui.ButtonBase(c).Padding(2, 0).Radius(4).Justify(ui.Start).AlignItems(ui.Center)
	if fixedW > 0 {
		btn.Width(fixedW)
	} else {
		// The name column absorbs the room the metadata columns leave, as
		// Finder's does, until the user drags it to a width of their own.
		btn.Grow(1).Shrink(1).MinWidth(120)
	}

	btn.Children(func() {
		col := t.TextMuted
		if isActive {
			col = t.Text
		}
		ui.Text(c, label+arrow).FontSize(11).FontWeight(600).TextColor(col).SingleLine()
	})

	if btn.Clicked() {
		if pane.Sort.Field == field {
			if pane.Sort.Direction == domain.SortAsc {
				pane.Sort.Direction = domain.SortDesc
			} else {
				pane.Sort.Direction = domain.SortAsc
			}
		} else {
			pane.Sort.Field = field
			pane.Sort.Direction = domain.SortAsc
		}
		domain.SortFiles(pane.Entries, pane.Sort)
		coord.ApplyPaneFilter(pane)
		coord.Invalidate()
	}
}

// renderTableRow renders row i and reports whether the row took a click.
func renderTableRow(c *ui.Context, coord *app.Coordinator, pane *app.PaneState, i int, now time.Time) bool {
	t := c.Theme()
	item := pane.DisplayEntries[i]
	isChosen := pane.Chosen.Has(item.Path)
	isCursor := pane.SelectedIdx == i
	isRenaming := coord.RenamingPane == pane && coord.RenamingIdx == i
	// Choosen rows are filled with the theme's accent by MyGo's List row
	// itself; their text then draws in AccentText so it stays readable.
	invertText := isChosen && !isRenaming
	isCut := isItemCut(coord, item.Path)

	// 拖拽源挂在 row 上后，MyGo 会给 row 加 flagDraggable，使其成为 press 的
	// target（hitChain 取最内层 interactive 元素），List 行容器的单击选择因此
	// 失效。必须在行内再放一层 ButtonBase hit 层承接 press：单击由 hit 层上报
	// （ClickSelectRow），拖拽移动超过阈值后引擎 handPress 会把 press 从 hit 层
	// 移交给外层拖拽源，两者互不干扰。
	row := ui.Row(c).Height(26).Padding(0, 8).Radius(4).AlignItems(ui.Center)
	BindFileDragSource(c, row, coord, pane, item, i)

	isFolderDrop := false
	if item.IsDir && !item.IsAppBundle {
		isFolderDrop = BindFileDropTarget(c, row, item.Path, coord)
	}

	if isCut {
		// ⌘X 待移动的条目半透明，明确它已被剪切。
		row.Opacity(0.55)
	}

	switch {
	case isFolderDrop:
		row.Border(1.5, t.Accent).Background(Pal(t).Hover)
	case isRenaming:
		// 覆盖 List 外层的选中蓝底，重命名行保持清爽可读。
		row.Background(t.Surface)
	case invertText:
		// 选中行：不叠涂任何底色，使用 MyGo 原生 Finder 高亮蓝
		// （多选行自动连成圆角块的效果也来自外层）。
	case isCursor:
		// 键盘光标所在行但未被选中（⌘-点击取消后）：保留一层淡高亮。
		row.Background(t.Selection)
	case row.Hovered():
		row.Background(Pal(t).Hover)
	case i%2 == 1:
		row.Background(Pal(t).Stripe)
	}

	clicked := false
	row.Children(func() {
		hit := ui.ButtonBase(c).Grow(1).FillHeight().Justify(ui.Start).AlignItems(ui.Center)
		hit.Children(func() {
			renderNameCell(c, coord, pane, item, i, invertText, isRenaming)
			renderMetaCells(c, coord, item, now, invertText)
		})
		if hit.Clicked() {
			clicked = true
			// 点击其他行时先提交正在进行的重命名，避免草稿被静默丢弃。
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

	row.ContextMenu(func(m *ui.Menu) {
		if !pane.Chosen.Has(item.Path) {
			pane.Chosen.Clear()
			pane.Chosen.Add(item.Path)
			pane.SelectedIdx = i
			coord.LoadPreview(item, nil)
		}
		ShowFileContextMenu(m, c, coord, pane, item, i)
	})

	return clicked
}

func renderNameCell(c *ui.Context, coord *app.Coordinator, pane *app.PaneState, item domain.FileInfo, i int, invertText, isRenaming bool) {
	t := c.Theme()
	name := ui.Row(c).AlignItems(ui.Center).Gap(8).Clip()
	if coord.NameColManual {
		name.Width(coord.ColWidthName).Shrink(0)
	} else {
		name.Grow(1).Shrink(1).MinWidth(0)
	}

	iconCol := t.TextMuted
	textCol := t.Text
	if invertText {
		iconCol = t.AccentText
		textCol = t.AccentText
	}

	name.Children(func() {
		var bmp *ui.Bitmap
		if coord.Icons != nil {
			bmp = coord.Icons.GetOrQueue(item.Path, item.ModTime.Unix(), item.Size, 32, func() {
				coord.Invalidate()
			})
		}
		if bmp != nil {
			// 图标完全穿透鼠标，点击图标等同点击整行。
			ui.Image(c, bmp).Size(16, 16).IgnoreInput(true)
		} else {
			ic := IcFile
			if item.IsDir && !item.IsAppBundle {
				ic = IcFolder
			} else if item.IsAppBundle {
				ic = IcApp
			}
			ui.Icon(c, ic).Size(15, 15).TextColor(iconCol).IgnoreInput(true)
		}

		// Optional Finder color tag dot
		if len(item.ColorTags) > 0 {
			ColorDot(c, tagColorFromSlice(c, item.ColorTags), 7)
		}

		if isRenaming {
			renderInlineRenameInput(c, coord, pane, item, 12)
		} else {
			ui.Text(c, item.Name).FontSize(12).TextColor(textCol).Grow(1).Shrink(1).MinWidth(0).SingleLine()
		}
	})

	ui.Box(c).Width(8)
}

func renderMetaCells(c *ui.Context, coord *app.Coordinator, item domain.FileInfo, now time.Time, invertText bool) {
	t := c.Theme()
	col := t.TextMuted
	if invertText {
		// 深蓝选底上的高对比度亮白，替代看不清的灰色。
		col = t.AccentText.Alpha(0.92)
	}

	emit := func(text string, w float32) {
		ui.Text(c, text).FontSize(11).TextColor(col).Width(w).SingleLine().FontFeatures("tnum")
		ui.Box(c).Width(8)
	}

	if !coord.HideColCreateTime {
		createTimeText := domain.FormatModTime(item.CreateTime, now)
		if item.CreateTime.IsZero() {
			// Archive VFS entries carry no birthtime; show the mod time instead of
			// a bare placeholder so the column stays informative.
			createTimeText = domain.FormatModTime(item.ModTime, now)
		}
		emit(createTimeText, coord.ColWidthCreateTime)
	}
	if !coord.HideColModTime {
		emit(domain.FormatModTime(item.ModTime, now), coord.ColWidthModTime)
	}
	if !coord.HideColSize {
		sz := "--"
		if !item.IsDir || item.IsAppBundle {
			sz = domain.FormatSize(item.Size)
		}
		emit(sz, coord.ColWidthSize)
	}
	if !coord.HideColKind {
		emit(item.Kind, coord.ColWidthKind)
	}
}
