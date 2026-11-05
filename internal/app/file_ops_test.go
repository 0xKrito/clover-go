package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bighaidao/clover-go/internal/domain"
)

// TestSelectedItems_PaneIsolation locks the dual-pane P0 fix: selection is
// per-pane, and operations must read the pane they are handed.
func TestSelectedItems_PaneIsolation(t *testing.T) {
	coord, tempDir := setupTestCoordinator(t)
	defer coord.Close()

	fRight := filepath.Join(tempDir, "right_only.txt")
	_ = os.WriteFile(fRight, []byte("r"), 0644)

	tab := coord.Tabs.ActiveTab()
	tab.ToggleDualPane()
	left, right := tab.Left(), tab.Right()

	left.DisplayEntries = []domain.FileInfo{{Name: "left.txt", Path: filepath.Join(tempDir, "left.txt")}}
	right.DisplayEntries = []domain.FileInfo{{Name: "right_only.txt", Path: fRight}}

	// Selection lives in the RIGHT pane only.
	right.Chosen.Add(fRight)
	right.SelectedIdx = 0

	gotRight := coord.SelectedItems(right)
	if len(gotRight) != 1 || gotRight[0].Path != fRight {
		t.Fatalf("right pane selection lost: %v", gotRight)
	}
	if got := coord.SelectedItems(left); len(got) != 0 {
		t.Fatalf("left pane must not see right-pane selection: %v", got)
	}
}

// TestNavigatePane_TabTitleIsolation ensures right-pane navigation never
// rewrites the tab-level path/title (tab chrome tracks the left pane).
func TestNavigatePane_TabTitleIsolation(t *testing.T) {
	tm := NewTabManager("/tmp")
	tab := tm.ActiveTab()
	tab.ToggleDualPane()

	tm.NavigatePane(tm.ActiveIndex, tab.Right(), "/Users")
	if tab.Path != "/tmp" {
		t.Errorf("right-pane navigation leaked into tab.Path: %s", tab.Path)
	}
	if tab.Title != "tmp" {
		t.Errorf("right-pane navigation leaked into tab.Title: %s", tab.Title)
	}
	if tab.Right().Path != "/Users" {
		t.Errorf("right pane path not updated: %s", tab.Right().Path)
	}

	tm.NavigatePane(tm.ActiveIndex, tab.Left(), "/Users/shared")
	if tab.Path != "/Users/shared" || tab.Title != "shared" {
		t.Errorf("left-pane navigation must update tab chrome: %s %s", tab.Path, tab.Title)
	}
}

// TestPaneNavigate_ClearsSelection verifies Finder semantics: selection never
// survives a directory change within a pane.
func TestPaneNavigate_ClearsSelection(t *testing.T) {
	tm := NewTabManager("/tmp")
	tab := tm.ActiveTab()
	pane := tab.Left()

	pane.DisplayEntries = []domain.FileInfo{{Name: "a.txt", Path: "/tmp/a.txt"}}
	pane.Chosen.Add("/tmp/a.txt")
	pane.SelectedIdx = 0

	tm.NavigatePane(tm.ActiveIndex, pane, "/Users")

	if pane.Chosen.Len() != 0 || pane.SelectedIdx != -1 {
		t.Errorf("stale selection after navigation: chosen=%d idx=%d", pane.Chosen.Len(), pane.SelectedIdx)
	}
}

func TestCheckBatchRenameCollisions(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	existing := filepath.Join(dir, "taken.txt")
	_ = os.WriteFile(src, []byte("1"), 0644)
	_ = os.WriteFile(existing, []byte("2"), 0644)

	// 1. Target occupied by an unrelated on-disk file -> collision.
	pairs := []domain.RenamePair{{
		OriginalPath: src, OldName: "a.txt",
		NewPath: existing, NewName: "taken.txt", Changed: true,
	}}
	if got := checkBatchRenameCollisions(pairs); len(got) != 1 {
		t.Fatalf("expected collision for occupied target, got %v", got)
	}

	// 2. Case-only rename on case-insensitive FS -> exempt.
	casePairs := []domain.RenamePair{{
		OriginalPath: src, OldName: "a.txt",
		NewPath: filepath.Join(dir, "A.TXT"), NewName: "A.TXT", Changed: true,
	}}
	if got := checkBatchRenameCollisions(casePairs); len(got) != 0 {
		t.Fatalf("case-only rename must not collide, got %v", got)
	}

	// 3. Unchanged pairs are ignored.
	same := []domain.RenamePair{{
		OriginalPath: src, OldName: "a.txt",
		NewPath: src, NewName: "a.txt", Changed: false,
	}}
	if got := checkBatchRenameCollisions(same); len(got) != 0 {
		t.Fatalf("unchanged pairs must be skipped, got %v", got)
	}
}

// TestExecuteBatchRename_CollisionBlocked proves the executor refuses to
// overwrite an unrelated on-disk file and surfaces the error in the modal.
func TestExecuteBatchRename_CollisionBlocked(t *testing.T) {
	coord, tempDir := setupTestCoordinator(t)
	defer coord.Close()

	src := filepath.Join(tempDir, "img.png")
	existing := filepath.Join(tempDir, "keep.png")
	_ = os.WriteFile(src, []byte("x"), 0644)
	_ = os.WriteFile(existing, []byte("y"), 0644)

	tab := coord.Tabs.ActiveTab()
	pane := tab.Left()
	pane.DisplayEntries = []domain.FileInfo{{Name: "img.png", Path: src}}
	pane.Chosen.Add(src)

	coord.OpenBatchRename(pane)
	coord.BatchRenameMode = domain.ModeReplace
	coord.BatchRenameFind = "img"
	coord.BatchRenameReplace = "keep"

	if err := coord.ExecuteBatchRename(); err == nil {
		t.Fatalf("expected collision error, got nil")
	}
	if coord.BatchRenameError == "" {
		t.Errorf("BatchRenameError must be populated for the modal")
	}
	if _, err := os.Stat(src); err != nil {
		t.Errorf("source file must remain untouched")
	}
	if !coord.ShowBatchRenameModal {
		t.Errorf("modal must stay open on failure")
	}
}

// TestBatchRenameOptions_PreviewExecutionParity locks WYSIWYG: the option set
// returned by BatchRenameOptions() is exactly what execution applies.
func TestBatchRenameOptions_PreviewExecutionParity(t *testing.T) {
	coord, tempDir := setupTestCoordinator(t)
	defer coord.Close()

	f1 := filepath.Join(tempDir, "one.txt")
	f2 := filepath.Join(tempDir, "two.txt")
	_ = os.WriteFile(f1, []byte("1"), 0644)
	_ = os.WriteFile(f2, []byte("2"), 0644)

	tab := coord.Tabs.ActiveTab()
	pane := tab.Left()
	pane.DisplayEntries = []domain.FileInfo{
		{Name: "one.txt", Path: f1},
		{Name: "two.txt", Path: f2},
	}
	pane.Chosen.Add(f1)
	pane.Chosen.Add(f2)

	coord.OpenBatchRename(pane)

	// Preview side: compute pairs from the shared option set.
	pairs, err := domain.CalculateBatchRename(coord.BatchRenameItems, coord.BatchRenameOptions())
	if err != nil {
		t.Fatalf("preview calculation failed: %v", err)
	}

	// Execution side: run the real rename.
	if err := coord.ExecuteBatchRename(); err != nil {
		t.Fatalf("ExecuteBatchRename error: %v", err)
	}

	for i, p := range pairs {
		if _, err := os.Stat(p.NewPath); err != nil {
			t.Errorf("preview pair %d target missing on disk: %s", i, p.NewPath)
		}
	}
	// Sequential defaults must be item_01 / item_02 (padding 2, "_" separator).
	if pairs[0].NewName != "item_01.txt" || pairs[1].NewName != "item_02.txt" {
		t.Errorf("preview/execution options drifted: %s, %s", pairs[0].NewName, pairs[1].NewName)
	}
}
