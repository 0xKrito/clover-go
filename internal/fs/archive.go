package fs

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CreateZip compresses srcPath into dstZip using macOS native ditto.
func CreateZip(srcPath, dstZip string) error {
	return CreateZipMulti([]string{srcPath}, dstZip)
}

// CreateZipMulti compresses one or more source paths into a single zip via
// macOS ditto. Ditto overwrites an existing archive entirely (no append), so
// all sources must be archived in one call; issuing one CreateZip per item
// into the same dstZip would silently drop all but the last entry.
//
// ditto cannot archive multiple flat sources into a zip ("Can't archive
// multiple sources"), so multi-selection is staged into a hidden temp
// directory next to dstZip (same volume, no cross-device copy) first; the
// archive root then equals the staged directory's contents, matching Finder's
// flat "归档.zip" layout.
func CreateZipMulti(srcPaths []string, dstZip string) error {
	if len(srcPaths) == 0 {
		return fmt.Errorf("no source items to compress")
	}
	for _, src := range srcPaths {
		if _, err := os.Lstat(src); err != nil {
			return fmt.Errorf("source item not found: %w", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(dstZip), 0755); err != nil {
		return fmt.Errorf("create zip destination dir: %w", err)
	}

	stageDir := ""
	archiveSrc := srcPaths[0]
	if len(srcPaths) > 1 {
		var err error
		stageDir, err = os.MkdirTemp(filepath.Dir(dstZip), ".clover-zip-*")
		if err != nil {
			return fmt.Errorf("create staging dir: %w", err)
		}
		defer os.RemoveAll(stageDir)

		for _, src := range srcPaths {
			// KeepBoth matches Finder when items from different folders share
			// a name ("x.txt", "x 2.txt").
			if _, err := CopyItem(src, stageDir, PolicyKeepBoth); err != nil {
				return fmt.Errorf("stage %q: %w", src, err)
			}
		}
		archiveSrc = stageDir
	}

	// -c: create archive
	// -k: create PKZip format
	// --sequesterRsrc: preserve Mac resource forks and Finder metadata
	cmd := exec.Command("/usr/bin/ditto", "-c", "-k", "--sequesterRsrc", archiveSrc, dstZip)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ditto zip failed: %s (%w)", stderr.String(), err)
	}
	return nil
}

// ExtractZip extracts zipPath into dstDir using macOS native ditto.
func ExtractZip(zipPath, dstDir string) error {
	if _, err := os.Lstat(zipPath); err != nil {
		return fmt.Errorf("zip file not found: %w", err)
	}

	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("create extraction dir: %w", err)
	}

	// -x: extract archive
	// -k: PKZip format
	cmd := exec.Command("/usr/bin/ditto", "-x", "-k", zipPath, dstDir)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ditto unzip failed: %s (%w)", stderr.String(), err)
	}
	return nil
}

// ExtractArchive extracts zip, tar, or tar.gz to dstDir using native macOS tools.
func ExtractArchive(archivePath, dstDir string) error {
	lower := strings.ToLower(archivePath)
	if strings.HasSuffix(lower, ".zip") {
		return ExtractZip(archivePath, dstDir)
	}

	if _, err := os.Lstat(archivePath); err != nil {
		return fmt.Errorf("archive file not found: %w", err)
	}
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("create extraction dir: %w", err)
	}

	var cmd *exec.Cmd
	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		cmd = exec.Command("/usr/bin/tar", "-xzf", archivePath, "-C", dstDir)
	} else if strings.HasSuffix(lower, ".tar") {
		cmd = exec.Command("/usr/bin/tar", "-xf", archivePath, "-C", dstDir)
	} else {
		return fmt.Errorf("unsupported archive format: %s", archivePath)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tar extract failed: %s (%w)", stderr.String(), err)
	}
	return nil
}
