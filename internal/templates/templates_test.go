package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	if Version != "2.10.0" {
		t.Errorf("Version = %q, want 2.10.0", Version)
	}
}

func TestTechStackTemplateDependencySections(t *testing.T) {
	set, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	content, err := set.Raw("base/TECH_STACK.md.tmpl")
	if err != nil {
		t.Fatalf("failed to read base/TECH_STACK.md.tmpl: %v", err)
	}
	if !strings.Contains(content, "### 8.1 Package and library dependencies") {
		t.Errorf("base/TECH_STACK.md.tmpl missing Section 8.1 header")
	}
	if !strings.Contains(content, "### 8.2 Cross-project and service dependencies") {
		t.Errorf("base/TECH_STACK.md.tmpl missing Section 8.2 header")
	}
	canonicalHeader := "| Name | Repo / URL | Direction | Contract / Version | Owner | Decision | Status |"
	if !strings.Contains(content, canonicalHeader) {
		t.Errorf("base/TECH_STACK.md.tmpl missing canonical 7-column header: %s", canonicalHeader)
	}
}

func TestContractTemplateCoordinatorAndWorkflow(t *testing.T) {
	set, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	content, err := set.Raw("base/contract.md.tmpl")
	if err != nil {
		t.Fatalf("failed to read base/contract.md.tmpl: %v", err)
	}

	assertions := []string{
		"| Phase | Coordinator & Subagent Actions | Output | Gate |",
		"Coordinator assumes or dispatches `architect` subagent",
		"Coordinator dispatches independent `plan-reviewer` subagent",
		"Coordinator provisions isolated worktree",
		"dispatches `implementer` subagent",
		"Coordinator dispatches `qa` subagent",
		"Spec Auditor (`spec-auditor`)",
		"AppSec Specialist (`appsec-specialist`)",
		"Pattern Auditor (`pattern-auditor`)",
		"Tier-Calibrated Subagent Dispatch",
		"standard model tier",
		"frontier model tier",
		"Two-tier agent hierarchy & role dispatch",
		".sdd/roles.json",
		"tasks/<typology>/",
	}

	for _, want := range assertions {
		if !strings.Contains(content, want) {
			t.Errorf("base/contract.md.tmpl missing expected substring %q", want)
		}
	}
}

func TestEveryLanguageHasACompletePack(t *testing.T) {
	set, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range LanguageIDs() {
		p, err := set.Pack(id)
		if err != nil {
			t.Fatalf("pack %s: %v", id, err)
		}
		if p.Title == "" || p.Standards == "" || p.Contract == "" || p.SkillMap == "" || p.ReviewStandards == "" {
			t.Errorf("pack %s has an empty part: %+v", id, p)
		}
		for _, row := range strings.Split(p.SkillMap, "\n") {
			if strings.Count(row, "|") != 5 {
				t.Errorf("pack %s skill-map row must have 4 cells: %q", id, row)
			}
		}
		for _, bad := range []string{"{{", "}}"} {
			if strings.Contains(p.Standards+p.Contract+p.SkillMap+p.ReviewStandards, bad) {
				t.Errorf("pack %s contains %q", id, bad)
			}
		}
	}
	if _, err := set.Pack("cobol"); err == nil {
		t.Error("unknown pack must be an error")
	}
}

func TestEveryEmbeddedTemplateParses(t *testing.T) {
	set, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"base/MISSION.md.tmpl", "base/TECH_STACK.md.tmpl", "base/ROADMAP.md.tmpl",
		"base/SKILLS.md.tmpl", "base/README.md.tmpl", "base/contract.md.tmpl", "base/prompt.md.tmpl",
		"features/requirements.md", "features/plan.md", "features/validation.md"} {
		if _, err := set.Raw(name); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestRenderErrors(t *testing.T) {
	over := t.TempDir()
	if err := os.MkdirAll(filepath.Join(over, "base"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, text string) {
		if err := os.WriteFile(filepath.Join(over, "base", name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("bad-syntax.tmpl", "{{ if }}")
	write("good.tmpl", "hello {{.Name}}")
	set, err := New(over)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Render("base/bad-syntax.tmpl", nil); err == nil {
		t.Error("syntax error must be reported")
	}
	if _, err := set.Render("base/missing.tmpl", nil); err == nil {
		t.Error("missing template must be reported")
	}
	out, err := set.Render("base/good.tmpl", struct{ Name string }{"sdd"})
	if err != nil || out != "hello sdd" {
		t.Errorf("render = %q, %v", out, err)
	}
	if set.Override() != over {
		t.Errorf("override = %q", set.Override())
	}
}

func TestOverrideErrors(t *testing.T) {
	if _, err := New(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("missing override dir must be an error")
	}
	f := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := New(f); err == nil {
		t.Error("override that is a file must be an error")
	}
}

func TestEmbeddedRoleSkills(t *testing.T) {
	set, err := New("")
	if err != nil {
		t.Fatal(err)
	}

	roleSkills := []string{
		"coordinator",
		"architect",
		"codebase-analyst",
		"discovery-challenger",
		"plan-reviewer",
		"spec-auditor",
		"appsec-specialist",
		"pattern-auditor",
		"implementer",
		"qa",
		"devops",
	}

	for _, name := range roleSkills {
		tmplPath := "skills/" + name + "/SKILL.md"
		content, err := set.Raw(tmplPath)
		if err != nil {
			t.Errorf("failed to read embedded role skill %s: %v", tmplPath, err)
			continue
		}
		if !strings.HasPrefix(content, "---\nname: "+name+"\n") {
			t.Errorf("embedded role skill %s missing expected frontmatter header", tmplPath)
		}
		if !strings.Contains(content, "ALWAYS BE DETERMINISTIC") {
			t.Errorf("embedded role skill %s missing SDD anthem", tmplPath)
		}
	}
}

func TestExtendedEmbeddedRoleSkills(t *testing.T) {
	set, err := New("")
	if err != nil {
		t.Fatal(err)
	}

	extendedRoles := []string{
		"implementer",
		"qa",
		"devops",
	}

	for _, name := range extendedRoles {
		tmplPath := "skills/" + name + "/SKILL.md"
		content, err := set.Raw(tmplPath)
		if err != nil {
			t.Fatalf("failed to read extended embedded role skill %s: %v", tmplPath, err)
		}
		if !strings.HasPrefix(content, "---\nname: "+name+"\n") {
			t.Errorf("extended embedded role skill %s missing expected frontmatter header", tmplPath)
		}
		if !strings.Contains(content, "ALWAYS BE DETERMINISTIC") {
			t.Errorf("extended embedded role skill %s missing SDD anthem", tmplPath)
		}
	}
}

func TestReviewTemplateLanguageStandards(t *testing.T) {
	set, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	testCases := []struct {
		lang     string
		expected string
	}{
		{"go", "Effective Go conventions, context.Context first, clean linting (make lint)"},
		{"java", "Idiomatic Java conventions, thread safety, clean build (gradle check / mvn verify)"},
		{"python", "PEP 8 / PEP 257 standards, type hints, clean linting (ruff / pytest)"},
		{"typescript", "Strict TypeScript idioms, async/await error propagation, clean linting (npm test / eslint)"},
		{"other", "Idiomatic language conventions, clean build and test suite"},
	}

	for _, tc := range testCases {
		pack, err := set.Pack(tc.lang)
		if err != nil {
			t.Fatalf("pack %s: %v", tc.lang, err)
		}
		data := struct {
			ReviewStandards string
		}{
			ReviewStandards: pack.ReviewStandards,
		}
		for _, tmpl := range []string{"tasks/features/review.md", "tasks/bugfixes/review.md"} {
			rendered, err := set.Render(tmpl, data)
			if err != nil {
				t.Fatalf("render %s for %s: %v", tmpl, tc.lang, err)
			}
			if !strings.Contains(rendered, tc.expected) {
				t.Errorf("%s for %s missing expected standards %q; got:\n%s", tmpl, tc.lang, tc.expected, rendered)
			}
		}
	}
}

func TestPromptTemplateOrder(t *testing.T) {
	set, err := New("")
	if err != nil {
		t.Fatal(err)
	}

	// 1. Verify standard prompt template (base/prompt.md.tmpl)
	promptContent, err := set.Raw("base/prompt.md.tmpl")
	if err != nil {
		t.Fatalf("failed to read base/prompt.md.tmpl: %v", err)
	}

	const marker = "<!-- sdd:cache-boundary:end-prefix -->"
	markerIdx := strings.Index(promptContent, marker)
	if markerIdx == -1 {
		t.Fatalf("base/prompt.md.tmpl missing cache boundary marker: %s", marker)
	}

	dynamicIdx := strings.Index(promptContent, "Dynamic Project State & Context")
	if dynamicIdx == -1 {
		t.Fatalf("base/prompt.md.tmpl missing Dynamic Project State & Context header")
	}

	if markerIdx > dynamicIdx {
		t.Errorf("cache boundary marker (%d) must appear before dynamic context (%d)", markerIdx, dynamicIdx)
	}

	staticPrefix := promptContent[:markerIdx]
	if len(staticPrefix) < 4096 {
		t.Errorf("base/prompt.md.tmpl raw static prefix is %d bytes, want >= 4096 bytes", len(staticPrefix))
	}

	// Also verify rendered template without .Skills meets >= 4096 bytes floor
	type promptTestData struct {
		Project, Root, AgentName, AgentFile, Date string
		AgentFiles                                []string
		OpenTasks                                 []any
		FocusTask                                 string
		FocusTaskID                               string
		StagedContext                             []any
		TaskContext                               []any
		Skills                                    []any
		HasRefineSkills                           bool
	}
	rendered, err := set.Render("base/prompt.md.tmpl", promptTestData{
		AgentName:  "Antigravity",
		Project:    "sdd",
		Root:       "/test/root",
		AgentFile:  "AGENTS.md",
		AgentFiles: []string{"CLAUDE.md", "GEMINI.md", "AGENTS.md"},
		Date:       "2026-11-25",
	})
	if err != nil {
		t.Fatalf("failed to render base/prompt.md.tmpl: %v", err)
	}
	renderedMarkerIdx := strings.Index(rendered, marker)
	if renderedMarkerIdx == -1 {
		t.Fatalf("rendered base/prompt.md missing cache boundary marker")
	}
	renderedPrefix := rendered[:renderedMarkerIdx]
	if len(renderedPrefix) < 4096 {
		t.Errorf("rendered base/prompt.md static prefix without skills is %d bytes, want >= 4096 bytes", len(renderedPrefix))
	}

	anthemIdx := strings.Index(promptContent, "ANTHEM")
	rulesIdx := strings.Index(promptContent, "Hard rules")
	if anthemIdx == -1 || rulesIdx == -1 || anthemIdx > markerIdx || rulesIdx > markerIdx {
		t.Errorf("core contract sections must appear before cache boundary marker")
	}

	// 2. Verify compact prompt template (base/prompt_compact.md.tmpl)
	compactContent, err := set.Raw("base/prompt_compact.md.tmpl")
	if err != nil {
		t.Fatalf("failed to read base/prompt_compact.md.tmpl: %v", err)
	}

	compactMarkerIdx := strings.Index(compactContent, marker)
	if compactMarkerIdx == -1 {
		t.Fatalf("base/prompt_compact.md.tmpl missing cache boundary marker: %s", marker)
	}

	gatesIdx := strings.Index(compactContent, "## Gates & Protocol")
	activeStateIdx := strings.Index(compactContent, "## Active State")
	if gatesIdx == -1 || activeStateIdx == -1 {
		t.Fatalf("base/prompt_compact.md.tmpl missing Gates & Protocol or Active State")
	}

	if gatesIdx > compactMarkerIdx {
		t.Errorf("## Gates & Protocol (%d) must appear before cache boundary marker (%d)", gatesIdx, compactMarkerIdx)
	}
	if compactMarkerIdx > activeStateIdx {
		t.Errorf("cache boundary marker (%d) must appear before ## Active State (%d)", compactMarkerIdx, activeStateIdx)
	}
}
