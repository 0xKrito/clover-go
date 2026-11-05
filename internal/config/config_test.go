package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrefsManager(t *testing.T) {
	tempDir := t.TempDir()
	prefsPath := filepath.Join(tempDir, "prefs.json")

	mgr, err := NewPrefsManager(prefsPath)
	if err != nil {
		t.Fatalf("NewPrefsManager failed: %v", err)
	}

	p := mgr.Get()
	if p.Appearance != "system" {
		t.Errorf("expected system appearance, got %s", p.Appearance)
	}

	p.AccentColor = "#FF3B30"
	p.ShowHidden = true
	if err := mgr.Save(p); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Reload from disk
	mgr2, err := NewPrefsManager(prefsPath)
	if err != nil {
		t.Fatalf("NewPrefsManager 2 failed: %v", err)
	}
	p2 := mgr2.Get()
	if p2.AccentColor != "#FF3B30" || !p2.ShowHidden {
		t.Errorf("saved preferences not restored: %+v", p2)
	}
}

func TestWorkspaceManager(t *testing.T) {
	tempDir := t.TempDir()
	spacesPath := filepath.Join(tempDir, "spaces.json")

	mgr, err := NewWorkspaceManager(spacesPath)
	if err != nil {
		t.Fatalf("NewWorkspaceManager failed: %v", err)
	}

	d := mgr.Get()
	d.Bookmarks = append(d.Bookmarks, BookmarkItem{Name: "Test", Path: "/test"})
	if err := mgr.Save(d); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	mgr2, _ := NewWorkspaceManager(spacesPath)
	d2 := mgr2.Get()
	if len(d2.Bookmarks) != len(d.Bookmarks) {
		t.Errorf("bookmarks count mismatch: %d vs %d", len(d2.Bookmarks), len(d.Bookmarks))
	}
}

func TestPrefsManager_BackwardCompatibility(t *testing.T) {
	tempDir := t.TempDir()
	prefsPath := filepath.Join(tempDir, "legacy_prefs.json")

	// Simulate old prefs file without new fields
	legacyJSON := `{"appearance":"dark","grid_icon_size":96}`
	if err := os.WriteFile(prefsPath, []byte(legacyJSON), 0644); err != nil {
		t.Fatalf("write legacy prefs: %v", err)
	}

	mgr, err := NewPrefsManager(prefsPath)
	if err != nil {
		t.Fatalf("NewPrefsManager failed: %v", err)
	}

	p := mgr.Get()
	if p.SidebarWidth != 155 {
		t.Errorf("expected default SidebarWidth 155, got %v", p.SidebarWidth)
	}
	if p.DualPaneRatio != 50 {
		t.Errorf("expected default DualPaneRatio 50, got %v", p.DualPaneRatio)
	}
	if p.PreviewWidth != 250 {
		t.Errorf("expected default PreviewWidth 250, got %v", p.PreviewWidth)
	}
	if p.DragAction != "move" {
		t.Errorf("expected default DragAction 'move', got %s", p.DragAction)
	}
	if p.ConflictAction != "ask" {
		t.Errorf("expected default ConflictAction 'ask', got %s", p.ConflictAction)
	}
	if p.SortField != "create_time" {
		t.Errorf("expected default SortField 'create_time', got %s", p.SortField)
	}
	if p.SortDirection != "desc" {
		t.Errorf("expected default SortDirection 'desc', got %s", p.SortDirection)
	}

	// Unknown stored sort values must be normalized back to the default.
	badJSON := `{"sort_field":"hacker","sort_direction":"sideways"}`
	if err := os.WriteFile(prefsPath, []byte(badJSON), 0644); err != nil {
		t.Fatalf("write bad prefs: %v", err)
	}
	mgr2, err := NewPrefsManager(prefsPath)
	if err != nil {
		t.Fatalf("NewPrefsManager 2 failed: %v", err)
	}
	p2 := mgr2.Get()
	if p2.SortField != "create_time" || p2.SortDirection != "desc" {
		t.Errorf("invalid sort prefs not normalized: %s/%s", p2.SortField, p2.SortDirection)
	}
}
