package domain

import (
	"testing"
)

func TestCollapseHome(t *testing.T) {
	home := "/Users/developer"
	tests := []struct {
		path     string
		expected string
	}{
		{"/Users/developer", "~"},
		{"/Users/developer/Documents", "~/Documents"},
		{"/Users/developer/Documents/Work", "~/Documents/Work"},
		{"/Library/Application Support", "/Library/Application Support"},
		{"/", "/"},
	}

	for _, tt := range tests {
		got := CollapseHome(tt.path, home)
		if got != tt.expected {
			t.Errorf("CollapseHome(%q, %q) = %q, want %q", tt.path, home, got, tt.expected)
		}
	}
}

func TestSplitPathCrumbs(t *testing.T) {
	home := "/Users/developer"
	crumbs := SplitPathCrumbs("/Users/developer/Projects/clover", home)

	if len(crumbs) != 3 {
		t.Fatalf("expected 3 crumbs, got %d", len(crumbs))
	}

	if crumbs[0].Name != "developer" || !crumbs[0].IsHome {
		t.Errorf("crumb 0 should be home user folder 'developer', got %+v", crumbs[0])
	}
	if crumbs[1].Name != "Projects" || crumbs[1].FullPath != "/Users/developer/Projects" {
		t.Errorf("crumb 1 mismatch: %+v", crumbs[1])
	}
	if crumbs[2].Name != "clover" || !crumbs[2].IsLast {
		t.Errorf("crumb 2 mismatch: %+v", crumbs[2])
	}

	// Archive VFS path crumbs test
	archiveCrumbs := SplitPathCrumbs("/Users/developer/Projects/test.zip#sub/images", home)
	if len(archiveCrumbs) != 5 {
		t.Fatalf("expected 5 archive crumbs, got %d", len(archiveCrumbs))
	}
	if archiveCrumbs[2].Name != "test.zip" || archiveCrumbs[2].FullPath != "/Users/developer/Projects/test.zip#" {
		t.Errorf("archive root crumb mismatch: %+v", archiveCrumbs[2])
	}
	if archiveCrumbs[3].Name != "sub" || archiveCrumbs[3].FullPath != "/Users/developer/Projects/test.zip#sub" {
		t.Errorf("archive sub crumb mismatch: %+v", archiveCrumbs[3])
	}
	if archiveCrumbs[4].Name != "images" || !archiveCrumbs[4].IsLast || archiveCrumbs[4].FullPath != "/Users/developer/Projects/test.zip#sub/images" {
		t.Errorf("archive leaf crumb mismatch: %+v", archiveCrumbs[4])
	}
}

func TestDisambiguateTabs(t *testing.T) {
	tabs := []TabSummary{
		{ID: "tab1", Path: "/Users/developer/projA/src"},
		{ID: "tab2", Path: "/Users/developer/projB/src"},
		{ID: "tab3", Path: "/Users/developer/Downloads"},
	}

	titles := DisambiguateTabs(tabs)

	if titles["tab1"] != "src — projA" {
		t.Errorf("tab1 got %q, want 'src — projA'", titles["tab1"])
	}
	if titles["tab2"] != "src — projB" {
		t.Errorf("tab2 got %q, want 'src — projB'", titles["tab2"])
	}
	if titles["tab3"] != "Downloads" {
		t.Errorf("tab3 got %q, want 'Downloads'", titles["tab3"])
	}
}
