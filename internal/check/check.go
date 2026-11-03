// Package check verifies, mechanically, that a project's spec-driven kit is
// internally consistent: required files exist, agent contracts match, every
// decision and task reference resolves, tables and code fences are intact,
// and task statuses agree with the feature folders.
package check

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/vikashkumar21/sdd/internal/block"
	"github.com/vikashkumar21/sdd/internal/context"
	"github.com/vikashkumar21/sdd/internal/markdown"
	"github.com/vikashkumar21/sdd/internal/roadmap"
	"github.com/vikashkumar21/sdd/internal/scaffold"
	"github.com/vikashkumar21/sdd/internal/suppress"
)

// Severity of a finding.
type Severity string

// Severities, from most to least serious.
const (
	Error Severity = "ERROR"
	Warn  Severity = "WARN"
	Info  Severity = "INFO"
)

// Finding is one problem (or note) found by Run.
type Finding struct {
	Severity       Severity `json:"severity"`
	File           string   `json:"file"`
	Line           int      `json:"line,omitempty"` // 0 when not applicable
	Message        string   `json:"message"`
	Rule           string   `json:"rule,omitempty"`
	Suppressed     bool     `json:"suppressed,omitempty"`
	SuppressReason string   `json:"suppress_reason,omitempty"`
}

// Summary provides aggregate counts by severity.
type Summary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Notes    int `json:"notes"`
}

// JSONReport is the machine-readable output structure for sdd check.
type JSONReport struct {
	Valid    bool      `json:"valid"`
	Summary  Summary   `json:"summary"`
	Findings []Finding `json:"findings"`
}

// Report is the result of Run.
type Report struct {
	Findings     []Finding
	StaleWaivers []suppress.Entry
}

// Count returns how many findings have the given severity.
func (r Report) Count(s Severity) int {
	n := 0
	for _, f := range r.Findings {
		if f.Severity == s {
			n++
		}
	}
	return n
}

// Failed reports whether the check should fail: any error, or any warning in
// strict mode.
func (r Report) Failed(strict bool) bool {
	return r.Count(Error) > 0 || (strict && r.Count(Warn) > 0)
}

// JSON renders the report as indented, machine-readable JSON.
func (r Report) JSON(strict bool) ([]byte, error) {
	findings := r.Findings
	if findings == nil {
		findings = []Finding{}
	}
	jr := JSONReport{
		Valid: !r.Failed(strict),
		Summary: Summary{
			Errors:   r.Count(Error),
			Warnings: r.Count(Warn),
			Notes:    r.Count(Info),
		},
		Findings: findings,
	}
	return json.MarshalIndent(jr, "", "  ")
}

var (
	decisionRowRe  = regexp.MustCompile(`(?m)^\|\s*(D-\d{3})\s*\|`)
	decisionRefRe  = regexp.MustCompile(`\bD-\d{3}\b`)
	taskRefRe      = regexp.MustCompile(`\bT-\d{3}\b`)
	featureDirRe   = regexp.MustCompile(`^(T-\d{3})-[a-z0-9]+(?:-[a-z0-9]+)*$`)
	feedbackRe     = regexp.MustCompile(`(?m)^##\s+.*Human feedback rules`)
	approvalRe     = regexp.MustCompile(`Approved by:\s*_{3,}`)
	factHdrRe      = regexp.MustCompile(`(?i)^\|\s*Date\s*\|\s*Fact\s*\|\s*Source\s*\|\s*Re-verify at\s*\|`)
	milestoneRefRe = regexp.MustCompile(`\bM\d+\b`)
	statusHeaderRe = regexp.MustCompile(`(?m)^Status:\s*([^\r\n]+)`)
)

var coreDocs = []string{"MISSION.md", "TECH_STACK.md", "ROADMAP.md", "SKILLS.md"}

var templateFiles = []string{
	"features/_templates/requirements.md", "features/_templates/plan.md", "features/_templates/validation.md",
}

// Run checks the project at root. It only reads.
func Run(root string) (Report, error) {
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return Report{}, fmt.Errorf("check: %s is not a readable directory", root)
	}
	c := &checker{root: root, texts: map[string]string{}}
	c.run()

	// Load suppressions and apply them
	var staleWaivers []suppress.Entry
	suppManifest, hasSupp, suppErr := suppress.Load(root)
	if suppErr != nil {
		c.add(Error, suppress.SuppressionsFile, 0, "suppress-syntax", "invalid suppressions manifest: %v", suppErr)
	} else if hasSupp && len(suppManifest.Suppressions) > 0 {
		matcher := suppress.NewMatcher(suppManifest)
		for i := range c.findings {
			f := &c.findings[i]
			if entry, ok := matcher.Match(f.File, f.Rule); ok {
				f.Suppressed = true
				f.SuppressReason = entry.Reason
				f.Severity = Info
				f.Message = fmt.Sprintf("%s (suppressed: %s)", f.Message, entry.Reason)
			}
		}
		staleWaivers = matcher.Stale()
		for _, stale := range staleWaivers {
			c.add(Info, suppress.SuppressionsFile, 0, "stale-waiver", "stale waiver for %s (%s) is no longer triggered", stale.File, stale.Rule)
		}
	}

	sort.SliceStable(c.findings, func(i, j int) bool {
		a, b := c.findings[i], c.findings[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Message < b.Message
	})
	return Report{Findings: c.findings, StaleWaivers: staleWaivers}, nil
}

type checker struct {
	root     string
	texts    map[string]string // slash-relative path -> content (CRLF normalised)
	findings []Finding
}

func (c *checker) add(sev Severity, file string, line int, rule string, format string, args ...any) {
	c.findings = append(c.findings, Finding{
		Severity: sev,
		File:     file,
		Line:     line,
		Rule:     rule,
		Message:  fmt.Sprintf(format, args...),
	})
}

func (c *checker) load(rel string) (string, bool) {
	if t, ok := c.texts[rel]; ok {
		return t, true
	}
	data, err := os.ReadFile(filepath.Join(c.root, filepath.FromSlash(rel)))
	if err != nil {
		return "", false
	}
	t := strings.ReplaceAll(string(data), "\r\n", "\n")
	c.texts[rel] = t
	return t, true
}

func (c *checker) run() {
	manifest, hasManifest, mErr := scaffold.ReadManifest(c.root)
	agents := []string{"CLAUDE.md", "GEMINI.md"}
	switch {
	case mErr != nil:
		c.add(Error, scaffold.ManifestPath, 0, "manifest-syntax", "%v", mErr)
	case hasManifest:
		if manifest.Schema != 1 {
			c.add(Error, scaffold.ManifestPath, 0, "manifest-schema", "unsupported manifest schema %d (this sdd understands 1)", manifest.Schema)
		}
		agents = nil
		for _, id := range manifest.Answers.Agents {
			agents = append(agents, scaffold.AgentFile(id))
		}
	}

	c.requireFiles(agents)
	c.checkReadme()
	docs := c.markdownFiles()
	for _, rel := range docs {
		text, ok := c.load(rel)
		if !ok {
			continue
		}
		c.checkFences(rel, text)
		c.checkTables(rel, text)
	}
	c.checkReferences(docs)
	c.checkParity(agents)
	c.checkSkills()
	c.checkTodos(append(append([]string{}, coreDocs...), agents...))
	tasks := c.checkRoadmap()
	c.checkFeatures(tasks)
	c.checkVerifiedFactsExpiry(tasks)
	if hasManifest && mErr == nil {
		c.checkManifestDrift(manifest)
	}
	c.checkOptionalAgents(agents)
	c.checkContextManifest()
	c.checkPermissions()
	c.checkAgentSkills()
}

func (c *checker) checkPermissions() {
	claudePath := filepath.Join(c.root, ".claude", "settings.json")
	if data, err := os.ReadFile(claudePath); err == nil {
		var obj map[string]any
		if err := json.Unmarshal(data, &obj); err != nil {
			c.add(Error, ".claude/settings.json", 0, "permissions-syntax", "invalid JSON in Claude settings: %v", err)
		}
	}
}

func (c *checker) checkContextManifest() {
	m, ok, err := context.ReadManifest(c.root)
	if !ok {
		return
	}
	if err != nil {
		c.add(Error, context.ManifestPath, 0, "context-syntax", "%v", err)
		return
	}
	for _, entry := range m.Entries {
		if entry.Path == "" {
			c.add(Error, context.ManifestPath, 0, "context-entry", "entry %s has empty path", entry.ID)
			continue
		}
		path := filepath.Join(c.root, filepath.FromSlash(entry.Path))
		if _, err := os.Stat(path); err != nil {
			c.add(Error, context.ManifestPath, 0, "context-missing", "staged context file %s does not exist: %s", entry.ID, entry.Path)
		}
	}
}

func (c *checker) requireFiles(agents []string) {
	required := append(append(append([]string{}, agents...), coreDocs...), templateFiles...)
	for _, rel := range required {
		if _, ok := c.load(rel); !ok {
			c.add(Error, rel, 0, "required-file", "required file is missing (run `sdd init`, or restore it)")
		}
	}
}

func (c *checker) checkReadme() {
	if _, ok := c.load("README.md"); !ok {
		c.add(Warn, "README.md", 0, "required-file", "README.md is missing")
	}
}

func (c *checker) checkOptionalAgents(required []string) {
	have := map[string]bool{}
	for _, r := range required {
		have[r] = true
	}
	if _, ok := c.load("AGENTS.md"); !ok && !have["AGENTS.md"] {
		c.add(Info, "AGENTS.md", 0, "optional-file", "AGENTS.md is not present (optional unless the manifest lists it)")
	}
}

// markdownFiles returns the root-level .md files and every .md file under
// features/ except the templates, in sorted order.
func (c *checker) markdownFiles() []string {
	var out []string
	entries, err := os.ReadDir(c.root)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				out = append(out, e.Name())
			}
		}
	}
	featuresDir := filepath.Join(c.root, "features")
	_ = filepath.WalkDir(featuresDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, rerr := filepath.Rel(c.root, path)
		if rerr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() && rel == "features/_templates" {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(rel, ".md") {
			out = append(out, rel)
		}
		return nil
	})
	sort.Strings(out)
	return out
}

func (c *checker) checkFences(rel, text string) {
	for _, f := range markdown.CheckFences(text) {
		c.add(Error, rel, f.Line, "code-fence", "unbalanced code fence: %s", f.Message)
	}
}

func (c *checker) checkTables(rel, text string) {
	tables := markdown.ExtractTables(text)
	for _, t := range tables {
		want := t.Header.Columns
		for _, r := range t.Rows {
			if r.Columns != want {
				c.add(Error, rel, r.LineNo, "table-format", "table row has %d columns, header has %d (pipes mismatch or unescaped literal pipe)", r.Columns, want)
			}
		}
	}
}

func (c *checker) checkReferences(docs []string) {
	decisions := map[string]bool{}
	if t, ok := c.load("TECH_STACK.md"); ok {
		for _, m := range decisionRowRe.FindAllStringSubmatch(t, -1) {
			decisions[m[1]] = true
		}
	}
	tasks := map[string]bool{}
	if t, ok := c.load("ROADMAP.md"); ok {
		for _, task := range roadmap.Parse(t) {
			tasks[task.ID] = true
			tasks[strings.SplitN(task.ID, ".", 2)[0]] = true
		}
	}
	_, haveStack := c.load("TECH_STACK.md")
	_, haveRoad := c.load("ROADMAP.md")
	reportedD, reportedT := map[string]bool{}, map[string]bool{}
	for _, rel := range docs {
		text, ok := c.load(rel)
		if !ok {
			continue
		}
		for i, line := range strings.Split(text, "\n") {
			if haveStack {
				for _, id := range decisionRefRe.FindAllString(line, -1) {
					if !decisions[id] && !reportedD[id] {
						reportedD[id] = true
						c.add(Error, rel, i+1, "unresolved-decision", "%s is referenced but has no row in the Decision register (TECH_STACK.md)", id)
					}
				}
			}
			if haveRoad {
				for _, id := range taskRefRe.FindAllString(line, -1) {
					if !tasks[id] && !reportedT[id] {
						reportedT[id] = true
						c.add(Error, rel, i+1, "unresolved-task", "%s is referenced but has no task row in ROADMAP.md", id)
					}
				}
			}
		}
	}
}

// contractText returns the text that must be identical across agent files:
// the managed block when there is one, otherwise everything after the title
// line (for kits written before managed blocks existed).
func contractText(content string) string {
	text, ok := block.Inner(content)
	if !ok {
		if i := strings.Index(content, "\n"); i >= 0 {
			text = content[i+1:]
		}
	}
	lines := strings.Split(text, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \r\t")
	}
	return strings.Trim(strings.Join(lines, "\n"), "\n")
}

func (c *checker) checkParity(required []string) {
	seen := map[string]bool{}
	var files []string
	for _, f := range append(append([]string{}, required...), "CLAUDE.md", "GEMINI.md", "AGENTS.md") {
		if seen[f] {
			continue
		}
		seen[f] = true
		if _, ok := c.load(f); ok {
			files = append(files, f)
		}
	}
	if len(files) < 2 {
		return
	}
	base, _ := c.load(files[0])
	baseText := contractText(base)
	for _, f := range files[1:] {
		other, _ := c.load(f)
		otherText := contractText(other)
		if otherText == baseText {
			continue
		}
		a, b := strings.Split(baseText, "\n"), strings.Split(otherText, "\n")
		line := 0
		for line < len(a) && line < len(b) && a[line] == b[line] {
			line++
		}
		c.add(Error, f, line+1, "contract-parity", "agent contract differs from %s (first difference at contract line %d); the rules must be identical in every agent file", files[0], line+1)
	}
}

func (c *checker) checkSkills() {
	t, ok := c.load("SKILLS.md")
	if ok && !feedbackRe.MatchString(t) {
		c.add(Error, "SKILLS.md", 0, "human-feedback-rules", `no "Human feedback rules" section; the agent contract depends on it`)
	}
}

func (c *checker) checkTodos(files []string) {
	total := 0
	var parts []string
	for _, f := range files {
		t, ok := c.load(f)
		if !ok {
			continue
		}
		if n := strings.Count(t, scaffold.Todo); n > 0 {
			total += n
			parts = append(parts, fmt.Sprintf("%s: %d", f, n))
		}
	}
	if total > 0 {
		c.add(Warn, "", 0, "todo-placeholder", "%d unresolved %s placeholder(s) (%s)", total, scaffold.Todo, strings.Join(parts, ", "))
	}
}

func (c *checker) checkRoadmap() []roadmap.Task {
	t, ok := c.load("ROADMAP.md")
	if !ok {
		return nil
	}
	tasks := roadmap.Parse(t)
	if len(tasks) == 0 {
		c.add(Warn, "ROADMAP.md", 0, "roadmap-drift", "no task rows found (rows must start with | T-nnn |)")
	}
	taskMap := map[string]roadmap.Task{}
	for _, task := range tasks {
		taskMap[task.ID] = task
	}
	seen := map[string]bool{}
	for _, task := range tasks {
		if seen[task.ID] {
			c.add(Error, "ROADMAP.md", task.Line, "roadmap-drift", "duplicate task ID %s", task.ID)
		}
		seen[task.ID] = true
		if !roadmap.ValidStatus(task.Status) {
			c.add(Warn, "ROADMAP.md", task.Line, "roadmap-drift", "%s has an unrecognised status %q", task.ID, roadmap.Normalize(task.Status))
		}
		kind := roadmap.Kind(task.Status)
		if kind == "implementing" || kind == "awaiting review" || kind == "done" {
			for _, depID := range task.DependsOn {
				dep, exists := taskMap[depID]
				if !exists {
					continue
				}
				depKind := roadmap.Kind(dep.Status)
				if depKind == "not started" || depKind == "specifying" || depKind == "blocked" {
					c.add(Warn, "ROADMAP.md", task.Line, "roadmap-drift", "%s is %s but prerequisite %s is %s",
						task.ID, roadmap.Normalize(task.Status), depID, roadmap.Normalize(dep.Status))
				}
			}
		}
	}
	return tasks
}

func (c *checker) checkFeatures(tasks []roadmap.Task) {
	status := map[string]string{}
	for _, task := range tasks {
		if _, dup := status[task.ID]; !dup {
			status[task.ID] = roadmap.Kind(task.Status)
		}
	}
	folders := map[string]string{} // task ID -> folder name
	entries, _ := os.ReadDir(filepath.Join(c.root, "features"))
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "_templates" {
			continue
		}
		rel := "features/" + e.Name()
		m := featureDirRe.FindStringSubmatch(e.Name())
		if m == nil {
			c.add(Error, rel, 0, "feature-folder", "feature folder name must look like T-001-short-slug (lower-case, hyphens)")
			continue
		}
		id := m[1]
		if prev, dup := folders[id]; dup {
			c.add(Error, rel, 0, "feature-folder", "%s already has a feature folder (%s)", id, prev)
		}
		folders[id] = e.Name()
		if len(tasks) > 0 {
			if _, ok := status[id]; !ok {
				c.add(Error, rel, 0, "feature-folder", "%s has no task row in ROADMAP.md", id)
			}
		}
		for _, f := range []string{"requirements.md", "plan.md", "validation.md"} {
			if _, ok := c.load(rel + "/" + f); !ok {
				c.add(Error, rel+"/"+f, 0, "feature-folder", "feature folder is missing %s", f)
			}
		}
		c.checkGateDiscipline(rel, status[id])
	}
	for _, task := range tasks {
		base := strings.SplitN(task.ID, ".", 2)[0]
		if base == "T-000" || strings.Contains(task.ID, ".") {
			continue
		}
		if _, has := folders[task.ID]; has {
			continue
		}
		switch roadmap.Kind(task.Status) {
		case "specifying", "awaiting plan approval", "implementing", "awaiting review":
			c.add(Error, "ROADMAP.md", task.Line, "feature-folder", "%s is %q but has no features/%s-<slug>/ folder", task.ID, roadmap.Normalize(task.Status), task.ID)
		case "done":
			c.add(Warn, "ROADMAP.md", task.Line, "feature-folder", "%s is Done but has no features/%s-<slug>/ folder", task.ID, task.ID)
		}
	}
}

func (c *checker) checkGateDiscipline(rel, kind string) {
	switch kind {
	case "implementing", "awaiting review", "done":
		if plan, ok := c.load(rel + "/plan.md"); ok && approvalRe.MatchString(plan) {
			c.add(Warn, rel+"/plan.md", 0, "gate-discipline", "task is past plan approval but the approval record is still blank")
		}
	}
	switch kind {
	case "awaiting review", "done":
		if val, ok := c.load(rel + "/validation.md"); ok && strings.Contains(val, "NOT RUN") {
			c.add(Warn, rel+"/validation.md", 0, "gate-discipline", "task is awaiting review or done but validation.md still contains NOT RUN")
		}
	}
	if kind == "done" {
		if val, ok := c.load(rel + "/validation.md"); ok {
			if m := statusHeaderRe.FindStringSubmatch(val); m != nil {
				st := strings.ToLower(m[1])
				if (strings.Contains(st, "not run") || strings.Contains(st, "in progress")) && !strings.Contains(st, "complete") {
					c.add(Warn, rel+"/validation.md", 0, "gate-discipline", "task is Done in ROADMAP.md but validation.md status is %q", strings.TrimSpace(m[1]))
				}
			}
		}
		if plan, ok := c.load(rel + "/plan.md"); ok {
			if m := statusHeaderRe.FindStringSubmatch(plan); m != nil {
				st := strings.ToLower(m[1])
				if (strings.Contains(st, "draft") || strings.Contains(st, "awaiting approval")) && !strings.Contains(st, "approved") && !strings.Contains(st, "implemented") {
					c.add(Warn, rel+"/plan.md", 0, "gate-discipline", "task is Done in ROADMAP.md but plan.md status is %q", strings.TrimSpace(m[1]))
				}
			}
		}
	}
}

func (c *checker) checkVerifiedFactsExpiry(tasks []roadmap.Task) {
	text, ok := c.load("TECH_STACK.md")
	if !ok {
		return
	}
	taskMap := map[string]roadmap.Task{}
	milestoneTasks := map[string][]roadmap.Task{}
	for _, t := range tasks {
		taskMap[t.ID] = t
		if t.Milestone != "" {
			milestoneTasks[t.Milestone] = append(milestoneTasks[t.Milestone], t)
		}
	}

	completedMilestones := map[string]bool{}
	for m, mTasks := range milestoneTasks {
		if len(mTasks) == 0 {
			continue
		}
		allDone := true
		for _, t := range mTasks {
			if roadmap.Kind(t.Status) != "done" {
				allDone = false
				break
			}
		}
		if allDone {
			completedMilestones[m] = true
		}
	}

	lines := strings.Split(text, "\n")
	inFactsTable := false
	for i, line := range lines {
		lineNo := i + 1
		trimmed := strings.TrimSpace(line)
		if factHdrRe.MatchString(trimmed) {
			inFactsTable = true
			continue
		}
		if inFactsTable {
			if !strings.HasPrefix(trimmed, "|") {
				inFactsTable = false
				continue
			}
			if strings.Contains(trimmed, "---") {
				continue
			}
			cells := splitRow(trimmed)
			if len(cells) < 4 {
				continue
			}
			fact := cells[1]
			reVerifyAt := cells[3]
			if fact == "" || strings.HasPrefix(fact, "(") {
				continue
			}

			// Check task references
			for _, tID := range taskRefRe.FindAllString(reVerifyAt, -1) {
				if t, exists := taskMap[tID]; exists {
					if roadmap.Kind(t.Status) == "done" {
						c.add(Warn, "TECH_STACK.md", lineNo, "fact-expiry",
							"verified fact %q is past due for re-verification (%s is Done)", truncate(fact, 50), tID)
					}
				}
			}

			// Check milestone references
			for _, mID := range milestoneRefRe.FindAllString(reVerifyAt, -1) {
				if completedMilestones[strings.ToUpper(mID)] {
					c.add(Warn, "TECH_STACK.md", lineNo, "fact-expiry",
						"verified fact %q is past due for re-verification (milestone %s is completed)", truncate(fact, 50), mID)
				}
			}
		}
	}
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func splitRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func (c *checker) checkManifestDrift(m scaffold.Manifest) {
	modified := 0
	for _, f := range m.Files {
		if f.State == "existing" || f.SHA256 == "" {
			continue
		}
		text, ok := c.load(f.Path)
		if !ok {
			continue
		}
		var current string
		if f.Kind == "block" {
			inner, has := block.Inner(text)
			if !has {
				modified++
				continue
			}
			current = inner
		} else {
			current = text
		}
		if sumHex(current) != f.SHA256 {
			modified++
		}
	}
	if modified > 0 {
		c.add(Info, scaffold.ManifestPath, 0, "manifest-modified", "%d generated file(s) have changed since sdd generated them (expected: you edit them)", modified)
	}
}
