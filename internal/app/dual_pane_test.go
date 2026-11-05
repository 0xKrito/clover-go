package app

import (
	"path/filepath"
	"testing"
)

func TestTabState_DualPaneToggleAndSwitch(t *testing.T) {
	tm := NewTabManager("/tmp")
	tab := tm.ActiveTab()

	if tab.IsDual {
		t.Fatalf("expected IsDual to be false initially")
	}
	if tab.ActiveIdx != 0 {
		t.Fatalf("expected ActiveIdx to be 0 initially")
	}

	// 1. Toggle dual pane on
	tab.ToggleDualPane()
	if !tab.IsDual {
		t.Fatalf("expected IsDual to be true after toggle")
	}
	if tab.RightPane == nil {
		t.Fatalf("expected RightPane to be initialized")
	}
	if tab.ActiveIdx != 1 {
		t.Fatalf("expected ActiveIdx to be 1 after opening dual pane")
	}
	if tab.ActivePane() != tab.RightPane {
		t.Fatalf("expected ActivePane to return RightPane when ActiveIdx == 1")
	}

	// 2. Switch focus back to left pane
	tab.SwitchActivePane()
	if tab.ActiveIdx != 0 {
		t.Fatalf("expected ActiveIdx to be 0 after switch")
	}
	if tab.ActivePane() != &tab.PaneState {
		t.Fatalf("expected ActivePane to return LeftPane when ActiveIdx == 0")
	}

	// 3. Navigate right pane
	tab.SwitchActivePane() // ActiveIdx == 1
	subPath := filepath.Join("/tmp", "subfolder")
	tm.Navigate(tm.ActiveIndex, subPath)

	if tab.RightPane.Path != subPath {
		t.Errorf("expected RightPane path to be %s, got %s", subPath, tab.RightPane.Path)
	}
	if tab.Path != "/tmp" {
		t.Errorf("expected LeftPane path to remain /tmp, got %s", tab.Path)
	}

	// 4. Toggle dual pane off
	tab.ToggleDualPane()
	if tab.IsDual {
		t.Fatalf("expected IsDual to be false after second toggle")
	}
	if tab.ActiveIdx != 0 {
		t.Fatalf("expected ActiveIdx to be reset to 0")
	}
	if tab.ActivePane() != &tab.PaneState {
		t.Fatalf("expected ActivePane to be LeftPane")
	}
}

func TestPaneState_ArchiveVFSNavigation(t *testing.T) {
	p := NewPaneState("test-pane", "/tmp/archive.zip#docs/assets")

	if !p.CanGoUp() {
		t.Fatalf("expected CanGoUp to be true in archive subfolder")
	}

	// 1. Up from docs/assets to docs
	p1 := p.GoUp()
	if p1 != "/tmp/archive.zip#docs" {
		t.Errorf("expected /tmp/archive.zip#docs, got %s", p1)
	}

	// 2. Up from docs to archive root
	p2 := p.GoUp()
	if p2 != "/tmp/archive.zip#" {
		t.Errorf("expected /tmp/archive.zip#, got %s", p2)
	}

	// 3. Up from archive root to /tmp on disk
	p3 := p.GoUp()
	if p3 != "/tmp" {
		t.Errorf("expected /tmp, got %s", p3)
	}
}
