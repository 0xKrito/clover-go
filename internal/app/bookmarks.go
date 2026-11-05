package app

import (
	"path/filepath"

	"github.com/bighaidao/clover-go/internal/config"
	"github.com/bighaidao/clover-go/internal/domain"
)

// AddBookmark adds a path to user bookmarks if not already added.
func (c *Coordinator) AddBookmark(path string, name string) error {
	if c.Workspaces == nil {
		return nil
	}
	cleanPath := domain.NormalizePath(path)
	if name == "" {
		name = filepath.Base(cleanPath)
		if name == "/" || name == "." {
			name = "Macintosh HD"
		}
	}
	ws := c.Workspaces.Get()
	for _, b := range ws.Bookmarks {
		if domain.NormalizePath(b.Path) == cleanPath {
			return nil
		}
	}
	ws.Bookmarks = append(ws.Bookmarks, config.BookmarkItem{
		Name: name,
		Path: cleanPath,
	})
	if err := c.Workspaces.Save(ws); err != nil {
		return err
	}
	c.Invalidate()
	return nil
}

// RemoveBookmark removes a directory from persistent bookmarks/favorites.
func (c *Coordinator) RemoveBookmark(path string) error {
	if c.Workspaces == nil {
		return nil
	}
	cleanPath := domain.NormalizePath(path)
	ws := c.Workspaces.Get()
	filtered := make([]config.BookmarkItem, 0, len(ws.Bookmarks))
	for _, b := range ws.Bookmarks {
		if domain.NormalizePath(b.Path) != cleanPath {
			filtered = append(filtered, b)
		}
	}
	ws.Bookmarks = filtered
	if err := c.Workspaces.Save(ws); err != nil {
		return err
	}
	c.Invalidate()
	return nil
}

// IsBookmarked checks whether a path is already saved in bookmarks/favorites.
func (c *Coordinator) IsBookmarked(path string) bool {
	if c.Workspaces == nil {
		return false
	}
	cleanPath := domain.NormalizePath(path)
	ws := c.Workspaces.Get()
	for _, b := range ws.Bookmarks {
		if domain.NormalizePath(b.Path) == cleanPath {
			return true
		}
	}
	return false
}
