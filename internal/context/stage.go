package context

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vikashkumar21/sdd/internal/fsutil"
)

var (
	taskIDRe = regexp.MustCompile(`^T-\d{3}$`)
)

// Item holds an un-staged context item.
type Item struct {
	Source string // "raw-text", "file", "stdin"
	Origin string // source file path, label, or empty
	Name   string // custom slug name, or empty to derive
	Task   string // optional task ID e.g. "T-001"
	Format string // "text", "markdown", "json", "pdf", etc. (empty to infer)
	Data   []byte
}

// Stage writes item into .sdd/context/ beneath root and updates .sdd/context/manifest.json.
// All operations are confined to root via os.OpenRoot.
func Stage(root string, item Item, now time.Time) (Entry, error) {
	if len(item.Data) > MaxInputBytes {
		return Entry{}, fmt.Errorf("context: item size %d bytes exceeds maximum allowed (%d bytes)", len(item.Data), MaxInputBytes)
	}
	if item.Task != "" && !taskIDRe.MatchString(item.Task) {
		return Entry{}, fmt.Errorf("context: invalid task ID %q (expected format T-xxx)", item.Task)
	}

	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return Entry{}, fmt.Errorf("context: open root %s: %w", root, err)
	}
	defer func() { _ = rootHandle.Close() }()

	h := sha256.Sum256(item.Data)
	hashStr := hex.EncodeToString(h[:])

	format := item.Format
	if format == "" {
		format = inferFormat(item.Origin, item.Data)
	}

	ext := extensionForFormat(format, item.Origin)
	slug := sanitizeSlug(item.Name)
	if slug == "" {
		slug = sanitizeSlug(filepath.Base(item.Origin))
	}
	if slug == "" {
		slug = "context"
	}

	timestamp := now.UTC().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s%s", timestamp, slug, ext)

	// If associated with a task, group in subfolder .sdd/context/<task>/
	var relPath string
	if item.Task != "" {
		relPath = filepath.Join(ContextDir, item.Task, filename)
	} else {
		relPath = filepath.Join(ContextDir, filename)
	}

	slashRel := filepath.ToSlash(relPath)
	if err := fsutil.ValidateLocalRelPath(filepath.FromSlash(slashRel)); err != nil {
		return Entry{}, fmt.Errorf("context: unsafe staged path %q: %w", slashRel, err)
	}

	if err := fsutil.WriteAtomicRoot(rootHandle, filepath.FromSlash(slashRel), item.Data, 0o644); err != nil {
		return Entry{}, fmt.Errorf("context: write staged file: %w", err)
	}

	m, ok, err := ReadManifest(root)
	if err != nil && ok {
		return Entry{}, err
	}
	if !ok {
		m = Manifest{
			Schema: 1,
		}
	}
	m.Updated = now.UTC().Format("2006-01-02T15:04:05Z")

	id := fmt.Sprintf("ctx-%03d", len(m.Entries)+1)
	entry := Entry{
		ID:        id,
		Source:    item.Source,
		Origin:    item.Origin,
		Path:      slashRel,
		Task:      item.Task,
		Format:    format,
		Bytes:     int64(len(item.Data)),
		SHA256:    hashStr,
		CreatedAt: now.UTC().Format("2006-01-02T15:04:05Z"),
	}
	m.Entries = append(m.Entries, entry)

	if err := WriteManifest(rootHandle, m); err != nil {
		return Entry{}, err
	}

	return entry, nil
}

func sanitizeSlug(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if ext := filepath.Ext(s); ext != "" {
		s = strings.TrimSuffix(s, ext)
	}
	var b strings.Builder
	lastHyphen := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastHyphen = false
		} else if !lastHyphen && b.Len() > 0 {
			b.WriteByte('-')
			lastHyphen = true
		}
	}
	res := strings.Trim(b.String(), "-")
	return res
}

func inferFormat(origin string, data []byte) string {
	ext := strings.ToLower(filepath.Ext(origin))
	switch ext {
	case ".md", ".markdown":
		return "markdown"
	case ".json":
		return "json"
	case ".pdf":
		return "pdf"
	case ".txt":
		return "text"
	}
	// Content inspection
	if len(data) >= 4 && string(data[:4]) == "%PDF" {
		return "pdf"
	}
	trimmed := strings.TrimSpace(string(data))
	if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
		return "json"
	}
	return "text"
}

func extensionForFormat(format, origin string) string {
	origExt := strings.ToLower(filepath.Ext(origin))
	if origExt != "" {
		return origExt
	}
	switch format {
	case "markdown":
		return ".md"
	case "json":
		return ".json"
	case "pdf":
		return ".pdf"
	default:
		return ".txt"
	}
}
