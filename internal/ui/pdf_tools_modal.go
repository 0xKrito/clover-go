package ui

import (
	"fmt"
	"path/filepath"

	"github.com/bighaidao/clover-go/internal/app"
	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/egoist/mygo/ui"
)

// PDFToolsModal renders the PDF micro-tools popup dialog.
func PDFToolsModal(c *ui.Context, coord *app.Coordinator) {
	if !coord.ShowPDFModal {
		return
	}

	t := c.Theme()
	pal := Pal(t)

	// Modal backdrop
	backdrop := ui.Box(c).Fill().Background(ui.Hex("#000000").Alpha(0.4)).AlignItems(ui.Center).Justify(ui.Center)
	backdrop.Children(func() {
		// Modal Dialog Window
		dialog := ui.Column(c).Width(480).Background(t.Background).Radius(10).Border(1, pal.Line).Padding(16).Gap(12)
		dialog.Shadow(0, 8, 24, 0, ui.Hex("#000000").Alpha(0.25))

		dialog.Children(func() {
			renderPDFHeader(c, coord)
			ui.Box(c).Height(1).Background(pal.Line)

			if coord.PDFOptions.Mode == domain.PDFModeMerge {
				renderPDFMergeBody(c, coord)
			} else {
				renderPDFSingleBody(c, coord)
			}

			if coord.PDFError != "" {
				ui.Text(c, "错误："+coord.PDFError).FontSize(11).TextColor(ui.Hex("#FF3B30"))
			}

			ui.Box(c).Height(1).Background(pal.Line)
			renderPDFFooter(c, coord)
		})
	})
}

func renderPDFHeader(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	title := "📄 PDF 瑞士军刀微工具"
	if coord.PDFOptions.Mode == domain.PDFModeMerge {
		title = fmt.Sprintf("📄 PDF 合并工具 (已选 %d 个文件)", len(coord.PDFOptions.SourceFiles))
	}

	ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
		ui.Text(c, title).FontSize(14).FontWeight(600).TextColor(t.Text)
		btnClose := ui.ButtonBase(c).Padding(2).Radius(4)
		btnClose.Children(func() {
			ui.Icon(c, IcClose).Size(12, 12).TextColor(t.TextMuted)
		})
		if btnClose.Clicked() {
			coord.ClosePDFTools()
		}
	})
}

func renderPDFSingleBody(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	srcName := filepath.Base(coord.PDFOptions.SourceFiles[0])

	ui.Text(c, fmt.Sprintf("针对文件: %s (共 %d 页)", srcName, coord.PDFTotalPages)).FontSize(11).TextColor(t.TextMuted)

	// Mode tabs
	ui.Row(c).Gap(6).Children(func() {
		renderModeBtn(c, coord, "页面提取", domain.PDFModeExtract)
		renderModeBtn(c, coord, "✂️ 按页拆分", domain.PDFModeSplit)
		renderModeBtn(c, coord, "🖼️ 转为图片", domain.PDFModeExportImages)
	})

	if coord.PDFOptions.Mode == domain.PDFModeExtract || coord.PDFOptions.Mode == domain.PDFModeExportImages {
		ui.Column(c).Gap(4).Children(func() {
			ui.Text(c, "提取页码范围:").FontSize(11).TextColor(t.Text)
			ui.TextInput(c, &coord.PDFOptions.PageRanges).FillWidth().FontSize(12).Placeholder(fmt.Sprintf("示例: 1-%d, 留空表示全部页面", coord.PDFTotalPages))
		})
	}

	if coord.PDFOptions.Mode == domain.PDFModeExportImages {
		renderImageExportOptions(c, coord)
	} else if coord.PDFOptions.Mode == domain.PDFModeExtract {
		ui.Column(c).Gap(4).Children(func() {
			ui.Text(c, "输出文件名:").FontSize(11).TextColor(t.Text)
			ui.TextInput(c, &coord.PDFOptions.OutputName).FillWidth().FontSize(12)
		})
	}
}

func renderModeBtn(c *ui.Context, coord *app.Coordinator, label string, mode domain.PDFMode) {
	t := c.Theme()
	isSel := coord.PDFOptions.Mode == mode
	bg := t.Background
	txtCol := t.TextMuted
	if isSel {
		bg = t.Accent.Alpha(0.15)
		txtCol = t.Accent
	}

	btn := ui.ButtonBase(c).Padding(6, 10).Radius(5).Background(bg)
	btn.Children(func() {
		ui.Text(c, label).FontSize(11).FontWeight(500).TextColor(txtCol)
	})
	if btn.Clicked() {
		coord.PDFOptions.Mode = mode
	}
}

func renderImageExportOptions(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "图片格式:").FontSize(11).TextColor(t.Text)
		renderRadioBtn(c, "PNG (无损)", coord.PDFOptions.ImageFormat == "png", func() {
			coord.PDFOptions.ImageFormat = "png"
		})
		renderRadioBtn(c, "JPG (高压缩)", coord.PDFOptions.ImageFormat == "jpeg" || coord.PDFOptions.ImageFormat == "jpg", func() {
			coord.PDFOptions.ImageFormat = "jpeg"
		})
	})

	ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "分辨率:").FontSize(11).TextColor(t.Text)
		renderRadioBtn(c, "150 DPI (标准)", coord.PDFOptions.ImageDPI == 150, func() {
			coord.PDFOptions.ImageDPI = 150
		})
		renderRadioBtn(c, "300 DPI (高清)", coord.PDFOptions.ImageDPI == 300, func() {
			coord.PDFOptions.ImageDPI = 300
		})
	})
}

func renderPDFMergeBody(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	ui.Text(c, "即将按以下顺序合并为一个 PDF 文件:").FontSize(11).TextColor(t.TextMuted)

	ui.Column(c).MaxHeight(100).Radius(4).Border(1, Pal(t).Line).Padding(6).Gap(4).Children(func() {
		for i, p := range coord.PDFOptions.SourceFiles {
			ui.Text(c, fmt.Sprintf("%d. %s", i+1, filepath.Base(p))).FontSize(11).TextColor(t.Text).SingleLine()
		}
	})

	ui.Column(c).Gap(4).Children(func() {
		ui.Text(c, "合并后文件名称:").FontSize(11).TextColor(t.Text)
		ui.TextInput(c, &coord.PDFOptions.OutputName).FillWidth().FontSize(12)
	})
}

func renderPDFFooter(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	ui.Row(c).Justify(ui.End).Gap(8).AlignItems(ui.Center).Children(func() {
		btnCancel := ui.ButtonBase(c).Padding(5, 12).Radius(4).Background(Pal(t).Line)
		btnCancel.Children(func() {
			ui.Text(c, "取消").FontSize(11).TextColor(t.Text)
		})
		if btnCancel.Clicked() {
			coord.ClosePDFTools()
		}

		btnExec := ui.ButtonBase(c).Padding(5, 16).Radius(4).Background(t.Accent)
		btnExec.Children(func() {
			label := "开始执行"
			if coord.PDFIsProcessing {
				label = "处理中..."
			}
			ui.Text(c, label).FontSize(11).FontWeight(600).TextColor(ui.Hex("#FFFFFF"))
		})
		if btnExec.Clicked() && !coord.PDFIsProcessing {
			coord.ExecutePDFTools()
		}
	})
}

func renderRadioBtn(c *ui.Context, label string, checked bool, onClick func()) {
	t := c.Theme()
	btn := ui.Row(c).Gap(4).AlignItems(ui.Center)
	btn.Children(func() {
		dot := "○"
		col := t.TextMuted
		if checked {
			dot = "●"
			col = t.Accent
		}
		ui.Text(c, dot).FontSize(12).TextColor(col)
		ui.Text(c, label).FontSize(11).TextColor(t.Text)
	})
	if btn.Clicked() {
		onClick()
	}
}
