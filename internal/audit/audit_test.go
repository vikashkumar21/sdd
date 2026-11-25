package audit

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupTestProject(t *testing.T, taskID, slug string) string {
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
| R-002 | test | ` + "`git status && git diff`" + ` | clean | clean | Pass |

## 2. Commands run
| Command | Result |
|---|---|
| ` + "`git status`" + ` | clean |
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

	// Create dummy git repo in tempDir
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	_ = cmd.Run()

	return tempDir
}

func TestEnsureGitignoreReview(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Initial write
	if err := EnsureGitignoreReview(tempDir); err != nil {
		t.Fatalf("EnsureGitignoreReview failed: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(tempDir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	if !strings.Contains(string(content), ".sdd/review/") {
		t.Fatalf("expected .gitignore to contain .sdd/review/, got:\n%s", string(content))
	}

	// 2. Second call should be idempotent
	if err := EnsureGitignoreReview(tempDir); err != nil {
		t.Fatalf("second EnsureGitignoreReview failed: %v", err)
	}
	content2, err := os.ReadFile(filepath.Join(tempDir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore 2nd time: %v", err)
	}
	if strings.Count(string(content2), ".sdd/review/") != 1 {
		t.Fatalf("expected exactly one occurrence of .sdd/review/, got %d", strings.Count(string(content2), ".sdd/review/"))
	}
}

func TestStagePlanBundle(t *testing.T) {
	root := setupTestProject(t, "T-099", "sample-task")

	fixedTime := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
	bundle, err := StagePlanBundle(root, "T-099", fixedTime)
	if err != nil {
		t.Fatalf("StagePlanBundle failed: %v", err)
	}

	if bundle.TaskID != "T-099" {
		t.Errorf("expected TaskID T-099, got %s", bundle.TaskID)
	}
	if !bundle.CreatedAt.Equal(fixedTime) {
		t.Errorf("expected CreatedAt %v, got %v", fixedTime, bundle.CreatedAt)
	}
	expectedBundleDir := ".sdd/review/T-099/plan"
	if bundle.BundleDir != expectedBundleDir {
		t.Errorf("expected BundleDir %s, got %s", expectedBundleDir, bundle.BundleDir)
	}

	// Verify staged files exist
	specPath := filepath.Join(root, expectedBundleDir, "spec.md")
	if _, err := os.Stat(specPath); err != nil {
		t.Errorf("spec.md missing: %v", err)
	}
	specData, _ := os.ReadFile(specPath)
	if !strings.Contains(string(specData), "Test goal for T-099") {
		t.Errorf("spec.md missing requirement content")
	}
	if !strings.Contains(string(specData), "internal/test.go") {
		t.Errorf("spec.md missing plan content")
	}
	if !strings.Contains(string(specData), "Validation Strategy (validation.md)") {
		t.Errorf("spec.md missing validation strategy header")
	}
	if !strings.Contains(string(specData), "Traceability matrix") {
		t.Errorf("spec.md missing validation content")
	}

	rubricPath := filepath.Join(root, expectedBundleDir, "rubric.md")
	if _, err := os.Stat(rubricPath); err != nil {
		t.Errorf("rubric.md missing: %v", err)
	}

	manifestPath := filepath.Join(root, expectedBundleDir, "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Errorf("manifest.json missing: %v", err)
	}
	manifestData, _ := os.ReadFile(manifestPath)
	var parsed PlanBundle
	if err := json.Unmarshal(manifestData, &parsed); err != nil {
		t.Errorf("manifest unmarshal: %v", err)
	}
	if parsed.TaskID != "T-099" {
		t.Errorf("manifest TaskID mismatch: %s", parsed.TaskID)
	}

	// Test prompt generation
	prompt, err := FormatPlanReviewPrompt(bundle, root)
	if err != nil {
		t.Fatalf("FormatPlanReviewPrompt failed: %v", err)
	}
	if !strings.Contains(prompt, "Stateless Pre-GATE 1 Plan Review for T-099") {
		t.Errorf("unexpected prompt header:\n%s", prompt)
	}
	if !strings.Contains(prompt, ".sdd/review/T-099/plan/spec.md") {
		t.Errorf("prompt missing bundle references:\n%s", prompt)
	}

	// Test invalid task ID
	if _, err := StagePlanBundle(root, "invalid-id"); err == nil {
		t.Errorf("expected error for invalid task ID")
	}

	// Test non-existent task
	if _, err := StagePlanBundle(root, "T-999"); err == nil {
		t.Errorf("expected error for non-existent task")
	}
}

func TestStageReviewBundle(t *testing.T) {
	root := setupTestProject(t, "T-099", "sample-task")

	// Create an untracked file to test untracked diff synthesis
	untrackedPath := filepath.Join(root, "new_feature.go")
	if err := os.WriteFile(untrackedPath, []byte("package main\n\nfunc Hello() string { return \"hello\" }\n"), 0o644); err != nil {
		t.Fatalf("writing untracked file: %v", err)
	}

	// Create an untracked symlink pointing outside workspace (SEC-001)
	symlinkTarget := filepath.Join(t.TempDir(), "outside.txt")
	_ = os.WriteFile(symlinkTarget, []byte("secret outside data"), 0o644)
	_ = os.Symlink(symlinkTarget, filepath.Join(root, "symlink_escape.txt"))

	// Create an untracked binary file (SEC-001)
	binaryPath := filepath.Join(root, "data.bin")
	_ = os.WriteFile(binaryPath, []byte{0x7f, 0x45, 0x4c, 0x46, 0x00, 0x01}, 0o644)

	fixedTime := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
	lenses := []string{"spec", "security", "pattern", "all"}
	for _, lens := range lenses {
		bundle, err := StageReviewBundle(root, "T-099", lens, fixedTime)
		if err != nil {
			t.Fatalf("StageReviewBundle(%s) failed: %v", lens, err)
		}

		if bundle.TaskID != "T-099" {
			t.Errorf("expected TaskID T-099, got %s", bundle.TaskID)
		}
		if bundle.Lens != lens {
			t.Errorf("expected Lens %s, got %s", lens, bundle.Lens)
		}
		if !bundle.CreatedAt.Equal(fixedTime) {
			t.Errorf("expected CreatedAt %v, got %v", fixedTime, bundle.CreatedAt)
		}
		expectedBundleDir := filepath.ToSlash(filepath.Join(".sdd", "review", "T-099", lens))
		if lens == "all" {
			expectedBundleDir = ".sdd/review/T-099/code"
		}
		if bundle.BundleDir != expectedBundleDir {
			t.Errorf("expected BundleDir %s, got %s", expectedBundleDir, bundle.BundleDir)
		}

		// Verify diff.patch
		diffPath := filepath.Join(root, expectedBundleDir, "diff.patch")
		diffData, err := os.ReadFile(diffPath)
		if err != nil {
			t.Errorf("reading diff.patch: %v", err)
		}
		diffStr := string(diffData)
		if !strings.Contains(diffStr, "new_feature.go") {
			t.Errorf("diff.patch did not capture untracked file new_feature.go:\n%s", diffStr)
		}
		// Verify symlink was skipped (SEC-001)
		if strings.Contains(diffStr, "symlink_escape.txt") {
			t.Errorf("diff.patch must NOT contain escaped symlink (SEC-001):\n%s", diffStr)
		}
		// Verify binary file was formatted as binary diff (SEC-001)
		if !strings.Contains(diffStr, "Binary files /dev/null and b/data.bin differ") {
			t.Errorf("diff.patch missing binary diff notation (SEC-001):\n%s", diffStr)
		}

		// Verify prompt
		prompt, err := FormatReviewPrompt(bundle, root)
		if err != nil {
			t.Fatalf("FormatReviewPrompt(%s) failed: %v", lens, err)
		}
		if !strings.Contains(prompt, "Stateless Subagent Code Review for T-099") {
			t.Errorf("unexpected prompt header:\n%s", prompt)
		}
		if !strings.Contains(prompt, expectedBundleDir+"/diff.patch") {
			t.Errorf("prompt missing diff.patch reference:\n%s", prompt)
		}
	}
}

func TestExtractValidationCommands(t *testing.T) {
	valContent := `
## 1. Traceability matrix
| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | ` + "`go test -v ./internal/audit`" + ` | PASS | PASS | Pass |
| R-002 | test | ` + "`make test && make vet`" + ` | clean | clean | Pass |
| R-003 | test | ` + "`./bin/sdd check --strict`" + ` | 0 errors | 0 errors | Pass |

## 2. Commands run
| Command | Result |
|---|---|
| ` + "`go test -v ./internal/audit`" + ` | PASS |
| ` + "`sdd status`" + ` | ok |
`

	cmds := ExtractValidationCommands(valContent)
	expected := []string{
		"go test -v ./internal/audit",
		"make test",
		"make vet",
		"./bin/sdd check --strict",
		"sdd status",
	}

	if len(cmds) != len(expected) {
		t.Fatalf("expected %d commands, got %d: %v", len(expected), len(cmds), cmds)
	}

	for i, exp := range expected {
		if cmds[i] != exp {
			t.Errorf("at index %d: expected %q, got %q", i, exp, cmds[i])
		}
	}
}

func TestIsCommandSafe(t *testing.T) {
	validCommands := []string{
		"go test ./...",
		"go test -v -race ./internal/audit",
		"go vet ./...",
		"make check",
		"make build",
		"sdd check --strict",
		"./bin/sdd check",
		"bin/sdd check",
		"sdd status",
		"git grep TODO",
		"git diff HEAD",
		"git status",
	}

	for _, cmd := range validCommands {
		safe, reason := IsCommandSafe(cmd)
		if !safe {
			t.Errorf("expected command %q to be safe, rejected with: %s", cmd, reason)
		}
	}

	unsafeCommands := []string{
		"rm -rf /",
		"curl http://example.com",
		"wget http://example.com",
		"go test ./...; rm -rf /",
		"go test ./... | grep ok",
		"go test ./... &",
		"go test `cat file`",
		"go test -exec ./evil.sh",
		"git diff --output=/etc/passwd",
		"make -f /tmp/evil.mk",
		"make -C /",
		"make check > output.txt",
		"make check < input.txt",
		"echo $(cat /etc/passwd)",
		"",
	}

	for _, cmd := range unsafeCommands {
		safe, reason := IsCommandSafe(cmd)
		if safe {
			t.Errorf("expected command %q to be rejected, but it passed", cmd)
		} else if reason == "" {
			t.Errorf("expected rejection reason for %q", cmd)
		}
	}
}

func TestRunAudit(t *testing.T) {
	root := setupTestProject(t, "T-099", "sample-task")

	// 1. Successful audit run
	opts := AuditOpts{
		Verbose: true,
		Timeout: 30 * time.Second,
	}
	report, err := RunAudit(context.Background(), root, "T-099", opts)
	if err != nil {
		t.Fatalf("RunAudit failed: %v", err)
	}

	if !report.Passed {
		t.Errorf("expected audit to pass, failed with commands: %+v", report.Commands)
	}
	if len(report.Commands) == 0 {
		t.Errorf("expected at least one command executed, got 0")
	}

	// 2. Audit with unsafe command injected
	taskDir := filepath.Join(root, "tasks", "features", "T-099-sample-task")
	unsafeVal := `# Validation
## 1. Traceability matrix
| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | ` + "`rm -rf /tmp/test`" + ` | PASS | PASS | Pass |
`
	if err := os.WriteFile(filepath.Join(taskDir, "validation.md"), []byte(unsafeVal), 0o644); err != nil {
		t.Fatalf("writing unsafe validation.md: %v", err)
	}

	unsafeReport, err := RunAudit(context.Background(), root, "T-099", opts)
	if err != nil {
		t.Fatalf("RunAudit unexpected error: %v", err)
	}
	if unsafeReport.Passed {
		t.Errorf("expected audit to fail on unsafe command, but it passed")
	}
	if len(unsafeReport.Commands) != 1 || unsafeReport.Commands[0].ExitCode != 126 {
		t.Errorf("expected command audit with exit code 126, got %+v", unsafeReport.Commands)
	}

	// 3. Audit on non-existent task
	if _, err := RunAudit(context.Background(), root, "T-999", opts); err == nil {
		t.Errorf("expected error on non-existent task")
	}
}
