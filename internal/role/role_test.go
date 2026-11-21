package role_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vikashkumar21/sdd/internal/role"
	"github.com/vikashkumar21/sdd/internal/templates"
)

func TestRoleConfigSchema(t *testing.T) {
	cfg := role.DefaultRoleConfig()
	if cfg.Schema != role.CurrentSchema {
		t.Fatalf("expected schema %d, got %d", role.CurrentSchema, cfg.Schema)
	}

	names := role.CanonicalRoleNames()
	if len(names) != 11 {
		t.Fatalf("expected 11 canonical role names, got %d", len(names))
	}

	for _, name := range names {
		info, ok := cfg.Roles[name]
		if !ok {
			t.Errorf("missing canonical role %q in default config", name)
			continue
		}
		if info.Skill != name {
			t.Errorf("role %q has skill %q, expected %q", name, info.Skill, name)
		}
		if info.Description == "" {
			t.Errorf("role %q has empty description", name)
		}
	}

	// Verify phase mappings
	phaseDiscovery, ok := cfg.Phases["phase-discovery"]
	if !ok || phaseDiscovery.Analyst != "codebase-analyst" || phaseDiscovery.Challenger != "discovery-challenger" || phaseDiscovery.Planner != "architect" {
		t.Errorf("unexpected phase-discovery mapping: %+v", phaseDiscovery)
	}

	phaseA, ok := cfg.Phases["phase-a"]
	if !ok || phaseA.Planner != "architect" || phaseA.Reviewer != "plan-reviewer" {
		t.Errorf("unexpected phase-a mapping: %+v", phaseA)
	}

	phaseB, ok := cfg.Phases["phase-b"]
	if !ok || phaseB.Implementer != "implementer" {
		t.Errorf("unexpected phase-b mapping: %+v", phaseB)
	}

	phaseC2, ok := cfg.Phases["phase-c.2"]
	if !ok || len(phaseC2.Reviewers) != 3 {
		t.Errorf("unexpected phase-c.2 mapping: %+v", phaseC2)
	}

	phaseD, ok := cfg.Phases["phase-d"]
	if !ok || phaseD.Coordinator != "coordinator" {
		t.Errorf("unexpected phase-d mapping: %+v", phaseD)
	}

	gov, ok := cfg.Phases["governance"]
	if !ok || gov.Coordinator != "coordinator" {
		t.Errorf("unexpected governance mapping: %+v", gov)
	}

	ops, ok := cfg.Phases["operations"]
	if !ok || ops.QA != "qa" || ops.DevOps != "devops" {
		t.Errorf("unexpected operations mapping: %+v", ops)
	}
}

func TestCanonicalRoleNames(t *testing.T) {
	names := role.CanonicalRoleNames()
	expected := []string{
		"appsec-specialist",
		"architect",
		"codebase-analyst",
		"coordinator",
		"devops",
		"discovery-challenger",
		"implementer",
		"pattern-auditor",
		"plan-reviewer",
		"qa",
		"spec-auditor",
	}
	if len(names) != len(expected) {
		t.Fatalf("expected %d names, got %d", len(expected), len(names))
	}
	for i, v := range expected {
		if names[i] != v {
			t.Errorf("at index %d: expected %q, got %q", i, v, names[i])
		}
	}
}

func TestLoadRoles_Default(t *testing.T) {
	tmp := t.TempDir()
	cfg, err := role.LoadRoles(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil || cfg.Schema != role.CurrentSchema {
		t.Fatalf("expected default config, got %+v", cfg)
	}
	if len(cfg.Roles) != 11 {
		t.Errorf("expected 11 default roles, got %d", len(cfg.Roles))
	}
}

func TestSaveAndLoadRoles(t *testing.T) {
	tmp := t.TempDir()
	cfg := role.DefaultRoleConfig()
	cfg.Roles["custom-role"] = role.RoleInfo{
		Skill:       "custom-skill",
		Description: "Custom specialized worker",
	}

	if err := role.SaveRoles(tmp, cfg); err != nil {
		t.Fatalf("failed to save roles: %v", err)
	}

	loaded, err := role.LoadRoles(tmp)
	if err != nil {
		t.Fatalf("failed to load roles: %v", err)
	}

	if loaded.Schema != cfg.Schema {
		t.Errorf("schema mismatch: expected %d, got %d", cfg.Schema, loaded.Schema)
	}

	custom, ok := loaded.Roles["custom-role"]
	if !ok || custom.Skill != "custom-skill" {
		t.Errorf("custom role not preserved: %+v", custom)
	}

	roleNames := loaded.RoleNames()
	if len(roleNames) != 12 {
		t.Errorf("expected 12 role names, got %d", len(roleNames))
	}
}

func TestLoadRoles_InvalidSchema(t *testing.T) {
	tmp := t.TempDir()
	path := role.RolesFilePath(tmp)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	invalidJSON := `{"schema": 999, "roles": {}}`
	if err := os.WriteFile(path, []byte(invalidJSON), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := role.LoadRoles(tmp)
	if err == nil {
		t.Fatal("expected error on invalid schema version, got nil")
	}
}

func TestLoadRoles_InvalidJSON(t *testing.T) {
	tmp := t.TempDir()
	path := role.RolesFilePath(tmp)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{not valid json`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := role.LoadRoles(tmp)
	if err == nil {
		t.Fatal("expected error on invalid JSON, got nil")
	}
}

func TestResolveSkillTwoTier(t *testing.T) {
	tmp := t.TempDir()

	// 1. In empty tmp dir, resolving non-existent skill fails
	_, err := role.ResolveSkill(tmp, "non-existent-skill", nil)
	if err == nil {
		t.Fatal("expected error resolving non-existent skill, got nil")
	}

	// 2. Put skill in .sdd/skills/test-role/SKILL.md (infrastructure tier)
	infraPath := filepath.Join(tmp, ".sdd", "skills", "test-role", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(infraPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(infraPath, []byte("INFRASTRUCTURE CONTENT"), 0644); err != nil {
		t.Fatal(err)
	}

	content, err := role.ResolveSkill(tmp, "test-role", nil)
	if err != nil {
		t.Fatalf("unexpected error resolving infra skill: %v", err)
	}
	if content != "INFRASTRUCTURE CONTENT" {
		t.Errorf("expected INFRASTRUCTURE CONTENT, got %q", content)
	}

	// 3. Put skill in skills/test-role/SKILL.md (project override tier)
	projPath := filepath.Join(tmp, "skills", "test-role", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(projPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projPath, []byte("PROJECT OVERRIDE CONTENT"), 0644); err != nil {
		t.Fatal(err)
	}

	// Project override must take precedence over infra tier
	content, err = role.ResolveSkill(tmp, "test-role", nil)
	if err != nil {
		t.Fatalf("unexpected error resolving project override skill: %v", err)
	}
	if content != "PROJECT OVERRIDE CONTENT" {
		t.Errorf("expected PROJECT OVERRIDE CONTENT, got %q", content)
	}

	// 4. Resolve embedded skill (task-coordinator exists in embedded templates)
	set, err := templates.New("")
	if err != nil {
		t.Fatal(err)
	}
	coordContent, err := role.ResolveSkill(tmp, "task-coordinator", set)
	if err != nil {
		t.Fatalf("unexpected error resolving embedded task-coordinator: %v", err)
	}
	if coordContent == "" {
		t.Error("expected non-empty embedded task-coordinator content")
	}

	// 5. Empty and invalid traversal skill name errors
	if _, err := role.ResolveSkill(tmp, "", nil); err == nil {
		t.Error("expected error on empty skill name, got nil")
	}
	for _, bad := range []string{"../escape", "../../etc/passwd", "/absolute", "name with spaces", "UPPERCASE"} {
		if _, err := role.ResolveSkill(tmp, bad, nil); err == nil {
			t.Errorf("expected error resolving traversal/invalid skill %q, got nil", bad)
		}
	}
}

func TestResolveCoordinatorAlias(t *testing.T) {
	tmp := t.TempDir()

	// 1. Embedded resolution: both coordinator and task-coordinator resolve with correct frontmatter names
	cContent, err := role.ResolveSkill(tmp, "coordinator", nil)
	if err != nil {
		t.Fatalf("failed to resolve embedded coordinator: %v", err)
	}
	if !strings.HasPrefix(cContent, "---\nname: coordinator\n") {
		t.Errorf("expected coordinator frontmatter name, got:\n%s", cContent[:40])
	}

	tcContent, err := role.ResolveSkill(tmp, "task-coordinator", nil)
	if err != nil {
		t.Fatalf("failed to resolve embedded task-coordinator: %v", err)
	}
	if !strings.HasPrefix(tcContent, "---\nname: task-coordinator\n") {
		t.Errorf("expected task-coordinator frontmatter name, got:\n%s", tcContent[:40])
	}

	// 2. Project override aliasing: only skills/coordinator exists, task-coordinator resolves it with normalized name
	tmp2 := t.TempDir()
	coordDir := filepath.Join(tmp2, "skills", "coordinator")
	if err := os.MkdirAll(coordDir, 0755); err != nil {
		t.Fatal(err)
	}
	customCoord := "---\nname: coordinator\ndescription: Custom\n---\n# Custom Coordinator Body\n"
	if err := os.WriteFile(filepath.Join(coordDir, "SKILL.md"), []byte(customCoord), 0644); err != nil {
		t.Fatal(err)
	}

	aliasedTC, err := role.ResolveSkill(tmp2, "task-coordinator", nil)
	if err != nil {
		t.Fatalf("failed to resolve aliased task-coordinator: %v", err)
	}
	if !strings.HasPrefix(aliasedTC, "---\nname: task-coordinator\n") {
		t.Errorf("expected normalized task-coordinator frontmatter, got:\n%s", aliasedTC[:50])
	}
	if !strings.Contains(aliasedTC, "# Custom Coordinator Body") {
		t.Errorf("expected custom body in aliased resolution")
	}

	// 3. Reverse project override aliasing: only skills/task-coordinator exists, coordinator resolves it with normalized name
	tmp3 := t.TempDir()
	tcDir := filepath.Join(tmp3, "skills", "task-coordinator")
	if err := os.MkdirAll(tcDir, 0755); err != nil {
		t.Fatal(err)
	}
	customTC := "---\nname: task-coordinator\ndescription: Custom TC\n---\n# Custom Task Coordinator Body\n"
	if err := os.WriteFile(filepath.Join(tcDir, "SKILL.md"), []byte(customTC), 0644); err != nil {
		t.Fatal(err)
	}

	aliasedCoord, err := role.ResolveSkill(tmp3, "coordinator", nil)
	if err != nil {
		t.Fatalf("failed to resolve aliased coordinator: %v", err)
	}
	if !strings.HasPrefix(aliasedCoord, "---\nname: coordinator\n") {
		t.Errorf("expected normalized coordinator frontmatter, got:\n%s", aliasedCoord[:50])
	}
	if !strings.Contains(aliasedCoord, "# Custom Task Coordinator Body") {
		t.Errorf("expected custom body in aliased resolution")
	}
}
