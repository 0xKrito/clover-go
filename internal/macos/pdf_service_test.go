package macos

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPDFService(t *testing.T) {
	// 1. Create a dummy test PDF via the helper itself
	tempDir := t.TempDir()
	testPdf := filepath.Join(tempDir, "sample.pdf")

	if _, err := runHelper("make-test-pdf", testPdf, "3"); err != nil {
		t.Fatalf("failed to setup test PDF: %v", err)
	}

	// 2. Test GetPDFPageCount
	count, err := GetPDFPageCount(testPdf)
	if err != nil {
		t.Fatalf("GetPDFPageCount failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 pages, got %d", count)
	}

	// 3. Test ExtractPDFPages
	extractedPdf := filepath.Join(tempDir, "extracted.pdf")
	if err := ExtractPDFPages(testPdf, extractedPdf, []int{1, 3}); err != nil {
		t.Fatalf("ExtractPDFPages failed: %v", err)
	}
	extCount, err := GetPDFPageCount(extractedPdf)
	if err != nil || extCount != 2 {
		t.Errorf("expected 2 pages in extracted PDF, got %d (err: %v)", extCount, err)
	}

	// 4. Test SplitPDF
	splitDir := filepath.Join(tempDir, "split_out")
	_ = os.MkdirAll(splitDir, 0755)
	if err := SplitPDF(testPdf, splitDir); err != nil {
		t.Fatalf("SplitPDF failed: %v", err)
	}
	files, _ := os.ReadDir(splitDir)
	if len(files) != 3 {
		t.Errorf("expected 3 split PDF files, got %d", len(files))
	}

	// 5. Test MergePDFs
	mergedPdf := filepath.Join(tempDir, "merged.pdf")
	if err := MergePDFs([]string{extractedPdf, testPdf}, mergedPdf); err != nil {
		t.Fatalf("MergePDFs failed: %v", err)
	}
	mergedCount, err := GetPDFPageCount(mergedPdf)
	if err != nil || mergedCount != 5 {
		t.Errorf("expected 5 pages in merged PDF, got %d (err: %v)", mergedCount, err)
	}
}
