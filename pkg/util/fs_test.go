package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcFile := filepath.Join(srcDir, "test.txt")
	content := []byte("hello world")
	// Use 0644 permissions
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	dstFile := filepath.Join(dstDir, "test_copy.txt")

	if err := CopyFile(srcFile, dstFile); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify content
	got, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", string(got), string(content))
	}

	// Verify permissions
	info, err := os.Stat(dstFile)
	if err != nil {
		t.Fatal(err)
	}
	// Check specifically for user read/write (0600 part) as umask might affect group/world
	if info.Mode()&0600 != 0600 {
		t.Errorf("permission mismatch: got %v, expected at least 0600", info.Mode())
	}
}

func TestCopyDir(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create structure:
	// src/
	//   file1.txt
	//   subdir/
	//     file2.txt

	if err := os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("file1"), 0644); err != nil {
		t.Fatal(err)
	}

	subDir := filepath.Join(srcDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("file2"), 0644); err != nil {
		t.Fatal(err)
	}

	targetDir := filepath.Join(dstDir, "target")

	if err := CopyDir(srcDir, targetDir); err != nil {
		t.Fatalf("CopyDir failed: %v", err)
	}

	// Verify file1
	got1, err := os.ReadFile(filepath.Join(targetDir, "file1.txt"))
	if err != nil {
		t.Errorf("file1 not found: %v", err)
	} else if string(got1) != "file1" {
		t.Errorf("file1 content mismatch")
	}

	// Verify file2
	got2, err := os.ReadFile(filepath.Join(targetDir, "subdir", "file2.txt"))
	if err != nil {
		t.Errorf("file2 not found: %v", err)
	} else if string(got2) != "file2" {
		t.Errorf("file2 content mismatch")
	}
}
