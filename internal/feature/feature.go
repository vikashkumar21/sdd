// Package feature creates the per-task folder (requirements, plan, validation)
// from the project's own templates, so that Phase A starts the same way every
// time.
package feature

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vikashkumar21/sdd/internal/fsutil"
	"github.com/vikashkumar21/sdd/internal/roadmap"
)

var (
	idRe   = regexp.MustCompile(`^T-\d{3}$`)
	slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

var templateNames = []string{"requirements.md", "plan.md", "validation.md"}

// Result describes what New created.
type Result struct {
	Dir           string   // relative, slash-separated
	Files         []string // relative, slash-separated
	StatusUpdated bool     // the ROADMAP status changed from Not started to Specifying
}

// New creates features/<id>-<slug>/ with the three documents. The task must
// already exist in ROADMAP.md; planning a task comes before specifying it.
// When title is empty, the title from the ROADMAP row is used.
func New(root, id, slug, title string) (Result, error) {
	if !idRe.MatchString(id) {
		return Result{}, fmt.Errorf("task ID %q must look like T-001", id)
	}
	if id == "T-000" {
		return Result{}, errors.New("the bootstrap task T-000 has no feature folder")
	}
	if !slugRe.MatchString(slug) {
		return Result{}, fmt.Errorf("slug %q must be lower-case words joined by hyphens (for example project-skeleton)", slug)
	}
	roadmapPath := filepath.Join(root, "ROADMAP.md")
	data, err := os.ReadFile(roadmapPath)
	if err != nil {
		return Result{}, fmt.Errorf("read ROADMAP.md: %w", err)
	}
	roadmapText := strings.ReplaceAll(string(data), "\r\n", "\n")
	tasks := roadmap.Parse(roadmapText)
	var task *roadmap.Task
	for i := range tasks {
		if tasks[i].ID == id {
			task = &tasks[i]
			break
		}
	}
	if task == nil {
		return Result{}, fmt.Errorf("%s has no task row in ROADMAP.md; add the task to the roadmap first", id)
	}
	if strings.TrimSpace(title) == "" {
		title = task.Title
	}
	if strings.ContainsAny(title, "\r\n") {
		return Result{}, errors.New("title must be a single line")
	}

	featuresDir := filepath.Join(root, "features")
	if entries, err := os.ReadDir(featuresDir); err == nil {
		for _, e := range entries {
			if e.IsDir() && strings.HasPrefix(e.Name(), id+"-") {
				return Result{}, fmt.Errorf("%s already has a feature folder: features/%s", id, e.Name())
			}
		}
	}

	rendered := make(map[string]string, len(templateNames))
	for _, name := range templateNames {
		rel := "features/_templates/" + name
		tpl, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return Result{}, fmt.Errorf("template %s is missing (run `sdd init`, or restore it): %w", rel, err)
		}
		text := strings.ReplaceAll(string(tpl), "T-xxx", id)
		text = strings.ReplaceAll(text, "<title>", title)
		rendered[name] = text
	}

	dirRel := "features/" + id + "-" + slug
	res := Result{Dir: dirRel}
	for _, name := range templateNames {
		rel := dirRel + "/" + name
		if err := fsutil.WriteAtomic(filepath.Join(root, filepath.FromSlash(rel)), []byte(rendered[name]), 0o644); err != nil {
			return res, err
		}
		res.Files = append(res.Files, rel)
	}

	if updated, ok := roadmap.SetStatus(roadmapText, id, "Not started", "Specifying"); ok {
		if err := fsutil.WriteAtomic(roadmapPath, []byte(updated), 0o644); err != nil {
			return res, err
		}
		res.StatusUpdated = true
	}
	return res, nil
}
