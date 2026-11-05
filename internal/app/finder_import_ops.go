package app

import (
	"github.com/bighaidao/clover-go/internal/config"
	"github.com/bighaidao/clover-go/internal/macos"
)

// OpenFinderImport opens the Finder favorites import modal dialog.
func (c *Coordinator) OpenFinderImport() {
	c.ShowFinderImportModal = true
	c.FinderImportTarget = "bookmarks"
	c.FinderImportError = ""
	favs, _ := macos.ReadFinderFavorites()
	c.FinderFavorites = favs
}

// CloseFinderImport closes the Finder favorites import modal.
func (c *Coordinator) CloseFinderImport() {
	c.ShowFinderImportModal = false
	c.FinderImportError = ""
}

// ExecuteFinderImport imports the selected items into CloverGo bookmarks or sidebar favorites.
func (c *Coordinator) ExecuteFinderImport() error {
	var toImport []config.BookmarkItem
	for _, it := range c.FinderFavorites {
		if it.Selected && it.Exists {
			toImport = append(toImport, config.BookmarkItem{
				Name: it.Name,
				Path: it.Path,
			})
		}
	}

	var err error
	if len(toImport) > 0 {
		err = c.Workspaces.AddBookmarks(toImport)
	}

	c.CloseFinderImport()
	c.Invalidate()
	return err
}
