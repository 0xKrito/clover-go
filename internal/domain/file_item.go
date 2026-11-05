package domain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileInfo represents the domain-level metadata of a file or folder.
type FileInfo struct {
	Name        string      `json:"name"`
	Path        string      `json:"path"`
	IsDir       bool        `json:"is_dir"`
	IsAppBundle bool        `json:"is_app_bundle,omitempty"`
	IsSymlink   bool        `json:"is_symlink"`
	SymlinkDest string      `json:"symlink_dest,omitempty"`
	IsHidden    bool        `json:"is_hidden"`
	Size        int64       `json:"size"`
	ModTime     time.Time   `json:"mod_time"`
	CreateTime  time.Time   `json:"create_time"` // macOS birthtime; zero value falls back to ModTime when sorting/displaying
	Mode        os.FileMode `json:"mode"`
	Kind        string      `json:"kind"`
	Extension   string      `json:"extension"`
	ColorTags   []string    `json:"color_tags,omitempty"`
}

// FormatSize converts a byte count into a human-readable string (KB, MB, GB).
func FormatSize(bytes int64) string {
	if bytes < 0 {
		return "--"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	suffixes := []string{"KB", "MB", "GB", "TB", "PB"}
	val := float64(bytes) / float64(div)
	if val >= 100 {
		return fmt.Sprintf("%.0f %s", val, suffixes[exp])
	}
	return fmt.Sprintf("%.1f %s", val, suffixes[exp])
}

// FormatModTime formats modification/creation time into unified standard "2006/1/2 15:04" (e.g. 2026/9/20 14:40).
func FormatModTime(t time.Time, _ time.Time) string {
	if t.IsZero() {
		return "--"
	}
	return t.Local().Format("2006/1/2 15:04")
}

// DetectKind returns a human-friendly description of the file type.
func DetectKind(ext string, isDir bool) string {
	cleanExt := strings.ToLower(strings.TrimPrefix(ext, "."))
	if cleanExt == "app" {
		return "macOS 应用程序"
	}
	if isDir {
		return "文件夹"
	}
	switch cleanExt {
	case "go":
		return "Go 源码"
	case "rs":
		return "Rust 源码"
	case "swift":
		return "Swift 源码"
	case "c", "h", "cpp", "hpp":
		return "C/C++ 源码"
	case "py":
		return "Python 脚本"
	case "js", "ts", "jsx", "tsx":
		return "TypeScript/JavaScript"
	case "json", "yaml", "yml", "toml", "xml":
		return "配置文件"
	case "md", "markdown":
		return "Markdown 文档"
	case "txt", "log":
		return "纯文本文档"
	case "pdf":
		return "PDF 文档"
	case "doc", "docx", "pages":
		return "文稿文档"
	case "xls", "xlsx", "numbers", "csv":
		return "电子表格"
	case "ppt", "pptx", "key":
		return "演示文稿"
	case "png", "jpg", "jpeg", "gif", "webp", "svg", "bmp", "ico", "heic", "tiff":
		return "图像文件"
	case "mp4", "mov", "mkv", "avi", "webm":
		return "视频文件"
	case "mp3", "m4a", "wav", "flac", "aac":
		return "音频文件"
	case "zip", "tar", "gz", "bz2", "xz", "7z", "dmg", "pkg":
		return "归档文件"
	default:
		if cleanExt == "" {
			return "文件"
		}
		return strings.ToUpper(cleanExt) + " 文件"
	}
}

// IsHiddenName checks if a file name represents a hidden file (starts with a dot).
func IsHiddenName(name string) bool {
	return strings.HasPrefix(filepath.Base(name), ".")
}
