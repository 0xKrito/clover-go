package ui

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/bighaidao/clover-go/internal/app"
	"github.com/bighaidao/clover-go/internal/config"
	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/bighaidao/clover-go/internal/macos"
	"github.com/egoist/mygo/ui"
)

var testMainMu sync.Mutex

func createTestCoordinator(t *testing.T) *app.Coordinator {
	tempDir := t.TempDir()
	prefsMgr, _ := config.NewPrefsManager(filepath.Join(tempDir, "prefs.json"))
	wsMgr, _ := config.NewWorkspaceManager(filepath.Join(tempDir, "spaces.json"))
	return app.NewCoordinator(tempDir, prefsMgr, wsMgr, nil, func(f func()) {
		testMainMu.Lock()
		defer testMainMu.Unlock()
		f()
	})
}

func TestSettingsModal_Render(t *testing.T) {
	coord := createTestCoordinator(t)
	coord.ShowSettingsModal = true
	coord.SettingsActiveTab = "general"

	tt := ui.NewTester(func(c *ui.Context) {
		SettingsModal(c, coord)
	}, 800, 600)
	tt.Frame()

	// Switch tab
	coord.SettingsActiveTab = "dnd"
	tt.Frame()

	coord.SettingsActiveTab = "shortcuts"
	tt.Frame()

	coord.SettingsActiveTab = "about"
	tt.Frame()
}

func TestConflictModal_Render(t *testing.T) {
	coord := createTestCoordinator(t)
	coord.ShowConflictModal = true
	coord.PendingConflict = &app.PendingConflict{
		SrcPath:    "/src/test.txt",
		DstDir:     "/dst",
		DstPath:    "/dst/test.txt",
		Action:     app.ActionMove,
		SrcSize:    1024,
		SrcModTime: time.Now(),
		DstSize:    2048,
		DstModTime: time.Now(),
	}

	tt := ui.NewTester(func(c *ui.Context) {
		ConflictModal(c, coord)
	}, 800, 600)
	tt.Frame()
}

func TestAboutModal_Render(t *testing.T) {
	coord := createTestCoordinator(t)
	coord.ShowAboutModal = true

	tt := ui.NewTester(func(c *ui.Context) {
		AboutModal(c, coord)
	}, 800, 600)
	tt.Frame()
}

func TestAppState_ViewRender(t *testing.T) {
	coord := createTestCoordinator(t)
	appState := NewAppState(coord)

	tt := ui.NewTester(func(c *ui.Context) {
		testMainMu.Lock()
		defer testMainMu.Unlock()
		appState.View(c)
	}, 900, 600)
	tt.Frame()

	// Test dual pane view
	coord.ToggleDualPane()
	tt.Frame()

	// Test toast rendering
	coord.ToastMessage = "测试提示"
	tt.Frame()
}

func TestSpaceKeyQuickLookAndRename_List(t *testing.T) {
	var toggledPath string
	macos.ToggleQuickLookHook = func(p string) bool {
		toggledPath = p
		return true
	}
	defer func() { macos.ToggleQuickLookHook = nil }()

	coord := createTestCoordinator(t)
	pane := coord.Tabs.ActiveTab().ActivePane()
	pane.DisplayEntries = []domain.FileInfo{
		{Name: "test.txt", Path: "/tmp/test.txt", IsDir: false},
	}
	pane.SelectedIdx = 0

	appState := NewAppState(coord)
	tt := ui.NewTester(func(c *ui.Context) {
		appState.View(c)
	}, 900, 600)
	tt.Frame()

	// Press Space key to trigger QuickLook
	tt.Key(0, ui.KeySpace)
	tt.Frame()
	if toggledPath != "/tmp/test.txt" {
		t.Errorf("expected QuickLook toggled for /tmp/test.txt, got %q", toggledPath)
	}

	// Press Enter to trigger Rename
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if coord.RenamingIdx != 0 {
		t.Errorf("expected RenamingIdx=0 after Enter, got %d", coord.RenamingIdx)
	}
}

func TestSpaceKeyQuickLookAndRename_Grid(t *testing.T) {
	var toggledPath string
	macos.ToggleQuickLookHook = func(p string) bool {
		toggledPath = p
		return true
	}
	defer func() { macos.ToggleQuickLookHook = nil }()

	coord := createTestCoordinator(t)
	coord.ViewMode = config.ViewGrid
	pane := coord.Tabs.ActiveTab().ActivePane()
	pane.DisplayEntries = []domain.FileInfo{
		{Name: "test.txt", Path: "/tmp/test.txt", IsDir: false},
	}
	pane.SelectedIdx = 0

	appState := NewAppState(coord)
	tt := ui.NewTester(func(c *ui.Context) {
		appState.View(c)
	}, 900, 600)
	tt.Frame()

	tt.Key(0, ui.KeySpace)
	tt.Frame()
	if toggledPath != "/tmp/test.txt" {
		t.Errorf("expected QuickLook in Grid View for /tmp/test.txt, got %q", toggledPath)
	}

	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if coord.RenamingIdx != 0 {
		t.Errorf("expected RenamingIdx=0 in Grid View after Enter, got %d", coord.RenamingIdx)
	}

	var closedQuickLook bool
	macos.CloseQuickLookHook = func() bool {
		closedQuickLook = true
		return true
	}
	defer func() { macos.CloseQuickLookHook = nil }()
	macos.IsQuickLookActiveHook = func() bool { return true }
	defer func() { macos.IsQuickLookActiveHook = nil }()

	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if !closedQuickLook {
		t.Errorf("expected Escape to close QuickLook")
	}
}

func TestFileListView_AdaptiveWindowAndColumnResize(t *testing.T) {
	coord := createTestCoordinator(t)
	pane := coord.Tabs.ActiveTab().Left()
	now := time.Now()
	pane.Entries = []domain.FileInfo{
		{Name: "short.txt", Path: "/tmp/short.txt", ModTime: now.Add(-time.Hour), Size: 100, Kind: "纯文本"},
		{Name: "a-very-long-file-name-that-needs-column-expansion.pdf", Path: "/tmp/long.pdf", ModTime: now, Size: 2048, Kind: "PDF 文档"},
	}
	pane.DisplayEntries = pane.Entries

	// Test responsive window sizing (adaptive width scaling at 600, 1000, 1400)
	for _, w := range []int{600, 1000, 1400} {
		ttW := ui.NewTester(func(c *ui.Context) {
			FileListViewForPane(c, coord, pane)
		}, w, 600)
		ttW.Frame()
	}

	// Adjust column width via coordinator
	coord.UpdateColWidth("create_time", 20)
	if coord.ColWidthCreateTime != 135 {
		t.Errorf("expected ColWidthCreateTime=135, got %v", coord.ColWidthCreateTime)
	}

	// Reset column width
	coord.ResetColWidth("create_time")
	if coord.ColWidthCreateTime != 115 {
		t.Errorf("expected ColWidthCreateTime=115, got %v", coord.ColWidthCreateTime)
	}

	// Live drag widens without committing; cancel rolls back to the start.
	coord.DragColWidth("create_time", 20)
	if coord.ColWidthCreateTime != 135 {
		t.Errorf("expected ColWidthCreateTime=135 during drag, got %v", coord.ColWidthCreateTime)
	}
	coord.CancelColDrag("create_time")
	if coord.ColWidthCreateTime != 115 {
		t.Errorf("expected ColWidthCreateTime=115 after cancel, got %v", coord.ColWidthCreateTime)
	}

	// A released drag persists the new width.
	coord.DragColWidth("create_time", 10)
	coord.EndColDrag()
	if coord.ColWidthCreateTime != 125 {
		t.Errorf("expected ColWidthCreateTime=125 after commit, got %v", coord.ColWidthCreateTime)
	}
}

func TestFileListView_RightClickContextMenu(t *testing.T) {
	coord := createTestCoordinator(t)
	pane := coord.Tabs.ActiveTab().Left()
	pane.Entries = []domain.FileInfo{
		{Name: "short.txt", Path: "/tmp/short.txt", Kind: "纯文本"},
	}
	pane.DisplayEntries = pane.Entries
	pane.SelectedIdx = 0

	tt := ui.NewTester(func(c *ui.Context) {
		FileListViewForPane(c, coord, pane)
	}, 900, 600)
	tt.Frame()

	rect, found := tt.Find("short.txt")
	if !found {
		t.Fatal("expected short.txt row to be found")
	}
	tt.RightClickAt(rect.X+20, rect.Y+rect.H/2)
	tt.Frame()

	menu := tt.Menu()
	if len(menu) == 0 {
		t.Fatal("expected a context menu after right clicking a row")
	}
	for _, want := range []string{"快速查看", "移到废纸篓"} {
		found := false
		for _, label := range menu {
			if label == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected menu item %q, got %v", want, menu)
		}
	}
}

func TestBreadcrumbBar_LongPathAndFoldedMenu(t *testing.T) {
	coord := createTestCoordinator(t)
	var navigatedPath string
	onNav := func(path string) { navigatedPath = path }

	crumbs := []domain.PathCrumb{
		{Name: "~", FullPath: "/Users/test"},
		{Name: "Projects", FullPath: "/Users/test/Projects"},
		{Name: "Go", FullPath: "/Users/test/Projects/Go"},
		{Name: "Clover", FullPath: "/Users/test/Projects/Go/Clover"},
		{Name: "Internal", FullPath: "/Users/test/Projects/Go/Clover/Internal"},
		{Name: "UI", FullPath: "/Users/test/Projects/Go/Clover/Internal/UI"},
		{Name: "Views", FullPath: "/Users/test/Projects/Go/Clover/Internal/UI/Views"},
		{Name: "ExtraLongFinalDirectoryNameForTestingOverflow", FullPath: "/Users/test/Projects/Go/Clover/Internal/UI/Views/ExtraLongFinalDirectoryNameForTestingOverflow"},
	}

	tt := ui.NewTester(func(c *ui.Context) {
		BreadcrumbBar(c, crumbs, coord, onNav)
	}, 600, 100)
	tt.Frame()

	rect, found := tt.Find("…")
	if !found {
		t.Fatal("expected folded breadcrumbs ellipsis '…' button")
	}
	tt.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
	tt.Frame()

	menu := tt.Menu()
	if len(menu) == 0 {
		t.Fatal("expected popover menu for folded breadcrumbs")
	}

	if err := tt.ChooseMenuItem("Clover"); err != nil {
		t.Fatalf("failed to choose Clover in breadcrumbs popover menu: %v", err)
	}
	tt.Frame()

	if navigatedPath != "/Users/test/Projects/Go/Clover" {
		t.Errorf("expected navigatedPath '/Users/test/Projects/Go/Clover', got %q", navigatedPath)
	}
}

func TestLongNames_ViewsRenderNoPanic(t *testing.T) {
	coord := createTestCoordinator(t)
	longName := "超长文件名_用来验证长名称场景下Sidebar_Toolbar_BreadcrumbBar_FileListView_IconGridView稳定无panic_1234567890.pdf"
	pane := coord.Tabs.ActiveTab().Left()
	pane.Path = "/Users/test/Very/Deep/Path/" + longName
	pane.Entries = []domain.FileInfo{
		{Name: longName, Path: pane.Path, Kind: "PDF 文档", Size: 1048576, ModTime: time.Now()},
		{Name: "normal.txt", Path: "/Users/test/normal.txt", Kind: "纯文本", Size: 100, ModTime: time.Now()},
	}
	pane.DisplayEntries = pane.Entries
	_ = coord.AddBookmark(pane.Path, "超长书签名称用来测试Toolbar防挤爆截断")
	_ = coord.SaveCurrentWorkspace("超长工作区名称防止挤飞按钮")

	tt := ui.NewTester(func(c *ui.Context) {
		SidebarView(c, coord, func(string) {})
		ToolbarView(c, coord, func() {})
		FileListViewForPane(c, coord, pane)
		IconGridViewForPane(c, coord, pane)
	}, 800, 600)
	tt.Frame()

	tt.SetSize(400, 500)
	tt.Frame()
}

