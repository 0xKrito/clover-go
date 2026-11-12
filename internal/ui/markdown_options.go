package ui

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bighaidao/clover-go/internal/app"
	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/bighaidao/clover-go/internal/macos"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
)

// markdownAnchor carries in-page "#heading" navigation across frames, since
// the options are rebuilt each frame and the view keeps no anchor of its own.
var (
	markdownAnchorMu sync.Mutex
	markdownAnchor   struct {
		path    string
		slug    string
		version uint64
	}
)

// buildMarkdownOptions configures the markdown view: GitHub alert colors,
// relative-link navigation/preview anchored at the document's directory, and
// in-page heading anchors.
func buildMarkdownOptions(c *ui.Context, coord *app.Coordinator, pane *app.PaneState, prev *app.PreviewSnapshot) *markdown.Options {
	t := c.Theme()
	dir := filepath.Dir(prev.Path)
	opts := &markdown.Options{
		Theme: &markdown.Theme{Alerts: [5]ui.Color{
			TintBlue.C(t), TintGreen.C(t), TintPurple.C(t), TintOrange.C(t), TintRed.C(t),
		}},
		OnLink: func(dest string) { openMarkdownLink(coord, pane, prev.Path, dest) },
		Preview: func(c *ui.Context, dest string) func(tip ui.Element) {
			return markdownLinkTip(c, dir, dest)
		},
	}
	if slug, version := markdownAnchorFor(prev.Path); slug != "" {
		opts.Anchor = slug
		opts.AnchorVersion = version
	}
	return opts
}

// openMarkdownLink routes a clicked link: in-page anchors scroll, web links
// open in the browser, and existing local paths navigate the active pane.
func openMarkdownLink(coord *app.Coordinator, pane *app.PaneState, docPath, dest string) {
	if strings.HasPrefix(dest, "#") {
		setMarkdownAnchor(docPath, strings.TrimPrefix(dest, "#"))
		coord.Invalidate()
		return
	}
	if isExternalLink(dest) {
		_ = mygo.Shell.OpenExternal(dest)
		return
	}
	target, ok := resolveMarkdownLink(filepath.Dir(docPath), dest)
	if !ok {
		return
	}
	fi, err := os.Stat(target)
	if err != nil {
		return
	}
	if fi.IsDir() {
		coord.Tabs.NavigatePane(coord.Tabs.ActiveIndex, pane, target)
		coord.LoadCurrentTab(nil)
		return
	}
	coord.Tabs.NavigatePane(coord.Tabs.ActiveIndex, pane, filepath.Dir(target))
	coord.LoadCurrentTab(nil)
	coord.LoadPreview(domain.FileInfo{Name: filepath.Base(target), Path: target}, nil)
}

// markdownLinkTip shows the target file's name, kind and size while the
// pointer rests on a relative link, or nil when the target is not on disk.
func markdownLinkTip(c *ui.Context, dir, dest string) func(tip ui.Element) {
	if isExternalLink(dest) || strings.HasPrefix(dest, "#") {
		return nil
	}
	target, ok := resolveMarkdownLink(dir, dest)
	if !ok {
		return nil
	}
	fi, err := os.Stat(target)
	if err != nil {
		return nil
	}
	kind := "文件"
	if fi.IsDir() {
		kind = "文件夹"
	}
	name := filepath.Base(target)
	summary := fmt.Sprintf("%s · %s", kind, domain.FormatSize(fi.Size()))
	return func(tip ui.Element) {
		tip.Children(func() {
			ui.Text(c, name).FontSize(11).FontWeight(600).SingleLine()
			ui.Text(c, summary).FontSize(10).SingleLine()
		})
	}
}

// resolveMarkdownLink resolves dest against the document's directory, dropping
// any query or fragment, and reports false for web and in-page links.
func resolveMarkdownLink(dir, dest string) (string, bool) {
	if dest == "" || isExternalLink(dest) || strings.HasPrefix(dest, "#") {
		return "", false
	}
	path := dest
	if i := strings.IndexAny(path, "?#"); i >= 0 {
		path = path[:i]
	}
	if path == "" {
		return "", false
	}
	if unescaped, err := url.PathUnescape(path); err == nil {
		path = unescaped
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, path)
	}
	return filepath.Clean(path), true
}

func isExternalLink(dest string) bool {
	u, err := url.Parse(dest)
	return err == nil && u.Scheme != "" && u.Scheme != "file"
}

func markdownAnchorFor(path string) (string, uint64) {
	markdownAnchorMu.Lock()
	defer markdownAnchorMu.Unlock()
	if markdownAnchor.path != path {
		return "", 0
	}
	return markdownAnchor.slug, markdownAnchor.version
}

func setMarkdownAnchor(path, slug string) {
	markdownAnchorMu.Lock()
	defer markdownAnchorMu.Unlock()
	markdownAnchor.version++
	markdownAnchor.path = path
	markdownAnchor.slug = slug
}

func renderMarkdownPane(c *ui.Context, coord *app.Coordinator, pane *app.PaneState, prev *app.PreviewSnapshot, pWidth float32) {
	pal := Pal(c.Theme())
	_ = pWidth

	renderMarkdownHeader(c, coord, prev)
	ui.Box(c).Height(1).Background(pal.Line)

	renderMarkdownBody(c, coord, pane, prev)
	ui.Box(c).Height(1).Background(pal.Line)

	renderColorTagsPicker(c, coord, pane, prev)
	renderQuickActions(c, coord, prev)
}

func renderMarkdownHeader(c *ui.Context, coord *app.Coordinator, prev *app.PreviewSnapshot) {
	t := c.Theme()
	pal := Pal(t)

	ui.Column(c).Gap(6).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
			ui.Icon(c, IcFile).Size(16, 16).TextColor(t.Accent).Shrink(0)
			ui.Text(c, prev.Name).FontSize(13).FontWeight(600).TextColor(t.Text).SingleLine().Grow(1).Shrink(1).MinWidth(0).Tooltip(prev.Path)
		})
		ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(6).Children(func() {
			ui.Text(c, fmt.Sprintf("Markdown 文档 · %s", domain.FormatSize(prev.Size))).FontSize(11).TextColor(t.TextMuted).SingleLine().Grow(1).Shrink(1).MinWidth(0)
			btnQL := ui.ButtonBase(c).Padding(2, 6).Radius(4).Background(pal.Inset).Shrink(0)
			btnQL.Children(func() {
				ui.Row(c).Gap(3).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, IcEye).Size(11, 11).TextColor(t.TextMuted).Shrink(0)
					ui.Text(c, "空格快速查看").FontSize(10).TextColor(t.TextMuted).Shrink(0)
				})
			})
			if btnQL.Clicked() {
				macos.ToggleQuickLook(prev.Path)
			}
		})

		ui.Row(c).Gap(6).Padding(4, 0).Children(func() {
			btnPreview := pillSegmentButton(c, IcEye, "渲染预览", coord.MarkdownViewMode == 0)
			if btnPreview.Clicked() {
				coord.MarkdownViewMode = 0
				coord.Invalidate()
			}
			btnSource := pillSegmentButton(c, IcEdit, "源码查看", coord.MarkdownViewMode == 1)
			if btnSource.Clicked() {
				coord.MarkdownViewMode = 1
				coord.Invalidate()
			}
		})
	})
}

func renderMarkdownBody(c *ui.Context, coord *app.Coordinator, pane *app.PaneState, prev *app.PreviewSnapshot) {
	t := c.Theme()
	pal := Pal(t)

	if coord.MarkdownViewMode == 0 {
		ui.Scroll(c).Grow(1).FillWidth().Padding(8).Radius(6).Background(pal.Inset).Children(func() {
			markdown.View(c, prev.Snippet, buildMarkdownOptions(c, coord, pane, prev))
		})
		return
	}

	if coord.CurrentEditorState != nil && coord.CurrentEditorPath == prev.Path {
		ui.Box(c).Grow(1).FillWidth().Padding(4).Radius(6).Background(pal.Inset).Children(func() {
			editor.View(c, coord.CurrentEditorState).Fill()
		})
	} else {
		lines := prev.Snippet
		ui.Scroll(c).Grow(1).FillWidth().Padding(6).Radius(6).Background(pal.Inset).Children(func() {
			ui.Text(c, lines).FontSize(10).TextColor(t.Text).Lines(1, 100).PreserveSpaces()
		})
	}
}

func pillSegmentButton(c *ui.Context, icon *ui.SVG, label string, active bool) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Label(label).Padding(3, 8).Gap(4).Radius(5).AlignItems(ui.Center)
	bg := t.Text.Alpha(0.06)
	col := t.TextMuted
	if active {
		bg = t.Accent
		col = ui.Hex("#ffffff")
	} else if b.Hovered() {
		bg = t.Text.Alpha(0.12)
		col = t.Text
	}
	b.Background(bg)
	return b.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(12, 12).TextColor(col).Shrink(0)
		}
		ui.Text(c, label).FontSize(11).TextColor(col).FontWeight(500)
	})
}
