package domain

import (
	"testing"
	"time"
)

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{-1, "--"},
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{10 * 1024 * 1024, "10.0 MB"},
		{150 * 1024 * 1024, "150 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{5 * 1024 * 1024 * 1024 * 1024, "5.0 TB"},
	}

	for _, tt := range tests {
		got := FormatSize(tt.bytes)
		if got != tt.expected {
			t.Errorf("FormatSize(%d) = %q, want %q", tt.bytes, got, tt.expected)
		}
	}
}

func TestFormatModTime(t *testing.T) {
	now := time.Date(2026, 10, 28, 14, 0, 0, 0, time.Local)

	target1 := time.Date(2026, 9, 20, 14, 40, 0, 0, time.Local)
	if got := FormatModTime(target1, now); got != "2026/9/20 14:40" {
		t.Errorf("FormatModTime target1 = %q, want '2026/9/20 14:40'", got)
	}

	target2 := time.Date(2025, 11, 18, 8, 30, 0, 0, time.Local)
	if got := FormatModTime(target2, now); got != "2025/11/18 08:30" {
		t.Errorf("FormatModTime target2 = %q, want '2025/11/18 08:30'", got)
	}

	target3 := time.Date(2024, 12, 7, 16, 2, 0, 0, time.Local)
	if got := FormatModTime(target3, now); got != "2024/12/7 16:02" {
		t.Errorf("FormatModTime target3 = %q, want '2024/12/7 16:02'", got)
	}

	if got := FormatModTime(time.Time{}, now); got != "--" {
		t.Errorf("FormatModTime zero = %q, want '--'", got)
	}
}

func TestDetectKind(t *testing.T) {
	if got := DetectKind(".app", true); got != "macOS 应用程序" {
		t.Errorf("expected macOS 应用程序 for .app dir, got %q", got)
	}
	if got := DetectKind(".app", false); got != "macOS 应用程序" {
		t.Errorf("expected macOS 应用程序 for .app file, got %q", got)
	}
	if got := DetectKind("", true); got != "文件夹" {
		t.Errorf("expected 文件夹, got %q", got)
	}
	if got := DetectKind(".go", false); got != "Go 源码" {
		t.Errorf("expected Go 源码, got %q", got)
	}
	if got := DetectKind(".png", false); got != "图像文件" {
		t.Errorf("expected 图像文件, got %q", got)
	}
	if got := DetectKind(".mp4", false); got != "视频文件" {
		t.Errorf("expected 视频文件, got %q", got)
	}
	if got := DetectKind(".zip", false); got != "归档文件" {
		t.Errorf("expected 归档文件, got %q", got)
	}
	if got := DetectKind(".unknown_xyz", false); got != "UNKNOWN_XYZ 文件" {
		t.Errorf("expected UNKNOWN_XYZ 文件, got %q", got)
	}
	if got := DetectKind("", false); got != "文件" {
		t.Errorf("expected 文件, got %q", got)
	}
}

func TestIsHiddenName(t *testing.T) {
	if !IsHiddenName(".git") {
		t.Errorf("expected true for .git")
	}
	if !IsHiddenName("/path/to/.DS_Store") {
		t.Errorf("expected true for /path/to/.DS_Store")
	}
	if IsHiddenName("main.go") {
		t.Errorf("expected false for main.go")
	}
}
