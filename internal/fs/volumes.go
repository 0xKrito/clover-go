package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// VolumeInfo carries information about a mounted disk volume.
type VolumeInfo struct {
	Name        string `json:"name"`
	MountPoint  string `json:"mount_point"`
	TotalBytes  uint64 `json:"total_bytes"`
	FreeBytes   uint64 `json:"free_bytes"`
	IsRemovable bool   `json:"is_removable"`
}

// GetVolumeStats returns storage metrics for a mount point using unix.Statfs.
func GetVolumeStats(mountPoint string) (total uint64, free uint64, err error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(mountPoint, &stat); err != nil {
		return 0, 0, fmt.Errorf("statfs %q: %w", mountPoint, err)
	}

	total = stat.Blocks * uint64(stat.Bsize)
	free = stat.Bavail * uint64(stat.Bsize)
	return total, free, nil
}

// ListVolumes enumerates root disk and mounted external volumes.
func ListVolumes() ([]VolumeInfo, error) {
	var results []VolumeInfo

	// 1. Root volume (Macintosh HD)
	rootTotal, rootFree, err := GetVolumeStats("/")
	if err == nil {
		results = append(results, VolumeInfo{
			Name:        "Macintosh HD",
			MountPoint:  "/",
			TotalBytes:  rootTotal,
			FreeBytes:   rootFree,
			IsRemovable: false,
		})
	}

	// 2. Mounted external volumes in /Volumes
	volEntries, err := os.ReadDir("/Volumes")
	if err != nil {
		return results, nil
	}

	for _, entry := range volEntries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		mountPoint := filepath.Join("/Volumes", name)

		// Check if it's a symlink to / (Macintosh HD often has a symlink in /Volumes)
		if dest, err := filepath.EvalSymlinks(mountPoint); err == nil && dest == "/" {
			continue
		}

		tot, fre, err := GetVolumeStats(mountPoint)
		if err != nil {
			continue
		}

		results = append(results, VolumeInfo{
			Name:        name,
			MountPoint:  mountPoint,
			TotalBytes:  tot,
			FreeBytes:   fre,
			IsRemovable: true,
		})
	}

	return results, nil
}
