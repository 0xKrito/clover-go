package macos

import (
	"context"
	"sync"
	"time"

	"github.com/egoist/mygo"
)

// watchRetryDelay 是事件流建立失败或中断后再次尝试的间隔：既避免目录不可用
// 时高频重试，也让监听在 FSEvents 流失效、系统睡眠唤醒、目录被删除后重建等
// 情况下自动恢复，而不是静默失效直到用户手动重载目录。
const watchRetryDelay = time.Second

// DirWatcher watches a directory for filesystem events with debouncing.
//
// 监听是自愈的：mygo 的事件流一旦结束（目录被删除或替换、系统丢掉事件、
// 流本身失效），这里会丢弃旧流、等待 watchRetryDelay 后重建；重建成功后再
// 主动回调一次 onChange —— 中断期间的变化不会被重放，这次回调让界面重新读取
// 目录，避免文件增删被静默遗漏。WatchFiles 建立失败同样只是重试，不会让该
// 目录永久失去监听。
type DirWatcher struct {
	mu        sync.Mutex
	path      string
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
		path:      dirPath,
		cancel:    cancel,
		onChange:  onChange,
		debounceD: debounce,
	}
	go dw.eventLoop(ctx)
	return dw
}

func (w *DirWatcher) eventLoop(ctx context.Context) {
	// retry 表示这一轮之前已有过一次连接，本次连接成功说明监听刚从中断中
	// 恢复：中断期间的变化需要补一次刷新。
	retry := false
	for {
		fw, ok := w.connect(ctx)
		if !ok {
			return
		}
		if retry {
			w.fire()
		}
		for {
			_, err := fw.Next(ctx)
			if err == nil {
				w.schedule()
				continue
			}
			if ctx.Err() != nil {
				return
			}
			// 事件流结束（目录被删除/替换、系统丢弃事件、流失效）：丢弃旧流重建。
			break
		}
		w.disconnect()
		retry = true
		select {
		case <-ctx.Done():
			return
		case <-time.After(watchRetryDelay):
		}
	}
}

// connect 建立事件流，失败时按 watchRetryDelay 重试；ctx 结束或 watcher 已
// 关闭时返回 false。
func (w *DirWatcher) connect(ctx context.Context) (*mygo.FileWatcher, bool) {
	for {
		fw, err := mygo.WatchFiles(ctx, w.path, mygo.FileWatchOptions{Recursive: false})
		if err == nil {
			w.mu.Lock()
			if w.isClosed {
				w.mu.Unlock()
				_ = fw.Close()
				return nil, false
			}
			w.watcher = fw
			w.mu.Unlock()
			return fw, true
		}
		select {
		case <-ctx.Done():
			return nil, false
		case <-time.After(watchRetryDelay):
		}
	}
}

// disconnect 释放当前事件流，等待 connect 重建。
func (w *DirWatcher) disconnect() {
	w.mu.Lock()
	fw := w.watcher
	w.watcher = nil
	w.mu.Unlock()
	if fw != nil {
		_ = fw.Close()
	}
}

// schedule 重置防抖定时器，变化停止 debounceD 后再回调。
func (w *DirWatcher) schedule() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.isClosed {
		return
	}
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}
	w.debounceTimer = time.AfterFunc(w.debounceD, w.fire)
}

// fire 立即回调一次 onChange；watcher 关闭后不再回调。
func (w *DirWatcher) fire() {
	w.mu.Lock()
	closed := w.isClosed
	w.mu.Unlock()
	if closed || w.onChange == nil {
		return
	}
	w.onChange()
}

// Stop cleanly terminates the watcher.
func (w *DirWatcher) Stop() {
	w.mu.Lock()
	if w.isClosed {
		w.mu.Unlock()
		return
	}
	w.isClosed = true
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
		w.debounceTimer = nil
	}
	fw := w.watcher
	w.watcher = nil
	w.mu.Unlock()

	w.cancel()
	if fw != nil {
		_ = fw.Close()
	}
}
