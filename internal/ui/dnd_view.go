package ui

import (
	"github.com/bighaidao/clover-go/internal/app"
	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

// BindFileDragSource attaches both in-window DnD payload (FileDragPayload)
// and native macOS transfer data (file:// URIs via transfer.FileData).
func BindFileDragSource(c *ui.Context, elem ui.Element, coord *app.Coordinator, pane *app.PaneState, item domain.FileInfo, idx int) ui.Element {
	var paths []string
	if pane.Chosen.Has(item.Path) && pane.Chosen.Len() > 1 {
		for _, entry := range pane.DisplayEntries {
			if pane.Chosen.Has(entry.Path) {
				paths = append(paths, entry.Path)
			}
		}
	} else {
		paths = []string{item.Path}
	}

	elem.Drag(app.FileDragPayload{
		SrcPaneID: pane.ID,
		Paths:     paths,
	})

	elem.DragDataFrom(func() transfer.Data {
		d, _ := transfer.FileData(paths...)
		return d
	}, transfer.DragOptions{Operations: transfer.Copy | transfer.Move})

	return elem
}

// transferActionFor resolves the drag action from user preferences (default: move).
func transferActionFor(coord *app.Coordinator) app.TransferAction {
	if coord.Prefs.Get().DragAction == "copy" {
		return app.ActionCopy
	}
	return app.ActionMove
}

// BindFileDropTarget binds both internal and native external drop targets to elem.
// Returns true if items are currently being dragged over elem.
func BindFileDropTarget(c *ui.Context, elem ui.Element, targetDirPath string, coord *app.Coordinator) bool {
	isOver := false

	// 1. In-app drop (from other pane, tab, or folder)
	if payload, dropped := ui.Drop[app.FileDragPayload](elem); dropped {
		coord.TransferFiles(payload.Paths, targetDirPath, transferActionFor(coord))
	} else if _, over := ui.DragOver[app.FileDragPayload](elem); over {
		isOver = true
	}

	// 2. Native OS drop (from external Finder, Desktop, WeChat, etc.)
	if extFiles := elem.DroppedFiles(); len(extFiles) > 0 {
		coord.TransferFiles(extFiles, targetDirPath, transferActionFor(coord))
	} else if elem.FileDragOver() {
		isOver = true
	}

	return isOver
}
