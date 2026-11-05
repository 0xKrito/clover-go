package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ConflictPolicy defines how to handle filename collisions during copy/move.
type ConflictPolicy int

const (
	PolicyPrompt ConflictPolicy = iota
	PolicyOverwrite
	PolicySkip
	PolicyKeepBoth
)

// NextAvailableName generates a non-colliding name like "filename 2.ext" in the same directory.
func NextAvailableName(targetPath string) string {
	if _, err := os.Lstat(targetPath); os.IsNotExist(err) {
		return targetPath
	}

	dir := filepath.Dir(targetPath)
	base := filepath.Base(targetPath)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)

	// Check if target is directory; if so, ext shouldn't be separated
	if fi, err := os.Lstat(targetPath); err == nil && fi.IsDir() {
		stem = base
		ext = ""
	}

	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s %d%s", stem, i, ext)
		candidatePath := filepath.Join(dir, candidate)
		if _, err := os.Lstat(candidatePath); os.IsNotExist(err) {
			return candidatePath
		}
	}
}

// CreateDir creates a new directory under parent with 0755 permissions.
func CreateDir(parent, name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("directory name cannot be empty")
	}
	target := filepath.Join(parent, name)
	if _, err := os.Lstat(target); err == nil {
		target = NextAvailableName(target)
	}
	if err := os.MkdirAll(target, 0755); err != nil {
		return "", fmt.Errorf("create directory %q: %w", target, err)
	}
	return target, nil
}

// RenameItem renames a file or directory within its parent folder.
func RenameItem(oldPath, newName string) (string, error) {
	if strings.TrimSpace(newName) == "" {
		return "", fmt.Errorf("new name cannot be empty")
	}
	if strings.Contains(newName, "/") {
		return "", fmt.Errorf("name cannot contain '/'")
	}
	parent := filepath.Dir(oldPath)
	newPath := filepath.Join(parent, newName)
	if oldPath == newPath {
		return newPath, nil
	}
	if _, err := os.Lstat(newPath); err == nil {
		return "", fmt.Errorf("item already exists: %s", newName)
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return "", fmt.Errorf("rename %q to %q: %w", oldPath, newPath, err)
	}
	return newPath, nil
}

// DuplicateItem creates a copy of the given item with " 2" or next index suffix.
func DuplicateItem(src string) (string, error) {
	dir := filepath.Dir(src)
	nextPath := NextAvailableName(src)
	if nextPath == src {
		nextPath = NextAvailableName(filepath.Join(dir, filepath.Base(src)+" copy"))
	}
	return CopyItem(src, dir, PolicyKeepBoth)
}

// CopyItem copies a file or directory into dstDir, adhering to the given ConflictPolicy.
func CopyItem(src, dstDir string, policy ConflictPolicy) (string, error) {
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return "", fmt.Errorf("stat source %q: %w", src, err)
	}

	target := filepath.Join(dstDir, srcInfo.Name())
	finalTarget, skip, err := resolveDestination(target, policy)
	if err != nil || skip {
		return finalTarget, err
	}

	if srcInfo.IsDir() {
		return finalTarget, copyDirectory(src, finalTarget, srcInfo.Mode())
	}
	return finalTarget, copySingleFile(src, finalTarget, srcInfo.Mode())
}

// MoveItem moves a file or directory into dstDir adhering to ConflictPolicy.
func MoveItem(src, dstDir string, policy ConflictPolicy) (string, error) {
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return "", fmt.Errorf("stat source %q: %w", src, err)
	}

	target := filepath.Join(dstDir, srcInfo.Name())
	finalTarget, skip, err := resolveDestination(target, policy)
	if err != nil || skip {
		return finalTarget, err
	}

	// Try atomic rename first
	if err := os.Rename(src, finalTarget); err == nil {
		return finalTarget, nil
	}

	// Fallback to cross-device copy and delete
	if _, err := CopyItem(src, dstDir, policy); err != nil {
		return "", fmt.Errorf("move fallback copy: %w", err)
	}
	return finalTarget, os.RemoveAll(src)
}

func resolveDestination(target string, policy ConflictPolicy) (string, bool, error) {
	if _, err := os.Lstat(target); os.IsNotExist(err) {
		return target, false, nil
	}

	switch policy {
	case PolicySkip:
		return target, true, nil
	case PolicyOverwrite:
		// 仅目录与符号链接需要先移除（rename 无法覆盖非空目录、写入文件会跟随链接）；
		// 普通文件交给 rename 原子覆盖 / O_TRUNC 写入，避免“先删后拷”在 copy
		// 中途失败（磁盘满、只读卷等）时丢失目标位置的旧文件。
		if fi, statErr := os.Lstat(target); statErr == nil && (fi.IsDir() || fi.Mode()&os.ModeSymlink != 0) {
			_ = os.RemoveAll(target)
		}
		return target, false, nil
	case PolicyKeepBoth:
		return NextAvailableName(target), false, nil
	case PolicyPrompt:
		return "", false, fmt.Errorf("conflict: %q already exists", filepath.Base(target))
	default:
		return target, false, nil
	}
}

func copySingleFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}

	if info, err := in.Stat(); err == nil {
		_ = os.Chtimes(dst, info.ModTime(), info.ModTime())
	}
	return nil
}

func copyDirectory(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(dst, mode); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		sSub := filepath.Join(src, entry.Name())
		dSub := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDirectory(sSub, dSub, 0755); err != nil {
				return err
			}
		} else {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if err := copySingleFile(sSub, dSub, info.Mode()); err != nil {
				return err
			}
		}
	}
	return nil
}

// ValidateTransfer validates whether srcs can be legally and safely transferred into dstDir.
// It prevents self-drop, virtual archive writes, and recursive nesting (parent into child).
func ValidateTransfer(srcs []string, dstDir string) error {
	dstDir = strings.TrimSpace(dstDir)
	if dstDir == "" {
		return fmt.Errorf("目标目录无效")
	}
	if IsArchiveVirtualPath(dstDir) {
		return fmt.Errorf("只读压缩包内部不支持写入文件")
	}
	if len(srcs) == 0 {
		return fmt.Errorf("没有选中的文件")
	}

	cleanDst := filepath.Clean(dstDir)
	dstPrefix := cleanDst + string(filepath.Separator)

	for _, src := range srcs {
		cleanSrc := filepath.Clean(src)
		if cleanSrc == cleanDst {
			return fmt.Errorf("目标目录不能与源项目相同: %s", filepath.Base(cleanSrc))
		}

		// Prevent moving a directory into its own subfolder: e.g. /a/b into /a/b/c
		srcPrefix := cleanSrc + string(filepath.Separator)
		if strings.HasPrefix(dstPrefix, srcPrefix) {
			return fmt.Errorf("不能将文件夹 “%s” 放入其子文件夹中", filepath.Base(cleanSrc))
		}

		// 词法前缀检查无法识别 symlink 别名路径（如 /alias → /a），
		// 对疑似目录源（含指向目录的符号链接，Lstat 不会跟随）按真实路径兑底一次，
		// 且仅在真实目标确为目录时拦截，防止把目录经别名移入自身子树造成递归死循环。
		if fi, statErr := os.Lstat(cleanSrc); statErr == nil && (fi.IsDir() || fi.Mode()&os.ModeSymlink != 0) {
			realSrc, err1 := filepath.EvalSymlinks(cleanSrc)
			realDst, err2 := filepath.EvalSymlinks(cleanDst)
			if err1 == nil && err2 == nil {
				if realFi, statErr := os.Stat(realSrc); statErr == nil && realFi.IsDir() &&
					strings.HasPrefix(realDst+string(filepath.Separator), realSrc+string(filepath.Separator)) {
					return fmt.Errorf("不能将文件夹 “%s” 放入其子文件夹中", filepath.Base(cleanSrc))
				}
			}
		}
	}
	return nil
}

// IsSameDirectory reports whether src is directly contained within dstDir.
func IsSameDirectory(src, dstDir string) bool {
	return filepath.Clean(filepath.Dir(src)) == filepath.Clean(dstDir)
}
