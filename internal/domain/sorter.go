package domain

import (
	"sort"
	"strings"
	"time"
	"unicode"
)

// SortField defines the attribute to sort items by.
type SortField string

const (
	SortByName       SortField = "name"
	SortByModTime    SortField = "mod_time"
	SortByCreateTime SortField = "create_time"
	SortBySize       SortField = "size"
	SortByKind       SortField = "kind"
)

// SortDirection defines sorting order.
type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

// SortOptions specifies sorting preferences.
type SortOptions struct {
	Field     SortField
	Direction SortDirection
}

// SortFiles sorts a slice of FileInfo adhering to macOS Finder rules.
func SortFiles(items []FileInfo, opts SortOptions) {
	if len(items) <= 1 {
		return
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]

		// Finder rule: when sorting by a time field (ModTime/CreateTime),
		// folders and files are mixed. For other fields (Name, Size, Kind),
		// folders are placed first.
		if opts.Field != SortByModTime && opts.Field != SortByCreateTime {
			if a.IsDir != b.IsDir {
				return a.IsDir // folders first
			}
		}

		var cmp int
		switch opts.Field {
		case SortByModTime:
			cmp = compareModTime(a, b)
		case SortByCreateTime:
			cmp = compareCreateTime(a, b)
		case SortBySize:
			cmp = compareSize(a, b)
		case SortByKind:
			cmp = compareKind(a, b)
		default: // SortByName
			cmp = NaturalCompare(a.Name, b.Name)
		}

		if cmp == 0 {
			cmp = NaturalCompare(a.Name, b.Name)
		}

		if opts.Direction == SortDesc {
			return cmp > 0
		}
		return cmp < 0
	})
}

func compareModTime(a, b FileInfo) int {
	if a.ModTime.Equal(b.ModTime) {
		return 0
	}
	if a.ModTime.Before(b.ModTime) {
		return -1
	}
	return 1
}

// effectiveCreateTime returns the sortable creation time; entries without a
// birthtime (e.g. archive VFS entries) fall back to their modification time.
func effectiveCreateTime(f FileInfo) time.Time {
	if f.CreateTime.IsZero() {
		return f.ModTime
	}
	return f.CreateTime
}

func compareCreateTime(a, b FileInfo) int {
	aTime, bTime := effectiveCreateTime(a), effectiveCreateTime(b)
	if aTime.Equal(bTime) {
		return 0
	}
	if aTime.Before(bTime) {
		return -1
	}
	return 1
}

func compareSize(a, b FileInfo) int {
	if a.Size == b.Size {
		return 0
	}
	if a.Size < b.Size {
		return -1
	}
	return 1
}

func compareKind(a, b FileInfo) int {
	cmp := strings.Compare(strings.ToLower(a.Kind), strings.ToLower(b.Kind))
	if cmp != 0 {
		return cmp
	}
	return NaturalCompare(a.Name, b.Name)
}

// NaturalCompare compares strings naturally, treating embedded numeric sequences as numbers.
func NaturalCompare(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	len1, len2 := len(r1), len(r2)
	i, j := 0, 0

	for i < len1 && j < len2 {
		ch1, ch2 := r1[i], r2[j]

		if unicode.IsDigit(ch1) && unicode.IsDigit(ch2) {
			numStart1, numStart2 := i, j
			for i < len1 && unicode.IsDigit(r1[i]) {
				i++
			}
			for j < len2 && unicode.IsDigit(r2[j]) {
				j++
			}

			n1Str := strings.TrimLeft(string(r1[numStart1:i]), "0")
			n2Str := strings.TrimLeft(string(r2[numStart2:j]), "0")

			if len(n1Str) != len(n2Str) {
				if len(n1Str) < len(n2Str) {
					return -1
				}
				return 1
			}
			if n1Str != n2Str {
				if n1Str < n2Str {
					return -1
				}
				return 1
			}
			continue
		}

		lower1 := unicode.ToLower(ch1)
		lower2 := unicode.ToLower(ch2)
		if lower1 != lower2 {
			if lower1 < lower2 {
				return -1
			}
			return 1
		}
		i++
		j++
	}

	if i < len1 {
		return 1
	}
	if j < len2 {
		return -1
	}
	return 0
}
