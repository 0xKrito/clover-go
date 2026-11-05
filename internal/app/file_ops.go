package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/bighaidao/clover-go/internal/fs"
	"github.com/egoist/mygo"
)

// 本文件收敛全部"条目级"文件操作（选择/剪贴板/压缩/废纸篓/重命名）。
// 所有方法都以 *PaneState 为操作目标：双栏模式下操作永远落在用户
// 正在交互的窗格，而不是固定的左栏（阶段一 review 的 P0 根因）。

// FilenameStemRunes returns the rune count of the filename stem before its extension
// (Finder semantics: folders, dotfiles like ".gitignore", or files without extension select all).
func FilenameStemRunes(name string, isDir bool) int {
	total := len([]rune(name))
	if isDir {
		return total
	}
	dot := strings.LastIndexByte(name, '.')
	if dot <= 0 {
		return total
	}
	return len([]rune(name[:dot]))
}

// SelectedItems returns the selected items of the given pane: multi-selection
// first, falling back to the single cursor selection.
func (c *Coordinator) SelectedItems(pane *PaneState) []domain.FileInfo {
	if pane == nil {
		return nil
	}
	var items []domain.FileInfo
	if pane.Chosen.Len() > 0 {
		for _, entry := range pane.DisplayEntries {
			if pane.Chosen.Has(entry.Path) {
				items = append(items, entry)
			}
		}
	}
	if len(items) == 0 && pane.SelectedIdx >= 0 && pane.SelectedIdx < len(pane.DisplayEntries) {
		items = append(items, pane.DisplayEntries[pane.SelectedIdx])
	}
	return items
}

// clipboardPaths extracts the selected paths and stages them on the internal
// clipboard; returns nil when the pane has no selection.
func (c *Coordinator) clipboardPaths(pane *PaneState, op ClipboardOp) []string {
	items := c.SelectedItems(pane)
	if len(items) == 0 {
		return nil
	}
	paths := make([]string, len(items))
	for i, item := range items {
		paths[i] = item.Path
	}
	c.Clipboard = InternalClipboard{Op: op, Paths: paths}
	return paths
}

// CopySelected puts the selected items onto internal and system clipboards.
func (c *Coordinator) CopySelected(pane *PaneState) {
	paths := c.clipboardPaths(pane, ClipCopy)
	if paths == nil {
		return
	}
	if c.WriteSysClipboard != nil {
		_ = c.WriteSysClipboard(paths...)
	}
}

// CutSelected stages the selected items for moving onto internal and system clipboards.
func (c *Coordinator) CutSelected(pane *PaneState) {
	paths := c.clipboardPaths(pane, ClipCut)
	if paths == nil {
		return
	}
	if c.WriteSysClipboard != nil {
		_ = c.WriteSysClipboard(paths...)
	}
}

// Paste executes copy or move from clipboard into the pane's directory.
func (c *Coordinator) Paste(pane *PaneState) {
	if pane == nil {
		return
	}
	paths := c.Clipboard.Paths
	if len(paths) == 0 && c.ReadSysClipboard != nil {
		if sysFiles, err := c.ReadSysClipboard(); err == nil && len(sysFiles) > 0 {
			paths = sysFiles
		}
	}
	if len(paths) == 0 {
		return
	}

	isCut := c.Clipboard.Op == ClipCut
	dstDir := pane.Path

	go func() {
		for _, src := range paths {
			if isCut {
				_, _ = fs.MoveItem(src, dstDir, fs.PolicyKeepBoth)
			} else {
				_, _ = fs.CopyItem(src, dstDir, fs.PolicyKeepBoth)
			}
		}
		c.runOnMain(func() {
			if isCut {
				c.Clipboard = InternalClipboard{}
			}
			c.LoadCurrentTab(nil)
		})
	}()
}

// DuplicateSelected duplicates all selected items in the pane.
func (c *Coordinator) DuplicateSelected(pane *PaneState) {
	items := c.SelectedItems(pane)
	if len(items) == 0 {
		return
	}
	go func() {
		for _, it := range items {
			_, _ = fs.DuplicateItem(it.Path)
		}
		c.runOnMain(func() {
			c.LoadCurrentTab(nil)
		})
	}()
}

// TrashSelected moves all selected items to Trash.
func (c *Coordinator) TrashSelected(pane *PaneState) {
	items := c.SelectedItems(pane)
	if len(items) == 0 {
		return
	}
	for _, it := range items {
		_ = mygo.Shell.TrashItem(it.Path)
	}
	pane.Chosen.Clear()
	pane.SelectedIdx = -1
	c.LoadCurrentTab(nil)
}

// CompressSelected compresses selected items into a single .zip using macOS
// ditto. Multi-selection must go through one CreateZipMulti call: ditto
// overwrites an existing archive, so per-item calls into the same target
// would silently keep only the last entry.
func (c *Coordinator) CompressSelected(pane *PaneState) {
	items := c.SelectedItems(pane)
	if len(items) == 0 {
		return
	}
	dstDir := pane.Path

	var zipName string
	if len(items) == 1 {
		zipName = items[0].Name + ".zip"
	} else {
		zipName = "归档.zip"
	}
	targetZip := fs.NextAvailableName(filepath.Join(dstDir, zipName))

	paths := make([]string, len(items))
	for i, it := range items {
		paths[i] = it.Path
	}

	go func() {
		if err := fs.CreateZipMulti(paths, targetZip); err != nil {
			log.Printf("压缩失败 %v: %v", paths, err)
		}
		c.runOnMain(func() {
			c.LoadCurrentTab(nil)
		})
	}()
}

// ExtractSelected extracts selected zip archives into the pane's directory.
func (c *Coordinator) ExtractSelected(pane *PaneState) {
	items := c.SelectedItems(pane)
	if len(items) == 0 {
		return
	}
	dstDir := pane.Path
	go func() {
		for _, it := range items {
			if fs.IsArchiveFile(it.Name) {
				_ = fs.ExtractArchive(it.Path, dstDir)
			}
		}
		c.runOnMain(func() { c.LoadCurrentTab(nil) })
	}()
}

// ExtractVirtualSelected extracts selected items from an archive into the parent folder on disk.
func (c *Coordinator) ExtractVirtualSelected(pane *PaneState) {
	if pane == nil || !fs.IsArchiveVirtualPath(pane.Path) {
		return
	}
	archivePath, _, ok := fs.ParseArchiveVirtualPath(pane.Path)
	if !ok {
		return
	}
	destDir := filepath.Dir(archivePath)
	items := c.SelectedItems(pane)
	go func() {
		for _, it := range items {
			_ = fs.ExtractVirtualItem(it.Path, destDir)
		}
		c.runOnMain(func() { c.LoadCurrentTab(nil) })
	}()
}

// ExtractEntireArchive extracts all entries from the archive to the disk folder next to the archive.
func (c *Coordinator) ExtractEntireArchive(pane *PaneState) {
	if pane == nil || !fs.IsArchiveVirtualPath(pane.Path) {
		return
	}
	archivePath, _, ok := fs.ParseArchiveVirtualPath(pane.Path)
	if !ok {
		return
	}
	destDir := filepath.Dir(archivePath)
	go func() {
		_ = fs.ExtractArchive(archivePath, destDir)
		c.runOnMain(func() {
			pane.Navigate(destDir)
			c.LoadCurrentTab(nil)
		})
	}()
}

// CreateNewFolder creates a new folder under the pane's directory and refreshes.
// It selects the new folder and immediately enters inline rename on it, as Finder does.
func (c *Coordinator) CreateNewFolder(pane *PaneState, name string) (string, error) {
	if pane == nil {
		return "", fmt.Errorf("pane is nil")
	}
	target, err := fs.CreateDir(pane.Path, name)
	if err != nil {
		return "", err
	}
	// selectAndRename runs once against the current listing (when the folder is
	// already in it) and again after the async reload, so the inline rename is
	// always staged on the fresh row index.
	selectAndRename := func() {
		idx := -1
		for i, it := range pane.DisplayEntries {
			if it.Path == target {
				idx = i
				break
			}
		}
		if idx < 0 {
			return
		}
		pane.Chosen.Clear()
		pane.Chosen.Add(target)
		pane.SelectedIdx = idx
		// 已在同一行重命名时保留用户已输入的草稿，不重复重置。
		if c.RenamingPane != pane || c.RenamingIdx != idx {
			c.StartRename(pane, idx)
		}
		c.Invalidate()
	}
	selectAndRename()
	c.LoadCurrentTab(selectAndRename)
	return target, nil
}

// StartRename enters inline rename mode for the item at idx in the given pane.
func (c *Coordinator) StartRename(pane *PaneState, idx int) {
	if pane == nil || idx < 0 || idx >= len(pane.DisplayEntries) {
		return
	}
	if fs.IsArchiveVirtualPath(pane.Path) {
		c.Toast("压缩包内为只读预览，不支持重命名")
		return
	}
	c.RenamingPane = pane
	c.RenamingIdx = idx
	c.RenameDraft = pane.DisplayEntries[idx].Name
	c.RenameSelectPending = true
	c.RenameFocusedOnce = false
}

// CommitRename attempts to execute inline rename and reloads on success.
// The guard requires the rename to be staged on this exact pane, so a commit
// can never hit a sibling pane's row (dual-pane P0 fix).
func (c *Coordinator) CommitRename(pane *PaneState) (string, error) {
	if pane == nil || c.RenamingPane != pane || c.RenamingIdx < 0 || c.RenamingIdx >= len(pane.DisplayEntries) {
		return "", nil
	}
	item := pane.DisplayEntries[c.RenamingIdx]
	newName := strings.TrimSpace(c.RenameDraft)
	c.CancelRename()

	if newName == "" || newName == item.Name {
		return item.Path, nil
	}

	newPath, err := fs.RenameItem(item.Path, newName)
	if err != nil {
		c.Toast(fmt.Sprintf("重命名失败: %v", err))
		return "", err
	}
	// 刷新后重新选中重命名后的行，避免重命名完成后选中态丢失或落到别的条目上。
	c.LoadCurrentTab(func() {
		idx := -1
		for i, it := range pane.DisplayEntries {
			if it.Path == newPath {
				idx = i
				break
			}
		}
		if idx < 0 {
			return
		}
		pane.Chosen.Clear()
		pane.Chosen.Add(newPath)
		pane.SelectedIdx = idx
		c.Invalidate()
	})
	return newPath, nil
}

// CancelRename exits inline rename without changes.
func (c *Coordinator) CancelRename() {
	c.RenamingPane = nil
	c.RenamingIdx = -1
	c.RenameDraft = ""
	c.RenameSelectPending = false
	c.RenameFocusedOnce = false
}

// BatchRenameOptions returns the single canonical option set shared by the
// live preview and the executor, keeping the modal WYSIWYG.
func (c *Coordinator) BatchRenameOptions() domain.BatchRenameOptions {
	return domain.BatchRenameOptions{
		Mode:        c.BatchRenameMode,
		BaseName:    c.BatchRenameBase,
		StartNum:    1,
		Padding:     2,
		Separator:   "_",
		Prefix:      c.BatchRenamePrefix,
		Suffix:      c.BatchRenameSuffix,
		FindText:    c.BatchRenameFind,
		ReplaceText: c.BatchRenameReplace,
	}
}

// OpenBatchRename initializes batch renaming modal state with the pane's
// selected items; the staging pane is remembered so a later pane switch
// cannot misdirect the post-run selection cleanup.
func (c *Coordinator) OpenBatchRename(pane *PaneState) {
	items := c.SelectedItems(pane)
	if len(items) == 0 {
		return
	}
	c.BatchRenameItems = items
	c.BatchRenamePane = pane
	c.BatchRenameMode = domain.ModeSequential
	c.BatchRenameBase = "item"
	c.BatchRenameFind = ""
	c.BatchRenameReplace = ""
	c.BatchRenamePrefix = ""
	c.BatchRenameSuffix = ""
	c.BatchRenameError = ""
	c.ShowBatchRenameModal = true
}

// CloseBatchRename cancels batch renaming dialog.
func (c *Coordinator) CloseBatchRename() {
	c.ShowBatchRenameModal = false
	c.BatchRenameItems = nil
	c.BatchRenamePane = nil
	c.BatchRenameError = ""
}

// ExecuteBatchRename calculates and executes renaming for all staged items.
// Validation errors, on-disk collisions and partial failures are stored in
// BatchRenameError and rendered inside the modal instead of failing silently.
func (c *Coordinator) ExecuteBatchRename() error {
	if len(c.BatchRenameItems) == 0 {
		c.CloseBatchRename()
		return nil
	}

	pairs, err := domain.CalculateBatchRename(c.BatchRenameItems, c.BatchRenameOptions())
	if err != nil {
		c.BatchRenameError = err.Error()
		return err
	}

	// On-disk collision guard: never overwrite an existing file that is not
	// part of this batch. Case-only renames are exempt (macOS APFS/HFS+ are
	// case-insensitive and os.Rename handles them in place).
	if collisions := checkBatchRenameCollisions(pairs); len(collisions) > 0 {
		err := fmt.Errorf("目标名与现有文件冲突: %s", strings.Join(collisions, ", "))
		c.BatchRenameError = err.Error()
		return err
	}

	var failed []string
	for _, p := range pairs {
		if !p.Changed {
			continue
		}
		if rErr := os.Rename(p.OriginalPath, p.NewPath); rErr != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", p.OldName, rErr))
		}
	}
	if len(failed) > 0 {
		err := fmt.Errorf("%d 项重命名失败: %s", len(failed), strings.Join(failed, "; "))
		c.BatchRenameError = err.Error()
		c.LoadCurrentTab(nil)
		return err
	}

	// Capture the staging pane before CloseBatchRename clears it.
	stagedPane := c.BatchRenamePane
	c.CloseBatchRename()
	if stagedPane != nil {
		stagedPane.Chosen.Clear()
		stagedPane.SelectedIdx = -1
	}
	c.LoadCurrentTab(nil)
	return nil
}

// checkBatchRenameCollisions returns target names that already exist on disk.
// A target colliding with another staged source is also rejected: sequential
// os.Rename execution would corrupt such swap-style batches.
func checkBatchRenameCollisions(pairs []domain.RenamePair) []string {
	var collisions []string
	for _, p := range pairs {
		if !p.Changed {
			continue
		}
		// Case-only rename resolves to the source file itself on a
		// case-insensitive FS; safe to proceed.
		if strings.EqualFold(p.OriginalPath, p.NewPath) {
			continue
		}
		if _, err := os.Stat(p.NewPath); err == nil {
			collisions = append(collisions, p.NewName)
		}
	}
	return collisions
}
