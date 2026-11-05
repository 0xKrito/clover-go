package macos

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bighaidao/clover-go/internal/domain"
)

// ApplyWatermark processes the images according to opts via the precompiled
// helper (AppKit). Per-file failures are reported instead of failing silently.
func ApplyWatermark(opts domain.WatermarkOptions) error {
	if len(opts.SourceFiles) == 0 {
		return fmt.Errorf("no source images to watermark")
	}
	if strings.TrimSpace(opts.Text) == "" {
		return fmt.Errorf("watermark text cannot be empty")
	}
	if err := validateWatermarkOptions(opts); err != nil {
		return err
	}

	argv, err := buildWatermarkArgs(opts)
	if err != nil {
		return err
	}
	out, err := runHelper(argv[0], argv[1:]...)
	if err != nil {
		return err
	}
	// helper 对单个文件失败会打印 CLOVER_SKIP 行；调用方统计后上报。
	if skipped := strings.Count(out, "CLOVER_SKIP\n"); skipped > 0 {
		return fmt.Errorf("%d 个图片水印写入失败", skipped)
	}
	return nil
}

// buildWatermarkArgs renders helper argv: text tile opacity angle fontSize
// r g b extsCSV files...; output extensions are pre-computed per file
// (single source of truth) and output paths are made collision-free in the helper.
func buildWatermarkArgs(opts domain.WatermarkOptions) ([]string, error) {
	exts := make([]string, len(opts.SourceFiles))
	for i, f := range opts.SourceFiles {
		ext := domain.ResolveWatermarkOutputExt(filepath.Ext(f), opts.OutputFormat)
		if ext == "" {
			return nil, fmt.Errorf("unsupported image extension: %s", f)
		}
		exts[i] = ext
	}

	r, g, b := watermarkColorRGB(opts.TextColor)
	tile := "0"
	if opts.Layout == domain.WatermarkLayoutTile {
		tile = "1"
	}

	argv := []string{
		"watermark",
		opts.Text,
		tile,
		strconv.FormatFloat(opts.Opacity, 'f', -1, 64),
		strconv.FormatFloat(opts.Angle, 'f', -1, 64),
		strconv.Itoa(opts.FontSize),
		strconv.FormatFloat(r, 'f', -1, 64),
		strconv.FormatFloat(g, 'f', -1, 64),
		strconv.FormatFloat(b, 'f', -1, 64),
		strings.Join(exts, ","),
	}
	return append(argv, opts.SourceFiles...), nil
}

// validateWatermarkOptions rejects out-of-range numeric parameters.
func validateWatermarkOptions(opts domain.WatermarkOptions) error {
	if opts.Opacity < 0.05 || opts.Opacity > 1.0 {
		return fmt.Errorf("opacity %v out of range (0.05..1.0)", opts.Opacity)
	}
	if opts.FontSize <= 0 {
		return fmt.Errorf("font size must be positive")
	}
	return nil
}

// watermarkColorRGB maps a text color key to calibrated RGB components.
func watermarkColorRGB(color string) (float64, float64, float64) {
	switch color {
	case "red":
		return 0.85, 0.2, 0.2
	case "white":
		return 1.0, 1.0, 1.0
	case "black":
		return 0.05, 0.05, 0.05
	default: // gray
		return 0.5, 0.5, 0.5
	}
}

// IsImageFile reports whether the given file path is a supported image for watermarking.
func IsImageFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".heic", ".tiff", ".bmp", ".gif":
		return true
	default:
		return false
	}
}
