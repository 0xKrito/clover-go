package macos

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui/markdown/parser"
)

var (
	qlOnce       sync.Once
	qlMu         sync.Mutex
	qlPanel      objc.ID
	qlDataSource objc.ID
	qlCurrent    string
	qlOriginPath string

	clsNSString objc.Class
	clsNSURL    objc.Class

	selStringWithUTF8       objc.SEL
	selFileURLWithPath      objc.SEL
	selSharedPreviewPanel   objc.SEL
	selSetDataSource        objc.SEL
	selReloadData           objc.SEL
	selMakeKeyAndOrderFront objc.SEL
	selOrderOut             objc.SEL
	selIsVisible            objc.SEL

	runOnMain = mygo.RunOnMain
)

// SetRunOnMainForTest allows headless tests to override main thread dispatch.
func SetRunOnMainForTest(fn func(func())) (restore func()) {
	old := runOnMain
	runOnMain = fn
	return func() { runOnMain = old }
}

func initQuickLookFramework() {
	_, _ = purego.Dlopen("/System/Library/Frameworks/Quartz.framework/Frameworks/QuickLookUI.framework/QuickLookUI", purego.RTLD_GLOBAL|purego.RTLD_NOW)
	_, _ = purego.Dlopen("/System/Library/Frameworks/QuickLookUI.framework/QuickLookUI", purego.RTLD_GLOBAL|purego.RTLD_NOW)

	clsNSString = objc.GetClass("NSString")
	clsNSURL = objc.GetClass("NSURL")

	selStringWithUTF8 = objc.RegisterName("stringWithUTF8String:")
	selFileURLWithPath = objc.RegisterName("fileURLWithPath:")
	selSharedPreviewPanel = objc.RegisterName("sharedPreviewPanel")
	selSetDataSource = objc.RegisterName("setDataSource:")
	selReloadData = objc.RegisterName("reloadData")
	selMakeKeyAndOrderFront = objc.RegisterName("makeKeyAndOrderFront:")
	selOrderOut = objc.RegisterName("orderOut:")
	selIsVisible = objc.RegisterName("isVisible")

	clsNSObject := objc.GetClass("NSObject")
	if clsNSObject == 0 {
		return
	}

	methods := []objc.MethodDef{
		{
			Cmd: objc.RegisterName("numberOfPreviewItemsInPreviewPanel:"),
			Fn: func(self objc.ID, _cmd objc.SEL, panel objc.ID) int {
				qlMu.Lock()
				p := qlCurrent
				qlMu.Unlock()
				if p == "" {
					return 0
				}
				return 1
			},
		},
		{
			Cmd: objc.RegisterName("previewPanel:previewItemAtIndex:"),
			Fn: func(self objc.ID, _cmd objc.SEL, panel objc.ID, idx int) objc.ID {
				qlMu.Lock()
				p := qlCurrent
				qlMu.Unlock()
				if p == "" || clsNSString == 0 || clsNSURL == 0 {
					return 0
				}
				absPath, err := filepath.Abs(p)
				if err != nil {
					absPath = p
				}
				cstr := append([]byte(absPath), 0)
				nsStr := objc.ID(clsNSString).Send(selStringWithUTF8, unsafe.Pointer(&cstr[0]))
				if nsStr == 0 {
					return 0
				}
				return objc.ID(clsNSURL).Send(selFileURLWithPath, nsStr)
			},
		},
	}

	dsClass, err := objc.RegisterClass("CloverQLDataSource", clsNSObject, nil, nil, methods)
	if err == nil && dsClass != 0 {
		qlDataSource = objc.ID(dsClass).Send(objc.RegisterName("new"))
	}
}

func ensureQLPanelOnMain() bool {
	qlOnce.Do(initQuickLookFramework)
	if qlPanel == 0 {
		clsQLPreviewPanel := objc.GetClass("QLPreviewPanel")
		if clsQLPreviewPanel == 0 {
			return false
		}
		qlPanel = objc.ID(clsQLPreviewPanel).Send(selSharedPreviewPanel)
		if qlPanel != 0 && qlDataSource != 0 {
			qlPanel.Send(selSetDataSource, qlDataSource)
		}
	}
	return qlPanel != 0
}

const mdHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title>
<style>
  :root { color-scheme: light dark; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
    font-size: 14px;
    line-height: 1.6;
    padding: 32px 40px;
    max-width: 860px;
    margin: 0 auto;
    color: #24292f;
    background-color: #ffffff;
  }
  @media (prefers-color-scheme: dark) {
    body { color: #c9d1d9; background-color: #0d1117; }
    h1, h2, h3, h4, h5, h6 { border-color: #21262d !important; color: #f0f6fc !important; }
    pre, code { background-color: #161b22 !important; color: #e6edf3 !important; }
    blockquote { border-color: #30363d !important; color: #8b949e !important; }
    table th, table td { border-color: #30363d !important; }
    table tr { background-color: #0d1117 !important; }
    table tr:nth-child(2n) { background-color: #161b22 !important; }
    hr { background-color: #30363d !important; }
  }
  h1, h2, h3, h4, h5, h6 { margin-top: 24px; margin-bottom: 16px; font-weight: 600; line-height: 1.25; }
  h1 { font-size: 2em; padding-bottom: 0.3em; border-bottom: 1px solid #d0d7de; }
  h2 { font-size: 1.5em; padding-bottom: 0.3em; border-bottom: 1px solid #d0d7de; }
  h3 { font-size: 1.25em; }
  p, blockquote, ul, ol, dl, table, pre { margin-top: 0; margin-bottom: 16px; }
  blockquote { padding: 0 1em; color: #57606a; border-left: 0.25em solid #d0d7de; }
  ul, ol { padding-left: 2em; }
  code {
    padding: 0.2em 0.4em;
    margin: 0;
    font-size: 85%%;
    font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
    background-color: rgba(175, 184, 193, 0.2);
    border-radius: 4px;
  }
  pre { padding: 16px; overflow: auto; font-size: 85%%; line-height: 1.45; background-color: #f6f8fa; border-radius: 6px; }
  pre code { padding: 0; margin: 0; background-color: transparent; }
  table { border-spacing: 0; border-collapse: collapse; display: block; width: max-content; max-width: 100%%; overflow: auto; }
  table th, table td { padding: 6px 13px; border: 1px solid #d0d7de; }
  table tr:nth-child(2n) { background-color: #f6f8fa; }
  hr { height: 0.25em; padding: 0; margin: 24px 0; background-color: #d0d7de; border: 0; }
  img { max-width: 100%%; box-sizing: content-box; }
</style>
</head>
<body>
%s
</body>
</html>`

func resolveQuickLookPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".md" && ext != ".markdown" && ext != ".mdown" {
		return path
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return path
	}
	if len(content) > 2*1024*1024 {
		content = content[:2*1024*1024]
	}
	doc := parser.New(parser.Default)
	doc.Set(string(content))
	bodyHTML := doc.HTML()

	title := filepath.Base(path)
	html := fmt.Sprintf(mdHTMLTemplate, title, bodyHTML)

	tmpFile := filepath.Join(os.TempDir(), "clover_md_quicklook.html")
	if err := os.WriteFile(tmpFile, []byte(html), 0644); err != nil {
		return path
	}
	return tmpFile
}

// QuickLook shows the macOS in-process QLPreviewPanel for the target path.
func QuickLook(path string) {
	if path == "" {
		return
	}
	runOnMain(func() {
		if !ensureQLPanelOnMain() {
			return
		}
		qlMu.Lock()
		qlOriginPath = path
		qlCurrent = resolveQuickLookPath(path)
		qlMu.Unlock()

		qlPanel.Send(selReloadData)
		qlPanel.Send(selMakeKeyAndOrderFront, objc.ID(0))
	})
}

// UpdateQuickLook updates the previewed file in-place if the Quick Look panel is currently active.
// If the panel is closed, this is a no-op.
func UpdateQuickLook(path string) {
	if path == "" || qlPanel == 0 {
		return
	}
	runOnMain(func() {
		if qlPanel == 0 || !objc.Send[bool](qlPanel, selIsVisible) {
			return
		}
		qlMu.Lock()
		if qlOriginPath == path {
			qlMu.Unlock()
			return
		}
		qlOriginPath = path
		qlCurrent = resolveQuickLookPath(path)
		qlMu.Unlock()

		qlPanel.Send(selReloadData)
	})
}

// ToggleQuickLookHook allows headless tests or hooks to intercept QuickLook toggling.
var ToggleQuickLookHook func(path string) bool

// ToggleQuickLook toggles the Quick Look preview panel.
// If the panel is open on the same file, it closes.
// If open on a different file, it seamlessly updates in-place.
// If closed, it opens the panel.
func ToggleQuickLook(path string) {
	if path == "" {
		return
	}
	if ToggleQuickLookHook != nil && ToggleQuickLookHook(path) {
		return
	}
	runOnMain(func() {
		if !ensureQLPanelOnMain() {
			return
		}
		if objc.Send[bool](qlPanel, selIsVisible) {
			qlMu.Lock()
			origin := qlOriginPath
			qlMu.Unlock()
			if origin == path {
				qlPanel.Send(selOrderOut, objc.ID(0))
				qlMu.Lock()
				qlCurrent = ""
				qlOriginPath = ""
				qlMu.Unlock()
				return
			}
		}
		qlMu.Lock()
		qlOriginPath = path
		qlCurrent = resolveQuickLookPath(path)
		qlMu.Unlock()

		qlPanel.Send(selReloadData)
		qlPanel.Send(selMakeKeyAndOrderFront, objc.ID(0))
	})
}

// CloseQuickLookHook allows headless tests to intercept CloseQuickLook.
var CloseQuickLookHook func() bool

// CloseQuickLook shuts down any active Quick Look preview panel.
func CloseQuickLook() {
	if CloseQuickLookHook != nil && CloseQuickLookHook() {
		return
	}
	runOnMain(func() {
		if qlPanel != 0 && objc.Send[bool](qlPanel, selIsVisible) {
			qlPanel.Send(selOrderOut, objc.ID(0))
		}
		qlMu.Lock()
		qlCurrent = ""
		qlOriginPath = ""
		qlMu.Unlock()
	})
}

// IsQuickLookActiveHook allows headless tests to stub IsQuickLookActive.
var IsQuickLookActiveHook func() bool

// IsQuickLookActive reports whether the native Quick Look panel is currently visible.
func IsQuickLookActive() bool {
	if IsQuickLookActiveHook != nil {
		return IsQuickLookActiveHook()
	}
	if qlPanel == 0 {
		return false
	}
	var active bool
	runOnMain(func() {
		if qlPanel != 0 {
			active = objc.Send[bool](qlPanel, selIsVisible)
		}
	})
	return active
}
