package macos

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var (
	appKitOnce                sync.Once
	clsNSImage                objc.Class
	clsNSBitmapImageRep       objc.Class
	selAlloc                  objc.SEL
	selInitWithContentsOfFile objc.SEL
	selTIFFRepresentation     objc.SEL
	selImageRepWithData       objc.SEL
	selRepUsingTypeProperties objc.SEL
	selLength                 objc.SEL
	selBytes                  objc.SEL
)

func initAppKitThumb() {
	_, _ = purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_GLOBAL|purego.RTLD_NOW)
	_, _ = purego.Dlopen("/System/Library/Frameworks/PDFKit.framework/PDFKit", purego.RTLD_GLOBAL|purego.RTLD_NOW)

	if clsNSString == 0 {
		clsNSString = objc.GetClass("NSString")
		selStringWithUTF8 = objc.RegisterName("stringWithUTF8String:")
	}
	clsNSImage = objc.GetClass("NSImage")
	clsNSBitmapImageRep = objc.GetClass("NSBitmapImageRep")

	selAlloc = objc.RegisterName("alloc")
	selInitWithContentsOfFile = objc.RegisterName("initWithContentsOfFile:")
	selTIFFRepresentation = objc.RegisterName("TIFFRepresentation")
	selImageRepWithData = objc.RegisterName("imageRepWithData:")
	selRepUsingTypeProperties = objc.RegisterName("representationUsingType:properties:")
	selLength = objc.RegisterName("length")
	selBytes = objc.RegisterName("bytes")
}

// GenerateNativeThumbnail extracts a high-fidelity PNG thumbnail from macOS native formats
// (HEIC/TIFF/RAW/PSD/ICNS/PDF) in-process without spawning external GUI debug processes,
// and delegates video files (MP4/MOV) to the background clover-helper.
func GenerateNativeThumbnail(path string, maxPx int) ([]byte, error) {
	ext := strings.ToLower(filepath.Ext(path))

	// Video formats: use helper to extract frame
	if isVideoExtension(ext) {
		return generateVideoThumbnail(path, maxPx)
	}

	appKitOnce.Do(initAppKitThumb)
	if clsNSImage == 0 || clsNSBitmapImageRep == 0 || clsNSString == 0 {
		return nil, fmt.Errorf("appkit image classes unavailable")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	cstr := append([]byte(absPath), 0)
	nsPath := objc.ID(clsNSString).Send(selStringWithUTF8, unsafe.Pointer(&cstr[0]))
	if nsPath == 0 {
		return nil, fmt.Errorf("failed to allocate NSString for path: %s", path)
	}

	imgAlloc := objc.ID(clsNSImage).Send(selAlloc)
	img := imgAlloc.Send(selInitWithContentsOfFile, nsPath)
	if img == 0 {
		return nil, fmt.Errorf("NSImage failed to load: %s", path)
	}

	tiffData := img.Send(selTIFFRepresentation)
	if tiffData == 0 {
		return nil, fmt.Errorf("TIFFRepresentation returned nil for: %s", path)
	}

	rep := objc.ID(clsNSBitmapImageRep).Send(selImageRepWithData, tiffData)
	if rep == 0 {
		return nil, fmt.Errorf("NSBitmapImageRep failed to parse tiff for: %s", path)
	}

	// 4 = NSPNGFileType
	pngData := rep.Send(selRepUsingTypeProperties, 4, objc.ID(0))
	if pngData == 0 {
		return nil, fmt.Errorf("representationUsingType failed for: %s", path)
	}

	length := objc.Send[int](pngData, selLength)
	bytesPtr := objc.Send[unsafe.Pointer](pngData, selBytes)
	if length <= 0 || bytesPtr == nil {
		return nil, fmt.Errorf("invalid PNG representation: %s", path)
	}

	out := make([]byte, length)
	copy(out, unsafe.Slice((*byte)(bytesPtr), length))
	return out, nil
}

func isVideoExtension(ext string) bool {
	switch ext {
	case ".mp4", ".mov", ".m4v", ".avi", ".mkv":
		return true
	default:
		return false
	}
}

func generateVideoThumbnail(path string, maxPx int) ([]byte, error) {
	if maxPx <= 0 {
		maxPx = 512
	}
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("clover-vthumb-%d-%d.png", time.Now().UnixNano(), os.Getpid()))
	defer os.Remove(tmpFile)

	_, err := runHelper("video-thumb", path, tmpFile, strconv.Itoa(maxPx))
	if err != nil {
		return nil, fmt.Errorf("video-thumb helper: %w", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil || len(data) == 0 {
		return nil, fmt.Errorf("failed to read video thumbnail from %s: %w", tmpFile, err)
	}
	return data, nil
}
