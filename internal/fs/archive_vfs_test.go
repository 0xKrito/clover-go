package fs

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func createTestZip(t *testing.T, targetZip string) {
	f, err := os.Create(targetZip)
	if err != nil {
		t.Fatalf("create zip file: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	// 1. Root file
	w1, err := zw.Create("readme.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w1.Write([]byte("hello world"))

	// 2. Subfolder file
	w2, err := zw.Create("icons/logo.svg")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w2.Write([]byte("<svg></svg>"))

	// 3. Deep subfolder file
	w3, err := zw.Create("icons/sub/deep.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w3.Write([]byte("pngdata"))
}

func TestArchiveVFS_ReadZipDir(t *testing.T) {
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "test.zip")
	createTestZip(t, zipPath)

	// Test 1: IsArchiveFile & IsArchiveVirtualPath
	if !IsArchiveFile(zipPath) {
		t.Fatalf("expected IsArchiveFile(%q) to be true", zipPath)
	}
	rootVPath := FormatArchiveVirtualPath(zipPath, "")
	if !IsArchiveVirtualPath(rootVPath) {
		t.Fatalf("expected IsArchiveVirtualPath(%q) to be true", rootVPath)
	}

	// Test 2: Read root entries of zip
	items, err := ReadArchiveDir(rootVPath)
	if err != nil {
		t.Fatalf("ReadArchiveDir root failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items at root (readme.txt and icons), got %d: %v", len(items), items)
	}

	var hasReadme, hasIconsDir bool
	for _, it := range items {
		if it.Name == "readme.txt" && !it.IsDir {
			hasReadme = true
		}
		if it.Name == "icons" && it.IsDir {
			hasIconsDir = true
		}
	}
	if !hasReadme || !hasIconsDir {
		t.Errorf("missing expected entries in zip root: readme=%v, icons=%v", hasReadme, hasIconsDir)
	}

	// Test 3: Read subdirectory inside zip
	subVPath := FormatArchiveVirtualPath(zipPath, "icons")
	subItems, err := ReadArchiveDir(subVPath)
	if err != nil {
		t.Fatalf("ReadArchiveDir subfolder failed: %v", err)
	}
	if len(subItems) != 2 {
		t.Fatalf("expected 2 items in icons (logo.svg and sub), got %d: %v", len(subItems), subItems)
	}

	// Test 4: Extract file to temp
	extracted, err := ExtractArchiveFileToTemp(FormatArchiveVirtualPath(zipPath, "readme.txt"))
	if err != nil {
		t.Fatalf("ExtractArchiveFileToTemp failed: %v", err)
	}
	content, err := os.ReadFile(extracted)
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if string(content) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(content))
	}

	// Test 5: ExtractVirtualItem to real disk
	outExtractDir := filepath.Join(tempDir, "extracted_out")
	err = ExtractVirtualItem(FormatArchiveVirtualPath(zipPath, "icons/logo.svg"), outExtractDir)
	if err != nil {
		t.Fatalf("ExtractVirtualItem failed: %v", err)
	}
	extractedLogo := filepath.Join(outExtractDir, "logo.svg")
	logoData, err := os.ReadFile(extractedLogo)
	if err != nil {
		t.Fatalf("read extracted logo: %v", err)
	}
	if string(logoData) != "<svg></svg>" {
		t.Errorf("expected '<svg></svg>', got %q", string(logoData))
	}
}

// createTestTarGz builds a minimal tar.gz archive for VFS tests.
func createTestTarGz(t *testing.T, targetPath string) {
	f, err := os.Create(targetPath)
	if err != nil {
		t.Fatalf("create tar.gz: %v", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	addFile := func(name, body string) {
		tw.WriteHeader(&tar.Header{
			Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg,
		})
		_, _ = tw.Write([]byte(body))
	}
	addFile("docs/readme.md", "tar readme")
	addFile("docs/assets/img.txt", "tar img")
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestArchiveVFS_TarGz locks the tar/tar.gz support that was previously
// zip-only (QuickLook/extract silently failed for tar archives).
func TestArchiveVFS_TarGz(t *testing.T) {
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "bundle.tar.gz")
	createTestTarGz(t, archivePath)

	if !IsArchiveFile(archivePath) {
		t.Fatalf("expected .tar.gz to be a supported archive")
	}
	rootItems, err := ReadArchiveDir(FormatArchiveVirtualPath(archivePath, ""))
	if err != nil {
		t.Fatalf("ReadArchiveDir root failed: %v", err)
	}
	if len(rootItems) != 1 || rootItems[0].Name != "docs" || !rootItems[0].IsDir {
		t.Fatalf("expected single docs dir at root, got %v", rootItems)
	}

	subItems, err := ReadArchiveDir(FormatArchiveVirtualPath(archivePath, "docs"))
	if err != nil {
		t.Fatalf("ReadArchiveDir docs failed: %v", err)
	}
	if len(subItems) != 2 {
		t.Fatalf("expected 2 items in docs, got %v", subItems)
	}

	// 空格预览路径：tar 条目提取到隔离临时文件。
	tmpFile, err := ExtractArchiveFileToTemp(FormatArchiveVirtualPath(archivePath, "docs/readme.md"))
	if err != nil {
		t.Fatalf("ExtractArchiveFileToTemp tar failed: %v", err)
	}
	if body, err := os.ReadFile(tmpFile); err != nil || string(body) != "tar readme" {
		t.Errorf("unexpected tar temp content: %q err=%v", body, err)
	}

	// 右键提取路径：tar 子目录落盘。
	outDir := filepath.Join(tempDir, "tar_out")
	if err := ExtractVirtualItem(FormatArchiveVirtualPath(archivePath, "docs/assets"), outDir); err != nil {
		t.Fatalf("ExtractVirtualItem tar failed: %v", err)
	}
	if body, err := os.ReadFile(filepath.Join(outDir, "assets", "img.txt")); err != nil || string(body) != "tar img" {
		t.Errorf("unexpected tar extract content: %q err=%v", body, err)
	}
}

// TestExtractVirtualItem_ZipSlipRejected proves malicious archive entries
// containing "../" cannot write outside the destination directory.
func TestExtractVirtualItem_ZipSlipRejected(t *testing.T) {
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "evil.zip")

	// 构造含 ../ 逃逸条目的恶意 zip（archive/zip writer 不拦）。
	zf, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(zf)
	w, err := zw.Create("a/../../evil_escape.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("pwned"))
	_ = zw.Close()
	_ = zf.Close()

	dstDir := filepath.Join(tempDir, "dst")
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		t.Fatal(err)
	}

	err = ExtractVirtualItem(FormatArchiveVirtualPath(zipPath, "a"), dstDir)
	if err == nil {
		t.Fatalf("expected Zip-Slip entry to be rejected")
	}

	// 逃逸目标（tempDir/evil_escape.txt）绝不能存在。
	if _, statErr := os.Stat(filepath.Join(tempDir, "evil_escape.txt")); statErr == nil {
		t.Errorf("Zip-Slip escape succeeded: file written outside dst")
	}
}

// TestExtractArchiveFileToTemp_Isolated verifies that same-named files from
// different archives no longer clobber each other in the QuickLook scratch dir.
func TestExtractArchiveFileToTemp_Isolated(t *testing.T) {
	tempDir := t.TempDir()
	p1 := filepath.Join(tempDir, "one.zip")
	p2 := filepath.Join(tempDir, "two.zip")
	makeZip := func(path, body string) {
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		zw := zip.NewWriter(f)
		w, _ := zw.Create("same.txt")
		_, _ = w.Write([]byte(body))
		_ = zw.Close()
		_ = f.Close()
	}
	makeZip(p1, "content-one")
	makeZip(p2, "content-two")

	t1, err := ExtractArchiveFileToTemp(FormatArchiveVirtualPath(p1, "same.txt"))
	if err != nil {
		t.Fatal(err)
	}
	t2, err := ExtractArchiveFileToTemp(FormatArchiveVirtualPath(p2, "same.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if t1 == t2 {
		t.Fatalf("expected per-archive temp paths, got identical %s", t1)
	}
	if body, _ := os.ReadFile(t1); string(body) != "content-one" {
		t.Errorf("archive one temp content clobbered: %q", body)
	}
	if body, _ := os.ReadFile(t2); string(body) != "content-two" {
		t.Errorf("archive two temp content clobbered: %q", body)
	}
}
