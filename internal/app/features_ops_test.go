package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bighaidao/clover-go/internal/config"
	"github.com/bighaidao/clover-go/internal/domain"
)

func TestWorkspaceSaveAndRestore(t *testing.T) {
	tempDir := t.TempDir()
	wsFile := filepath.Join(tempDir, "workspaces.json")
	wsMgr, err := config.NewWorkspaceManager(wsFile)
	if err != nil {
		t.Fatal(err)
	}

	prefsFile := filepath.Join(tempDir, "prefs.json")
	prefsMgr, err := config.NewPrefsManager(prefsFile)
	if err != nil {
		t.Fatal(err)
	}

	coord := &Coordinator{
		Tabs:       NewTabManager("/tmp"),
		Workspaces: wsMgr,
		Prefs:      prefsMgr,
		runOnMain:  func(fn func()) { fn() },
	}

	// 1. Setup tabs: Tab 1 (dual pane), Tab 2 (single pane)
	coord.Tabs.Tabs[0].Path = "/tmp/left1"
	coord.Tabs.Tabs[0].IsDual = true
	coord.Tabs.Tabs[0].RightPane = NewPaneState("t1-r", "/tmp/right1")

	t2 := coord.Tabs.NewTab("/tmp/tab2")
	_ = t2

	// 2. Save workspace
	err = coord.SaveCurrentWorkspace("My Project")
	if err != nil {
		t.Fatalf("SaveCurrentWorkspace failed: %v", err)
	}

	data := wsMgr.Get()
	if len(data.Workspaces) != 1 {
		t.Fatalf("expected 1 workspace saved, got %d", len(data.Workspaces))
	}
	ws := data.Workspaces[0]
	if ws.Name != "My Project" {
		t.Errorf("expected workspace name 'My Project', got %q", ws.Name)
	}
	if len(ws.Tabs) != 2 {
		t.Fatalf("expected 2 tabs in workspace, got %d", len(ws.Tabs))
	}
	if !ws.Tabs[0].IsDual || ws.Tabs[0].RightPath != "/tmp/right1" {
		t.Errorf("tab 0 dual state not preserved: %+v", ws.Tabs[0])
	}

	// 3. Clear tabs and restore
	coord.Tabs.Tabs = nil
	coord.RestoreWorkspace(ws)

	if len(coord.Tabs.Tabs) != 2 {
		t.Fatalf("expected 2 tabs restored, got %d", len(coord.Tabs.Tabs))
	}
	if coord.Tabs.Tabs[0].Path != "/tmp/left1" {
		t.Errorf("tab 0 path = %q, want /tmp/left1", coord.Tabs.Tabs[0].Path)
	}
	if !coord.Tabs.Tabs[0].IsDual || coord.Tabs.Tabs[0].RightPane == nil || coord.Tabs.Tabs[0].RightPane.Path != "/tmp/right1" {
		t.Errorf("tab 0 dual pane not restored properly")
	}

	// 4. Delete workspace
	if err := coord.DeleteWorkspace(ws.ID); err != nil {
		t.Fatalf("DeleteWorkspace failed: %v", err)
	}
	if len(wsMgr.Get().Workspaces) != 0 {
		t.Errorf("expected 0 workspaces after deletion")
	}
}

func TestFinderImportOps(t *testing.T) {
	tempDir := t.TempDir()
	wsFile := filepath.Join(tempDir, "workspaces.json")
	wsMgr, err := config.NewWorkspaceManager(wsFile)
	if err != nil {
		t.Fatal(err)
	}

	coord := &Coordinator{
		Workspaces: wsMgr,
		runOnMain:  func(fn func()) { fn() },
	}

	coord.OpenFinderImport()
	if !coord.ShowFinderImportModal {
		t.Fatalf("expected ShowFinderImportModal to be true")
	}

	// Simulate selecting an item
	testDir := filepath.Join(tempDir, "fav1")
	_ = os.MkdirAll(testDir, 0755)

	coord.FinderFavorites = []domain.FinderFavoriteItem{
		{Name: "fav1", Path: testDir, Selected: true, Exists: true},
		{Name: "unselected", Path: "/non/existent", Selected: false, Exists: false},
	}

	coord.ExecuteFinderImport()
	if coord.ShowFinderImportModal {
		t.Errorf("expected modal to close after execution")
	}

	bks := wsMgr.Get().Bookmarks
	hasFav1 := false
	for _, b := range bks {
		if b.Name == "fav1" && b.Path == testDir {
			hasFav1 = true
			break
		}
	}
	if !hasFav1 {
		t.Errorf("expected fav1 to be imported into bookmarks")
	}
}

func TestWorkspaceRestoreLegacyPaths(t *testing.T) {
	tempDir := t.TempDir()
	wsFile := filepath.Join(tempDir, "workspaces.json")
	wsMgr, _ := config.NewWorkspaceManager(wsFile)
	prefsFile := filepath.Join(tempDir, "prefs.json")
	prefsMgr, _ := config.NewPrefsManager(prefsFile)

	coord := &Coordinator{
		Tabs:       NewTabManager("/tmp"),
		Workspaces: wsMgr,
		Prefs:      prefsMgr,
		runOnMain:  func(fn func()) { fn() },
	}

	wsLegacy := config.NamedWorkspace{
		ID:    "legacy-1",
		Name:  "Legacy WS",
		Paths: []string{"/tmp/path1", "/tmp/path2"},
	}

	coord.Tabs.Tabs = nil
	coord.RestoreWorkspace(wsLegacy)

	if len(coord.Tabs.Tabs) != 2 {
		t.Fatalf("expected 2 tabs restored from legacy paths, got %d", len(coord.Tabs.Tabs))
	}
	if coord.Tabs.Tabs[0].Path != "/tmp/path1" || coord.Tabs.Tabs[1].Path != "/tmp/path2" {
		t.Errorf("unexpected restored paths: %+v", coord.Tabs.Tabs)
	}
}

func TestTogglePreviewPersistsPrefs(t *testing.T) {
	tempDir := t.TempDir()
	prefsFile := filepath.Join(tempDir, "prefs.json")
	prefsMgr, err := config.NewPrefsManager(prefsFile)
	if err != nil {
		t.Fatal(err)
	}

	coord := &Coordinator{
		Prefs:       prefsMgr,
		ShowPreview: false,
		runOnMain:   func(fn func()) { fn() },
	}

	coord.TogglePreview()
	if !coord.ShowPreview {
		t.Errorf("expected ShowPreview to be true")
	}
	if !prefsMgr.Get().ShowPreview {
		t.Errorf("expected prefs.ShowPreview to be true")
	}

	coord.TogglePreview()
	if coord.ShowPreview {
		t.Errorf("expected ShowPreview to be false")
	}
	if prefsMgr.Get().ShowPreview {
		t.Errorf("expected prefs.ShowPreview to be false")
	}
}

func TestPreviewUpgrades_MarkdownCodeAndGIF(t *testing.T) {
	tempDir := t.TempDir()
	mdFile := filepath.Join(tempDir, "README.md")
	goFile := filepath.Join(tempDir, "main.go")
	gifFile := filepath.Join(tempDir, "test.gif")

	_ = os.WriteFile(mdFile, []byte("# Hello CloverGo\nNative Markdown preview"), 0644)
	_ = os.WriteFile(goFile, []byte("package main\n\nfunc main() {}\n"), 0644)
	// Minimal valid GIF 1x1
	minimalGIF := []byte{
		0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00,
		0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0x21,
		0xf9, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00,
		0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44,
		0x01, 0x00, 0x3b,
	}
	_ = os.WriteFile(gifFile, minimalGIF, 0644)

	coord := &Coordinator{
		runOnMain: func(fn func()) { fn() },
	}

	// 1. Test Markdown
	doneMD := make(chan struct{})
	coord.LoadPreview(domain.FileInfo{
		Path:    mdFile,
		Name:    "README.md",
		ModTime: time.Now(),
		Size:    35,
	}, func() {
		close(doneMD)
	})
	<-doneMD

	if coord.CurrentPreview == nil || !coord.CurrentPreview.IsMarkdown {
		t.Errorf("expected IsMarkdown=true, got %+v", coord.CurrentPreview)
	}

	// 2. Test Go Code
	doneGo := make(chan struct{})
	coord.LoadPreview(domain.FileInfo{
		Path:    goFile,
		Name:    "main.go",
		ModTime: time.Now(),
		Size:    30,
	}, func() {
		close(doneGo)
	})
	<-doneGo

	if coord.CurrentPreview == nil || !coord.CurrentPreview.IsCode || coord.CurrentPreview.CodeLang != "Go" {
		t.Errorf("expected IsCode=true and CodeLang='Go', got %+v", coord.CurrentPreview)
	}
	if coord.CurrentEditorState == nil {
		t.Errorf("expected CurrentEditorState to be initialized for Go code")
	}

	// 3. Test GIF
	doneGIF := make(chan struct{})
	coord.LoadPreview(domain.FileInfo{
		Path:    gifFile,
		Name:    "test.gif",
		ModTime: time.Now(),
		Size:    int64(len(minimalGIF)),
	}, func() {
		close(doneGIF)
	})
	<-doneGIF

	if coord.CurrentPreview == nil || !coord.CurrentPreview.IsGIF {
		t.Errorf("expected IsGIF=true, got %+v", coord.CurrentPreview)
	}

	// 4. Test Resource Cleanup
	coord.ClosePreviewResources()
	if coord.CurrentVideoPlayer != nil || coord.CurrentEditorState != nil {
		t.Errorf("expected VideoPlayer and EditorState to be nil after ClosePreviewResources")
	}
}
