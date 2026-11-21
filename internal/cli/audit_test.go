package cli

import (
	"bytes"
	"encoding/json"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vikashkumar21/sdd/internal/audit"
)

func setupAuditTestProject(t *testing.T, taskID, slug string) string {
	t.Helper()
	tempDir := t.TempDir()

	taskDir := filepath.Join(tempDir, "tasks", "features", taskID+"-"+slug)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("failed to create task dir: %v", err)
	}

	reqContent := `# Requirements — ` + taskID + `
Status: Approved
## 1. Goal
Test goal
`
	planContent := `# Plan — ` + taskID + `
## 4. Implementation steps
| # | Step | Files | Proving test |
|---|---|---|---|
| 1 | Step 1 | internal/test.go | git status |
## 5. Files to create or modify
| Path | Action | Reason |
|---|---|---|
| internal/test.go | create | test file |
`
	valContent := `# Validation — ` + taskID + `
## 1. Traceability matrix
| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | ` + "`git status`" + ` | clean | clean | Pass |
| R-002 | test | ` + "`git status && git diff`" + ` | clean | clean | Pass |

## 2. Commands run
| Command | Result |
|---|---|
| ` + "`git status`" + ` | clean |
`
	revContent := `# Code Review — ` + taskID + `
Status: Pending
`

	_ = os.WriteFile(filepath.Join(taskDir, "requirements.md"), []byte(reqContent), 0o644)
	_ = os.WriteFile(filepath.Join(taskDir, "plan.md"), []byte(planContent), 0o644)
	_ = os.WriteFile(filepath.Join(taskDir, "validation.md"), []byte(valContent), 0o644)
	_ = os.WriteFile(filepath.Join(taskDir, "review.md"), []byte(revContent), 0o644)

	// Create git repo in tempDir
	cmd := osexec.Command("git", "init")
	cmd.Dir = tempDir
	_ = cmd.Run()

	return tempDir
}

func TestTaskAudit(t *testing.T) {
	root := setupAuditTestProject(t, "T-077", "audit-feature")

	// 1. Successful task audit run
	var outBuf, errBuf bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &outBuf,
		Err: &errBuf,
	}

	code := runTaskAudit([]string{"T-077", "--dir", root}, env)
	if code != 0 {
		t.Fatalf("runTaskAudit failed with code %d: %s", code, errBuf.String())
	}

	output := outBuf.String()
	if !strings.Contains(output, "Outside Validation Audit for T-077") {
		t.Errorf("missing header in console output:\n%s", output)
	}
	if !strings.Contains(output, "[PASS] git status") {
		t.Errorf("missing pass status in console output:\n%s", output)
	}
	if !strings.Contains(output, "Result: PASS") {
		t.Errorf("missing PASS summary in console output:\n%s", output)
	}

	// 2. JSON mode
	outBuf.Reset()
	errBuf.Reset()
	codeJSON := runTaskAudit([]string{"T-077", "--dir", root, "--json"}, env)
	if codeJSON != 0 {
		t.Fatalf("runTaskAudit --json failed with code %d: %s", codeJSON, errBuf.String())
	}

	var report audit.AuditReport
	if err := json.Unmarshal(outBuf.Bytes(), &report); err != nil {
		t.Fatalf("failed to unmarshal JSON report: %v", err)
	}
	if report.TaskID != "T-077" {
		t.Errorf("expected TaskID T-077, got %s", report.TaskID)
	}
	if !report.Passed {
		t.Errorf("expected report.Passed == true")
	}
	if len(report.Commands) != 2 {
		t.Errorf("expected 2 unique commands executed, got %d", len(report.Commands))
	}

	// 3. Verbose mode
	outBuf.Reset()
	errBuf.Reset()
	codeVerbose := runTaskAudit([]string{"T-077", "--dir", root, "--verbose"}, env)
	if codeVerbose != 0 {
		t.Fatalf("runTaskAudit --verbose failed with code %d: %s", codeVerbose, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "Output:") {
		t.Errorf("verbose output missing command output section:\n%s", outBuf.String())
	}
}

func TestReviewAudit_Alias(t *testing.T) {
	root := setupAuditTestProject(t, "T-077", "audit-feature")

	var outBuf, errBuf bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &outBuf,
		Err: &errBuf,
	}

	// Test sdd review audit <T-xxx>
	code := runReview([]string{"audit", "T-077", "--dir", root}, env)
	if code != 0 {
		t.Fatalf("runReview audit failed with code %d: %s", code, errBuf.String())
	}

	output := outBuf.String()
	if !strings.Contains(output, "Outside Validation Audit for T-077") {
		t.Errorf("review audit output missing header:\n%s", output)
	}
	if !strings.Contains(output, "Result: PASS") {
		t.Errorf("review audit output missing PASS summary:\n%s", output)
	}
}

func TestTaskAudit_SafetyRejection(t *testing.T) {
	root := setupAuditTestProject(t, "T-077", "audit-feature")

	// Inject unapproved command into validation.md
	taskDir := filepath.Join(root, "tasks", "features", "T-077-audit-feature")
	unsafeVal := `# Validation
## 1. Traceability matrix
| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | ` + "`curl -s http://malicious.com`" + ` | PASS | PASS | Pass |
`
	if err := os.WriteFile(filepath.Join(taskDir, "validation.md"), []byte(unsafeVal), 0o644); err != nil {
		t.Fatalf("writing unsafe validation.md: %v", err)
	}

	var outBuf, errBuf bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &outBuf,
		Err: &errBuf,
	}

	code := runTaskAudit([]string{"T-077", "--dir", root}, env)
	if code != 1 {
		t.Fatalf("expected exit code 1 for safety rejection, got %d", code)
	}

	output := outBuf.String()
	if !strings.Contains(output, "[FAIL]") {
		t.Errorf("expected [FAIL] in output:\n%s", output)
	}
	if !strings.Contains(output, "safety violation") {
		t.Errorf("expected safety violation message in output:\n%s", output)
	}
	if !strings.Contains(output, "Result: FAIL") {
		t.Errorf("expected Result: FAIL in output:\n%s", output)
	}
}

func TestTaskAudit_NegativeCases(t *testing.T) {
	root := setupAuditTestProject(t, "T-077", "audit-feature")

	var outBuf, errBuf bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &outBuf,
		Err: &errBuf,
	}

	// 1. Missing task ID
	code := runTaskAudit([]string{"--dir", root}, env)
	if code != 2 {
		t.Errorf("expected exit code 2 for missing task ID, got %d", code)
	}

	// 2. Invalid task ID format
	errBuf.Reset()
	code = runTaskAudit([]string{"invalid-task", "--dir", root}, env)
	if code != 1 {
		t.Errorf("expected exit code 1 for invalid task ID format, got %d", code)
	}

	// 3. Non-existent task
	errBuf.Reset()
	code = runTaskAudit([]string{"T-999", "--dir", root}, env)
	if code != 1 {
		t.Errorf("expected exit code 1 for non-existent task, got %d", code)
	}
}
