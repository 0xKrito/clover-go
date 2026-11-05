package domain

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkSort100k(b *testing.B) {
	const count = 100000
	baseItems := make([]FileInfo, count)
	now := time.Now()

	for i := 0; i < count; i++ {
		baseItems[i] = FileInfo{
			Name:      fmt.Sprintf("file_%06d.txt", (count-i)%count),
			Path:      fmt.Sprintf("/mock/path/file_%06d.txt", i),
			IsDir:     i%10 == 0, // 10% folders
			Size:      int64(i * 1024),
			ModTime:   now.Add(time.Duration(-i) * time.Minute),
			Kind:      "文本",
			Extension: ".txt",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		items := make([]FileInfo, count)
		copy(items, baseItems)
		SortFiles(items, SortOptions{Field: SortByName, Direction: SortAsc})
	}
}

func BenchmarkFirstScreen200(b *testing.B) {
	const count = 200
	items := make([]FileInfo, count)
	now := time.Now()

	for i := 0; i < count; i++ {
		items[i] = FileInfo{
			Name:      fmt.Sprintf("file_%03d.txt", (count-i)%count),
			Path:      fmt.Sprintf("/mock/file_%03d.txt", i),
			IsDir:     i%5 == 0,
			Size:      int64(i * 1024),
			ModTime:   now.Add(time.Duration(-i) * time.Minute),
			Kind:      "文本",
			Extension: ".txt",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slice := make([]FileInfo, count)
		copy(slice, items)
		SortFiles(slice, SortOptions{Field: SortByName, Direction: SortAsc})
	}
}
