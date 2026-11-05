package domain

import (
	"testing"
)

func TestCalculateBatchRename_Sequential(t *testing.T) {
	items := []FileInfo{
		{Name: "photoA.jpg", Path: "/tmp/photoA.jpg", IsDir: false},
		{Name: "photoB.jpg", Path: "/tmp/photoB.jpg", IsDir: false},
	}

	opts := BatchRenameOptions{
		Mode:      ModeSequential,
		BaseName:  "trip",
		StartNum:  1,
		Padding:   3,
		Separator: "-",
	}

	pairs, err := CalculateBatchRename(items, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(pairs))
	}

	if pairs[0].NewName != "trip-001.jpg" {
		t.Errorf("pair 0 got %s, want trip-001.jpg", pairs[0].NewName)
	}
	if pairs[1].NewName != "trip-002.jpg" {
		t.Errorf("pair 1 got %s, want trip-002.jpg", pairs[1].NewName)
	}
}

func TestCalculateBatchRename_PrefixAndSuffix(t *testing.T) {
	items := []FileInfo{
		{Name: "doc.txt", Path: "/tmp/doc.txt", IsDir: false},
	}

	// Prefix
	pPairs, err := CalculateBatchRename(items, BatchRenameOptions{
		Mode:   ModeAddPrefix,
		Prefix: "draft-",
	})
	if err != nil || pPairs[0].NewName != "draft-doc.txt" {
		t.Errorf("AddPrefix failed: %v, got %s", err, pPairs[0].NewName)
	}

	// Suffix
	sPairs, err := CalculateBatchRename(items, BatchRenameOptions{
		Mode:   ModeAddSuffix,
		Suffix: "-v2",
	})
	if err != nil || sPairs[0].NewName != "doc-v2.txt" {
		t.Errorf("AddSuffix failed: %v, got %s", err, sPairs[0].NewName)
	}
}

func TestCalculateBatchRename_Replace(t *testing.T) {
	items := []FileInfo{
		{Name: "old_doc_old.txt", Path: "/tmp/old_doc_old.txt", IsDir: false},
	}

	pairs, err := CalculateBatchRename(items, BatchRenameOptions{
		Mode:        ModeReplace,
		FindText:    "old",
		ReplaceText: "new",
	})
	if err != nil || pairs[0].NewName != "new_doc_new.txt" {
		t.Errorf("Replace failed: %v, got %s", err, pairs[0].NewName)
	}
}

func TestCalculateBatchRename_Conflicts(t *testing.T) {
	items := []FileInfo{
		{Name: "a.txt", Path: "/tmp/a.txt"},
		{Name: "b.txt", Path: "/tmp/b.txt"},
	}

	// Sequential with same number will conflict if start is same and items produce duplicate
	_, err := CalculateBatchRename(items, BatchRenameOptions{
		Mode:        ModeReplace,
		FindText:    "b",
		ReplaceText: "a",
	})
	if err == nil {
		t.Fatalf("expected conflict error, got nil")
	}

	// Invalid characters
	_, err = CalculateBatchRename(items, BatchRenameOptions{
		Mode:   ModeAddPrefix,
		Prefix: "bad/prefix/",
	})
	if err == nil {
		t.Fatalf("expected invalid character error, got nil")
	}
}
