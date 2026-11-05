package domain

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// PDFMode defines the PDF transformation operation.
type PDFMode string

const (
	PDFModeExtract      PDFMode = "extract"
	PDFModeSplit        PDFMode = "split"
	PDFModeExportImages PDFMode = "images"
	PDFModeMerge        PDFMode = "merge"
)

// PDFToolsOptions holds configuration for a PDF micro-tool operation.
type PDFToolsOptions struct {
	Mode        PDFMode
	SourceFiles []string
	PageRanges  string
	ImageFormat string // "png" or "jpeg"
	ImageDPI    int    // 150 or 300
	OutputName  string
	OutputDir   string
}

// ParsePageRanges parses expressions like "1-3, 5, 8-10" into a 1-based, deduplicated list of page numbers.
// If expr is empty or "all", returns all pages from 1 to pageCount.
func ParsePageRanges(expr string, pageCount int) ([]int, error) {
	if pageCount <= 0 {
		return nil, fmt.Errorf("pageCount must be greater than 0")
	}

	trimmed := strings.TrimSpace(expr)
	if trimmed == "" || strings.EqualFold(trimmed, "all") {
		pages := make([]int, pageCount)
		for i := range pages {
			pages[i] = i + 1
		}
		return pages, nil
	}

	seen := make(map[int]bool)
	var result []int
	for _, seg := range strings.Split(trimmed, ",") {
		if err := appendPageSegment(seg, pageCount, seen, &result); err != nil {
			return nil, err
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid pages specified")
	}
	sort.Ints(result)
	return result, nil
}

// appendPageSegment parses one comma-separated segment (single page or range).
func appendPageSegment(seg string, pageCount int, seen map[int]bool, result *[]int) error {
	s := strings.TrimSpace(seg)
	if s == "" {
		return nil
	}
	if strings.Contains(s, "-") {
		return appendPageRange(s, pageCount, seen, result)
	}
	p, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("invalid page number %q: %w", s, err)
	}
	return appendPage(p, pageCount, seen, result)
}

// appendPageRange parses "a-b" (bounds may be swapped) and expands it.
func appendPageRange(s string, pageCount int, seen map[int]bool, result *[]int) error {
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return fmt.Errorf("invalid range format: %q", s)
	}
	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return fmt.Errorf("invalid range start %q: %w", parts[0], err)
	}
	end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return fmt.Errorf("invalid range end %q: %w", parts[1], err)
	}
	if start > end {
		start, end = end, start
	}
	for p := start; p <= end; p++ {
		if err := appendPage(p, pageCount, seen, result); err != nil {
			return err
		}
	}
	return nil
}

// appendPage validates bounds and appends the page unless already present.
func appendPage(p, pageCount int, seen map[int]bool, result *[]int) error {
	if p < 1 || p > pageCount {
		return fmt.Errorf("page %d out of bounds (1..%d)", p, pageCount)
	}
	if !seen[p] {
		seen[p] = true
		*result = append(*result, p)
	}
	return nil
}
