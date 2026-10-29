// Package check verifies, mechanically, that a project's spec-driven kit is
// internally consistent: required files exist, agent contracts match, every
// decision and task reference resolves, tables and code fences are intact,
// and task statuses agree with the feature folders.
package check

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/vikashkumar21/sdd/internal/block"
	"github.com/vikashkumar21/sdd/internal/roadmap"
	"github.com/vikashkumar21/sdd/internal/scaffold"
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
	Severity Severity
	File     string
	Line     int // 0 when not applicable
	Message  string
}

// Report is the result of Run.
type Report struct{ Findings []Finding }

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

var (
	decisionRowRe = regexp.MustCompile(`(?m)^\|\s*(D-\d{3})\s*\|`)
	decisionRefRe = regexp.MustCompile(`\bD-\d{3}\b`)
	taskRefRe     = regexp.MustCompile(`\bT-\d{3}\b`)
	featureDirRe  = regexp.MustCompile(`^(T-\d{3})-[a-z0-9]+(?:-[a-z0-9]+)*$`)
	feedbackRe    = regexp.MustCompile(`(?m)^##\s+.*Human feedback rules`)
	approvalRe    = regexp.MustCompile(`Approved by:\s*_{3,}`)
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
	return Report{Findings: c.findings}, nil
}

type checker struct {
	root     string
	texts    map[string]string // slash-relative path -> content (CRLF normalised)
	findings []Finding
}

func (c *checker) add(sev Severity, file string, line int, format string, args ...any) {
	c.findings = append(c.findings, Finding{Severity: sev, File: file, Line: line, Message: fmt.Sprintf(format, args...)})
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
		c.add(Error, scaffold.ManifestPath, 0, "%v", mErr)
	case hasManifest:
		if manifest.Schema != 1 {
			c.add(Error, scaffold.ManifestPath, 0, "unsupported manifest schema %d (this sdd understands 1)", manifest.Schema)
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
	if hasManifest && mErr == nil {
		c.checkManifestDrift(manifest)
	}
	c.checkOptionalAgents(agents)
}

func (c *checker) requireFiles(agents []string) {
	required := append(append(append([]string{}, agents...), coreDocs...), templateFiles...)
	for _, rel := range required {
		if _, ok := c.load(rel); !ok {
			c.add(Error, rel, 0, "required file is missing (run `sdd init`, or restore it)")
		}
	}
}

func (c *checker) checkReadme() {
	if _, ok := c.load("README.md"); !ok {
		c.add(Warn, "README.md", 0, "README.md is missing")
	}
}

func (c *checker) checkOptionalAgents(required []string) {
	have := map[string]bool{}
	for _, r := range required {
		have[r] = true
	}
	if _, ok := c.load("AGENTS.md"); !ok && !have["AGENTS.md"] {
		c.add(Info, "AGENTS.md", 0, "AGENTS.md is not present (optional unless the manifest lists it)")
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
	if strings.Count(text, "```")%2 != 0 {
		c.add(Error, rel, 0, "unbalanced code fence (odd number of ``` markers)")
	}
}

func (c *checker) checkTables(rel, text string) {
	inFence := false
	var block []string
	var lineNos []int
	flush := func() {
		if len(block) >= 2 {
			want := strings.Count(block[0], "|")
			for i, l := range block {
				if strings.Count(l, "|") != want {
					c.add(Error, rel, lineNos[i], "table row has %d pipes, header has %d (a cell may contain a literal pipe)", strings.Count(l, "|"), want)
				}
			}
		}
		block, lineNos = nil, nil
	}
	for i, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
		}
		if !inFence && strings.HasPrefix(line, "|") {
			block = append(block, line)
			lineNos = append(lineNos, i+1)
			continue
		}
		flush()
	}
	flush()
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
						c.add(Error, rel, i+1, "%s is referenced but has no row in the Decision register (TECH_STACK.md)", id)
					}
				}
			}
			if haveRoad {
				for _, id := range taskRefRe.FindAllString(line, -1) {
					if !tasks[id] && !reportedT[id] {
						reportedT[id] = true
						c.add(Error, rel, i+1, "%s is referenced but has no task row in ROADMAP.md", id)
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
		lines[i] = strings.TrimRight(lines[i], " \t")
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
		c.add(Error, f, line+1, "agent contract differs from %s (first difference at contract line %d); the rules must be identical in every agent file", files[0], line+1)
	}
}

func (c *checker) checkSkills() {
	t, ok := c.load("SKILLS.md")
	if ok && !feedbackRe.MatchString(t) {
		c.add(Error, "SKILLS.md", 0, `no "Human feedback rules" section; the agent contract depends on it`)
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
		c.add(Warn, "", 0, "%d unresolved %s placeholder(s) (%s)", total, scaffold.Todo, strings.Join(parts, ", "))
	}
}

func (c *checker) checkRoadmap() []roadmap.Task {
	t, ok := c.load("ROADMAP.md")
	if !ok {
		return nil
	}
	tasks := roadmap.Parse(t)
	if len(tasks) == 0 {
		c.add(Warn, "ROADMAP.md", 0, "no task rows found (rows must start with | T-nnn |)")
	}
	seen := map[string]bool{}
	for _, task := range tasks {
		if seen[task.ID] {
			c.add(Error, "ROADMAP.md", task.Line, "duplicate task ID %s", task.ID)
		}
		seen[task.ID] = true
		if !roadmap.ValidStatus(task.Status) {
			c.add(Warn, "ROADMAP.md", task.Line, "%s has an unrecognised status %q", task.ID, roadmap.Normalize(task.Status))
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
			c.add(Error, rel, 0, "feature folder name must look like T-001-short-slug (lower-case, hyphens)")
			continue
		}
		id := m[1]
		if prev, dup := folders[id]; dup {
			c.add(Error, rel, 0, "%s already has a feature folder (%s)", id, prev)
		}
		folders[id] = e.Name()
		if len(tasks) > 0 {
			if _, ok := status[id]; !ok {
				c.add(Error, rel, 0, "%s has no task row in ROADMAP.md", id)
			}
		}
		for _, f := range []string{"requirements.md", "plan.md", "validation.md"} {
			if _, ok := c.load(rel + "/" + f); !ok {
				c.add(Error, rel+"/"+f, 0, "feature folder is missing %s", f)
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
			c.add(Error, "ROADMAP.md", task.Line, "%s is %q but has no features/%s-<slug>/ folder", task.ID, roadmap.Normalize(task.Status), task.ID)
		case "done":
			c.add(Warn, "ROADMAP.md", task.Line, "%s is Done but has no features/%s-<slug>/ folder", task.ID, task.ID)
		}
	}
}

func (c *checker) checkGateDiscipline(rel, kind string) {
	switch kind {
	case "implementing", "awaiting review", "done":
		if plan, ok := c.load(rel + "/plan.md"); ok && approvalRe.MatchString(plan) {
			c.add(Warn, rel+"/plan.md", 0, "task is past plan approval but the approval record is still blank")
		}
	}
	switch kind {
	case "awaiting review", "done":
		if val, ok := c.load(rel + "/validation.md"); ok && strings.Contains(val, "NOT RUN") {
			c.add(Warn, rel+"/validation.md", 0, "task is awaiting review or done but validation.md still contains NOT RUN")
		}
	}
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
		c.add(Info, scaffold.ManifestPath, 0, "%d generated file(s) have changed since sdd generated them (expected: you edit them)", modified)
	}
}
