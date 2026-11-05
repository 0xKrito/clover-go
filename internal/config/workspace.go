package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// BookmarkItem represents a pinned directory shortcut or folder in the bookmark bar.
type BookmarkItem struct {
	Name     string         `json:"name"`
	Path     string         `json:"path,omitempty"`
	Children []BookmarkItem `json:"children,omitempty"`
}

// WorkspacePaneSnapshot records the multi-pane navigation state of a single tab.
type WorkspacePaneSnapshot struct {
	LeftPath  string `json:"left_path"`
	RightPath string `json:"right_path,omitempty"`
	IsDual    bool   `json:"is_dual"`
	ActiveIdx int    `json:"active_idx"`
}

// NamedWorkspace groups a collection of open directories and pane layout under a single name.
type NamedWorkspace struct {
	ID        string                  `json:"id"`
	Name      string                  `json:"name"`
	Paths     []string                `json:"paths,omitempty"`
	Tabs      []WorkspacePaneSnapshot `json:"tabs,omitempty"`
	ActiveTab int                     `json:"active_tab,omitempty"`
}

// WorkspaceData holds persistent bookmarks, custom favorites, and workspaces.
type WorkspaceData struct {
	Bookmarks  []BookmarkItem   `json:"bookmarks"`
	Favorites  []string         `json:"favorites"`
	Workspaces []NamedWorkspace `json:"workspaces"`
}

// WorkspaceManager coordinates persistence for bookmarks and workspaces.
type WorkspaceManager struct {
	mu       sync.RWMutex
	filePath string
	data     WorkspaceData
}

// NewWorkspaceManager loads or creates workspace data at filePath.
func NewWorkspaceManager(filePath string) (*WorkspaceManager, error) {
	mgr := &WorkspaceManager{
		filePath: filePath,
		data: WorkspaceData{
			Bookmarks: []BookmarkItem{
				{Name: "主目录", Path: os.Getenv("HOME")},
				{Name: "下载", Path: filepath.Join(os.Getenv("HOME"), "Downloads")},
			},
			Favorites:  []string{},
			Workspaces: []NamedWorkspace{},
		},
	}

	data, err := os.ReadFile(filePath)
	if err == nil {
		_ = json.Unmarshal(data, &mgr.data)
	} else if os.IsNotExist(err) {
		_ = mgr.Save(mgr.data)
	}

	return mgr, nil
}

// Get returns a thread-safe copy of workspace data.
func (m *WorkspaceManager) Get() WorkspaceData {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.data
}

// Save atomically writes workspace data to disk.
func (m *WorkspaceManager) Save(d WorkspaceData) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal workspaces: %w", err)
	}

	dir := filepath.Dir(m.filePath)
	tmpFile, err := os.CreateTemp(dir, "workspaces-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp workspaces file: %w", err)
	}
	tmpName := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("write temp workspaces: %w", err)
	}
	_ = tmpFile.Close()

	if err := os.Rename(tmpName, m.filePath); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("atomic rename workspaces: %w", err)
	}

	m.data = d
	return nil
}

// AddOrUpdateWorkspace inserts or updates a named workspace snapshot.
func (m *WorkspaceManager) AddOrUpdateWorkspace(ws NamedWorkspace) error {
	d := m.Get()
	found := false
	for i, existing := range d.Workspaces {
		if existing.ID == ws.ID || existing.Name == ws.Name {
			d.Workspaces[i] = ws
			found = true
			break
		}
	}
	if !found {
		d.Workspaces = append(d.Workspaces, ws)
	}
	return m.Save(d)
}

// RemoveWorkspace removes a named workspace by ID or name.
func (m *WorkspaceManager) RemoveWorkspace(idOrName string) error {
	d := m.Get()
	var updated []NamedWorkspace
	for _, ws := range d.Workspaces {
		if ws.ID != idOrName && ws.Name != idOrName {
			updated = append(updated, ws)
		}
	}
	d.Workspaces = updated
	return m.Save(d)
}

// AddBookmarks appends new bookmark items, skipping duplicates (paths are
// normalized via filepath.Clean so trailing-slash variants dedupe correctly).
func (m *WorkspaceManager) AddBookmarks(items []BookmarkItem) error {
	d := m.Get()
	seen := make(map[string]bool)
	for _, b := range d.Bookmarks {
		seen[filepath.Clean(b.Path)] = true
	}
	for _, item := range items {
		key := filepath.Clean(item.Path)
		if !seen[key] {
			d.Bookmarks = append(d.Bookmarks, item)
			seen[key] = true
		}
	}
	return m.Save(d)
}
