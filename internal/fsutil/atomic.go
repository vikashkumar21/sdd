// Package fsutil holds small file-system helpers shared by sdd packages.
package fsutil

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// WriteAtomic writes data to a temporary file in the same directory and then
// renames it into place, so a crash never leaves a half-written file. Parent
// directories are created as needed.
func WriteAtomic(path string, data []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".sdd-tmp-*")
	if err != nil {
		return fmt.Errorf("temp file in %s: %w", dir, err)
	}
	name := tmp.Name()
	cleanup := func() { _ = os.Remove(name) }
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close %s: %w", path, err)
	}
	if err := os.Chmod(name, perm); err != nil {
		cleanup()
		return fmt.Errorf("chmod %s: %w", path, err)
	}
	if err := os.Rename(name, path); err != nil {
		cleanup()
		return fmt.Errorf("rename to %s: %w", path, err)
	}
	return nil
}
