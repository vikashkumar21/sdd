package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/context"
)

func TestPromptDiscovery_Basic(t *testing.T) {
	tmp := t.TempDir()

	// Create fake go.mod and main.go
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module testmod\n\ngo 1.27\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
		Now: time.Now,
	}

	code := runPromptDiscovery([]string{"--dir", tmp}, env)
	if code != 0 {
		t.Fatalf("runPromptDiscovery failed with code %d: %s", code, stderr.String())
	}

	out := stdout.String()
	for _, expected := range []string{
		"Handover: Codebase Discovery & Forensic Extraction",
		"Codebase Analyst & Forensic Technical Auditor",
		"name: codebase-analyst",
		"1. AST & Module Map",
		"2. Design System & UI Patterns",
		"3. Data & State Modeling",
		"4. Anti-Patterns & Technical Debt",
		"Brownfield Living Document Synthesis Protocol",
		"TECH_STACK.md",
		"SKILLS.md",
		"ROADMAP.md",
		"Detected Languages:** go",
	} {
		if !strings.Contains(out, expected) {
			t.Errorf("expected discovery prompt to contain %q, but was missing", expected)
		}
	}
}

func TestPromptDiscovery_RoleSkill(t *testing.T) {
	tmp := t.TempDir()
	var stdout, stderr bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
		Now: time.Now,
	}

	code := runPromptDiscovery([]string{"--dir", tmp}, env)
	if code != 0 {
		t.Fatalf("runPromptDiscovery failed: %s", stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Canonical Role Skill: Staff Reverse-Engineer & Technical Auditor (codebase-analyst)") {
		t.Errorf("missing canonical role skill header in output")
	}
	if !strings.Contains(out, "name: codebase-analyst") {
		t.Errorf("missing codebase-analyst frontmatter in output")
	}
}

func TestPromptDiscovery_ExtractionChecklist(t *testing.T) {
	tmp := t.TempDir()
	var stdout, stderr bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
		Now: time.Now,
	}

	code := runPromptDiscovery([]string{"--dir", tmp}, env)
	if code != 0 {
		t.Fatalf("runPromptDiscovery failed: %s", stderr.String())
	}
	out := stdout.String()
	for _, pillar := range []string{
		"1. AST & Module Map (Architecture & Topology)",
		"2. Design System & UI Patterns (Frontend & Layout)",
		"3. Data & State Modeling (Data Flow & Storage)",
		"4. Anti-Patterns & Technical Debt (Hygiene & Pitfalls)",
	} {
		if !strings.Contains(out, pillar) {
			t.Errorf("missing pillar %q in extraction checklist", pillar)
		}
	}
}

func TestPromptDiscovery_LivingDocsProtocol(t *testing.T) {
	tmp := t.TempDir()
	var stdout, stderr bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
		Now: time.Now,
	}

	code := runPromptDiscovery([]string{"--dir", tmp}, env)
	if code != 0 {
		t.Fatalf("runPromptDiscovery failed: %s", stderr.String())
	}
	out := stdout.String()
	for _, doc := range []string{
		"TECH_STACK.md",
		"SKILLS.md",
		"ROADMAP.md",
		"Architecture & Directory Layout",
		"Decision Register",
		"Verified-Facts Log",
		"Dependency Register",
		"Pitfalls Log",
		"Parking Lot",
	} {
		if !strings.Contains(out, doc) {
			t.Errorf("missing living doc synthesis target %q", doc)
		}
	}
}

func TestPromptDiscovery_TaskResolution(t *testing.T) {
	tmp := t.TempDir()
	setupTestTask(t, tmp, "T-001")

	var stdout, stderr bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
		Now: time.Now,
	}

	// 1. Positional task ID
	code := runPromptDiscovery([]string{"--dir", tmp, "T-001"}, env)
	if code != 0 {
		t.Fatalf("runPromptDiscovery with positional task ID failed: %s", stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Task T-001 (tasks/features/T-001-test-feature)") {
		t.Errorf("missing task header in prompt:\n%s", out)
	}
	if !strings.Contains(out, "Requirements (`requirements.md`)") || !strings.Contains(out, "Implementation Plan (`plan.md`)") {
		t.Errorf("missing requirements or plan section in task-focused prompt")
	}

	// 2. Flag task ID
	stdout.Reset()
	code = runPromptDiscovery([]string{"--dir", tmp, "--task", "T-001"}, env)
	if code != 0 {
		t.Fatalf("runPromptDiscovery with --task failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Task T-001 (tasks/features/T-001-test-feature)") {
		t.Errorf("missing task header with --task flag")
	}
}

func TestPromptDiscovery_JSON(t *testing.T) {
	tmp := t.TempDir()
	setupTestTask(t, tmp, "T-002")

	// Stage context
	item := context.Item{
		Source: "raw-text",
		Origin: "spec-doc.txt",
		Data:   []byte("Discovery reference specification"),
	}
	if _, err := context.Stage(tmp, item, time.Now()); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
		Now: time.Now,
	}

	code := runPromptDiscovery([]string{"--dir", tmp, "T-002", "--json"}, env)
	if code != 0 {
		t.Fatalf("runPromptDiscovery --json failed with code %d: %s", code, stderr.String())
	}

	var ctx DiscoveryPromptContext
	if err := json.Unmarshal(stdout.Bytes(), &ctx); err != nil {
		t.Fatalf("failed to unmarshal JSON discovery context: %v", err)
	}

	if ctx.TaskID != "T-002" {
		t.Errorf("TaskID = %q, want T-002", ctx.TaskID)
	}
	if ctx.RoleSkill == "" || !strings.Contains(ctx.RoleSkill, "codebase-analyst") {
		t.Errorf("RoleSkill missing codebase-analyst content")
	}
	if len(ctx.LivingDocs) != 3 {
		t.Errorf("expected 3 living docs, got %d", len(ctx.LivingDocs))
	}
	if len(ctx.Checklist) != 4 {
		t.Errorf("expected 4 checklist pillars, got %d", len(ctx.Checklist))
	}
	if len(ctx.StagedContext) == 0 {
		t.Errorf("expected non-empty StagedContext")
	}
	if ctx.Prompt == "" {
		t.Errorf("expected non-empty Prompt text in JSON context")
	}
}

func TestPromptDiscovery_TwoTierOverride(t *testing.T) {
	tmp := t.TempDir()

	// Write custom override in skills/codebase-analyst/SKILL.md
	overrideDir := filepath.Join(tmp, "skills", "codebase-analyst")
	if err := os.MkdirAll(overrideDir, 0755); err != nil {
		t.Fatal(err)
	}
	overrideContent := "---\nname: codebase-analyst\ndescription: Custom reverse engineering override\n---\n# Custom Codebase Analyst\n"
	if err := os.WriteFile(filepath.Join(overrideDir, "SKILL.md"), []byte(overrideContent), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
		Now: time.Now,
	}

	code := runPromptDiscovery([]string{"--dir", tmp}, env)
	if code != 0 {
		t.Fatalf("runPromptDiscovery with override failed: %s", stderr.String())
	}

	if !strings.Contains(stdout.String(), "Custom reverse engineering override") || !strings.Contains(stdout.String(), "# Custom Codebase Analyst") {
		t.Errorf("discovery prompt did not use project override skill")
	}
}

func TestPromptDiscovery_NegativeCases(t *testing.T) {
	tmp := t.TempDir()

	var stdout, stderr bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
		Now: time.Now,
	}

	// 1. Invalid task ID format
	code := runPromptDiscovery([]string{"--dir", tmp, "INVALID-FORMAT"}, env)
	if code != 1 {
		t.Errorf("expected exit code 1 for invalid task format, got %d", code)
	}
	if !strings.Contains(stderr.String(), "invalid task ID format") {
		t.Errorf("expected invalid task ID format error, got: %s", stderr.String())
	}

	// 2. Non-existent task directory
	stderr.Reset()
	code = runPromptDiscovery([]string{"--dir", tmp, "T-987"}, env)
	if code != 1 {
		t.Errorf("expected exit code 1 for non-existent task, got %d", code)
	}
	if !strings.Contains(stderr.String(), "T-987 folder not found") {
		t.Errorf("expected no task directory error, got: %s", stderr.String())
	}

	// 3. Excess positional arguments
	stderr.Reset()
	code = runPromptDiscovery([]string{"--dir", tmp, "T-001", "EXTRA"}, env)
	if code != 2 {
		t.Errorf("expected exit code 2 for excess arguments, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage: sdd prompt discovery") {
		t.Errorf("expected usage error, got: %s", stderr.String())
	}

	// 4. Conflicting dual task specifications
	stderr.Reset()
	code = runPromptDiscovery([]string{"--dir", tmp, "T-001", "--task", "T-002"}, env)
	if code != 2 {
		t.Errorf("expected exit code 2 for conflicting task IDs, got %d", code)
	}
	if !strings.Contains(stderr.String(), "conflicting task IDs specified") {
		t.Errorf("expected conflicting task IDs error, got: %s", stderr.String())
	}
}

func TestPromptDiscovery_RoadmapFallback(t *testing.T) {
	tmp := t.TempDir()

	roadmapContent := `# ROADMAP

| ID | Title | Status |
|---|---|---|
| T-001 | Initial codebase discovery and exploration | Not started |
`
	if err := os.WriteFile(filepath.Join(tmp, "ROADMAP.md"), []byte(roadmapContent), 0644); err != nil {
		t.Fatal(err)
	}

	setupTestTask(t, tmp, "T-001")

	var stdout, stderr bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
		Now: time.Now,
	}

	// Run without explicit task ID; should discover T-001 from ROADMAP
	code := runPromptDiscovery([]string{"--dir", tmp}, env)
	if code != 0 {
		t.Fatalf("runPromptDiscovery failed: %s", stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "Task T-001 (tasks/features/T-001-test-feature)") {
		t.Errorf("expected roadmap discovery task T-001 to be picked up automatically:\n%s", out)
	}
}

func TestPromptDiscovery_PatchFallback(t *testing.T) {
	tmp := t.TempDir()
	taskDir := filepath.Join(tmp, "tasks", "bugfixes", "T-010-reproduce-bug")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "requirements.md"), []byte("# Requirements\nBug details\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "patch.md"), []byte("# Patch\nPatch details\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
		Now: time.Now,
	}

	code := runPromptDiscovery([]string{"--dir", tmp, "T-010"}, env)
	if code != 0 {
		t.Fatalf("runPromptDiscovery failed: %s", stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "Patch details") {
		t.Errorf("expected patch.md content in discovery prompt:\n%s", out)
	}
}

func TestPromptDispatch_Discovery(t *testing.T) {
	tmp := t.TempDir()
	var stdout, stderr bytes.Buffer
	env := IO{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
		Now: time.Now,
	}

	code := Run([]string{"prompt", "discovery", "--dir", tmp}, env)
	if code != 0 {
		t.Fatalf("Run prompt discovery failed with code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Codebase Analyst & Forensic Technical Auditor") {
		t.Errorf("prompt discovery dispatch missing expected content")
	}
}
