// Package context manages staged context artifacts in .sdd/context/.
package context

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/vikashkumar21/sdd/internal/fsutil"
)

// ManifestPath is the relative path of the context manifest within the project root.
const ManifestPath = ".sdd/context/manifest.json"

// ContextDir is the relative directory for staged context artifacts.
const ContextDir = ".sdd/context"

// MaxInputBytes is the fail-safe size limit for any single staged context artifact (10 MB).
const MaxInputBytes = 10 * 1024 * 1024

// Entry describes one staged context artifact.
type Entry struct {
	ID        string `json:"id"`
	Source    string `json:"source"`         // "raw-text", "file", "stdin"
	Origin    string `json:"origin"`         // original filename or label
	Path      string `json:"path"`           // slash-separated, relative to project root
	Task      string `json:"task,omitempty"` // e.g. "T-001"
	Format    string `json:"format"`         // "text", "markdown", "json", "pdf", etc.
	Bytes     int64  `json:"bytes"`
	SHA256    string `json:"sha256"`
	CreatedAt string `json:"created_at"`
}

// Manifest records all staged context artifacts for the project.
type Manifest struct {
	Schema  int     `json:"schema"`
	Updated string  `json:"updated"`
	Entries []Entry `json:"entries"`
}

// ReadManifest loads .sdd/context/manifest.json if present.
func ReadManifest(root string) (Manifest, bool, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Manifest{}, false, nil
		}
		return Manifest{}, false, fmt.Errorf("context: open root: %w", err)
	}
	defer func() { _ = r.Close() }()

	data, err := r.ReadFile(filepath.FromSlash(ManifestPath))
	if errors.Is(err, fs.ErrNotExist) {
		return Manifest{}, false, nil
	}
	if err != nil {
		return Manifest{}, false, fmt.Errorf("context: read manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, true, fmt.Errorf("context: parse manifest: %w", err)
	}
	return m, true, nil
}

// WriteManifest writes m to .sdd/context/manifest.json atomically and confined to root.
func WriteManifest(root *os.Root, m Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("context: marshal manifest: %w", err)
	}
	return fsutil.WriteAtomicRoot(root, filepath.FromSlash(ManifestPath), append(data, '\n'), 0o644)
}
