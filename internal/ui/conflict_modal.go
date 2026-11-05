package ui

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/bighaidao/clover-go/internal/app"
	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/egoist/mygo/ui"
)

// ConflictModal renders an interactive collision resolution dialog
// (Replace / Keep Both / Skip / Cancel).
func ConflictModal(c *ui.Context, coord *app.Coordinator) {
	conflict := coord.PendingConflict
	if conflict == nil {
		return
	}

	t := c.Theme()
	pal := Pal(t)
	filename := filepath.Base(conflict.DstPath)

	backdrop := ui.Box(c).Fill().Background(ui.Hex("#000000").Alpha(0.45)).AlignItems(ui.Center).Justify(ui.Center)
	backdrop.Children(func() {
		dialog := ui.Column(c).Width(440).Radius(10).Background(t.Surface).Padding(20).Gap(14)
		dialog.Shadow(0, 8, 24, 0, ui.Hex("#000000").Alpha(0.25))

		dialog.Children(func() {
			// Title Header
			ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, IcFolder).Size(16, 16).TextColor(t.Accent)
					ui.Text(c, "文件已存在").FontSize(14).FontWeight(600).TextColor(t.Text)
				})

				btnClose := ui.ButtonBase(c).Padding(2).Radius(3).AlignItems(ui.Center)
				btnClose.Children(func() {
					ui.Icon(c, IcClose).Size(12, 12).TextColor(t.TextMuted)
				})
				if btnClose.Clicked() {
					coord.ResolveConflict(app.ChoiceCancel, false)
				}
			})

			ui.Box(c).Height(1).Background(pal.Line)

			desc := fmt.Sprintf("目标目录中已存在同名项目 “%s”。您想要替换它还是保留两者？", filename)
			ui.Text(c, desc).FontSize(12).TextColor(t.Text).Wrap()

			// Comparison Box
			renderConflictComparison(c, conflict)

			// "Apply to all remaining conflicts" checkbox
			if len(conflict.Remaining) > 0 {
				btnCheck := ui.ButtonBase(c).Padding(4, 6).Radius(4).AlignItems(ui.Center).Gap(6)
				chkText := "□ 应用到后续所有冲突项"
				if coord.ConflictApplyToAll {
					chkText = "☑ 应用到后续所有冲突项"
				}
				btnCheck.Children(func() {
					ui.Text(c, chkText).FontSize(11).TextColor(t.TextMuted)
				})
				if btnCheck.Clicked() {
					coord.ConflictApplyToAll = !coord.ConflictApplyToAll
				}
			}

			// Actions Footer
			renderConflictActions(c, coord)
		})
	})
}

func renderConflictComparison(c *ui.Context, conflict *app.PendingConflict) {
	t := c.Theme()
	pal := Pal(t)
	now := time.Now()
	actionName := "移动"
	if conflict.Action == app.ActionCopy {
		actionName = "复制"
	}

	ui.Column(c).Radius(6).Background(pal.Inset).Padding(10).Gap(8).Children(func() {
		// Existing file
		ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, "现存版本:").FontSize(11).FontWeight(600).TextColor(t.TextMuted)
				info := fmt.Sprintf("%s · 修改时间: %s", domain.FormatSize(conflict.DstSize), domain.FormatModTime(conflict.DstModTime, now))
				ui.Text(c, info).FontSize(11).TextColor(t.Text)
			})
		})

		ui.Box(c).Height(1).Background(pal.Line)

		// Incoming file
		ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, "即将移入版本 ("+actionName+"):").FontSize(11).FontWeight(600).TextColor(t.Accent)
				info := fmt.Sprintf("%s · 修改时间: %s", domain.FormatSize(conflict.SrcSize), domain.FormatModTime(conflict.SrcModTime, now))
				ui.Text(c, info).FontSize(11).TextColor(t.Text)
			})
		})
	})
}

func renderConflictActions(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	ui.Row(c).Justify(ui.End).Gap(8).AlignItems(ui.Center).Children(func() {
		btnCancel := ui.ButtonBase(c).Padding(5, 12).Radius(6).AlignItems(ui.Center)
		if btnCancel.Hovered() {
			btnCancel.Background(t.Text.Alpha(0.08))
		}
		btnCancel.Children(func() {
			ui.Text(c, "取消").FontSize(12).TextColor(t.Text)
		})
		if btnCancel.Clicked() {
			coord.ResolveConflict(app.ChoiceCancel, false)
		}

		btnSkip := ui.ButtonBase(c).Padding(5, 12).Radius(6).AlignItems(ui.Center)
		if btnSkip.Hovered() {
			btnSkip.Background(t.Text.Alpha(0.08))
		}
		btnSkip.Children(func() {
			ui.Text(c, "跳过").FontSize(12).TextColor(t.Text)
		})
		if btnSkip.Clicked() {
			coord.ResolveConflict(app.ChoiceSkip, coord.ConflictApplyToAll)
		}

		btnKeepBoth := ui.ButtonBase(c).Padding(5, 12).Radius(6).AlignItems(ui.Center)
		if btnKeepBoth.Hovered() {
			btnKeepBoth.Background(t.Text.Alpha(0.08))
		}
		btnKeepBoth.Children(func() {
			ui.Text(c, "保留两者").FontSize(12).TextColor(t.Accent)
		})
		if btnKeepBoth.Clicked() {
			coord.ResolveConflict(app.ChoiceKeepBoth, coord.ConflictApplyToAll)
		}

		btnReplace := ui.ButtonBase(c).Padding(5, 14).Radius(6).Background(t.Accent).AlignItems(ui.Center)
		btnReplace.Children(func() {
			ui.Text(c, "替换").FontSize(12).FontWeight(600).TextColor(ui.Hex("#FFFFFF"))
		})
		if btnReplace.Clicked() {
			coord.ResolveConflict(app.ChoiceReplace, coord.ConflictApplyToAll)
		}
	})
}
