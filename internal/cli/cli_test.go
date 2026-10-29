package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/scaffold"
)

var fixedNow = func() time.Time { return time.Date(2026, 10, 22, 9, 0, 0, 0, time.UTC) }

type result struct {
	code     int
	out, err string
}

func exec(t *testing.T, stdin string, interactive bool, home string, args ...string) result {
	t.Helper()
	var out, errb bytes.Buffer
	env := IO{
		In: strings.NewReader(stdin), Out: &out, Err: &errb, Interactive: interactive, Now: fixedNow,
		Getenv: func(k string) string {
			if k == "USER" {
				return "vikash"
			}
			return ""
		},
		Home: func() string { return home },
	}
	code := Run(args, env)
	return result{code, out.String(), errb.String()}
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

func read(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func exists(root, rel string) bool {
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil
}

func TestTopLevelDispatch(t *testing.T) {
	if r := exec(t, "", false, "", "version"); r.code != 0 || !strings.HasPrefix(r.out, "sdd ") {
		t.Errorf("version: %+v", r)
	}
	if r := exec(t, "", false, ""); r.code != 2 || !strings.Contains(r.err, "usage: sdd") {
		t.Errorf("no args: %+v", r)
	}
	if r := exec(t, "", false, "", "frobnicate"); r.code != 2 || !strings.Contains(r.err, `unknown command "frobnicate"`) {
		t.Errorf("unknown: %+v", r)
	}
	if r := exec(t, "", false, "", "help"); r.code != 0 || !strings.Contains(r.out, "spec-driven") || r.err != "" {
		t.Errorf("help: %+v", r)
	}
	if r := exec(t, "", false, "", "init", "-h"); r.code != 0 {
		t.Errorf("init -h should succeed: %+v", r)
	}
	if r := exec(t, "", false, "", "init", "--nonsense"); r.code != 2 {
		t.Errorf("bad flag should be a usage error: %+v", r)
	}
}

func TestInitYesGreenfieldThenCheckPasses(t *testing.T) {
	root := t.TempDir()
	r := exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "acme", "--purpose", "Ships acme.", "--lang", "go,typescript")
	if r.code != 0 {
		t.Fatalf("init failed: %+v", r)
	}
	for _, f := range []string{"CLAUDE.md", "GEMINI.md", "AGENTS.md", "MISSION.md", ".sdd/manifest.json", "features/_templates/plan.md"} {
		if !exists(root, f) {
			t.Errorf("missing %s", f)
		}
	}
	if !strings.Contains(r.out, "Next steps:") || !strings.Contains(r.out, "sdd prompt gemini") {
		t.Errorf("next steps missing:\n%s", r.out)
	}
	c := exec(t, "", false, "", "check", "--dir", root)
	if c.code != 0 || !strings.Contains(c.out, "0 error(s)") {
		t.Errorf("check on a fresh kit should pass: %+v", c)
	}
	if s := exec(t, "", false, "", "check", "--dir", root, "--strict"); s.code != 1 {
		t.Errorf("--strict must fail while TODO placeholders remain: %+v", s)
	}
}

func TestInitRefusals(t *testing.T) {
	home := t.TempDir()
	if r := exec(t, "", false, home, "init", "--yes", "--dir", home); r.code != 1 || !strings.Contains(r.err, "home directory") {
		t.Errorf("home dir not refused: %+v", r)
	}
	if r := exec(t, "", false, "", "init", "--yes", "--dir", string(filepath.Separator)); r.code != 1 || !strings.Contains(r.err, "filesystem root") {
		t.Errorf("root dir not refused: %+v", r)
	}
	proj := t.TempDir()
	if r := exec(t, "", false, "", "init", "--dir", proj); r.code != 1 || !strings.Contains(r.err, "--yes") {
		t.Errorf("non-interactive without --yes must be refused: %+v", r)
	}
	if exists(proj, "CLAUDE.md") {
		t.Error("nothing may be written on refusal")
	}
	if r := exec(t, "", false, "", "init", "--yes", "--dir", proj, "--lang", "cobol"); r.code != 1 || !strings.Contains(r.err, "cobol") {
		t.Errorf("unknown language must be rejected: %+v", r)
	}
	if r := exec(t, "", false, "", "init", "--yes", "--dir", proj, "stray"); r.code != 2 {
		t.Errorf("stray argument is a usage error: %+v", r)
	}
	exec(t, "", false, "", "init", "--yes", "--dir", proj)
	if r := exec(t, "", false, "", "init", "--yes", "--dir", proj); r.code != 1 || !strings.Contains(r.err, "already initialised") {
		t.Errorf("second init must be refused: %+v", r)
	}
}

func TestInitDryRunWritesNothing(t *testing.T) {
	root := t.TempDir()
	put(t, root, "main.go", "package main\n")
	r := exec(t, "", false, "", "init", "--yes", "--dry-run", "--dir", root)
	if r.code != 0 || !strings.Contains(r.out, "Dry run: nothing was written.") || !strings.Contains(r.out, "brownfield") {
		t.Fatalf("dry run: %+v", r)
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 1 {
		t.Errorf("dry run created files: %v", entries)
	}
}

func TestInteractiveInit(t *testing.T) {
	root := t.TempDir()
	// mode (invalid, then valid), name, purpose, languages, maintainer, build, test, lint, format, agents, two rules, finish, confirm
	script := strings.Join([]string{
		"yellowfield", "greenfield", "widgets", "Sells widgets.", "go, python", "Vik", "make build", "make test", "", "",
		"claude,agents", "Always use the sandbox.", "Never email customers.", "", "y",
	}, "\n") + "\n"
	r := exec(t, script, true, "", "init", "--dir", root)
	if r.code != 0 {
		t.Fatalf("interactive init failed: %+v", r)
	}
	if !strings.Contains(r.out, "answer greenfield or brownfield") {
		t.Errorf("invalid mode was not rejected with a message:\n%s", r.out)
	}
	m, ok, err := scaffold.ReadManifest(root)
	if err != nil || !ok {
		t.Fatalf("manifest: %v %v", ok, err)
	}
	a := m.Answers
	if a.Project != "widgets" || a.Maintainer != "Vik" || a.Build != "make build" || a.Lint != "" {
		t.Errorf("answers not captured: %+v", a)
	}
	if strings.Join(a.Languages, ",") != "go,python" || strings.Join(a.Agents, ",") != "claude,agents" {
		t.Errorf("lists wrong: %v %v", a.Languages, a.Agents)
	}
	if len(a.ExtraRules) != 2 {
		t.Errorf("rules = %v", a.ExtraRules)
	}
	if exists(root, "GEMINI.md") {
		t.Error("GEMINI.md was not requested")
	}
	if !strings.Contains(read(t, root, "CLAUDE.md"), "14. Never email customers.") {
		t.Error("extra rules not rendered in the contract")
	}
}

func TestInteractiveAbortWritesNothing(t *testing.T) {
	root := t.TempDir()
	script := strings.Repeat("\n", 12) + "n\n"
	r := exec(t, script, true, "", "init", "--dir", root)
	if r.code != 1 || !strings.Contains(r.out, "Aborted") {
		t.Fatalf("expected abort: %+v", r)
	}
	if exists(root, "CLAUDE.md") || exists(root, ".sdd/manifest.json") {
		t.Error("abort must not write files")
	}
}

func TestInteractiveEndOfInputDefaultsButNeverConfirms(t *testing.T) {
	root := t.TempDir()
	r := exec(t, "", true, "", "init", "--dir", root)
	if r.code != 1 || exists(root, "MISSION.md") {
		t.Fatalf("EOF must not be treated as confirmation: %+v", r)
	}
}

func TestBrownfieldKeepsExistingFilesAndMergesAgentFile(t *testing.T) {
	root := t.TempDir()
	put(t, root, "go.mod", "module x\n")
	put(t, root, "main.go", "package main\n")
	put(t, root, "README.md", "# existing\n")
	put(t, root, "CLAUDE.md", "# house rules\n\nBe nice.\n")
	put(t, root, "MISSION.md", "# our own mission\n")
	r := exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "legacy")
	if r.code != 0 {
		t.Fatalf("init failed: %+v", r)
	}
	if !strings.Contains(r.out, "merge    CLAUDE.md") || !strings.Contains(r.out, "skip     README.md") {
		t.Errorf("plan output wrong:\n%s", r.out)
	}
	if read(t, root, "README.md") != "# existing\n" || !strings.HasPrefix(read(t, root, "CLAUDE.md"), "# house rules\n\nBe nice.\n") {
		t.Error("existing human content was not preserved")
	}
	if read(t, root, "MISSION.md") != "# our own mission\n" || !strings.Contains(r.out, "skip     MISSION.md") {
		t.Error("an existing MISSION.md must be skipped, not replaced")
	}
	if !strings.Contains(read(t, root, "ROADMAP.md"), "Discovery") {
		t.Error("brownfield roadmap should seed a Discovery task")
	}
	if c := exec(t, "", false, "", "check", "--dir", root); c.code != 0 {
		t.Errorf("check after a brownfield init should pass:\n%s", c.out)
	}
}

func TestFeatureNewAndPromptWorkflow(t *testing.T) {
	root := t.TempDir()
	exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "flow", "--lang", "java")

	// flags may follow the positional arguments
	r := exec(t, "", false, "", "feature", "new", "T-001", "scope", "--dir", root, "--title", "Define the scope")
	if r.code != 0 || !strings.Contains(r.out, "features/T-001-scope") || !strings.Contains(r.out, "Specifying") {
		t.Fatalf("feature new: %+v", r)
	}
	if !strings.Contains(read(t, root, "features/T-001-scope/plan.md"), "# Plan — T-001 Define the scope") {
		t.Error("title flag not applied")
	}
	if r := exec(t, "", false, "", "feature", "new", "T-001", "again", "--dir", root); r.code != 1 {
		t.Errorf("second folder for the same task must fail: %+v", r)
	}
	if r := exec(t, "", false, "", "feature", "new", "T-001"); r.code != 2 {
		t.Errorf("missing slug is a usage error: %+v", r)
	}
	if r := exec(t, "", false, "", "feature"); r.code != 2 {
		t.Errorf("missing subcommand is a usage error: %+v", r)
	}
	if c := exec(t, "", false, "", "check", "--dir", root); c.code != 0 {
		t.Errorf("check after feature new should pass:\n%s", c.out)
	}

	p := exec(t, "", false, "", "prompt", "gemini", "--dir", root)
	if p.code != 0 {
		t.Fatalf("prompt: %+v", p)
	}
	for _, want := range []string{"You are Gemini", "flow", "GEMINI.md", "T-001", "Specifying", "APPROVED T-xxx"} {
		if !strings.Contains(p.out, want) {
			t.Errorf("prompt is missing %q", want)
		}
	}
	if strings.Contains(p.out, "{{") || strings.Contains(p.out, "<no value>") {
		t.Error("prompt has template leftovers")
	}
	if r := exec(t, "", false, "", "prompt", "gemini", "--dir", root, "--task", "T-099"); r.code != 1 {
		t.Errorf("unknown focus task must fail: %+v", r)
	}
	if r := exec(t, "", false, "", "prompt", "claude", "--dir", t.TempDir()); r.code != 1 || !strings.Contains(r.err, "sdd init") {
		t.Errorf("prompt without a kit must point to init: %+v", r)
	}
	if r := exec(t, "", false, "", "prompt", "--dir", root); r.code != 2 {
		t.Errorf("prompt needs an agent argument: %+v", r)
	}
	other := exec(t, "", false, "", "prompt", "Codex", "--dir", root)
	if other.code != 0 || !strings.Contains(other.out, "You are Codex") || !strings.Contains(other.out, "AGENTS.md") {
		t.Errorf("unknown agent names should use AGENTS.md: %+v", other)
	}
}

func TestInitIsDeterministic(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	args := []string{"init", "--yes", "--name", "same", "--purpose", "x", "--lang", "go,java,python,typescript", "--maintainer", "V"}
	exec(t, "", false, "", append(args, "--dir", a)...)
	exec(t, "", false, "", append(args, "--dir", b)...)
	for _, f := range []string{"CLAUDE.md", "AGENTS.md", "MISSION.md", "TECH_STACK.md", "ROADMAP.md", "SKILLS.md", "README.md", ".sdd/manifest.json"} {
		if read(t, a, f) != read(t, b, f) {
			t.Errorf("%s differs between identical runs", f)
		}
	}
}

func TestForceReinitBacksUpAndKeepsReadme(t *testing.T) {
	root := t.TempDir()
	exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "p")
	put(t, root, "MISSION.md", "my careful mission\n")
	put(t, root, "README.md", "my readme\n")
	r := exec(t, "", false, "", "init", "--yes", "--force", "--dir", root, "--name", "p")
	if r.code != 0 || !strings.Contains(r.out, "backed up") {
		t.Fatalf("force init: %+v", r)
	}
	if read(t, root, "README.md") != "my readme\n" {
		t.Error("README.md must survive --force")
	}
	entries, err := os.ReadDir(filepath.Join(root, ".sdd", "backup"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected one backup directory: %v %v", entries, err)
	}
	if read(t, root, ".sdd/backup/"+entries[0].Name()+"/MISSION.md") != "my careful mission\n" {
		t.Error("the original MISSION.md was not backed up")
	}
}

func TestSdDirectoryIgnoredByDetection(t *testing.T) {
	root := t.TempDir()
	exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "p")
	// a re-run of detection must not count the generated kit as source
	r := exec(t, "", false, "", "init", "--yes", "--dry-run", "--force", "--dir", root, "--name", "p")
	if !strings.Contains(r.out, "source files   0") {
		t.Errorf("generated docs counted as source:\n%s", r.out)
	}
}

func TestInitRefusesHandMadeKitWithoutManifest(t *testing.T) {
	root := t.TempDir()
	for _, f := range []string{"MISSION.md", "TECH_STACK.md", "ROADMAP.md", "SKILLS.md"} {
		put(t, root, f, "# "+f+"\n")
	}
	put(t, root, "CLAUDE.md", "# hand-made contract\n")
	r := exec(t, "", false, "", "init", "--yes", "--dir", root)
	if r.code != 1 || !strings.Contains(r.err, "made by hand") {
		t.Fatalf("a hand-made kit must be refused: %+v", r)
	}
	if read(t, root, "CLAUDE.md") != "# hand-made contract\n" || exists(root, ".sdd/manifest.json") || exists(root, "AGENTS.md") {
		t.Error("nothing may be written when init refuses")
	}
	// a partial set of the project's own documents is fine: those files are simply skipped
	partial := t.TempDir()
	put(t, partial, "ROADMAP.md", "# my roadmap\n")
	if r := exec(t, "", false, "", "init", "--yes", "--dir", partial); r.code != 0 {
		t.Errorf("a partial set of existing docs must not block init: %+v", r)
	}
	// the explicit override still works
	if r := exec(t, "", false, "", "init", "--yes", "--force", "--dir", root); r.code != 0 {
		t.Errorf("--force should override the refusal: %+v", r)
	}
}
