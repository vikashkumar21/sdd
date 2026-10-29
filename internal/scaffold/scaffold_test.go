package scaffold

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/block"
	"github.com/vikashkumar21/sdd/internal/detect"
	"github.com/vikashkumar21/sdd/internal/templates"
)

var fixedNow = time.Date(2026, 10, 22, 9, 30, 0, 0, time.UTC)

func answers() Answers {
	return Answers{
		Project: "demo", Purpose: "A demo project.", Mode: ModeGreenfield, Maintainer: "Vikash",
		Languages: []string{"go", "java"}, Build: "make build", Test: "make test",
	}
}

func plan(t *testing.T, root string, a Answers, force bool) Plan {
	t.Helper()
	set, err := templates.New("")
	if err != nil {
		t.Fatal(err)
	}
	det, err := detect.Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	p, err := BuildPlan(root, a, det, set, fixedNow, force)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func apply(t *testing.T, p Plan) Report {
	t.Helper()
	rep, err := Apply(p, fixedNow, "")
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

func read(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func put(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGreenfieldCreatesTheWholeKit(t *testing.T) {
	root := t.TempDir()
	rep := apply(t, plan(t, root, answers(), false))
	want := []string{
		"CLAUDE.md", "GEMINI.md", "AGENTS.md", "MISSION.md", "TECH_STACK.md", "ROADMAP.md", "SKILLS.md",
		"README.md", "features/_templates/requirements.md", "features/_templates/plan.md",
		"features/_templates/validation.md", ".sdd/manifest.json",
	}
	for _, rel := range want {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
	if rep.BackupDir != "" {
		t.Errorf("greenfield must not create backups, got %s", rep.BackupDir)
	}
	if rep.Todos == 0 {
		t.Error("expected TODO(human) placeholders to be counted")
	}
}

func TestRenderedFilesHaveNoTemplateLeftovers(t *testing.T) {
	for _, mode := range []string{ModeGreenfield, ModeBrownfield} {
		root := t.TempDir()
		a := answers()
		a.Mode = mode
		a.Languages = []string{"go", "java", "python", "typescript", "other"}
		a.ExtraRules = []string{"Always use the staging cluster."}
		apply(t, plan(t, root, a, false))
		for _, rel := range []string{"CLAUDE.md", "MISSION.md", "TECH_STACK.md", "ROADMAP.md", "SKILLS.md", "README.md"} {
			text := read(t, root, rel)
			for _, bad := range []string{"{{", "}}", "<no value>", "%!"} {
				if strings.Contains(text, bad) {
					t.Errorf("%s (%s) contains %q", rel, mode, bad)
				}
			}
		}
		if !strings.Contains(read(t, root, "CLAUDE.md"), "13. Always use the staging cluster.") {
			t.Errorf("%s: extra rule was not numbered 13", mode)
		}
	}
}

func TestAgentFilesShareOneManagedBlock(t *testing.T) {
	root := t.TempDir()
	apply(t, plan(t, root, answers(), false))
	var inners []string
	for _, f := range []string{"CLAUDE.md", "GEMINI.md", "AGENTS.md"} {
		inner, ok := block.Inner(read(t, root, f))
		if !ok {
			t.Fatalf("%s has no managed block", f)
		}
		inners = append(inners, inner)
	}
	if inners[0] != inners[1] || inners[1] != inners[2] {
		t.Fatal("agent contract blocks differ between files")
	}
	if !strings.HasPrefix(read(t, root, "GEMINI.md"), "# GEMINI.md — demo agent contract (Gemini)") {
		t.Error("unexpected GEMINI.md title")
	}
}

func TestBrownfieldNeverTouchesExistingFiles(t *testing.T) {
	root := t.TempDir()
	put(t, root, "README.md", "# Existing readme\n")
	put(t, root, "ROADMAP.md", "# my own roadmap\n")
	put(t, root, "main.go", "package main\n")
	a := answers()
	a.Mode = ModeBrownfield
	apply(t, plan(t, root, a, false))
	if got := read(t, root, "README.md"); got != "# Existing readme\n" {
		t.Errorf("README.md was modified: %q", got)
	}
	if got := read(t, root, "ROADMAP.md"); got != "# my own roadmap\n" {
		t.Errorf("ROADMAP.md was modified: %q", got)
	}
	if got := read(t, root, "main.go"); got != "package main\n" {
		t.Errorf("source file was modified: %q", got)
	}
}

func TestMergeIntoExistingAgentFileIsIdempotent(t *testing.T) {
	root := t.TempDir()
	original := "# Our team rules\n\nBe kind.\n"
	put(t, root, "CLAUDE.md", original)
	first := plan(t, root, answers(), false)
	var op Op
	for _, o := range first.Ops {
		if o.Path == "CLAUDE.md" {
			op = o
		}
	}
	if op.Kind != KindMerge {
		t.Fatalf("expected merge, got %s", op.Kind)
	}
	apply(t, first)
	merged := read(t, root, "CLAUDE.md")
	if !strings.HasPrefix(merged, original) {
		t.Fatalf("original content not preserved:\n%s", merged)
	}
	if !block.Has(merged) {
		t.Fatal("managed block missing after merge")
	}
	// A second plan must now skip the file, and the file must not change.
	second := plan(t, root, answers(), false)
	for _, o := range second.Ops {
		if o.Path == "CLAUDE.md" && o.Kind != KindSkip {
			t.Fatalf("second run should skip CLAUDE.md, got %s", o.Kind)
		}
	}
}

func TestExistingDocsAreSkippedWithoutForce(t *testing.T) {
	root := t.TempDir()
	put(t, root, "MISSION.md", "mine\n")
	p := plan(t, root, answers(), false)
	apply(t, p)
	if read(t, root, "MISSION.md") != "mine\n" {
		t.Fatal("MISSION.md was overwritten without --force")
	}
}

func TestForceReplacesWithBackupButNeverReadme(t *testing.T) {
	root := t.TempDir()
	put(t, root, "MISSION.md", "mine\n")
	put(t, root, "README.md", "readme\n")
	put(t, root, "CLAUDE.md", "# keep this line\n\n"+block.Begin("0.0.1")+"\nold contract\n"+block.End+"\n\ntrailing human text\n")
	rep := apply(t, plan(t, root, answers(), true))
	if rep.BackupDir == "" {
		t.Fatal("force replaced files but reported no backup directory")
	}
	if got := read(t, root, filepath.ToSlash(filepath.Join(rep.BackupDir, "MISSION.md"))); got != "mine\n" {
		t.Errorf("backup of MISSION.md wrong: %q", got)
	}
	if strings.Contains(read(t, root, "MISSION.md"), "mine") {
		t.Error("MISSION.md was not replaced")
	}
	if read(t, root, "README.md") != "readme\n" {
		t.Error("README.md must never be modified, even with --force")
	}
	claude := read(t, root, "CLAUDE.md")
	if !strings.Contains(claude, "# keep this line") || !strings.Contains(claude, "trailing human text") {
		t.Errorf("text outside the managed block was lost:\n%s", claude)
	}
	if strings.Contains(claude, "old contract") {
		t.Error("managed block was not refreshed")
	}
}

func TestOutputIsDeterministic(t *testing.T) {
	rootA, rootB := t.TempDir(), t.TempDir()
	apply(t, plan(t, rootA, answers(), false))
	apply(t, plan(t, rootB, answers(), false))
	for _, rel := range []string{"CLAUDE.md", "MISSION.md", "TECH_STACK.md", "ROADMAP.md", "SKILLS.md", "README.md", ".sdd/manifest.json"} {
		if read(t, rootA, rel) != read(t, rootB, rel) {
			t.Errorf("%s differs between two identical runs", rel)
		}
	}
}

func TestManifestRecordsWhatHappened(t *testing.T) {
	root := t.TempDir()
	put(t, root, "README.md", "x\n")
	apply(t, plan(t, root, answers(), false))
	m, ok, err := ReadManifest(root)
	if err != nil || !ok {
		t.Fatalf("manifest: ok=%v err=%v", ok, err)
	}
	if m.Schema != 1 || m.Generated != "2026-10-22" || m.Answers.Project != "demo" {
		t.Errorf("unexpected manifest header: %+v", m)
	}
	states := map[string]string{}
	for _, f := range m.Files {
		states[f.Path] = f.State
	}
	if states["README.md"] != "existing" || states["MISSION.md"] != "created" {
		t.Errorf("unexpected file states: %v", states)
	}
	if !reflect.DeepEqual(m.Answers.Languages, []string{"go", "java"}) {
		t.Errorf("languages = %v", m.Answers.Languages)
	}
}

func TestNormalizeRejectsBadInput(t *testing.T) {
	tests := []struct {
		name string
		mut  func(*Answers)
	}{
		{"empty project", func(a *Answers) { a.Project = " " }},
		{"pipe in project", func(a *Answers) { a.Project = "a|b" }},
		{"backtick in command", func(a *Answers) { a.Build = "echo `x`" }},
		{"pipe in command", func(a *Answers) { a.Test = "go test | tee" }},
		{"newline in maintainer", func(a *Answers) { a.Maintainer = "a\nb" }},
		{"bad mode", func(a *Answers) { a.Mode = "yellowfield" }},
		{"unknown language", func(a *Answers) { a.Languages = []string{"cobol"} }},
		{"unknown agent", func(a *Answers) { a.Agents = []string{"hal"} }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := answers()
			tc.mut(&a)
			if err := a.Normalize(); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestNormalizeAppliesDefaultsAndCanonicalOrder(t *testing.T) {
	a := Answers{Project: "p", Mode: ModeGreenfield, Languages: []string{"TypeScript", "go", "go"}, Agents: []string{"agents", "claude"}}
	if err := a.Normalize(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a.Languages, []string{"go", "typescript"}) || !reflect.DeepEqual(a.Agents, []string{"claude", "agents"}) {
		t.Errorf("not canonical: %v %v", a.Languages, a.Agents)
	}
	if !strings.Contains(a.Purpose, Todo) || a.Maintainer != Todo {
		t.Errorf("defaults not applied: %q %q", a.Purpose, a.Maintainer)
	}
	empty := Answers{Project: "p", Mode: ModeBrownfield}
	if err := empty.Normalize(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(empty.Languages, []string{"other"}) || len(empty.Agents) != 3 {
		t.Errorf("empty defaults wrong: %v %v", empty.Languages, empty.Agents)
	}
}

func TestSafeRel(t *testing.T) {
	for _, bad := range []string{"", "/etc/passwd", "../x", "a/../../x"} {
		if err := safeRel(bad); err == nil {
			t.Errorf("safeRel(%q) should fail", bad)
		}
	}
	if err := safeRel("features/_templates/plan.md"); err != nil {
		t.Errorf("safeRel rejected a normal path: %v", err)
	}
}

func TestTemplateOverrideWinsPerFile(t *testing.T) {
	over := t.TempDir()
	put(t, over, "base/README.md.tmpl", "# overridden {{.Project}}\n")
	set, err := templates.New(over)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	det, _ := detect.Detect(root)
	p, err := BuildPlan(root, answers(), det, set, fixedNow, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(p, fixedNow, over); err != nil {
		t.Fatal(err)
	}
	if read(t, root, "README.md") != "# overridden demo\n" {
		t.Errorf("override not used: %q", read(t, root, "README.md"))
	}
	if !strings.Contains(read(t, root, "MISSION.md"), "# MISSION — demo") {
		t.Error("non-overridden templates must fall back to the embedded ones")
	}
}

func TestBrokenOverrideTemplateIsAnError(t *testing.T) {
	over := t.TempDir()
	put(t, over, "base/MISSION.md.tmpl", "{{ .Nope }}")
	set, err := templates.New(over)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	det, _ := detect.Detect(root)
	if _, err := BuildPlan(root, answers(), det, set, fixedNow, false); err == nil {
		t.Fatal("expected an error for a template referencing an unknown field")
	}
}
