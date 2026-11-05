package macos

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bighaidao/clover-go/internal/domain"
)

func TestApplyWatermark(t *testing.T) {
	tempDir := t.TempDir()
	baseImg := filepath.Join(tempDir, "base.jpg")

	// Create a dummy image via the helper itself
	if _, err := runHelper("make-test-image", baseImg, "200", "200", "jpg"); err != nil {
		t.Fatalf("setup test image failed: %v", err)
	}

	opts := domain.DefaultWatermarkOptions()
	opts.SourceFiles = []string{baseImg}
	opts.Layout = domain.WatermarkLayoutCenter

	if err := ApplyWatermark(opts); err != nil {
		t.Fatalf("ApplyWatermark failed: %v", err)
	}

	watermarkedPath := filepath.Join(tempDir, "base_watermarked.jpg")
	info, err := os.Stat(watermarkedPath)
	if err != nil || info.Size() == 0 {
		t.Fatalf("expected watermarked image at %s, err: %v", watermarkedPath, err)
	}
}
