package domain

import "testing"

// TestResolveWatermarkOutputExt locks the output format decision: "keep" is
// only honored for formats AppKit can encode; everything else falls back to
// png so HEIC/WEBP/GIF inputs never produce mislabeled or broken files.
func TestResolveWatermarkOutputExt(t *testing.T) {
	cases := []struct {
		origExt, outFormat, want string
	}{
		{"png", "keep", "png"},
		{"jpg", "keep", "jpg"},
		{"jpeg", "keep", "jpg"},
		{"heic", "keep", "png"}, // iPhone 实拍默认路径，原为损坏输出
		{"webp", "keep", "png"}, // AppKit 无法编码，原为静默无输出
		{"gif", "keep", "png"},  // 动图丢帧
		{"tiff", "keep", "png"},
		{"bmp", "keep", "png"},
		{"png", "png", "png"},
		{"heic", "png", "png"},
		{"heic", "jpeg", "jpg"},
		{"png", "JPG", "jpg"}, // 大小写不敏感
		{"", "keep", "png"},   // 无扩展名兜底
	}
	for _, c := range cases {
		if got := ResolveWatermarkOutputExt(c.origExt, c.outFormat); got != c.want {
			t.Errorf("ResolveWatermarkOutputExt(%q, %q) = %q, want %q", c.origExt, c.outFormat, got, c.want)
		}
	}
}
