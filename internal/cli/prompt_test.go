package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
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

		expectedSubdir := lens
		if lens == "all" {
			expectedSubdir = "code"
		}
		expectedBundleDirRel := fmt.Sprintf(".sdd/review/T-088/%s", expectedSubdir)

		promptOutput := outBuf.String()
		if !strings.Contains(promptOutput, "Stateless Review Bundle (.sdd/review/)") {
			t.Errorf("prompt output missing stateless bundle section for lens %s:\n%s", lens, promptOutput)
		}
		if !strings.Contains(promptOutput, expectedBundleDirRel+"/diff.patch") {
			t.Errorf("prompt output missing bundle diff.patch reference for lens %s:\n%s", lens, promptOutput)
		}
		if !strings.Contains(promptOutput, expectedBundleDirRel+"/ledger.md") {
			t.Errorf("prompt output missing bundle ledger.md reference for lens %s:\n%s", lens, promptOutput)
		}
		if !strings.Contains(promptOutput, "Unanchored Traceability Scan") {
			t.Errorf("prompt output missing Unanchored Traceability Scan for lens %s:\n%s", lens, promptOutput)
		}

		// Verify bundle was staged on disk
		bundleDir := filepath.Join(root, ".sdd", "review", "T-088", expectedSubdir)
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
		if ctx.BundleDir != expectedBundleDirRel {
			t.Errorf("expected BundleDir %s, got %s", expectedBundleDirRel, ctx.BundleDir)
		}
		if len(ctx.BundleFiles) == 0 {
			t.Errorf("expected non-empty BundleFiles for lens %s", lens)
		}
		if ctx.RawDiffBytes < 0 || ctx.FilteredDiffBytes < 0 || ctx.DiffSavingsPercent < 0.0 {
			t.Errorf("invalid telemetry in ctx: raw=%d, filtered=%d, savings=%f", ctx.RawDiffBytes, ctx.FilteredDiffBytes, ctx.DiffSavingsPercent)
		}
	}
}

func TestPromptReview_Loop(t *testing.T) {
	root := setupPromptTestProject(t, "T-088", "review-test")

	// Add sample findings to review.md
	reviewPath := filepath.Join(root, "tasks", "features", "T-088-review-test", "review.md")
	reviewContent := `# Review
## 3. Risk & Findings Register
| ID | Category | Severity | Finding | Suggested Action | Status |
|---|---|---|---|---|---|
| F-001 | Spec | 🔴 Blocker | Missing error check | Handle error | Resolved |
`
	if err := os.WriteFile(reviewPath, []byte(reviewContent), 0644); err != nil {
		t.Fatalf("writing review.md: %v", err)
	}

	// 1. Invalid loop bounds
	var outBuf, errBuf bytes.Buffer
	env := IO{In: strings.NewReader(""), Out: &outBuf, Err: &errBuf}

	if code := runPromptReview([]string{"T-088", "--loop", "0", "--dir", root}, env); code != 2 {
		t.Errorf("expected code 2 for --loop 0, got %d", code)
	}
	errBuf.Reset()
	if code := runPromptReview([]string{"T-088", "--loop", "4", "--dir", root}, env); code != 2 {
		t.Errorf("expected code 2 for --loop 4, got %d", code)
	}

	// 2. Loop 2 text output
	outBuf.Reset()
	errBuf.Reset()
	if code := runPromptReview([]string{"T-088", "--loop", "2", "--lens", "spec", "--dir", root}, env); code != 0 {
		t.Fatalf("runPromptReview --loop 2 failed: %d: %s", code, errBuf.String())
	}
	outText := outBuf.String()
	if !strings.Contains(outText, "Remediation Review (Loop 2)") {
		t.Errorf("expected Remediation Review (Loop 2) in prompt output:\n%s", outText)
	}
	if !strings.Contains(outText, "findings_to_verify.md") {
		t.Errorf("expected findings_to_verify.md in prompt output:\n%s", outText)
	}
	if !strings.Contains(outText, "Defect Remediation Verification") {
		t.Errorf("expected Defect Remediation Verification step in prompt output:\n%s", outText)
	}

	// 3. Loop 2 JSON output
	outBuf.Reset()
	errBuf.Reset()
	if code := runPromptReview([]string{"T-088", "--loop", "2", "--lens", "spec", "--dir", root, "--json"}, env); code != 0 {
		t.Fatalf("runPromptReview --loop 2 --json failed: %d: %s", code, errBuf.String())
	}
	var ctx ReviewPromptContext
	if err := json.Unmarshal(outBuf.Bytes(), &ctx); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if ctx.Loop != 2 {
		t.Errorf("expected ctx.Loop == 2, got %d", ctx.Loop)
	}
}

func TestPromptReview_ModelRouting(t *testing.T) {
	root := setupPromptTestProject(t, "T-059", "routing-test")

	// 1. Text output for lens=spec
	var outBuf, errBuf bytes.Buffer
	env := IO{In: strings.NewReader(""), Out: &outBuf, Err: &errBuf}
	if code := runPromptReview([]string{"T-059", "--lens", "spec", "--dir", root}, env); code != 0 {
		t.Fatalf("runPromptReview failed: %d: %s", code, errBuf.String())
	}
	outText := outBuf.String()
	if !strings.Contains(outText, "## Model Routing Guidance") {
		t.Errorf("expected '## Model Routing Guidance' in prompt output:\n%s", outText)
	}
	if !strings.Contains(outText, "- **Role:** spec-auditor") || !strings.Contains(outText, "- **Model Tier:** standard") {
		t.Errorf("expected spec-auditor standard tier in prompt output:\n%s", outText)
	}
	if !strings.Contains(outText, `Model="flash"`) || !strings.Contains(outText, `Role="Independent Spec Auditor"`) {
		t.Errorf("expected invoke_subagent snippet with Model=\"flash\" for spec-auditor:\n%s", outText)
	}

	// 2. JSON output for lens=spec
	outBuf.Reset()
	errBuf.Reset()
	if code := runPromptReview([]string{"T-059", "--lens", "spec", "--dir", root, "--json"}, env); code != 0 {
		t.Fatalf("runPromptReview --json failed: %d: %s", code, errBuf.String())
	}
	var ctx ReviewPromptContext
	if err := json.Unmarshal(outBuf.Bytes(), &ctx); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if ctx.ModelRouting == nil {
		t.Fatal("expected non-nil ctx.ModelRouting")
	}
	if ctx.ModelRouting.Tier != "standard" || ctx.ModelRouting.ReasoningEffort != "medium" {
		t.Errorf("unexpected model_routing: %+v", ctx.ModelRouting)
	}

	// 2b. Text output for lens=security
	outBuf.Reset()
	errBuf.Reset()
	if code := runPromptReview([]string{"T-059", "--lens", "security", "--dir", root}, env); code != 0 {
		t.Fatalf("runPromptReview security failed: %d: %s", code, errBuf.String())
	}
	secText := outBuf.String()
	if !strings.Contains(secText, "- **Role:** appsec-specialist") || !strings.Contains(secText, `Model="inherit"`) {
		t.Errorf("expected appsec-specialist frontier tier with Model=\"inherit\":\n%s", secText)
	}

	// 2c. Text output for lens=pattern
	outBuf.Reset()
	errBuf.Reset()
	if code := runPromptReview([]string{"T-059", "--lens", "pattern", "--dir", root}, env); code != 0 {
		t.Fatalf("runPromptReview pattern failed: %d: %s", code, errBuf.String())
	}
	patText := outBuf.String()
	if !strings.Contains(patText, "- **Role:** pattern-auditor") || !strings.Contains(patText, `Model="flash"`) {
		t.Errorf("expected pattern-auditor standard tier with Model=\"flash\":\n%s", patText)
	}

	// 3. Text output for lens=all (triad routing)
	outBuf.Reset()
	errBuf.Reset()
	if code := runPromptReview([]string{"T-059", "--lens", "all", "--dir", root}, env); code != 0 {
		t.Fatalf("runPromptReview all failed: %d: %s", code, errBuf.String())
	}
	allText := outBuf.String()
	if !strings.Contains(allText, "## Model Routing Guidance") {
		t.Errorf("expected '## Model Routing Guidance' in all prompt:\n%s", allText)
	}
	if !strings.Contains(allText, "Spec Auditor (spec-auditor)") || !strings.Contains(allText, "AppSec Specialist (appsec-specialist)") {
		t.Errorf("expected triad roles in guidance:\n%s", allText)
	}
	if !strings.Contains(allText, `Model="flash"`) || !strings.Contains(allText, `Model="inherit"`) {
		t.Errorf("expected triad subagent invocation snippets with flash and inherit:\n%s", allText)
	}

	// 4. JSON output for lens=all
	outBuf.Reset()
	errBuf.Reset()
	if code := runPromptReview([]string{"T-059", "--lens", "all", "--dir", root, "--json"}, env); code != 0 {
		t.Fatalf("runPromptReview all --json failed: %d: %s", code, errBuf.String())
	}
	var allCtx ReviewPromptContext
	if err := json.Unmarshal(outBuf.Bytes(), &allCtx); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if allCtx.ModelRouting == nil {
		t.Fatal("expected non-nil allCtx.ModelRouting")
	}
	if allCtx.TriadRouting == nil || len(allCtx.TriadRouting) != 3 {
		t.Errorf("expected 3 triad routing entries, got: %+v", allCtx.TriadRouting)
	}
}

func TestPromptPlanReview_ModelRouting(t *testing.T) {
	root := setupPromptTestProject(t, "T-059", "routing-test")

	var outBuf, errBuf bytes.Buffer
	env := IO{In: strings.NewReader(""), Out: &outBuf, Err: &errBuf}
	if code := runPromptPlanReview([]string{"T-059", "--dir", root}, env); code != 0 {
		t.Fatalf("runPromptPlanReview failed: %d: %s", code, errBuf.String())
	}
	outText := outBuf.String()
	if !strings.Contains(outText, "## Model Routing Guidance") {
		t.Errorf("expected '## Model Routing Guidance' in plan review prompt:\n%s", outText)
	}
	if !strings.Contains(outText, "- **Role:** plan-reviewer") || !strings.Contains(outText, "- **Model Tier:** advanced") {
		t.Errorf("expected plan-reviewer advanced tier in prompt output:\n%s", outText)
	}
	if !strings.Contains(outText, `Model="inherit"`) || !strings.Contains(outText, `Role="Independent Plan Reviewer"`) {
		t.Errorf("expected plan-reviewer invoke_subagent snippet with Model=\"inherit\":\n%s", outText)
	}

	// JSON output
	outBuf.Reset()
	errBuf.Reset()
	if code := runPromptPlanReview([]string{"T-059", "--dir", root, "--json"}, env); code != 0 {
		t.Fatalf("runPromptPlanReview --json failed: %d: %s", code, errBuf.String())
	}
	var ctx PlanReviewPromptContext
	if err := json.Unmarshal(outBuf.Bytes(), &ctx); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if ctx.ModelRouting == nil {
		t.Fatal("expected non-nil ctx.ModelRouting")
	}
	if ctx.ModelRouting.Tier != "advanced" || ctx.ModelRouting.ReasoningEffort != "high" {
		t.Errorf("unexpected plan review model routing: %+v", ctx.ModelRouting)
	}
}

func TestPromptImplement_ModelRouting(t *testing.T) {
	root := setupPromptTestProject(t, "T-059", "routing-test")

	var outBuf, errBuf bytes.Buffer
	env := IO{In: strings.NewReader(""), Out: &outBuf, Err: &errBuf}
	if code := runPromptImplement([]string{"T-059", "--dir", root}, env); code != 0 {
		t.Fatalf("runPromptImplement failed: %d: %s", code, errBuf.String())
	}
	outText := outBuf.String()
	if !strings.Contains(outText, "## Model Routing Guidance") {
		t.Errorf("expected '## Model Routing Guidance' in implement prompt:\n%s", outText)
	}
	if !strings.Contains(outText, "- **Role:** implementer") || !strings.Contains(outText, "- **Model Tier:** advanced") {
		t.Errorf("expected implementer advanced tier in prompt output:\n%s", outText)
	}
	if !strings.Contains(outText, `Model="inherit"`) || !strings.Contains(outText, `Role="Senior Software Engineer (Implementer)"`) {
		t.Errorf("expected implementer invoke_subagent snippet with Model=\"inherit\":\n%s", outText)
	}

	// 1b. Text output for role=qa (standard tier)
	var qaOutBuf, qaErrBuf bytes.Buffer
	qaEnv := IO{In: strings.NewReader(""), Out: &qaOutBuf, Err: &qaErrBuf}
	if code := runPromptImplement([]string{"T-059", "--role", "qa", "--dir", root}, qaEnv); code != 0 {
		t.Fatalf("runPromptImplement --role qa failed: %d: %s", code, qaErrBuf.String())
	}
	qaText := qaOutBuf.String()
	if !strings.Contains(qaText, `Model="flash"`) || !strings.Contains(qaText, `Role="Senior QA Automation Engineer (QA)"`) {
		t.Errorf("expected qa invoke_subagent snippet with Model=\"flash\":\n%s", qaText)
	}

	// 1c. Text output for role=devops (standard tier)
	var devopsOutBuf, devopsErrBuf bytes.Buffer
	devopsEnv := IO{In: strings.NewReader(""), Out: &devopsOutBuf, Err: &devopsErrBuf}
	if code := runPromptImplement([]string{"T-059", "--role", "devops", "--dir", root}, devopsEnv); code != 0 {
		t.Fatalf("runPromptImplement --role devops failed: %d: %s", code, devopsErrBuf.String())
	}
	devopsText := devopsOutBuf.String()
	if !strings.Contains(devopsText, `Model="flash"`) || !strings.Contains(devopsText, `Role="Senior DevOps / SRE Engineer (DevOps)"`) {
		t.Errorf("expected devops invoke_subagent snippet with Model=\"flash\":\n%s", devopsText)
	}

	// JSON output
	outBuf.Reset()
	errBuf.Reset()
	if code := runPromptImplement([]string{"T-059", "--dir", root, "--json"}, env); code != 0 {
		t.Fatalf("runPromptImplement --json failed: %d: %s", code, errBuf.String())
	}
	var ctx ImplementPromptContext
	if err := json.Unmarshal(outBuf.Bytes(), &ctx); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if ctx.ModelRouting == nil {
		t.Fatal("expected non-nil ctx.ModelRouting")
	}
	if ctx.ModelRouting.Tier != "advanced" || ctx.ModelRouting.ReasoningEffort != "medium" {
		t.Errorf("unexpected implement model routing: %+v", ctx.ModelRouting)
	}
}

func TestResolveSubagentModelParam(t *testing.T) {
	tests := []struct {
		tier     string
		expected string
	}{
		{"standard", "flash"},
		{"fast", "flash_lite"},
		{"frontier", "inherit"},
		{"advanced", "inherit"},
		{"  STANDARD  ", "flash"},
		{"  FAST  ", "flash_lite"},
		{"  FRONTIER  ", "inherit"},
		{"  ADVANCED  ", "inherit"},
		{"", "inherit"},
		{"   ", "inherit"},
		{"unknown-tier", "inherit"},
		{"custom", "inherit"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run("tier_"+tc.tier, func(t *testing.T) {
			got := resolveSubagentModelParam(tc.tier)
			if got != tc.expected {
				t.Errorf("resolveSubagentModelParam(%q) = %q, want %q", tc.tier, got, tc.expected)
			}
		})
	}
}

func TestRoleTitleFor(t *testing.T) {
	tests := []struct {
		roleName string
		expected string
	}{
		{"spec-auditor", "Independent Spec Auditor"},
		{"appsec-specialist", "Independent AppSec Specialist"},
		{"pattern-auditor", "Independent Pattern & Quality Auditor"},
		{"plan-reviewer", "Independent Plan Reviewer"},
		{"implementer", "Senior Software Engineer (Implementer)"},
		{"qa", "Senior QA Automation Engineer (QA)"},
		{"devops", "Senior DevOps / SRE Engineer (DevOps)"},
		{"codebase-analyst", "Codebase Discovery Analyst"},
		{"discovery-challenger", "Adversarial Discovery Challenger"},
		{"", "Subagent"},
		{"   ", "Subagent"},
		{"custom-specialist", "custom-specialist"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run("role_"+tc.roleName, func(t *testing.T) {
			got := roleTitleFor(tc.roleName)
			if got != tc.expected {
				t.Errorf("roleTitleFor(%q) = %q, want %q", tc.roleName, got, tc.expected)
			}
		})
	}
}
