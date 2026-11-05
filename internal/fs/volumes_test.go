package fs

import (
	"testing"
)

func TestGetVolumeStats(t *testing.T) {
	total, free, err := GetVolumeStats("/")
	if err != nil {
		t.Fatalf("GetVolumeStats(/) failed: %v", err)
	}
	if total == 0 {
		t.Errorf("expected total > 0, got %d", total)
	}
	if free == 0 {
		t.Errorf("expected free > 0, got %d", free)
	}
}

func TestListVolumes(t *testing.T) {
	vols, err := ListVolumes()
	if err != nil {
		t.Fatalf("ListVolumes failed: %v", err)
	}
	if len(vols) == 0 {
		t.Fatalf("expected at least 1 volume (root), got 0")
	}

	foundRoot := false
	for _, v := range vols {
		if v.MountPoint == "/" {
			foundRoot = true
			if v.TotalBytes == 0 || v.FreeBytes == 0 {
				t.Errorf("root volume metrics zero: %+v", v)
			}
		}
	}
	if !foundRoot {
		t.Errorf("root volume not found in ListVolumes")
	}
}
