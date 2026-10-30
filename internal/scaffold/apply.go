package scaffold

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vikashkumar21/sdd/internal/block"
	"github.com/vikashkumar21/sdd/internal/fsutil"
	"github.com/vikashkumar21/sdd/internal/templates"
	"github.com/vikashkumar21/sdd/internal/version"
)

// ManifestPath is where sdd records what it generated, relative to the root.
const ManifestPath = ".sdd/manifest.json"

// Manifest records what `sdd init` generated, so later tools can tell
// untouched files from edited ones.
type Manifest struct {
	Schema    int           `json:"schema"`
	Tool      ToolInfo      `json:"tool"`
	Templates TemplatesInfo `json:"templates"`
	Generated string        `json:"generated"`
	Answers   Answers       `json:"answers"`
	Files     []FileEntry   `json:"files"`
}

// ToolInfo identifies the sdd build that wrote the manifest.
type ToolInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// TemplatesInfo identifies the template set used.
type TemplatesInfo struct {
	Version  string `json:"version"`
	Override string `json:"override,omitempty"`
}

// FileEntry describes one file in the manifest.
type FileEntry struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`  // "file" or "block"
	State  string `json:"state"` // created, merged, replaced, existing
	SHA256 string `json:"sha256,omitempty"`
}

// ReadManifest loads the manifest if there is one.
func ReadManifest(root string) (Manifest, bool, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Manifest{}, false, nil
		}
		return Manifest{}, false, fmt.Errorf("scaffold: open root: %w", err)
	}
	defer func() { _ = r.Close() }()

	data, err := r.ReadFile(filepath.FromSlash(ManifestPath))
	if errors.Is(err, fs.ErrNotExist) {
		return Manifest{}, false, nil
	}
	if err != nil {
		return Manifest{}, false, fmt.Errorf("scaffold: read manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, true, fmt.Errorf("scaffold: parse manifest: %w", err)
	}
	return m, true, nil
}

// Entry reports what happened to one file.
type Entry struct {
	Path   string
	Kind   Kind
	Reason string
}

// Report is the outcome of Apply.
type Report struct {
	Entries   []Entry
	BackupDir string // relative to the root; empty when nothing was backed up
	Todos     int    // number of TODO(human) markers in generated content
}

// Apply performs the plan. It never overwrites a file the plan marks as
// skipped, writes every file atomically, and backs up anything it replaces.
// All reads, writes, and backups are strictly confined within p.Root via os.Root.
func Apply(p Plan, now time.Time, templatesOverride string) (Report, error) {
	var rep Report
	rootHandle, err := os.OpenRoot(p.Root)
	if err != nil {
		return rep, fmt.Errorf("scaffold: open project root %s: %w", p.Root, err)
	}
	defer func() { _ = rootHandle.Close() }()

	backupRel := ""
	backup := func(rel string, original []byte) error {
		if backupRel == "" {
			backupRel = filepath.ToSlash(filepath.Join(".sdd", "backup", now.UTC().Format("20060102-150405")))
		}
		dst := filepath.Join(filepath.FromSlash(backupRel), filepath.FromSlash(rel))
		return fsutil.WriteAtomicRoot(rootHandle, dst, original, 0o644)
	}

	manifest := Manifest{
		Schema:    1,
		Tool:      ToolInfo{Name: "sdd", Version: version.Version},
		Templates: TemplatesInfo{Version: templates.Version, Override: templatesOverride},
		Generated: now.UTC().Format("2006-01-02"),
		Answers:   p.Data.Answers,
	}

	for _, op := range p.Ops {
		if err := safeRel(op.Path); err != nil {
			return rep, err
		}
		rel := filepath.FromSlash(op.Path)
		entryKind, state := "file", ""
		if op.Block || op.Managed {
			entryKind = "block"
		}
		switch op.Kind {
		case KindCreate:
			if err := fsutil.WriteAtomicRoot(rootHandle, rel, []byte(op.Content), 0o644); err != nil {
				return rep, err
			}
			state = "created"
		case KindMerge:
			existing, err := rootHandle.ReadFile(rel)
			if err != nil {
				return rep, fmt.Errorf("scaffold: read %s: %w", op.Path, err)
			}
			text := strings.TrimRight(string(existing), "\r\n")
			if text != "" {
				text += "\n\n"
			}
			if err := fsutil.WriteAtomicRoot(rootHandle, rel, []byte(text+op.Content+"\n"), 0o644); err != nil {
				return rep, err
			}
			state = "merged"
		case KindReplace:
			existing, err := rootHandle.ReadFile(rel)
			if err != nil {
				return rep, fmt.Errorf("scaffold: read %s: %w", op.Path, err)
			}
			if err := backup(op.Path, existing); err != nil {
				return rep, err
			}
			out := []byte(op.Content)
			if op.Block {
				bs, _, _, be, ok := block.Locate(string(existing))
				if !ok {
					return rep, fmt.Errorf("scaffold: %s no longer contains a managed block", op.Path)
				}
				out = []byte(string(existing)[:bs] + op.Content + string(existing)[be:])
			}
			if err := fsutil.WriteAtomicRoot(rootHandle, rel, out, 0o644); err != nil {
				return rep, err
			}
			state = "replaced"
		case KindSkip:
			state = "existing"
		}
		if op.Kind == KindCreate || op.Kind == KindReplace {
			rep.Todos += strings.Count(op.Content, Todo)
		}
		rep.Entries = append(rep.Entries, Entry{Path: op.Path, Kind: op.Kind, Reason: op.Reason})
		manifest.Files = append(manifest.Files, FileEntry{Path: op.Path, Kind: entryKind, State: state, SHA256: op.Hash})
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return rep, fmt.Errorf("scaffold: encode manifest: %w", err)
	}
	manifestRel := filepath.FromSlash(ManifestPath)
	if existing, err := rootHandle.ReadFile(manifestRel); err == nil {
		if err := backup(ManifestPath, existing); err != nil {
			return rep, err
		}
	}
	if err := fsutil.WriteAtomicRoot(rootHandle, manifestRel, append(data, '\n'), 0o644); err != nil {
		return rep, err
	}
	rep.Entries = append(rep.Entries, Entry{Path: ManifestPath, Kind: KindCreate})
	rep.BackupDir = backupRel
	return rep, nil
}

func safeRel(rel string) error {
	if err := fsutil.ValidateLocalRelPath(filepath.FromSlash(rel)); err != nil {
		return fmt.Errorf("scaffold: %w", err)
	}
	return nil
}
