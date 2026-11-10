// Package feature creates the per-task folder (requirements, plan, validation, review)
// from the project's own templates (or embedded fallbacks), supporting features,
// bugfixes, hotfixes, and spikes, so that Phase A starts deterministically every time.
package feature

import (
	"errors"
	"fmt"
	"io/fs"
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
	"github.com/vikashkumar21/sdd/internal/templates"
)

var (
	idRe   = regexp.MustCompile(`^T-\d{3}$`)
	slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

// Typology defines the work type.
type Typology string

const (
	TypologyFeature Typology = "feature"
	TypologyBugfix  Typology = "bugfix"
	TypologyHotfix  Typology = "hotfix"
	TypologySpike   Typology = "spike"
)

// AllTypologies lists the canonical supported typologies.
var AllTypologies = []Typology{
	TypologyFeature,
	TypologyBugfix,
	TypologyHotfix,
	TypologySpike,
}

// ParseTypology parses and normalizes a typology string.
func ParseTypology(s string) (Typology, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "feature", "features", "":
		return TypologyFeature, nil
	case "bugfix", "bugfixes", "fix", "bugs", "bug":
		return TypologyBugfix, nil
	case "hotfix", "hotfixes":
		return TypologyHotfix, nil
	case "spike", "spikes":
		return TypologySpike, nil
	default:
		return "", fmt.Errorf("unknown task typology %q (expected feature, bugfix, hotfix, or spike)", s)
	}
}

// Directory returns the canonical directory for this typology under the unified tasks/ hierarchy.
func (t Typology) Directory() string {
	switch t {
	case TypologyBugfix:
		return "tasks/bugfixes"
	case TypologyHotfix:
		return "tasks/hotfixes"
	case TypologySpike:
		return "tasks/spikes"
	default:
		return "tasks/features"
	}
}

// LegacyDirectory returns the root-level directory for this typology used prior to v1.4.0.
func (t Typology) LegacyDirectory() string {
	switch t {
	case TypologyBugfix:
		return "bugfixes"
	case TypologyHotfix:
		return "hotfixes"
	case TypologySpike:
		return "spikes"
	default:
		return "features"
	}
}

// DirectoryIn returns the typology directory to use in the given root project.
// If tasks/ or tasks/<typology> exists, or if neither exists, it returns tasks/<typology>.
// If legacy root <typology> exists and tasks/ does not exist, it returns the legacy root path.
func (t Typology) DirectoryIn(root string) string {
	canonical := t.Directory()
	legacy := t.LegacyDirectory()
	if root == "" {
		return canonical
	}
	tasksDir := filepath.Join(root, "tasks")
	if info, err := os.Stat(tasksDir); err == nil && info.IsDir() {
		return canonical
	}
	legacyDir := filepath.Join(root, legacy)
	if info, err := os.Stat(legacyDir); err == nil && info.IsDir() {
		return legacy
	}
	return canonical
}

// RequiredTemplates returns the mandatory template names for this typology.
func (t Typology) RequiredTemplates() []string {
	switch t {
	case TypologyBugfix:
		return []string{"reproduction.md", "rca.md", "plan.md", "validation.md"}
	case TypologyHotfix:
		return []string{"incident.md", "patch.md", "validation.md", "post_mortem.md"}
	case TypologySpike:
		return []string{"hypotheses.md", "findings.md", "plan.md"}
	default:
		return []string{"requirements.md", "plan.md", "validation.md"}
	}
}

// OptionalTemplates returns optional templates (e.g. review.md) for this typology.
func (t Typology) OptionalTemplates() []string {
	switch t {
	case TypologyFeature, TypologyBugfix:
		return []string{"review.md"}
	default:
		return nil
	}
}

// Result describes what New created.
type Result struct {
	Dir           string   // relative, slash-separated
	Files         []string // relative, slash-separated
	StatusUpdated bool     // the ROADMAP status changed from Not started to Specifying
	StagedContext string   // relative path to staged context artifact if any
}

// Options provides optional parameters when creating a new task folder.
type Options struct {
	Type         Typology
	Title        string
	FromJiraData []byte
	JiraOrigin   string    // filename or label of Jira source
	FromDocData  []byte    // raw document or PDF bytes
	FromDocPath  string    // original filename / path of document
	Now          time.Time // clock injection for staging context (defaults to time.Now())
}

// New creates features/<id>-<slug>/ with the standard feature documents.
func New(root, id, slug, title string) (Result, error) {
	return NewWithOptions(root, id, slug, Options{Title: title, Type: TypologyFeature})
}

// NewWithOptions creates a task folder with options including typology and ingestion.
func NewWithOptions(root, id, slug string, opts Options) (Result, error) {
	if !idRe.MatchString(id) {
		return Result{}, fmt.Errorf("task ID %q must look like T-001", id)
	}
	if id == "T-000" {
		return Result{}, errors.New("the bootstrap task T-000 has no task folder")
	}
	if !slugRe.MatchString(slug) {
		return Result{}, fmt.Errorf("slug %q must be lower-case words joined by hyphens (for example project-skeleton)", slug)
	}

	typ := opts.Type
	if typ == "" {
		typ = TypologyFeature
	}

	var jiraIssue *jira.Issue
	if len(opts.FromJiraData) > 0 {
		iss, err := jira.Parse(opts.FromJiraData)
		if err != nil {
			return Result{}, fmt.Errorf("task: parse jira issue: %w", err)
		}
		jiraIssue = &iss
	}

	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return Result{}, fmt.Errorf("task: open root %s: %w", root, err)
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

	// Check if this task already has a folder in ANY typology directory (tasks/ or legacy root)
	allDirs := []string{
		"tasks/features", "tasks/bugfixes", "tasks/hotfixes", "tasks/spikes",
		"features", "bugfixes", "hotfixes", "spikes",
	}
	for _, d := range allDirs {
		if f, err := rootHandle.Open(filepath.FromSlash(d)); err == nil {
			entries, readErr := f.ReadDir(-1)
			_ = f.Close()
			if readErr == nil {
				for _, e := range entries {
					if e.IsDir() && strings.HasPrefix(e.Name(), id+"-") {
						return Result{}, fmt.Errorf("%s already has a %s folder: %s/%s", id, d, d, e.Name())
					}
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
				return Result{}, fmt.Errorf("task: %w", err)
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
			return Result{}, fmt.Errorf("task: stage doc context: %w", err)
		}
		docStagedPath = entry.Path
		docFormat = entry.Format
	}

	topDir := typ.DirectoryIn(root)
	tmplDir := topDir + "/_templates"
	reqTemplates := typ.RequiredTemplates()
	optTemplates := typ.OptionalTemplates()

	// Check if project has project-level template override directory
	hasProjectTmplDir := false
	if _, err := rootHandle.Stat(tmplDir); err == nil {
		hasProjectTmplDir = true
	}

	templatesToRender := make([]string, 0, len(reqTemplates)+len(optTemplates))
	templatesToRender = append(templatesToRender, reqTemplates...)

	set, _ := templates.New("")

	for _, name := range optTemplates {
		rel := tmplDir + "/" + name
		if _, err := rootHandle.Stat(filepath.FromSlash(rel)); err == nil {
			templatesToRender = append(templatesToRender, name)
		} else if !hasProjectTmplDir && set != nil {
			// Only fall back to embedded if the project doesn't have a template directory for this typology yet
			if _, err := set.Raw(topDir + "/" + name); err == nil {
				templatesToRender = append(templatesToRender, name)
			}
		}
	}

	rendered := make(map[string]string, len(templatesToRender))
	for _, name := range templatesToRender {
		rel := tmplDir + "/" + name
		tpl, err := rootHandle.ReadFile(filepath.FromSlash(rel))
		var content string
		if err == nil {
			content = string(tpl)
		} else if errors.Is(err, fs.ErrNotExist) && !hasProjectTmplDir && set != nil {
			// Fallback to embedded template ONLY if project does not have tmplDir for this typology
			embeddedContent, embErr := set.Raw(topDir + "/" + name)
			if embErr != nil {
				return Result{}, fmt.Errorf("template %s is missing: %w", rel, embErr)
			}
			content = embeddedContent
		} else {
			return Result{}, fmt.Errorf("template %s is missing (run `sdd init`, or restore it): %w", rel, err)
		}

		text := strings.ReplaceAll(content, "T-xxx", id)
		text = strings.ReplaceAll(text, "<title>", title)

		if name == "requirements.md" || name == "reproduction.md" {
			if jiraIssue != nil {
				text = jira.InjectRequirements(text, *jiraIssue)
			}
			if docStagedPath != "" {
				text = doc.InjectDocReference(text, docStagedPath, docOrigin, docFormat)
			}
		}
		rendered[name] = text
	}

	dirRel := topDir + "/" + id + "-" + slug
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
			return res, fmt.Errorf("task: stage jira context: %w", err)
		}
		res.StagedContext = entry.Path
	}

	return res, nil
}
