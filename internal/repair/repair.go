// Package repair provides safe, mechanical repairs for project kits:
// contract synchronization across agent files and markdown table delimiter alignment.
// All writes are previewed via diffs, confirmed by the human, and preceded by backups.
package repair

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vikashkumar21/sdd/internal/block"
	"github.com/vikashkumar21/sdd/internal/fsutil"
	"github.com/vikashkumar21/sdd/internal/markdown"
	"github.com/vikashkumar21/sdd/internal/suppress"
)

// Patch represents a planned repair for a single file.
type Patch struct {
	Path           string // relative slash-separated path
	Rule           string // rule identifier (e.g. "contract-parity", "table-format")
	Description    string // human-readable explanation
	Original       string
	Modified       string
	Diff           string // unified diff preview
	Suppressed     bool   // true if currently waived in .sdd/suppressions.json
	SuppressReason string // human waiver reason
}

// Plan contains all planned repairs.
type Plan struct {
	Patches []Patch
}

// Result describes the outcome of applying repairs.
type Result struct {
	BackupDir string   // relative path to backup directory
	Repaired  []string // list of repaired files
}

var agentFiles = []string{"AGENTS.md", "CLAUDE.md", "GEMINI.md"}

var ignoredDirs = map[string]bool{
	".git": true, ".sdd": true, "node_modules": true, "vendor": true, "target": true,
	"build": true, "dist": true, "bin": true, ".venv": true, "venv": true,
}

// PlanRepairs inspects root and computes mechanical repairs.
// When filterFiles is provided, only files matching the filter are considered.
func PlanRepairs(root string, filterFiles []string) (Plan, error) {
	filter := map[string]bool{}
	for _, f := range filterFiles {
		filter[filepath.ToSlash(filepath.Clean(f))] = true
	}
	shouldInclude := func(path string) bool {
		if len(filter) == 0 {
			return true
		}
		return filter[path]
	}

	var plan Plan

	// 1. Contract synchronization
	agentPatches, err := planContractRepairs(root, shouldInclude)
	if err != nil {
		return plan, err
	}
	plan.Patches = append(plan.Patches, agentPatches...)

	// 2. Table delimiter repairs
	tablePatches, err := planTableRepairs(root, shouldInclude)
	if err != nil {
		return plan, err
	}
	plan.Patches = append(plan.Patches, tablePatches...)

	// 3. Check for active suppressions
	suppManifest, hasSupp, _ := suppress.Load(root)
	if hasSupp && len(suppManifest.Suppressions) > 0 {
		for i := range plan.Patches {
			p := &plan.Patches[i]
			for _, entry := range suppManifest.Suppressions {
				if entry.Matches(p.Path, p.Rule) {
					p.Suppressed = true
					p.SuppressReason = entry.Reason
					break
				}
			}
		}
	}

	return plan, nil
}

func planContractRepairs(root string, shouldInclude func(string) bool) ([]Patch, error) {
	var canonicalInner string
	var canonicalSource string

	// Find the canonical contract: prefer AGENTS.md, then CLAUDE.md, then GEMINI.md
	for _, name := range agentFiles {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			continue
		}
		if inner, ok := block.Inner(string(data)); ok && strings.TrimSpace(inner) != "" {
			canonicalInner = inner
			canonicalSource = name
			break
		}
	}

	if canonicalInner == "" {
		return nil, nil // no managed block found anywhere; cannot synchronize
	}

	var patches []Patch
	for _, name := range agentFiles {
		if !shouldInclude(name) {
			continue
		}
		path := filepath.Join(root, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue // file does not exist
		}
		content := string(data)
		normContent := strings.ReplaceAll(content, "\r\n", "\n")

		_, innerStart, innerEnd, _, ok := block.Locate(normContent)
		if ok {
			existingInner := normContent[innerStart:innerEnd]
			if strings.TrimSpace(existingInner) != strings.TrimSpace(canonicalInner) {
				mod := normContent[:innerStart] + canonicalInner + normContent[innerEnd:]
				diff := Diff(normContent, mod, name)
				patches = append(patches, Patch{
					Path:        name,
					Rule:        "contract-parity",
					Description: fmt.Sprintf("synchronize managed contract block to match %s", canonicalSource),
					Original:    normContent,
					Modified:    mod,
					Diff:        diff,
				})
			}
		} else {
			// Append managed block
			mod := strings.TrimRight(normContent, "\n") + "\n\n" + block.Begin("1.0.0") + "\n" + canonicalInner + "\n<!-- sdd:end contract -->\n"
			diff := Diff(normContent, mod, name)
			patches = append(patches, Patch{
				Path:        name,
				Rule:        "contract-parity",
				Description: fmt.Sprintf("append managed contract block from %s", canonicalSource),
				Original:    normContent,
				Modified:    mod,
				Diff:        diff,
			})
		}
	}
	return patches, nil
}

func planTableRepairs(root string, shouldInclude func(string) bool) ([]Patch, error) {
	var patches []Patch
	var mdFiles []string

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == "." {
			return nil
		}
		if d.IsDir() {
			if ignoredDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			slashRel := filepath.ToSlash(rel)
			if shouldInclude(slashRel) {
				mdFiles = append(mdFiles, slashRel)
			}
		}
		return nil
	})

	for _, rel := range mdFiles {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		content := string(data)
		normContent := strings.ReplaceAll(content, "\r\n", "\n")

		tables := markdown.ExtractTables(normContent)
		lines := strings.Split(normContent, "\n")
		modified := false

		for _, t := range tables {
			want := t.Header.Columns
			if want <= 0 {
				continue
			}
			for _, r := range t.Rows {
				if r.IsDelim && r.Columns != want && r.LineNo > 0 && r.LineNo <= len(lines) {
					// Delimiter row mismatch: replace with correct delimiter count
					delim := "|" + strings.Repeat("---|", want)
					lines[r.LineNo-1] = delim
					modified = true
				}
			}
		}

		if modified {
			modText := strings.Join(lines, "\n")
			diff := Diff(normContent, modText, rel)
			patches = append(patches, Patch{
				Path:        rel,
				Rule:        "table-format",
				Description: "repair malformed markdown table delimiters",
				Original:    normContent,
				Modified:    modText,
				Diff:        diff,
			})
		}
	}
	return patches, nil
}

// Apply executes the approved patches in plan, backing up originals first.
// If approvedPaths is non-empty, only paths in approvedPaths are applied.
func Apply(root string, plan Plan, approvedPaths map[string]bool, now time.Time) (Result, error) {
	var res Result
	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return res, fmt.Errorf("repair: open root %s: %w", root, err)
	}
	defer func() { _ = rootHandle.Close() }()

	backupRel := filepath.ToSlash(filepath.Join(".sdd", "backup", now.UTC().Format("20060102-150405")))

	for _, p := range plan.Patches {
		if len(approvedPaths) > 0 && !approvedPaths[p.Path] {
			continue
		}
		// 1. Back up original
		backupPath := filepath.Join(backupRel, filepath.FromSlash(p.Path))
		if err := fsutil.WriteAtomicRoot(rootHandle, backupPath, []byte(p.Original), 0o644); err != nil {
			return res, fmt.Errorf("repair: backup %s: %w", p.Path, err)
		}

		// 2. Write modified
		if err := fsutil.WriteAtomicRoot(rootHandle, filepath.FromSlash(p.Path), []byte(p.Modified), 0o644); err != nil {
			return res, fmt.Errorf("repair: write %s: %w", p.Path, err)
		}

		res.Repaired = append(res.Repaired, p.Path)
		// 3. Clear suppression waiver for repaired file
		_ = suppress.RemoveFileRules(root, p.Path, p.Rule)
	}

	if len(res.Repaired) > 0 {
		res.BackupDir = backupRel
	}
	return res, nil
}
