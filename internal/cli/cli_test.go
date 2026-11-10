package cli

import (
	"bytes"
	"encoding/json"
	"os"
	osexec "os/exec"
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
	return execWithEnv(t, stdin, interactive, home, func(k string) string {
		if k == "USER" {
			return "vikash"
		}
		return ""
	}, args...)
}

func execWithEnv(t *testing.T, stdin string, interactive bool, home string, getenv func(string) string, args ...string) result {
	t.Helper()
	var out, errb bytes.Buffer
	env := IO{
		In: strings.NewReader(stdin), Out: &out, Err: &errb, Interactive: interactive, Now: fixedNow,
		Getenv: getenv,
		Home:   func() string { return home },
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
	for _, f := range []string{"CLAUDE.md", "GEMINI.md", "AGENTS.md", "MISSION.md", ".sdd/manifest.json", "tasks/features/_templates/plan.md"} {
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
	if !strings.Contains(read(t, root, "CLAUDE.md"), "16. Never email customers.") {
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
	if r.code != 0 || !strings.Contains(r.out, "tasks/features/T-001-scope") || !strings.Contains(r.out, "Specifying") {
		t.Fatalf("feature new: %+v", r)
	}
	if !strings.Contains(read(t, root, "tasks/features/T-001-scope/plan.md"), "# Plan — T-001 Define the scope") {
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

func TestInitDriveRootRefusals(t *testing.T) {
	for _, root := range []string{"/", `\`, `C:\`, `D:/`, `c:`, `\\server\share`} {
		r := exec(t, "", false, "", "init", "--yes", "--dir", root)
		if r.code != 1 || !strings.Contains(r.err, "refusing to run in the filesystem root") {
			t.Errorf("expected drive root refusal for %q, got: %+v", root, r)
		}
	}
}

func TestInitMaintainerWindowsFallback(t *testing.T) {
	root := t.TempDir()
	var out, errb bytes.Buffer
	env := IO{
		In: strings.NewReader(""), Out: &out, Err: &errb, Interactive: false, Now: fixedNow,
		Getenv: func(k string) string {
			if k == "USERNAME" {
				return "winuser"
			}
			return ""
		},
		Home: func() string { return "" },
	}
	code := Run([]string{"init", "--yes", "--dir", root, "--name", "winproj"}, env)
	if code != 0 {
		t.Fatalf("expected 0, got %d; err: %s", code, errb.String())
	}
	content := read(t, root, "MISSION.md")
	if !strings.Contains(content, "winuser") {
		t.Errorf("expected maintainer 'winuser' in MISSION.md, got:\n%s", content)
	}
}

func TestSkillNew(t *testing.T) {
	root := t.TempDir()

	// 1. Missing args or invalid subaction
	r := exec(t, "", false, "", "skill")
	if r.code != 2 {
		t.Errorf("expected exit code 2 for sdd skill without subaction, got %d", r.code)
	}

	r = exec(t, "", false, "", "skill", "unknown")
	if r.code != 2 {
		t.Errorf("expected exit code 2 for sdd skill unknown, got %d", r.code)
	}

	r = exec(t, "", false, "", "skill", "new")
	if r.code != 2 {
		t.Errorf("expected exit code 2 for sdd skill new without name, got %d", r.code)
	}

	// 2. Successful skill new
	r = exec(t, "", false, "", "skill", "new", "deploy-checker", "--dir", root, "--description", "Verify deployment health")
	if r.code != 0 {
		t.Fatalf("expected 0, got %d; err: %s", r.code, r.err)
	}
	if !strings.Contains(r.out, "Created skill deploy-checker at skills/deploy-checker/SKILL.md") {
		t.Errorf("unexpected stdout: %s", r.out)
	}
	skillPath := filepath.Join(root, "skills", "deploy-checker", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}
	if !strings.Contains(string(content), "name: deploy-checker") {
		t.Errorf("missing name in skill file: %s", string(content))
	}

	// 3. Collision refusal without --force
	r = exec(t, "", false, "", "skill", "new", "deploy-checker", "--dir", root)
	if r.code != 1 {
		t.Fatalf("expected exit code 1 for duplicate skill without --force, got %d", r.code)
	}
	if !strings.Contains(r.err, "already exists") {
		t.Errorf("unexpected error for duplicate skill: %s", r.err)
	}

	// 4. Overwrite with --force and backup
	r = exec(t, "", false, "", "skill", "new", "deploy-checker", "--dir", root, "--from-text", "New instructions", "--force")
	if r.code != 0 {
		t.Fatalf("expected 0 with --force, got %d; err: %s", r.code, r.err)
	}
	if !strings.Contains(r.out, "Previous version backed up to .sdd/backup/") {
		t.Errorf("expected backup confirmation in stdout, got: %s", r.out)
	}

	// 5. Ingestion with --from-file
	notesFile := filepath.Join(root, "my_notes.txt")
	if err := os.WriteFile(notesFile, []byte("Raw steps from meeting"), 0o644); err != nil {
		t.Fatal(err)
	}
	r = exec(t, "", false, "", "skill", "new", "meeting-notes", "--dir", root, "--from-file", "my_notes.txt")
	if r.code != 0 {
		t.Fatalf("expected 0 for --from-file, got %d; err: %s", r.code, r.err)
	}
	notesSkillPath := filepath.Join(root, "skills", "meeting-notes", "SKILL.md")
	notesContent, err := os.ReadFile(notesSkillPath)
	if err != nil {
		t.Fatalf("failed to read notes skill: %v", err)
	}
	if !strings.Contains(string(notesContent), "Raw steps from meeting") {
		t.Errorf("missing ingested file content in skill: %s", string(notesContent))
	}
	if !strings.Contains(string(notesContent), "<!-- sdd:skill-refine:begin -->") {
		t.Errorf("missing refinement block in skill: %s", string(notesContent))
	}
}

func TestSkillList(t *testing.T) {
	root := t.TempDir()

	// 1. Empty list
	r := exec(t, "", false, "", "skill", "list", "--dir", root)
	if r.code != 0 {
		t.Fatalf("expected 0 on empty skill list, got %d: %s", r.code, r.err)
	}
	if !strings.Contains(r.out, "No skills found.") {
		t.Errorf("expected 'No skills found.', got:\n%s", r.out)
	}

	// 2. Unexpected positional argument
	r = exec(t, "", false, "", "skill", "list", "extra-arg", "--dir", root)
	if r.code != 2 {
		t.Errorf("expected exit code 2 for unexpected argument, got %d", r.code)
	}

	// 3. Create project skill and internal skill
	r1 := exec(t, "", false, "", "skill", "new", "beta-project", "--dir", root, "--description", "Project skill beta")
	if r1.code != 0 {
		t.Fatalf("failed to create beta-project: %s", r1.err)
	}
	r2 := exec(t, "", false, "", "skill", "new", "alpha-internal", "--dir", root, "--internal", "--description", "Internal skill alpha")
	if r2.code != 0 {
		t.Fatalf("failed to create alpha-internal: %s", r2.err)
	}

	// 4. Formatted table output (deterministic sorting)
	r = exec(t, "", false, "", "skill", "list", "--dir", root)
	if r.code != 0 {
		t.Fatalf("expected 0, got %d: %s", r.code, r.err)
	}
	lines := strings.Split(strings.TrimSpace(r.out), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected header + 2 rows, got %d lines:\n%s", len(lines), r.out)
	}
	if !strings.Contains(lines[0], "NAME") || !strings.Contains(lines[0], "SCOPE") {
		t.Errorf("header line invalid: %s", lines[0])
	}
	// alpha-internal must precede beta-project
	if !strings.Contains(lines[1], "alpha-internal") || !strings.Contains(lines[1], "internal") {
		t.Errorf("first row should be alpha-internal: %s", lines[1])
	}
	if !strings.Contains(lines[2], "beta-project") || !strings.Contains(lines[2], "project") {
		t.Errorf("second row should be beta-project: %s", lines[2])
	}

	// 5. JSON output
	r = exec(t, "", false, "", "skill", "list", "--dir", root, "--json")
	if r.code != 0 {
		t.Fatalf("expected 0 on --json, got %d: %s", r.code, r.err)
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(r.out), &items); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput was:\n%s", err, r.out)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items in JSON array, got %d", len(items))
	}
	if items[0]["name"] != "alpha-internal" || items[0]["scope"] != "internal" {
		t.Errorf("unexpected first item: %+v", items[0])
	}
	if items[1]["name"] != "beta-project" || items[1]["scope"] != "project" {
		t.Errorf("unexpected second item: %+v", items[1])
	}
}

func TestSkillExport(t *testing.T) {
	root := t.TempDir()

	// 1. Export to project root skills/ (defaults to all)
	r := exec(t, "", false, "", "skill", "export", "--dir", root)
	if r.code != 0 {
		t.Fatalf("expected 0 for skill export, got %d: %s", r.code, r.err)
	}
	if !strings.Contains(r.out, "Exported skill sdd-protocol to") || !strings.Contains(r.out, "Exported skill code-review to") || !strings.Contains(r.out, "Exported skill task-coordinator to") {
		t.Errorf("unexpected stdout: %s", r.out)
	}
	skillPath := filepath.Join(root, "skills", "sdd-protocol", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("failed to read exported skill: %v", err)
	}
	if !strings.Contains(string(content), "name: sdd-protocol") {
		t.Errorf("missing name in exported skill: %s", string(content))
	}
	revSkillPath := filepath.Join(root, "skills", "code-review", "SKILL.md")
	revContent, err := os.ReadFile(revSkillPath)
	if err != nil {
		t.Fatalf("failed to read exported code-review skill: %v", err)
	}
	if !strings.Contains(string(revContent), "name: code-review") {
		t.Errorf("missing name in exported code-review skill: %s", string(revContent))
	}
	coordSkillPath := filepath.Join(root, "skills", "task-coordinator", "SKILL.md")
	coordContent, err := os.ReadFile(coordSkillPath)
	if err != nil {
		t.Fatalf("failed to read exported task-coordinator skill: %v", err)
	}
	if !strings.Contains(string(coordContent), "name: task-coordinator") {
		t.Errorf("missing name in exported task-coordinator skill: %s", string(coordContent))
	}

	// 2. Collision refusal without --force
	r = exec(t, "", false, "", "skill", "export", "--dir", root)
	if r.code != 1 {
		t.Fatalf("expected 1 for duplicate export without --force, got %d", r.code)
	}
	if !strings.Contains(r.err, "already exists") {
		t.Errorf("unexpected error: %s", r.err)
	}

	// 3. Overwrite with --force
	r = exec(t, "", false, "", "skill", "export", "--dir", root, "--force")
	if r.code != 0 {
		t.Fatalf("expected 0 for export with --force, got %d: %s", r.code, r.err)
	}
	if !strings.Contains(r.out, "Previous version backed up to .sdd/backup/") {
		t.Errorf("expected backup confirmation in stdout: %s", r.out)
	}

	// 4. Export to custom target directory
	targetDir := t.TempDir()
	r = exec(t, "", false, "", "skill", "export", "--dir", root, "--target", targetDir)
	if r.code != 0 {
		t.Fatalf("expected 0 for export --target, got %d: %s", r.code, r.err)
	}
	customSkillPath := filepath.Join(targetDir, "sdd-protocol", "SKILL.md")
	if _, err := os.Stat(customSkillPath); err != nil {
		t.Fatalf("custom skill file not created: %v", err)
	}

	// 5. Export specific skill
	specDir := t.TempDir()
	r = exec(t, "", false, "", "skill", "export", "--dir", root, "--target", specDir, "--skill", "code-review")
	if r.code != 0 {
		t.Fatalf("expected 0 for export --skill code-review, got %d: %s", r.code, r.err)
	}
	if !strings.Contains(r.out, "Exported skill code-review to") {
		t.Errorf("unexpected output: %s", r.out)
	}
	if _, err := os.Stat(filepath.Join(specDir, "code-review", "SKILL.md")); err != nil {
		t.Errorf("code-review skill not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(specDir, "sdd-protocol", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("sdd-protocol should not have been exported")
	}

	// 6. Unexpected argument
	r = exec(t, "", false, "", "skill", "export", "extra", "--dir", root)
	if r.code != 2 {
		t.Errorf("expected 2 for unexpected argument, got %d", r.code)
	}
}

func TestInstallCli(t *testing.T) {
	home := t.TempDir()
	sourceDir := t.TempDir()
	sourceBin := filepath.Join(sourceDir, "sdd")
	if err := os.WriteFile(sourceBin, []byte("echo binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	getenv := func(k string) string {
		if k == "SDD_EXEC_PATH" {
			return sourceBin
		}
		return ""
	}

	// 1. Dry run
	r := execWithEnv(t, "", false, home, getenv, "install", "--dry-run")
	if r.code != 0 {
		t.Fatalf("expected 0 for install --dry-run, got %d: %s", r.code, r.err)
	}
	if !strings.Contains(r.out, "Dry run: would install sdd to") {
		t.Errorf("unexpected dry run output: %s", r.out)
	}

	// 2. Real install
	r = execWithEnv(t, "", false, home, getenv, "install")
	if r.code != 0 {
		t.Fatalf("expected 0 for install, got %d: %s", r.code, r.err)
	}
	if !strings.Contains(r.out, "Installed sdd to") {
		t.Errorf("expected installation confirmation: %s", r.out)
	}

	installedBin := filepath.Join(home, ".local", "bin", "sdd")
	if _, err := os.Stat(installedBin); err != nil {
		t.Fatalf("installed binary not found at %s: %v", installedBin, err)
	}

	// 3. Repeated install (already installed)
	installedGetenv := func(k string) string {
		if k == "SDD_EXEC_PATH" {
			return installedBin
		}
		return ""
	}
	r = execWithEnv(t, "", false, home, installedGetenv, "install")
	if r.code != 0 {
		t.Fatalf("expected 0 on repeated install, got %d", r.code)
	}
	if !strings.Contains(r.out, "already installed") {
		t.Errorf("expected already installed message, got: %s", r.out)
	}

	// 4. Force install
	r = execWithEnv(t, "", false, home, installedGetenv, "install", "--force")
	if r.code != 0 {
		t.Fatalf("expected 0 on force install, got %d", r.code)
	}
	if !strings.Contains(r.out, "Installed sdd to") {
		t.Errorf("expected re-installation confirmation, got: %s", r.out)
	}
}

func TestUninstallCli(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	installedBin := filepath.Join(binDir, "sdd")
	if err := os.WriteFile(installedBin, []byte("echo binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	zshrc := filepath.Join(home, ".zshrc")
	zshContent := "# sdd:begin path\nexport PATH=\"$HOME/.local/bin:$PATH\"\n# sdd:end path\n"
	if err := os.WriteFile(zshrc, []byte(zshContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1. Dry run
	r := exec(t, "", false, home, "uninstall", "--dry-run")
	if r.code != 0 {
		t.Fatalf("expected 0 for uninstall --dry-run, got %d: %s", r.code, r.err)
	}
	if !strings.Contains(r.out, "Dry run: would remove") {
		t.Errorf("unexpected dry run output: %s", r.out)
	}

	// 2. Real uninstall with --yes
	r = exec(t, "", false, home, "uninstall", "--yes")
	if r.code != 0 {
		t.Fatalf("expected 0 for uninstall --yes, got %d: %s", r.code, r.err)
	}
	if !strings.Contains(r.out, "Removed") {
		t.Errorf("expected removal confirmation: %s", r.out)
	}

	if _, err := os.Stat(installedBin); !os.IsNotExist(err) {
		t.Errorf("binary was not removed")
	}

	cleaned, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cleaned), "sdd:begin path") {
		t.Errorf("markers still present in zshrc: %s", string(cleaned))
	}
}

func TestInteractiveZeroArgInstallPrompt(t *testing.T) {
	home := t.TempDir()
	sourceDir := t.TempDir()
	sourceBin := filepath.Join(sourceDir, "sdd")
	if err := os.WriteFile(sourceBin, []byte("echo binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	getenv := func(k string) string {
		if k == "SDD_EXEC_PATH" {
			return sourceBin
		}
		return ""
	}

	// 1. Interactive zero-arg: user says "y" followed by enter to exit
	r := execWithEnv(t, "y\n\n", true, home, getenv)
	if r.code != 0 {
		t.Fatalf("expected 0 for interactive install, got %d: %s (err: %s)", r.code, r.out, r.err)
	}
	if !strings.Contains(r.out, "Successfully installed sdd to") {
		t.Errorf("expected successful install message: %s", r.out)
	}
	if !strings.Contains(r.out, "Press Enter to exit...") {
		t.Errorf("expected pause prompt 'Press Enter to exit...': %s", r.out)
	}

	// 2. Interactive zero-arg: user says "n"
	r = execWithEnv(t, "n\n", true, home, getenv)
	if r.code != 2 {
		t.Fatalf("expected exit 2 when user declines, got %d", r.code)
	}
	if !strings.Contains(r.err, "usage: sdd <command> [flags]") {
		t.Errorf("expected usage text on decline: %s", r.err)
	}

	// 3. Non-interactive zero-arg: immediately prints usage and exits 2
	r = execWithEnv(t, "", false, home, getenv)
	if r.code != 2 {
		t.Fatalf("expected exit 2 for non-interactive zero-arg, got %d", r.code)
	}
}

func TestStatusCLI(t *testing.T) {
	dir := t.TempDir()
	roadmapContent := `# ROADMAP — sdd

## 3. Milestones

| Milestone | Goal | Tasks | Est. hours | Exit criteria |
|---|---|---|---|---|
| M0 | Scaffolding | T-000 | 1 | Done |
| M1 | Status Command | T-001 | 3 | In progress |

## 4. Tasks

### M0 — Scaffolding

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-000 | Scaffolding task | S | none | Setup | D-001 | Done |

### M1 — Status Command

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-001 | Add sdd status | S | T-000 | Status CLI | none | Specifying |
`
	put(t, dir, "ROADMAP.md", roadmapContent)

	// 1. Human-readable text status
	r := exec(t, "", false, "", "status", "--dir", dir)
	if r.code != 0 {
		t.Fatalf("expected exit 0, got %d: %s (err: %s)", r.code, r.out, r.err)
	}
	if !strings.Contains(r.out, "Active Task:") {
		t.Errorf("expected 'Active Task:' in output: %s", r.out)
	}
	if !strings.Contains(r.out, "T-001") {
		t.Errorf("expected 'T-001' in output: %s", r.out)
	}
	if !strings.Contains(r.out, "GATE 1") {
		t.Errorf("expected 'GATE 1' in output: %s", r.out)
	}
	if !strings.Contains(r.out, "Roadmap Milestones:") {
		t.Errorf("expected 'Roadmap Milestones:' in output: %s", r.out)
	}

	// 2. Machine-readable JSON output
	r = exec(t, "", false, "", "status", "--dir", dir, "--json")
	if r.code != 0 {
		t.Fatalf("expected exit 0 with --json, got %d: %s (err: %s)", r.code, r.out, r.err)
	}
	var res map[string]any
	if err := json.Unmarshal([]byte(r.out), &res); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, r.out)
	}
	if summary, ok := res["task_summary"].(map[string]any); !ok || summary["total"] != float64(2) {
		t.Errorf("expected task_summary.total = 2, got: %v", res["task_summary"])
	}

	// 3. Missing ROADMAP.md error
	emptyDir := t.TempDir()
	r = exec(t, "", false, "", "status", "--dir", emptyDir)
	if r.code != 1 {
		t.Fatalf("expected exit 1 for missing ROADMAP.md, got %d: %s (err: %s)", r.code, r.out, r.err)
	}
	if !strings.Contains(r.err, "ROADMAP.md not found") {
		t.Errorf("expected 'ROADMAP.md not found' in stderr, got: %s", r.err)
	}

	// 4. Unexpected positional argument
	r = exec(t, "", false, "", "status", "unexpected")
	if r.code != 2 {
		t.Fatalf("expected exit 2 for unexpected argument, got %d", r.code)
	}
	if !strings.Contains(r.err, `unexpected argument "unexpected"`) {
		t.Errorf("expected 'unexpected argument' in stderr, got: %s", r.err)
	}

	// 5. Invalid flag
	r = exec(t, "", false, "", "status", "--unknown-flag")
	if r.code != 2 {
		t.Fatalf("expected exit 2 for unknown flag, got %d", r.code)
	}
}

func TestTaskScaffolding(t *testing.T) {
	dir := t.TempDir()
	roadmapContent := `# ROADMAP — sdd

## 3. Milestones

| Milestone | Goal | Tasks | Est. hours | Exit criteria |
|---|---|---|---|---|
| M1 | Tasks | T-001, T-002, T-003, T-004 | 10 | Done |

## 4. Tasks

### M1 — Tasks

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-001 | Regular feature | S | none | Feature | none | Not started |
| T-002 | Bug in parser | S | none | Bugfix | none | Not started |
| T-003 | Production issue | S | none | Hotfix | none | Not started |
| T-004 | Tech spike | S | none | Spike | none | Not started |
`
	put(t, dir, "ROADMAP.md", roadmapContent)

	// 1. sdd bugfix new
	r := exec(t, "", false, "", "bugfix", "new", "T-002", "parser-bug", "--dir", dir)
	if r.code != 0 {
		t.Fatalf("expected exit 0 for bugfix new, got %d: %s (err: %s)", r.code, r.out, r.err)
	}
	if !exists(dir, "tasks/bugfixes/T-002-parser-bug/reproduction.md") {
		t.Errorf("expected reproduction.md to exist in tasks/bugfixes/T-002-parser-bug")
	}

	// 2. sdd hotfix new
	r = exec(t, "", false, "", "hotfix", "new", "T-003", "prod-fix", "--dir", dir)
	if r.code != 0 {
		t.Fatalf("expected exit 0 for hotfix new, got %d: %s (err: %s)", r.code, r.out, r.err)
	}
	if !exists(dir, "tasks/hotfixes/T-003-prod-fix/incident.md") {
		t.Errorf("expected incident.md to exist in tasks/hotfixes/T-003-prod-fix")
	}

	// 3. sdd spike new
	r = exec(t, "", false, "", "spike", "new", "T-004", "eval-tool", "--dir", dir)
	if r.code != 0 {
		t.Fatalf("expected exit 0 for spike new, got %d: %s (err: %s)", r.code, r.out, r.err)
	}
	if !exists(dir, "tasks/spikes/T-004-eval-tool/hypotheses.md") {
		t.Errorf("expected hypotheses.md to exist in tasks/spikes/T-004-eval-tool")
	}

	// 4. sdd task new --type feature
	r = exec(t, "", false, "", "task", "new", "--type", "feature", "T-001", "my-feature", "--dir", dir)
	if r.code != 0 {
		t.Fatalf("expected exit 0 for task new --type feature, got %d: %s (err: %s)", r.code, r.out, r.err)
	}
	if !exists(dir, "tasks/features/T-001-my-feature/requirements.md") {
		t.Errorf("expected requirements.md to exist in tasks/features/T-001-my-feature")
	}

	// 5. sdd task new with unknown typology
	r = exec(t, "", false, "", "task", "new", "--type", "unknown", "T-001", "fail", "--dir", dir)
	if r.code != 2 {
		t.Fatalf("expected exit 2 for unknown typology, got %d", r.code)
	}
}

func TestWorktreeCommands(t *testing.T) {
	dir := t.TempDir()

	// Initialise git repo
	runGit := func(args ...string) {
		cmd := osexec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v (%s)", strings.Join(args, " "), err, string(out))
		}
	}
	runGit("init")
	runGit("config", "user.name", "Test User")
	runGit("config", "user.email", "test@example.com")
	put(t, dir, "README.md", "# Repo\n")
	runGit("add", "README.md")
	runGit("commit", "-m", "init")

	// Create task folder
	taskFolder := filepath.Join(dir, "features", "T-001-worktree-cli")
	if err := os.MkdirAll(taskFolder, 0o755); err != nil {
		t.Fatal(err)
	}

	// 1. sdd task start
	r := exec(t, "", false, "", "task", "start", "T-001", "--dir", dir)
	if r.code != 0 {
		t.Fatalf("expected exit 0 for task start, got %d: %s (err: %s)", r.code, r.out, r.err)
	}
	if !strings.Contains(r.out, "Provisioned worktree for T-001:") {
		t.Errorf("expected provisioned confirmation, got:\n%s", r.out)
	}
	if !strings.Contains(r.out, "feature/T-001-worktree-cli") {
		t.Errorf("expected branch feature/T-001-worktree-cli, got:\n%s", r.out)
	}

	// 2. sdd task list --json
	r = exec(t, "", false, "", "task", "list", "--dir", dir, "--json")
	if r.code != 0 {
		t.Fatalf("expected exit 0 for task list, got %d: %s (err: %s)", r.code, r.out, r.err)
	}
	var listItems []map[string]any
	if err := json.Unmarshal([]byte(r.out), &listItems); err != nil {
		t.Fatalf("failed to parse task list JSON: %v\nOutput:\n%s", err, r.out)
	}
	if len(listItems) != 1 || listItems[0]["task_id"] != "T-001" {
		t.Errorf("unexpected list JSON: %+v", listItems)
	}

	// 3. sdd task list (text)
	r = exec(t, "", false, "", "task", "list", "--dir", dir)
	if r.code != 0 {
		t.Fatalf("expected exit 0 for task list text, got %d: %s", r.code, r.err)
	}
	if !strings.Contains(r.out, "Active task worktrees:") || !strings.Contains(r.out, "T-001") {
		t.Errorf("unexpected task list text:\n%s", r.out)
	}

	// 4. sdd task clean
	r = exec(t, "", false, "", "task", "clean", "T-001", "--dir", dir, "--delete-branch")
	if r.code != 0 {
		t.Fatalf("expected exit 0 for task clean, got %d: %s (err: %s)", r.code, r.out, r.err)
	}
	if !strings.Contains(r.out, "Cleaned worktree for T-001") {
		t.Errorf("expected clean confirmation, got:\n%s", r.out)
	}

	// 5. sdd task list after clean
	r = exec(t, "", false, "", "task", "list", "--dir", dir)
	if r.code != 0 {
		t.Fatalf("expected exit 0 for empty task list, got %d", r.code)
	}
	if !strings.Contains(r.out, "No active task worktrees.") {
		t.Errorf("expected empty message, got:\n%s", r.out)
	}
}

func TestTaskDeconflict(t *testing.T) {
	dir := t.TempDir()

	roadmapContent := `# ROADMAP — sdd

## 3. Milestones

| Milestone | Goal | Tasks | Est. hours | Exit criteria |
|---|---|---|---|---|
| M1 | Tasks | T-001, T-002 | 10 | Done |

## 4. Tasks

### M1 — Tasks

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-001 | First feature | S | none | Feature | none | Implementing |
| T-002 | Second feature | S | none | Feature | none | Implementing |
`
	put(t, dir, "ROADMAP.md", roadmapContent)

	task1Dir := filepath.Join(dir, "features", "T-001-first-feature")
	task2Dir := filepath.Join(dir, "features", "T-002-second-feature")
	if err := os.MkdirAll(task1Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(task2Dir, 0o755); err != nil {
		t.Fatal(err)
	}

	plan1 := `# Plan — T-001

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| ` + "`" + `cmd/main.go` + "`" + ` | modify | entry |
| internal/a.go | create | core |
`
	plan2Disjoint := `# Plan — T-002

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| ` + "`" + `internal/b.go` + "`" + ` | create | other |
`
	put(t, dir, "features/T-001-first-feature/plan.md", plan1)
	put(t, dir, "features/T-002-second-feature/plan.md", plan2Disjoint)

	// 1. Positional disjoint
	r := exec(t, "", false, "", "task", "deconflict", "T-001", "T-002", "--dir", dir)
	if r.code != 0 {
		t.Fatalf("expected exit 0 for disjoint deconflict, got %d: %s (err: %s)", r.code, r.out, r.err)
	}
	if !strings.Contains(r.out, "Concurrency Status: SAFE") {
		t.Errorf("expected SAFE concurrency status, got:\n%s", r.out)
	}
	if !strings.Contains(r.out, "Wave 1 (Parallel Batch): T-001, T-002") {
		t.Errorf("expected parallel batch in schedule, got:\n%s", r.out)
	}

	// 2. Overlapping plans
	plan2Overlap := `# Plan — T-002

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| ` + "`" + `cmd/main.go` + "`" + ` | modify | conflict |
| internal/b.go | create | other |
`
	put(t, dir, "features/T-002-second-feature/plan.md", plan2Overlap)

	r = exec(t, "", false, "", "task", "deconflict", "T-001", "T-002", "--dir", dir)
	if r.code != 0 {
		t.Fatalf("expected exit 0 for overlapping deconflict, got %d: %s", r.code, r.err)
	}
	if !strings.Contains(r.out, "COLLISION HAZARD") {
		t.Errorf("expected COLLISION HAZARD, got:\n%s", r.out)
	}
	if !strings.Contains(r.out, "cmd/main.go") {
		t.Errorf("expected cmd/main.go in conflict list, got:\n%s", r.out)
	}
	if !strings.Contains(r.out, "Wave 1") || !strings.Contains(r.out, "Wave 2") {
		t.Errorf("expected 2 waves in schedule, got:\n%s", r.out)
	}

	// 3. Auto-discovery of active in-flight tasks
	r = exec(t, "", false, "", "task", "deconflict", "--dir", dir)
	if r.code != 0 {
		t.Fatalf("expected exit 0 for auto-discovery, got %d: %s", r.code, r.err)
	}
	if !strings.Contains(r.out, "Analyzed: 2 task(s) (T-001, T-002)") {
		t.Errorf("expected 2 tasks analyzed, got:\n%s", r.out)
	}

	// 4. Invalid task ID argument
	r = exec(t, "", false, "", "task", "deconflict", "T-999", "T-888", "--dir", dir)
	if r.code != 1 {
		t.Fatalf("expected exit 1 for unknown task, got %d", r.code)
	}

	// 5. Only 1 positional argument
	r = exec(t, "", false, "", "task", "deconflict", "T-001", "--dir", dir)
	if r.code != 2 {
		t.Fatalf("expected exit 2 for single task query, got %d", r.code)
	}
}

func TestTaskDeconflictJSON(t *testing.T) {
	dir := t.TempDir()

	roadmapContent := `# ROADMAP — sdd

## 3. Milestones

| Milestone | Goal | Tasks | Est. hours | Exit criteria |
|---|---|---|---|---|
| M1 | Tasks | T-001, T-002 | 10 | Done |

## 4. Tasks

### M1 — Tasks

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-001 | First feature | S | none | Feature | none | Implementing |
| T-002 | Second feature | S | none | Feature | none | Implementing |
`
	put(t, dir, "ROADMAP.md", roadmapContent)

	task1Dir := filepath.Join(dir, "features", "T-001-first-feature")
	task2Dir := filepath.Join(dir, "features", "T-002-second-feature")
	_ = os.MkdirAll(task1Dir, 0o755)
	_ = os.MkdirAll(task2Dir, 0o755)

	plan1 := "# Plan — T-001\n\n## 5. Files to create or modify\n\n| Path | Action | Reason |\n|---|---|---|\n| cmd/main.go | modify | entry |\n"
	plan2 := "# Plan — T-002\n\n## 5. Files to create or modify\n\n| Path | Action | Reason |\n|---|---|---|\n| cmd/main.go | modify | conflict |\n"
	put(t, dir, "features/T-001-first-feature/plan.md", plan1)
	put(t, dir, "features/T-002-second-feature/plan.md", plan2)

	r := exec(t, "", false, "", "task", "deconflict", "T-001", "T-002", "--dir", dir, "--json")
	if r.code != 0 {
		t.Fatalf("expected exit 0 for json flag, got %d: %s", r.code, r.err)
	}

	var rep struct {
		HasConflict bool `json:"has_conflict"`
		Conflicts   []struct {
			TaskA       string   `json:"task_a"`
			TaskB       string   `json:"task_b"`
			SharedFiles []string `json:"shared_files"`
		} `json:"conflicts"`
		Batches [][]string `json:"batches"`
	}

	if err := json.Unmarshal([]byte(r.out), &rep); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput:\n%s", err, r.out)
	}

	if !rep.HasConflict {
		t.Errorf("expected has_conflict to be true")
	}
	if len(rep.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(rep.Conflicts))
	}
	if rep.Conflicts[0].TaskA != "T-001" || rep.Conflicts[0].TaskB != "T-002" {
		t.Errorf("unexpected conflict pair: %+v", rep.Conflicts[0])
	}
	if len(rep.Batches) != 2 {
		t.Errorf("expected 2 batches, got %+v", rep.Batches)
	}
}

func TestTaskMerge(t *testing.T) {
	dir := t.TempDir()

	runGit := func(args ...string) {
		cmd := osexec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v (%s)", strings.Join(args, " "), err, string(out))
		}
	}
	runGit("init")
	runGit("config", "user.name", "Test User")
	runGit("config", "user.email", "test@example.com")

	targetBranch := "main"
	// Check initial branch
	initBranchCmd := osexec.Command("git", "branch", "--show-current")
	initBranchCmd.Dir = dir
	if out, err := initBranchCmd.Output(); err == nil && strings.TrimSpace(string(out)) != "" {
		targetBranch = strings.TrimSpace(string(out))
	}

	put(t, dir, "go.mod", "module test.mod\n\ngo 1.27\n")
	put(t, dir, "README.md", "# Base\n")
	roadmapContent := `# ROADMAP — sdd

## 4. Tasks

### M1 — Tasks

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-001 | Test feature | S | none | outcome | none | Implementing |
`
	put(t, dir, "ROADMAP.md", roadmapContent)
	runGit("add", "go.mod", "README.md", "ROADMAP.md")
	runGit("commit", "-m", "init")

	// Task folder
	taskDir := filepath.Join(dir, "features", "T-001-test-feature")
	_ = os.MkdirAll(taskDir, 0o755)
	put(t, dir, "features/T-001-test-feature/plan.md", "# Plan — T-001\n\n## 5. Files to create or modify\n\n| Path | Action | Reason |\n|---|---|---|\n| feature.go | create | code |\n")
	put(t, dir, "features/T-001-test-feature/review.md", "# Code Review — T-001\nStatus: Approved\n\n## 3. Risk & Findings Register\n\n| ID | Category | Severity | Finding | Suggested Action | Status |\n|---|---|---|---|---|---|\n")

	// Feature branch
	runGit("checkout", "-b", "feature/T-001-test-feature")
	put(t, dir, "feature.go", "package main\n")
	runGit("add", "feature.go", "features/T-001-test-feature/plan.md", "features/T-001-test-feature/review.md")
	runGit("commit", "-m", "implement feature")

	// 1. Dry run
	r := exec(t, "", false, "", "task", "merge", "T-001", "--dir", dir, "--target", targetBranch, "--dry-run", "--skip-flake-check")
	if r.code != 0 {
		t.Fatalf("expected exit 0 for dry-run merge, got %d: %s (err: %s)", r.code, r.out, r.err)
	}
	if !strings.Contains(r.out, "Pre-Merge Qualification Report: T-001") {
		t.Errorf("expected qualification report in output:\n%s", r.out)
	}
	if !strings.Contains(r.out, "READY TO MERGE") {
		t.Errorf("expected READY TO MERGE in output:\n%s", r.out)
	}
	if !strings.Contains(r.out, "APPROVED MERGE T-001") {
		t.Errorf("expected APPROVED MERGE in presentation:\n%s", r.out)
	}

	// 2. JSON output
	r = exec(t, "", false, "", "task", "merge", "T-001", "--dir", dir, "--target", targetBranch, "--json", "--skip-flake-check")
	if r.code != 0 {
		t.Fatalf("expected exit 0 for json flag, got %d: %s (err: %s)", r.code, r.out, r.err)
	}
	var rep map[string]any
	if err := json.Unmarshal([]byte(r.out), &rep); err != nil {
		t.Fatalf("failed to parse JSON report: %v\nOutput:\n%s", err, r.out)
	}
	if rep["task_id"] != "T-001" || rep["ready_to_merge"] != true {
		t.Errorf("unexpected JSON report: %+v", rep)
	}

	// 3. Negative case: review blocker
	put(t, dir, "features/T-001-test-feature/review.md", "# Code Review — T-001\nStatus: Approved\n\n## 3. Risk & Findings Register\n\n| ID | Category | Severity | Finding | Suggested Action | Status |\n|---|---|---|---|---|---|\n| F-001 | Bug | 🔴 Blocker | defect | fix | Open |\n")
	runGit("add", "features/T-001-test-feature/review.md")
	runGit("commit", "-m", "add blocker")

	r = exec(t, "", false, "", "task", "merge", "T-001", "--dir", dir, "--target", targetBranch, "--dry-run", "--skip-flake-check")
	if r.code != 1 {
		t.Fatalf("expected exit 1 when review has blocker, got %d: %s", r.code, r.out)
	}
	if !strings.Contains(r.out, "NOT READY (BLOCKERS DETECTED)") {
		t.Errorf("expected blocker warning in output:\n%s", r.out)
	}
}

func TestPromptReview(t *testing.T) {
	root := t.TempDir()

	featDir := filepath.Join(root, "features", "T-001-test-feature")
	if err := os.MkdirAll(featDir, 0o755); err != nil {
		t.Fatal(err)
	}

	put(t, featDir, "requirements.md", "# Requirements — T-001\n\n| ID | Requirement |\n|---|---|\n| R-001 | The system SHALL work |\n")
	put(t, featDir, "plan.md", "# Plan — T-001\n\n## 5. Files to create or modify\n\n| Path | Action | Reason |\n|---|---|---|\n| internal/service/service.go | create | core service |\n| internal/service/service_test.go | create | tests |\n")
	put(t, featDir, "validation.md", "# Validation — T-001\n\nTraceability matrix\n")
	put(t, featDir, "review.md", "# Code Review — T-001\nStatus: Pending\n")

	put(t, root, "SKILLS.md", "# SKILLS\n\n## 8. Human feedback rules (binding; grows over time)\n\n### Register\n\n| ID | Date | Rule | Applies to | Source | Verified by | Status |\n|---|---|---|---|---|---|---|\n| F-001 | 2026-11-05 | No bare panics allowed | Go code | review | make test | Active |\n")

	// 1. Standard text handover prompt
	r := exec(t, "", false, "", "prompt", "review", "T-001", "--dir", root)
	if r.code != 0 {
		t.Fatalf("expected exit 0 for prompt review, got %d: %s (err: %s)", r.code, r.out, r.err)
	}
	if !strings.Contains(r.out, "# Handover: Independent Subagent Code & Spec Review for T-001") {
		t.Errorf("missing header in prompt: %s", r.out)
	}
	if !strings.Contains(r.out, "🔴 **Blocker") {
		t.Errorf("missing rubric in prompt: %s", r.out)
	}
	if !strings.Contains(r.out, "internal/service/service.go") {
		t.Errorf("missing changed file in prompt: %s", r.out)
	}
	if !strings.Contains(r.out, "F-001") || !strings.Contains(r.out, "No bare panics allowed") {
		t.Errorf("missing feedback rule in prompt: %s", r.out)
	}

	// 2. Unknown task ID failure
	r = exec(t, "", false, "", "prompt", "review", "T-999", "--dir", root)
	if r.code != 1 {
		t.Fatalf("expected exit 1 for unknown task, got %d", r.code)
	}
	if !strings.Contains(r.err, "not found across") {
		t.Errorf("expected not found error, got: %s", r.err)
	}

	// 3. Missing task argument failure
	r = exec(t, "", false, "", "prompt", "review", "--dir", root)
	if r.code != 2 {
		t.Fatalf("expected exit 2 for missing task argument, got %d", r.code)
	}

	// 4. Invalid task ID format
	r = exec(t, "", false, "", "prompt", "review", "invalid_task", "--dir", root)
	if r.code != 1 {
		t.Fatalf("expected exit 1 for invalid task format, got %d", r.code)
	}
}

func TestPromptReviewJSON(t *testing.T) {
	root := t.TempDir()

	hotfixDir := filepath.Join(root, "hotfixes", "T-042-prod-patch")
	if err := os.MkdirAll(hotfixDir, 0o755); err != nil {
		t.Fatal(err)
	}

	put(t, hotfixDir, "requirements.md", "# Requirements — T-042\n")
	put(t, hotfixDir, "patch.md", "# Patch Plan — T-042\n\n## 5. Files to create or modify\n\n| Path | Action | Reason |\n|---|---|---|\n| cmd/hotfix.go | modify | quickfix |\n")
	put(t, hotfixDir, "validation.md", "# Validation — T-042\n")
	put(t, hotfixDir, "review.md", "# Code Review — T-042\nStatus: Pending\n")

	r := exec(t, "", false, "", "prompt", "review", "T-042", "--dir", root, "--json")
	if r.code != 0 {
		t.Fatalf("expected exit 0 for prompt review --json, got %d: %s (err: %s)", r.code, r.out, r.err)
	}

	var ctx ReviewPromptContext
	if err := json.Unmarshal([]byte(r.out), &ctx); err != nil {
		t.Fatalf("failed to parse JSON context: %v\nOutput:\n%s", err, r.out)
	}

	if ctx.TaskID != "T-042" {
		t.Errorf("expected TaskID T-042, got %q", ctx.TaskID)
	}
	if ctx.Typology != "hotfixes" {
		t.Errorf("expected Typology hotfixes, got %q", ctx.Typology)
	}
	if ctx.TaskDir != "hotfixes/T-042-prod-patch" {
		t.Errorf("expected TaskDir hotfixes/T-042-prod-patch, got %q", ctx.TaskDir)
	}
	if len(ctx.ChangedFiles) != 1 || ctx.ChangedFiles[0] != "cmd/hotfix.go" {
		t.Errorf("expected cmd/hotfix.go in ChangedFiles, got %v", ctx.ChangedFiles)
	}
	if !strings.Contains(ctx.Prompt, "Handover: Independent Subagent Code & Spec Review for T-042") {
		t.Errorf("Prompt field missing expected text: %s", ctx.Prompt)
	}
}

func TestPromptReviewLens(t *testing.T) {
	root := t.TempDir()

	featDir := filepath.Join(root, "features", "T-001-lens-test")
	if err := os.MkdirAll(featDir, 0o755); err != nil {
		t.Fatal(err)
	}

	put(t, featDir, "requirements.md", "# Requirements — T-001\n")
	put(t, featDir, "plan.md", "# Plan — T-001\n\n## 5. Files to create or modify\n\n| Path | Action | Reason |\n|---|---|---|\n| pkg/auth.go | modify | auth |\n")
	put(t, featDir, "validation.md", "# Validation — T-001\n")
	put(t, featDir, "review.md", "# Code Review — T-001\n")

	// 1. Spec lens
	rSpec := exec(t, "", false, "", "prompt", "review", "T-001", "--dir", root, "--lens", "spec")
	if rSpec.code != 0 {
		t.Fatalf("expected 0 for lens spec, got %d: %s", rSpec.code, rSpec.err)
	}
	if !strings.Contains(rSpec.out, "Spec Auditor Lens") || !strings.Contains(rSpec.out, "Traceability:") {
		t.Errorf("unexpected output for lens spec:\n%s", rSpec.out)
	}
	if !strings.Contains(rSpec.out, "Per Rule F-001, all findings") {
		t.Errorf("missing Rule F-001 instruction in lens spec output:\n%s", rSpec.out)
	}
	if !strings.Contains(rSpec.out, "Parallel Review Triad Protocol") {
		t.Errorf("expected Parallel Review Triad Protocol in single-lens prompt:\n%s", rSpec.out)
	}

	// 2. Security lens
	rSec := exec(t, "", false, "", "prompt", "review", "T-001", "--dir", root, "--lens", "security")
	if rSec.code != 0 {
		t.Fatalf("expected 0 for lens security, got %d: %s", rSec.code, rSec.err)
	}
	if !strings.Contains(rSec.out, "AppSec Specialist Lens") || !strings.Contains(rSec.out, "Filesystem confinement") {
		t.Errorf("unexpected output for lens security:\n%s", rSec.out)
	}

	// 3. Appsec alias
	rAppsec := exec(t, "", false, "", "prompt", "review", "T-001", "--dir", root, "--lens", "appsec")
	if rAppsec.code != 0 {
		t.Fatalf("expected 0 for lens appsec alias, got %d: %s", rAppsec.code, rAppsec.err)
	}
	if !strings.Contains(rAppsec.out, "AppSec Specialist Lens") {
		t.Errorf("expected AppSec Specialist Lens for appsec alias:\n%s", rAppsec.out)
	}

	// 4. Pattern lens
	rPat := exec(t, "", false, "", "prompt", "review", "T-001", "--dir", root, "--lens", "pattern")
	if rPat.code != 0 {
		t.Fatalf("expected 0 for lens pattern, got %d: %s", rPat.code, rPat.err)
	}
	if !strings.Contains(rPat.out, "Pattern & Quality Auditor Lens") || !strings.Contains(rPat.out, "Architectural decision") {
		t.Errorf("unexpected output for lens pattern:\n%s", rPat.out)
	}

	// 5. All / default lens
	rAll := exec(t, "", false, "", "prompt", "review", "T-001", "--dir", root)
	if rAll.code != 0 {
		t.Fatalf("expected 0 for default lens, got %d: %s", rAll.code, rAll.err)
	}
	if !strings.Contains(rAll.out, "Ensemble Review Triad") {
		t.Errorf("expected Ensemble Review Triad in default prompt:\n%s", rAll.out)
	}

	// 6. Invalid lens
	rInv := exec(t, "", false, "", "prompt", "review", "T-001", "--dir", root, "--lens", "nonexistent")
	if rInv.code != 2 {
		t.Errorf("expected exit code 2 for invalid lens, got %d", rInv.code)
	}
	if !strings.Contains(rInv.err, "invalid --lens") || !strings.Contains(rInv.err, "spec, security, pattern, or all") {
		t.Errorf("expected invalid lens error message, got: %s", rInv.err)
	}

	// 7. Structured JSON with lens metadata
	rJSON := exec(t, "", false, "", "prompt", "review", "T-001", "--dir", root, "--lens", "security", "--json")
	if rJSON.code != 0 {
		t.Fatalf("expected 0 for json lens, got %d: %s", rJSON.code, rJSON.err)
	}
	var ctx ReviewPromptContext
	if err := json.Unmarshal([]byte(rJSON.out), &ctx); err != nil {
		t.Fatalf("failed to parse json output: %v", err)
	}
	if ctx.Lens != "security" {
		t.Errorf("expected Lens security, got %q", ctx.Lens)
	}
	if len(ctx.LensFocus) == 0 {
		t.Errorf("expected non-empty LensFocus in json output")
	}
}

func TestPromptPlanReview(t *testing.T) {
	root := t.TempDir()

	featDir := filepath.Join(root, "tasks", "features", "T-010-plan-test")
	if err := os.MkdirAll(featDir, 0o755); err != nil {
		t.Fatal(err)
	}

	put(t, featDir, "requirements.md", "# Requirements — T-010\n\n## 1. Goal\nTest goal\n")
	put(t, featDir, "plan.md", "# Plan — T-010\n\n## 5. Files to create or modify\n\n| Path | Action | Reason |\n|---|---|---|\n| internal/worker/worker.go | create | core logic |\n")
	put(t, featDir, "validation.md", "# Validation — T-010\n\nTraceability matrix\n")

	put(t, root, "TECH_STACK.md", "# TECH_STACK\n\n## 6. Decision register\n\n| ID | Decision | Status | Date | Outcome |\n|---|---|---|---|---|\n| D-007 | Standard library only | Approved | 2026-10-26 | Use stdlib only |\n")
	put(t, root, "SKILLS.md", "# SKILLS\n\n## 8. Human feedback rules\n\n### Register\n\n| ID | Date | Rule | Applies to | Source | Verified by | Status |\n|---|---|---|---|---|---|---|\n| F-001 | 2026-11-05 | No panics in production | Go code | review | make test | Active |\n")

	// 1. Successful plan review prompt
	r := exec(t, "", false, "", "prompt", "plan-review", "T-010", "--dir", root)
	if r.code != 0 {
		t.Fatalf("expected exit 0 for prompt plan-review, got %d: %s (err: %s)", r.code, r.out, r.err)
	}
	if !strings.Contains(r.out, "# Handover: Independent Pre-GATE 1 Plan Review for T-010") {
		t.Errorf("missing header in plan review prompt: %s", r.out)
	}
	if !strings.Contains(r.out, "Plan Review Rubric") {
		t.Errorf("missing rubric in plan review prompt: %s", r.out)
	}
	if !strings.Contains(r.out, "internal/worker/worker.go") {
		t.Errorf("missing planned file in prompt: %s", r.out)
	}
	if !strings.Contains(r.out, "D-007: Standard library only") {
		t.Errorf("missing decision in prompt: %s", r.out)
	}
	if !strings.Contains(r.out, "F-001") || !strings.Contains(r.out, "No panics in production") {
		t.Errorf("missing feedback rule in prompt: %s", r.out)
	}

	// 2. Unknown task ID failure
	r = exec(t, "", false, "", "prompt", "plan-review", "T-999", "--dir", root)
	if r.code != 1 {
		t.Fatalf("expected exit 1 for unknown task, got %d", r.code)
	}
	if !strings.Contains(r.err, "not found across") {
		t.Errorf("expected not found error, got: %s", r.err)
	}

	// 3. Missing task argument failure
	r = exec(t, "", false, "", "prompt", "plan-review", "--dir", root)
	if r.code != 2 {
		t.Fatalf("expected exit 2 for missing task argument, got %d", r.code)
	}
}

func TestPromptPlanReviewJSON(t *testing.T) {
	root := t.TempDir()

	featDir := filepath.Join(root, "tasks", "features", "T-010-plan-test")
	if err := os.MkdirAll(featDir, 0o755); err != nil {
		t.Fatal(err)
	}

	put(t, featDir, "requirements.md", "# Requirements — T-010\n")
	put(t, featDir, "plan.md", "# Plan — T-010\n\n## 5. Files to create or modify\n\n| Path | Action | Reason |\n|---|---|---|\n| internal/pkg/pkg.go | create | logic |\n")
	put(t, featDir, "validation.md", "# Validation — T-010\n")

	r := exec(t, "", false, "", "prompt", "plan-review", "T-010", "--dir", root, "--json")
	if r.code != 0 {
		t.Fatalf("expected exit 0 for prompt plan-review --json, got %d: %s (err: %s)", r.code, r.out, r.err)
	}

	var ctx PlanReviewPromptContext
	if err := json.Unmarshal([]byte(r.out), &ctx); err != nil {
		t.Fatalf("failed to parse json output: %v", err)
	}
	if ctx.TaskID != "T-010" {
		t.Errorf("expected TaskID T-010, got %q", ctx.TaskID)
	}
	if ctx.Typology != "features" {
		t.Errorf("expected Typology features, got %q", ctx.Typology)
	}
	if len(ctx.PlannedFiles) != 1 || ctx.PlannedFiles[0] != "internal/pkg/pkg.go" {
		t.Errorf("unexpected PlannedFiles: %v", ctx.PlannedFiles)
	}
	if len(ctx.Rubric) == 0 {
		t.Errorf("expected non-empty Rubric in json output")
	}
}
