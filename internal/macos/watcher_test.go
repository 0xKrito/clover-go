package macos

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// waitFor 轮询等待 cond 成立，超时返回 false。
func waitFor(t *testing.T, d time.Duration, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return cond()
}

// TestDirWatcher_NotifyOnChange 验证目录内新增文件会触发回调。
func TestDirWatcher_NotifyOnChange(t *testing.T) {
	dir := t.TempDir()
	var hits int32
	dw := WatchDirectory(dir, 50*time.Millisecond, func() { atomic.AddInt32(&hits, 1) })
	defer dw.Stop()

	// 等首次事件流建立完成：建立前的内容不会上报。
	time.Sleep(300 * time.Millisecond)
	base := atomic.LoadInt32(&hits)

	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !waitFor(t, 3*time.Second, func() bool { return atomic.LoadInt32(&hits) > base }) {
		t.Fatal("expected onChange after creating a file")
	}
}

// TestDirWatcher_RecoversAfterStreamEnd 验证被监听目录被删除、重建后，
// 监听会自愈并重新上报变化 —— 否则该目录会永久静默失联。
func TestDirWatcher_RecoversAfterStreamEnd(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "watched")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	var hits int32
	dw := WatchDirectory(dir, 50*time.Millisecond, func() { atomic.AddInt32(&hits, 1) })
	defer dw.Stop()

	time.Sleep(300 * time.Millisecond)
	base := atomic.LoadInt32(&hits)

	// 删除目录：事件流结束，watcher 进入退避重建。
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	time.Sleep(watchRetryDelay + 200*time.Millisecond)

	// 重建目录后应当自愈（重建成功会补一次回调）。
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if !waitFor(t, 4*time.Second, func() bool { return atomic.LoadInt32(&hits) > base }) {
		t.Fatal("expected onChange after the watched directory was recreated")
	}

	// 自愈后的流仍在工作：再次写入应当继续触发回调。
	before := atomic.LoadInt32(&hits)
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !waitFor(t, 3*time.Second, func() bool { return atomic.LoadInt32(&hits) > before }) {
		t.Fatal("expected onChange after the watch recovered")
	}
}

// TestDirWatcher_StopSilencesCallback 验证 Stop 之后不再回调，避免关窗后
// 回调把监听复活。
func TestDirWatcher_StopSilencesCallback(t *testing.T) {
	dir := t.TempDir()
	var hits int32
	dw := WatchDirectory(dir, 50*time.Millisecond, func() { atomic.AddInt32(&hits, 1) })
	time.Sleep(300 * time.Millisecond)
	dw.Stop()
	base := atomic.LoadInt32(&hits)

	if err := os.WriteFile(filepath.Join(dir, "c.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	if got := atomic.LoadInt32(&hits); got != base {
		t.Fatalf("expected no onChange after Stop, got %d (base %d)", got, base)
	}
}
