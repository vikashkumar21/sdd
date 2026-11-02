// Package suppress manages finding waivers and suppressions in .sdd/suppressions.json.
// Suppressed findings are downgraded to informational notes in sdd check, surfaced
// with [SUPPRESSED] indicators in sdd check --fix, and can be interactively waived or pruned.
package suppress

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/vikashkumar21/sdd/internal/fsutil"
)

const (
	// CurrentSchema is the schema version for .sdd/suppressions.json.
	CurrentSchema = 1

	// SuppressionsFile is the relative path to the suppressions register.
	SuppressionsFile = ".sdd/suppressions.json"
)

// Entry defines a single waived finding.
type Entry struct {
	File   string `json:"file"`
	Rule   string `json:"rule"`
	Reason string `json:"reason"`
	Date   string `json:"date"`
}

// Manifest represents the structure of .sdd/suppressions.json.
type Manifest struct {
	Schema       int     `json:"schema"`
	Suppressions []Entry `json:"suppressions"`
}

// NormalizePath returns a clean, slash-separated relative path for matching.
// Leading slashes and parent traversals (../) are stripped to enforce local confinement.
func NormalizePath(p string) string {
	cleaned := filepath.Clean(filepath.ToSlash(strings.TrimSpace(p)))
	cleaned = strings.TrimPrefix(cleaned, "./")
	for strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") {
		cleaned = strings.TrimPrefix(cleaned, "../")
		cleaned = strings.TrimPrefix(cleaned, "/")
	}
	return cleaned
}

// Matches reports whether an entry applies to the given file and rule.
// A wildcard ("*" or "") for file or rule matches any file or rule.
func (e Entry) Matches(file, rule string) bool {
	if e.File != "" && e.File != "*" && NormalizePath(e.File) != NormalizePath(file) {
		return false
	}
	if e.Rule == "" || e.Rule == "*" || strings.EqualFold(e.Rule, rule) {
		return true
	}
	return false
}

// Load reads and parses .sdd/suppressions.json from root using os.OpenRoot confinement.
// If the file does not exist, it returns an empty Manifest, exists=false, and nil error.
func Load(root string) (Manifest, bool, error) {
	empty := Manifest{Schema: CurrentSchema, Suppressions: []Entry{}}

	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return empty, false, fmt.Errorf("suppress: open root %s: %w", root, err)
	}
	defer func() { _ = rootHandle.Close() }()

	f, err := rootHandle.Open(filepath.FromSlash(SuppressionsFile))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || os.IsNotExist(err) {
			return empty, false, nil
		}
		return empty, false, fmt.Errorf("suppress: open %s: %w", SuppressionsFile, err)
	}
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(f)
	if err != nil {
		return empty, true, fmt.Errorf("suppress: read %s: %w", SuppressionsFile, err)
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return empty, true, nil
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return empty, true, fmt.Errorf("suppress: parse %s: %w", SuppressionsFile, err)
	}

	if m.Schema != CurrentSchema {
		return empty, true, fmt.Errorf("suppress: unsupported schema version %d (expected %d)", m.Schema, CurrentSchema)
	}
	if m.Suppressions == nil {
		m.Suppressions = []Entry{}
	}

	return m, true, nil
}

// Save writes m to .sdd/suppressions.json beneath root using fsutil.WriteAtomicRoot.
func Save(root string, m Manifest) error {
	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return fmt.Errorf("suppress: open root %s: %w", root, err)
	}
	defer func() { _ = rootHandle.Close() }()

	m.Schema = CurrentSchema
	if m.Suppressions == nil {
		m.Suppressions = []Entry{}
	}

	// Deterministic sorting: by file, then rule, then date
	sort.Slice(m.Suppressions, func(i, j int) bool {
		fi := NormalizePath(m.Suppressions[i].File)
		fj := NormalizePath(m.Suppressions[j].File)
		if fi != fj {
			return fi < fj
		}
		if m.Suppressions[i].Rule != m.Suppressions[j].Rule {
			return m.Suppressions[i].Rule < m.Suppressions[j].Rule
		}
		return m.Suppressions[i].Date < m.Suppressions[j].Date
	})

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("suppress: serialize: %w", err)
	}
	data = append(data, '\n')

	if err := fsutil.WriteAtomicRoot(rootHandle, filepath.FromSlash(SuppressionsFile), data, 0o644); err != nil {
		return fmt.Errorf("suppress: write %s: %w", SuppressionsFile, err)
	}
	return nil
}

// Add appends or updates an entry in .sdd/suppressions.json.
func Add(root string, entry Entry) error {
	normFile := NormalizePath(entry.File)
	if normFile == "" || normFile == "." {
		return fmt.Errorf("suppress: file cannot be empty")
	}
	if err := fsutil.ValidateLocalRelPath(normFile); err != nil {
		return fmt.Errorf("suppress: invalid file path %q: %w", entry.File, err)
	}

	m, _, err := Load(root)
	if err != nil {
		return err
	}

	entry.File = normFile
	updated := false
	for i, existing := range m.Suppressions {
		if NormalizePath(existing.File) == entry.File && strings.EqualFold(existing.Rule, entry.Rule) {
			m.Suppressions[i] = entry
			updated = true
			break
		}
	}
	if !updated {
		m.Suppressions = append(m.Suppressions, entry)
	}

	return Save(root, m)
}

// RemoveFileRules removes suppressions matching the specified file and rule from .sdd/suppressions.json.
func RemoveFileRules(root string, file, rule string) error {
	m, exists, err := Load(root)
	if err != nil {
		return err
	}
	if !exists || len(m.Suppressions) == 0 {
		return nil
	}

	normFile := NormalizePath(file)
	filtered := make([]Entry, 0, len(m.Suppressions))
	for _, e := range m.Suppressions {
		if NormalizePath(e.File) == normFile && (rule == "" || strings.EqualFold(e.Rule, rule)) {
			continue
		}
		filtered = append(filtered, e)
	}

	m.Suppressions = filtered
	return Save(root, m)
}

// RemoveEntries removes specific entries from .sdd/suppressions.json.
func RemoveEntries(root string, entries []Entry) error {
	if len(entries) == 0 {
		return nil
	}
	m, exists, err := Load(root)
	if err != nil {
		return err
	}
	if !exists || len(m.Suppressions) == 0 {
		return nil
	}

	toRemove := make(map[string]bool)
	for _, e := range entries {
		key := NormalizePath(e.File) + "::" + strings.ToLower(e.Rule)
		toRemove[key] = true
	}

	filtered := make([]Entry, 0, len(m.Suppressions))
	for _, e := range m.Suppressions {
		key := NormalizePath(e.File) + "::" + strings.ToLower(e.Rule)
		if !toRemove[key] {
			filtered = append(filtered, e)
		}
	}

	m.Suppressions = filtered
	return Save(root, m)
}

// Matcher tracks active suppressions and records which ones are matched
// in order to identify stale waivers whose findings no longer trigger.
type Matcher struct {
	mu       sync.Mutex
	manifest Manifest
	used     []bool
}

// NewMatcher creates a Matcher for manifest m.
func NewMatcher(m Manifest) *Matcher {
	return &Matcher{
		manifest: m,
		used:     make([]bool, len(m.Suppressions)),
	}
}

// Match checks if any suppression matches file and rule, marking it used if found.
func (m *Matcher) Match(file, rule string) (Entry, bool) {
	if m == nil {
		return Entry{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, entry := range m.manifest.Suppressions {
		if entry.Matches(file, rule) {
			m.used[i] = true
			return entry, true
		}
	}
	return Entry{}, false
}

// Stale returns all suppression entries that were never matched.
func (m *Matcher) Stale() []Entry {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var stale []Entry
	for i, u := range m.used {
		if !u {
			stale = append(stale, m.manifest.Suppressions[i])
		}
	}
	return stale
}
