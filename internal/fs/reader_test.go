package fs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReadDirPage(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	_ = os.WriteFile(filepath.Join(tempDir, "a.txt"), []byte("hello"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "b.txt"), []byte("world"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, ".hidden"), []byte("secret"), 0644)
	_ = os.Mkdir(filepath.Join(tempDir, "sub"), 0755)

	// Test without hidden files
	items, total, err := ReadDirPage(ReadDirOptions{
		Path:       tempDir,
		ShowHidden: false,
	})
	if err != nil {
		t.Fatalf("ReadDirPage failed: %v", err)
	}
	if total != 3 {
		t.Errorf("expected 3 items without hidden, got %d", total)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 returned items, got %d", len(items))
	}

	// Test with hidden files
	_, totalWithHidden, err := ReadDirPage(ReadDirOptions{
		Path:       tempDir,
		ShowHidden: true,
	})
	if err != nil || totalWithHidden != 4 {
		t.Errorf("expected 4 items with hidden, got %d (err: %v)", totalWithHidden, err)
	}

	// Test pagination: Offset 1, Limit 1
	pagedItems, totalPaged, err := ReadDirPage(ReadDirOptions{
		Path:       tempDir,
		ShowHidden: false,
		Offset:     1,
		Limit:      1,
	})
	if err != nil {
		t.Fatalf("paged ReadDirPage failed: %v", err)
	}
	if totalPaged != 3 {
		t.Errorf("expected total 3, got %d", totalPaged)
	}
	if len(pagedItems) != 1 {
		t.Errorf("expected 1 item in page, got %d", len(pagedItems))
	}
}

func TestCalculateDirSize(t *testing.T) {
	tempDir := t.TempDir()

	_ = os.WriteFile(filepath.Join(tempDir, "file1.bin"), make([]byte, 100), 0644)
	subDir := filepath.Join(tempDir, "subdir")
	_ = os.Mkdir(subDir, 0755)
	_ = os.WriteFile(filepath.Join(subDir, "file2.bin"), make([]byte, 200), 0644)

	ctx := context.Background()
	bytes, count, err := CalculateDirSize(ctx, tempDir)
	if err != nil {
		t.Fatalf("CalculateDirSize failed: %v", err)
	}

	if bytes != 300 {
		t.Errorf("expected 300 bytes, got %d", bytes)
	}
	// count includes file1.bin, subdir, and file2.bin = 3
	if count != 3 {
		t.Errorf("expected 3 items count, got %d", count)
	}

	// Test cancellation
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()
	_, _, err = CalculateDirSize(cancelCtx, tempDir)
	if err == nil {
		t.Errorf("expected error on canceled context, got nil")
	}
}
