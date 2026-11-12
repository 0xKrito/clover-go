package ui

import (
	"fmt"

	"github.com/bighaidao/clover-go/internal/app"
	"github.com/bighaidao/clover-go/internal/config"
	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/egoist/mygo/ui"
)

// RenderModals renders active modal dialogs (batch rename, settings, pdf, watermark, finder import, workspaces).
func RenderModals(c *ui.Context, coord *app.Coordinator, activeTab *app.TabState) {
	// 弹窗必须挂在 mygo 的 overlay 层（ui.Overlay + Absolute 全屏 backdrop）。
	// 直接作为根 Column 的流式子元素会把弹窗排到状态栏之后、窗口可视区之外，
	// 导致设置等弹窗可见区域错乱、点击无响应。
	if coord.ShowConflictModal {
		renderModalOverlay(c, func() { coord.ResolveConflict(app.ChoiceCancel, false) }, func() { ConflictModal(c, coord) })
		return
	}
	if coord.ShowAboutModal {
		renderModalOverlay(c, func() { coord.ShowAboutModal = false; coord.Invalidate() }, func() { AboutModal(c, coord) })
		return
	}
	if coord.ShowSettingsModal {
		renderModalOverlay(c, func() { coord.CloseSettings() }, func() { SettingsModal(c, coord) })
		return
	}
	if coord.ShowBatchRenameModal {
		renderModalOverlay(c, func() { coord.CloseBatchRename() }, func() { renderBatchRenameModal(c, coord) })
		return
	}
	if coord.ShowPDFModal {
		renderModalOverlay(c, func() { coord.ClosePDFTools() }, func() { PDFToolsModal(c, coord) })
		return
	}
	if coord.ShowWatermarkModal {
		renderModalOverlay(c, func() { coord.CloseWatermark() }, func() { WatermarkModal(c, coord) })
		return
	}
	if coord.ShowFinderImportModal {
		renderModalOverlay(c, func() { coord.CloseFinderImport() }, func() { FinderImportModal(c, coord) })
		return
	}
	if coord.ShowWorkspacesModal {
		renderModalOverlay(c, func() { coord.ShowWorkspacesModal = false; coord.Invalidate() }, func() { WorkspacesModal(c, coord) })
		return
	}
}

// renderModalOverlay mounts a full-screen modal backdrop on the overlay layer,
// following the official mygo modal pattern (TestModalOverlayOfYourOwn):
// Absolute().Left(0).Top(0).Right(0).Bottom(0).Modal() covers the whole window
// and marks the dialog as modal (focus trap + a11y shielding).
// 支持通过 back.OverlayShortcut 捕获 Esc 快捷键关闭弹窗。
func renderModalOverlay(c *ui.Context, onDismiss func(), content func()) {
	ui.Overlay(c, func() {
		back := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Modal()
		if onDismiss != nil && back.OverlayShortcut(0, ui.KeyEscape) {
			onDismiss()
		}
		back.Children(content)
	})
}

// renderBatchRenameModal draws the multi-pattern rename dialog.
func renderBatchRenameModal(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	pal := Pal(t)

	// Modal Backdrop overlay
	backdrop := ui.Box(c).Fill().Background(ui.Hex("#000000").Alpha(0.4)).AlignItems(ui.Center).Justify(ui.Center)
	backdrop.Children(func() {
		// Modal Dialog Box
		dialog := ui.Column(c).Width(480).Radius(10).Background(t.Surface).Padding(20).Gap(14)
		dialog.Shadow(0, 8, 24, 0, ui.Hex("#000000").Alpha(0.25))

		dialog.Children(func() {
			renderBatchRenameHeader(c, coord)
			ui.Box(c).Height(1).Background(pal.Line)

			renderModeSelector(c, coord)
			renderModeInputs(c, coord)
			renderBatchRenamePreviewTable(c, coord)

			if coord.BatchRenameError != "" {
				ui.Text(c, "错误："+coord.BatchRenameError).FontSize(11).TextColor(ui.Hex("#E5484D")).Wrap()
			}

			renderBatchRenameFooter(c, coord)
		})
	})
}

func renderBatchRenameHeader(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
		title := fmt.Sprintf("批量重命名 (%d 个项目)", len(coord.BatchRenameItems))
		ui.Text(c, title).FontSize(14).FontWeight(600).TextColor(t.Text)

		btnClose := ui.ButtonBase(c).Padding(2).Radius(3).AlignItems(ui.Center)
		btnClose.Children(func() {
			ui.Icon(c, IcClose).Size(12, 12).TextColor(t.TextMuted)
		})
		if btnClose.Clicked() {
			coord.CloseBatchRename()
		}
	})
}

func renderBatchRenameFooter(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	ui.Row(c).Justify(ui.End).Gap(10).Children(func() {
		btnCancel := ui.ButtonBase(c).Padding(6, 14).Radius(6).AlignItems(ui.Center)
		if btnCancel.Hovered() {
			btnCancel.Background(t.Text.Alpha(0.08))
		}
		btnCancel.Children(func() {
			ui.Text(c, "取消").FontSize(12).TextColor(t.Text)
		})
		if btnCancel.Clicked() {
			coord.CloseBatchRename()
		}

		btnOk := ui.ButtonBase(c).Padding(6, 16).Radius(6).Background(t.Accent).AlignItems(ui.Center)
		btnOk.Children(func() {
			ui.Text(c, "重命名").FontSize(12).FontWeight(600).TextColor(ui.Hex("#FFFFFF"))
		})
		if btnOk.Clicked() {
			_ = coord.ExecuteBatchRename()
		}
	})
}

// renderSegmentedRow draws "label [opt][opt][opt]" style segmented choice.
func renderSegmentedRow(c *ui.Context, label string, values, labels []string, current string, onPick func(string)) {
	t := c.Theme()
	ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(t.Text).Width(180)
		ui.Row(c).Gap(4).Children(func() {
			for i, v := range values {
				val := v
				isActive := current == val
				btn := ui.ButtonBase(c).Padding(4, 10).Radius(5).AlignItems(ui.Center)
				if isActive {
					btn.Background(t.Accent)
				} else if btn.Hovered() {
					btn.Background(t.Text.Alpha(0.08))
				}
				btn.Children(func() {
					col := t.Text
					if isActive {
						col = ui.Hex("#FFFFFF")
					}
					ui.Text(c, labels[i]).FontSize(11).TextColor(col)
				})
				if btn.Clicked() {
					onPick(val)
				}
			}
		})
	})
}

// renderCheckRow draws "label [开启/关闭]" toggle row as a stateless button;
// persistence goes through the apply() callback (Prefs.Save).
func renderCheckRow(c *ui.Context, label string, checked bool, onToggle func(bool)) {
	t := c.Theme()
	ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(t.Text).Grow(1)
		btn := ui.ButtonBase(c).Padding(2, 12).Radius(10).AlignItems(ui.Center)
		if checked {
			btn.Background(t.Accent)
		} else {
			btn.Background(t.Text.Alpha(0.15))
		}
		btn.Children(func() {
			txt, col := "关闭", t.TextMuted
			if checked {
				txt, col = "开启", ui.Hex("#FFFFFF")
			}
			ui.Text(c, txt).FontSize(11).TextColor(col)
		})
		if btn.Clicked() {
			onToggle(!checked)
		}
	})
}

// renderIconSizeRow draws fixed icon size choices for grid view.
func renderIconSizeRow(c *ui.Context, coord *app.Coordinator, prefs config.Preferences, apply func(func(*config.Preferences))) {
	t := c.Theme()
	sizes := []int{64, 72, 88, 104, 128}
	ui.Row(c).Gap(4).Children(func() {
		for _, s := range sizes {
			size := s
			isActive := prefs.GridIconSize == size
			btn := ui.ButtonBase(c).Padding(4, 10).Radius(5).AlignItems(ui.Center)
			if isActive {
				btn.Background(t.Accent)
			} else if btn.Hovered() {
				btn.Background(t.Text.Alpha(0.08))
			}
			btn.Children(func() {
				col := t.Text
				if isActive {
					col = ui.Hex("#FFFFFF")
				}
				ui.Text(c, fmt.Sprintf("%dpx", size)).FontSize(11).TextColor(col)
			})
			if btn.Clicked() {
				apply(func(p *config.Preferences) { p.GridIconSize = size })
				coord.GridIconSize = size
			}
		}
	})
}

func renderModeSelector(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	modes := []struct {
		Mode  domain.RenameMode
		Label string
	}{
		{domain.ModeSequential, "序号排序"},
		{domain.ModeReplace, "替换文本"},
		{domain.ModeAddPrefix, "添加前缀"},
		{domain.ModeAddSuffix, "添加后缀"},
	}

	ui.Row(c).Gap(6).Children(func() {
		for _, m := range modes {
			item := m
			isActive := coord.BatchRenameMode == item.Mode
			btn := ui.ButtonBase(c).Padding(4, 10).Radius(5).AlignItems(ui.Center)
			if isActive {
				btn.Background(t.Accent)
			} else if btn.Hovered() {
				btn.Background(t.Text.Alpha(0.08))
			}
			btn.Children(func() {
				col := t.TextMuted
				if isActive {
					col = ui.Hex("#FFFFFF")
				}
				ui.Text(c, item.Label).FontSize(11).TextColor(col)
			})
			if btn.Clicked() {
				coord.BatchRenameMode = item.Mode
			}
		}
	})
}

func renderModeInputs(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	switch coord.BatchRenameMode {
	case domain.ModeSequential:
		ui.Column(c).Gap(6).Children(func() {
			ui.Text(c, "基准名称:").FontSize(11).TextColor(t.TextMuted)
			ui.TextInput(c, &coord.BatchRenameBase).FillWidth().FontSize(12).Placeholder("例如：photo")
		})
	case domain.ModeReplace:
		ui.Column(c).Gap(6).Children(func() {
			ui.Text(c, "查找文本:").FontSize(11).TextColor(t.TextMuted)
			ui.TextInput(c, &coord.BatchRenameFind).FillWidth().FontSize(12).Placeholder("要被替换的文字")
			ui.Text(c, "替换为:").FontSize(11).TextColor(t.TextMuted)
			ui.TextInput(c, &coord.BatchRenameReplace).FillWidth().FontSize(12).Placeholder("新的文字")
		})
	case domain.ModeAddPrefix:
		ui.Column(c).Gap(6).Children(func() {
			ui.Text(c, "前缀:").FontSize(11).TextColor(t.TextMuted)
			ui.TextInput(c, &coord.BatchRenamePrefix).FillWidth().FontSize(12).Placeholder("例如：[2026]_")
		})
	case domain.ModeAddSuffix:
		ui.Column(c).Gap(6).Children(func() {
			ui.Text(c, "后缀:").FontSize(11).TextColor(t.TextMuted)
			ui.TextInput(c, &coord.BatchRenameSuffix).FillWidth().FontSize(12).Placeholder("例如：_draft")
		})
	}
}

func renderBatchRenamePreviewTable(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	pal := Pal(t)

	// Single source of truth: the exact option set the executor uses.
	pairs, err := domain.CalculateBatchRename(coord.BatchRenameItems, coord.BatchRenameOptions())

	ui.Column(c).MaxHeight(130).Radius(6).Background(pal.Inset).Padding(8, 10).Gap(4).Children(func() {
		ui.Row(c).Height(18).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "原文件名").FontSize(10).FontWeight(600).TextColor(t.TextMuted).Grow(1).Shrink(1).MinWidth(0).SingleLine()
			ui.Text(c, "→").FontSize(10).TextColor(t.TextMuted).Shrink(0)
			ui.Text(c, "重命名为").FontSize(10).FontWeight(600).TextColor(t.TextMuted).Grow(1).Shrink(1).MinWidth(0).SingleLine()
		})
		ui.Box(c).Height(1).Background(pal.Line)

		if err != nil {
			// Rule conflicts (duplicate targets, invalid names) render inline
			// so the preview can never silently show stale pairs.
			ui.Text(c, "错误："+err.Error()).FontSize(11).TextColor(ui.Hex("#E5484D")).Wrap()
			return
		}

		limit := len(pairs)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			p := pairs[i]
			ui.Row(c).Height(18).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, p.OldName).FontSize(11).TextColor(t.TextMuted).Grow(1).Shrink(1).MinWidth(0).SingleLine().Tooltip(p.OldName)
				ui.Text(c, "→").FontSize(10).TextColor(t.Accent).Shrink(0)
				newColor := t.Text
				if p.Changed {
					newColor = t.Accent
				}
				ui.Text(c, p.NewName).FontSize(11).TextColor(newColor).Grow(1).Shrink(1).MinWidth(0).SingleLine().Tooltip(p.NewName)
			})
		}
		if len(pairs) > limit {
			ui.Text(c, fmt.Sprintf("... 另有 %d 个项目", len(pairs)-limit)).FontSize(10).TextColor(t.TextMuted)
		}
	})
}
