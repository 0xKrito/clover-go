package macos

import (
	"context"
	"sync"
	"time"

	"github.com/egoist/mygo"
)

// DirWatcher watches a directory for filesystem events with debouncing.
type DirWatcher struct {
	mu        sync.Mutex
	cancel    context.CancelFunc
	watcher   *mygo.FileWatcher
	isClosed  bool
	onChange  func()
	debounceD time.Duration
	// debounceTimer 持有当前防抖定时器，确保 Stop() 时能同步终止未触发的回调，
	// 避免回调在 Close 之后再触发 onChange 导致 watcher 被 restartWatcher 复活。
	debounceTimer *time.Timer
}

// WatchDirectory starts watching dirPath, invoking onChange when changes settle.
func WatchDirectory(dirPath string, debounce time.Duration, onChange func()) *DirWatcher {
	ctx, cancel := context.WithCancel(context.Background())
	dw := &DirWatcher{
		cancel:    cancel,
		onChange:  onChange,
		debounceD: debounce,
	}

	fw, err := mygo.WatchFiles(ctx, dirPath, mygo.FileWatchOptions{Recursive: false})
	if err != nil {
		cancel()
		return dw
	}
	dw.watcher = fw

	go dw.eventLoop(ctx)
	return dw
}

func (w *DirWatcher) eventLoop(ctx context.Context) {
	for {
		_, err := w.watcher.Next(ctx)
		if err != nil {
			return
		}

		w.mu.Lock()
		if w.isClosed {
			w.mu.Unlock()
			return
		}

		if w.debounceTimer != nil {
			w.debounceTimer.Stop()
		}
		w.debounceTimer = time.AfterFunc(w.debounceD, func() {
			// 二次守卫：定时器触发时若 watcher 已 Stop，直接放弃回调，
			// 杜绝 Stop 后 onChange -> LoadCurrentTab -> restartWatcher 的复活链。
			w.mu.Lock()
			closed := w.isClosed
			w.mu.Unlock()
			if closed {
				return
			}
			if w.onChange != nil {
				w.onChange()
			}
		})
		w.mu.Unlock()
	}
}

// Stop cleanly terminates the watcher.
func (w *DirWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.isClosed {
		w.isClosed = true
		if w.debounceTimer != nil {
			w.debounceTimer.Stop()
			w.debounceTimer = nil
		}
		w.cancel()
		if w.watcher != nil {
			_ = w.watcher.Close()
		}
	}
}
