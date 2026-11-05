package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateAndExtractZip(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Prepare source directory
	srcDir := filepath.Join(tempDir, "to_zip")
	_ = os.Mkdir(srcDir, 0755)
	_ = os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("archive content"), 0644)

	zipFile := filepath.Join(tempDir, "archive.zip")

	// 2. Compress with ditto
	if err := CreateZip(srcDir, zipFile); err != nil {
		t.Fatalf("CreateZip failed: %v", err)
	}

	if _, err := os.Stat(zipFile); err != nil {
		t.Fatalf("zip file was not created: %v", err)
	}

	// 3. Extract with ditto
	extractDir := filepath.Join(tempDir, "extracted")
	if err := ExtractZip(zipFile, extractDir); err != nil {
		t.Fatalf("ExtractZip failed: %v", err)
	}

	extractedFile := filepath.Join(extractDir, "test.txt")
	content, err := os.ReadFile(extractedFile)
	if err != nil {
		t.Fatalf("read extracted file failed: %v", err)
	}
	if string(content) != "archive content" {
		t.Errorf("got %q, want 'archive content'", string(content))
	}
}

// TestCreateZipMulti_Regression: ditto overwrites an existing archive
// entirely, so compressing multiple items requires one multi-source call.
// The previous per-item loop into the same target silently kept only the
// last entry — this test guards that regression.
func TestCreateZipMulti_Regression(t *testing.T) {
	tempDir := t.TempDir()

	src1 := filepath.Join(tempDir, "one.txt")
	src2 := filepath.Join(tempDir, "two.txt")
	_ = os.WriteFile(src1, []byte("first"), 0644)
	_ = os.WriteFile(src2, []byte("second"), 0644)

	zipFile := filepath.Join(tempDir, "multi.zip")
	if err := CreateZipMulti([]string{src1, src2}, zipFile); err != nil {
		t.Fatalf("CreateZipMulti failed: %v", err)
	}

	extractDir := filepath.Join(tempDir, "out")
	if err := ExtractZip(zipFile, extractDir); err != nil {
		t.Fatalf("ExtractZip failed: %v", err)
	}

	for name, want := range map[string]string{"one.txt": "first", "two.txt": "second"} {
		got, err := os.ReadFile(filepath.Join(extractDir, name))
		if err != nil {
			t.Fatalf("extracted archive missing %s: %v", name, err)
		}
		if string(got) != want {
			t.Errorf("%s: got %q, want %q", name, got, want)
		}
	}
}
