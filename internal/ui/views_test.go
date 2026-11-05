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

func TestSpaceKeyQuickLookAndRename(t *testing.T) {
	var toggledPath string
	macos.ToggleQuickLookHook = func(p string) bool {
		toggledPath = p
		return true
	}
	defer func() { macos.ToggleQuickLookHook = nil }()

	coord := createTestCoordinator(t)
	activeTab := coord.Tabs.ActiveTab()
	pane := activeTab.ActivePane()
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

	// Now test Grid View
	coord.RenamingIdx = -1
	toggledPath = ""
	coord.ViewMode = config.ViewGrid
	tt.Frame()

	// Press Space key in Grid View
	tt.Key(0, ui.KeySpace)
	tt.Frame()

	if toggledPath != "/tmp/test.txt" {
		t.Errorf("expected QuickLook toggled in Grid View for /tmp/test.txt, got %q", toggledPath)
	}

	// Press Enter to trigger Rename in Grid View
	tt.Key(0, ui.KeyEnter)
	tt.Frame()

	if coord.RenamingIdx != 0 {
		t.Errorf("expected RenamingIdx=0 in Grid View after Enter, got %d", coord.RenamingIdx)
	}

	// Test Escape closes QuickLook
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
}
