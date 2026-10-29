package fsutil

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteAtomicCreate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "newfile.txt")
	content := []byte("hello atomic world\n")

	if err := WriteAtomic(path, content, 0o644); err != nil {
		t.Fatalf("WriteAtomic failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("content = %q, want %q", got, content)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Mode().Perm()&0o600 != 0o600 {
		t.Errorf("expected readable/writable permissions, got %v", info.Mode().Perm())
	}
}

func TestWriteAtomicReplace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "replace.txt")

	if err := WriteAtomic(path, []byte("first content"), 0o644); err != nil {
		t.Fatalf("WriteAtomic first write failed: %v", err)
	}

	second := []byte("second updated content")
	if err := WriteAtomic(path, second, 0o644); err != nil {
		t.Fatalf("WriteAtomic replace failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !bytes.Equal(got, second) {
		t.Fatalf("content = %q, want %q", got, second)
	}
}

func TestWriteAtomicNestedDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "deep", "nested", "file.txt")
	content := []byte("deeply nested content")

	if err := WriteAtomic(path, content, 0o644); err != nil {
		t.Fatalf("WriteAtomic nested dir failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("content = %q, want %q", got, content)
	}
}

func TestWriteAtomicReadOnlyDir(t *testing.T) {
	dir := t.TempDir()
	roDir := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(roDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(roDir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(roDir, 0o755) }()

	path := filepath.Join(roDir, "fail.txt")
	err := WriteAtomic(path, []byte("should fail"), 0o644)
	if err == nil {
		t.Fatal("expected error writing into read-only directory, got nil")
	}

	// Verify no temporary files remain
	entries, readErr := os.ReadDir(roDir)
	if readErr == nil {
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".sdd-tmp-") {
				t.Errorf("found lingering temporary file: %s", e.Name())
			}
		}
	}
}

func TestWriteAtomicRenameToDir(t *testing.T) {
	dir := t.TempDir()
	existingDir := filepath.Join(dir, "targetDir")
	if err := os.Mkdir(existingDir, 0o755); err != nil {
		t.Fatal(err)
	}

	err := WriteAtomic(existingDir, []byte("data"), 0o644)
	if err == nil {
		t.Fatal("expected error renaming file onto existing directory, got nil")
	}

	// Verify temporary file was cleaned up
	entries, readErr := os.ReadDir(dir)
	if readErr == nil {
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".sdd-tmp-") {
				t.Errorf("found lingering temporary file: %s", e.Name())
			}
		}
	}
}

func TestWriteAtomicInvalidPath(t *testing.T) {
	dir := t.TempDir()
	existingFile := filepath.Join(dir, "existing-file")
	if err := os.WriteFile(existingFile, []byte("i am a file"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Attempt to create a file inside a path where a component is a regular file
	badPath := filepath.Join(existingFile, "subfile.txt")
	err := WriteAtomic(badPath, []byte("data"), 0o644)
	if err == nil {
		t.Fatal("expected error creating file under regular file path, got nil")
	}
}
