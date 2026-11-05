package search

import (
	"context"
	"io/fs"
	"path/filepath"
	"sync"
	"time"

	"github.com/bighaidao/clover-go/internal/domain"
)

const (
	defaultTTL     = 30 * time.Second
	maxCachedItems = 50000
)

type cacheEntry struct {
	items     []domain.FileInfo
	indexedAt time.Time
}

// Engine coordinates directory indexing and fuzzy search.
type Engine struct {
	mu    sync.RWMutex
	cache map[string]cacheEntry
	ttl   time.Duration
}

// NewEngine creates an initialized search engine.
func NewEngine() *Engine {
	return &Engine{
		cache: make(map[string]cacheEntry),
		ttl:   defaultTTL,
	}
}

// Search queries files recursively under root matching the query string.
func (e *Engine) Search(ctx context.Context, root, query string, maxResults int) ([]domain.FileInfo, error) {
	normRoot := domain.NormalizePath(root)
	items, err := e.getOrBuildIndex(ctx, normRoot)
	if err != nil {
		return nil, err
	}

	cleanQuery := query
	var matched []SearchResult
	for _, item := range items {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if score, ok := ScoreItem(cleanQuery, item.Name); ok {
			matched = append(matched, SearchResult{Item: item, Score: score})
		}
	}

	SortSearchResults(matched)

	limit := len(matched)
	if maxResults > 0 && limit > maxResults {
		limit = maxResults
	}

	finalList := make([]domain.FileInfo, limit)
	for i := 0; i < limit; i++ {
		finalList[i] = matched[i].Item
	}

	return finalList, nil
}

func (e *Engine) getOrBuildIndex(ctx context.Context, root string) ([]domain.FileInfo, error) {
	e.mu.RLock()
	entry, found := e.cache[root]
	e.mu.RUnlock()

	if found && time.Since(entry.indexedAt) < e.ttl {
		return entry.items, nil
	}

	items, err := e.scanDirectory(ctx, root)
	if err != nil {
		return nil, err
	}

	e.mu.Lock()
	e.cache[root] = cacheEntry{
		items:     items,
		indexedAt: time.Now(),
	}
	e.mu.Unlock()

	return items, nil
}

func (e *Engine) scanDirectory(ctx context.Context, root string) ([]domain.FileInfo, error) {
	var items []domain.FileInfo

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible directories gracefully
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if path == root {
			return nil
		}

		name := d.Name()
		// Skip giant hidden system/version-control trees
		if d.IsDir() && (name == ".git" || name == ".Trash" || name == "node_modules") {
			return filepath.SkipDir
		}

		if len(items) >= maxCachedItems {
			return filepath.SkipAll
		}

		isDir := d.IsDir()
		ext := filepath.Ext(name)
		items = append(items, domain.FileInfo{
			Name:      name,
			Path:      path,
			IsDir:     isDir,
			IsHidden:  domain.IsHiddenName(name),
			Extension: ext,
			Kind:      domain.DetectKind(ext, isDir),
		})

		return nil
	})

	return items, err
}

// ClearCache flushes cached indices (e.g. on directory file change events).
func (e *Engine) ClearCache(root string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if root == "" {
		e.cache = make(map[string]cacheEntry)
	} else {
		delete(e.cache, domain.NormalizePath(root))
	}
}
