package domain

import "strings"

// WatermarkLayout defines the layout strategy of the watermark text.
type WatermarkLayout string

const (
	WatermarkLayoutCenter WatermarkLayout = "center"
	WatermarkLayoutTile   WatermarkLayout = "tile"
)

// WatermarkOptions defines parameters for applying watermarks to images.
type WatermarkOptions struct {
	Text         string
	Layout       WatermarkLayout
	Opacity      float64 // 0.1 to 1.0, e.g. 0.35
	Angle        float64 // degrees, e.g. -30
	FontSize     int     // pt, e.g. 24
	TextColor    string  // "gray", "red", "white", "black"
	OutputFormat string  // "keep", "png", "jpeg"
	SourceFiles  []string
	OutputDir    string
}

// DefaultWatermarkOptions returns standard default settings for watermarking.
// Note: re-encoding via AppKit never carries the source EXIF/GPS metadata, so
// stripping is an inherent property of the output rather than a toggle.
func DefaultWatermarkOptions() WatermarkOptions {
	return WatermarkOptions{
		Text:         "CONFIDENTIAL · 内部资料",
		Layout:       WatermarkLayoutTile,
		Opacity:      0.35,
		Angle:        -30,
		FontSize:     24,
		TextColor:    "gray",
		OutputFormat: "keep",
	}
}

// ResolveWatermarkOutputExt decides the output file extension for a source
// image. "keep" is only honored for formats AppKit can actually encode
// (png/jpeg); every other source (heic/webp/gif/bmp/tiff) falls back to png,
// so the output is always a decodable, correctly-labeled image.
func ResolveWatermarkOutputExt(origExt, outFormat string) string {
	ext := strings.ToLower(strings.TrimPrefix(origExt, "."))
	switch strings.ToLower(outFormat) {
	case "png":
		return "png"
	case "jpeg", "jpg":
		return "jpg"
	default: // "keep"
		switch ext {
		case "png":
			return "png"
		case "jpg", "jpeg":
			return "jpg"
		default:
			return "png"
		}
	}
}
