package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/role"
)

func setupTestTask(t *testing.T, root, taskID string) {
	t.Helper()
	taskDir := filepath.Join(root, "tasks", "features", taskID+"-test-feature")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatal(err)
	}

	reqContent := "# Requirements\n\n## 1. Goal\nTest\n"
	planContent := "# Plan\n\n## 5. Files to create or modify\n| Path | Action | Reason |\n|---|---|---|\n| internal/test.go | create | test file |\n"
	valContent := "# Validation\n\n## 1. Traceability\n"
	revContent := "# Review\n\n## 1. Traceability\n"

	if err := os.WriteFile(filepath.Join(taskDir, "requirements.md"), []byte(reqContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "plan.md"), []byte(planContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "validation.md"), []byte(valContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "review.md"), []byte(revContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Also write empty TECH_STACK.md and SKILLS.md
	_ = os.WriteFile(filepath.Join(root, "TECH_STACK.md"), []byte("# Tech Stack\n"), 0644)
	_ = os.WriteFile(filepath.Join(root, "SKILLS.md"), []byte("# Skills\n## 8. Human feedback rules\n- **F-001:** rule\n"), 0644)
}

func TestSkillExport_Roles(t *testing.T) {
	tmp := t.TempDir()
	var stdout, stderr bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
		Now: time.Now,
	}

	code := runSkillExport([]string{"--dir", tmp, "--roles"}, env)
	if code != 0 {
		t.Fatalf("runSkillExport failed with code %d: %s", code, stderr.String())
	}

	for _, name := range role.CanonicalRoleNames() {
		skillFile := filepath.Join(tmp, "skills", name, "SKILL.md")
		data, err := os.ReadFile(skillFile)
		if err != nil {
			t.Fatalf("missing exported role skill %s: %v", name, err)
		}
		if !strings.Contains(string(data), "name: "+name) {
			t.Errorf("role skill %s does not contain name in frontmatter", name)
		}
	}
}

func TestSkillExport_SingleRole(t *testing.T) {
	tmp := t.TempDir()
	var stdout, stderr bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
		Now: time.Now,
	}

	code := runSkillExport([]string{"--dir", tmp, "--role", "appsec-specialist"}, env)
	if code != 0 {
		t.Fatalf("runSkillExport failed with code %d: %s", code, stderr.String())
	}

	skillFile := filepath.Join(tmp, "skills", "appsec-specialist", "SKILL.md")
	data, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("missing exported role skill: %v", err)
	}
	if !strings.Contains(string(data), "name: appsec-specialist") {
		t.Errorf("role skill does not contain expected name")
	}

	// Verify other roles were NOT exported
	otherFile := filepath.Join(tmp, "skills", "coordinator", "SKILL.md")
	if _, err := os.Stat(otherFile); err == nil {
		t.Errorf("coordinator skill should not have been exported")
	}
}

func TestSkillExport_InvalidRole(t *testing.T) {
	tmp := t.TempDir()
	var stdout, stderr bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
		Now: time.Now,
	}

	code := runSkillExport([]string{"--dir", tmp, "--role", "non-existent-role"}, env)
	if code != 1 {
		t.Fatalf("expected exit code 1 for invalid role, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown canonical role") {
		t.Errorf("expected unknown canonical role error, got: %s", stderr.String())
	}
}

func TestPromptPlanReview_RoleSkill(t *testing.T) {
	tmp := t.TempDir()
	setupTestTask(t, tmp, "T-040")

	var stdout, stderr bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
		Now: time.Now,
	}

	// 1. Text output
	code := runPromptPlanReview([]string{"--dir", tmp, "T-040"}, env)
	if code != 0 {
		t.Fatalf("runPromptPlanReview failed with code %d: %s", code, stderr.String())
	}
	textOut := stdout.String()
	if !strings.Contains(textOut, "Canonical Role Skill: Plan Reviewer") {
		t.Errorf("plan review prompt missing Canonical Role Skill header")
	}
	if !strings.Contains(textOut, "name: plan-reviewer") {
		t.Errorf("plan review prompt missing plan-reviewer skill frontmatter")
	}

	// 2. JSON output
	stdout.Reset()
	code = runPromptPlanReview([]string{"--dir", tmp, "T-040", "--json"}, env)
	if code != 0 {
		t.Fatalf("runPromptPlanReview --json failed with code %d: %s", code, stderr.String())
	}
	var ctx PlanReviewPromptContext
	if err := json.Unmarshal(stdout.Bytes(), &ctx); err != nil {
		t.Fatalf("failed to parse JSON context: %v", err)
	}
	if ctx.RoleSkill == "" {
		t.Errorf("expected non-empty RoleSkill in JSON context")
	}
	if !strings.Contains(ctx.RoleSkill, "name: plan-reviewer") {
		t.Errorf("JSON RoleSkill missing plan-reviewer frontmatter")
	}
}

func TestPromptReview_RoleSkillLens(t *testing.T) {
	tmp := t.TempDir()
	setupTestTask(t, tmp, "T-040")

	var stdout, stderr bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
		Now: time.Now,
	}

	// 1. Spec lens
	code := runPromptReview([]string{"--dir", tmp, "T-040", "--lens", "spec"}, env)
	if code != 0 {
		t.Fatalf("runPromptReview failed with code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "name: spec-auditor") {
		t.Errorf("spec lens prompt missing spec-auditor role skill")
	}

	// 2. Security lens (JSON)
	stdout.Reset()
	code = runPromptReview([]string{"--dir", tmp, "T-040", "--lens", "security", "--json"}, env)
	if code != 0 {
		t.Fatalf("runPromptReview --json failed with code %d: %s", code, stderr.String())
	}
	var ctx ReviewPromptContext
	if err := json.Unmarshal(stdout.Bytes(), &ctx); err != nil {
		t.Fatalf("failed to parse JSON context: %v", err)
	}
	if !strings.Contains(ctx.RoleSkill, "name: appsec-specialist") {
		t.Errorf("security lens JSON missing appsec-specialist role skill")
	}

	// 3. Pattern lens
	stdout.Reset()
	code = runPromptReview([]string{"--dir", tmp, "T-040", "--lens", "pattern"}, env)
	if code != 0 {
		t.Fatalf("runPromptReview failed with code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "name: pattern-auditor") {
		t.Errorf("pattern lens prompt missing pattern-auditor role skill")
	}

	// 4. Two-tier project override test (AC-5)
	overrideDir := filepath.Join(tmp, "skills", "spec-auditor")
	if err := os.MkdirAll(overrideDir, 0755); err != nil {
		t.Fatal(err)
	}
	overrideContent := "---\nname: spec-auditor\ndescription: Custom project override\n---\n# Custom Spec Auditor\n"
	if err := os.WriteFile(filepath.Join(overrideDir, "SKILL.md"), []byte(overrideContent), 0644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	code = runPromptReview([]string{"--dir", tmp, "T-040", "--lens", "spec"}, env)
	if code != 0 {
		t.Fatalf("runPromptReview with override failed with code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Custom project override") || !strings.Contains(stdout.String(), "# Custom Spec Auditor") {
		t.Errorf("prompt review did not use project override skill: %s", stdout.String())
	}
}

func TestSkillExport_DefaultBackwardCompat(t *testing.T) {
	tmp := t.TempDir()
	var stdout, stderr bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
		Now: time.Now,
	}

	code := runSkillExport([]string{"--dir", tmp}, env)
	if code != 0 {
		t.Fatalf("default runSkillExport failed with code %d: %s", code, stderr.String())
	}

	for _, name := range []string{"sdd-protocol", "code-review", "task-coordinator"} {
		skillFile := filepath.Join(tmp, "skills", name, "SKILL.md")
		if _, err := os.Stat(skillFile); err != nil {
			t.Fatalf("missing default canonical skill %s: %v", name, err)
		}
	}
}

func TestPromptImplement_TurnkeyPrompt(t *testing.T) {
	tmp := t.TempDir()
	setupTestTask(t, tmp, "T-041")

	var stdout, stderr bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
		Now: time.Now,
	}

	// 1. Default implementer prompt
	code := runPromptImplement([]string{"--dir", tmp, "T-041"}, env)
	if code != 0 {
		t.Fatalf("runPromptImplement failed with code %d: %s", code, stderr.String())
	}
	outText := stdout.String()
	if !strings.Contains(outText, "Canonical Role Skill: Senior Software Engineer (Implementer) (implementer)") {
		t.Errorf("prompt missing implementer role header")
	}
	if !strings.Contains(outText, "name: implementer") {
		t.Errorf("prompt missing implementer skill frontmatter")
	}
	if !strings.Contains(outText, "internal/test.go") {
		t.Errorf("prompt missing planned file boundary")
	}

	// 2. QA role prompt
	stdout.Reset()
	code = runPromptImplement([]string{"--dir", tmp, "T-041", "--role", "qa"}, env)
	if code != 0 {
		t.Fatalf("runPromptImplement --role qa failed with code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "name: qa") || !strings.Contains(stdout.String(), "Senior QA Automation Engineer (QA)") {
		t.Errorf("prompt missing QA role skill content")
	}

	// 3. DevOps role prompt
	stdout.Reset()
	code = runPromptImplement([]string{"--dir", tmp, "T-041", "--role", "devops"}, env)
	if code != 0 {
		t.Fatalf("runPromptImplement --role devops failed with code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "name: devops") || !strings.Contains(stdout.String(), "Senior DevOps / SRE Engineer (DevOps)") {
		t.Errorf("prompt missing DevOps role skill content")
	}

	// 4. JSON output
	stdout.Reset()
	code = runPromptImplement([]string{"--dir", tmp, "T-041", "--role", "implementer", "--json"}, env)
	if code != 0 {
		t.Fatalf("runPromptImplement --json failed with code %d: %s", code, stderr.String())
	}
	var ctx ImplementPromptContext
	if err := json.Unmarshal(stdout.Bytes(), &ctx); err != nil {
		t.Fatalf("failed to unmarshal JSON context: %v", err)
	}
	if ctx.Role != "implementer" {
		t.Errorf("expected role implementer, got %s", ctx.Role)
	}
	if !strings.Contains(ctx.RoleSkill, "name: implementer") {
		t.Errorf("JSON RoleSkill missing implementer frontmatter")
	}
	if len(ctx.PlannedFiles) == 0 || ctx.PlannedFiles[0] != "internal/test.go" {
		t.Errorf("unexpected planned files in JSON context: %v", ctx.PlannedFiles)
	}

	// 5. Invalid role
	stderr.Reset()
	code = runPromptImplement([]string{"--dir", tmp, "T-041", "--role", "invalid-role"}, env)
	if code != 1 {
		t.Fatalf("expected code 1 for invalid role, got %d", code)
	}
	if !strings.Contains(stderr.String(), "invalid role") {
		t.Errorf("expected invalid role error message, got: %s", stderr.String())
	}

	// 6. Invalid task ID
	stderr.Reset()
	code = runPromptImplement([]string{"--dir", tmp, "INVALID_TASK"}, env)
	if code != 1 {
		t.Fatalf("expected code 1 for invalid task ID, got %d", code)
	}
	if !strings.Contains(stderr.String(), "invalid task ID format") {
		t.Errorf("expected invalid task ID format error, got: %s", stderr.String())
	}
}

func TestPromptImplement_ConditionalPlanActivation(t *testing.T) {
	tmp := t.TempDir()
	taskDir := filepath.Join(tmp, "tasks", "features", "T-042-conditional-feature")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatal(err)
	}

	planWithOps := `# Plan
## 4. Implementation steps
| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Core logic | internal/app.go | go test ./internal/app |
| 2 | QA automation with race detector and fuzzing | internal/app_test.go | go test -race ./internal/app |
| 3 | DevOps CI/CD packaging and release pipeline | Makefile | make dist |

## 5. Files to create or modify
| Path | Action | Reason |
|---|---|---|
| internal/app.go | create | logic |
`
	if err := os.WriteFile(filepath.Join(taskDir, "requirements.md"), []byte("# Req\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "plan.md"), []byte(planWithOps), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "validation.md"), []byte("# Val\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
		Now: time.Now,
	}

	code := runPromptImplement([]string{"--dir", tmp, "T-042", "--json"}, env)
	if code != 0 {
		t.Fatalf("runPromptImplement failed with code %d: %s", code, stderr.String())
	}

	var ctx ImplementPromptContext
	if err := json.Unmarshal(stdout.Bytes(), &ctx); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(ctx.ActivatedSkills) != 2 {
		t.Fatalf("expected 2 activated operational skills, got %v", ctx.ActivatedSkills)
	}
	if ctx.ActivatedSkills[0] != "devops" || ctx.ActivatedSkills[1] != "qa" {
		t.Errorf("unexpected activated skills: %v", ctx.ActivatedSkills)
	}

	// Verify text output includes conditional activation section
	stdout.Reset()
	code = runPromptImplement([]string{"--dir", tmp, "T-042"}, env)
	if code != 0 {
		t.Fatalf("runPromptImplement text failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Conditional Operational Skills Activated by Plan") {
		t.Errorf("text prompt missing conditional operational skills section")
	}
	if !strings.Contains(stdout.String(), "- **qa**") || !strings.Contains(stdout.String(), "- **devops**") {
		t.Errorf("text prompt missing detected skill bullet points")
	}

	// Verify -race alone triggers QA detection (SEC-001)
	planWithRaceOnly := `# Plan
## 4. Implementation steps
| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Concurrency sweep | internal/app.go | go test -race ./... |
`
	if err := os.WriteFile(filepath.Join(taskDir, "plan.md"), []byte(planWithRaceOnly), 0644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	code = runPromptImplement([]string{"--dir", tmp, "T-042", "--json"}, env)
	if code != 0 {
		t.Fatalf("runPromptImplement failed: %s", stderr.String())
	}
	var ctxRace ImplementPromptContext
	if err := json.Unmarshal(stdout.Bytes(), &ctxRace); err != nil {
		t.Fatal(err)
	}
	if len(ctxRace.ActivatedSkills) != 1 || ctxRace.ActivatedSkills[0] != "qa" {
		t.Errorf("expected qa activated by -race alone, got: %v", ctxRace.ActivatedSkills)
	}
}

func TestSkillExport_AllNineRoles(t *testing.T) {
	tmp := t.TempDir()
	var stdout, stderr bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
		Now: time.Now,
	}

	code := runSkillExport([]string{"--dir", tmp, "--roles"}, env)
	if code != 0 {
		t.Fatalf("runSkillExport failed with code %d: %s", code, stderr.String())
	}

	names := role.CanonicalRoleNames()
	if len(names) != 9 {
		t.Fatalf("expected 9 canonical roles, got %d", len(names))
	}
	for _, name := range names {
		p := filepath.Join(tmp, "skills", name, "SKILL.md")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected exported skill %s, err: %v", name, err)
		}
	}
}

func TestSkillExport_ImplementerOps(t *testing.T) {
	tmp := t.TempDir()
	var stdout, stderr bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
		Now: time.Now,
	}

	for _, name := range []string{"implementer", "qa", "devops"} {
		stdout.Reset()
		code := runSkillExport([]string{"--dir", tmp, "--role", name}, env)
		if code != 0 {
			t.Fatalf("runSkillExport --role %s failed with code %d: %s", name, code, stderr.String())
		}
		p := filepath.Join(tmp, "skills", name, "SKILL.md")
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("missing exported skill %s: %v", name, err)
		}
		if !strings.Contains(string(data), "name: "+name) {
			t.Errorf("exported skill %s missing expected frontmatter", name)
		}
	}
}
