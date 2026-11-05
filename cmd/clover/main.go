package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/bighaidao/clover-go/internal/app"
	"github.com/bighaidao/clover-go/internal/config"
	"github.com/bighaidao/clover-go/internal/fs"
	"github.com/bighaidao/clover-go/internal/icon"
	"github.com/bighaidao/clover-go/internal/macos"
	"github.com/bighaidao/clover-go/internal/ui"
	"github.com/egoist/mygo"
	mygo_ui "github.com/egoist/mygo/ui"
)

const version = "1.0.0"

func main() {
	if handleCLIFlags() {
		return
	}

	// VFS 预览临时文件惰性清理：启动时清理一次过期残留。
	go fs.CleanupQuickLookTemp()

	paths, err := config.ResolvePaths()
	if err != nil {
		log.Fatalf("初始化沙箱路径失败: %v", err)
	}

	// 1. Acquire single instance lock
	lock, err := config.AcquireInstanceLock(paths.LockFile)
	if err != nil {
		log.Printf("提示: %v，已退出当前进程", err)
		return
	}
	defer lock.Release()

	// 2. Configure MyGo application metadata & deep links
	setupAppMetadata(paths)

	// 3. Initialize managers & coordinator
	prefsMgr, _ := config.NewPrefsManager(paths.PrefsFile)
	wsMgr, _ := config.NewWorkspaceManager(paths.SpacesFile)
	iconPipeline := icon.NewPipeline(paths.ThumbsDir)
	defer iconPipeline.Close()

	initialDir := getInitialDirectory()
	coord := app.NewCoordinator(initialDir, prefsMgr, wsMgr, iconPipeline, ui.RunOnMain)
	coord.WriteSysClipboard = mygo.Clipboard.WriteFiles
	coord.ReadSysClipboard = mygo.Clipboard.ReadFiles
	appState := ui.NewAppState(coord)

	// 4. Application menu bar (Clover menu with Settings…, File/Edit/View/Window)
	setupAppMenu(coord)

	// 5. Deep link listeners
	setupDeepLinks(coord)

	// 6. Launch Metal GPU window
	sessionFile := filepath.Join(paths.RootDir, "session.json")
	registry := app.NewWindowRegistry(sessionFile, func(tab *app.TabState, pos *mygo.Point) {
		spawnDetachedWindow(tab, pos, prefsMgr, wsMgr, iconPipeline)
	})

	launchWindow(coord, appState, registry)

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

func handleCLIFlags() bool {
	if len(os.Args) > 1 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		fmt.Printf("CloverGo v%s (Pure Go + MyGo Metal GPU)\n", version)
		return true
	}
	return false
}

func setupAppMetadata(paths *config.SandboxPaths) {
	mygo.App.SetName("CloverGo")
	mygo.App.SetPath(mygo.PathUserData, paths.RootDir)
	mygo.App.SetPath(mygo.PathCache, paths.CacheDir)
	mygo.App.SetPath(mygo.PathLogs, paths.LogsDir)
	mygo.App.RegisterURLScheme("fclover")
}

func setupDeepLinks(coord *app.Coordinator) {
	mygo.App.OnOpenURL(func(rawURL string) {
		_, target, err := macos.ParseCloverURL(rawURL)
		if err == nil && target != "" {
			ui.RunOnMain(func() {
				coord.Tabs.NewTab(target)
				coord.LoadCurrentTab(nil)
			})
		}
	})
	mygo.App.OnOpenFile(func(filePath string) {
		ui.RunOnMain(func() {
			coord.Tabs.NewTab(filePath)
			coord.LoadCurrentTab(nil)
		})
	})
}

// setupAppMenu is defined in menu.go

func getInitialDirectory() string {
	if len(os.Args) > 1 && os.Args[1] != "" && !isFlag(os.Args[1]) {
		if fi, err := os.Stat(os.Args[1]); err == nil && fi.IsDir() {
			return os.Args[1]
		}
	}
	home, _ := os.UserHomeDir()
	return home
}

func isFlag(arg string) bool {
	return len(arg) > 0 && arg[0] == '-'
}

func launchWindow(coord *app.Coordinator, appState *ui.AppState, reg *app.WindowRegistry) {
	vibrancy := mygo.VibrancyNone
	if runtime.GOOS == "darwin" && coord.Prefs.Get().Vibrancy {
		vibrancy = mygo.VibrancySidebar
	}

	mygo.App.WhenReady(func() {
		ui.ApplyAppearance(coord.Prefs.Get().Appearance)

		win := mygo.NewWindow(mygo.WindowOptions{
			StateKey:             "clover-main-window",
			Title:                "CloverGo",
			Width:                1140,
			Height:               720,
			MinWidth:             840,
			MinHeight:            520,
			TitleBarStyle:        mygo.TitleBarHidden,
			TrafficLightPosition: &mygo.Point{X: 16, Y: 14},
			Vibrancy:             vibrancy,
			Content:              mygo_ui.View(appState.View),
		})

		reg.Register("main", win, coord)
		coord.SetInvalidate(func() {
			win.Invalidate()
		})
		coord.OnDetach = func(tabIdx int) {
			reg.DetachTab("main", tabIdx, nil)
		}
		// Deterministic teardown: release the FSEvents watcher when the
		// window closes instead of leaking it until process exit.
		win.OnClosed(func() {
			coord.Close()
			reg.Unregister("main")
		})
		win.Show()
		win.Focus()

		coord.LoadCurrentTab(func() {
			win.Invalidate()
		})
	})
}

func spawnDetachedWindow(tab *app.TabState, pos *mygo.Point, prefs *config.PrefsManager, ws *config.WorkspaceManager, icons *icon.Pipeline) {
	newCoord := app.NewCoordinator(tab.Path, prefs, ws, icons, ui.RunOnMain)
	newCoord.WriteSysClipboard = mygo.Clipboard.WriteFiles
	newCoord.ReadSysClipboard = mygo.Clipboard.ReadFiles
	newState := ui.NewAppState(newCoord)

	opt := mygo.WindowOptions{
		Title:                "CloverGo",
		Width:                1000,
		Height:               650,
		TitleBarStyle:        mygo.TitleBarHidden,
		TrafficLightPosition: &mygo.Point{X: 16, Y: 14},
		Vibrancy:             mygo.VibrancySidebar,
		Content:              mygo_ui.View(newState.View),
	}
	if pos != nil {
		opt.X = pos.X
		opt.Y = pos.Y
	}
	newWin := mygo.NewWindow(opt)
	newCoord.SetInvalidate(func() {
		newWin.Invalidate()
	})
	// Detached windows own their coordinator: tear down its watcher on close.
	newWin.OnClosed(func() {
		newCoord.Close()
	})
	newWin.Show()
	newCoord.LoadCurrentTab(func() {
		newWin.Invalidate()
	})
}
