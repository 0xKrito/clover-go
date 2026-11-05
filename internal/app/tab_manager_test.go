package app

import (
	"testing"

	"github.com/bighaidao/clover-go/internal/domain"
)

// TestTabManager_DefaultSort 验证偏好默认排序被应用到新标签页；
// 未设置 DefaultSort 时保持原有 name/asc 回退。
func TestTabManager_DefaultSort(t *testing.T) {
	tm := NewTabManager("/tmp")
	tm.DefaultSort = domain.SortOptions{Field: domain.SortByCreateTime, Direction: domain.SortDesc}

	tab := tm.NewTab("/tmp")
	if tab.PaneState.Sort.Field != domain.SortByCreateTime || tab.PaneState.Sort.Direction != domain.SortDesc {
		t.Fatalf("new tab did not pick up default sort: %+v", tab.PaneState.Sort)
	}

	// 双栏右栏的默认排序由 Coordinator.ToggleDualPane 赋值（dual_pane.go），
	// Manager 层无引用，此处不重复测。

	tmFallback := NewTabManager("/tmp")
	if tmFallback.Tabs[0].PaneState.Sort.Field != domain.SortByName ||
		tmFallback.Tabs[0].PaneState.Sort.Direction != domain.SortAsc {
		t.Fatalf("fallback sort broken: %+v", tmFallback.Tabs[0].PaneState.Sort)
	}
}

func TestTabManager_Lifecycle(t *testing.T) {
	tm := NewTabManager("/Users/demo/Documents")

	if len(tm.Tabs) != 1 {
		t.Fatalf("expected 1 tab, got %d", len(tm.Tabs))
	}
	if tm.ActiveTab().Title != "Documents" {
		t.Errorf("expected title Documents, got %s", tm.ActiveTab().Title)
	}

	// Add second tab
	tab2 := tm.NewTab("/Users/demo/Downloads")
	if len(tm.Tabs) != 2 {
		t.Fatalf("expected 2 tabs, got %d", len(tm.Tabs))
	}
	if tm.ActiveTab().ID != tab2.ID {
		t.Errorf("active tab should be the newly added tab")
	}

	// Close tab 2
	ok := tm.CloseTab(1)
	if !ok || len(tm.Tabs) != 1 {
		t.Fatalf("expected tab closed, got len=%d", len(tm.Tabs))
	}

	// Restore tab 2 (⇧⌘T)
	restored := tm.RestoreLastClosedTab()
	if restored == nil || restored.Path != "/Users/demo/Downloads" {
		t.Fatalf("expected restored Downloads tab, got %+v", restored)
	}
}

func TestTabManager_Pin(t *testing.T) {
	tm := NewTabManager("/Users/demo")
	tm.TogglePin(0)

	if !tm.Tabs[0].IsPinned {
		t.Fatalf("tab should be pinned")
	}

	// Closing pinned tab must be blocked
	closed := tm.CloseTab(0)
	if closed || len(tm.Tabs) != 1 {
		t.Errorf("closing pinned tab should return false and not delete tab")
	}
}

func TestTabManager_History(t *testing.T) {
	tm := NewTabManager("/Users/demo/Projects")
	tm.Navigate(0, "/Users/demo/Projects/clover")

	if !tm.CanGoBack(0) {
		t.Fatalf("expected CanGoBack true")
	}
	if tm.CanGoForward(0) {
		t.Fatalf("expected CanGoForward false")
	}

	backPath := tm.GoBack(0)
	if backPath != "/Users/demo/Projects" {
		t.Errorf("expected back to Projects, got %s", backPath)
	}

	if !tm.CanGoForward(0) {
		t.Fatalf("expected CanGoForward true")
	}
	fwdPath := tm.GoForward(0)
	if fwdPath != "/Users/demo/Projects/clover" {
		t.Errorf("expected fwd to clover, got %s", fwdPath)
	}
}

func TestTabManager_Disambiguate(t *testing.T) {
	tm := NewTabManager("/Users/demo/projA/src")
	tm.NewTab("/Users/demo/projB/src")

	tm.Disambiguate()

	if tm.Tabs[0].Subtitle != "src — projA" {
		t.Errorf("tab 0 subtitle mismatch: got %q", tm.Tabs[0].Subtitle)
	}
	if tm.Tabs[1].Subtitle != "src — projB" {
		t.Errorf("tab 1 subtitle mismatch: got %q", tm.Tabs[1].Subtitle)
	}
}
