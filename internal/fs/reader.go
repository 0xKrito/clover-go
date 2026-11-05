package fs

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/bighaidao/clover-go/internal/macos"
)

// ReadDirOptions controls pagination and filtering when querying directory entries.
type ReadDirOptions struct {
	Path       string
	ShowHidden bool
	Offset     int
	Limit      int // 0 means return all remaining items
}

// ReadDirPage reads a directory or archive VFS path, parses basic metadata, and applies pagination.
func ReadDirPage(opts ReadDirOptions) ([]domain.FileInfo, int, error) {
	if IsArchiveVirtualPath(opts.Path) {
		items, err := ReadArchiveDir(opts.Path)
		if err != nil {
			return nil, 0, err
		}
		total := len(items)
		start := opts.Offset
		if start < 0 {
			start = 0
		}
		if start >= total {
			return []domain.FileInfo{}, total, nil
		}
		end := total
		if opts.Limit > 0 && start+opts.Limit < total {
			end = start + opts.Limit
		}
		return items[start:end], total, nil
	}

	normPath := domain.NormalizePath(opts.Path)
	entries, err := os.ReadDir(normPath)
	if err != nil {
		return nil, 0, fmt.Errorf("read directory %q: %w", normPath, err)
	}

	filtered := make([]fs.DirEntry, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !opts.ShowHidden && domain.IsHiddenName(name) {
			continue
		}
		filtered = append(filtered, entry)
	}

	total := len(filtered)
	if total == 0 {
		return []domain.FileInfo{}, 0, nil
	}

	start := opts.Offset
	if start < 0 {
		start = 0
	}
	if start >= total {
		return []domain.FileInfo{}, total, nil
	}

	end := total
	if opts.Limit > 0 && start+opts.Limit < total {
		end = start + opts.Limit
	}

	pageSlice := filtered[start:end]
	items := make([]domain.FileInfo, 0, len(pageSlice))
	for _, entry := range pageSlice {
		item := buildFileInfo(normPath, entry)
		items = append(items, item)
	}

	return items, total, nil
}

func buildFileInfo(parent string, entry fs.DirEntry) domain.FileInfo {
	name := entry.Name()
	fullPath := filepath.Join(parent, name)
	isDir := entry.IsDir()
	isSymlink := entry.Type()&os.ModeSymlink != 0
	isAppBundle := strings.HasSuffix(strings.ToLower(name), ".app")

	var symlinkDest string
	if isSymlink {
		if dest, err := os.Readlink(fullPath); err == nil {
			symlinkDest = dest
		}
	}

	ext := filepath.Ext(name)
	info, err := entry.Info()
	if err != nil {
		return domain.FileInfo{
			Name:        name,
			Path:        fullPath,
			IsDir:       isDir,
			IsAppBundle: isAppBundle,
			IsSymlink:   isSymlink,
			SymlinkDest: symlinkDest,
			IsHidden:    domain.IsHiddenName(name),
			Extension:   ext,
			Kind:        domain.DetectKind(ext, isDir),
		}
	}

	// If symlink target is a directory, mark as directory if target exists
	if isSymlink && !isDir {
		if targetInfo, err := os.Stat(fullPath); err == nil {
			isDir = targetInfo.IsDir()
		}
	}

	var size int64
	if !isDir {
		size = info.Size()
	}

	var colorTags []string
	if tags, _, err := macos.GetFileTags(fullPath); err == nil && len(tags) > 0 {
		colorTags = tags
	}

	return domain.FileInfo{
		Name:        name,
		Path:        fullPath,
		IsDir:       isDir,
		IsAppBundle: isAppBundle,
		IsSymlink:   isSymlink,
		SymlinkDest: symlinkDest,
		IsHidden:    domain.IsHiddenName(name),
		Size:        size,
		ModTime:     info.ModTime(),
		CreateTime:  birthTime(info),
		Mode:        info.Mode(),
		Kind:        domain.DetectKind(ext, isDir),
		Extension:   ext,
		ColorTags:   colorTags,
	}
}

// birthTime extracts the macOS file creation time (birthtime) from the
// syscall stat buffer; falls back to ModTime when unavailable.
func birthTime(info os.FileInfo) time.Time {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return time.Unix(st.Birthtimespec.Sec, st.Birthtimespec.Nsec)
	}
	return info.ModTime()
}

// CalculateDirSize recursively aggregates directory byte count and file count.
func CalculateDirSize(ctx context.Context, dirPath string) (int64, int, error) {
	var totalBytes int64
	var totalItems int

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible items gracefully
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if path == dirPath {
			return nil
		}
		totalItems++
		if !d.IsDir() {
			if info, err := d.Info(); err == nil {
				totalBytes += info.Size()
			}
		}
		return nil
	})

	return totalBytes, totalItems, err
}
