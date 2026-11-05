package app

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/bighaidao/clover-go/internal/fs"
	"github.com/bighaidao/clover-go/internal/macos"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	_ "golang.org/x/image/webp"
)

// PreviewSnapshot captures cached metadata and snippet for zero-I/O UI rendering.
type PreviewSnapshot struct {
	Path            string
	Name            string
	Kind            string
	Size            int64
	ModTime         string
	ModTimeUnix     int64
	Mode            string
	Snippet         string
	IsBinary        bool
	ColorTag        int
	ImageDimensions string
	PDFPages        int
	TotalLines      int
	IsMarkdown      bool
	IsCode          bool
	CodeLang        string
	IsGIF           bool
}

// LoadPreview asynchronously reads item metadata and initial text snippet.
func (c *Coordinator) LoadPreview(item domain.FileInfo, onDone func()) {
	// If native Quick Look panel is active, update it in-place seamlessly
	if macos.IsQuickLookActive() {
		targetPath := item.Path
		if fs.IsArchiveVirtualPath(targetPath) {
			if !item.IsDir {
				if tmp, err := fs.ExtractArchiveFileToTemp(targetPath); err == nil {
					targetPath = tmp
				}
			}
		}
		macos.UpdateQuickLook(targetPath)
	}

	// Release previous GIF video player if moving to a different item
	if c.CurrentVideoPlayer != nil && c.CurrentVideoPath != item.Path {
		c.CurrentVideoPlayer.Close()
		c.CurrentVideoPlayer = nil
		c.CurrentVideoPath = ""
	}

	ext := strings.ToLower(filepath.Ext(item.Path))
	isMarkdown := ext == ".md" || ext == ".markdown" || ext == ".mdown"
	isCode := false
	codeLang := ""
	if !isMarkdown {
		if lang := editor.LanguageFor(item.Path); lang != nil {
			isCode = true
			codeLang = lang.Name
		}
	}

	// Tag color resolution is pure in-memory: FileInfo.ColorTags was already
	// parsed from xattr inside ReadDirPage's background goroutine, so the UI
	// thread must never call GetFileTags (unix.Getxattr) here.
	c.CurrentPreview = &PreviewSnapshot{
		Path:        item.Path,
		Name:        item.Name,
		Kind:        item.Kind,
		Size:        item.Size,
		ModTime:     domain.FormatModTime(item.ModTime, time.Now()),
		ModTimeUnix: item.ModTime.Unix(),
		Mode:        item.Mode.String(),
		ColorTag:    macos.ColorIndexFromTags(item.ColorTags),
		IsMarkdown:  isMarkdown,
		IsCode:      isCode,
		CodeLang:    codeLang,
		IsGIF:       ext == ".gif",
	}

	if item.IsDir {
		if onDone != nil {
			onDone()
		}
		return
	}

	path := item.Path

	go func() {
		var imgDim string
		var pdfPages int
		var gifBytes []byte

		// Image dimension resolution: v0.7.3 ui.DecodeImage handles SVG and bitmaps
		if ext == ".svg" {
			if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
				if img, err := ui.DecodeImage(data); err == nil && img != nil {
					if w, h := ui.ImageSize(img); w > 0 && h > 0 {
						imgDim = fmt.Sprintf("%.0f × %.0f", w, h)
					}
				}
			}
		} else if isImageExt(ext) {
			if f, err := os.Open(path); err == nil {
				if cfg, _, err := image.DecodeConfig(f); err == nil && cfg.Width > 0 && cfg.Height > 0 {
					imgDim = fmt.Sprintf("%d × %d", cfg.Width, cfg.Height)
				}
				_ = f.Close()
			}
			// For animated GIFs, read up to 8MB in background for ui.NewGIFDecoder
			if ext == ".gif" {
				if fi, err := os.Stat(path); err == nil && fi.Size() <= 8*1024*1024 {
					gifBytes, _ = os.ReadFile(path)
				}
			}
		} else if ext == ".pdf" {
			if count, err := macos.GetPDFPageCount(path); err == nil && count > 0 {
				pdfPages = count
			}
		}

		f, err := os.Open(path)
		if err != nil {
			c.runOnMain(func() {
				if c.CurrentPreview != nil && c.CurrentPreview.Path == path {
					c.CurrentPreview.ImageDimensions = imgDim
					c.CurrentPreview.PDFPages = pdfPages
					if onDone != nil {
						onDone()
					}
				}
			})
			return
		}
		defer f.Close()

		buf := make([]byte, 32768)
		n, _ := f.Read(buf)
		chunk := buf[:n]

		isBinary := false
		for _, b := range chunk {
			if b == 0 {
				isBinary = true
				break
			}
		}

		var snippet string
		var totalLines int
		if !isBinary && n > 0 {
			snippet = string(chunk)
			totalLines = strings.Count(snippet, "\n") + 1
		}

		c.runOnMain(func() {
			if c.CurrentPreview != nil && c.CurrentPreview.Path == path {
				c.CurrentPreview.Snippet = snippet
				c.CurrentPreview.IsBinary = isBinary
				c.CurrentPreview.ImageDimensions = imgDim
				c.CurrentPreview.PDFPages = pdfPages
				c.CurrentPreview.TotalLines = totalLines

				// Initialize animated GIF video player if valid GIF data
				if len(gifBytes) > 0 {
					if c.CurrentVideoPlayer != nil {
						c.CurrentVideoPlayer.Close()
						c.CurrentVideoPlayer = nil
					}
					if d, err := ui.NewGIFDecoder(gifBytes); err == nil {
						if p, err := ui.NewVideoPlayer(d); err == nil {
							p.SetLoop(true)
							c.CurrentVideoPlayer = p
							c.CurrentVideoPath = path
						}
					}
				}

				// Initialize Zed-style code editor state if code or markdown file
				if (isCode || isMarkdown) && snippet != "" {
					buf := editor.NewBuffer(snippet)
					if lang := editor.LanguageFor(path); lang != nil {
						buf.SetLanguage(lang)
					}
					c.CurrentEditorState = &editor.State{Buffer: buf, ReadOnly: true}
					c.CurrentEditorPath = path
				} else {
					c.CurrentEditorState = nil
					c.CurrentEditorPath = ""
				}

				if onDone != nil {
					onDone()
				}
			}
		})
	}()
}

func isImageExt(ext string) bool {
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg":
		return true
	default:
		return false
	}
}
