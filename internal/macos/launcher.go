package macos

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"unsafe"

	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/ebitengine/purego/objc"
	"github.com/egoist/mygo"
)

// OpenInTerminal launches the preferred available terminal at dirPath.
// If dirPath is a file, it opens the directory containing that file.
func OpenInTerminal(dirPath string) error {
	norm := domain.NormalizePath(dirPath)
	if fi, err := os.Stat(norm); err == nil && !fi.IsDir() {
		norm = filepath.Dir(norm)
	}

	terminals := []string{
		"Ghostty",
		"iTerm",
		"Warp",
		"Terminal",
	}

	for _, term := range terminals {
		if isAppInstalled(term) {
			cmd := exec.Command("/usr/bin/open", "-a", term, norm)
			if err := cmd.Run(); err == nil {
				return nil
			}
		}
	}

	cmd := exec.Command("/usr/bin/open", "-a", "Terminal", norm)
	return cmd.Run()
}

// OpenInEditor launches the preferred code editor on the given path.
// If no third-party editor is found, it falls back to macOS TextEdit (-e).
// It strictly avoids calling bare `/usr/bin/open path` which would launch whatever
// default program is associated with the file type (e.g. music/video players).
func OpenInEditor(path string) error {
	norm := domain.NormalizePath(path)
	editors := []string{
		"Cursor",
		"Visual Studio Code",
		"Zed",
		"Sublime Text",
	}

	for _, ed := range editors {
		if isAppInstalled(ed) {
			cmd := exec.Command("/usr/bin/open", "-a", ed, norm)
			if err := cmd.Run(); err == nil {
				return nil
			}
		}
	}

	// Guaranteed fallback to system text editor without blind type-association dispatch
	cmd := exec.Command("/usr/bin/open", "-e", norm)
	return cmd.Run()
}

func isAppInstalled(appName string) bool {
	clsNSWorkspace := objc.GetClass("NSWorkspace")
	if clsNSWorkspace == 0 {
		return false
	}
	ws := objc.ID(clsNSWorkspace).Send(objc.RegisterName("sharedWorkspace"))
	if ws == 0 {
		return false
	}
	if clsNSString == 0 {
		clsNSString = objc.GetClass("NSString")
		selStringWithUTF8 = objc.RegisterName("stringWithUTF8String:")
	}
	cstr := append([]byte(appName), 0)
	nsApp := objc.ID(clsNSString).Send(selStringWithUTF8, unsafe.Pointer(&cstr[0]))
	if nsApp == 0 {
		return false
	}
	pathID := ws.Send(objc.RegisterName("fullPathForApplication:"), nsApp)
	return pathID != 0
}

// ShowGetInfo reveals the Finder file info window (Cmd+I) where users can change default associations.
func ShowGetInfo(path string) error {
	norm := domain.NormalizePath(path)
	script := fmt.Sprintf(`tell application "Finder" to open information window of (POSIX file "%s" as alias)`, norm)
	cmd := exec.Command("/usr/bin/osascript", "-e", script)
	return cmd.Run()
}

// CopyPath copies the path to the system clipboard in specified format: "posix", "tilde", or "url".
func CopyPath(path, format string) error {
	norm := domain.NormalizePath(path)
	var text string

	switch format {
	case "tilde":
		home, _ := os.UserHomeDir()
		text = domain.CollapseHome(norm, home)
	case "url":
		text = fmt.Sprintf("file://%s", filepath.ToSlash(norm))
	default: // posix
		text = norm
	}

	mygo.Clipboard.WriteText(text)
	return nil
}
