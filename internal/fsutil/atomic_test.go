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

func TestWriteAtomicRoot(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatalf("OpenRoot failed: %v", err)
	}
	defer func() { _ = root.Close() }()

	content := []byte("root confined content\n")
	if err := WriteAtomicRoot(root, "file.txt", content, 0o644); err != nil {
		t.Fatalf("WriteAtomicRoot root-level failed: %v", err)
	}

	got, err := root.ReadFile("file.txt")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("content = %q, want %q", got, content)
	}

	// Nested dir creation and write
	nestedContent := []byte("nested confined content\n")
	if err := WriteAtomicRoot(root, "a/b/c/nested.txt", nestedContent, 0o644); err != nil {
		t.Fatalf("WriteAtomicRoot nested failed: %v", err)
	}

	gotNested, err := root.ReadFile("a/b/c/nested.txt")
	if err != nil {
		t.Fatalf("ReadFile nested failed: %v", err)
	}
	if !bytes.Equal(gotNested, nestedContent) {
		t.Fatalf("content = %q, want %q", gotNested, nestedContent)
	}

	// Replace existing
	updated := []byte("updated content\n")
	if err := WriteAtomicRoot(root, "a/b/c/nested.txt", updated, 0o644); err != nil {
		t.Fatalf("WriteAtomicRoot replace failed: %v", err)
	}
	gotUpdated, err := root.ReadFile("a/b/c/nested.txt")
	if err != nil {
		t.Fatalf("ReadFile updated failed: %v", err)
	}
	if !bytes.Equal(gotUpdated, updated) {
		t.Fatalf("content = %q, want %q", gotUpdated, updated)
	}
}

func TestWriteAtomicRootEscape(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatalf("OpenRoot failed: %v", err)
	}
	defer func() { _ = root.Close() }()

	escapes := []string{
		"../escape.txt",
		"../../escape.txt",
		"sub/../../escape.txt",
		"/abs.txt",
		"",
		".",
	}

	for _, path := range escapes {
		t.Run(path, func(t *testing.T) {
			err := WriteAtomicRoot(root, path, []byte("danger"), 0o644)
			if err == nil {
				t.Fatalf("expected error for escaping path %q, got nil", path)
			}
		})
	}

	// Verify no files were created outside dir
	parentEntries, readErr := os.ReadDir(filepath.Dir(dir))
	if readErr == nil {
		for _, e := range parentEntries {
			if e.Name() == "escape.txt" {
				t.Fatalf("found escaped file %s in parent directory", e.Name())
			}
		}
	}
}

func TestWriteAtomicRootSymlinkEscape(t *testing.T) {
	outsideDir := t.TempDir()
	projectDir := t.TempDir()

	// Create symlink inside projectDir pointing to outsideDir
	symlinkPath := filepath.Join(projectDir, "outside_link")
	if err := os.Symlink(outsideDir, symlinkPath); err != nil {
		t.Skipf("symlink creation not supported or permitted: %v", err)
	}

	root, err := os.OpenRoot(projectDir)
	if err != nil {
		t.Fatalf("OpenRoot failed: %v", err)
	}
	defer func() { _ = root.Close() }()

	// Attempting to write into outside_link/evil.txt must fail due to os.Root symlink confinement
	err = WriteAtomicRoot(root, "outside_link/evil.txt", []byte("evil"), 0o644)
	if err == nil {
		t.Fatal("expected error writing through symlink pointing outside root, got nil")
	}

	// Verify outside directory remains untouched
	targetFile := filepath.Join(outsideDir, "evil.txt")
	if _, err := os.Stat(targetFile); !os.IsNotExist(err) {
		t.Fatalf("file %s was written outside root!", targetFile)
	}
}

func TestWriteAtomicRootReservedNames(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatalf("OpenRoot failed: %v", err)
	}
	defer func() { _ = root.Close() }()

	reserved := []string{
		"nul",
		"NUL.txt",
		"con",
		"CON.md",
		"prn",
		"PRN.log",
		"aux",
		"AUX.json",
		"com1",
		"COM1.txt",
		"com9.bin",
		"lpt1",
		"LPT3.txt",
		"sub/nested/nul.txt",
		"a/b/COM2",
	}

	for _, name := range reserved {
		t.Run(name, func(t *testing.T) {
			err := WriteAtomicRoot(root, name, []byte("data"), 0o644)
			if err == nil {
				t.Fatalf("expected error for reserved name %q, got nil", name)
			}
		})
	}
}

func TestIsReservedDeviceName(t *testing.T) {
	tests := []struct {
		name     string
		reserved bool
	}{
		{"nul", true},
		{"NUL", true},
		{"nul.txt", true},
		{"Nul.tar.gz", true},
		{"con", true},
		{"CON.md", true},
		{"aux", true},
		{"aux.go", true},
		{"prn", true},
		{"PRN.doc", true},
		{"com0", true},
		{"com1", true},
		{"COM9.log", true},
		{"lpt0", true},
		{"lpt1", true},
		{"LPT8.dat", true},
		{"valid.txt", false},
		{"constants.go", false},
		{"controller.ts", false},
		{"auxiliary.py", false},
		{"printer.java", false},
		{"com10.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsReservedDeviceName(tt.name)
			if got != tt.reserved {
				t.Errorf("IsReservedDeviceName(%q) = %v, want %v", tt.name, got, tt.reserved)
			}
		})
	}
}
