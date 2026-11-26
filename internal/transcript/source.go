package transcript

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var uuidLowerRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ValidateConversationID checks that id is a strict lowercase UUID (8-4-4-4-12 hex).
func ValidateConversationID(id string) error {
	if !uuidLowerRe.MatchString(id) {
		return fmt.Errorf("invalid conversation id %q: must be a lowercase UUID", id)
	}
	return nil
}

// DefaultRoot returns the default Antigravity transcript root for home.
func DefaultRoot(home string) (string, error) {
	if strings.TrimSpace(home) == "" {
		return "", errors.New("home directory is empty")
	}
	return filepath.Join(home, ".gemini", "antigravity", "brain"), nil
}

// OpenConversation opens <rootDir>/<id>/.system_generated/logs/transcript.jsonl
// using os.OpenRoot confinement, rejecting invalid IDs, symlinks, and directories.
func OpenConversation(rootDir, id string) (io.ReadCloser, error) {
	if err := ValidateConversationID(id); err != nil {
		return nil, err
	}
	if strings.TrimSpace(rootDir) == "" {
		return nil, errors.New("transcript root directory is empty")
	}

	root, err := os.OpenRoot(rootDir)
	if err != nil {
		return nil, fmt.Errorf("open transcript root: %w", err)
	}
	defer func() { _ = root.Close() }()

	rel := filepath.Join(id, ".system_generated", "logs", "transcript.jsonl")
	f, err := root.Open(rel)
	if err != nil {
		return nil, fmt.Errorf("open conversation transcript: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("stat conversation transcript: %w", err)
	}
	if !info.Mode().IsRegular() {
		_ = f.Close()
		return nil, fmt.Errorf("conversation transcript is not a regular file")
	}
	return f, nil
}

// OpenFile opens an explicit transcript file path, rejecting symlinks and non-regular files
// and verifying os.SameFile between Lstat and the opened file descriptor.
func OpenFile(path string) (io.ReadCloser, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("transcript path is empty")
	}
	linfo, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("stat transcript file: %w", err)
	}
	if !linfo.Mode().IsRegular() {
		return nil, fmt.Errorf("transcript path is not a regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open transcript file: %w", err)
	}
	finfo, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("stat opened transcript file: %w", err)
	}
	if !finfo.Mode().IsRegular() || !os.SameFile(linfo, finfo) {
		_ = f.Close()
		return nil, fmt.Errorf("transcript file changed or is not a regular file")
	}
	return f, nil
}

// SourceKeyForPath returns a path-free deterministic key ("file:" + 16 hex of SHA-256).
func SourceKeyForPath(absPath string) string {
	cleaned := filepath.Clean(absPath)
	sum := sha256.Sum256([]byte(cleaned))
	return "file:" + hex.EncodeToString(sum[:8])
}
