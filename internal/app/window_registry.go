package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/egoist/mygo"
)

// WindowSessionSnapshot records open tabs for a window during exit.
type WindowSessionSnapshot struct {
	WindowID  string   `json:"window_id"`
	Tabs      []string `json:"tabs"`
	ActiveTab int      `json:"active_tab"`
}

// SessionData holds the array of window snapshots.
type SessionData struct {
	Windows []WindowSessionSnapshot `json:"windows"`
}

// WindowHandle binds a window instance with its coordinator.
type WindowHandle struct {
	ID    string
	Win   *mygo.Window
	Coord *Coordinator
}

// WindowRegistry manages the global set of open windows.
type WindowRegistry struct {
	mu          sync.RWMutex
	windows     map[string]*WindowHandle
	sessionPath string
	onSpawn     func(tab *TabState, pos *mygo.Point)
}

// NewWindowRegistry creates a new window registry.
func NewWindowRegistry(sessionPath string, spawner func(tab *TabState, pos *mygo.Point)) *WindowRegistry {
	return &WindowRegistry{
		windows:     make(map[string]*WindowHandle),
		sessionPath: sessionPath,
		onSpawn:     spawner,
	}
}

// Register adds a window to the registry.
func (r *WindowRegistry) Register(id string, win *mygo.Window, coord *Coordinator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.windows[id] = &WindowHandle{
		ID:    id,
		Win:   win,
		Coord: coord,
	}
}

// Unregister removes a window and saves session if none remain.
func (r *WindowRegistry) Unregister(id string) {
	r.mu.Lock()
	delete(r.windows, id)
	count := len(r.windows)
	r.mu.Unlock()

	if count == 0 {
		_ = r.SaveSession()
	}
}

// Count returns the number of active windows.
func (r *WindowRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.windows)
}

// DetachTab spawns a new window containing only the detached tab.
func (r *WindowRegistry) DetachTab(sourceWinID string, tabIdx int, pos *mygo.Point) {
	r.mu.Lock()
	source, ok := r.windows[sourceWinID]
	r.mu.Unlock()
	if !ok || source.Coord == nil {
		return
	}

	tm := source.Coord.Tabs
	if tabIdx < 0 || tabIdx >= len(tm.Tabs) || len(tm.Tabs) <= 1 {
		return // Do not detach if it's the only tab
	}

	tab := tm.Tabs[tabIdx]
	tm.Tabs = append(tm.Tabs[:tabIdx], tm.Tabs[tabIdx+1:]...)
	if tm.ActiveIndex >= len(tm.Tabs) {
		tm.ActiveIndex = len(tm.Tabs) - 1
	}

	if r.onSpawn != nil {
		r.onSpawn(tab, pos)
	}
}

// SaveSession serializes currently open windows and tabs to disk.
func (r *WindowRegistry) SaveSession() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var data SessionData
	for _, h := range r.windows {
		if h.Coord == nil {
			continue
		}
		tm := h.Coord.Tabs
		paths := make([]string, len(tm.Tabs))
		for i, t := range tm.Tabs {
			paths[i] = t.Path
		}
		data.Windows = append(data.Windows, WindowSessionSnapshot{
			WindowID:  h.ID,
			Tabs:      paths,
			ActiveTab: tm.ActiveIndex,
		})
	}

	if r.sessionPath == "" {
		return nil
	}

	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	_ = os.MkdirAll(filepath.Dir(r.sessionPath), 0755)
	return os.WriteFile(r.sessionPath, raw, 0644)
}

// LoadSession reads persistent window session from disk.
func (r *WindowRegistry) LoadSession() (*SessionData, error) {
	if r.sessionPath == "" {
		return nil, os.ErrNotExist
	}
	raw, err := os.ReadFile(r.sessionPath)
	if err != nil {
		return nil, err
	}
	var data SessionData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
