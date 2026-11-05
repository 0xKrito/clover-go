package config

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// InstanceLock represents an exclusive file lock on gui.lock.
type InstanceLock struct {
	file *os.File
}

// Release unlocks and closes the file lock.
func (l *InstanceLock) Release() {
	if l.file != nil {
		_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
		_ = l.file.Close()
		l.file = nil
	}
}

// SandboxPaths provides resolved paths for CloverGo application state.
type SandboxPaths struct {
	RootDir    string
	ConfigDir  string
	CacheDir   string
	LogsDir    string
	ThumbsDir  string
	LockFile   string
	PrefsFile  string
	SpacesFile string
}

// ResolvePaths initializes and creates sandbox directories under ~/.clover-go/.
func ResolvePaths() (*SandboxPaths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user home directory: %w", err)
	}

	root := filepath.Join(home, ".clover-go")
	cache := filepath.Join(root, "cache")
	logs := filepath.Join(root, "logs")
	thumbs := filepath.Join(cache, "thumbs")

	for _, dir := range []string{root, cache, logs, thumbs} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create sandbox directory %q: %w", dir, err)
		}
	}

	return &SandboxPaths{
		RootDir:    root,
		ConfigDir:  root,
		CacheDir:   cache,
		LogsDir:    logs,
		ThumbsDir:  thumbs,
		LockFile:   filepath.Join(root, "gui.lock"),
		PrefsFile:  filepath.Join(root, "preferences.json"),
		SpacesFile: filepath.Join(root, "workspaces.json"),
	}, nil
}

// AcquireInstanceLock obtains an advisory lock to prevent multiple instances.
func AcquireInstanceLock(lockPath string) (*InstanceLock, error) {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("open lock file %q: %w", lockPath, err)
	}

	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("another instance of CloverGo is already running")
	}

	return &InstanceLock{file: f}, nil
}
