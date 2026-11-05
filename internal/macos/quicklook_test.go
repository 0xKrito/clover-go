package macos

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ebitengine/purego/objc"
)

func TestQuickLookDataSourceAndClasses(t *testing.T) {
	initQuickLookFramework()

	if clsNSString == 0 {
		t.Fatalf("NSString class not available")
	}
	if clsNSURL == 0 {
		t.Fatalf("NSURL class not available")
	}
	if qlDataSource == 0 {
		t.Fatalf("CloverQLDataSource not initialized")
	}

	qlMu.Lock()
	qlCurrent = "/tmp/test_item.txt"
	qlMu.Unlock()

	num := objc.Send[int](qlDataSource, objc.RegisterName("numberOfPreviewItemsInPreviewPanel:"), objc.ID(0))
	if num != 1 {
		t.Errorf("expected 1 preview item, got %d", num)
	}

	urlID := qlDataSource.Send(objc.RegisterName("previewPanel:previewItemAtIndex:"), objc.ID(0), 0)
	if urlID == 0 {
		t.Errorf("expected non-nil preview item URL, got 0")
	}

	qlMu.Lock()
	qlCurrent = ""
	qlMu.Unlock()

	num = objc.Send[int](qlDataSource, objc.RegisterName("numberOfPreviewItemsInPreviewPanel:"), objc.ID(0))
	if num != 0 {
		t.Errorf("expected 0 preview items when path is empty, got %d", num)
	}
}

func TestGenerateNativeThumbnail(t *testing.T) {
	// 1. Test ICNS / image file thumbnailing
	icnsPath := filepath.Join("..", "..", "assets", "AppIcon.icns")
	if _, err := os.Stat(icnsPath); os.IsNotExist(err) {
		icnsPath = filepath.Join("..", "..", "build", "AppIcon.icns")
	}
	if _, err := os.Stat(icnsPath); err == nil {
		data, err := GenerateNativeThumbnail(icnsPath, 128)
		if err != nil {
			t.Fatalf("GenerateNativeThumbnail on icns failed: %v", err)
		}
		if len(data) == 0 {
			t.Fatalf("expected non-empty png bytes")
		}
	}

	// 2. Test PDF thumbnailing
	pdfPath := filepath.Join(t.TempDir(), "test_thumb.pdf")
	_, err := runHelper("make-test-pdf", pdfPath, "1")
	if err == nil {
		data, err := GenerateNativeThumbnail(pdfPath, 128)
		if err != nil {
			t.Fatalf("GenerateNativeThumbnail on pdf failed: %v", err)
		}
		if len(data) == 0 {
			t.Fatalf("expected non-empty png bytes for pdf")
		}
	}
}

func TestResolveQuickLookPath_Markdown(t *testing.T) {
	tempDir := t.TempDir()
	mdFile := filepath.Join(tempDir, "test.md")
	txtFile := filepath.Join(tempDir, "test.txt")

	_ = os.WriteFile(mdFile, []byte("# Hello Markdown\n- item 1\n- item 2\n"), 0644)
	_ = os.WriteFile(txtFile, []byte("plain text"), 0644)

	resTxt := resolveQuickLookPath(txtFile)
	if resTxt != txtFile {
		t.Errorf("expected plain text path to be unchanged, got %q", resTxt)
	}

	resMD := resolveQuickLookPath(mdFile)
	if resMD == mdFile {
		t.Errorf("expected markdown path to resolve to HTML, got %q", resMD)
	}
	if filepath.Ext(resMD) != ".html" {
		t.Errorf("expected resolved path extension to be .html, got %q", resMD)
	}
	data, err := os.ReadFile(resMD)
	if err != nil {
		t.Fatalf("failed to read resolved HTML file: %v", err)
	}
	htmlStr := string(data)
	if !strings.Contains(htmlStr, "Hello Markdown") || !strings.Contains(htmlStr, "<h1>") {
		t.Errorf("expected HTML to contain rendered markdown elements, got: %s", htmlStr)
	}
}

func TestAppInstalledCheck(t *testing.T) {
	if !isAppInstalled("Terminal") {
		t.Errorf("expected Terminal to be installed on macOS")
	}
	if isAppInstalled("DefinitivelyNonExistentApp_987654321") {
		t.Errorf("expected nonexistent app to return false")
	}
}
