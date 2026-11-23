// Package docflow provides generation and management for deep domain flow
// and visual architecture documentation with Mermaid diagrams.
package docflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vikashkumar21/sdd/internal/fsutil"
	"github.com/vikashkumar21/sdd/internal/scaffold"
	"github.com/vikashkumar21/sdd/internal/templates"
)

var flowNameRegex = regexp.MustCompile(`^[a-z0-9]+(?:[-_][a-z0-9]+)*$`)

// InitOptions configures the sdd doc init command.
type InitOptions struct {
	Dir         string
	Force       bool
	DryRun      bool
	JSON        bool
	Now         time.Time
	ProjectName string
}

// FlowOptions configures the sdd doc flow command.
type FlowOptions struct {
	Dir         string
	Name        string
	Title       string
	Force       bool
	DryRun      bool
	JSON        bool
	Now         time.Time
	ProjectName string
}

// PlanEntry represents one planned document operation.
type PlanEntry struct {
	RelPath string `json:"path"`
	Action  string `json:"action"` // "create", "skip", "replace"
	Reason  string `json:"reason"`
	Content []byte `json:"-"`
}

// Result summarizes the execution of a doc operation.
type Result struct {
	Entries []PlanEntry `json:"entries"`
	Created []string    `json:"created,omitempty"`
	Skipped []string    `json:"skipped,omitempty"`
	Backups []string    `json:"backups,omitempty"`
}

// BuildInitPlan computes the planned files for sdd doc init without writing to disk.
func BuildInitPlan(opts InitOptions) ([]PlanEntry, error) {
	root := opts.Dir
	if root == "" {
		root = "."
	}
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	projectName := opts.ProjectName
	if projectName == "" {
		abs, err := filepath.Abs(root)
		if err == nil {
			projectName = filepath.Base(abs)
		} else {
			projectName = "Project"
		}
	}

	tmplSet, err := templates.New("")
	if err != nil {
		return nil, fmt.Errorf("docflow: load templates: %w", err)
	}

	targets := []struct {
		relPath  string
		tmplName string
		flowName string
		title    string
	}{
		{
			relPath:  filepath.Join("docs", "architecture", "overview.md"),
			tmplName: "docs/architecture/overview.md.tmpl",
		},
		{
			relPath:  filepath.Join("docs", "architecture", "data_model.md"),
			tmplName: "docs/architecture/data_model.md.tmpl",
		},
		{
			relPath:  filepath.Join("docs", "business_flows", "overview.md"),
			tmplName: "docs/business_flows/overview.md.tmpl",
		},
		{
			relPath:  filepath.Join("docs", "business_flows", "sample_flow.md"),
			tmplName: "docs/business_flows/flow.md.tmpl",
			flowName: "sample_flow",
			title:    "Sample Business Flow",
		},
	}

	var plan []PlanEntry
	for _, target := range targets {
		data := struct {
			ProjectName string
			Date        string
			Name        string
			Title       string
		}{
			ProjectName: projectName,
			Date:        now.Format("2006-01-02"),
			Name:        target.flowName,
			Title:       target.title,
		}

		rendered, rerr := tmplSet.Render(target.tmplName, data)
		if rerr != nil {
			return nil, fmt.Errorf("docflow: render %s: %w", target.tmplName, rerr)
		}

		cleanRel := filepath.ToSlash(target.relPath)
		fullPath := filepath.Join(root, target.relPath)

		action := "create"
		reason := "create new documentation file"
		if _, serr := os.Stat(fullPath); serr == nil {
			if opts.Force {
				action = "replace"
				reason = "file exists; --force will replace and backup"
			} else {
				action = "skip"
				reason = "file already exists; use --force to overwrite"
			}
		}

		plan = append(plan, PlanEntry{
			RelPath: cleanRel,
			Action:  action,
			Reason:  reason,
			Content: []byte(rendered),
		})
	}

	return plan, nil
}

// BuildFlowPlan computes the planned file for sdd doc flow <name> without writing to disk.
func BuildFlowPlan(opts FlowOptions) ([]PlanEntry, error) {
	root := opts.Dir
	if root == "" {
		root = "."
	}
	name := strings.TrimSpace(strings.ToLower(opts.Name))
	if name == "" {
		return nil, errors.New("docflow: flow name cannot be empty")
	}
	if len(name) > 64 {
		return nil, errors.New("docflow: flow name exceeds 64 characters")
	}
	if !flowNameRegex.MatchString(name) {
		return nil, fmt.Errorf("docflow: invalid flow name %q (expected lowercase alphanumeric slug)", name)
	}
	if !filepath.IsLocal(name) {
		return nil, fmt.Errorf("docflow: flow name %q escapes directory", name)
	}

	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = humanizeSlug(name)
	}
	if len(title) > 120 {
		return nil, errors.New("docflow: flow title exceeds 120 characters")
	}
	if strings.ContainsAny(title, "\r\n`") {
		return nil, errors.New("docflow: flow title must be a single line without backticks")
	}

	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	projectName := opts.ProjectName
	if projectName == "" {
		abs, err := filepath.Abs(root)
		if err == nil {
			projectName = filepath.Base(abs)
		} else {
			projectName = "Project"
		}
	}

	tmplSet, err := templates.New("")
	if err != nil {
		return nil, fmt.Errorf("docflow: load templates: %w", err)
	}

	data := struct {
		ProjectName string
		Date        string
		Name        string
		Title       string
	}{
		ProjectName: projectName,
		Date:        now.Format("2006-01-02"),
		Name:        name,
		Title:       title,
	}

	rendered, err := tmplSet.Render("docs/business_flows/flow.md.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("docflow: render flow template: %w", err)
	}

	relPath := filepath.Join("docs", "business_flows", name+".md")
	cleanRel := filepath.ToSlash(relPath)
	fullPath := filepath.Join(root, relPath)

	action := "create"
	reason := "create new domain flow document"
	if _, serr := os.Stat(fullPath); serr == nil {
		if opts.Force {
			action = "replace"
			reason = "file exists; --force will replace and backup"
		} else {
			action = "skip"
			reason = "file already exists; use --force to overwrite"
		}
	}

	return []PlanEntry{
		{
			RelPath: cleanRel,
			Action:  action,
			Reason:  reason,
			Content: []byte(rendered),
		},
	}, nil
}

// Apply executes a list of planned entries safely and updates the manifest if present.
func Apply(root string, plan []PlanEntry, now time.Time) (*Result, error) {
	if root == "" {
		root = "."
	}
	if now.IsZero() {
		now = time.Now()
	}

	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("docflow: open root: %w", err)
	}
	defer func() { _ = rootHandle.Close() }()

	res := &Result{
		Entries: plan,
	}

	type fileMutation struct {
		relPath string
		hash    string
		state   string
	}
	var mutations []fileMutation

	backupDir := filepath.Join(".sdd", "backup", "docs")

	for _, entry := range plan {
		relOS := filepath.FromSlash(entry.RelPath)
		switch entry.Action {
		case "skip":
			res.Skipped = append(res.Skipped, entry.RelPath)
		case "replace":
			existing, rerr := rootHandle.ReadFile(relOS)
			if rerr != nil {
				return res, fmt.Errorf("docflow: read existing %s for backup: %w", entry.RelPath, rerr)
			}
			ts := now.Format("20060102-150405")
			flatPath := strings.ReplaceAll(filepath.ToSlash(entry.RelPath), "/", "-")
			bName := fmt.Sprintf("%s-%s", ts, flatPath)
			bRel := filepath.Join(backupDir, bName)
			if berr := fsutil.WriteAtomicRoot(rootHandle, filepath.FromSlash(bRel), existing, 0o644); berr != nil {
				return res, fmt.Errorf("docflow: write backup %s: %w", bRel, berr)
			}
			res.Backups = append(res.Backups, filepath.ToSlash(bRel))
			if werr := fsutil.WriteAtomicRoot(rootHandle, relOS, entry.Content, 0o644); werr != nil {
				return res, fmt.Errorf("docflow: write %s: %w", entry.RelPath, werr)
			}
			h := sha256.Sum256(entry.Content)
			mutations = append(mutations, fileMutation{
				relPath: entry.RelPath,
				hash:    hex.EncodeToString(h[:]),
				state:   "replaced",
			})
			res.Created = append(res.Created, entry.RelPath)
		case "create":
			if werr := fsutil.WriteAtomicRoot(rootHandle, relOS, entry.Content, 0o644); werr != nil {
				return res, fmt.Errorf("docflow: write %s: %w", entry.RelPath, werr)
			}
			h := sha256.Sum256(entry.Content)
			mutations = append(mutations, fileMutation{
				relPath: entry.RelPath,
				hash:    hex.EncodeToString(h[:]),
				state:   "created",
			})
			res.Created = append(res.Created, entry.RelPath)
		}
	}

	// Update manifest if present
	manifest, hasManifest, err := scaffold.ReadManifest(root)
	if err == nil && hasManifest {
		for _, m := range mutations {
			updated := false
			for i, fe := range manifest.Files {
				if fe.Path == m.relPath {
					manifest.Files[i].SHA256 = m.hash
					manifest.Files[i].State = m.state
					updated = true
					break
				}
			}
			if !updated {
				manifest.Files = append(manifest.Files, scaffold.FileEntry{
					Path:   m.relPath,
					Kind:   "file",
					State:  m.state,
					SHA256: m.hash,
				})
			}
		}

		mData, merr := json.MarshalIndent(manifest, "", "  ")
		if merr != nil {
			return res, fmt.Errorf("docflow: marshal manifest: %w", merr)
		}
		if werr := fsutil.WriteAtomicRoot(rootHandle, filepath.FromSlash(scaffold.ManifestPath), append(mData, '\n'), 0o644); werr != nil {
			return res, fmt.Errorf("docflow: update manifest: %w", werr)
		}
	}

	return res, nil
}

// humanizeSlug converts "user-login" to "User Login".
func humanizeSlug(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_'
	})
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}
