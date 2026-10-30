// Package fsutil holds small file-system helpers shared by sdd packages.
package fsutil

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	tmpMu  sync.Mutex
	tmpSeq uint64
)

func nextTempName(dir string) string {
	tmpMu.Lock()
	tmpSeq++
	seq := tmpSeq
	tmpMu.Unlock()
	base := fmt.Sprintf(".sdd-tmp-%d-%d-%d", os.Getpid(), time.Now().UnixNano(), seq)
	if dir == "." || dir == "" {
		return base
	}
	return filepath.Join(dir, base)
}

// IsReservedDeviceName reports whether base is a reserved Windows device name
// (e.g. CON, PRN, AUX, NUL, COM0..9, LPT0..9), with or without an extension.
func IsReservedDeviceName(base string) bool {
	if idx := strings.IndexByte(base, '.'); idx >= 0 {
		base = base[:idx]
	}
	base = strings.ToUpper(base)
	switch base {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	if len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) {
		b := base[3]
		return b >= '0' && b <= '9'
	}
	return false
}

// ValidateLocalRelPath verifies that relPath is a safe relative path beneath root:
// it must be a valid local path (no ".." escaping, not absolute, no drive letters)
// and must not contain any component that is a reserved device name or references the root directory directly.
func ValidateLocalRelPath(relPath string) error {
	if !filepath.IsLocal(relPath) || relPath == "." {
		return fmt.Errorf("unsafe path %q: must be a local relative path beneath root", relPath)
	}
	cleaned := filepath.ToSlash(filepath.Clean(relPath))
	for _, part := range strings.Split(cleaned, "/") {
		if part == "" || part == "." {
			continue
		}
		if IsReservedDeviceName(part) {
			return fmt.Errorf("unsafe path %q: component %q is a reserved device name", relPath, part)
		}
	}
	return nil
}

// WriteAtomicRoot writes data to a temporary file inside dir beneath root, fsyncs it,
// sets the desired permissions, and then renames it into place. All filesystem operations
// are confined to root. Parent directories are created with 0o755 if needed.
func WriteAtomicRoot(root *os.Root, relPath string, data []byte, perm fs.FileMode) error {
	if root == nil {
		return fmt.Errorf("root is nil")
	}
	if err := ValidateLocalRelPath(relPath); err != nil {
		return fmt.Errorf("write atomic: %w", err)
	}

	cleanRel := filepath.Clean(relPath)
	dir := filepath.Dir(cleanRel)
	if dir != "." && dir != "" {
		if err := root.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	var (
		tmp    *os.File
		tmpRel string
		err    error
	)
	for i := 0; i < 10; i++ {
		tmpRel = nextTempName(dir)
		tmp, err = root.OpenFile(tmpRel, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("temp file in %s: %w", dir, err)
		}
	}
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}

	cleanup := func() { _ = root.Remove(tmpRel) }

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write %s: %w", cleanRel, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("sync %s: %w", cleanRel, err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close %s: %w", cleanRel, err)
	}
	if err := root.Chmod(tmpRel, perm); err != nil {
		cleanup()
		return fmt.Errorf("chmod %s: %w", cleanRel, err)
	}
	if err := root.Rename(tmpRel, cleanRel); err != nil {
		cleanup()
		return fmt.Errorf("rename to %s: %w", cleanRel, err)
	}
	return nil
}

// WriteAtomic writes data to a temporary file in the same directory and then
// renames it into place, so a crash never leaves a half-written file.
// Operations are confined to filepath.Dir(path) using os.OpenRoot.
func WriteAtomic(path string, data []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return fmt.Errorf("open root %s: %w", dir, err)
	}
	defer func() { _ = root.Close() }()
	return WriteAtomicRoot(root, filepath.Base(path), data, perm)
}
