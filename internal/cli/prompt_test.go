package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupPromptTestProject(t *testing.T, taskID, slug string) string {
	t.Helper()
	tempDir := t.TempDir()

	taskDir := filepath.Join(tempDir, "tasks", "features", taskID+"-"+slug)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("failed to create task dir: %v", err)
	}

	reqContent := `# Requirements — ` + taskID + `
Status: Approved
## 1. Goal
Test goal for ` + taskID + `
## 4. Functional requirements
| ID | Requirement | Priority | Verification |
|---|---|---|---|
| R-001 | System SHALL pass | MUST | test |
## 7. Open questions
(None)
`
	planContent := `# Plan — ` + taskID + `
## 4. Implementation steps
| # | Step | Files | Proving test |
|---|---|---|---|
| 1 | Test step | internal/test.go | go test |
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
`
	revContent := `# Code Review — ` + taskID + `
Status: Pending
`

	if err := os.WriteFile(filepath.Join(taskDir, "requirements.md"), []byte(reqContent), 0o644); err != nil {
		t.Fatalf("writing requirements.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "plan.md"), []byte(planContent), 0o644); err != nil {
		t.Fatalf("writing plan.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "validation.md"), []byte(valContent), 0o644); err != nil {
		t.Fatalf("writing validation.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "review.md"), []byte(revContent), 0o644); err != nil {
		t.Fatalf("writing review.md: %v", err)
	}

	// Create dummy TECH_STACK.md and SKILLS.md
	techContent := `# TECH_STACK
## 6. Decision register
| ID | Decision | Status |
|---|---|---|
| D-001 | Test decision | Approved 2026-01-01 |
`
	skillsContent := `# SKILLS
## 8. Human feedback rules
### Register
| ID | Date | Rule | Applies to | Source | Verified by | Status |
|---|---|---|---|---|---|---|
| F-001 | 2026-11-10 | Worker agents must resolve all findings | All tasks | Review | Check | Active |
`
	_ = os.WriteFile(filepath.Join(tempDir, "TECH_STACK.md"), []byte(techContent), 0o644)
	_ = os.WriteFile(filepath.Join(tempDir, "SKILLS.md"), []byte(skillsContent), 0o644)

	return tempDir
}

func TestPromptPlanReview_StatelessBundle(t *testing.T) {
	root := setupPromptTestProject(t, "T-088", "plan-test")

	// 1. Text mode
	var outBuf, errBuf bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &outBuf,
		Err: &errBuf,
	}

	code := runPromptPlanReview([]string{"T-088", "--dir", root}, env)
	if code != 0 {
		t.Fatalf("runPromptPlanReview failed with code %d: %s", code, errBuf.String())
	}

	promptOutput := outBuf.String()
	if !strings.Contains(promptOutput, "Stateless Review Bundle (.sdd/review/)") {
		t.Errorf("prompt output missing stateless bundle section:\n%s", promptOutput)
	}
	if !strings.Contains(promptOutput, ".sdd/review/T-088/plan/spec.md") {
		t.Errorf("prompt output missing bundle spec.md reference:\n%s", promptOutput)
	}

	// Verify bundle was staged on disk
	bundleDir := filepath.Join(root, ".sdd", "review", "T-088", "plan")
	if _, err := os.Stat(filepath.Join(bundleDir, "spec.md")); err != nil {
		t.Errorf("expected staged spec.md in %s", bundleDir)
	}
	if _, err := os.Stat(filepath.Join(bundleDir, "rubric.md")); err != nil {
		t.Errorf("expected staged rubric.md in %s", bundleDir)
	}
	if _, err := os.Stat(filepath.Join(bundleDir, "manifest.json")); err != nil {
		t.Errorf("expected staged manifest.json in %s", bundleDir)
	}

	// 2. JSON mode
	outBuf.Reset()
	errBuf.Reset()
	codeJSON := runPromptPlanReview([]string{"T-088", "--dir", root, "--json"}, env)
	if codeJSON != 0 {
		t.Fatalf("runPromptPlanReview --json failed with code %d: %s", codeJSON, errBuf.String())
	}

	var ctx PlanReviewPromptContext
	if err := json.Unmarshal(outBuf.Bytes(), &ctx); err != nil {
		t.Fatalf("failed to unmarshal JSON output: %v", err)
	}
	if ctx.TaskID != "T-088" {
		t.Errorf("expected TaskID T-088, got %s", ctx.TaskID)
	}
	if ctx.BundleDir != ".sdd/review/T-088/plan" {
		t.Errorf("expected BundleDir .sdd/review/T-088/plan, got %s", ctx.BundleDir)
	}
	if len(ctx.BundleFiles) == 0 {
		t.Errorf("expected non-empty BundleFiles")
	}
}

func TestPromptReview_StatelessBundle(t *testing.T) {
	root := setupPromptTestProject(t, "T-088", "review-test")

	lenses := []string{"spec", "security", "pattern", "all"}
	for _, lens := range lenses {
		var outBuf, errBuf bytes.Buffer
		env := IO{
			In:  strings.NewReader(""),
			Out: &outBuf,
			Err: &errBuf,
		}

		code := runPromptReview([]string{"T-088", "--lens", lens, "--dir", root}, env)
		if code != 0 {
			t.Fatalf("runPromptReview(%s) failed with code %d: %s", lens, code, errBuf.String())
		}

		promptOutput := outBuf.String()
		if !strings.Contains(promptOutput, "Stateless Review Bundle (.sdd/review/)") {
			t.Errorf("prompt output missing stateless bundle section for lens %s:\n%s", lens, promptOutput)
		}
		if !strings.Contains(promptOutput, ".sdd/review/T-088/code/diff.patch") {
			t.Errorf("prompt output missing bundle diff.patch reference for lens %s:\n%s", lens, promptOutput)
		}
		if !strings.Contains(promptOutput, ".sdd/review/T-088/code/ledger.md") {
			t.Errorf("prompt output missing bundle ledger.md reference for lens %s:\n%s", lens, promptOutput)
		}
		if !strings.Contains(promptOutput, "Unanchored Traceability Scan") {
			t.Errorf("prompt output missing Unanchored Traceability Scan for lens %s:\n%s", lens, promptOutput)
		}

		// Verify bundle was staged on disk
		bundleDir := filepath.Join(root, ".sdd", "review", "T-088", "code")
		if _, err := os.Stat(filepath.Join(bundleDir, "spec.md")); err != nil {
			t.Errorf("expected staged spec.md in %s", bundleDir)
		}
		if _, err := os.Stat(filepath.Join(bundleDir, "ledger.md")); err != nil {
			t.Errorf("expected staged ledger.md in %s", bundleDir)
		}
		if _, err := os.Stat(filepath.Join(bundleDir, "diff.patch")); err != nil {
			t.Errorf("expected staged diff.patch in %s", bundleDir)
		}
		if _, err := os.Stat(filepath.Join(bundleDir, "rubric.md")); err != nil {
			t.Errorf("expected staged rubric.md in %s", bundleDir)
		}
		if _, err := os.Stat(filepath.Join(bundleDir, "manifest.json")); err != nil {
			t.Errorf("expected staged manifest.json in %s", bundleDir)
		}

		// Test JSON mode
		outBuf.Reset()
		errBuf.Reset()
		codeJSON := runPromptReview([]string{"T-088", "--lens", lens, "--dir", root, "--json"}, env)
		if codeJSON != 0 {
			t.Fatalf("runPromptReview(%s) --json failed with code %d: %s", lens, codeJSON, errBuf.String())
		}

		var ctx ReviewPromptContext
		if err := json.Unmarshal(outBuf.Bytes(), &ctx); err != nil {
			t.Fatalf("failed to unmarshal JSON output for lens %s: %v", lens, err)
		}
		if ctx.TaskID != "T-088" {
			t.Errorf("expected TaskID T-088, got %s", ctx.TaskID)
		}
		if ctx.BundleDir != ".sdd/review/T-088/code" {
			t.Errorf("expected BundleDir .sdd/review/T-088/code, got %s", ctx.BundleDir)
		}
		if len(ctx.BundleFiles) == 0 {
			t.Errorf("expected non-empty BundleFiles for lens %s", lens)
		}
	}
}
