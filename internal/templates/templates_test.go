package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	if Version != "2.0.0" {
		t.Errorf("Version = %q, want 2.0.0", Version)
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
		if p.Title == "" || p.Standards == "" || p.Contract == "" || p.SkillMap == "" {
			t.Errorf("pack %s has an empty part: %+v", id, p)
		}
		for _, row := range strings.Split(p.SkillMap, "\n") {
			if strings.Count(row, "|") != 5 {
				t.Errorf("pack %s skill-map row must have 4 cells: %q", id, row)
			}
		}
		for _, bad := range []string{"{{", "}}"} {
			if strings.Contains(p.Standards+p.Contract+p.SkillMap, bad) {
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
