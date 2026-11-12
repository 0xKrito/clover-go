package ui

import (
	"fmt"
	"path/filepath"

	"github.com/bighaidao/clover-go/internal/app"
	"github.com/bighaidao/clover-go/internal/fs"
	"github.com/egoist/mygo/ui"
)

// renderArchiveBanner renders an informational read-only bar when browsing inside an archive.
func renderArchiveBanner(c *ui.Context, coord *app.Coordinator, pane *app.PaneState) {
	t := c.Theme()
	pal := Pal(t)
	archivePath, internalPath, _ := fs.ParseArchiveVirtualPath(pane.Path)
	archiveName := filepath.Base(archivePath)

	ui.Row(c).Height(30).Padding(0, 8).Background(pal.Line.Alpha(0.35)).Radius(4).AlignItems(ui.Center).Gap(8).Children(func() {
		ui.Icon(c, IcFolder).Size(13, 13).TextColor(t.Accent).Shrink(0)
		label := fmt.Sprintf("虚拟浏览 (只读) · %s", archiveName)
		if internalPath != "" {
			label += " / " + internalPath
		}
		ui.Text(c, label).FontSize(11).TextColor(t.Text).Grow(1).Shrink(1).MinWidth(0).SingleLine().Tooltip(label)

		btnExtract := ui.ButtonBase(c).Padding(3, 8).Radius(3).Background(t.Accent.Alpha(0.15)).Shrink(0)
		btnExtract.Children(func() {
			ui.Text(c, "全部解压").FontSize(11).TextColor(t.Accent)
		})
		if btnExtract.Clicked() {
			coord.ExtractEntireArchive(pane)
		}

		btnExit := ui.ButtonBase(c).Padding(3, 8).Radius(3).Background(pal.Line).Shrink(0)
		btnExit.Children(func() {
			ui.Text(c, "退出").FontSize(11).TextColor(t.TextMuted)
		})
		if btnExit.Clicked() {
			// 与右键“退出压缩包”/⌘↑ 同一逐层退栈语义（包内→包根→物理父目录）。
			pane.GoUp()
			coord.LoadCurrentTab(nil)
		}
	})
	ui.Box(c).Height(1).Background(pal.Line.Alpha(0.5))
}
