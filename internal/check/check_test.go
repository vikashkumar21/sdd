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

func TestCheckPermissions(t *testing.T) {
	root := generate(t)

	// Valid settings.json produces no error
	claudeDir := filepath.Join(root, ".claude")
	_ = os.MkdirAll(claudeDir, 0o755)
	validJSON := `{"permissions":{"deny":["git commit *"]}}`
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(validJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range r.Findings {
		if f.Rule == "permissions-syntax" {
			t.Fatalf("unexpected permissions-syntax finding for valid JSON: %+v", f)
		}
	}

	// Corrupted settings.json produces permissions-syntax error
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte("{broken json"), 0o644); err != nil {
		t.Fatal(err)
	}
	rBad, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range rBad.Findings {
		if f.Rule == "permissions-syntax" && f.Severity == Error {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected permissions-syntax error in %+v", rBad.Findings)
	}
}

func TestCheckCRLFInvariance(t *testing.T) {
	root := generate(t)

	// Baseline run on pure LF files
	baseRes, err := Run(root)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if baseRes.Count(Error) != 0 {
		t.Fatalf("expected 0 errors on clean generated kit, got %d", baseRes.Count(Error))
	}

	// 1. Asymmetric CRLF test: convert only CLAUDE.md to CRLF
	claudePath := filepath.Join(root, "CLAUDE.md")
	claudeData, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}
	claudeCRLF := strings.ReplaceAll(string(claudeData), "\r\n", "\n")
	claudeCRLF = strings.ReplaceAll(claudeCRLF, "\n", "\r\n")
	if err := os.WriteFile(claudePath, []byte(claudeCRLF), 0o644); err != nil {
		t.Fatal(err)
	}

	asymRes, err := Run(root)
	if err != nil {
		t.Fatalf("Run failed on asymmetric CRLF: %v", err)
	}
	for _, f := range asymRes.Findings {
		if f.Rule == "parity-match" {
			t.Fatalf("parity check broke on asymmetric CRLF: %+v", f)
		}
	}

	// 2. Convert all markdown and JSON files to CRLF
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".json") {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			crlf := strings.ReplaceAll(string(data), "\r\n", "\n")
			crlf = strings.ReplaceAll(crlf, "\n", "\r\n")
			return os.WriteFile(path, []byte(crlf), 0o644)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	crlfRes, err := Run(root)
	if err != nil {
		t.Fatalf("Run failed on all CRLF: %v", err)
	}
	if crlfRes.Count(Error) != 0 {
		t.Fatalf("expected 0 errors on CRLF kit, got %d: %+v", crlfRes.Count(Error), crlfRes.Findings)
	}
	if len(crlfRes.Findings) != len(baseRes.Findings) {
		t.Errorf("findings count mismatch: LF has %d, CRLF has %d", len(baseRes.Findings), len(crlfRes.Findings))
	}
}

func TestCheckSkills(t *testing.T) {
	root := generate(t)

	// 1. Valid skill
	skillDir := filepath.Join(root, "skills", "code-review")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	validSKILL := `---
name: code-review
description: Review pull requests for security and style.
---

# Code Review Guide
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(validSKILL), 0o644); err != nil {
		t.Fatal(err)
	}

	rep, err := Run(root)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if rep.Count(Error) != 0 {
		t.Fatalf("expected 0 errors with valid skill, got %d: %+v", rep.Count(Error), rep.Findings)
	}

	// 2. Skill with name mismatching folder
	mismatchDir := filepath.Join(root, "skills", "folder-a")
	if err := os.MkdirAll(mismatchDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mismatchSKILL := `---
name: folder-b
description: Some description.
---
Body.
`
	if err := os.WriteFile(filepath.Join(mismatchDir, "SKILL.md"), []byte(mismatchSKILL), 0o644); err != nil {
		t.Fatal(err)
	}

	rep2, err := Run(root)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	foundMismatch := false
	for _, f := range rep2.Findings {
		if f.Rule == "skill-name" && strings.Contains(f.Message, "does not match parent directory") {
			foundMismatch = true
			break
		}
	}
	if !foundMismatch {
		t.Errorf("expected skill-name finding for directory mismatch: %+v", rep2.Findings)
	}

	// 3. Skill with missing frontmatter
	badFrontmatterDir := filepath.Join(root, "skills", "no-frontmatter")
	if err := os.MkdirAll(badFrontmatterDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badFrontmatterDir, "SKILL.md"), []byte("Just plain text"), 0o644); err != nil {
		t.Fatal(err)
	}

	rep3, err := Run(root)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	foundSyntax := false
	for _, f := range rep3.Findings {
		if f.Rule == "skill-frontmatter" {
			foundSyntax = true
			break
		}
	}
	if !foundSyntax {
		t.Errorf("expected skill-frontmatter finding for missing frontmatter: %+v", rep3.Findings)
	}
}

func TestCheckReviewDocument(t *testing.T) {
	root := generate(t)
	featureDir := filepath.Join(root, "features", "T-001-scope")
	if err := os.MkdirAll(featureDir, 0o755); err != nil {
		t.Fatal(err)
	}

	put(t, root, "features/T-001-scope/requirements.md", "# Requirements — T-001\nStatus: Complete\n")
	put(t, root, "features/T-001-scope/plan.md", "# Plan — T-001\nStatus: Implemented\nApproved by: Vikash\n")
	put(t, root, "features/T-001-scope/validation.md", "# Validation — T-001\nStatus: Complete\n")

	// Update ROADMAP T-001 to Awaiting review
	edit(t, root, "ROADMAP.md", func(s string) string {
		lines := strings.Split(s, "\n")
		for i, l := range lines {
			if strings.HasPrefix(l, "| T-001 |") {
				lines[i] = strings.Replace(l, "Not started", "Awaiting review", 1)
				break
			}
		}
		return strings.Join(lines, "\n")
	})

	// 1. review.md with an unresolved 🔴 Blocker
	badReview := `# Code Review — T-001
Status: Changes requested

## 3. Risk & Findings Register
| ID | Category | Severity | Finding | Suggested Action | Status |
|---|---|---|---|---|---|
| F-001 | Correctness | 🔴 Blocker | Missing bounds check | Add bounds check | Open |
`
	put(t, root, "features/T-001-scope/review.md", badReview)

	rep, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	foundBlocker := false
	for _, f := range rep.Findings {
		if f.Rule == "review" && strings.Contains(f.Message, "unresolved 🔴 Blocker") {
			foundBlocker = true
			if f.Severity != Error {
				t.Errorf("expected Blocker finding to be Error severity, got %v", f.Severity)
			}
		}
	}
	if !foundBlocker {
		t.Errorf("expected unresolved blocker error in findings: %+v", rep.Findings)
	}

	// 2. Resolve blocker and set status to Approved
	goodReview := `# Code Review — T-001
Status: Approved
Reviewer: Independent Subagent Auditor

## 3. Risk & Findings Register
| ID | Category | Severity | Finding | Suggested Action | Status |
|---|---|---|---|---|---|
| F-001 | Correctness | 🔴 Blocker | Missing bounds check | Add bounds check | Resolved |
`
	put(t, root, "features/T-001-scope/review.md", goodReview)

	repGood, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range repGood.Findings {
		if f.Rule == "review" && strings.Contains(f.Message, "Blocker") {
			t.Errorf("unexpected blocker error after resolution: %+v", f)
		}
	}

	// 3. Invalid status header
	invalidStatusReview := `# Code Review — T-001
Status: UnknownStatus

## 3. Risk & Findings Register
`
	put(t, root, "features/T-001-scope/review.md", invalidStatusReview)

	repInvalid, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	foundStatusWarn := false
	for _, f := range repInvalid.Findings {
		if f.Rule == "review" && strings.Contains(f.Message, "review.md status is") {
			foundStatusWarn = true
			break
		}
	}
	if !foundStatusWarn {
		t.Errorf("expected warning for invalid review status: %+v", repInvalid.Findings)
	}

	// 4. Approved review with self-approved reviewer triggers warning
	selfApprovedReview := `# Code Review — T-001
Status: Approved
Reviewer: Antigravity (self-audit)

## 3. Risk & Findings Register
`
	put(t, root, "features/T-001-scope/review.md", selfApprovedReview)

	repSelf, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	foundSelfWarn := false
	for _, f := range repSelf.Findings {
		if f.Rule == "review" && strings.Contains(f.Message, "self-approved") {
			foundSelfWarn = true
			if f.Severity != Warn {
				t.Errorf("expected self-approved reviewer to be Warn severity, got %v", f.Severity)
			}
			break
		}
	}
	if !foundSelfWarn {
		t.Errorf("expected warning for self-approved reviewer: %+v", repSelf.Findings)
	}

	// 5. Approved review with Multi-Lens Ensemble Review missing a lens triggers review-triad warning
	missingLensReview := `# Code Review — T-001
Status: Approved
Reviewer: Independent Subagent Reviewer

## 1. Multi-Lens Ensemble Review
### 🔎 Spec Auditor Lens
- [x] verified

## 2. Risk & Findings Register
`
	put(t, root, "features/T-001-scope/review.md", missingLensReview)

	repMissingLens, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	foundTriadWarn := false
	for _, f := range repMissingLens.Findings {
		if f.Rule == "review-triad" && strings.Contains(f.Message, "missing AppSec Specialist") {
			foundTriadWarn = true
			break
		}
	}
	if !foundTriadWarn {
		t.Errorf("expected warning for missing review lens in ensemble review: %+v", repMissingLens.Findings)
	}

	// 6. Review with unresolved Warning and Minor triggers review-findings warning (Rule F-001)
	unresolvedWarningsReview := `# Code Review — T-001
Status: Approved
Reviewer: Independent Subagent Reviewer

## 1. Multi-Lens Ensemble Review
### 🔎 Spec Auditor Lens
- [x] pass
### 🛡️ AppSec Specialist Lens
- [x] pass
### 📐 Pattern & Quality Auditor Lens
- [x] pass

## 2. Risk & Findings Register
| ID | Category | Severity | Finding | Suggested Action | Status |
|---|---|---|---|---|---|
| W-001 | Observability | 🟡 Warning | Missing logging in edge branch | Add log | Open |
| M-001 | Style | 🟢 Minor | Variable name could be more descriptive | Rename variable | Pending |
`
	put(t, root, "features/T-001-scope/review.md", unresolvedWarningsReview)

	repUnresolved, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	warnCount := 0
	for _, f := range repUnresolved.Findings {
		if f.Rule == "review-findings" && strings.Contains(f.Message, "Rule F-001") {
			warnCount++
		}
	}
	if warnCount != 2 {
		t.Errorf("expected 2 review-findings warnings for unresolved warning and minor, got %d (findings: %+v)", warnCount, repUnresolved.Findings)
	}

	// 7. Review with resolved and justified Warning/Minor passes cleanly
	resolvedReview := `# Code Review — T-001
Status: Approved
Reviewer: Independent Subagent Reviewer

## 1. Multi-Lens Ensemble Review
### 🔎 Spec Auditor Lens
- [x] pass
### 🛡️ AppSec Specialist Lens
- [x] pass
### 📐 Pattern & Quality Auditor Lens
- [x] pass

## 2. Risk & Findings Register
| ID | Category | Severity | Finding | Suggested Action | Status |
|---|---|---|---|---|---|
| W-001 | Observability | 🟡 Warning | Missing logging in edge branch | Add log | Resolved |
| M-001 | Style | 🟢 Minor | Variable name could be more descriptive | Rename variable | Justified |
`
	put(t, root, "features/T-001-scope/review.md", resolvedReview)

	repResolved, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range repResolved.Findings {
		if f.Rule == "review-findings" {
			t.Errorf("unexpected review-findings finding for resolved/justified review: %+v", f)
		}
	}

	// 8. Review with code syntax containing '<', descriptive 'Justified per D-032', and finding row mentioning Severity
	advancedResolvedReview := `# Code Review — T-001
Status: Approved
Reviewer: Independent Subagent Reviewer

## 1. Multi-Lens Ensemble Review
### 🔎 Spec Auditor Lens
- [x] pass
### 🛡️ AppSec Specialist Lens
- [x] pass
### 📐 Pattern & Quality Auditor Lens
- [x] pass

## 2. Risk & Findings Register
| ID | Category | Severity | Finding | Suggested Action | Status |
|---|---|---|---|---|---|
| W-001 | Bound Checks | 🟡 Warning | Check len(x) < 5 boundary condition | Add bound check | Justified per D-032 |
| M-001 | Severity Logic | 🟢 Minor | Severity parsing edge cases | Refine parsing | Waived by human |
`
	put(t, root, "features/T-001-scope/review.md", advancedResolvedReview)

	repAdv, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range repAdv.Findings {
		if f.Rule == "review-findings" {
			t.Errorf("unexpected review-findings finding for advanced resolved/justified review: %+v", f)
		}
	}
}

func TestContextBudgetGuard(t *testing.T) {
	root := generate(t)
	featureDir := filepath.Join(root, "features", "T-001-scope")
	if err := os.MkdirAll(featureDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// 70KB content (exceeds 65536 bytes)
	largeContent := "# Requirements — T-001\n" + strings.Repeat("a", 70000)
	put(t, root, "features/T-001-scope/requirements.md", largeContent)
	put(t, root, "features/T-001-scope/plan.md", "# Plan — T-001\nStatus: Implemented\nApproved by: Vikash\n")
	put(t, root, "features/T-001-scope/validation.md", "# Validation — T-001\nStatus: Complete\n")

	rep, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}

	foundBudgetWarn := false
	for _, f := range rep.Findings {
		if f.Rule == "context-budget" && strings.Contains(f.Message, "64KB context budget") {
			foundBudgetWarn = true
			if f.Severity != Warn {
				t.Errorf("expected context-budget severity to be WARN, got %v", f.Severity)
			}
		}
	}
	if !foundBudgetWarn {
		t.Errorf("expected context-budget warning, got: %+v", rep.Findings)
	}
}

func TestTypologyChecks(t *testing.T) {
	root := generate(t)

	// Update ROADMAP to include T-001 (bugfix), T-002 (hotfix), T-003 (spike)
	edit(t, root, "ROADMAP.md", func(s string) string {
		lines := strings.Split(s, "\n")
		var out []string
		for _, l := range lines {
			if strings.HasPrefix(l, "| T-001 |") {
				out = append(out, "| T-001 | Bugfix memory leak | S | none | outcome | none | Awaiting review |")
				out = append(out, "| T-002 | Hotfix panic | S | none | outcome | none | Specifying |")
				out = append(out, "| T-003 | Spike prototype | S | none | outcome | none | Specifying |")
			} else {
				out = append(out, l)
			}
		}
		return strings.Join(out, "\n")
	})

	// 1. Bugfix folder with unverified Step 0
	bugDir := filepath.Join(root, "bugfixes", "T-001-mem-leak")
	if err := os.MkdirAll(bugDir, 0o755); err != nil {
		t.Fatal(err)
	}
	put(t, root, "bugfixes/T-001-mem-leak/reproduction.md", "# Reproduction\n[ ] Yes, verified\n```bash\n<paste real failure output here>\n```\n")
	put(t, root, "bugfixes/T-001-mem-leak/rca.md", "# RCA\nRoot cause\n")
	put(t, root, "bugfixes/T-001-mem-leak/plan.md", "# Plan\nStatus: Implemented\nApproved by: Vikash\n")
	put(t, root, "bugfixes/T-001-mem-leak/validation.md", "# Validation\nStatus: Complete\n")
	put(t, root, "bugfixes/T-001-mem-leak/review.md", "# Review\nStatus: Approved\nReviewer: Independent Subagent Reviewer\n")

	// 2. Hotfix folder
	hotfixDir := filepath.Join(root, "hotfixes", "T-002-panic")
	if err := os.MkdirAll(hotfixDir, 0o755); err != nil {
		t.Fatal(err)
	}
	put(t, root, "hotfixes/T-002-panic/incident.md", "# Incident\n")
	put(t, root, "hotfixes/T-002-panic/patch.md", "# Patch\n")
	put(t, root, "hotfixes/T-002-panic/validation.md", "# Validation\n")
	put(t, root, "hotfixes/T-002-panic/post_mortem.md", "# Post-mortem\n")

	// 3. Spike folder
	spikeDir := filepath.Join(root, "spikes", "T-003-proto")
	if err := os.MkdirAll(spikeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	put(t, root, "spikes/T-003-proto/hypotheses.md", "# Hypotheses\n")
	put(t, root, "spikes/T-003-proto/findings.md", "# Findings\n")
	put(t, root, "spikes/T-003-proto/plan.md", "# Plan\n")

	rep, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}

	foundReproErr := false
	for _, f := range rep.Findings {
		if f.Rule == "bugfix-reproduction" {
			foundReproErr = true
			if f.Severity != Error {
				t.Errorf("expected bugfix-reproduction severity to be ERROR, got %v", f.Severity)
			}
		}
	}
	if !foundReproErr {
		t.Errorf("expected bugfix-reproduction error for unverified reproduction test, got findings: %+v", rep.Findings)
	}

	// Now fix reproduction.md with verified output
	put(t, root, "bugfixes/T-001-mem-leak/reproduction.md", "# Reproduction\n[x] Yes, verified\n```bash\npanic: runtime error\nexit status 2\n```\n")

	repGood, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range repGood.Findings {
		if f.Rule == "bugfix-reproduction" {
			t.Errorf("unexpected bugfix-reproduction error after verification: %+v", f)
		}
	}
}

func TestCheckWorktrees(t *testing.T) {
	root := generate(t)

	// 1. Invalid branch name & unknown task ID
	badReg := `{
  "schema": 1,
  "worktrees": [
    {
      "task_id": "T-999",
      "typology": "feature",
      "slug": "bad-task",
      "branch": "invalid_branch_name",
      "path": ".sdd/worktrees/T-999-bad-task",
      "base_ref": "HEAD",
      "created_at": "2026-11-05T17:00:00Z"
    }
  ]
}`
	put(t, root, ".sdd/worktrees.json", badReg)

	rep, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}

	foundBranchWarn := false
	foundTaskErr := false
	for _, f := range rep.Findings {
		if f.Rule == "worktree-branch" {
			foundBranchWarn = true
			if f.Severity != Warn {
				t.Errorf("expected worktree-branch to be WARN, got %v", f.Severity)
			}
		}
		if f.Rule == "worktree-registry" {
			foundTaskErr = true
			if f.Severity != Error {
				t.Errorf("expected worktree-registry to be ERROR, got %v", f.Severity)
			}
		}
	}
	if !foundBranchWarn {
		t.Errorf("expected worktree-branch warning, got: %+v", rep.Findings)
	}
	if !foundTaskErr {
		t.Errorf("expected worktree-registry error, got: %+v", rep.Findings)
	}

	// 2. Valid worktree entry
	goodReg := `{
  "schema": 1,
  "worktrees": [
    {
      "task_id": "T-001",
      "typology": "feature",
      "slug": "scope",
      "branch": "feature/T-001-scope",
      "path": ".sdd/worktrees/T-001-scope",
      "base_ref": "HEAD",
      "created_at": "2026-11-05T17:00:00Z"
    }
  ]
}`
	put(t, root, ".sdd/worktrees.json", goodReg)

	repGood, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range repGood.Findings {
		if f.Rule == "worktree-branch" || f.Rule == "worktree-registry" {
			t.Errorf("unexpected worktree finding for valid entry: %+v", f)
		}
	}
}

func TestCheckTaskOverlap(t *testing.T) {
	root := generate(t)

	// Add T-002 to ROADMAP.md
	edit(t, root, "ROADMAP.md", func(s string) string {
		lines := strings.Split(s, "\n")
		var out []string
		for _, l := range lines {
			if strings.HasPrefix(l, "| T-001 |") {
				out = append(out, "| T-001 | First feature | S | none | outcome | none | Implementing |")
				out = append(out, "| T-002 | Second feature | S | none | outcome | none | Implementing |")
			} else {
				out = append(out, l)
			}
		}
		return strings.Join(out, "\n")
	})

	// Create T-001 folder and plan
	t1Dir := filepath.Join(root, "features", "T-001-first-feature")
	_ = os.MkdirAll(t1Dir, 0o755)
	put(t, root, "features/T-001-first-feature/requirements.md", "# Req T-001\n")
	put(t, root, "features/T-001-first-feature/validation.md", "# Val T-001\n")
	plan1 := `# Plan — T-001
Status: Approved

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| internal/cli/cli.go | modify | cli |
| cmd/main.go | modify | main |

## 10. Approval record
Approved by: human Date: 2026-11-05 Phrase received: APPROVED T-001
`
	put(t, root, "features/T-001-first-feature/plan.md", plan1)

	// Create T-002 folder and plan
	t2Dir := filepath.Join(root, "features", "T-002-second-feature")
	_ = os.MkdirAll(t2Dir, 0o755)
	put(t, root, "features/T-002-second-feature/requirements.md", "# Req T-002\n")
	put(t, root, "features/T-002-second-feature/validation.md", "# Val T-002\n")

	// 1. Overlapping plan on internal/cli/cli.go
	plan2Overlap := `# Plan — T-002
Status: Approved

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| internal/cli/cli.go | modify | conflict |
| internal/other.go | create | new |

## 10. Approval record
Approved by: human Date: 2026-11-05 Phrase received: APPROVED T-002
`
	put(t, root, "features/T-002-second-feature/plan.md", plan2Overlap)

	rep, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}

	foundOverlap := false
	for _, f := range rep.Findings {
		if f.Rule == "task-overlap" {
			foundOverlap = true
			if f.Severity != Warn {
				t.Errorf("expected task-overlap finding to be WARN, got %v", f.Severity)
			}
			if !strings.Contains(f.Message, "concurrent tasks T-001 and T-002 share planned target file(s): internal/cli/cli.go") {
				t.Errorf("unexpected message: %s", f.Message)
			}
		}
	}
	if !foundOverlap {
		t.Errorf("expected task-overlap warning for overlapping tasks, got: %+v", rep.Findings)
	}

	// 2. Disjoint plans
	plan2Disjoint := `# Plan — T-002
Status: Approved

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| internal/disjoint.go | create | no conflict |

## 10. Approval record
Approved by: human Date: 2026-11-05 Phrase received: APPROVED T-002
`
	put(t, root, "features/T-002-second-feature/plan.md", plan2Disjoint)

	repDisjoint, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range repDisjoint.Findings {
		if f.Rule == "task-overlap" {
			t.Errorf("expected no task-overlap warning for disjoint tasks, got: %+v", f)
		}
	}
}
