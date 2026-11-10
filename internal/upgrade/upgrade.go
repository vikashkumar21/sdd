// Package upgrade refreshes generated spec-driven development kit files
// from newer templates using manifest SHA256 hashes, preserving human edits.
package upgrade

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vikashkumar21/sdd/internal/block"
	"github.com/vikashkumar21/sdd/internal/detect"
	"github.com/vikashkumar21/sdd/internal/fsutil"
	"github.com/vikashkumar21/sdd/internal/repair"
	"github.com/vikashkumar21/sdd/internal/scaffold"
	"github.com/vikashkumar21/sdd/internal/templates"
	"github.com/vikashkumar21/sdd/internal/version"
)

// Action represents what will happen to a file during upgrade.
type Action string

const (
	ActionUnchanged Action = "unchanged" // file already matches new template
	ActionUpdate    Action = "update"    // untouched file updated in place with backup
	ActionProposal  Action = "proposal"  // edited file preserved; .sdd-new written with diff
	ActionCreate    Action = "create"    // new template file created
	ActionForce     Action = "force"     // edited file overwritten under --force with backup
)

// Options configures upgrade planning.
type Options struct {
	TemplatesDir string // optional templates override directory
	Force        bool   // replace edited files instead of creating proposals
	Now          time.Time
}

// Item describes the planned upgrade operation for one file.
type Item struct {
	Path            string
	Kind            string // "file" or "block"
	Action          Action
	Reason          string
	OriginalContent string
	ProposedContent string
	Diff            string
	NewHash         string
}

// FolderMigration describes a legacy root folder to be moved to tasks/.
type FolderMigration struct {
	From string // e.g. "features"
	To   string // e.g. "tasks/features"
}

// Plan represents the full set of planned upgrade operations.
type Plan struct {
	Root              string
	OldTemplatesVer   string
	NewTemplatesVer   string
	TemplatesOverride string
	Migrations        []FolderMigration
	Items             []Item
	Manifest          scaffold.Manifest
	UpToDate          bool
}

// Report summarizes the executed upgrade.
type Report struct {
	Migrated  []string
	Updated   []string
	Proposals []string
	Created   []string
	Unchanged []string
	BackupDir string
	UpToDate  bool
}

func sum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// BuildPlan inspects the project manifest, compares file hashes, and decides
// which files can be safely updated and which require side-by-side proposals.
func BuildPlan(root string, opts Options) (*Plan, error) {
	manifest, ok, err := scaffold.ReadManifest(root)
	if err != nil {
		return nil, fmt.Errorf("upgrade: %w", err)
	}
	if !ok {
		return nil, errors.New("cannot upgrade: project has no manifest (.sdd/manifest.json); run sdd init or sdd adopt first")
	}
	if manifest.Schema != 1 {
		return nil, fmt.Errorf("cannot upgrade: unsupported manifest schema %d (expected 1)", manifest.Schema)
	}

	set, err := templates.New(opts.TemplatesDir)
	if err != nil {
		return nil, fmt.Errorf("upgrade: load templates: %w", err)
	}

	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	// Prepare template data from manifest answers
	ans := manifest.Answers
	det, _ := detect.Detect(root)
	data := scaffold.Data{
		Answers:         ans,
		Date:            now.UTC().Format("2006-01-02"),
		ToolVersion:     version.Version,
		TemplateVersion: templates.Version,
		Greenfield:      ans.Mode == scaffold.ModeGreenfield,
		Brownfield:      ans.Mode == scaffold.ModeBrownfield,
		Detected:        det,
	}
	var titles []string
	for _, l := range ans.Languages {
		pack, err := set.Pack(l)
		if err != nil {
			return nil, fmt.Errorf("upgrade: load pack %s: %w", l, err)
		}
		data.Packs = append(data.Packs, pack)
		titles = append(titles, pack.Title)
	}
	data.LanguageList = strings.Join(titles, ", ")
	for _, id := range ans.Agents {
		data.AgentFiles = append(data.AgentFiles, scaffold.AgentFile(id))
	}

	// Index existing manifest entries by relative path
	manifestFiles := map[string]scaffold.FileEntry{}
	for _, f := range manifest.Files {
		manifestFiles[f.Path] = f
	}

	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("upgrade: open root: %w", err)
	}
	defer func() { _ = rootHandle.Close() }()

	plan := &Plan{
		Root:              root,
		OldTemplatesVer:   manifest.Templates.Version,
		NewTemplatesVer:   templates.Version,
		TemplatesOverride: opts.TemplatesDir,
		Manifest:          manifest,
	}

	// Render contract body for agent files
	body, err := set.Render("base/contract.md.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("upgrade: render contract: %w", err)
	}
	body = strings.Trim(body, "\n")
	blk := block.Begin(templates.Version) + "\n" + body + "\n" + block.End
	bodyHash := sum(body)

	// 1. Process agent files
	for _, id := range ans.Agents {
		path := scaffold.AgentFile(id)
		entry, hasEntry := manifestFiles[path]
		rawBytes, readErr := rootHandle.ReadFile(filepath.FromSlash(path))
		if errors.Is(readErr, fs.ErrNotExist) {
			title := agentTitle(id, ans.Project)
			fullContent := title + "\n\n" + blk + "\n"
			plan.Items = append(plan.Items, Item{
				Path:            path,
				Kind:            "block",
				Action:          ActionCreate,
				Reason:          "missing agent file recreated",
				ProposedContent: fullContent,
				NewHash:         bodyHash,
			})
			continue
		} else if readErr != nil {
			return nil, fmt.Errorf("upgrade: read %s: %w", path, readErr)
		}

		existing := strings.ReplaceAll(string(rawBytes), "\r\n", "\n")
		inner, hasBlock := block.Inner(existing)
		if !hasBlock {
			// Block missing or corrupted: treat as edited
			proposedFull := existing + "\n\n" + blk + "\n"
			diff := repair.Diff(existing, proposedFull, path)
			act := ActionProposal
			reason := "managed block was removed; proposal written"
			if opts.Force {
				act = ActionForce
				reason = "managed block re-appended (--force)"
			}
			plan.Items = append(plan.Items, Item{
				Path:            path,
				Kind:            "block",
				Action:          act,
				Reason:          reason,
				OriginalContent: existing,
				ProposedContent: proposedFull,
				Diff:            diff,
				NewHash:         bodyHash,
			})
			continue
		}

		// Calculate existing block inner hash
		innerHash := sum(inner)
		bs, _, _, be, _ := block.Locate(existing)
		proposedFull := existing[:bs] + blk + existing[be:]
		diff := repair.Diff(existing, proposedFull, path)

		// Compare with recorded manifest hash
		isUntouched := hasEntry && innerHash == entry.SHA256
		isUnchanged := innerHash == bodyHash

		switch {
		case isUnchanged:
			plan.Items = append(plan.Items, Item{
				Path:            path,
				Kind:            "block",
				Action:          ActionUnchanged,
				Reason:          "managed block is up-to-date",
				OriginalContent: existing,
				ProposedContent: existing,
				NewHash:         bodyHash,
			})
		case isUntouched:
			plan.Items = append(plan.Items, Item{
				Path:            path,
				Kind:            "block",
				Action:          ActionUpdate,
				Reason:          "untouched managed block refreshed",
				OriginalContent: existing,
				ProposedContent: proposedFull,
				Diff:            diff,
				NewHash:         bodyHash,
			})
		default: // Edited by human
			if opts.Force {
				plan.Items = append(plan.Items, Item{
					Path:            path,
					Kind:            "block",
					Action:          ActionForce,
					Reason:          "managed block replaced (--force); backup created",
					OriginalContent: existing,
					ProposedContent: proposedFull,
					Diff:            diff,
					NewHash:         bodyHash,
				})
			} else {
				plan.Items = append(plan.Items, Item{
					Path:            path,
					Kind:            "block",
					Action:          ActionProposal,
					Reason:          "human edits detected in managed block; proposal written",
					OriginalContent: existing,
					ProposedContent: proposedFull,
					Diff:            diff,
					NewHash:         bodyHash,
				})
			}
		}
	}

	// 2. Process core living documents
	coreDocs := []struct {
		path, tmpl string
		protect    bool
	}{
		{"MISSION.md", "base/MISSION.md.tmpl", false},
		{"TECH_STACK.md", "base/TECH_STACK.md.tmpl", false},
		{"ROADMAP.md", "base/ROADMAP.md.tmpl", false},
		{"SKILLS.md", "base/SKILLS.md.tmpl", false},
		{"README.md", "base/README.md.tmpl", true},
	}
	for _, doc := range coreDocs {
		rendered, err := set.Render(doc.tmpl, data)
		if err != nil {
			return nil, fmt.Errorf("upgrade: render %s: %w", doc.path, err)
		}
		newHash := sum(rendered)
		entry, hasEntry := manifestFiles[doc.path]

		rawBytes, readErr := rootHandle.ReadFile(filepath.FromSlash(doc.path))
		if errors.Is(readErr, fs.ErrNotExist) {
			plan.Items = append(plan.Items, Item{
				Path:            doc.path,
				Kind:            "file",
				Action:          ActionCreate,
				Reason:          "missing document recreated",
				ProposedContent: rendered,
				NewHash:         newHash,
			})
			continue
		} else if readErr != nil {
			return nil, fmt.Errorf("upgrade: read %s: %w", doc.path, readErr)
		}

		existing := strings.ReplaceAll(string(rawBytes), "\r\n", "\n")
		currHash := sum(existing)
		diff := repair.Diff(existing, rendered, doc.path)

		// README.md is never overwritten unless it was explicitly generated and untouched
		if doc.protect && (!hasEntry || entry.State == "existing" || entry.SHA256 == "") {
			plan.Items = append(plan.Items, Item{
				Path:            doc.path,
				Kind:            "file",
				Action:          ActionUnchanged,
				Reason:          "user README preserved",
				OriginalContent: existing,
				ProposedContent: existing,
				NewHash:         currHash,
			})
			continue
		}

		isUntouched := hasEntry && currHash == entry.SHA256
		isUnchanged := currHash == newHash

		switch {
		case isUnchanged:
			plan.Items = append(plan.Items, Item{
				Path:            doc.path,
				Kind:            "file",
				Action:          ActionUnchanged,
				Reason:          "document is up-to-date",
				OriginalContent: existing,
				ProposedContent: existing,
				NewHash:         newHash,
			})
		case isUntouched:
			plan.Items = append(plan.Items, Item{
				Path:            doc.path,
				Kind:            "file",
				Action:          ActionUpdate,
				Reason:          "untouched document updated to new template",
				OriginalContent: existing,
				ProposedContent: rendered,
				Diff:            diff,
				NewHash:         newHash,
			})
		default: // Edited by human
			if opts.Force {
				plan.Items = append(plan.Items, Item{
					Path:            doc.path,
					Kind:            "file",
					Action:          ActionForce,
					Reason:          "document overwritten (--force); backup created",
					OriginalContent: existing,
					ProposedContent: rendered,
					Diff:            diff,
					NewHash:         newHash,
				})
			} else {
				plan.Items = append(plan.Items, Item{
					Path:            doc.path,
					Kind:            "file",
					Action:          ActionProposal,
					Reason:          "human edits detected; proposal written without overwriting",
					OriginalContent: existing,
					ProposedContent: rendered,
					Diff:            diff,
					NewHash:         newHash,
				})
			}
		}
	}

	// 3. Process task templates across all 4 typologies
	typologyTemplates := map[string][]string{
		"features": {"requirements.md", "plan.md", "validation.md", "review.md"},
		"bugfixes": {"reproduction.md", "rca.md", "plan.md", "validation.md", "review.md"},
		"hotfixes": {"incident.md", "patch.md", "validation.md", "post_mortem.md"},
		"spikes":   {"hypotheses.md", "findings.md", "plan.md"},
	}
	typologyOrder := []string{"features", "bugfixes", "hotfixes", "spikes"}

	var migrations []FolderMigration
	for _, topDir := range typologyOrder {
		legacyPath := filepath.Join(root, topDir)
		canonicalPath := filepath.Join(root, "tasks", topDir)
		if info, err := os.Stat(legacyPath); err == nil && info.IsDir() {
			if _, err2 := os.Stat(canonicalPath); errors.Is(err2, fs.ErrNotExist) {
				migrations = append(migrations, FolderMigration{
					From: topDir,
					To:   "tasks/" + topDir,
				})
			}
		}
	}
	plan.Migrations = migrations

	for _, topDir := range typologyOrder {
		for _, name := range typologyTemplates[topDir] {
			tmplPath := "tasks/" + topDir + "/_templates/" + name
			legacyTmplPath := topDir + "/_templates/" + name

			rawTmpl, err := set.Raw("tasks/" + topDir + "/" + name)
			if err != nil {
				return nil, fmt.Errorf("upgrade: read template %s/%s: %w", topDir, name, err)
			}
			newHash := sum(rawTmpl)

			entry, hasEntry := manifestFiles[tmplPath]
			if !hasEntry {
				entry, hasEntry = manifestFiles[legacyTmplPath]
			}

			rawBytes, readErr := rootHandle.ReadFile(filepath.FromSlash(tmplPath))
			if errors.Is(readErr, fs.ErrNotExist) {
				rawBytes, readErr = rootHandle.ReadFile(filepath.FromSlash(legacyTmplPath))
			}
			if errors.Is(readErr, fs.ErrNotExist) {
				plan.Items = append(plan.Items, Item{
					Path:            tmplPath,
					Kind:            "file",
					Action:          ActionCreate,
					Reason:          "missing task template recreated",
					ProposedContent: rawTmpl,
					NewHash:         newHash,
				})
				continue
			} else if readErr != nil {
				return nil, fmt.Errorf("upgrade: read %s: %w", tmplPath, readErr)
			}

			existing := strings.ReplaceAll(string(rawBytes), "\r\n", "\n")
			currHash := sum(existing)
			diff := repair.Diff(existing, rawTmpl, tmplPath)

			isUntouched := hasEntry && currHash == entry.SHA256
			isUnchanged := currHash == newHash

			switch {
			case isUnchanged:
				plan.Items = append(plan.Items, Item{
					Path:            tmplPath,
					Kind:            "file",
					Action:          ActionUnchanged,
					Reason:          "task template is up-to-date",
					OriginalContent: existing,
					ProposedContent: existing,
					NewHash:         newHash,
				})
			case isUntouched:
				plan.Items = append(plan.Items, Item{
					Path:            tmplPath,
					Kind:            "file",
					Action:          ActionUpdate,
					Reason:          "untouched task template updated",
					OriginalContent: existing,
					ProposedContent: rawTmpl,
					Diff:            diff,
					NewHash:         newHash,
				})
			default: // Edited by human
				if opts.Force {
					plan.Items = append(plan.Items, Item{
						Path:            tmplPath,
						Kind:            "file",
						Action:          ActionForce,
						Reason:          "task template overwritten (--force); backup created",
						OriginalContent: existing,
						ProposedContent: rawTmpl,
						Diff:            diff,
						NewHash:         newHash,
					})
				} else {
					plan.Items = append(plan.Items, Item{
						Path:            tmplPath,
						Kind:            "file",
						Action:          ActionProposal,
						Reason:          "customized task template; proposal written",
						OriginalContent: existing,
						ProposedContent: rawTmpl,
						Diff:            diff,
						NewHash:         newHash,
					})
				}
			}
		}
	}

	// Check if everything is unchanged
	allUnchanged := len(plan.Migrations) == 0
	for _, it := range plan.Items {
		if it.Action != ActionUnchanged {
			allUnchanged = false
			break
		}
	}
	plan.UpToDate = allUnchanged

	return plan, nil
}

// Apply executes the upgrade plan atomically under os.Root, creating backups
// for any replaced files and writing .sdd-new proposals for edited files.
func Apply(plan *Plan, now time.Time) (*Report, error) {
	if plan == nil {
		return nil, errors.New("upgrade: plan is nil")
	}

	rep := &Report{
		UpToDate: plan.UpToDate,
	}
	if plan.UpToDate {
		return rep, nil
	}

	if now.IsZero() {
		now = time.Now()
	}

	rootHandle, err := os.OpenRoot(plan.Root)
	if err != nil {
		return nil, fmt.Errorf("upgrade: open root: %w", err)
	}
	defer func() { _ = rootHandle.Close() }()

	backupTimestamp := now.UTC().Format("20060102-150405")
	backupRel := filepath.ToSlash(filepath.Join(".sdd", "backup", backupTimestamp))

	backup := func(rel string, original []byte) error {
		rep.BackupDir = backupRel
		dst := filepath.Join(filepath.FromSlash(backupRel), filepath.FromSlash(rel))
		return fsutil.WriteAtomicRoot(rootHandle, dst, original, 0o644)
	}

	manifestMap := map[string]*scaffold.FileEntry{}
	for i := range plan.Manifest.Files {
		manifestMap[plan.Manifest.Files[i].Path] = &plan.Manifest.Files[i]
	}

	if len(plan.Migrations) > 0 {
		_ = os.MkdirAll(filepath.Join(plan.Root, "tasks"), 0o755)
		for _, mig := range plan.Migrations {
			fromPath := filepath.Join(plan.Root, filepath.FromSlash(mig.From))
			toPath := filepath.Join(plan.Root, filepath.FromSlash(mig.To))
			if err := os.Rename(fromPath, toPath); err != nil {
				return nil, fmt.Errorf("upgrade: migrate %s to %s: %w", mig.From, mig.To, err)
			}
			rep.Migrated = append(rep.Migrated, fmt.Sprintf("%s -> %s", mig.From, mig.To))
		}
		for i := range plan.Manifest.Files {
			for _, mig := range plan.Migrations {
				if strings.HasPrefix(plan.Manifest.Files[i].Path, mig.From+"/") {
					plan.Manifest.Files[i].Path = mig.To + "/" + strings.TrimPrefix(plan.Manifest.Files[i].Path, mig.From+"/")
				}
			}
		}
		manifestMap = map[string]*scaffold.FileEntry{}
		for i := range plan.Manifest.Files {
			manifestMap[plan.Manifest.Files[i].Path] = &plan.Manifest.Files[i]
		}
	}

	for _, item := range plan.Items {
		rel := filepath.FromSlash(item.Path)

		switch item.Action {
		case ActionUnchanged:
			rep.Unchanged = append(rep.Unchanged, item.Path)

		case ActionUpdate, ActionForce:
			// Read existing file to back it up
			existingBytes, err := rootHandle.ReadFile(rel)
			if err == nil {
				if err := backup(item.Path, existingBytes); err != nil {
					return nil, fmt.Errorf("upgrade: backup %s: %w", item.Path, err)
				}
			}
			// Write proposed content atomically
			if err := fsutil.WriteAtomicRoot(rootHandle, rel, []byte(item.ProposedContent), 0o644); err != nil {
				return nil, fmt.Errorf("upgrade: write %s: %w", item.Path, err)
			}
			rep.Updated = append(rep.Updated, item.Path)

			// Update manifest entry
			if entry, ok := manifestMap[item.Path]; ok {
				entry.SHA256 = item.NewHash
				entry.State = "replaced"
			} else {
				plan.Manifest.Files = append(plan.Manifest.Files, scaffold.FileEntry{
					Path:   item.Path,
					Kind:   item.Kind,
					State:  "replaced",
					SHA256: item.NewHash,
				})
			}

		case ActionCreate:
			if err := fsutil.WriteAtomicRoot(rootHandle, rel, []byte(item.ProposedContent), 0o644); err != nil {
				return nil, fmt.Errorf("upgrade: write %s: %w", item.Path, err)
			}
			rep.Created = append(rep.Created, item.Path)
			plan.Manifest.Files = append(plan.Manifest.Files, scaffold.FileEntry{
				Path:   item.Path,
				Kind:   item.Kind,
				State:  "created",
				SHA256: item.NewHash,
			})

		case ActionProposal:
			proposalPath := rel + ".sdd-new"
			if err := fsutil.WriteAtomicRoot(rootHandle, proposalPath, []byte(item.ProposedContent), 0o644); err != nil {
				return nil, fmt.Errorf("upgrade: write proposal for %s: %w", item.Path, err)
			}
			rep.Proposals = append(rep.Proposals, item.Path+".sdd-new")
		}
	}

	// Update manifest metadata
	plan.Manifest.Tool.Version = version.Version
	plan.Manifest.Templates.Version = plan.NewTemplatesVer
	plan.Manifest.Templates.Override = plan.TemplatesOverride

	data, err := json.MarshalIndent(plan.Manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("upgrade: encode manifest: %w", err)
	}
	if err := fsutil.WriteAtomicRoot(rootHandle, filepath.FromSlash(scaffold.ManifestPath), append(data, '\n'), 0o644); err != nil {
		return nil, fmt.Errorf("upgrade: write manifest: %w", err)
	}

	return rep, nil
}

func agentTitle(id, project string) string {
	switch id {
	case "claude":
		return fmt.Sprintf("# CLAUDE.md — %s agent contract (Claude)", project)
	case "gemini":
		return fmt.Sprintf("# GEMINI.md — %s agent contract (Gemini)", project)
	default:
		return fmt.Sprintf("# AGENTS.md — %s agent contract (all agents)", project)
	}
}
