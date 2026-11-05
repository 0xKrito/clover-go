package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bighaidao/clover-go/internal/config"
	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/bighaidao/clover-go/internal/icon"
	"github.com/bighaidao/clover-go/internal/macos"
)

func setupTestCoordinator(t *testing.T) (*Coordinator, string) {
	tempDir := t.TempDir()
	prefsFile := filepath.Join(tempDir, "prefs.json")
	spacesFile := filepath.Join(tempDir, "spaces.json")
	thumbsDir := filepath.Join(tempDir, "thumbs")

	prefs, err := config.NewPrefsManager(prefsFile)
	if err != nil {
		t.Fatalf("NewPrefsManager: %v", err)
	}
	spaces, err := config.NewWorkspaceManager(spacesFile)
	if err != nil {
		t.Fatalf("NewWorkspaceManager: %v", err)
	}
	pipeline := icon.NewPipeline(thumbsDir)

	coord := NewCoordinator(tempDir, prefs, spaces, pipeline, func(f func()) {
		f()
	})
	return coord, tempDir
}

func TestCoordinator_SelectedItems(t *testing.T) {
	coord, tempDir := setupTestCoordinator(t)
	defer coord.Close()

	file1 := filepath.Join(tempDir, "a.txt")
	file2 := filepath.Join(tempDir, "b.txt")
	_ = os.WriteFile(file1, []byte("1"), 0644)
	_ = os.WriteFile(file2, []byte("2"), 0644)

	tab := coord.Tabs.ActiveTab()
	pane := tab.Left()
	pane.DisplayEntries = []domain.FileInfo{
		{Name: "a.txt", Path: file1},
		{Name: "b.txt", Path: file2},
	}

	// 1. Single selection fallback
	pane.SelectedIdx = 0
	items := coord.SelectedItems(pane)
	if len(items) != 1 || items[0].Path != file1 {
		t.Errorf("expected single selected file1, got %v", items)
	}

	// 2. Multi-selection via Chosen
	pane.Chosen.Add(file1)
	pane.Chosen.Add(file2)
	items = coord.SelectedItems(pane)
	if len(items) != 2 {
		t.Errorf("expected 2 selected items, got %d", len(items))
	}
}

func TestCoordinator_ClipboardOperations(t *testing.T) {
	coord, tempDir := setupTestCoordinator(t)
	defer coord.Close()

	subDir := filepath.Join(tempDir, "dest")
	_ = os.Mkdir(subDir, 0755)

	srcFile := filepath.Join(tempDir, "copy_me.txt")
	_ = os.WriteFile(srcFile, []byte("hello"), 0644)

	tab := coord.Tabs.ActiveTab()
	pane := tab.Left()
	pane.DisplayEntries = []domain.FileInfo{
		{Name: "copy_me.txt", Path: srcFile},
	}
	pane.SelectedIdx = 0

	// 1. Copy
	coord.CopySelected(pane)
	if coord.Clipboard.Op != ClipCopy || len(coord.Clipboard.Paths) != 1 {
		t.Fatalf("CopySelected failed: %v", coord.Clipboard)
	}

	// 2. Paste into subDir
	pane.Path = subDir
	coord.Paste(pane)

	time.Sleep(50 * time.Millisecond)
	pastedFile := filepath.Join(subDir, "copy_me.txt")
	if _, err := os.Stat(pastedFile); err != nil {
		t.Errorf("pasted file not found at %s", pastedFile)
	}
}

func TestCoordinator_BatchRename(t *testing.T) {
	coord, tempDir := setupTestCoordinator(t)
	defer coord.Close()

	f1 := filepath.Join(tempDir, "img1.png")
	f2 := filepath.Join(tempDir, "img2.png")
	_ = os.WriteFile(f1, []byte("x"), 0644)
	_ = os.WriteFile(f2, []byte("y"), 0644)

	tab := coord.Tabs.ActiveTab()
	pane := tab.Left()
	pane.DisplayEntries = []domain.FileInfo{
		{Name: "img1.png", Path: f1},
		{Name: "img2.png", Path: f2},
	}
	pane.Chosen.Add(f1)
	pane.Chosen.Add(f2)

	coord.OpenBatchRename(pane)
	if !coord.ShowBatchRenameModal || len(coord.BatchRenameItems) != 2 {
		t.Fatalf("OpenBatchRename failed: %v", coord.ShowBatchRenameModal)
	}

	coord.BatchRenameBase = "vacation"
	coord.BatchRenameMode = domain.ModeSequential
	err := coord.ExecuteBatchRename()
	if err != nil {
		t.Fatalf("ExecuteBatchRename error: %v", err)
	}

	new1 := filepath.Join(tempDir, "vacation_01.png")
	new2 := filepath.Join(tempDir, "vacation_02.png")
	if _, err := os.Stat(new1); err != nil {
		t.Errorf("expected %s to exist", new1)
	}
	if _, err := os.Stat(new2); err != nil {
		t.Errorf("expected %s to exist", new2)
	}
}

func TestCoordinator_TagFilter(t *testing.T) {
	coord, tempDir := setupTestCoordinator(t)
	defer coord.Close()

	fRed := filepath.Join(tempDir, "red.txt")
	fNormal := filepath.Join(tempDir, "normal.txt")
	_ = os.WriteFile(fRed, []byte("r"), 0644)
	_ = os.WriteFile(fNormal, []byte("n"), 0644)

	_ = macos.SetFileTag(fRed, macos.TagRed)

	tab := coord.Tabs.ActiveTab()
	// ApplyFilter resolves tags against the FileInfo.ColorTags snapshot that
	// fs.ReadDirPage fills from xattr in its background loader; mirror that
	// real production data flow instead of relying on per-frame xattr reads.
	redTags, _, _ := macos.GetFileTags(fRed)
	tab.Entries = []domain.FileInfo{
		{Name: "red.txt", Path: fRed, ColorTags: redTags},
		{Name: "normal.txt", Path: fNormal},
	}

	coord.FilterTag = macos.TagRed
	coord.ApplyFilter(tab)

	if len(tab.DisplayEntries) != 1 || tab.DisplayEntries[0].Name != "red.txt" {
		t.Errorf("tag filter expected red.txt only, got %v", tab.DisplayEntries)
	}
}

func TestCoordinator_Bookmarks(t *testing.T) {
	coord, tempDir := setupTestCoordinator(t)
	defer coord.Close()

	testDir := filepath.Join(tempDir, "favorite_folder")
	_ = os.Mkdir(testDir, 0755)

	// 1. Initial state
	if coord.IsBookmarked(testDir) {
		t.Errorf("expected not bookmarked initially")
	}

	// 2. Add bookmark
	err := coord.AddBookmark(testDir, "我的项目")
	if err != nil {
		t.Fatalf("AddBookmark error: %v", err)
	}
	if !coord.IsBookmarked(testDir) {
		t.Errorf("expected bookmarked after adding")
	}

	// 3. Duplicate add should be idempotent
	_ = coord.AddBookmark(testDir, "我的项目")
	ws := coord.Workspaces.Get()
	count := 0
	for _, b := range ws.Bookmarks {
		if b.Path == testDir {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 bookmark entry, got %d", count)
	}

	// 4. Remove bookmark
	err = coord.RemoveBookmark(testDir)
	if err != nil {
		t.Fatalf("RemoveBookmark error: %v", err)
	}
	if coord.IsBookmarked(testDir) {
		t.Errorf("expected not bookmarked after removing")
	}
}

func TestCoordinator_DefaultSortAppliedOnStartup(t *testing.T) {
	coord, _ := setupTestCoordinator(t)
	defer coord.Close()

	tab := coord.Tabs.ActiveTab()
	if tab == nil {
		t.Fatalf("expected active tab")
	}
	pane := tab.Left()
	if pane.Sort.Field != domain.SortByCreateTime || pane.Sort.Direction != domain.SortDesc {
		t.Errorf("expected initial tab sort to be create_time/desc from prefs, got %v/%v", pane.Sort.Field, pane.Sort.Direction)
	}
}

func TestCoordinator_SetDefaultSortAndResort(t *testing.T) {
	coord, _ := setupTestCoordinator(t)
	defer coord.Close()

	pane := coord.Tabs.ActiveTab().Left()
	now := time.Now()
	pane.Entries = []domain.FileInfo{
		{Name: "b.txt", ModTime: now.Add(-10 * time.Minute)},
		{Name: "a.txt", ModTime: now},
	}
	coord.ApplyPaneFilter(pane)

	// Change default sort to mod_time desc (a.txt before b.txt)
	coord.SetDefaultSortAndResort(domain.SortByModTime, domain.SortDesc)
	if pane.Sort.Field != domain.SortByModTime || pane.Sort.Direction != domain.SortDesc {
		t.Errorf("pane sort not updated: %v/%v", pane.Sort.Field, pane.Sort.Direction)
	}
	if len(pane.DisplayEntries) != 2 || pane.DisplayEntries[0].Name != "a.txt" {
		t.Errorf("entries not resorted immediately: got %v", pane.DisplayEntries)
	}

	// Change default sort to name asc (a.txt before b.txt)
	coord.SetDefaultSortAndResort(domain.SortByName, domain.SortAsc)
	if pane.DisplayEntries[0].Name != "a.txt" {
		t.Errorf("expected a.txt first for name/asc: got %v", pane.DisplayEntries)
	}
}

func TestFilenameStemRunes(t *testing.T) {
	cases := []struct {
		name  string
		isDir bool
		want  int
	}{
		{"报告_v1.docx", false, 5},
		{"archive.tar.gz", false, 11},
		{".gitignore", false, 10},
		{"my.folder", true, 9},
		{"noext", false, 5},
	}
	for _, tc := range cases {
		if got := FilenameStemRunes(tc.name, tc.isDir); got != tc.want {
			t.Errorf("FilenameStemRunes(%q, dir=%v) = %d, want %d", tc.name, tc.isDir, got, tc.want)
		}
	}
}

func TestToggleColumnVisible(t *testing.T) {
	coord, _ := setupTestCoordinator(t)
	defer coord.Close()

	if coord.HideColSize {
		t.Fatalf("expected the size column visible by default")
	}

	coord.ToggleColumnVisible("size")
	if !coord.HideColSize {
		t.Errorf("expected the size column hidden after toggle")
	}
	if !coord.Prefs.Get().HideColSize {
		t.Errorf("expected the visibility to be persisted")
	}

	coord.ToggleColumnVisible("size")
	if coord.HideColSize {
		t.Errorf("expected the size column visible after a second toggle")
	}

	coord.HideColCreateTime = true
	coord.ColWidthName = 400
	coord.NameColManual = true
	coord.ResetAllColumns()
	if coord.HideColCreateTime || coord.HideColModTime || coord.HideColSize || coord.HideColKind {
		t.Errorf("expected every column visible after ResetAllColumns")
	}
	if coord.ColWidthName != 220 || coord.ColWidthCreateTime != 115 || coord.NameColManual {
		t.Errorf("expected default widths after ResetAllColumns, got %v/%v/manual=%v",
			coord.ColWidthName, coord.ColWidthCreateTime, coord.NameColManual)
	}
}

func TestCoordinator_ColWidthsUpdateAndReset(t *testing.T) {
	coord, _ := setupTestCoordinator(t)
	defer coord.Close()

	// Initial default
	if coord.ColWidthName != 220 {
		t.Errorf("expected ColWidthName=220, got %v", coord.ColWidthName)
	}

	// Update by +50
	coord.UpdateColWidth("name", 50)
	if coord.ColWidthName != 270 {
		t.Errorf("expected ColWidthName=270, got %v", coord.ColWidthName)
	}

	// Update below min clamp (80)
	coord.UpdateColWidth("name", -500)
	if coord.ColWidthName != 80 {
		t.Errorf("expected ColWidthName=80 clamped, got %v", coord.ColWidthName)
	}

	// Reset to default
	coord.ResetColWidth("name")
	if coord.ColWidthName != 220 {
		t.Errorf("expected ColWidthName=220 after reset, got %v", coord.ColWidthName)
	}
}
