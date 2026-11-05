package icon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestComputeCacheKey(t *testing.T) {
	k1 := computeCacheKey("/test/path", 12345, 6789, 64)
	k2 := computeCacheKey("/test/path", 12345, 6789, 64)
	k3 := computeCacheKey("/test/path", 12345, 6789, 128)

	if k1 != k2 {
		t.Errorf("cache keys should be deterministic")
	}
	if k1 == k3 {
		t.Errorf("different sizes should produce different keys")
	}
}

func TestImageThumbGeneration(t *testing.T) {
	tempDir := t.TempDir()
	imgPath := filepath.Join(tempDir, "sample.png")

	// Generate a 100x50 test image
	src := image.NewRGBA(image.Rect(0, 0, 100, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 100; x++ {
			src.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, src)
	_ = os.WriteFile(imgPath, buf.Bytes(), 0644)

	// Generate thumbnail scaled down to max 32px
	thumbData, err := generateImageThumb(imgPath, 32)
	if err != nil {
		t.Fatalf("generateImageThumb failed: %v", err)
	}

	decoded, _, err := image.Decode(bytes.NewReader(thumbData))
	if err != nil {
		t.Fatalf("decode thumb failed: %v", err)
	}

	if decoded.Bounds().Dx() > 32 || decoded.Bounds().Dy() > 32 {
		t.Errorf("thumbnail dimensions exceeded: %v", decoded.Bounds())
	}
}

func TestPipelineLRU(t *testing.T) {
	tempDir := t.TempDir()
	pipeline := NewPipeline(tempDir)
	defer pipeline.Close()

	// Initial query misses
	bmp := pipeline.GetOrQueue("/fake/path.txt", 100, 200, 32, nil)
	if bmp != nil {
		t.Errorf("expected initial LRU miss (nil)")
	}

	time.Sleep(50 * time.Millisecond)
}
