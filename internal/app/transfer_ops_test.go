package app

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/bighaidao/clover-go/internal/config"
	"github.com/bighaidao/clover-go/internal/fs"
)

func TestTransferFiles_MoveAndCopy(t *testing.T) {
	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "src")
	dstDir := filepath.Join(tempDir, "dst")
	_ = os.Mkdir(srcDir, 0755)
	_ = os.Mkdir(dstDir, 0755)

	file1 := filepath.Join(srcDir, "item1.txt")
	_ = os.WriteFile(file1, []byte("data1"), 0644)

	prefsMgr, _ := config.NewPrefsManager(filepath.Join(tempDir, "prefs.json"))
	wsMgr, _ := config.NewWorkspaceManager(filepath.Join(tempDir, "spaces.json"))
	var mainMu sync.Mutex
	coord := NewCoordinator(srcDir, prefsMgr, wsMgr, nil, func(f func()) {
		mainMu.Lock()
		defer mainMu.Unlock()
		f()
	})
	defer coord.Close()

	// Test Copy
	coord.TransferFiles([]string{file1}, dstDir, ActionCopy)
	// Allow goroutine
	target1 := filepath.Join(dstDir, "item1.txt")
	// Poll briefly for sync
	for i := 0; i < 50; i++ {
		if _, err := os.Stat(target1); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(target1); err != nil {
		t.Errorf("expected copied file to exist at %s", target1)
	}
	if _, err := os.Stat(file1); err != nil {
		t.Errorf("source file should still exist after copy")
	}
	time.Sleep(50 * time.Millisecond)

	// Test Move
	file2 := filepath.Join(srcDir, "item2.txt")
	_ = os.WriteFile(file2, []byte("data2"), 0644)
	coord.TransferFiles([]string{file2}, dstDir, ActionMove)
	target2 := filepath.Join(dstDir, "item2.txt")
	for i := 0; i < 50; i++ {
		if _, err := os.Stat(target2); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(target2); err != nil {
		t.Errorf("expected moved file to exist at %s", target2)
	}
}

func TestTransferFiles_SelfAndRecursiveBlocked(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "parent")
	childDir := filepath.Join(subDir, "child")
	_ = os.Mkdir(subDir, 0755)
	_ = os.Mkdir(childDir, 0755)

	prefsMgr, _ := config.NewPrefsManager(filepath.Join(tempDir, "prefs.json"))
	wsMgr, _ := config.NewWorkspaceManager(filepath.Join(tempDir, "spaces.json"))
	var mainMu sync.Mutex
	coord := NewCoordinator(tempDir, prefsMgr, wsMgr, nil, func(f func()) {
		mainMu.Lock()
		defer mainMu.Unlock()
		f()
	})
	defer coord.Close()

	// Attempt to move parent into child
	coord.TransferFiles([]string{subDir}, childDir, ActionMove)
	if coord.ToastMessage == "" {
		t.Errorf("expected toast warning when moving parent into child")
	}
}

func TestTransferFiles_ConflictInteractiveResolution(t *testing.T) {
	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "src")
	dstDir := filepath.Join(tempDir, "dst")
	_ = os.Mkdir(srcDir, 0755)
	_ = os.Mkdir(dstDir, 0755)

	fileSrc := filepath.Join(srcDir, "dup.txt")
	fileDst := filepath.Join(dstDir, "dup.txt")
	_ = os.WriteFile(fileSrc, []byte("new-data"), 0644)
	_ = os.WriteFile(fileDst, []byte("old-data"), 0644)

	prefsMgr, _ := config.NewPrefsManager(filepath.Join(tempDir, "prefs.json"))
	wsMgr, _ := config.NewWorkspaceManager(filepath.Join(tempDir, "spaces.json"))
	var mainMu sync.Mutex
	coord := NewCoordinator(tempDir, prefsMgr, wsMgr, nil, func(f func()) {
		mainMu.Lock()
		defer mainMu.Unlock()
		f()
	})
	defer coord.Close()

	// Triggers conflict prompt because default is "ask"
	coord.TransferFiles([]string{fileSrc}, dstDir, ActionCopy)
	if !coord.ShowConflictModal || coord.PendingConflict == nil {
		t.Fatalf("expected conflict modal to open on collision")
	}

	// Choose Keep Both (rename)
	coord.ResolveConflict(ChoiceKeepBoth, false)
	keptFile := fs.NextAvailableName(fileDst)
	for i := 0; i < 50; i++ {
		if _, err := os.Stat(keptFile); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(keptFile); err != nil {
		t.Errorf("expected renamed file %s to exist", keptFile)
	}
}

func TestTransferFiles_SameDirMoveDoesNotLockTransferBusy(t *testing.T) {
	tempDir := t.TempDir()
	file := filepath.Join(tempDir, "item.txt")
	_ = os.WriteFile(file, []byte("data"), 0644)

	prefsMgr, _ := config.NewPrefsManager(filepath.Join(tempDir, "prefs.json"))
	wsMgr, _ := config.NewWorkspaceManager(filepath.Join(tempDir, "spaces.json"))
	var mainMu sync.Mutex
	coord := NewCoordinator(tempDir, prefsMgr, wsMgr, nil, func(f func()) {
		mainMu.Lock()
		defer mainMu.Unlock()
		f()
	})
	defer coord.Close()

	// Same directory move is a no-op
	coord.TransferFiles([]string{file}, tempDir, ActionMove)
	if coord.TransferBusy {
		t.Errorf("expected TransferBusy=false after same directory move no-op")
	}

	// Subsequence transfer must not be blocked
	subDir := filepath.Join(tempDir, "sub")
	_ = os.Mkdir(subDir, 0755)
	coord.TransferFiles([]string{file}, subDir, ActionCopy)
	if coord.ToastMessage == "文件转移进行中，请稍候…" {
		t.Errorf("transfer was incorrectly blocked by previous no-op move")
	}
}
