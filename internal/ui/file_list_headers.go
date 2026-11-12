package ui

import (
	"github.com/bighaidao/clover-go/internal/app"
	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/egoist/mygo/ui"
)

func renderTableHeaders(c *ui.Context, coord *app.Coordinator, pane *app.PaneState) {
	nameW := float32(0)
	if coord.NameColManual {
		nameW = coord.ColWidthName
	}
	renderHeaderColumn(c, "名称", domain.SortByName, pane, coord, nameW)
	renderColDivider(c, "name", coord, func() { coord.ResetColWidth("name") })

	if !coord.HideColCreateTime {
		renderHeaderColumn(c, "创建时间", domain.SortByCreateTime, pane, coord, coord.ColWidthCreateTime)
		renderColDivider(c, "create_time", coord, func() { coord.ResetColWidth("create_time") })
	}
	if !coord.HideColModTime {
		renderHeaderColumn(c, "修改时间", domain.SortByModTime, pane, coord, coord.ColWidthModTime)
		renderColDivider(c, "mod_time", coord, func() { coord.ResetColWidth("mod_time") })
	}
	if !coord.HideColSize {
		renderHeaderColumn(c, "大小", domain.SortBySize, pane, coord, coord.ColWidthSize)
		renderColDivider(c, "size", coord, func() { coord.ResetColWidth("size") })
	}
	if !coord.HideColKind {
		renderHeaderColumn(c, "类型", domain.SortByKind, pane, coord, coord.ColWidthKind)
		renderColDivider(c, "kind", coord, func() { coord.ResetColWidth("kind") })
	}
}

func renderColDivider(c *ui.Context, col string, coord *app.Coordinator, onReset func()) {
	t := c.Theme()
	div := ui.Box(c).Width(8).FillHeight().Cursor(ui.CursorResizeEW).AlignItems(ui.Center).Justify(ui.Center).ZIndex(2)
	lineColor := Pal(t).Line
	if div.Hovered() {
		lineColor = t.Accent
	}
	div.Children(func() {
		ui.Box(c).Width(1).Height(14).Background(lineColor)
	})
	if dx, _, ok := div.Dragged(); ok && dx != 0 {
		coord.DragColWidth(col, dx)
	}
	if _, _, ok := div.DragEnded(); ok {
		coord.EndColDrag()
	}
	if div.PressCanceled() {
		coord.CancelColDrag(col)
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
		toggleColumnSort(pane, field, coord)
	}
}

func toggleColumnSort(pane *app.PaneState, field domain.SortField, coord *app.Coordinator) {
	pane.TableState.HoldPlace()
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
