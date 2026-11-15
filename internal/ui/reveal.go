package ui

import (
	"github.com/bighaidao/clover-go/internal/app"
	"github.com/egoist/mygo"
)

// revealInFinderOrSelf implements the "在访达中显示" menu entry.
//
// Once CloverGo has taken over Finder (NSFileViewer and public.folder point at
// com.clover.app), calling mygo.Shell.ShowItemInFolder from inside CloverGo is
// a round trip through the system that lands right back here: LaunchServices
// resolves the request to this very app, the reveal arrives as an
// application:openFiles: event, and RevealPath opens the parent directory —
// which the user already had open. The menu entry then looks like it does
// nothing at all.
//
// So the behaviour depends on who is currently the system's file viewer:
//
//   - CloverGo is the viewer: reveal in place, through the same RevealPath the
//     incoming system requests use.
//   - Finder is the viewer: hand the path to the system as before, so the
//     entry keeps opening the real Finder.
func revealInFinderOrSelf(coord *app.Coordinator, path string) {
	if path == "" {
		return
	}
	if coord != nil && coord.Prefs != nil && coord.Prefs.Get().TakeoverFinder {
		coord.RevealPath(path, true)
		return
	}
	mygo.Shell.ShowItemInFolder(path)
}

// revealDirectoryOrSelf opens a directory in CloverGo when it is the viewer,
// otherwise hands it to the system. Used by the "在访达中打开" button on a
// failed directory load, where the target is a folder rather than an item.
func revealDirectoryOrSelf(coord *app.Coordinator, dir string) {
	if dir == "" {
		return
	}
	if coord != nil && coord.Prefs != nil && coord.Prefs.Get().TakeoverFinder {
		coord.RevealPath(dir, false)
		return
	}
	mygo.Shell.ShowItemInFolder(dir)
}
