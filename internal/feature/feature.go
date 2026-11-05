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
	"time"

	sddcontext "github.com/vikashkumar21/sdd/internal/context"
	"github.com/vikashkumar21/sdd/internal/doc"
	"github.com/vikashkumar21/sdd/internal/fsutil"
	"github.com/vikashkumar21/sdd/internal/jira"
	"github.com/vikashkumar21/sdd/internal/roadmap"
)

var (
	idRe   = regexp.MustCompile(`^T-\d{3}$`)
	slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

var (
	requiredTemplateNames = []string{"requirements.md", "plan.md", "validation.md"}
	optionalTemplateNames = []string{"review.md"}
)

// Result describes what New created.
type Result struct {
	Dir           string   // relative, slash-separated
	Files         []string // relative, slash-separated
	StatusUpdated bool     // the ROADMAP status changed from Not started to Specifying
	StagedContext string   // relative path to staged context artifact if any
}

// Options provides optional parameters when creating a new feature folder.
type Options struct {
	Title        string
	FromJiraData []byte
	JiraOrigin   string    // filename or label of Jira source
	FromDocData  []byte    // raw document or PDF bytes
	FromDocPath  string    // original filename / path of document
	Now          time.Time // clock injection for staging context (defaults to time.Now())
}

// New creates features/<id>-<slug>/ with the three documents. The task must
// already exist in ROADMAP.md; planning a task comes before specifying it.
// When title is empty, the title from the ROADMAP row is used.
// All operations are confined to root via os.Root.
func New(root, id, slug, title string) (Result, error) {
	return NewWithOptions(root, id, slug, Options{Title: title})
}

// NewWithOptions creates features/<id>-<slug>/ with options including Jira ingestion.
func NewWithOptions(root, id, slug string, opts Options) (Result, error) {
	if !idRe.MatchString(id) {
		return Result{}, fmt.Errorf("task ID %q must look like T-001", id)
	}
	if id == "T-000" {
		return Result{}, errors.New("the bootstrap task T-000 has no feature folder")
	}
	if !slugRe.MatchString(slug) {
		return Result{}, fmt.Errorf("slug %q must be lower-case words joined by hyphens (for example project-skeleton)", slug)
	}

	var jiraIssue *jira.Issue
	if len(opts.FromJiraData) > 0 {
		iss, err := jira.Parse(opts.FromJiraData)
		if err != nil {
			return Result{}, fmt.Errorf("feature: parse jira issue: %w", err)
		}
		jiraIssue = &iss
	}

	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return Result{}, fmt.Errorf("feature: open root %s: %w", root, err)
	}
	defer func() { _ = rootHandle.Close() }()

	data, err := rootHandle.ReadFile("ROADMAP.md")
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
	title := opts.Title
	if strings.TrimSpace(title) == "" {
		title = task.Title
	}
	if strings.ContainsAny(title, "\r\n") {
		return Result{}, errors.New("title must be a single line")
	}

	if f, err := rootHandle.Open("features"); err == nil {
		entries, readErr := f.ReadDir(-1)
		_ = f.Close()
		if readErr == nil {
			for _, e := range entries {
				if e.IsDir() && strings.HasPrefix(e.Name(), id+"-") {
					return Result{}, fmt.Errorf("%s already has a feature folder: features/%s", id, e.Name())
				}
			}
		}
	}

	var docStagedPath string
	var docFormat string
	var docOrigin string
	if len(opts.FromDocData) > 0 {
		if doc.IsPDF(opts.FromDocPath, opts.FromDocData) {
			if _, err := doc.ValidatePDF(opts.FromDocData); err != nil {
				return Result{}, fmt.Errorf("feature: %w", err)
			}
			docFormat = "pdf"
		}
		now := opts.Now
		if now.IsZero() {
			now = time.Now()
		}
		docOrigin = filepath.Base(opts.FromDocPath)
		if docOrigin == "" || docOrigin == "." {
			docOrigin = "ref-doc"
		}
		docItem := sddcontext.Item{
			Source: "file",
			Origin: docOrigin,
			Name:   "ref-doc",
			Task:   id,
			Format: docFormat,
			Data:   opts.FromDocData,
		}
		entry, err := sddcontext.Stage(root, docItem, now)
		if err != nil {
			return Result{}, fmt.Errorf("feature: stage doc context: %w", err)
		}
		docStagedPath = entry.Path
		docFormat = entry.Format
	}

	templatesToRender := make([]string, 0, len(requiredTemplateNames)+len(optionalTemplateNames))
	templatesToRender = append(templatesToRender, requiredTemplateNames...)
	for _, name := range optionalTemplateNames {
		rel := "features/_templates/" + name
		if _, err := rootHandle.Stat(filepath.FromSlash(rel)); err == nil {
			templatesToRender = append(templatesToRender, name)
		}
	}

	rendered := make(map[string]string, len(templatesToRender))
	for _, name := range templatesToRender {
		rel := "features/_templates/" + name
		tpl, err := rootHandle.ReadFile(filepath.FromSlash(rel))
		if err != nil {
			return Result{}, fmt.Errorf("template %s is missing (run `sdd init`, or restore it): %w", rel, err)
		}
		text := strings.ReplaceAll(string(tpl), "T-xxx", id)
		text = strings.ReplaceAll(text, "<title>", title)
		if name == "requirements.md" {
			if jiraIssue != nil {
				text = jira.InjectRequirements(text, *jiraIssue)
			}
			if docStagedPath != "" {
				text = doc.InjectDocReference(text, docStagedPath, docOrigin, docFormat)
			}
		}
		rendered[name] = text
	}

	dirRel := "features/" + id + "-" + slug
	res := Result{Dir: dirRel, StagedContext: docStagedPath}
	for _, name := range templatesToRender {
		rel := dirRel + "/" + name
		if err := fsutil.WriteAtomicRoot(rootHandle, filepath.FromSlash(rel), []byte(rendered[name]), 0o644); err != nil {
			return res, err
		}
		res.Files = append(res.Files, rel)
	}

	if updated, ok := roadmap.SetStatus(roadmapText, id, "Not started", "Specifying"); ok {
		if err := fsutil.WriteAtomicRoot(rootHandle, "ROADMAP.md", []byte(updated), 0o644); err != nil {
			return res, err
		}
		res.StatusUpdated = true
	}

	if len(opts.FromJiraData) > 0 {
		now := opts.Now
		if now.IsZero() {
			now = time.Now()
		}
		origin := opts.JiraOrigin
		if origin == "" && jiraIssue != nil {
			origin = jiraIssue.Key
		}
		if origin == "" {
			origin = "jira"
		}
		item := sddcontext.Item{
			Source: "file",
			Origin: origin,
			Name:   "jira-issue",
			Task:   id,
			Format: "jira",
			Data:   opts.FromJiraData,
		}
		entry, err := sddcontext.Stage(root, item, now)
		if err != nil {
			return res, fmt.Errorf("feature: stage jira context: %w", err)
		}
		res.StagedContext = entry.Path
	}

	return res, nil
}
