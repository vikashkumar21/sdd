package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
