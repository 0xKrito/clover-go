package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bighaidao/clover-go/internal/app"
	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/bighaidao/clover-go/internal/fs"
	"github.com/bighaidao/clover-go/internal/macos"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// ShowFileContextMenu renders the comprehensive right-click context menu for
// a file item; the sections live in the context*Actions helpers below.
func ShowFileContextMenu(m *ui.Menu, c *ui.Context, coord *app.Coordinator, pane *app.PaneState, item domain.FileInfo, i int) {
	if fs.IsArchiveVirtualPath(pane.Path) {
		showArchiveItemContextMenu(m, coord, pane, item)
		return
	}
	if activeTab := coord.Tabs.ActiveTab(); activeTab != nil && activeTab.IsDual {
		if m.Item("复制到对侧分栏").Chosen() {
			coord.CopySelectedToOppositePane(activeTab)
		}
		if m.Item("移动到对侧分栏").Chosen() {
			coord.MoveSelectedToOppositePane(activeTab)
		}
		m.Separator()
	}
	contextOpenActions(m, c, coord, pane, item)
	m.Separator()
	contextSystemActions(m, coord, item)
	m.Separator()
	contextClipboardActions(m, coord, pane, item)
	m.Separator()
	contextRenameActions(m, coord, pane, i)
	m.Separator()
	contextTagTrashActions(m, coord, pane, item)
}

// showArchiveItemContextMenu provides a safe read-only context menu inside virtual archives.
func showArchiveItemContextMenu(m *ui.Menu, coord *app.Coordinator, pane *app.PaneState, item domain.FileInfo) {
	if m.Item("快速查看 (空格)").Chosen() {
		if !item.IsDir {
			if tmpPath, err := fs.ExtractArchiveFileToTemp(item.Path); err == nil {
				macos.ToggleQuickLook(tmpPath)
			}
		}
	}
	if m.Item("显示/隐藏预览栏 (⌘I)").Chosen() {
		coord.TogglePreview()
		if coord.ShowPreview {
			coord.LoadPreview(item, nil)
		}
	}
	if m.Item("打开").Chosen() {
		openItem(coord, pane, item)
	}
	m.Separator()
	if m.Item("提取选中项到同级目录...").Chosen() {
		coord.ExtractVirtualSelected(pane)
	}
	if m.Item("复制虚拟路径").Chosen() {
		_ = macos.CopyPath(item.Path, "posix")
	}
	m.Separator()
	if m.Item("在访达中显示压缩包").Chosen() {
		archivePath, _, _ := fs.ParseArchiveVirtualPath(pane.Path)
		mygo.Shell.ShowItemInFolder(archivePath)
	}
}

// contextOpenActions renders Quick Look / preview toggle / open / new tab / package-content entries.
func contextOpenActions(m *ui.Menu, c *ui.Context, coord *app.Coordinator, pane *app.PaneState, item domain.FileInfo) {
	// 1. 快速查看 (Native macOS QLPreviewPanel)
	if m.Item("快速查看 (空格)").Chosen() {
		macos.ToggleQuickLook(item.Path)
	}

	// 2. 预览栏显隐切换 (Inspector Pane)
	if m.Item("显示/隐藏预览栏 (⌘I)").Chosen() {
		coord.TogglePreview()
		if coord.ShowPreview {
			coord.LoadPreview(item, nil)
		}
	}

	// 3. 压缩包就地免解压浏览
	if fs.IsArchiveFile(item.Path) && !fs.IsArchiveVirtualPath(item.Path) {
		if m.Item("浏览压缩包内容 (就地免解压)").Chosen() {
			vPath := fs.FormatArchiveVirtualPath(item.Path, "")
			coord.Tabs.NavigatePane(coord.Tabs.ActiveIndex, pane, vPath)
			coord.LoadCurrentTab(nil)
		}
	}

	// 4. 打开
	if m.Item("打开").Chosen() {
		openItem(coord, pane, item)
	}

	// 5. PDF 瑞士军刀微工具 / 图片水印
	contextToolActions(m, coord, pane, item)

	// 6. 在新标签页中打开
	targetTabPath := item.Path
	if !item.IsDir || item.IsAppBundle {
		targetTabPath = filepath.Dir(item.Path)
	}
	if m.Item("在新标签页中打开").Chosen() {
		coord.Tabs.NewTab(targetTabPath)
		coord.LoadCurrentTab(nil)
	}

	if item.IsAppBundle {
		if m.Item("显示包内容").Chosen() {
			coord.Tabs.NavigatePane(coord.Tabs.ActiveIndex, pane, item.Path)
			coord.LoadCurrentTab(nil)
		}
	}
}

// contextToolActions mounts the PDF micro-tool and image watermark entries,
// gated by the actual PDF/image count of the current selection.
func contextToolActions(m *ui.Menu, coord *app.Coordinator, pane *app.PaneState, item domain.FileInfo) {
	if strings.EqualFold(filepath.Ext(item.Name), ".pdf") {
		pdfCount := countSelectionPDFs(coord, pane)
		if pdfCount > 1 {
			label := fmt.Sprintf("合并所选 PDF 文件 (%d个)...", pdfCount)
			if m.Item(label).Chosen() {
				coord.OpenPDFTools(pane, domain.PDFModeMerge)
			}
		} else {
			if m.Item("PDF 瑞士军刀 (提取/拆分/转图)...").Chosen() {
				coord.OpenPDFTools(pane, domain.PDFModeExtract)
			}
		}
	}

	if !item.IsDir && macos.IsImageFile(item.Path) {
		if m.Item("添加文字水印...").Chosen() {
			coord.OpenWatermark(pane)
		}
	}
}

// countSelectionPDFs counts how many of the pane's selected items are PDFs,
// so the merge menu label matches what OpenPDFTools will actually process.
func countSelectionPDFs(coord *app.Coordinator, pane *app.PaneState) int {
	count := 0
	for _, it := range coord.SelectedItems(pane) {
		if strings.EqualFold(filepath.Ext(it.Name), ".pdf") {
			count++
		}
	}
	return count
}

// contextSystemActions renders bookmark, Finder-reveal, path-copy, terminal
// and editor entries.
func contextSystemActions(m *ui.Menu, coord *app.Coordinator, item domain.FileInfo) {
	// 添加到/移出收藏 (书签)
	if item.IsDir && !item.IsAppBundle {
		if coord.IsBookmarked(item.Path) {
			if m.Item("从收藏中移除").Chosen() {
				_ = coord.RemoveBookmark(item.Path)
			}
		} else {
			if m.Item("添加到收藏 / 书签").Chosen() {
				_ = coord.AddBookmark(item.Path, item.Name)
			}
		}
	}

	if m.Item("在访达中显示").Chosen() {
		mygo.Shell.ShowItemInFolder(item.Path)
	}

	if m.Item("复制绝对路径 (⌥⌘C)").Chosen() {
		_ = macos.CopyPath(item.Path, "posix")
	}

	if m.Item("在终端中打开").Chosen() {
		_ = macos.OpenInTerminal(item.Path)
	}
	if m.Item("在编辑器中打开").Chosen() {
		_ = macos.OpenInEditor(item.Path)
	}
}

// contextClipboardActions renders cut/copy plus zip compress/extract entries.
func contextClipboardActions(m *ui.Menu, coord *app.Coordinator, pane *app.PaneState, item domain.FileInfo) {
	if m.Item("剪切 (⌘X)").Chosen() {
		coord.CutSelected(pane)
	}
	if m.Item("复制 (⌘C)").Chosen() {
		coord.CopySelected(pane)
	}

	if strings.EqualFold(filepath.Ext(item.Name), ".zip") {
		if m.Item("解压到当前文件夹").Chosen() {
			coord.ExtractSelected(pane)
		}
	} else {
		if m.Item("压缩为 Zip 归档").Chosen() {
			coord.CompressSelected(pane)
		}
	}
}

// contextRenameActions renders rename, batch rename and duplicate entries.
func contextRenameActions(m *ui.Menu, coord *app.Coordinator, pane *app.PaneState, i int) {
	if m.Item("重命名").Chosen() {
		coord.StartRename(pane, i)
	}
	if len(coord.SelectedItems(pane)) > 1 {
		if m.Item("批量重命名 (⇧⌘R)").Chosen() {
			coord.OpenBatchRename(pane)
		}
	}
	if m.Item("创建副本").Chosen() {
		coord.DuplicateSelected(pane)
	}
}

// contextTagTrashActions renders Finder color tags submenu and trash entry.
func contextTagTrashActions(m *ui.Menu, coord *app.Coordinator, pane *app.PaneState, item domain.FileInfo) {
	m.Submenu("标记颜色", func(sub *ui.Menu) {
		tags := []struct {
			Idx  int
			Name string
		}{
			{macos.TagRed, "红色"},
			{macos.TagOrange, "橙色"},
			{macos.TagYellow, "黄色"},
			{macos.TagGreen, "绿色"},
			{macos.TagBlue, "蓝色"},
			{macos.TagPurple, "紫色"},
			{macos.TagGray, "灰色"},
		}
		currentTag := macos.ColorIndexFromTags(item.ColorTags)
		for _, tg := range tags {
			tIdx := tg.Idx
			tName := tg.Name
			if sub.Item(tName).Radio(currentTag == tIdx).Chosen() {
				_ = macos.SetFileTag(item.Path, tIdx)
				applyTagToPaneItem(pane, item.Path, []string{tName})
				if coord.CurrentPreview != nil && coord.CurrentPreview.Path == item.Path {
					coord.CurrentPreview.ColorTag = tIdx
				}
				coord.Invalidate()
			}
		}
		sub.Separator()
		if sub.Item("清除颜色").Chosen() {
			_ = macos.RemoveFileTags(item.Path)
			applyTagToPaneItem(pane, item.Path, nil)
			if coord.CurrentPreview != nil && coord.CurrentPreview.Path == item.Path {
				coord.CurrentPreview.ColorTag = macos.TagNone
			}
			coord.Invalidate()
		}
	})

	if m.Item("移到废纸篓 (⌘⌫)").Chosen() {
		coord.TrashSelected(pane)
	}
}

// ShowEmptyAreaContextMenu renders the context menu for background blank area.
func ShowEmptyAreaContextMenu(m *ui.Menu, coord *app.Coordinator, pane *app.PaneState) {
	currentPath := pane.Path
	if fs.IsArchiveVirtualPath(currentPath) {
		showArchiveEmptyAreaMenu(m, coord, pane)
		return
	}

	if m.Item("新建文件夹 (⇧⌘N)").Chosen() {
		_, _ = coord.CreateNewFolder(pane, "未命名文件夹")
	}

	if m.Item("全选 (⌘A)").Chosen() {
		coord.SelectAll(pane)
	}
	if m.Item("刷新 (⌘R)").Chosen() {
		coord.LoadPane(pane, nil)
	}

	if coord.IsBookmarked(currentPath) {
		if m.Item("从收藏中移除当前文件夹").Chosen() {
			_ = coord.RemoveBookmark(currentPath)
		}
	} else {
		title := filepath.Base(currentPath)
		if m.Item("将当前文件夹添加到收藏").Chosen() {
			_ = coord.AddBookmark(currentPath, title)
		}
	}

	if len(coord.Clipboard.Paths) > 0 {
		if m.Item("粘贴项目 (⌘V)").Chosen() {
			coord.Paste(pane)
		}
	}

	m.Separator()

	if m.Item("复制当前目录绝对路径").Chosen() {
		_ = macos.CopyPath(currentPath, "posix")
	}

	if m.Item("在此处打开终端").Chosen() {
		_ = macos.OpenInTerminal(currentPath)
	}

	if m.Item("从 Finder 导入常用位置...").Chosen() {
		coord.OpenFinderImport()
	}
}

// showArchiveEmptyAreaMenu provides blank area options when inside a virtual archive.
func showArchiveEmptyAreaMenu(m *ui.Menu, coord *app.Coordinator, pane *app.PaneState) {
	if m.Item("全部解压到当前目录所在文件夹").Chosen() {
		coord.ExtractEntireArchive(pane)
	}
	if m.Item("退出压缩包").Chosen() {
		pane.GoUp()
		coord.LoadCurrentTab(nil)
	}
	m.Separator()
	if m.Item("复制当前虚拟路径").Chosen() {
		_ = macos.CopyPath(pane.Path, "posix")
	}
	if m.Item("刷新 (⌘R)").Chosen() {
		coord.LoadCurrentTab(nil)
	}
}

func applyTagToPaneItem(pane *app.PaneState, path string, tags []string) {
	if pane == nil {
		return
	}
	for i := range pane.Entries {
		if pane.Entries[i].Path == path {
			pane.Entries[i].ColorTags = tags
		}
	}
	for i := range pane.DisplayEntries {
		if pane.DisplayEntries[i].Path == path {
			pane.DisplayEntries[i].ColorTags = tags
		}
	}
}
