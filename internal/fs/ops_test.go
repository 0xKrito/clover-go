package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNextAvailableName(t *testing.T) {
	tempDir := t.TempDir()

	file1 := filepath.Join(tempDir, "test.txt")
	_ = os.WriteFile(file1, []byte("data"), 0644)

	// Next name for existing file
	next1 := NextAvailableName(file1)
	expected1 := filepath.Join(tempDir, "test 2.txt")
	if next1 != expected1 {
		t.Errorf("got %s, want %s", next1, expected1)
	}

	// Create test 2.txt and test again
	_ = os.WriteFile(next1, []byte("data 2"), 0644)
	next2 := NextAvailableName(file1)
	expected2 := filepath.Join(tempDir, "test 3.txt")
	if next2 != expected2 {
		t.Errorf("got %s, want %s", next2, expected2)
	}
}

func TestCreateDirAndRename(t *testing.T) {
	tempDir := t.TempDir()

	created, err := CreateDir(tempDir, "new_folder")
	if err != nil {
		t.Fatalf("CreateDir failed: %v", err)
	}
	if filepath.Base(created) != "new_folder" {
		t.Errorf("got %s, want new_folder", filepath.Base(created))
	}

	renamed, err := RenameItem(created, "renamed_folder")
	if err != nil {
		t.Fatalf("RenameItem failed: %v", err)
	}
	if filepath.Base(renamed) != "renamed_folder" {
		t.Errorf("got %s, want renamed_folder", filepath.Base(renamed))
	}
}

func TestCopyAndMoveItem(t *testing.T) {
	tempDir := t.TempDir()

	srcDir := filepath.Join(tempDir, "src")
	dstDir := filepath.Join(tempDir, "dst")
	_ = os.Mkdir(srcDir, 0755)
	_ = os.Mkdir(dstDir, 0755)

	srcFile := filepath.Join(srcDir, "doc.txt")
	_ = os.WriteFile(srcFile, []byte("hello world"), 0644)

	// 1. Copy
	copiedPath, err := CopyItem(srcFile, dstDir, PolicyKeepBoth)
	if err != nil {
		t.Fatalf("CopyItem failed: %v", err)
	}
	if _, err := os.Stat(copiedPath); err != nil {
		t.Fatalf("copied file does not exist: %v", err)
	}

	// 2. Copy again with PolicyKeepBoth (collision resolution)
	copied2, err := CopyItem(srcFile, dstDir, PolicyKeepBoth)
	if err != nil {
		t.Fatalf("CopyItem keep both failed: %v", err)
	}
	if filepath.Base(copied2) != "doc 2.txt" {
		t.Errorf("got %s, want doc 2.txt", filepath.Base(copied2))
	}

	// 3. Move
	moveTargetDir := filepath.Join(tempDir, "moved_box")
	_ = os.Mkdir(moveTargetDir, 0755)
	movedPath, err := MoveItem(srcFile, moveTargetDir, PolicyOverwrite)
	if err != nil {
		t.Fatalf("MoveItem failed: %v", err)
	}
	if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
		t.Errorf("srcFile should have been moved away, still exists")
	}
	if _, err := os.Stat(movedPath); err != nil {
		t.Errorf("moved file does not exist at %s", movedPath)
	}
}

func TestDuplicateItem(t *testing.T) {
	tempDir := t.TempDir()
	f := filepath.Join(tempDir, "orig.txt")
	_ = os.WriteFile(f, []byte("content"), 0644)

	dupe, err := DuplicateItem(f)
	if err != nil {
		t.Fatalf("DuplicateItem failed: %v", err)
	}
	if filepath.Base(dupe) != "orig 2.txt" {
		t.Errorf("got %s, want orig 2.txt", filepath.Base(dupe))
	}
}

func TestValidateTransfer(t *testing.T) {
	tempDir := t.TempDir()
	subA := filepath.Join(tempDir, "folderA")
	subB := filepath.Join(tempDir, "folderB")
	childOfA := filepath.Join(subA, "child")
	_ = os.Mkdir(subA, 0755)
	_ = os.Mkdir(subB, 0755)
	_ = os.Mkdir(childOfA, 0755)

	// 1. Valid transfer
	if err := ValidateTransfer([]string{filepath.Join(subA, "file.txt")}, subB); err != nil {
		t.Errorf("expected valid transfer, got: %v", err)
	}

	// 2. Drop into self
	if err := ValidateTransfer([]string{subA}, subA); err == nil {
		t.Errorf("expected error for drop into self, got nil")
	}

	// 3. Drop parent into its child (recursive disaster)
	if err := ValidateTransfer([]string{subA}, childOfA); err == nil {
		t.Errorf("expected error for parent into child, got nil")
	}

	// 4. Drop into virtual archive path
	vPath := FormatArchiveVirtualPath(filepath.Join(tempDir, "test.zip"), "inner")
	if err := ValidateTransfer([]string{subA}, vPath); err == nil {
		t.Errorf("expected error for dropping into virtual archive, got nil")
	}

	// 5. Test IsSameDirectory
	fileInA := filepath.Join(subA, "doc.txt")
	if !IsSameDirectory(fileInA, subA) {
		t.Errorf("expected true for file directly in folderA")
	}
	if IsSameDirectory(fileInA, subB) {
		t.Errorf("expected false for file in folderA vs folderB")
	}
}

func TestValidateTransfer_SymlinkAliasBlocked(t *testing.T) {
	tempDir := t.TempDir()
	boxA := filepath.Join(tempDir, "a")
	sub := filepath.Join(boxA, "sub")
	_ = os.MkdirAll(sub, 0755)
	alias := filepath.Join(tempDir, "alias")
	if err := os.Symlink(boxA, alias); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	// alias 真实指向 boxA，拖入 boxA/sub：词法前缀检查无法发现，
	// 必须由 EvalSymlinks 真实路径兑底拦截。
	if err := ValidateTransfer([]string{alias}, sub); err == nil {
		t.Errorf("expected error for symlink alias dir into its own subtree, got nil")
	}

	// 无关目录不受兑底影响
	other := filepath.Join(tempDir, "b")
	_ = os.Mkdir(other, 0755)
	if err := ValidateTransfer([]string{other}, sub); err != nil {
		t.Errorf("unrelated dir should pass validation: %v", err)
	}
}

func TestMoveItem_OverwriteFileKeepsTargetOnFailurePath(t *testing.T) {
	tempDir := t.TempDir()
	dstDir := filepath.Join(tempDir, "dst")
	srcRoot := filepath.Join(tempDir, "src")
	_ = os.MkdirAll(dstDir, 0755)
	_ = os.MkdirAll(srcRoot, 0755)

	// 同名文件冲突 + PolicyOverwrite：目标文件应由 rename 原子覆盖，
	// 不得出现“先 RemoveAll 后拷贝”的丢失窗口。
	_ = os.WriteFile(filepath.Join(dstDir, "a.txt"), []byte("old"), 0644)
	_ = os.WriteFile(filepath.Join(srcRoot, "a.txt"), []byte("new"), 0644)

	if _, err := MoveItem(filepath.Join(srcRoot, "a.txt"), dstDir, PolicyOverwrite); err != nil {
		t.Fatalf("MoveItem overwrite failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dstDir, "a.txt"))
	if err != nil {
		t.Fatalf("read overwritten target: %v", err)
	}
	if string(data) != "new" {
		t.Errorf("target content = %q, want %q", data, "new")
	}
	if _, err := os.Stat(filepath.Join(srcRoot, "a.txt")); !os.IsNotExist(err) {
		t.Errorf("source should have been moved away")
	}
}

func TestMoveItem_OverwriteDirectoryReplaced(t *testing.T) {
	tempDir := t.TempDir()
	dstDir := filepath.Join(tempDir, "dst")
	srcRoot := filepath.Join(tempDir, "src")

	// 目录替换仍需先清空目标目录（rename 无法覆盖非空目录）
	_ = os.MkdirAll(filepath.Join(dstDir, "dir"), 0755)
	_ = os.WriteFile(filepath.Join(dstDir, "dir", "stale.txt"), []byte("stale"), 0644)
	_ = os.MkdirAll(filepath.Join(srcRoot, "dir"), 0755)
	_ = os.WriteFile(filepath.Join(srcRoot, "dir", "fresh.txt"), []byte("fresh"), 0644)

	if _, err := MoveItem(filepath.Join(srcRoot, "dir"), dstDir, PolicyOverwrite); err != nil {
		t.Fatalf("MoveItem overwrite dir failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dstDir, "dir", "stale.txt")); !os.IsNotExist(err) {
		t.Errorf("stale content should have been removed")
	}
	if _, err := os.Stat(filepath.Join(dstDir, "dir", "fresh.txt")); err != nil {
		t.Errorf("fresh content missing: %v", err)
	}
}
