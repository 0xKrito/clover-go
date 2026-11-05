package ui

import (
	"fmt"

	"github.com/bighaidao/clover-go/internal/app"
	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/egoist/mygo/ui"
)

// FinderImportModal renders the dialog for importing favorites from macOS Finder sidebar.
func FinderImportModal(c *ui.Context, coord *app.Coordinator) {
	if !coord.ShowFinderImportModal {
		return
	}

	t := c.Theme()
	pal := Pal(t)

	// Backdrop
	backdrop := ui.Box(c).Fill().Background(ui.Hex("#000000").Alpha(0.4)).AlignItems(ui.Center).Justify(ui.Center)
	backdrop.Children(func() {
		// Modal Dialog Window
		dialog := ui.Column(c).Width(480).Background(t.Background).Radius(10).Border(1, pal.Line).Padding(16).Gap(12)
		dialog.Shadow(0, 8, 24, 0, ui.Hex("#000000").Alpha(0.25))

		dialog.Children(func() {
			renderFinderImportHeader(c, coord)
			ui.Box(c).Height(1).Background(pal.Line)

			renderFinderImportBody(c, coord)

			ui.Box(c).Height(1).Background(pal.Line)
			renderFinderImportFooter(c, coord)
		})
	})
}

func renderFinderImportHeader(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "从 Finder 导入常用位置").FontSize(14).FontWeight(600).TextColor(t.Text)
		btnClose := ui.ButtonBase(c).Padding(2).Radius(4)
		btnClose.Children(func() {
			ui.Icon(c, IcClose).Size(12, 12).TextColor(t.TextMuted)
		})
		if btnClose.Clicked() {
			coord.CloseFinderImport()
		}
	})
}

func renderFinderImportBody(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	pal := Pal(t)

	ui.Text(c, "检测到 macOS 系统访达收藏列表，勾选后将自动加入 CloverGo 顶部书签栏:").FontSize(11).TextColor(t.TextMuted)

	// Selection quick toggle
	ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
		btnAll := ui.ButtonBase(c).Padding(2, 6).Radius(3).Background(pal.Line)
		btnAll.Children(func() {
			ui.Text(c, "全选").FontSize(11).TextColor(t.Text)
		})
		if btnAll.Clicked() {
			for i := range coord.FinderFavorites {
				if coord.FinderFavorites[i].Exists {
					coord.FinderFavorites[i].Selected = true
				}
			}
		}

		btnNone := ui.ButtonBase(c).Padding(2, 6).Radius(3).Background(pal.Line)
		btnNone.Children(func() {
			ui.Text(c, "取消全选").FontSize(11).TextColor(t.Text)
		})
		if btnNone.Clicked() {
			for i := range coord.FinderFavorites {
				coord.FinderFavorites[i].Selected = false
			}
		}
	})

	// Items list
	ui.Column(c).MaxHeight(220).Radius(4).Border(1, pal.Line).Padding(4).Gap(2).Children(func() {
		if len(coord.FinderFavorites) == 0 {
			ui.Text(c, "未检测到系统访达收藏项").FontSize(11).TextColor(t.TextMuted).Padding(8)
			return
		}

		for i := range coord.FinderFavorites {
			it := &coord.FinderFavorites[i]
			renderFavRow(c, it)
		}
	})
}

func renderFavRow(c *ui.Context, it *domain.FinderFavoriteItem) {
	t := c.Theme()
	row := ui.Row(c).Height(26).Padding(0, 6).Gap(8).AlignItems(ui.Center).Radius(3)
	if row.Hovered() {
		row.Background(Pal(t).Hover)
	}

	row.Children(func() {
		check := "[ ]"
		checkCol := t.TextMuted
		if it.Selected {
			check = "[✓]"
			checkCol = t.Accent
		}
		if !it.Exists {
			check = "[✕]"
			checkCol = t.TextMuted
		}

		ui.Text(c, check).FontSize(11).TextColor(checkCol)
		ui.Icon(c, IcFolder).Size(12, 12).TextColor(t.Accent)
		ui.Text(c, it.Name).FontSize(11).FontWeight(500).TextColor(t.Text).Width(110).SingleLine()

		pathCol := t.TextMuted
		pathText := it.Path
		if !it.Exists {
			pathText += " (路径不存在)"
			pathCol = ui.Hex("#FF3B30")
		}
		ui.Text(c, pathText).FontSize(10).TextColor(pathCol).Grow(1).SingleLine()
	})

	if it.Exists && row.Clicked() {
		it.Selected = !it.Selected
	}
}

func renderFinderImportFooter(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	selectedCount := 0
	for _, it := range coord.FinderFavorites {
		if it.Selected && it.Exists {
			selectedCount++
		}
	}

	// 导入失败时的错误反馈（与 PDF/水印弹窗模式一致）。
	if coord.FinderImportError != "" {
		ui.Text(c, "错误："+coord.FinderImportError).FontSize(11).TextColor(ui.Hex("#E5484D")).Wrap()
	}

	ui.Row(c).Justify(ui.End).Gap(8).AlignItems(ui.Center).Children(func() {
		btnCancel := ui.ButtonBase(c).Padding(5, 12).Radius(4).Background(Pal(t).Line)
		btnCancel.Children(func() {
			ui.Text(c, "取消").FontSize(11).TextColor(t.Text)
		})
		if btnCancel.Clicked() {
			coord.CloseFinderImport()
		}

		btnImport := ui.ButtonBase(c).Padding(5, 16).Radius(4).Background(t.Accent)
		btnImport.Children(func() {
			ui.Text(c, fmt.Sprintf("导入所选 (%d)", selectedCount)).FontSize(11).FontWeight(600).TextColor(ui.Hex("#FFFFFF"))
		})
		if btnImport.Clicked() && selectedCount > 0 {
			if err := coord.ExecuteFinderImport(); err != nil {
				coord.FinderImportError = err.Error()
			}
		}
	})
}
