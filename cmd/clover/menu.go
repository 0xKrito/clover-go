package main

import (
	"os"
	"path/filepath"

	"github.com/bighaidao/clover-go/internal/app"
	"github.com/bighaidao/clover-go/internal/config"
	"github.com/egoist/mygo"
)

// setupAppMenu installs the macOS menu bar. Without it mygo falls back to a
// default menu with no Settings entry, so the app menu offered no way to
// reach preferences (macOS convention: Clover → Settings… ⌘,).
// Menu clicks already arrive on the main thread (mygo menu.go), so handlers
// may touch coordinator state directly.
func setupAppMenu(coord *app.Coordinator) {
	openSettings := func(_ *mygo.MenuItem, win *mygo.Window) {
		coord.OpenSettings()
		coord.Invalidate()
		if win != nil {
			win.Invalidate()
		}
	}
	openAbout := func(_ *mygo.MenuItem, win *mygo.Window) {
		coord.ShowAboutModal = true
		coord.Invalidate()
		if win != nil {
			win.Invalidate()
		}
	}
	checkUpdates := func(_ *mygo.MenuItem, win *mygo.Window) {
		coord.Toast("当前已是最新版本 (v" + version + ")")
		coord.Invalidate()
		if win != nil {
			win.Invalidate()
		}
	}

	home, _ := os.UserHomeDir()

	menu := mygo.NewMenu([]*mygo.MenuItem{
		// 1. CloverGo (Application Menu)
		{
			Label: "CloverGo",
			Role:  mygo.RoleAppMenu,
			Submenu: []*mygo.MenuItem{
				{Label: "关于 CloverGo", Click: openAbout},
				{Label: "检查更新…", Accelerator: "Alt+CmdOrCtrl+U", Click: checkUpdates},
				mygo.Separator(),
				{Label: "设置…", Accelerator: "CmdOrCtrl+,", Click: openSettings},
				mygo.Separator(),
				{Role: mygo.RoleServices},
				mygo.Separator(),
				{Role: mygo.RoleHide},
				{Role: mygo.RoleHideOthers},
				{Role: mygo.RoleUnhide},
				mygo.Separator(),
				{Role: mygo.RoleQuit},
			},
		},

		// 2. 文件 (File)
		{
			Label: "文件",
			Role:  mygo.RoleFileMenu,
			Submenu: []*mygo.MenuItem{
				{Label: "新建标签页", Accelerator: "CmdOrCtrl+T", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					activeTab := coord.Tabs.ActiveTab()
					if activeTab != nil {
						coord.Tabs.NewTab(activeTab.ActivePane().Path)
						coord.LoadCurrentTab(nil)
					}
				}},
				{Label: "关闭标签页", Accelerator: "CmdOrCtrl+W", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					coord.Tabs.CloseTab(coord.Tabs.ActiveIndex)
					coord.LoadCurrentTab(nil)
				}},
				{Label: "新建文件夹", Accelerator: "Shift+CmdOrCtrl+N", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					activeTab := coord.Tabs.ActiveTab()
					if activeTab != nil {
						_, _ = coord.CreateNewFolder(activeTab.ActivePane(), "未命名文件夹")
					}
				}},
				mygo.Separator(),
				{Label: "复制到对侧分栏", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					activeTab := coord.Tabs.ActiveTab()
					if activeTab != nil && activeTab.IsDual {
						coord.CopySelectedToOppositePane(activeTab)
					}
				}},
				{Label: "移动到对侧分栏", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					activeTab := coord.Tabs.ActiveTab()
					if activeTab != nil && activeTab.IsDual {
						coord.MoveSelectedToOppositePane(activeTab)
					}
				}},
				mygo.Separator(),
				{Label: "显示简介", Accelerator: "CmdOrCtrl+I", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					activeTab := coord.Tabs.ActiveTab()
					if activeTab != nil {
						coord.ShowPreview = true
						activePane := activeTab.ActivePane()
						if activePane.SelectedIdx >= 0 && activePane.SelectedIdx < len(activePane.DisplayEntries) {
							coord.LoadPreview(activePane.DisplayEntries[activePane.SelectedIdx], nil)
						}
					}
				}},
				{Label: "移到废纸篓", Accelerator: "CmdOrCtrl+Backspace", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					activeTab := coord.Tabs.ActiveTab()
					if activeTab != nil {
						coord.TrashSelected(activeTab.ActivePane())
					}
				}},
				mygo.Separator(),
				{Role: mygo.RoleClose},
			},
		},

		// 3. 编辑 (Edit)
		{
			Label: "编辑",
			Role:  mygo.RoleEditMenu,
			Submenu: []*mygo.MenuItem{
				{Role: mygo.RoleUndo},
				{Role: mygo.RoleRedo},
				mygo.Separator(),
				{Role: mygo.RoleCut},
				{Role: mygo.RoleCopy},
				{Label: "复制绝对路径", Accelerator: "Alt+CmdOrCtrl+C", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					activeTab := coord.Tabs.ActiveTab()
					if activeTab != nil {
						coord.CopySelectedPath(activeTab.ActivePane())
					}
				}},
				{Role: mygo.RolePaste},
				{Role: mygo.RoleSelectAll, Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					activeTab := coord.Tabs.ActiveTab()
					if activeTab != nil {
						coord.SelectAll(activeTab.ActivePane())
					}
				}},
				mygo.Separator(),
				{Label: "查找…", Accelerator: "CmdOrCtrl+F", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					coord.FocusFilter = true
					coord.Invalidate()
				}},
			},
		},

		// 4. 视图 (View)
		{
			Label: "视图",
			Role:  mygo.RoleViewMenu,
			Submenu: []*mygo.MenuItem{
				{Label: "作为列表", Accelerator: "CmdOrCtrl+1", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					coord.ViewMode = config.ViewList
					p := coord.Prefs.Get()
					p.DefaultView = config.ViewList
					_ = coord.Prefs.Save(p)
					coord.Invalidate()
				}},
				{Label: "作为图标", Accelerator: "CmdOrCtrl+2", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					coord.ViewMode = config.ViewGrid
					p := coord.Prefs.Get()
					p.DefaultView = config.ViewGrid
					_ = coord.Prefs.Save(p)
					coord.Invalidate()
				}},
				mygo.Separator(),
				{Label: "双栏模式", Accelerator: "CmdOrCtrl+B", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					coord.ToggleDualPane()
				}},
				{Label: "显示/隐藏预览栏", Accelerator: "Shift+CmdOrCtrl+P", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					coord.TogglePreview()
					coord.Invalidate()
				}},
				{Label: "显示隐藏文件", Accelerator: "Shift+CmdOrCtrl+.", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					p := coord.Prefs.Get()
					p.ShowHidden = !p.ShowHidden
					_ = coord.Prefs.Save(p)
					coord.LoadCurrentTab(nil)
				}},
				mygo.Separator(),
				{Label: "刷新", Accelerator: "CmdOrCtrl+R", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					coord.LoadCurrentTab(nil)
				}},
			},
		},

		// 5. 前往 (Go)
		{
			Label: "前往",
			Submenu: []*mygo.MenuItem{
				{Label: "上一级文件夹", Accelerator: "CmdOrCtrl+Up", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					idx := coord.Tabs.ActiveIndex
					if coord.Tabs.CanGoUp(idx) {
						coord.Tabs.GoUp(idx)
						coord.LoadCurrentTab(nil)
					}
				}},
				mygo.Separator(),
				{Label: "个人", Accelerator: "Shift+CmdOrCtrl+H", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					activeTab := coord.Tabs.ActiveTab()
					if activeTab != nil {
						activeTab.ActivePane().Navigate(home)
						coord.LoadCurrentTab(nil)
					}
				}},
				{Label: "桌面", Accelerator: "Shift+CmdOrCtrl+D", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					activeTab := coord.Tabs.ActiveTab()
					if activeTab != nil {
						activeTab.ActivePane().Navigate(filepath.Join(home, "Desktop"))
						coord.LoadCurrentTab(nil)
					}
				}},
				{Label: "文稿", Accelerator: "Shift+CmdOrCtrl+O", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					activeTab := coord.Tabs.ActiveTab()
					if activeTab != nil {
						activeTab.ActivePane().Navigate(filepath.Join(home, "Documents"))
						coord.LoadCurrentTab(nil)
					}
				}},
				{Label: "下载", Accelerator: "Alt+CmdOrCtrl+L", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					activeTab := coord.Tabs.ActiveTab()
					if activeTab != nil {
						activeTab.ActivePane().Navigate(filepath.Join(home, "Downloads"))
						coord.LoadCurrentTab(nil)
					}
				}},
			},
		},

		// 6. 窗口 (Window)
		{
			Label: "窗口",
			Role:  mygo.RoleWindowMenu,
			Submenu: []*mygo.MenuItem{
				{Role: mygo.RoleMinimize},
				{Role: mygo.RoleZoom},
				mygo.Separator(),
				{Label: "重置分栏布局", Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
					coord.SidebarWidth = 155
					coord.DualPaneRatio = 50
					coord.PreviewWidth = 250
					p := coord.Prefs.Get()
					p.SidebarWidth = 155
					p.DualPaneRatio = 50
					p.PreviewWidth = 250
					_ = coord.Prefs.Save(p)
					coord.Invalidate()
				}},
				mygo.Separator(),
				{Role: mygo.RoleFront},
			},
		},
	})

	mygo.App.SetMenu(menu)
}
