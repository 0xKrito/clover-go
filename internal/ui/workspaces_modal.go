package ui

import (
	"fmt"
	"strings"

	"github.com/bighaidao/clover-go/internal/app"
	"github.com/bighaidao/clover-go/internal/config"
	"github.com/egoist/mygo/ui"
)

// WorkspacesModal renders the dialog for managing and saving multi-pane workspaces.
func WorkspacesModal(c *ui.Context, coord *app.Coordinator) {
	if !coord.ShowWorkspacesModal {
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
			renderWorkspacesHeader(c, coord)
			ui.Box(c).Height(1).Background(pal.Line)

			renderNewWorkspaceSection(c, coord)
			ui.Box(c).Height(1).Background(pal.Line)

			renderSavedWorkspacesSection(c, coord)

			ui.Box(c).Height(1).Background(pal.Line)
			renderWorkspacesFooter(c, coord)
		})
	})
}

func renderWorkspacesHeader(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "工作区快照管理").FontSize(14).FontWeight(600).TextColor(t.Text)
		btnClose := ui.ButtonBase(c).Padding(2).Radius(4)
		btnClose.Children(func() {
			ui.Icon(c, IcClose).Size(12, 12).TextColor(t.TextMuted)
		})
		if btnClose.Clicked() {
			coord.ShowWorkspacesModal = false
		}
	})
}

func renderNewWorkspaceSection(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	ui.Column(c).Gap(6).Children(func() {
		ui.Text(c, "将当前窗口标签与分栏布局另存为工作区:").FontSize(11).TextColor(t.TextMuted)
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.TextInput(c, &coord.WorkspaceDraftName).FillWidth().FontSize(12).Placeholder("输入工作区名称 (如: 开发调试 / 素材归档)")
			btnSave := ui.ButtonBase(c).Padding(6, 12).Radius(4).Background(t.Accent)
			btnSave.Children(func() {
				ui.Text(c, "保存当前布局").FontSize(11).FontWeight(600).TextColor(ui.Hex("#FFFFFF"))
			})
			if btnSave.Clicked() && strings.TrimSpace(coord.WorkspaceDraftName) != "" {
				_ = coord.SaveCurrentWorkspace(coord.WorkspaceDraftName)
				coord.WorkspaceDraftName = ""
			}
		})
	})
}

func renderSavedWorkspacesSection(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	pal := Pal(t)
	d := coord.Workspaces.Get()

	ui.Column(c).Gap(4).Children(func() {
		ui.Text(c, "已保存的工作区:").FontSize(11).TextColor(t.Text)

		if len(d.Workspaces) == 0 {
			ui.Text(c, "暂无已保存的工作区，可使用上方表单将当前布局保存为快照。").FontSize(11).TextColor(t.TextMuted).Padding(8)
			return
		}

		ui.Column(c).MaxHeight(180).Radius(4).Border(1, pal.Line).Padding(4).Gap(4).Children(func() {
			for _, ws := range d.Workspaces {
				renderWorkspaceRow(c, coord, ws)
			}
		})
	})
}

func renderWorkspaceRow(c *ui.Context, coord *app.Coordinator, ws config.NamedWorkspace) {
	t := c.Theme()
	pal := Pal(t)
	row := ui.Row(c).Height(30).Padding(0, 8).Gap(8).AlignItems(ui.Center).Radius(4)
	if row.Hovered() {
		row.Background(pal.Hover)
	}

	isActive := (coord.ActiveWorkspaceID == ws.ID)
	row.Children(func() {
		check := "  "
		if isActive {
			check = "✓ "
		}
		ui.Text(c, check).FontSize(12).TextColor(t.Accent)
		ui.Text(c, ws.Name).FontSize(11).FontWeight(500).TextColor(t.Text).Grow(1).SingleLine()

		tabCountInfo := fmt.Sprintf("%d 个标签", len(ws.Tabs))
		if len(ws.Tabs) == 0 && len(ws.Paths) > 0 {
			tabCountInfo = fmt.Sprintf("%d 个路径", len(ws.Paths))
		}
		ui.Text(c, tabCountInfo).FontSize(10).TextColor(t.TextMuted)

		btnRestore := ui.ButtonBase(c).Padding(2, 6).Radius(3).Background(t.Accent.Alpha(0.12))
		btnRestore.Children(func() {
			ui.Text(c, "载入").FontSize(10).TextColor(t.Accent)
		})
		if btnRestore.Clicked() {
			coord.RestoreWorkspace(ws)
			coord.ShowWorkspacesModal = false
		}

		btnDel := ui.ButtonBase(c).Padding(2, 6).Radius(3).Background(pal.Line)
		btnDel.Children(func() {
			ui.Text(c, "删除").FontSize(10).TextColor(t.TextMuted)
		})
		if btnDel.Clicked() {
			_ = coord.DeleteWorkspace(ws.ID)
		}
	})
}

func renderWorkspacesFooter(c *ui.Context, coord *app.Coordinator) {
	t := c.Theme()
	ui.Row(c).Justify(ui.End).AlignItems(ui.Center).Children(func() {
		btnClose := ui.ButtonBase(c).Padding(5, 14).Radius(4).Background(Pal(t).Line)
		btnClose.Children(func() {
			ui.Text(c, "关闭").FontSize(11).TextColor(t.Text)
		})
		if btnClose.Clicked() {
			coord.ShowWorkspacesModal = false
		}
	})
}
