package ui

import (
	"fmt"

	"github.com/bighaidao/clover-go/internal/app"
	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/egoist/mygo/ui"
)

// WatermarkModal renders the image text watermark tool dialog.
func WatermarkModal(c *ui.Context, coord *app.Coordinator) {
	if !coord.ShowWatermarkModal {
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
			renderWatermarkHeader(c, coord)
			ui.Box(c).Height(1).Background(pal.Line)

			renderWatermarkBody(c, coord)

			if coord.WatermarkError != "" {
				ui.Text(c, "错误："+coord.WatermarkError).FontSize(11).TextColor(ui.Hex("#FF3B30"))
			}

			ui.Box(c).Height(1).Background(pal.Line)
			renderWatermarkFooter(c, coord)
		})
	})
}

func renderWatermarkHeader(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	title := fmt.Sprintf("图片文字水印 (已选 %d 张图片)", len(coord.WatermarkOptions.SourceFiles))

	ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
		ui.Text(c, title).FontSize(14).FontWeight(600).TextColor(t.Text)
		btnClose := ui.ButtonBase(c).Padding(2).Radius(4)
		btnClose.Children(func() {
			ui.Icon(c, IcClose).Size(12, 12).TextColor(t.TextMuted)
		})
		if btnClose.Clicked() {
			coord.CloseWatermark()
		}
	})
}

func renderWatermarkBody(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	opts := &coord.WatermarkOptions

	// Text input
	ui.Column(c).Gap(4).Children(func() {
		ui.Text(c, "水印文字:").FontSize(11).TextColor(t.Text)
		ui.TextInput(c, &opts.Text).FillWidth().FontSize(12).Placeholder("例如: CONFIDENTIAL · 内部资料")
	})

	renderWmOptionRow(c, "排版布局:", []string{"全屏平铺 (密集)", "居中单条"},
		func(v string) bool { return v == layoutLabel(opts.Layout) },
		func(v string) { opts.Layout = layoutOfLabel(v) })
	renderWmOptionRow(c, "字体颜色:", []string{"灰色", "红色", "白色", "黑色"},
		func(v string) bool { return colorKeyOfLabel(v) == opts.TextColor },
		func(v string) { opts.TextColor = colorKeyOfLabel(v) })
	renderWmOptionRow(c, "不透明度:", []string{"20%", "35%", "50%", "75%"},
		func(v string) bool { return opacityOfLabel(v) == opts.Opacity },
		func(v string) { opts.Opacity = opacityOfLabel(v) })
	renderWmOptionRow(c, "旋转角度:", []string{"-45°", "-30°", "-15°", "0°"},
		func(v string) bool { return angleOfLabel(v) == opts.Angle },
		func(v string) { opts.Angle = angleOfLabel(v) })
	renderWmOptionRow(c, "字体大小:", []string{"18", "24", "32", "48"},
		func(v string) bool { return fontSizeOfLabel(v) == opts.FontSize },
		func(v string) { opts.FontSize = fontSizeOfLabel(v) })
	renderWmOptionRow(c, "输出格式:", []string{"保持原格式", "PNG", "JPG"},
		func(v string) bool { return formatOfLabel(v) == opts.OutputFormat },
		func(v string) { opts.OutputFormat = formatOfLabel(v) })

	ui.Text(c, "说明：输出为同目录下的 “_watermarked” 文件；重新编码不携带源图 EXIF/GPS 元数据。").FontSize(10).TextColor(t.TextMuted)
}

// renderWmOptionRow renders one "label [a][b][c]" option row.
func renderWmOptionRow(c *ui.Context, label string, options []string, isActive func(string) bool, onPick func(string)) {
	t := c.Theme()
	ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(t.Text)
		for _, opt := range options {
			opt := opt
			renderRadioBtn(c, opt, isActive(opt), func() { onPick(opt) })
		}
	})
}

func layoutLabel(l domain.WatermarkLayout) string {
	if l == domain.WatermarkLayoutTile {
		return "全屏平铺 (密集)"
	}
	return "居中单条"
}

func layoutOfLabel(v string) domain.WatermarkLayout {
	if v == "全屏平铺 (密集)" {
		return domain.WatermarkLayoutTile
	}
	return domain.WatermarkLayoutCenter
}

func colorKeyOfLabel(v string) string {
	switch v {
	case "红色":
		return "red"
	case "白色":
		return "white"
	case "黑色":
		return "black"
	default:
		return "gray"
	}
}

func opacityOfLabel(v string) float64 {
	switch v {
	case "20%":
		return 0.20
	case "50%":
		return 0.50
	case "75%":
		return 0.75
	default:
		return 0.35
	}
}

func angleOfLabel(v string) float64 {
	switch v {
	case "-45°":
		return -45
	case "-15°":
		return -15
	case "0°":
		return 0
	default:
		return -30
	}
}

func fontSizeOfLabel(v string) int {
	switch v {
	case "18":
		return 18
	case "32":
		return 32
	case "48":
		return 48
	default:
		return 24
	}
}

func formatOfLabel(v string) string {
	switch v {
	case "PNG":
		return "png"
	case "JPG":
		return "jpeg"
	default:
		return "keep"
	}
}

func renderWatermarkFooter(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	ui.Row(c).Justify(ui.End).Gap(8).AlignItems(ui.Center).Children(func() {
		btnCancel := ui.ButtonBase(c).Padding(5, 12).Radius(4).Background(Pal(t).Line)
		btnCancel.Children(func() {
			ui.Text(c, "取消").FontSize(11).TextColor(t.Text)
		})
		if btnCancel.Clicked() {
			coord.CloseWatermark()
		}

		btnExec := ui.ButtonBase(c).Padding(5, 16).Radius(4).Background(t.Accent)
		btnExec.Children(func() {
			label := "开始处理"
			if coord.WatermarkIsRunning {
				label = "生成中..."
			}
			ui.Text(c, label).FontSize(11).FontWeight(600).TextColor(ui.Hex("#FFFFFF"))
		})
		if btnExec.Clicked() && !coord.WatermarkIsRunning {
			coord.ExecuteWatermark()
		}
	})
}
