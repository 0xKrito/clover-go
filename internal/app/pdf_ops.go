package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bighaidao/clover-go/internal/domain"
	"github.com/bighaidao/clover-go/internal/fs"
	"github.com/bighaidao/clover-go/internal/macos"
)

// OpenPDFTools opens the PDF tool modal for the items selected in the specified pane.
func (c *Coordinator) OpenPDFTools(pane *PaneState, initialMode domain.PDFMode) {
	if pane == nil {
		return
	}
	pdfPaths := selectedPDFPaths(c.SelectedItems(pane))
	if len(pdfPaths) == 0 {
		return
	}

	c.PDFTargetPane = pane
	c.PDFError = ""
	c.PDFIsProcessing = false
	c.PDFOptions = buildPDFOptions(pdfPaths, pane.Path, initialMode)

	// Merge 模式无单文档页数语义，以文件数占位。
	if len(pdfPaths) > 1 {
		c.PDFTotalPages = len(pdfPaths)
	} else {
		c.PDFTotalPages = 1
		go c.asyncQueryPageCount(pdfPaths[0])
	}
	c.ShowPDFModal = true
}

// selectedPDFPaths filters the selection down to PDF files.
func selectedPDFPaths(items []domain.FileInfo) []string {
	var paths []string
	for _, it := range items {
		if strings.EqualFold(filepath.Ext(it.Name), ".pdf") {
			paths = append(paths, it.Path)
		}
	}
	return paths
}

// buildPDFOptions seeds the modal options: multi-selection merges in order,
// single file uses the requested mode with default extraction settings.
func buildPDFOptions(pdfPaths []string, outputDir string, initialMode domain.PDFMode) domain.PDFToolsOptions {
	if len(pdfPaths) > 1 {
		return domain.PDFToolsOptions{
			Mode:        domain.PDFModeMerge,
			SourceFiles: pdfPaths,
			OutputName:  "merged.pdf",
			OutputDir:   outputDir,
		}
	}
	singlePdf := pdfPaths[0]
	base := strings.TrimSuffix(filepath.Base(singlePdf), filepath.Ext(singlePdf))
	return domain.PDFToolsOptions{
		Mode:        initialMode,
		SourceFiles: []string{singlePdf},
		ImageFormat: "png",
		ImageDPI:    150,
		OutputName:  base + "_extracted.pdf",
		OutputDir:   outputDir,
	}
}

// asyncQueryPageCount refreshes PDFTotalPages in the background for UI display.
func (c *Coordinator) asyncQueryPageCount(pdfPath string) {
	count, err := macos.GetPDFPageCount(pdfPath)
	if err != nil || count <= 0 {
		return
	}
	c.runOnMain(func() {
		c.PDFTotalPages = count
	})
}

// refreshPageCount ensures the page count is fresh before parsing ranges:
// the async query in OpenPDFTools may not have returned yet (defaulting to 1).
func refreshPageCount(opts domain.PDFToolsOptions, totalPages *int) error {
	if *totalPages > 1 {
		return nil
	}
	count, err := macos.GetPDFPageCount(opts.SourceFiles[0])
	if err != nil {
		return fmt.Errorf("read pdf page count: %w", err)
	}
	if count <= 0 {
		return fmt.Errorf("pdf has no pages")
	}
	*totalPages = count
	return nil
}

// ClosePDFTools closes the PDF tools modal dialog.
func (c *Coordinator) ClosePDFTools() {
	c.ShowPDFModal = false
	c.PDFError = ""
	c.PDFIsProcessing = false
}

// ExecutePDFTools runs the configured PDF transformation in the background.
func (c *Coordinator) ExecutePDFTools() {
	if c.PDFIsProcessing || len(c.PDFOptions.SourceFiles) == 0 {
		return
	}

	c.PDFIsProcessing = true
	c.PDFError = ""

	opts := c.PDFOptions
	totalPages := c.PDFTotalPages
	targetPane := c.PDFTargetPane

	go func() {
		err := executePDFMode(opts, &totalPages)
		c.runOnMain(func() {
			c.PDFIsProcessing = false
			if err != nil {
				c.PDFError = err.Error()
			} else {
				c.ClosePDFTools()
				if targetPane != nil {
					c.LoadCurrentTab(nil)
				}
			}
		})
	}()
}

// executePDFMode dispatches one PDF transformation; output targets use
// NextAvailableName so existing files are never overwritten.
func executePDFMode(opts domain.PDFToolsOptions, totalPages *int) error {
	switch opts.Mode {
	case domain.PDFModeExtract:
		if err := refreshPageCount(opts, totalPages); err != nil {
			return err
		}
		pages, err := domain.ParsePageRanges(opts.PageRanges, *totalPages)
		if err != nil {
			return err
		}
		dst := fs.NextAvailableName(filepath.Join(opts.OutputDir, opts.OutputName))
		return macos.ExtractPDFPages(opts.SourceFiles[0], dst, pages)
	case domain.PDFModeSplit:
		return macos.SplitPDF(opts.SourceFiles[0], opts.OutputDir)
	case domain.PDFModeExportImages:
		if err := refreshPageCount(opts, totalPages); err != nil {
			return err
		}
		pages, err := domain.ParsePageRanges(opts.PageRanges, *totalPages)
		if err != nil {
			return err
		}
		return macos.ExportPDFPagesToImages(opts.SourceFiles[0], opts.OutputDir, opts.ImageFormat, opts.ImageDPI, pages)
	case domain.PDFModeMerge:
		// 防覆盖：目标已存在时自动追加序号（与批量重命名规范一致）。
		dst := fs.NextAvailableName(filepath.Join(opts.OutputDir, opts.OutputName))
		return macos.MergePDFs(opts.SourceFiles, dst)
	default:
		return fmt.Errorf("unknown pdf mode: %s", opts.Mode)
	}
}
