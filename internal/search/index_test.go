package search

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bighaidao/clover-go/internal/domain"
)

func TestEngine_Search(t *testing.T) {
	tempDir := t.TempDir()

	_ = os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "util.go"), []byte("package util"), 0644)
	subDir := filepath.Join(tempDir, "sub")
	_ = os.Mkdir(subDir, 0755)
	_ = os.WriteFile(filepath.Join(subDir, "main_test.go"), []byte("package main"), 0644)

	engine := NewEngine()
	ctx := context.Background()

	// 1. Search for "main"
	results, err := engine.Search(ctx, tempDir, "main", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(results))
	}
	if results[0].Name != "main.go" {
		t.Errorf("expected main.go first, got %s", results[0].Name)
	}

	// 2. MaxResults limit
	limited, err := engine.Search(ctx, tempDir, "main", 1)
	if err != nil || len(limited) != 1 {
		t.Fatalf("expected 1 limited result, got %d (err: %v)", len(limited), err)
	}

	// 3. Cache verification
	engine.mu.RLock()
	entry, found := engine.cache[tempDir]
	engine.mu.RUnlock()
	if !found || len(entry.items) == 0 {
		t.Errorf("cache was not populated")
	}

	// 4. ClearCache
	engine.ClearCache(tempDir)
	engine.mu.RLock()
	_, foundAfter := engine.cache[tempDir]
	engine.mu.RUnlock()
	if foundAfter {
		t.Errorf("expected cache to be cleared")
	}
}

func TestEngine_Cancellation(t *testing.T) {
	tempDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("data"), 0644)

	engine := NewEngine()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := engine.Search(ctx, tempDir, "test", 10)
	if err == nil {
		t.Errorf("expected context canceled error, got nil")
	}
}

func TestEngine_TTL(t *testing.T) {
	engine := NewEngine()
	engine.ttl = 10 * time.Millisecond

	tempDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tempDir, "a.txt"), []byte("a"), 0644)

	ctx := context.Background()
	_, _ = engine.Search(ctx, tempDir, "a", 10)

	// Wait past TTL
	time.Sleep(20 * time.Millisecond)

	// Add new file
	_ = os.WriteFile(filepath.Join(tempDir, "b.txt"), []byte("b"), 0644)

	results, _ := engine.Search(ctx, tempDir, "b", 10)
	if len(results) != 1 || results[0].Name != "b.txt" {
		t.Errorf("expected fresh index after TTL expiration, got %v", results)
	}
}

func BenchmarkSearch100k(b *testing.B) {
	// 准备 100,000 个虚拟文件名
	items := make([]domain.FileInfo, 100_000)
	for i := range items {
		items[i] = domain.FileInfo{
			Name: filepath.Join("dir", "file_sub_component_log_2026_"+string(rune('a'+(i%26)))+".dat"),
			Path: "/virtual/path",
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var matches []SearchResult
		for _, it := range items {
			if score, ok := ScoreItem("component log", it.Name); ok {
				matches = append(matches, SearchResult{Item: it, Score: score})
				if len(matches) >= 50 {
					break
				}
			}
		}
		SortSearchResults(matches)
	}
}
