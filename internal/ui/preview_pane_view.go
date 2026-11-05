package ui

import (
	"fmt"
	"path/filepath"

	"github.com/bighaidao/clover-go/internal/app"
	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/bighaidao/clover-go/internal/macos"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
)

// PreviewPaneView renders the right-side inspector panel (⇧⌘P / ⌘I).
func PreviewPaneView(c *ui.Context, coord *app.Coordinator, activeTab *app.TabState) {
	t := c.Theme()
	pal := Pal(t)
	// The inspector mirrors the ACTIVE pane's selection (dual-pane aware).
	pane := activeTab.ActivePane()

	pWidth := coord.PreviewWidth
	if pWidth < 180 || pWidth > 500 {
		pWidth = 250
	}

	resizer := ui.Box(c).Absolute().Left(-4).Top(0).Width(8).FillHeight().Cursor(ui.CursorResizeEW)
	if dx, _, ok := resizer.Dragged(); ok && dx != 0 {
		newW := coord.PreviewWidth - dx
		if newW < 180 {
			newW = 180
		} else if newW > 500 {
			newW = 500
		}
		coord.PreviewWidth = newW
		p := coord.Prefs.Get()
		p.PreviewWidth = newW
		_ = coord.Prefs.Save(p)
	}

	ui.Column(c).Width(pWidth).FillHeight().Background(pal.Sidebar).Padding(12, 10).Gap(10).Children(func() {
		preview := coord.CurrentPreview
		if preview == nil || pane.SelectedIdx < 0 {
			renderFolderOverview(c, coord, pane)
			return
		}

		if preview.IsMarkdown {
			renderMarkdownPane(c, coord, pane, preview, pWidth)
			return
		}

		renderHeaderPreview(c, coord, preview, pWidth)
		ui.Box(c).Height(1).Background(pal.Line)

		renderMetadataSection(c, preview)
		ui.Box(c).Height(1).Background(pal.Line)

		renderColorTagsPicker(c, coord, pane, preview)
		ui.Box(c).Height(1).Background(pal.Line)

		renderQuickActions(c, coord, preview)

		if !preview.IsBinary && preview.Snippet != "" {
			ui.Box(c).Height(1).Background(pal.Line)
			renderSnippetPreview(c, coord, preview)
		}
	})
}

func renderFolderOverview(c *ui.Context, coord *app.Coordinator, pane *app.PaneState) {
	t := c.Theme()
	title := filepath.Base(pane.Path)
	if pane.Path == "/" {
		title = "Macintosh HD"
	}
	ui.Column(c).AlignItems(ui.Center).Gap(8).Padding(20, 0).Children(func() {
		var bmp *ui.Bitmap
		if coord.Icons != nil {
			bmp = coord.Icons.GetOrQueue(pane.Path, 0, 0, 128, func() {
				c.Services().Invalidate()
			})
		}
		if bmp != nil {
			ui.Image(c, bmp).Size(64, 64).Fit(ui.Contain)
		} else {
			ui.Icon(c, IcFolder).Size(56, 56).TextColor(t.Accent)
		}
		ui.Text(c, title).FontSize(14).FontWeight(600).TextColor(t.Text)
		ui.Text(c, domain.CollapseHome(pane.Path, "")).FontSize(11).TextColor(t.TextMuted).SingleLine()
		ui.Text(c, fmtItemCount(len(pane.DisplayEntries))).FontSize(11).TextColor(t.TextMuted)
	})
}

func fmtItemCount(n int) string {
	return fmt.Sprintf("%d 个项目", n)
}

func renderHeaderPreview(c *ui.Context, coord *app.Coordinator, prev *app.PreviewSnapshot, pWidth float32) {
	t := c.Theme()
	pal := Pal(t)

	// High-resolution canvas for rich file preview
	canvasW := pWidth - 20
	if canvasW < 160 {
		canvasW = 160
	}
	canvasH := float32(160)

	btnCanvas := ui.ButtonBase(c).Width(canvasW).Height(canvasH).Radius(6).Background(pal.Inset).AlignItems(ui.Center).Justify(ui.Center)
	if btnCanvas.Hovered() {
		btnCanvas.Background(pal.Hover)
	}

	btnCanvas.Children(func() {
		if coord.CurrentVideoPlayer != nil && coord.CurrentVideoPath == prev.Path {
			ui.Video(c, coord.CurrentVideoPlayer).Size(canvasW-16, canvasH-24).Fit(ui.Contain)
		} else {
			bmp := coord.Icons.GetOrQueue(prev.Path, prev.ModTimeUnix, prev.Size, 512, func() {
				c.Services().Invalidate()
			})
			if bmp != nil {
				ui.Image(c, bmp).Size(canvasW-16, canvasH-24).Fit(ui.Contain)
			} else {
				ic := IcFile
				if prev.Kind == "文件夹" {
					ic = IcFolder
				}
				ui.Icon(c, ic).Size(64, 64).TextColor(t.Accent)
			}
		}
	})

	// Clicking canvas invokes native in-process QLPreviewPanel
	if btnCanvas.Clicked() {
		macos.ToggleQuickLook(prev.Path)
	}

	ui.Row(c).Justify(ui.Center).Children(func() {
		ui.Text(c, "点击或按空格快速查看").FontSize(10).TextColor(t.TextMuted.Alpha(0.6))
	})

	ui.Column(c).AlignItems(ui.Center).Gap(2).Children(func() {
		ui.Text(c, prev.Name).FontSize(13).FontWeight(600).TextColor(t.Text).Lines(1, 2).TextAlign(ui.Center)

		subText := prev.Kind
		if prev.IsGIF {
			subText = "动态动图"
			if prev.ImageDimensions != "" {
				subText = fmt.Sprintf("动态动图 · %s", prev.ImageDimensions)
			}
		} else if prev.PDFPages > 0 {
			subText = fmt.Sprintf("%s · 共 %d 页", prev.Kind, prev.PDFPages)
		} else if prev.ImageDimensions != "" {
			subText = fmt.Sprintf("%s · %s", prev.Kind, prev.ImageDimensions)
		}
		ui.Text(c, subText).FontSize(11).TextColor(t.TextMuted)
	})
}

func renderMetadataSection(c *ui.Context, prev *app.PreviewSnapshot) {
	ui.Column(c).Gap(4).Children(func() {
		metaRow(c, "大小", domain.FormatSize(prev.Size))
		if prev.ImageDimensions != "" {
			metaRow(c, "尺寸", prev.ImageDimensions)
		}
		if prev.PDFPages > 0 {
			metaRow(c, "页数", fmt.Sprintf("%d 页", prev.PDFPages))
		}
		metaRow(c, "修改时间", prev.ModTime)
		metaRow(c, "权限", prev.Mode)
		metaRow(c, "路径", domain.CollapseHome(prev.Path, ""))
	})
}

func metaRow(c *ui.Context, label, val string) {
	t := c.Theme()
	ui.Row(c).Justify(ui.SpaceBetween).Gap(8).Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(t.TextMuted).Width(60)
		ui.Text(c, val).FontSize(11).TextColor(t.Text).SingleLine().Grow(1).TextAlign(ui.End)
	})
}

func renderColorTagsPicker(c *ui.Context, coord *app.Coordinator, pane *app.PaneState, prev *app.PreviewSnapshot) {
	t := c.Theme()
	tags := []struct {
		Idx   int
		Name  string
		Color ui.Color
	}{
		{macos.TagRed, "红色", TintRed.C(t)},
		{macos.TagOrange, "橙色", TintOrange.C(t)},
		{macos.TagYellow, "黄色", ui.Hex("#FFCC00")},
		{macos.TagGreen, "绿色", TintGreen.C(t)},
		{macos.TagBlue, "蓝色", TintBlue.C(t)},
		{macos.TagPurple, "紫色", TintPurple.C(t)},
		{macos.TagGray, "灰色", TintGray.C(t)},
	}

	ui.Column(c).Gap(6).Children(func() {
		ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Finder 颜色标签").FontSize(11).FontWeight(600).TextColor(t.TextMuted)
			if prev.ColorTag != macos.TagNone {
				btnClear := ui.ButtonBase(c).Padding(1, 4).Radius(3)
				btnClear.Children(func() {
					ui.Text(c, "清除").FontSize(10).TextColor(t.TextMuted)
				})
				if btnClear.Clicked() {
					_ = macos.RemoveFileTags(prev.Path)
					prev.ColorTag = macos.TagNone
					applyTagToPaneItem(pane, prev.Path, nil)
					coord.Invalidate()
				}
			}
		})
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			for _, tg := range tags {
				tagIdx := tg.Idx
				tagName := tg.Name
				isSelected := prev.ColorTag == tagIdx
				btn := ui.ButtonBase(c).Padding(2).Radius(10).AlignItems(ui.Center)
				if isSelected {
					btn.Border(1.5, t.Accent)
				}
				btn.Children(func() {
					ColorDot(c, tg.Color, 12)
				})
				if btn.Clicked() {
					if isSelected {
						_ = macos.RemoveFileTags(prev.Path)
						prev.ColorTag = macos.TagNone
						applyTagToPaneItem(pane, prev.Path, nil)
					} else {
						_ = macos.SetFileTag(prev.Path, tagIdx)
						prev.ColorTag = tagIdx
						applyTagToPaneItem(pane, prev.Path, []string{tagName})
					}
					coord.Invalidate()
				}
			}
		})
	})
}

func renderQuickActions(c *ui.Context, coord *app.Coordinator, prev *app.PreviewSnapshot) {
	ui.Row(c).Gap(6).Children(func() {
		btnTerm := PillButton(c, IcTerminal, "终端", false)
		if btnTerm.Clicked() {
			_ = macos.OpenInTerminal(prev.Path)
		}

		btnEdit := PillButton(c, IcEdit, "编辑", false)
		if btnEdit.Clicked() {
			_ = macos.OpenInEditor(prev.Path)
		}

		btnCopy := PillButton(c, IcCopy, "复制路径", false)
		if btnCopy.Clicked() {
			_ = macos.CopyPath(prev.Path, "posix")
		}
	})

	_ = coord
}

func renderSnippetPreview(c *ui.Context, coord *app.Coordinator, prev *app.PreviewSnapshot) {
	t := c.Theme()
	snippet := prev.Snippet
	truncated := len(snippet) >= 32768

	ui.Column(c).Gap(4).Children(func() {
		ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
			title := "内容预览"
			if prev.IsMarkdown {
				title = "Markdown 预览"
			} else if prev.IsCode && prev.CodeLang != "" {
				title = fmt.Sprintf("代码预览 (%s)", prev.CodeLang)
			}
			ui.Text(c, title).FontSize(11).FontWeight(600).TextColor(t.TextMuted)
			if truncated {
				ui.Text(c, "(展示前 32KB)").FontSize(10).TextColor(t.TextMuted.Alpha(0.6))
			}
		})

		if prev.IsMarkdown {
			ui.Scroll(c).MaxHeight(200).Padding(8).Radius(6).Background(Pal(t).Inset).Children(func() {
				markdown.View(c, snippet, nil)
			})
		} else if coord.CurrentEditorState != nil && coord.CurrentEditorPath == prev.Path {
			ui.Box(c).Height(180).Radius(6).Background(Pal(t).Inset).Padding(4).Children(func() {
				editor.View(c, coord.CurrentEditorState).Fill()
			})
		} else {
			lines := snippet
			if len(lines) > 4096 {
				lines = lines[:4096]
			}
			ui.Scroll(c).MaxHeight(160).Padding(6).Radius(6).Background(Pal(t).Inset).Children(func() {
				ui.Text(c, lines).FontSize(10).TextColor(t.Text).Lines(1, 100)
			})
		}
	})
}

func renderMarkdownPane(c *ui.Context, coord *app.Coordinator, pane *app.PaneState, prev *app.PreviewSnapshot, pWidth float32) {
	t := c.Theme()
	pal := Pal(t)

	// 1. Compact Header: Icon + Title + Size + Quick Look hint
	ui.Column(c).Gap(6).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
			ui.Icon(c, IcFile).Size(16, 16).TextColor(t.Accent)
			ui.Text(c, prev.Name).FontSize(13).FontWeight(600).TextColor(t.Text).SingleLine().Grow(1)
		})
		ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf("Markdown 文档 · %s", domain.FormatSize(prev.Size))).FontSize(11).TextColor(t.TextMuted)
			btnQL := ui.ButtonBase(c).Padding(2, 6).Radius(4).Background(pal.Inset)
			btnQL.Children(func() {
				ui.Row(c).Gap(3).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, IcEye).Size(11, 11).TextColor(t.TextMuted)
					ui.Text(c, "空格快速查看").FontSize(10).TextColor(t.TextMuted)
				})
			})
			if btnQL.Clicked() {
				macos.ToggleQuickLook(prev.Path)
			}
		})

		// Dual-mode Switcher: [ 👁 渲染预览 ]   [ ⌨️ 源码查看 ]
		ui.Row(c).Gap(6).Padding(4, 0).Children(func() {
			btnPreview := pillSegmentButton(c, IcEye, "渲染预览", coord.MarkdownViewMode == 0)
			if btnPreview.Clicked() {
				coord.MarkdownViewMode = 0
				coord.Invalidate()
			}
			btnSource := pillSegmentButton(c, IcEdit, "源码查看", coord.MarkdownViewMode == 1)
			if btnSource.Clicked() {
				coord.MarkdownViewMode = 1
				coord.Invalidate()
			}
		})
	})

	ui.Box(c).Height(1).Background(pal.Line)

	// 2. Full-height Markdown Viewer Body
	if coord.MarkdownViewMode == 0 {
		// Rendered Preview mode: rich markdown
		ui.Scroll(c).Grow(1).FillWidth().Padding(8).Radius(6).Background(pal.Inset).Children(func() {
			markdown.View(c, prev.Snippet, nil)
		})
	} else {
		// Source View mode: Zed-engine syntax highlighted code editor with line numbers
		if coord.CurrentEditorState != nil && coord.CurrentEditorPath == prev.Path {
			ui.Box(c).Grow(1).FillWidth().Padding(4).Radius(6).Background(pal.Inset).Children(func() {
				editor.View(c, coord.CurrentEditorState).Fill()
			})
		} else {
			lines := prev.Snippet
			ui.Scroll(c).Grow(1).FillWidth().Padding(6).Radius(6).Background(pal.Inset).Children(func() {
				ui.Text(c, lines).FontSize(10).TextColor(t.Text).Lines(1, 100)
			})
		}
	}

	ui.Box(c).Height(1).Background(pal.Line)

	// 3. Color tags & Quick Actions at bottom
	renderColorTagsPicker(c, coord, pane, prev)
	renderQuickActions(c, coord, prev)
}

func pillSegmentButton(c *ui.Context, icon *ui.SVG, label string, active bool) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Label(label).Padding(3, 8).Gap(4).Radius(5).AlignItems(ui.Center)
	bg := t.Text.Alpha(0.06)
	col := t.TextMuted
	if active {
		bg = t.Accent
		col = ui.Hex("#ffffff")
	} else if b.Hovered() {
		bg = t.Text.Alpha(0.12)
		col = t.Text
	}
	b.Background(bg)
	return b.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(12, 12).TextColor(col)
		}
		ui.Text(c, label).FontSize(11).TextColor(col).FontWeight(500)
	})
}
