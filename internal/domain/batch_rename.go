package domain

import (
	"fmt"
	"path/filepath"
	"strings"
)

// RenameMode defines the algorithm used for batch renaming.
type RenameMode string

const (
	ModeSequential RenameMode = "sequential"
	ModeAddPrefix  RenameMode = "add_prefix"
	ModeAddSuffix  RenameMode = "add_suffix"
	ModeReplace    RenameMode = "replace"
)

// BatchRenameOptions encapsulates user options for batch renaming.
type BatchRenameOptions struct {
	Mode        RenameMode
	BaseName    string // For ModeSequential
	StartNum    int    // For ModeSequential (e.g. 1)
	Padding     int    // Digits count, e.g. 3 -> "001"
	Separator   string // e.g. "-", "_"
	Prefix      string // For ModeAddPrefix
	Suffix      string // For ModeAddSuffix
	FindText    string // For ModeReplace
	ReplaceText string // For ModeReplace
}

// RenamePair represents the mapping from old name to new name.
type RenamePair struct {
	OriginalPath string `json:"original_path"`
	OldName      string `json:"old_name"`
	NewName      string `json:"new_name"`
	NewPath      string `json:"new_path"`
	Changed      bool   `json:"changed"`
}

// CalculateBatchRename computes the new names for all items and validates conflicts.
func CalculateBatchRename(items []FileInfo, opts BatchRenameOptions) ([]RenamePair, error) {
	if len(items) == 0 {
		return nil, nil
	}

	results := make([]RenamePair, 0, len(items))
	seenNames := make(map[string]bool, len(items))

	for idx, item := range items {
		ext := filepath.Ext(item.Name)
		stem := strings.TrimSuffix(item.Name, ext)
		if item.IsDir {
			stem = item.Name
			ext = ""
		}

		var newName string
		switch opts.Mode {
		case ModeSequential:
			numStr := formatSequentialNumber(opts.StartNum+idx, opts.Padding)
			base := strings.TrimSpace(opts.BaseName)
			if base == "" {
				base = "item"
			}
			if opts.Separator != "" {
				newName = fmt.Sprintf("%s%s%s%s", base, opts.Separator, numStr, ext)
			} else {
				newName = fmt.Sprintf("%s%s%s", base, numStr, ext)
			}

		case ModeAddPrefix:
			newName = opts.Prefix + item.Name

		case ModeAddSuffix:
			newName = stem + opts.Suffix + ext

		case ModeReplace:
			if opts.FindText != "" {
				newName = strings.ReplaceAll(item.Name, opts.FindText, opts.ReplaceText)
			} else {
				newName = item.Name
			}

		default:
			return nil, fmt.Errorf("unknown rename mode: %s", opts.Mode)
		}

		if err := validateFileName(newName); err != nil {
			return nil, fmt.Errorf("item %q rename failed: %w", item.Name, err)
		}

		lowerNew := strings.ToLower(newName)
		if seenNames[lowerNew] {
			return nil, fmt.Errorf("duplicate target name detected: %q", newName)
		}
		seenNames[lowerNew] = true

		parentDir := filepath.Dir(item.Path)
		results = append(results, RenamePair{
			OriginalPath: item.Path,
			OldName:      item.Name,
			NewName:      newName,
			NewPath:      filepath.Join(parentDir, newName),
			Changed:      item.Name != newName,
		})
	}

	return results, nil
}

func formatSequentialNumber(num int, padding int) string {
	if padding <= 1 {
		return fmt.Sprintf("%d", num)
	}
	formatStr := fmt.Sprintf("%%0%dd", padding)
	return fmt.Sprintf(formatStr, num)
}

func validateFileName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("file name cannot be empty")
	}
	if strings.Contains(name, "/") || strings.Contains(name, ":") {
		return fmt.Errorf("file name cannot contain '/' or ':'")
	}
	return nil
}
