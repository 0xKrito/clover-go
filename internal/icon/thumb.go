package icon

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/bighaidao/clover-go/internal/macos"
	"github.com/egoist/mygo"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// GenerateIconBytes extracts or generates a PNG thumbnail/icon of requested pixel size.
// 1. 常见图片走极速纯 Go 缩略图；
// 2. HEIC/TIFF/RAW/PSD/ICNS/PDF 及视频通过原生 AppKit/clover-helper 在后台内存渲染；
// 3. 兜底回退为原生系统高清晰文件图标。
// 全程零 qlmanage 调试子进程，绝不抢占焦点、绝不弹 Dock 图标。
func GenerateIconBytes(path string, px int) ([]byte, error) {
	ext := strings.ToLower(filepath.Ext(path))

	// 1. Image files: generate fast pure Go thumbnail
	if isImageExtension(ext) {
		if data, err := generateImageThumb(path, px); err == nil && len(data) > 0 {
			return data, nil
		}
	}

	// 2. Native image/document/video formats
	if isNativeThumbExtension(ext) {
		if data, err := macos.GenerateNativeThumbnail(path, px); err == nil && len(data) > 0 {
			return data, nil
		}
	}

	// 3. Fallback: native macOS system file icon
	data, err := mygo.Shell.FileIcon(path, px)
	if err != nil {
		return nil, fmt.Errorf("mygo.Shell.FileIcon %q: %w", path, err)
	}
	return data, nil
}

func isNativeThumbExtension(ext string) bool {
	switch ext {
	case ".heic", ".heif", ".tiff", ".tif", ".raw", ".cr2", ".nef", ".arw", ".dng", ".psd", ".icns", ".svg", ".pdf", ".mp4", ".mov", ".m4v":
		return true
	default:
		return false
	}
}

func isImageExtension(ext string) bool {
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp":
		return true
	default:
		return false
	}
}

func generateImageThumb(path string, maxPx int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	srcImg, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	bounds := srcImg.Bounds()
	sw, sh := bounds.Dx(), bounds.Dy()
	if sw <= 0 || sh <= 0 {
		return nil, fmt.Errorf("invalid image bounds")
	}

	dw, dh := computeAspectDimensions(sw, sh, maxPx)
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	xdraw.BiLinear.Scale(dst, dst.Rect, srcImg, bounds, draw.Over, nil)

	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func computeAspectDimensions(sw, sh, maxPx int) (int, int) {
	if sw <= maxPx && sh <= maxPx {
		return sw, sh
	}
	if sw > sh {
		dw := maxPx
		dh := int(float64(sh) * float64(maxPx) / float64(sw))
		if dh < 1 {
			dh = 1
		}
		return dw, dh
	}
	dh := maxPx
	dw := int(float64(sw) * float64(maxPx) / float64(sh))
	if dw < 1 {
		dw = 1
	}
	return dw, dh
}
