package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/detect"
	"github.com/vikashkumar21/sdd/internal/scaffold"
	"github.com/vikashkumar21/sdd/internal/templates"
)

var now = time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)

func generate(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	set, err := templates.New("")
	if err != nil {
		t.Fatal(err)
	}
	det, _ := detect.Detect(root)
	a := scaffold.Answers{Project: "demo", Mode: scaffold.ModeGreenfield, Languages: []string{"go"}}
	p, err := scaffold.BuildPlan(root, a, det, set, now, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scaffold.Apply(p, now, ""); err != nil {
		t.Fatal(err)
	}
	return root
}

func edit(t *testing.T, root, rel string, fn func(string) string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(fn(string(data))), 0o644); err != nil {
		t.Fatal(err)
	}
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

func run(t *testing.T, root string) Report {
	t.Helper()
	r, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func has(r Report, sev Severity, substr string) bool {
	for _, f := range r.Findings {
		if f.Severity == sev && strings.Contains(f.Message+" "+f.File, substr) {
			return true
		}
	}
	return false
}

func dump(r Report) string {
	var sb strings.Builder
	for _, f := range r.Findings {
		sb.WriteString(string(f.Severity) + " " + f.File + " " + f.Message + "\n")
	}
	return sb.String()
}

func TestFreshKitHasNoErrors(t *testing.T) {
	r := run(t, generate(t))
	if r.Count(Error) != 0 {
		t.Fatalf("a freshly generated kit must have no errors:\n%s", dump(r))
	}
	if !has(r, Warn, "TODO(human)") {
		t.Errorf("expected a warning about TODO placeholders:\n%s", dump(r))
	}
	if has(r, Info, "changed since sdd generated") {
		t.Errorf("a freshly generated kit must show no drift:\n%s", dump(r))
	}
	if r.Failed(false) {
		t.Error("warnings alone must not fail a non-strict check")
	}
	if !r.Failed(true) {
		t.Error("strict mode must fail while TODO placeholders remain")
	}
}

func TestCheckIsDeterministic(t *testing.T) {
	root := generate(t)
	first := dump(run(t, root))
	second := dump(run(t, root))
	if first != second {
		t.Fatalf("two runs gave different reports:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}

// The dbguard kit was written by hand before this checker existed; it must pass.
func TestRealWorldDbguardKitPasses(t *testing.T) {
	r := run(t, filepath.Join("testdata", "dbguard-kit"))
	if r.Count(Error) != 0 {
		t.Fatalf("the dbguard kit must have no errors:\n%s", dump(r))
	}
}

func TestDetectsProblems(t *testing.T) {
	tests := []struct {
		name string
		mut  func(t *testing.T, root string)
		sev  Severity
		want string
	}{
		{"missing required file", func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, "MISSION.md")); err != nil {
				t.Fatal(err)
			}
		}, Error, "MISSION.md"},
		{"agent contract drift", func(t *testing.T, root string) {
			edit(t, root, "GEMINI.md", func(s string) string { return strings.Replace(s, "Never guess.", "Guess freely.", 1) })
		}, Error, "agent contract differs"},
		{"dangling decision reference", func(t *testing.T, root string) {
			edit(t, root, "ROADMAP.md", func(s string) string { return s + "\nSee D-099.\n" })
		}, Error, "D-099"},
		{"dangling task reference", func(t *testing.T, root string) {
			edit(t, root, "MISSION.md", func(s string) string { return s + "\nBlocked by T-042.\n" })
		}, Error, "T-042"},
		{"broken table", func(t *testing.T, root string) {
			edit(t, root, "SKILLS.md", func(s string) string {
				return strings.Replace(s, "| Date | Pitfall | Source / task |", "| Date | Pitfall | Source | task |", 1)
			})
		}, Error, "pipes"},
		{"unbalanced fence", func(t *testing.T, root string) {
			edit(t, root, "MISSION.md", func(s string) string { return s + "\n```\nunclosed\n" })
		}, Error, "unbalanced code fence"},
		{"missing feedback section", func(t *testing.T, root string) {
			edit(t, root, "SKILLS.md", func(s string) string {
				return strings.Replace(s, "## 8. Human feedback rules", "## 8. Something else", 1)
			})
		}, Error, "Human feedback rules"},
		{"bad status", func(t *testing.T, root string) {
			edit(t, root, "ROADMAP.md", func(s string) string { return strings.Replace(s, "| Not started |", "| Almost there |", 1) })
		}, Warn, "unrecognised status"},
		{"in-flight task without feature folder", func(t *testing.T, root string) {
			edit(t, root, "ROADMAP.md", func(s string) string { return strings.Replace(s, "| Not started |", "| Implementing |", 1) })
		}, Error, "has no features/T-001"},
		{"feature folder without task row", func(t *testing.T, root string) {
			for _, f := range []string{"requirements.md", "plan.md", "validation.md"} {
				put(t, root, "features/T-077-ghost/"+f, "# x\n")
			}
		}, Error, "T-077"},
		{"feature folder missing a file", func(t *testing.T, root string) {
			put(t, root, "features/T-001-a/requirements.md", "# x\n")
		}, Error, "missing plan.md"},
		{"bad folder name", func(t *testing.T, root string) {
			put(t, root, "features/Task_One/requirements.md", "# x\n")
		}, Error, "short-slug"},
		{"blank approval record past approval", func(t *testing.T, root string) {
			for _, f := range []string{"requirements.md", "plan.md", "validation.md"} {
				put(t, root, "features/T-001-a/"+f, "# x\nApproved by: ______\nNOT RUN\n")
			}
			edit(t, root, "ROADMAP.md", func(s string) string { return strings.Replace(s, "| Not started |", "| Awaiting review |", 1) })
		}, Warn, "approval record is still blank"},
		{"duplicate task id", func(t *testing.T, root string) {
			edit(t, root, "ROADMAP.md", func(s string) string {
				return strings.Replace(s, "\n| T-001 |", "\n| T-000 |", 1)
			})
		}, Error, "duplicate task ID"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := generate(t)
			tc.mut(t, root)
			r := run(t, root)
			if !has(r, tc.sev, tc.want) {
				t.Fatalf("expected %s containing %q, got:\n%s", tc.sev, tc.want, dump(r))
			}
		})
	}
}

func TestManagedBlockParityIgnoresTextOutsideTheBlock(t *testing.T) {
	root := generate(t)
	edit(t, root, "CLAUDE.md", func(s string) string { return "# My own header\n\nTeam notes.\n\n" + s + "\nMore notes.\n" })
	if r := run(t, root); r.Count(Error) != 0 {
		t.Fatalf("text outside the managed block must not affect parity:\n%s", dump(r))
	}
}

func TestEditedGeneratedFilesAreReportedAsInfoOnly(t *testing.T) {
	root := generate(t)
	edit(t, root, "MISSION.md", func(s string) string { return s + "\nMy edit.\n" })
	r := run(t, root)
	if !has(r, Info, "changed since sdd generated") || r.Count(Error) != 0 {
		t.Fatalf("expected an INFO note and no errors:\n%s", dump(r))
	}
}

func TestEmptyDirectoryReportsMissingFiles(t *testing.T) {
	r := run(t, t.TempDir())
	if r.Count(Error) < 5 {
		t.Fatalf("expected many missing-file errors:\n%s", dump(r))
	}
}

func TestNotADirectory(t *testing.T) {
	if _, err := Run(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCheckContextManifest(t *testing.T) {
	root := generate(t)

	// Corrupted context manifest
	contextDir := filepath.Join(root, ".sdd", "context")
	if err := os.MkdirAll(contextDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(contextDir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte("invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := run(t, root)
	if !has(r, Error, "parse manifest") {
		t.Fatalf("expected error on invalid manifest json:\n%s", dump(r))
	}

	// Missing staged file referenced in manifest
	validJSON := `{
  "schema": 1,
  "updated": "2026-10-30T10:00:00Z",
  "entries": [
    {
      "id": "ctx-001",
      "source": "raw-text",
      "origin": "test",
      "path": ".sdd/context/missing.txt"
    }
  ]
}`
	if err := os.WriteFile(manifestPath, []byte(validJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	r = run(t, root)
	if !has(r, Error, "staged context file ctx-001 does not exist") {
		t.Fatalf("expected error on missing staged file:\n%s", dump(r))
	}

	// Valid manifest with existing file
	if err := os.WriteFile(filepath.Join(contextDir, "missing.txt"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	r = run(t, root)
	if has(r, Error, "staged context file") {
		t.Fatalf("unexpected error when file exists:\n%s", dump(r))
	}
}

func TestCheckTablesWithPipesInCodeAndEscaped(t *testing.T) {
	root := generate(t)
	// Add a markdown file containing table rows with inline code pipes and escaped pipes
	content := `# Tables with Pipes

| Command | Description |
|---|---|
| ` + "`grep | wc -l`" + ` | Counts occurrences |
| Option A \| B | Multiple options |
`
	if err := os.WriteFile(filepath.Join(root, "PIPES.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	r := run(t, root)
	if has(r, Error, "columns") || has(r, Error, "pipes") {
		t.Fatalf("expected tables with code pipes and escaped pipes to pass cleanly:\n%s", dump(r))
	}
}

func TestCheckJSONOutput(t *testing.T) {
	root := generate(t)
	r, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}

	data, err := r.JSON(false)
	if err != nil {
		t.Fatalf("r.JSON(false) failed: %v", err)
	}

	var jr JSONReport
	if err := json.Unmarshal(data, &jr); err != nil {
		t.Fatalf("json.Unmarshal failed: %v\nJSON:\n%s", err, string(data))
	}

	if !jr.Valid {
		t.Errorf("expected Valid to be true for clean kit, got false")
	}
	if jr.Summary.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", jr.Summary.Errors)
	}

	// Now introduce an error
	if err := os.Remove(filepath.Join(root, "MISSION.md")); err != nil {
		t.Fatal(err)
	}
	rFail, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	failData, err := rFail.JSON(false)
	if err != nil {
		t.Fatal(err)
	}
	var jrFail JSONReport
	if err := json.Unmarshal(failData, &jrFail); err != nil {
		t.Fatal(err)
	}
	if jrFail.Valid {
		t.Errorf("expected Valid to be false when MISSION.md is missing")
	}
	if jrFail.Summary.Errors == 0 {
		t.Errorf("expected >0 errors, got %d", jrFail.Summary.Errors)
	}
	found := false
	for _, f := range jrFail.Findings {
		if f.Severity == Error && f.File == "MISSION.md" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected finding for MISSION.md in %+v", jrFail.Findings)
	}
}

func TestCheckVerifiedFactsExpiry(t *testing.T) {
	root := generate(t)

	// Ensure ROADMAP.md has T-001 (Done) and T-002 (Not started)
	roadPath := filepath.Join(root, "ROADMAP.md")
	roadContent := `
### M1 — Milestone

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-001 | Expired task | S | none | done | D-001 | Done |
| T-002 | Active task | S | T-001 | pending | none | Not started |
`
	if err := os.WriteFile(roadPath, []byte(roadContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Append a verified facts section to TECH_STACK.md with one expired fact (T-001 is Done)
	// and one active fact (T-002 is Not started) and one permanent fact
	facts := `
## 7. Verified facts log (re-verify at the task shown; do not treat as permanent)

| Date | Fact | Source | Re-verify at |
|---|---|---|---|
| 2026-10-26 | Go builds cleanly | Sandbox | T-001 |
| 2026-10-26 | Windows compatibility check | Docs | T-002 |
| 2026-10-26 | Permanent constant | Specs | permanent |
`
	stackPath := filepath.Join(root, "TECH_STACK.md")
	data, err := os.ReadFile(stackPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stackPath, append(data, []byte(facts)...), 0o644); err != nil {
		t.Fatal(err)
	}

	r := run(t, root)
	if !has(r, Warn, "past due for re-verification (T-001 is Done)") {
		t.Fatalf("expected warning for expired fact:\n%s", dump(r))
	}
	if has(r, Warn, "T-002 is Done") {
		t.Fatalf("did not expect warning for active fact T-002:\n%s", dump(r))
	}
}

func TestCheckVerifiedFactsMilestoneExpiry(t *testing.T) {
	root := generate(t)

	facts := `
## 7. Verified facts log (re-verify at the task shown; do not treat as permanent)

| Date | Fact | Source | Re-verify at |
|---|---|---|---|
| 2026-10-26 | Milestone 1 fact | Sandbox | M1 |
`
	stackPath := filepath.Join(root, "TECH_STACK.md")
	data, err := os.ReadFile(stackPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stackPath, append(data, []byte(facts)...), 0o644); err != nil {
		t.Fatal(err)
	}

	// Update ROADMAP.md so all tasks in M1 are Done
	roadmapPath := filepath.Join(root, "ROADMAP.md")
	roadContent := `
### M1 — First Milestone

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-001 | Init | S | none | done | D-001 | Done |
| T-002 | Next | S | T-001 | done | none | Done |
`
	if err := os.WriteFile(roadmapPath, []byte(roadContent), 0o644); err != nil {
		t.Fatal(err)
	}

	r := run(t, root)
	if !has(r, Warn, "past due for re-verification (milestone M1 is completed)") {
		t.Fatalf("expected milestone expiry warning:\n%s", dump(r))
	}
}

func TestCheckDependencyDrift(t *testing.T) {
	root := generate(t)
	roadmapPath := filepath.Join(root, "ROADMAP.md")
	// T-002 is Done or Implementing, but T-001 prerequisite is Not started
	roadContent := `
### M1 — Milestone

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-001 | Prerequisite | S | none | pending | none | Not started |
| T-002 | Dependent | S | T-001 | done | none | Done |
`
	if err := os.WriteFile(roadmapPath, []byte(roadContent), 0o644); err != nil {
		t.Fatal(err)
	}

	r := run(t, root)
	if !has(r, Warn, "T-002 is Done but prerequisite T-001 is Not started") {
		t.Fatalf("expected prerequisite drift warning:\n%s", dump(r))
	}
}

func TestCheckFeatureStatusDrift(t *testing.T) {
	root := generate(t)
	// Create a feature folder for T-001 with incomplete status
	featDir := filepath.Join(root, "features", "T-001-sample")
	if err := os.MkdirAll(featDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(featDir, "requirements.md"), []byte("# Req\nStatus: Approved\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(featDir, "plan.md"), []byte("# Plan\nStatus: Draft\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(featDir, "validation.md"), []byte("# Val\nStatus: In progress\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	roadmapPath := filepath.Join(root, "ROADMAP.md")
	roadContent := `
### M1 — Milestone

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-001 | Sample | S | none | done | none | Done |
`
	if err := os.WriteFile(roadmapPath, []byte(roadContent), 0o644); err != nil {
		t.Fatal(err)
	}

	r := run(t, root)
	if !has(r, Warn, "task is Done in ROADMAP.md but validation.md status is") {
		t.Fatalf("expected validation status drift warning:\n%s", dump(r))
	}
	if !has(r, Warn, "task is Done in ROADMAP.md but plan.md status is") {
		t.Fatalf("expected plan status drift warning:\n%s", dump(r))
	}
}

func TestCheckJSONIncludesExpiryWarnings(t *testing.T) {
	root := generate(t)
	// Ensure T-001 is Done in ROADMAP.md
	roadPath := filepath.Join(root, "ROADMAP.md")
	roadContent := `
### M1 — Milestone

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-001 | Init | S | none | done | D-001 | Done |
`
	_ = os.WriteFile(roadPath, []byte(roadContent), 0o644)

	facts := `
## 7. Verified facts log (re-verify at the task shown; do not treat as permanent)

| Date | Fact | Source | Re-verify at |
|---|---|---|---|
| 2026-10-26 | Expired fact | Sandbox | T-001 |
`
	stackPath := filepath.Join(root, "TECH_STACK.md")
	data, _ := os.ReadFile(stackPath)
	_ = os.WriteFile(stackPath, append(data, []byte(facts)...), 0o644)

	r, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	jsonData, err := r.JSON(false)
	if err != nil {
		t.Fatal(err)
	}
	var jr JSONReport
	if err := json.Unmarshal(jsonData, &jr); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range jr.Findings {
		if f.Severity == Warn && strings.Contains(f.Message, "past due for re-verification") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected finding in JSONReport for expired fact: %+v", jr.Findings)
	}
}

func TestSuppressedFindingsPassStrict(t *testing.T) {
	root := generate(t)

	// Introduce contract parity divergence in GEMINI.md
	geminiPath := filepath.Join(root, "GEMINI.md")
	data, err := os.ReadFile(geminiPath)
	if err != nil {
		t.Fatal(err)
	}
	modData := strings.Replace(string(data), "ANTHEM: ALWAYS BE DETERMINISTIC.", "ANTHEM: DIFFERENT RULE.", 1)
	if err := os.WriteFile(geminiPath, []byte(modData), 0o644); err != nil {
		t.Fatal(err)
	}

	// Verify it fails check without suppression
	rUnsuppressed, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	if !rUnsuppressed.Failed(false) {
		t.Fatalf("expected unsuppressed parity error to fail check")
	}

	// Now register waivers in .sdd/suppressions.json (including todo placeholders for strict test)
	suppJSON := `{
  "schema": 1,
  "suppressions": [
    {
      "file": "GEMINI.md",
      "rule": "contract-parity",
      "reason": "intentional divergence for testing",
      "date": "2026-11-02"
    },
    {
      "file": "*",
      "rule": "todo-placeholder",
      "reason": "test fixtures have unedited todos",
      "date": "2026-11-02"
    }
  ]
}`
	sddDir := filepath.Join(root, ".sdd")
	_ = os.MkdirAll(sddDir, 0o755)
	if err := os.WriteFile(filepath.Join(sddDir, "suppressions.json"), []byte(suppJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run check again
	rSuppressed, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}

	// Should pass even in strict mode
	if rSuppressed.Failed(true) {
		t.Fatalf("expected check to pass strict mode when finding is suppressed: %+v", rSuppressed.Findings)
	}

	// Verify finding is downgraded to INFO and includes reason
	found := false
	for _, f := range rSuppressed.Findings {
		if f.File == "GEMINI.md" && f.Rule == "contract-parity" {
			found = true
			if f.Severity != Info {
				t.Errorf("expected Severity to be INFO, got %s", f.Severity)
			}
			if !f.Suppressed {
				t.Errorf("expected Suppressed to be true")
			}
			if f.SuppressReason != "intentional divergence for testing" {
				t.Errorf("unexpected SuppressReason: %s", f.SuppressReason)
			}
			if !strings.Contains(f.Message, "(suppressed: intentional divergence for testing)") {
				t.Errorf("unexpected message: %s", f.Message)
			}
		}
	}
	if !found {
		t.Fatalf("expected suppressed finding for GEMINI.md in %+v", rSuppressed.Findings)
	}
}

func TestCheckJSONIncludesSuppressionMetadata(t *testing.T) {
	root := generate(t)

	// Suppress README.md missing warning
	_ = os.Remove(filepath.Join(root, "README.md"))
	suppJSON := `{
  "schema": 1,
  "suppressions": [
    {
      "file": "README.md",
      "rule": "required-file",
      "reason": "omitted for custom doc site",
      "date": "2026-11-02"
    }
  ]
}`
	sddDir := filepath.Join(root, ".sdd")
	_ = os.MkdirAll(sddDir, 0o755)
	_ = os.WriteFile(filepath.Join(sddDir, "suppressions.json"), []byte(suppJSON), 0o644)

	r, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	data, err := r.JSON(false)
	if err != nil {
		t.Fatal(err)
	}

	var jr JSONReport
	if err := json.Unmarshal(data, &jr); err != nil {
		t.Fatal(err)
	}

	if !jr.Valid {
		t.Errorf("expected jr.Valid=true when warning is suppressed")
	}

	found := false
	for _, f := range jr.Findings {
		if f.File == "README.md" && f.Suppressed {
			found = true
			if f.SuppressReason != "omitted for custom doc site" {
				t.Errorf("unexpected reason in JSON: %s", f.SuppressReason)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected suppressed finding in JSON report: %s", string(data))
	}
}

func TestCheckStaleSuppressionDetection(t *testing.T) {
	root := generate(t)

	// Clean kit has no errors; register a suppression for a non-existent issue
	suppJSON := `{
  "schema": 1,
  "suppressions": [
    {
      "file": "CLAUDE.md",
      "rule": "contract-parity",
      "reason": "stale waiver that is not triggered",
      "date": "2026-11-02"
    }
  ]
}`
	sddDir := filepath.Join(root, ".sdd")
	_ = os.MkdirAll(sddDir, 0o755)
	_ = os.WriteFile(filepath.Join(sddDir, "suppressions.json"), []byte(suppJSON), 0o644)

	r, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}

	foundStale := false
	for _, f := range r.Findings {
		if f.Rule == "stale-waiver" && strings.Contains(f.Message, "CLAUDE.md") {
			foundStale = true
			if f.Severity != Info {
				t.Errorf("expected stale waiver severity to be INFO, got %s", f.Severity)
			}
			break
		}
	}
	if !foundStale {
		t.Fatalf("expected stale-waiver finding in %+v", r.Findings)
	}
}
