package domain

import (
	"testing"
	"time"
)

func TestNaturalCompare(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		expected int // <0, 0, >0
	}{
		{"file1", "file2", -1},
		{"file2", "file10", -1},
		{"file02", "file10", -1},
		{"file10", "file2", 1},
		{"item", "item", 0},
		{"A", "a", 0},
		{"a", "b", -1},
		{"b", "a", 1},
		{"version-1.9", "version-1.10", -1},
	}

	for _, tt := range tests {
		got := NaturalCompare(tt.s1, tt.s2)
		if (tt.expected < 0 && got >= 0) || (tt.expected > 0 && got <= 0) || (tt.expected == 0 && got != 0) {
			t.Errorf("NaturalCompare(%q, %q) = %d, expected sign %d", tt.s1, tt.s2, got, tt.expected)
		}
	}
}

func TestSortFiles_NameAndDirsFirst(t *testing.T) {
	items := []FileInfo{
		{Name: "file-b.txt", IsDir: false},
		{Name: "dir-b", IsDir: true},
		{Name: "file-a.txt", IsDir: false},
		{Name: "dir-a", IsDir: true},
	}

	SortFiles(items, SortOptions{Field: SortByName, Direction: SortAsc})

	expected := []string{"dir-a", "dir-b", "file-a.txt", "file-b.txt"}
	for i, name := range expected {
		if items[i].Name != name {
			t.Errorf("Index %d: got %s, want %s", i, items[i].Name, name)
		}
	}
}

func TestSortFiles_ModTimeMixed(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)

	items := []FileInfo{
		{Name: "old_dir", IsDir: true, ModTime: t1},
		{Name: "newest_file", IsDir: false, ModTime: t3},
		{Name: "mid_dir", IsDir: true, ModTime: t2},
	}

	// ModTime desc (newest first). Finder mixes folders and files here!
	SortFiles(items, SortOptions{Field: SortByModTime, Direction: SortDesc})

	expected := []string{"newest_file", "mid_dir", "old_dir"}
	for i, name := range expected {
		if items[i].Name != name {
			t.Errorf("ModTime Desc: index %d got %s, want %s", i, items[i].Name, name)
		}
	}
}

func TestSortFiles_CreateTimeMixed(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)

	items := []FileInfo{
		{Name: "old_dir", IsDir: true, CreateTime: t1},
		{Name: "newest_file", IsDir: false, CreateTime: t3},
		{Name: "mid_dir", IsDir: true, CreateTime: t2},
	}

	// CreateTime desc (newest first). Finder mixes folders and files here!
	SortFiles(items, SortOptions{Field: SortByCreateTime, Direction: SortDesc})

	expected := []string{"newest_file", "mid_dir", "old_dir"}
	for i, name := range expected {
		if items[i].Name != name {
			t.Errorf("CreateTime Desc: index %d got %s, want %s", i, items[i].Name, name)
		}
	}
}

func TestSortFiles_CreateTimeZeroFallsBackToModTime(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)

	// Archive VFS entries carry no CreateTime; they must not be treated as
	// the oldest possible time but fall back to their ModTime.
	items := []FileInfo{
		{Name: "vfs_oldest", ModTime: t1}, // CreateTime zero
		{Name: "newest", CreateTime: t3},
		{Name: "mid", CreateTime: t2},
	}

	SortFiles(items, SortOptions{Field: SortByCreateTime, Direction: SortAsc})

	expected := []string{"vfs_oldest", "mid", "newest"}
	for i, name := range expected {
		if items[i].Name != name {
			t.Errorf("CreateTime Asc fallback: index %d got %s, want %s", i, items[i].Name, name)
		}
	}
}

func TestSortFiles_Size(t *testing.T) {
	items := []FileInfo{
		{Name: "big_file.iso", IsDir: false, Size: 1000},
		{Name: "small_dir", IsDir: true, Size: 0},
		{Name: "small_file.txt", IsDir: false, Size: 10},
	}

	// Folders first, then size asc
	SortFiles(items, SortOptions{Field: SortBySize, Direction: SortAsc})

	if items[0].Name != "small_dir" {
		t.Errorf("expected small_dir first, got %s", items[0].Name)
	}
	if items[1].Name != "small_file.txt" {
		t.Errorf("expected small_file.txt second, got %s", items[1].Name)
	}
	if items[2].Name != "big_file.iso" {
		t.Errorf("expected big_file.iso third, got %s", items[2].Name)
	}
}
