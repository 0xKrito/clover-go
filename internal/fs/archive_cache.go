package fs

import (
	"os"
	"sync"

	"github.com/bighaidao/clover-go/internal/domain"
)

// archiveDirCache 缓存压缩包目录列表：tar 家族列目录需要全量流式解压（大包
// 秒级卡顿），缓存将重复导航/刷新降为 O(1)。key 含源档 mtime+size，文件变更
// 自然失效；超容量按插入顺序 FIFO 淘汰（导航场景足够，无需严格 LRU）。
const archiveCacheCapacity = 64

type archiveCacheKey struct {
	archivePath string
	internal    string
	mtimeNano   int64
	size        int64
}

type archiveDirCache struct {
	mu      sync.Mutex
	entries map[archiveCacheKey][]domain.FileInfo
	order   []archiveCacheKey
}

var dirCache = &archiveDirCache{entries: make(map[archiveCacheKey][]domain.FileInfo)}

func archiveKeyFor(archivePath, internal string) (archiveCacheKey, bool) {
	info, err := os.Stat(archivePath)
	if err != nil {
		return archiveCacheKey{}, false
	}
	return archiveCacheKey{
		archivePath: archivePath,
		internal:    internal,
		mtimeNano:   info.ModTime().UnixNano(),
		size:        info.Size(),
	}, true
}

// lookupArchiveDir returns a defensive copy of the cached listing so callers
// may re-sort their slice without polluting the cache.
func lookupArchiveDir(archivePath, internal string) ([]domain.FileInfo, bool) {
	key, ok := archiveKeyFor(archivePath, internal)
	if !ok {
		return nil, false
	}
	dirCache.mu.Lock()
	defer dirCache.mu.Unlock()
	items, ok := dirCache.entries[key]
	if !ok {
		return nil, false
	}
	out := make([]domain.FileInfo, len(items))
	copy(out, items)
	return out, true
}

// storeArchiveDir caches a copy of the listing for future lookups.
func storeArchiveDir(archivePath, internal string, items []domain.FileInfo) {
	key, ok := archiveKeyFor(archivePath, internal)
	if !ok {
		return
	}
	dirCache.mu.Lock()
	defer dirCache.mu.Unlock()

	if _, exists := dirCache.entries[key]; !exists {
		dirCache.order = append(dirCache.order, key)
		if len(dirCache.order) > archiveCacheCapacity {
			evict := dirCache.order[0]
			dirCache.order = dirCache.order[1:]
			delete(dirCache.entries, evict)
		}
	}
	cached := make([]domain.FileInfo, len(items))
	copy(cached, items)
	dirCache.entries[key] = cached
}
