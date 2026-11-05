package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bighaidao/clover-go/internal/fs"
	"github.com/bighaidao/clover-go/internal/macos"
	"github.com/egoist/mygo"
)

// FileDragPayload represents file transfer data during drag and drop within CloverGo.
type FileDragPayload struct {
	SrcPaneID string
	Paths     []string
}

// dragCount implements the badge counter interface expected by MyGo renderer.
func (p FileDragPayload) dragCount() int {
	return len(p.Paths)
}

// TransferAction specifies whether items should be moved or copied.
type TransferAction int

const (
	ActionMove TransferAction = iota
	ActionCopy
)

// ConflictChoice represents the user resolution for a filename collision.
type ConflictChoice int

const (
	ChoiceReplace ConflictChoice = iota
	ChoiceKeepBoth
	ChoiceSkip
	ChoiceCancel
)

// PendingConflict carries metadata for the interactive conflict dialog.
type PendingConflict struct {
	SrcPath    string
	DstDir     string
	DstPath    string
	Action     TransferAction
	SrcSize    int64
	SrcModTime time.Time
	DstSize    int64
	DstModTime time.Time
	Remaining  []string
}

// Toast displays a transient HUD notification at the bottom for 3 seconds.
func (c *Coordinator) Toast(msg string) {
	c.ToastMessage = msg
	c.ToastPending = true // 下一帧由 window_view 桥接到 mygo 原生 c.Toast
	c.Invalidate()
	if c.toastTimer != nil {
		c.toastTimer.Stop()
	}
	c.toastTimer = time.AfterFunc(3*time.Second, func() {
		c.runOnMain(func() {
			if c.ToastMessage == msg {
				c.ToastMessage = ""
				c.Invalidate()
			}
		})
	})
}

// TransferFiles dispatches file move or copy operations with validation and collision checks.
func (c *Coordinator) TransferFiles(srcs []string, dstDir string, action TransferAction) {
	if len(srcs) == 0 {
		return
	}
	// 转移会话互斥：冲突弹窗挂起与后台批次共用 TransferBusy，
	// 避免两个批次并发写同一目标目录产生 TOCTOU 重名覆盖。
	if c.TransferBusy {
		c.Toast("文件转移进行中，请稍候…")
		return
	}
	c.TransferBusy = true
	if err := fs.ValidateTransfer(srcs, dstDir); err != nil {
		c.TransferBusy = false
		c.Toast(err.Error())
		return
	}

	// Filter out no-op same directory moves
	var validSrcs []string
	for _, src := range srcs {
		if action == ActionMove && fs.IsSameDirectory(src, dstDir) {
			continue
		}
		validSrcs = append(validSrcs, src)
	}
	if len(validSrcs) == 0 {
		c.TransferBusy = false
		return
	}

	conflictPref := c.Prefs.Get().ConflictAction
	if conflictPref == "overwrite" {
		c.executeTransferBatch(validSrcs, dstDir, action, fs.PolicyOverwrite)
		return
	}
	if conflictPref == "rename" {
		c.executeTransferBatch(validSrcs, dstDir, action, fs.PolicyKeepBoth)
		return
	}

	// Default: "ask" - check for first collision
	c.checkAndPromptConflict(validSrcs, dstDir, action)
}

func (c *Coordinator) checkAndPromptConflict(srcs []string, dstDir string, action TransferAction) {
	for i, src := range srcs {
		target := filepath.Join(dstDir, filepath.Base(src))
		dstFi, err := os.Lstat(target)
		if err == nil {
			// Found conflict
			srcFi, _ := os.Lstat(src)
			var srcSize int64
			var srcMod time.Time
			if srcFi != nil {
				srcSize = srcFi.Size()
				srcMod = srcFi.ModTime()
			}
			c.PendingConflict = &PendingConflict{
				SrcPath:    src,
				DstDir:     dstDir,
				DstPath:    target,
				Action:     action,
				SrcSize:    srcSize,
				SrcModTime: srcMod,
				DstSize:    dstFi.Size(),
				DstModTime: dstFi.ModTime(),
				Remaining:  srcs[i+1:],
			}
			c.ShowConflictModal = true
			c.ConflictApplyToAll = false
			c.Invalidate()
			return
		}
	}

	// No conflict among remaining items: execute straight
	c.executeTransferBatch(srcs, dstDir, action, fs.PolicyKeepBoth)
}

// ResolveConflict applies the chosen resolution to the pending conflict.
func (c *Coordinator) ResolveConflict(choice ConflictChoice, applyAll bool) {
	conflict := c.PendingConflict
	c.ShowConflictModal = false
	c.PendingConflict = nil
	if conflict == nil || choice == ChoiceCancel {
		// 取消或无挂起项时立即结束整个转移会话，释放互斥标志
		c.TransferBusy = false
		c.Invalidate()
		return
	}

	policy := fs.PolicyKeepBoth
	switch choice {
	case ChoiceReplace:
		policy = fs.PolicyOverwrite
	case ChoiceKeepBoth:
		policy = fs.PolicyKeepBoth
	case ChoiceSkip:
		policy = fs.PolicySkip
	}

	// Perform single transfer for current item
	go func() {
		var err error
		if conflict.Action == ActionMove {
			_, err = fs.MoveItem(conflict.SrcPath, conflict.DstDir, policy)
		} else {
			_, err = fs.CopyItem(conflict.SrcPath, conflict.DstDir, policy)
		}

		c.runOnMain(func() {
			if err != nil {
				// 当前项失败：终止队列并明确反馈，避免静默吞掉剩余待决项
				c.TransferBusy = false
				c.LoadCurrentTab(nil)
				c.Toast(fmt.Sprintf("转移失败：%v", err))
				return
			}
			if len(conflict.Remaining) > 0 {
				if applyAll {
					c.executeTransferBatch(conflict.Remaining, conflict.DstDir, conflict.Action, policy)
				} else {
					c.checkAndPromptConflict(conflict.Remaining, conflict.DstDir, conflict.Action)
				}
			} else {
				c.TransferBusy = false
				c.LoadCurrentTab(nil)
			}
		})
	}()
}

func (c *Coordinator) executeTransferBatch(srcs []string, dstDir string, action TransferAction, policy fs.ConflictPolicy) {
	go func() {
		verb := "复制"
		if action == ActionMove {
			verb = "移动"
		}
		// 收集失败项：静默失败会导致用户误以为转移成功，必须明确反馈
		failed := 0
		var firstErr error
		for _, src := range srcs {
			var err error
			if action == ActionMove {
				_, err = fs.MoveItem(src, dstDir, policy)
			} else {
				_, err = fs.CopyItem(src, dstDir, policy)
			}
			if err != nil {
				failed++
				if firstErr == nil {
					firstErr = err
				}
			}
		}
		c.runOnMain(func() {
			c.TransferBusy = false
			c.LoadCurrentTab(nil)
			if failed > 0 {
				c.Toast(fmt.Sprintf("%s完成 %d 项，失败 %d 项：%v", verb, len(srcs)-failed, failed, firstErr))
			}
		})
	}()
}

// CopySelectedToOppositePane transfers selected items to the other pane in dual-pane mode.
func (c *Coordinator) CopySelectedToOppositePane(tab *TabState) {
	if tab == nil || !tab.IsDual || tab.RightPane == nil {
		return
	}
	srcPane := tab.ActivePane()
	targetPane := tab.OppositePane()
	items := c.SelectedItems(srcPane)
	if len(items) == 0 || targetPane == nil {
		return
	}
	paths := make([]string, len(items))
	for i, it := range items {
		paths[i] = it.Path
	}
	c.TransferFiles(paths, targetPane.Path, ActionCopy)
}

// MoveSelectedToOppositePane moves selected items to the other pane in dual-pane mode.
func (c *Coordinator) MoveSelectedToOppositePane(tab *TabState) {
	if tab == nil || !tab.IsDual || tab.RightPane == nil {
		return
	}
	srcPane := tab.ActivePane()
	targetPane := tab.OppositePane()
	items := c.SelectedItems(srcPane)
	if len(items) == 0 || targetPane == nil {
		return
	}
	paths := make([]string, len(items))
	for i, it := range items {
		paths[i] = it.Path
	}
	c.TransferFiles(paths, targetPane.Path, ActionMove)
}

// SelectAll selects all visible items in the specified pane.
func (c *Coordinator) SelectAll(pane *PaneState) {
	if pane == nil || len(pane.DisplayEntries) == 0 {
		return
	}
	pane.Chosen.Clear()
	for _, it := range pane.DisplayEntries {
		pane.Chosen.Add(it.Path)
	}
	pane.SelectedIdx = 0
	c.Invalidate()
}

// CopySelectedPath copies POSIX paths of selected items to clipboard.
func (c *Coordinator) CopySelectedPath(pane *PaneState) {
	items := c.SelectedItems(pane)
	if len(items) == 0 {
		return
	}
	var paths []string
	for _, it := range items {
		paths = append(paths, it.Path)
	}
	if len(paths) == 1 {
		_ = macos.CopyPath(paths[0], "posix")
	} else if c.WriteSysClipboard != nil {
		_ = c.WriteSysClipboard(paths...)
	}
	c.Toast(fmt.Sprintf("已复制 %d 个路径", len(paths)))
}

// OpenSelectedItem opens the currently highlighted item in the given pane.
func (c *Coordinator) OpenSelectedItem(pane *PaneState) {
	if pane == nil || pane.SelectedIdx < 0 || pane.SelectedIdx >= len(pane.DisplayEntries) {
		return
	}
	item := pane.DisplayEntries[pane.SelectedIdx]
	if item.IsAppBundle {
		_ = mygo.Shell.OpenPath(item.Path)
		return
	}
	if fs.IsArchiveFile(item.Path) && !fs.IsArchiveVirtualPath(item.Path) {
		vPath := fs.FormatArchiveVirtualPath(item.Path, "")
		c.Tabs.NavigatePane(c.Tabs.ActiveIndex, pane, vPath)
		c.LoadCurrentTab(nil)
		return
	}
	if fs.IsArchiveVirtualPath(item.Path) {
		if item.IsDir {
			c.Tabs.NavigatePane(c.Tabs.ActiveIndex, pane, item.Path)
			c.LoadCurrentTab(nil)
		} else {
			if tmp, err := fs.ExtractArchiveFileToTemp(item.Path); err == nil {
				_ = mygo.Shell.OpenPath(tmp)
			}
		}
		return
	}
	if item.IsDir {
		c.Tabs.NavigatePane(c.Tabs.ActiveIndex, pane, item.Path)
		c.LoadCurrentTab(nil)
	} else {
		_ = mygo.Shell.OpenPath(item.Path)
	}
}
