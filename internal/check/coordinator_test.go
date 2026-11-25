package check

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckCoordinatorTierDispatch_Valid(t *testing.T) {
	root := t.TempDir()

	// 1. Valid task-coordinator skill
	skillDir := filepath.Join(root, "skills", "task-coordinator")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	skillContent := `---
name: task-coordinator
description: Task coordinator standard.
---
# Task Coordinator Standard
### 5. Heterogeneous Role Model Routing & Tier Enforcement
- spec-auditor: standard tier, Model="flash"
- appsec-specialist: frontier tier, Model="inherit"
- pattern-auditor: standard tier, Model="flash"
invoke_subagent(Role="Independent Spec Auditor", Model="flash", Prompt="...")
invoke_subagent(Role="Independent AppSec Specialist", Model="inherit", Prompt="...")
invoke_subagent(Role="Independent Pattern & Quality Auditor", Model="flash", Prompt="...")
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	// 2. Valid agent contract
	claudeContent := `# CLAUDE.md
### Phase C.2 — 3-subagent parallel review triad
- Spec Auditor Lens (spec-auditor): standard model tier (Model: "flash" / haiku)
- AppSec Specialist Lens (appsec-specialist): frontier model tier (Model: "inherit" / pro)
- Pattern & Quality Auditor Lens (pattern-auditor): standard model tier (Model: "flash" / haiku)
- Tier-Calibrated Subagent Dispatch: coordinator must specify Model parameter.
`
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte(claudeContent), 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}

	c := &checker{root: root, texts: map[string]string{}}
	c.checkCoordinatorTierDispatch([]string{"CLAUDE.md"})

	if len(c.findings) != 0 {
		t.Fatalf("expected 0 findings, got %d: %+v", len(c.findings), c.findings)
	}
}

func TestCheckCoordinatorTierDispatch_MissingSkill_PassesCleanly(t *testing.T) {
	root := t.TempDir()

	// No skills/task-coordinator/SKILL.md present (legacy or greenfield kit)
	c := &checker{root: root, texts: map[string]string{}}
	c.checkCoordinatorTierDispatch([]string{"CLAUDE.md"})

	if len(c.findings) != 0 {
		t.Fatalf("expected 0 findings when skill is absent, got %d: %+v", len(c.findings), c.findings)
	}
}

func TestCheckCoordinatorTierDispatch_SkillMissingRole(t *testing.T) {
	root := t.TempDir()

	skillDir := filepath.Join(root, "skills", "task-coordinator")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Mentions spec-auditor and pattern-auditor, but omits appsec-specialist
	skillContent := `---
name: task-coordinator
description: Task coordinator standard.
---
# Task Coordinator Standard
### 5. Role Dispatch
- spec-auditor: Model="flash"
- pattern-auditor: Model="flash"
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	c := &checker{root: root, texts: map[string]string{}}
	c.checkCoordinatorTierDispatch(nil)

	if len(c.findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(c.findings), c.findings)
	}
	f := c.findings[0]
	if f.Rule != "coordinator-tier-dispatch" || f.Severity != Error {
		t.Errorf("unexpected finding: %+v", f)
	}
}

func TestCheckCoordinatorTierDispatch_SkillMissingFlash(t *testing.T) {
	root := t.TempDir()

	skillDir := filepath.Join(root, "skills", "task-coordinator")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Mentions all three roles but omits explicit Model="flash" assignment
	skillContent := `---
name: task-coordinator
description: Task coordinator standard.
---
# Task Coordinator Standard
### 5. Role Dispatch
- spec-auditor: standard tier
- appsec-specialist: frontier tier, Model="inherit"
- pattern-auditor: standard tier
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	c := &checker{root: root, texts: map[string]string{}}
	c.checkCoordinatorTierDispatch(nil)

	if len(c.findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(c.findings), c.findings)
	}
	f := c.findings[0]
	if f.Rule != "coordinator-tier-dispatch" || f.Severity != Error {
		t.Errorf("unexpected finding: %+v", f)
	}
}

func TestCheckCoordinatorTierDispatch_ContractMissingRole(t *testing.T) {
	root := t.TempDir()

	// Contract mentions Phase C.2 and spec-auditor, but omits pattern-auditor or appsec-specialist
	claudeContent := `# CLAUDE.md
### Phase C.2 — 3-subagent parallel review triad
- Spec Auditor Lens (spec-auditor): standard model tier (Model: "flash")
`
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte(claudeContent), 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}

	c := &checker{root: root, texts: map[string]string{}}
	c.checkCoordinatorTierDispatch([]string{"CLAUDE.md"})

	if len(c.findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(c.findings), c.findings)
	}
	f := c.findings[0]
	if f.Rule != "coordinator-tier-dispatch" || f.Severity != Error {
		t.Errorf("unexpected finding: %+v", f)
	}
}

func TestCheckCoordinatorTierDispatch_ContractMissingTierDispatch(t *testing.T) {
	root := t.TempDir()

	// Contract mentions Phase C.2 and all triad roles, but lacks tier dispatch rules
	claudeContent := `# CLAUDE.md
### Phase C.2 — 3-subagent parallel review triad
- Spec Auditor Lens (spec-auditor): audits requirements.
- AppSec Specialist Lens (appsec-specialist): audits security.
- Pattern Auditor Lens (pattern-auditor): audits architecture.
`
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte(claudeContent), 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}

	c := &checker{root: root, texts: map[string]string{}}
	c.checkCoordinatorTierDispatch([]string{"CLAUDE.md"})

	if len(c.findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(c.findings), c.findings)
	}
	f := c.findings[0]
	if f.Rule != "coordinator-tier-dispatch" || f.Severity != Error {
		t.Errorf("unexpected finding: %+v", f)
	}
}
