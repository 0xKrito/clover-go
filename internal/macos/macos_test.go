package macos

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCloverURL(t *testing.T) {
	// 1. Open action
	act, p, err := ParseCloverURL("fclover://open?path=/Users/test/Documents")
	if err != nil || act != "open" || p != "/Users/test/Documents" {
		t.Errorf("open URL mismatch: act=%s, path=%s, err=%v", act, p, err)
	}

	// 2. Reveal action
	act, p, err = ParseCloverURL("fclover://reveal?path=/Users/test/file.txt")
	if err != nil || act != "reveal" || p != "/Users/test/file.txt" {
		t.Errorf("reveal URL mismatch: act=%s, path=%s, err=%v", act, p, err)
	}

	// 3. Invalid scheme
	_, _, err = ParseCloverURL("http://example.com")
	if err == nil {
		t.Errorf("expected scheme error for http, got nil")
	}
}

func TestFileTags(t *testing.T) {
	tempDir := t.TempDir()
	f := filepath.Join(tempDir, "tagged.txt")
	_ = os.WriteFile(f, []byte("test"), 0644)

	// Set Red Tag
	if err := SetFileTag(f, TagRed); err != nil {
		t.Fatalf("SetFileTag failed: %v", err)
	}

	tags, colorIdx, err := GetFileTags(f)
	if err != nil {
		t.Fatalf("GetFileTags failed: %v", err)
	}
	if len(tags) == 0 || colorIdx != TagRed {
		t.Errorf("expected TagRed, got colorIdx=%d, tags=%v", colorIdx, tags)
	}

	// Remove Tag
	if err := RemoveFileTags(f); err != nil {
		t.Fatalf("RemoveFileTags failed: %v", err)
	}

	tags, colorIdx, _ = GetFileTags(f)
	if colorIdx != TagNone || len(tags) != 0 {
		t.Errorf("expected TagNone after removal, got %d, tags=%v", colorIdx, tags)
	}
}

func TestBplistCodec(t *testing.T) {
	cases := []string{
		"Red\n6",
		"Green\n2",
		"红色\n6",
		"蓝色\n4",
		"Work Project",
	}

	for _, c := range cases {
		enc := EncodeTagBplist([]string{c})
		if len(enc) == 0 {
			t.Fatalf("failed to encode: %s", c)
		}
		dec := DecodeTagBplist(enc)
		if len(dec) == 0 || dec[0] != c {
			t.Errorf("roundtrip failed: want %q, got %v", c, dec)
		}
	}
}
