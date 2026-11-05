package fs

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// quickLookTempRoot 是 VFS 预览临时文件的根目录；子目录按压缩包路径区分，
// 避免不同压缩包内同名文件互相覆盖，并在每次提取时惰性清理过期残留。
const quickLookTempRoot = "clover_vfs_ql"

const quickLookTempTTL = 24 * time.Hour

// ExtractArchiveFileToTemp extracts a single virtual archive file (any
// supported format) to an isolated temporary file for QuickLook.
func ExtractArchiveFileToTemp(vPath string) (string, error) {
	archivePath, internalPath, ok := ParseArchiveVirtualPath(vPath)
	if !ok {
		return "", fmt.Errorf("not an archive virtual path: %s", vPath)
	}

	tmpDir := quickLookDirFor(archivePath)
	outPath := filepath.Join(tmpDir, filepath.Base(internalPath))

	lower := strings.ToLower(archivePath)
	if strings.HasSuffix(lower, ".zip") {
		if err := extractZipEntryToFile(archivePath, internalPath, outPath); err != nil {
			return "", err
		}
		return outPath, nil
	}
	if isTarFamily(lower) {
		if err := extractTarEntryToFile(archivePath, internalPath, outPath); err != nil {
			return "", err
		}
		return outPath, nil
	}
	return "", fmt.Errorf("unsupported archive format: %s", archivePath)
}

// isTarFamily reports whether the lowercase path is a tar/tar.gz/tgz archive.
func isTarFamily(lowerPath string) bool {
	return strings.HasSuffix(lowerPath, ".tar") ||
		strings.HasSuffix(lowerPath, ".tar.gz") ||
		strings.HasSuffix(lowerPath, ".tgz")
}

// quickLookDirFor returns the per-archive temp dir and prunes expired files.
func quickLookDirFor(archivePath string) string {
	root := filepath.Join(os.TempDir(), quickLookTempRoot)
	_ = os.MkdirAll(root, 0755)
	pruneStaleQuickLookFiles(root)

	// 与物理路径一一对应的安全子目录名（避免路径分隔符/特殊字符）。
	key := strings.ReplaceAll(strings.TrimPrefix(archivePath, "/"), "/", "_")
	if key == "" {
		key = "root"
	}
	dir := filepath.Join(root, key)
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// pruneStaleQuickLookFiles removes preview temp files older than the TTL so
// the QuickLook scratch area never grows unbounded.
func pruneStaleQuickLookFiles(root string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-quickLookTempTTL)
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		_ = os.RemoveAll(filepath.Join(root, e.Name()))
	}
}

// openTarReader opens a tar/tar.gz/tgz archive and returns a tar reader.
func openTarReader(archivePath string) (*tar.Reader, func(), error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = f.Close() }

	lower := strings.ToLower(archivePath)
	if strings.HasSuffix(lower, ".tar") {
		return tar.NewReader(f), cleanup, nil
	}
	gzr, err := gzip.NewReader(f)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	return tar.NewReader(gzr), cleanup, nil
}

func extractZipEntryToFile(archivePath, internalPath, outPath string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	normInternal := strings.Trim(internalPath, "/")
	for _, f := range r.File {
		if strings.Trim(filepath.ToSlash(f.Name), "/") != normInternal {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		return writeFileAtomic(outPath, rc)
	}
	return fmt.Errorf("file not found in archive: %s", internalPath)
}

func extractTarEntryToFile(archivePath, internalPath, outPath string) error {
	tr, cleanup, err := openTarReader(archivePath)
	if err != nil {
		return err
	}
	defer cleanup()

	normInternal := strings.Trim(internalPath, "/")
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header.Typeflag == tar.TypeDir {
			continue
		}
		if strings.Trim(filepath.ToSlash(header.Name), "/") != normInternal {
			continue
		}
		return writeFileAtomic(outPath, tr)
	}
	return fmt.Errorf("file not found in archive: %s", internalPath)
}

// ExtractVirtualItem extracts a virtual archive file or folder into dstDir on
// the physical filesystem. Every target path is confined under dstDir to
// prevent Zip-Slip style traversal from malicious archive entries.
func ExtractVirtualItem(vPath, dstDir string) error {
	archivePath, internalPath, ok := ParseArchiveVirtualPath(vPath)
	if !ok {
		return fmt.Errorf("invalid archive virtual path: %s", vPath)
	}
	if strings.Trim(internalPath, "/") == "" {
		return ExtractArchive(archivePath, dstDir)
	}

	lower := strings.ToLower(archivePath)
	if strings.HasSuffix(lower, ".zip") {
		return extractVirtualFromZip(archivePath, internalPath, dstDir)
	}
	if isTarFamily(lower) {
		return extractVirtualFromTar(archivePath, internalPath, dstDir)
	}
	return fmt.Errorf("unsupported archive format: %s", archivePath)
}

// safeJoinUnder joins root and rel and rejects any result escaping root
// (Zip-Slip defense for entries containing "../").
func safeJoinUnder(root, rel string) (string, error) {
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	target := filepath.Join(cleanRoot, rel)
	if target != cleanRoot && !strings.HasPrefix(target, cleanRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("archive entry %q escapes destination directory", rel)
	}
	return target, nil
}

func extractVirtualFromZip(archivePath, internalPath, dstDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	normInternal := strings.Trim(internalPath, "/")
	for _, f := range r.File {
		entryName := strings.Trim(filepath.ToSlash(f.Name), "/")
		if entryName != normInternal && !strings.HasPrefix(entryName, normInternal+"/") {
			continue
		}
		rel := filepath.Base(normInternal) + strings.TrimPrefix(entryName, normInternal)
		targetPath, err := safeJoinUnder(dstDir, rel)
		if err != nil {
			return err
		}

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(targetPath, 0755)
			continue
		}
		if err := copyZipEntryToFile(f, targetPath); err != nil {
			return err
		}
	}
	return nil
}

func copyZipEntryToFile(f *zip.File, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, rc)
	return err
}

func extractVirtualFromTar(archivePath, internalPath, dstDir string) error {
	tr, cleanup, err := openTarReader(archivePath)
	if err != nil {
		return err
	}
	defer cleanup()

	normInternal := strings.Trim(internalPath, "/")
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		entryName := strings.Trim(filepath.ToSlash(header.Name), "/")
		if entryName != normInternal && !strings.HasPrefix(entryName, normInternal+"/") {
			continue
		}
		rel := filepath.Base(normInternal) + strings.TrimPrefix(entryName, normInternal)
		targetPath, err := safeJoinUnder(dstDir, rel)
		if err != nil {
			return err
		}

		if header.Typeflag == tar.TypeDir {
			_ = os.MkdirAll(targetPath, 0755)
			continue
		}
		if header.Typeflag != tar.TypeReg {
			continue // 跳过符号链接等非常规条目，保持只读语义安全
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}
		out, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, tr)
		_ = out.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

// CleanupQuickLookTemp prunes expired VFS preview temp files; call once at
// app startup so stale scratch data never survives across sessions.
func CleanupQuickLookTemp() {
	root := filepath.Join(os.TempDir(), quickLookTempRoot)
	if _, err := os.Stat(root); err != nil {
		return
	}
	pruneStaleQuickLookFiles(root)
}

// writeFileAtomic writes r to outPath via a temp file + rename so a failed
// extraction never leaves a truncated preview file behind.
func writeFileAtomic(outPath string, r io.Reader) error {
	tmp, err := os.CreateTemp(filepath.Dir(outPath), ".ql-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := io.Copy(tmp, r); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, outPath); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}
