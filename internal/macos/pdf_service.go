package macos

import (
	"fmt"
	"strconv"
	"strings"
)

// PDF 微工具全部通过预编译 helper（PDFKit）执行：argv 传参，stdout 返回结果。

// GetPDFPageCount returns the total number of pages in the given PDF file.
func GetPDFPageCount(filePath string) (int, error) {
	out, err := runHelper("pdf-pagecount", filePath)
	if err != nil {
		return 0, fmt.Errorf("read pdf page count: %w", err)
	}
	count, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return 0, fmt.Errorf("parse page count %q: %w", out, err)
	}
	return count, nil
}

// ExtractPDFPages extracts specific 1-based page indices from srcPath to dstPath.
func ExtractPDFPages(srcPath, dstPath string, pages []int) error {
	if len(pages) == 0 {
		return fmt.Errorf("no pages specified for extraction")
	}
	_, err := runHelper("pdf-extract", srcPath, dstPath, intSliceToCSV(pages))
	return err
}

// SplitPDF splits every page of srcPath into a separate PDF file inside outDir.
// Per-page write failures make the helper exit non-zero (no silent missing pages).
func SplitPDF(srcPath, outDir string) error {
	_, err := runHelper("pdf-split", srcPath, outDir)
	return err
}

// ExportPDFPagesToImages renders the specified pages as PNG or JPEG images into outDir.
func ExportPDFPagesToImages(srcPath, outDir, format string, dpi int, pages []int) error {
	if dpi <= 0 {
		dpi = 150
	}
	if len(pages) == 0 {
		return fmt.Errorf("no pages specified for export")
	}
	_, err := runHelper("pdf-export-images", srcPath, outDir, normalizeImageFormat(format), strconv.Itoa(dpi), intSliceToCSV(pages))
	return err
}

// MergePDFs merges multiple PDF files in order into a single dstPath PDF.
func MergePDFs(srcPaths []string, dstPath string) error {
	if len(srcPaths) == 0 {
		return fmt.Errorf("no pdfs to merge")
	}
	argv := append([]string{"pdf-merge", dstPath}, srcPaths...)
	_, err := runHelper(argv[0], argv[1:]...)
	return err
}

func intSliceToCSV(nums []int) string {
	parts := make([]string, len(nums))
	for i, n := range nums {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ",")
}

func normalizeImageFormat(format string) string {
	if strings.EqualFold(format, "jpg") || strings.EqualFold(format, "jpeg") {
		return "jpg"
	}
	return "png"
}
