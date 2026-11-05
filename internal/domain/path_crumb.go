package domain

import (
	"fmt"
	"path/filepath"
	"strings"
)

// PathCrumb represents a segment of a path for breadcrumb navigation.
type PathCrumb struct {
	Name     string `json:"name"`
	FullPath string `json:"full_path"`
	IsHome   bool   `json:"is_home"`
	IsLast   bool   `json:"is_last"`
}

// TabSummary carries minimal info needed for tab disambiguation.
type TabSummary struct {
	ID   string
	Path string
}

// NormalizePath cleans up and resolves clean POSIX path representation.
func NormalizePath(path string) string {
	cleaned := filepath.Clean(strings.TrimSpace(path))
	if cleaned == "" || cleaned == "." {
		return "/"
	}
	return cleaned
}

// CollapseHome replaces the homeDir prefix with "~".
func CollapseHome(path, homeDir string) string {
	normPath := NormalizePath(path)
	normHome := NormalizePath(homeDir)
	if normHome == "/" || normHome == "" {
		return normPath
	}

	if normPath == normHome {
		return "~"
	}
	if strings.HasPrefix(normPath, normHome+"/") {
		return "~" + strings.TrimPrefix(normPath, normHome)
	}
	return normPath
}

// SplitPathCrumbs breaks a path into clickable breadcrumb pieces.
func SplitPathCrumbs(path, homeDir string) []PathCrumb {
	if hashIdx := strings.Index(path, "#"); hashIdx != -1 {
		return splitArchiveCrumbs(path, hashIdx, homeDir)
	}

	normPath := NormalizePath(path)
	normHome := NormalizePath(homeDir)

	if normPath == "/" {
		return []PathCrumb{{Name: "Macintosh HD", FullPath: "/", IsLast: true}}
	}

	isUnderHome := normHome != "/" && normHome != "" && (normPath == normHome || strings.HasPrefix(normPath, normHome+"/"))
	if isUnderHome {
		return buildHomeCrumbs(normPath, normHome)
	}
	return buildRootCrumbs(normPath)
}

func splitArchiveCrumbs(path string, hashIdx int, homeDir string) []PathCrumb {
	archivePath := NormalizePath(path[:hashIdx])
	internalPart := strings.Trim(path[hashIdx+1:], "/")
	crumbs := SplitPathCrumbs(archivePath, homeDir)
	if len(crumbs) > 0 {
		crumbs[len(crumbs)-1].FullPath = archivePath + "#"
		crumbs[len(crumbs)-1].IsLast = (internalPart == "")
	}
	if internalPart != "" {
		subParts := strings.Split(internalPart, "/")
		accum := ""
		for i, sub := range subParts {
			if sub == "" {
				continue
			}
			if accum == "" {
				accum = sub
			} else {
				accum = accum + "/" + sub
			}
			crumbs = append(crumbs, PathCrumb{
				Name:     sub,
				FullPath: archivePath + "#" + accum,
				IsLast:   i == len(subParts)-1,
			})
		}
	}
	return crumbs
}

func buildHomeCrumbs(normPath, normHome string) []PathCrumb {
	homeName := filepath.Base(normHome)
	if homeName == "" || homeName == "/" || homeName == "." {
		homeName = "个人"
	}
	crumbs := []PathCrumb{
		{Name: homeName, FullPath: normHome, IsHome: true},
	}
	if normPath == normHome {
		crumbs[0].IsLast = true
		return crumbs
	}

	rel := strings.TrimPrefix(normPath, normHome+"/")
	parts := strings.Split(rel, "/")
	current := normHome
	for idx, part := range parts {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		crumbs = append(crumbs, PathCrumb{
			Name:     part,
			FullPath: current,
			IsLast:   idx == len(parts)-1,
		})
	}
	return crumbs
}

func buildRootCrumbs(normPath string) []PathCrumb {
	parts := strings.Split(strings.TrimPrefix(normPath, "/"), "/")
	crumbs := make([]PathCrumb, 0, len(parts)+1)
	crumbs = append(crumbs, PathCrumb{
		Name:     "Macintosh HD",
		FullPath: "/",
	})
	current := ""
	for idx, part := range parts {
		if part == "" {
			continue
		}
		current = current + "/" + part
		crumbs = append(crumbs, PathCrumb{
			Name:     part,
			FullPath: current,
			IsLast:   idx == len(parts)-1,
		})
	}
	if len(crumbs) > 0 {
		crumbs[len(crumbs)-1].IsLast = true
	}
	return crumbs
}

// DisambiguateTabs generates subtitle disambiguations for tabs sharing identical leaf folder names.
func DisambiguateTabs(tabs []TabSummary) map[string]string {
	result := make(map[string]string, len(tabs))
	if len(tabs) == 0 {
		return result
	}

	leafGroups := make(map[string][]TabSummary)
	for _, t := range tabs {
		leaf := filepath.Base(NormalizePath(t.Path))
		leafGroups[leaf] = append(leafGroups[leaf], t)
	}

	for leaf, group := range leafGroups {
		if len(group) == 1 {
			result[group[0].ID] = leaf
			continue
		}

		for _, t := range group {
			parent := filepath.Base(filepath.Dir(NormalizePath(t.Path)))
			if parent == "/" || parent == "." || parent == "" {
				result[t.ID] = leaf
			} else {
				result[t.ID] = fmt.Sprintf("%s — %s", leaf, parent)
			}
		}
	}

	return result
}
