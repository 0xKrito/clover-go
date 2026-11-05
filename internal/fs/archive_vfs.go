package fs

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/bighaidao/clover-go/internal/domain"
)

// SupportedArchiveExtensions lists extensions supported for in-place VFS drill-down.
var SupportedArchiveExtensions = []string{
	".zip",
	".tar.gz",
	".tgz",
	".tar",
}

// IsArchiveFile reports whether path points to a supported archive file.
func IsArchiveFile(filePath string) bool {
	lower := strings.ToLower(filePath)
	for _, ext := range SupportedArchiveExtensions {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// IsArchiveVirtualPath checks whether path represents an archive VFS path (contains '#').
func IsArchiveVirtualPath(p string) bool {
	archivePath, _, ok := ParseArchiveVirtualPath(p)
	return ok && IsArchiveFile(archivePath)
}

// ParseArchiveVirtualPath splits a virtual path into the real archive path and internal path.
func ParseArchiveVirtualPath(p string) (archivePath, internalPath string, ok bool) {
	idx := strings.Index(p, "#")
	if idx == -1 {
		return "", "", false
	}
	archivePath = domain.NormalizePath(p[:idx])
	rawInternal := p[idx+1:]
	internalPath = strings.Trim(filepath.ToSlash(rawInternal), "/")
	return archivePath, internalPath, true
}

// FormatArchiveVirtualPath constructs an archive VFS path.
func FormatArchiveVirtualPath(archivePath, internalPath string) string {
	normArchive := domain.NormalizePath(archivePath)
	cleanInternal := strings.Trim(filepath.ToSlash(internalPath), "/")
	return normArchive + "#" + cleanInternal
}

// ReadArchiveDir reads the entries inside an archive at internalPath.
// Results are cached per (archive, mtime, size); callers receive a defensive
// copy and may re-sort freely.
func ReadArchiveDir(vPath string) ([]domain.FileInfo, error) {
	archivePath, internalPath, ok := ParseArchiveVirtualPath(vPath)
	if !ok {
		return nil, fmt.Errorf("invalid archive virtual path: %q", vPath)
	}
	normInternal := strings.Trim(internalPath, "/")
	if items, ok := lookupArchiveDir(archivePath, normInternal); ok {
		return items, nil
	}

	items, err := readArchiveDirUncached(archivePath, normInternal)
	if err == nil {
		storeArchiveDir(archivePath, normInternal, items)
	}
	return items, err
}

func readArchiveDirUncached(archivePath, normInternal string) ([]domain.FileInfo, error) {
	lower := strings.ToLower(archivePath)
	if strings.HasSuffix(lower, ".zip") {
		return readZipDir(archivePath, normInternal)
	}
	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		return readTarGzDir(archivePath, normInternal)
	}
	if strings.HasSuffix(lower, ".tar") {
		return readTarDir(archivePath, normInternal)
	}

	return nil, fmt.Errorf("unsupported archive format: %s", archivePath)
}

func readZipDir(archivePath, internalPath string) ([]domain.FileInfo, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open zip %q: %w", archivePath, err)
	}
	defer r.Close()

	normInternal := strings.Trim(internalPath, "/")
	dirsSeen := make(map[string]bool)
	var items []domain.FileInfo

	for _, f := range r.File {
		entryName := strings.Trim(filepath.ToSlash(f.Name), "/")
		if entryName == "" || entryName == normInternal {
			continue
		}
		rel, ok := relUnderInternal(entryName, normInternal)
		if !ok {
			continue
		}
		isDir := len(strings.Split(rel, "/")) > 1 || f.FileInfo().IsDir()
		appendArchiveChild(&items, dirsSeen, archivePath, normInternal, rel, isDir,
			int64(f.UncompressedSize64), f.Modified)
	}

	return items, nil
}

func readTarGzDir(archivePath, internalPath string) ([]domain.FileInfo, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	return readTarStream(tar.NewReader(gzr), archivePath, internalPath)
}

func readTarDir(archivePath, internalPath string) ([]domain.FileInfo, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return readTarStream(tar.NewReader(f), archivePath, internalPath)
}

func readTarStream(tr *tar.Reader, archivePath, internalPath string) ([]domain.FileInfo, error) {
	normInternal := strings.Trim(internalPath, "/")
	dirsSeen := make(map[string]bool)
	var items []domain.FileInfo

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		entryName := strings.Trim(filepath.ToSlash(header.Name), "/")
		if entryName == "" || entryName == normInternal {
			continue
		}
		rel, ok := relUnderInternal(entryName, normInternal)
		if !ok {
			continue
		}
		isDir := len(strings.Split(rel, "/")) > 1 || header.Typeflag == tar.TypeDir
		appendArchiveChild(&items, dirsSeen, archivePath, normInternal, rel, isDir,
			header.Size, header.ModTime)
	}

	return items, nil
}

// relUnderInternal returns the entry path relative to the browsed internal
// directory, or ok=false when the entry lives outside it.
func relUnderInternal(entryName, normInternal string) (string, bool) {
	if normInternal == "" {
		return entryName, true
	}
	if strings.HasPrefix(entryName, normInternal+"/") {
		return strings.TrimPrefix(entryName, normInternal+"/"), true
	}
	return "", false
}

// appendArchiveChild aggregates one archive entry into a first-level child
// item (directories are deduplicated via dirsSeen), shared by zip and tar.
func appendArchiveChild(items *[]domain.FileInfo, dirsSeen map[string]bool, archivePath, normInternal, rel string, isDir bool, size int64, modTime time.Time) {
	childName := strings.Split(rel, "/")[0]
	if childName == "" {
		return
	}

	subPath := childName
	if normInternal != "" {
		subPath = path.Join(normInternal, childName)
	}
	if isDir {
		if dirsSeen[childName] {
			return
		}
		dirsSeen[childName] = true
		*items = append(*items, domain.FileInfo{
			Name:    childName,
			Path:    FormatArchiveVirtualPath(archivePath, subPath),
			IsDir:   true,
			ModTime: modTime,
			Kind:    "文件夹",
		})
		return
	}

	// Only a direct child (no deeper path segment) is listed as a file.
	if strings.Contains(rel, "/") {
		return
	}
	*items = append(*items, domain.FileInfo{
		Name:      childName,
		Path:      FormatArchiveVirtualPath(archivePath, subPath),
		IsDir:     false,
		Size:      size,
		ModTime:   modTime,
		Kind:      domain.DetectKind(childName, false),
		Extension: strings.ToLower(filepath.Ext(childName)),
	})
}
