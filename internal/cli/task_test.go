package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vikashkumar21/sdd/internal/taskpr"
)

func setupTaskForPRTest(t *testing.T, taskID, slug, reviewStatus string, blockers int) string {
	t.Helper()
	root := t.TempDir()

	taskDir := filepath.Join(root, "tasks", "features", taskID+"-"+slug)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}

	reqContent := "# Requirements — " + taskID + " Test CI PR Packaging\n\n## 1. Goal\nSynthesize task documents into production PRs.\n"
	if err := os.WriteFile(filepath.Join(taskDir, "requirements.md"), []byte(reqContent), 0o644); err != nil {
		t.Fatal(err)
	}

	planContent := "# Plan — " + taskID + "\n\n## 5. Files to touch\n\n| File | Action | Purpose |\n|---|---|---|\n| `internal/ci/ci.go` | create | ci engine |\n| `cmd/sdd/main.go` | modify | entrypoint |\n"
	if err := os.WriteFile(filepath.Join(taskDir, "plan.md"), []byte(planContent), 0o644); err != nil {
		t.Fatal(err)
	}

	valContent := "# Validation — " + taskID + "\n\n## 1. Traceability matrix\n\n| ID | Method | Command | Expected | Actual | Status |\n|---|---|---|---|---|---|\n| R-001 | test | `make test` | pass | pass | PASS |\n"
	if err := os.WriteFile(filepath.Join(taskDir, "validation.md"), []byte(valContent), 0o644); err != nil {
		t.Fatal(err)
	}

	if reviewStatus != "" {
		revContent := "# Code Review — " + taskID + "\nStatus: " + reviewStatus + "\n\n## 2. Risk & Findings Register\n\n| ID | Category | Severity | Finding | Suggested Action | Status |\n|---|---|---|---|---|---|\n"
		for i := 0; i < blockers; i++ {
			revContent += "| B-001 | Defect | 🔴 Blocker | critical defect | fix | Open |\n"
		}
		if err := os.WriteFile(filepath.Join(taskDir, "review.md"), []byte(revContent), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

func TestTaskPR_Usage(t *testing.T) {
	var out, errOut bytes.Buffer
	env := IO{Out: &out, Err: &errOut}

	code := runTaskPR([]string{}, env)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(errOut.String(), "usage: sdd task pr <T-xxx>") {
		t.Errorf("expected usage message, got %q", errOut.String())
	}
}

func TestTaskPR_Qualified_MarkdownAndDryRun(t *testing.T) {
	root := setupTaskForPRTest(t, "T-046", "ci-packaging", "Approved", 0)

	var out, errOut bytes.Buffer
	env := IO{Out: &out, Err: &errOut}

	code := runTaskPR([]string{"T-046", "--dir", root, "--dry-run"}, env)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, errOut.String())
	}

	output := out.String()
	if !strings.Contains(output, "Pull Request Title:") {
		t.Errorf("expected PR Title header, got: %s", output)
	}
	if !strings.Contains(output, "feat(T-046): Test CI PR Packaging") {
		t.Errorf("expected formatted PR title, got: %s", output)
	}
	if !strings.Contains(output, "Synthesize task documents into production PRs") {
		t.Errorf("expected summary in output, got: %s", output)
	}
	if !strings.Contains(output, "QUALIFIED") {
		t.Errorf("expected QUALIFIED status in output, got: %s", output)
	}
}

func TestTaskPR_JSON(t *testing.T) {
	root := setupTaskForPRTest(t, "T-046", "ci-packaging", "Approved", 0)

	var out, errOut bytes.Buffer
	env := IO{Out: &out, Err: &errOut}

	code := runTaskPR([]string{"T-046", "--dir", root, "--json"}, env)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, errOut.String())
	}

	var bundle taskpr.PRBundle
	if err := json.Unmarshal(out.Bytes(), &bundle); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v (raw: %s)", err, out.String())
	}

	if bundle.TaskID != "T-046" {
		t.Errorf("task ID = %q, want T-046", bundle.TaskID)
	}
	if bundle.Title != "feat(T-046): Test CI PR Packaging" {
		t.Errorf("title = %q", bundle.Title)
	}
	if !bundle.QualifiesForPR {
		t.Errorf("expected QualifiesForPR=true")
	}
	if len(bundle.ChangedFiles) != 2 {
		t.Errorf("expected 2 changed files, got %d", len(bundle.ChangedFiles))
	}
}

func TestTaskPR_Unreviewed_FailCode1(t *testing.T) {
	root := setupTaskForPRTest(t, "T-046", "ci-packaging", "Pending", 1)

	var out, errOut bytes.Buffer
	env := IO{Out: &out, Err: &errOut}

	// In text mode: prints output, logs warning to stderr, exits 1
	code := runTaskPR([]string{"T-046", "--dir", root}, env)
	if code != 1 {
		t.Fatalf("expected exit code 1 for task with blockers, got %d", code)
	}
	if !strings.Contains(errOut.String(), "Task PR Qualification FAILED") {
		t.Errorf("expected failure message in stderr, got: %s", errOut.String())
	}

	// In JSON mode: prints JSON bundle with qualifies_for_pr: false, exits 1
	out.Reset()
	errOut.Reset()
	code = runTaskPR([]string{"T-046", "--dir", root, "--json"}, env)
	if code != 1 {
		t.Fatalf("expected exit code 1 in JSON mode, got %d", code)
	}
	var bundle taskpr.PRBundle
	if err := json.Unmarshal(out.Bytes(), &bundle); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if bundle.QualifiesForPR {
		t.Errorf("expected qualifies_for_pr=false")
	}
	if len(bundle.Warnings) == 0 {
		t.Errorf("expected warnings in JSON bundle")
	}
}

func TestTaskPR_AllowUnreviewed(t *testing.T) {
	root := setupTaskForPRTest(t, "T-046", "ci-packaging", "Pending", 1)

	var out, errOut bytes.Buffer
	env := IO{Out: &out, Err: &errOut}

	code := runTaskPR([]string{"T-046", "--dir", root, "--allow-unreviewed"}, env)
	if code != 0 {
		t.Fatalf("expected exit code 0 with --allow-unreviewed, got %d (stderr: %s)", code, errOut.String())
	}
}

func TestTaskPR_OutputFile(t *testing.T) {
	root := setupTaskForPRTest(t, "T-046", "ci-packaging", "Approved", 0)
	outFile := filepath.Join(root, "pr_description.md")

	var out, errOut bytes.Buffer
	env := IO{Out: &out, Err: &errOut}

	code := runTaskPR([]string{"T-046", "--dir", root, "--output", outFile}, env)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, errOut.String())
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if !strings.Contains(string(data), "## Summary") {
		t.Errorf("expected Summary in generated file, got: %s", string(data))
	}
}

func TestBugfixNew_FromCI(t *testing.T) {
	root := t.TempDir()
	var out, errOut bytes.Buffer
	env := IO{Out: &out, Err: &errOut}

	// 1. Initialize kit
	initCode := Run([]string{"init", "--yes", "--dir", root, "--name", "demo"}, env)
	if initCode != 0 {
		t.Fatalf("init failed: %s", errOut.String())
	}

	ciLog := `=== RUN   TestAuthRefresh
--- FAIL: TestAuthRefresh (0.01s)
    auth_test.go:10: refresh token expired
FAIL
exit status 1
`
	ciFile := filepath.Join(t.TempDir(), "test-ci.log")
	if err := os.WriteFile(ciFile, []byte(ciLog), 0o644); err != nil {
		t.Fatal(err)
	}

	// 2. bugfix new with --from-ci <file>
	out.Reset()
	errOut.Reset()
	code := Run([]string{"bugfix", "new", "T-001", "auth-refresh-fix", "--from-ci", ciFile, "--dir", root}, env)
	if code != 0 {
		t.Fatalf("bugfix new --from-ci failed with %d: %s", code, errOut.String())
	}

	reproPath := filepath.Join(root, "tasks", "bugfixes", "T-001-auth-refresh-fix", "reproduction.md")
	reproBytes, err := os.ReadFile(reproPath)
	if err != nil {
		t.Fatalf("failed to read reproduction.md: %v", err)
	}
	repro := string(reproBytes)
	if !strings.Contains(repro, "TestAuthRefresh") {
		t.Errorf("expected reproduction.md to contain TestAuthRefresh:\n%s", repro)
	}
	if !strings.Contains(repro, "refresh token expired") {
		t.Errorf("expected reproduction.md to contain failure snippet:\n%s", repro)
	}

	// 3. task new --type bugfix with --from-ci <staged-id>
	// Append T-002 row to ROADMAP.md
	roadmapPath := filepath.Join(root, "ROADMAP.md")
	rmContent, err := os.ReadFile(roadmapPath)
	if err != nil {
		t.Fatal(err)
	}
	newRM := string(rmContent) + "\n| T-002 | Auth error fix | S | T-001 | fix | none | Not started |\n"
	if err := os.WriteFile(roadmapPath, []byte(newRM), 0o644); err != nil {
		t.Fatal(err)
	}

	out.Reset()
	errOut.Reset()
	code = Run([]string{"task", "new", "--type", "bugfix", "T-002", "auth-staged-fix", "--from-ci", "ctx-001", "--dir", root}, env)
	if code != 0 {
		t.Fatalf("task new --type bugfix --from-ci failed with %d: %s", code, errOut.String())
	}

	reproPath2 := filepath.Join(root, "tasks", "bugfixes", "T-002-auth-staged-fix", "reproduction.md")
	reproBytes2, err := os.ReadFile(reproPath2)
	if err != nil {
		t.Fatalf("failed to read reproduction.md for T-002: %v", err)
	}
	repro2 := string(reproBytes2)
	if !strings.Contains(repro2, "TestAuthRefresh") {
		t.Errorf("expected reproduction.md for T-002 to contain TestAuthRefresh:\n%s", repro2)
	}

	// 4. Negative test: non-existent file or context ID
	out.Reset()
	errOut.Reset()
	code = Run([]string{"bugfix", "new", "T-003", "auth-missing-fix", "--from-ci", "non-existent-file.log", "--dir", root}, env)
	if code != 1 {
		t.Errorf("expected exit code 1 for non-existent --from-ci target, got %d", code)
	}
	if !strings.Contains(errOut.String(), "read ci log file") {
		t.Errorf("expected read error message, got: %s", errOut.String())
	}
}
