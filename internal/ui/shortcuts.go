package ui

import (
	"github.com/bighaidao/clover-go/internal/app"
	"github.com/bighaidao/clover-go/internal/config"
	"github.com/bighaidao/clover-go/internal/fs"
	"github.com/bighaidao/clover-go/internal/macos"
	"github.com/egoist/mygo/ui"
)

func (s *AppState) handleShortcuts(c *ui.Context, activeTab *app.TabState) {
	coord := s.Coord
	// All item-level shortcuts target the focused pane, never the left pane.
	activePane := activeTab.ActivePane()

	s.handleTabShortcuts(c, coord, activePane)
	s.handleEditAndViewShortcuts(c, coord, activePane)
	s.handleHistoryShortcuts(c, coord)
	s.handleItemShortcuts(c, coord, activePane)
}

// handleTabShortcuts covers tab lifecycle, modal dismissal and dual-pane toggling.
func (s *AppState) handleTabShortcuts(c *ui.Context, coord *app.Coordinator, activePane *app.PaneState) {
	if c.Shortcut(ui.Cmd, ui.KeyT) {
		coord.Tabs.NewTab(activePane.Path)
		coord.LoadCurrentTab(nil)
	}
	if c.Shortcut(ui.Cmd, ui.KeyW) {
		coord.Tabs.CloseTab(coord.Tabs.ActiveIndex)
		coord.LoadCurrentTab(nil)
	}
	if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyT) {
		if restored := coord.Tabs.RestoreLastClosedTab(); restored != nil {
			coord.LoadCurrentTab(nil)
		}
	}
	if c.Shortcut(0, ui.KeyEscape) {
		if macos.IsQuickLookActive() {
			macos.CloseQuickLook()
			return
		}
		if coord.ShowConflictModal {
			coord.ResolveConflict(app.ChoiceCancel, false)
		} else if coord.ShowAboutModal {
			coord.ShowAboutModal = false
		} else if coord.ShowSettingsModal {
			coord.CloseSettings()
		} else if coord.ShowBatchRenameModal {
			coord.CloseBatchRename()
		} else if coord.ShowPDFModal {
			coord.ClosePDFTools()
		} else if coord.ShowWatermarkModal {
			coord.CloseWatermark()
		} else if coord.ShowFinderImportModal {
			coord.CloseFinderImport()
		} else if coord.ShowWorkspacesModal {
			coord.ShowWorkspacesModal = false
		} else {
			coord.IsEditing = false
			coord.CancelRename()
		}
	}
	// Toggle Dual-Pane (⌘\ or ⌥⌘D)
	if c.Shortcut(ui.Cmd, ui.KeyBackslash) || c.Shortcut(ui.Cmd|ui.Alt, ui.KeyD) {
		coord.ToggleDualPane()
	}
	// Switch Active Pane when in dual-pane mode (Tab key, if not editing)
	if coord.Tabs.ActiveTab().IsDual && !coord.IsEditing && coord.RenamingIdx < 0 {
		if c.Shortcut(0, ui.KeyTab) {
			coord.SwitchActivePane()
		}
	}
}

// handleEditAndViewShortcuts covers path edit, filter focus, view/panel
// toggles, batch rename, reload and hidden files.
func (s *AppState) handleEditAndViewShortcuts(c *ui.Context, coord *app.Coordinator, activePane *app.PaneState) {
	if c.Shortcut(ui.Cmd, ui.KeyL) {
		coord.BeginPathEdit(activePane)
	}
	// ⌘F focuses the filter/search box in the toolbar (Finder semantics);
	// path editing stays on ⌘L.
	if c.Shortcut(ui.Cmd, ui.KeyF) {
		coord.FocusFilter = true
	}
	if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyP) {
		coord.TogglePreview()
	}
	// Preferences (macOS convention ⌘,)
	if c.Shortcut(ui.Cmd, ui.KeyComma) {
		coord.OpenSettings()
	}
	if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyR) {
		coord.OpenBatchRename(activePane)
	}
	// Reload the active pane's directory (⌘R)：手动刷新强制重读，
	// 不被搜索模式的静默跳过拦下。
	if c.Shortcut(ui.Cmd, ui.KeyR) {
		coord.ReloadCurrentTab()
	}
	// View mode shortcuts (⌘1 List, ⌘2 Grid)
	if c.Shortcut(ui.Cmd, ui.Key1) {
		coord.ViewMode = config.ViewList
		p := coord.Prefs.Get()
		p.DefaultView = config.ViewList
		_ = coord.Prefs.Save(p)
		coord.Invalidate()
	}
	if c.Shortcut(ui.Cmd, ui.Key2) {
		coord.ViewMode = config.ViewGrid
		p := coord.Prefs.Get()
		p.DefaultView = config.ViewGrid
		_ = coord.Prefs.Save(p)
		coord.Invalidate()
	}
	// Toggle Dual-Pane (⌘B)
	if c.Shortcut(ui.Cmd, ui.KeyB) {
		coord.ToggleDualPane()
	}
	// Toggle hidden files (⇧⌘.)
	if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyPeriod) {
		p := coord.Prefs.Get()
		p.ShowHidden = !p.ShowHidden
		_ = coord.Prefs.Save(p)
		coord.LoadCurrentTab(nil)
	}
}

// handleHistoryShortcuts covers ⌘↑ / ⌘[ / ⌘] navigation on the active pane.
func (s *AppState) handleHistoryShortcuts(c *ui.Context, coord *app.Coordinator) {
	idx := coord.Tabs.ActiveIndex
	if c.Shortcut(ui.Cmd, ui.KeyUp) && coord.Tabs.CanGoUp(idx) {
		coord.Tabs.GoUp(idx)
		coord.LoadCurrentTab(nil)
	}
	if c.Shortcut(ui.Cmd, ui.KeyBracketLeft) && coord.Tabs.CanGoBack(idx) {
		coord.Tabs.GoBack(idx)
		coord.LoadCurrentTab(nil)
	}
	if c.Shortcut(ui.Cmd, ui.KeyBracketRight) && coord.Tabs.CanGoForward(idx) {
		coord.Tabs.GoForward(idx)
		coord.LoadCurrentTab(nil)
	}
}

// TriggerQuickLook toggles the macOS Quick Look preview for the currently selected item in activePane.
func TriggerQuickLook(coord *app.Coordinator, activePane *app.PaneState) {
	if coord == nil || coord.IsEditing || coord.RenamingIdx >= 0 || activePane == nil {
		return
	}
	if activePane.SelectedIdx < 0 || activePane.SelectedIdx >= len(activePane.DisplayEntries) {
		return
	}
	item := activePane.DisplayEntries[activePane.SelectedIdx]
	targetPath := item.Path
	if fs.IsArchiveVirtualPath(targetPath) {
		if !item.IsDir {
			if tmp, err := fs.ExtractArchiveFileToTemp(targetPath); err == nil {
				targetPath = tmp
			}
		}
	}
	macos.ToggleQuickLook(targetPath)
}

// handleItemShortcuts covers clipboard, rename, new folder and Quick Look;
// all act on the focused pane and are suppressed during text editing.
func (s *AppState) handleItemShortcuts(c *ui.Context, coord *app.Coordinator, activePane *app.PaneState) {
	if !coord.IsEditing && coord.RenamingIdx < 0 {
		if c.Shortcut(ui.Cmd, ui.KeyA) {
			coord.SelectAll(activePane)
		}
		if (c.Shortcut(ui.Cmd, ui.KeyO) || c.Shortcut(ui.Cmd, ui.KeyDown)) && activePane.SelectedIdx >= 0 {
			coord.OpenSelectedItem(activePane)
		}
		if c.Shortcut(ui.Cmd, ui.KeyC) {
			coord.CopySelected(activePane)
		}
		if c.Shortcut(ui.Cmd, ui.KeyX) {
			coord.CutSelected(activePane)
		}
		if c.Shortcut(ui.Cmd, ui.KeyV) {
			coord.Paste(activePane)
		}
		if c.Shortcut(ui.Cmd, ui.KeyD) {
			coord.DuplicateSelected(activePane)
		}
		if c.Shortcut(ui.Cmd, ui.KeyI) {
			// ⌘I 开关预览栏：已开启且有选中项时刷新预览内容。
			coord.TogglePreview()
			if coord.ShowPreview && activePane.SelectedIdx >= 0 && activePane.SelectedIdx < len(activePane.DisplayEntries) {
				coord.LoadPreview(activePane.DisplayEntries[activePane.SelectedIdx], nil)
			}
		}
		if c.Shortcut(ui.Cmd|ui.Alt, ui.KeyC) {
			coord.CopySelectedPath(activePane)
		}
		if c.Shortcut(ui.Cmd, ui.KeyBackspace) {
			coord.TrashSelected(activePane)
		}
		if c.Shortcut(0, ui.KeyEnter) && activePane.SelectedIdx >= 0 {
			coord.StartRename(activePane, activePane.SelectedIdx)
		}
		if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyN) {
			_, _ = coord.CreateNewFolder(activePane, "未命名文件夹")
		}
	}

	// Space for Quick Look (macOS native in-process QLPreviewPanel toggle)
	if c.Shortcut(0, ui.KeySpace) {
		TriggerQuickLook(coord, activePane)
	}
}
