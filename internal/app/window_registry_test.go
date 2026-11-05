package app

import (
	"path/filepath"
	"testing"
)

func TestWindowRegistry_SaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	sessionFile := filepath.Join(tempDir, "session.json")

	reg := NewWindowRegistry(sessionFile, nil)

	// Save empty session
	if err := reg.SaveSession(); err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	data, err := reg.LoadSession()
	if err != nil {
		t.Fatalf("LoadSession failed: %v", err)
	}
	if len(data.Windows) != 0 {
		t.Errorf("expected 0 windows, got %d", len(data.Windows))
	}
}

func TestWindowRegistry_Count(t *testing.T) {
	reg := NewWindowRegistry("", nil)
	reg.Register("win1", nil, nil)
	reg.Register("win2", nil, nil)

	if reg.Count() != 2 {
		t.Errorf("expected 2 windows, got %d", reg.Count())
	}

	reg.Unregister("win1")
	if reg.Count() != 1 {
		t.Errorf("expected 1 window, got %d", reg.Count())
	}
}
