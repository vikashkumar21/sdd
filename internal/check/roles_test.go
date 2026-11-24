package check_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vikashkumar21/sdd/internal/check"
	"github.com/vikashkumar21/sdd/internal/role"
)

func TestRolesCheck_Absent(t *testing.T) {
	tmp := t.TempDir()
	// An empty directory without .sdd/roles.json should have 0 roles-related findings
	r, err := check.Run(tmp)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range r.Findings {
		if f.File == ".sdd/roles.json" {
			t.Errorf("unexpected finding for absent roles manifest: %+v", f)
		}
	}
}

func TestRolesCheck_ValidDefault(t *testing.T) {
	tmp := t.TempDir()
	if err := role.SaveRoles(tmp, role.DefaultRoleConfig()); err != nil {
		t.Fatal(err)
	}

	r, err := check.Run(tmp)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range r.Findings {
		if f.File == ".sdd/roles.json" {
			t.Errorf("unexpected finding for default roles manifest: %+v", f)
		}
	}
}

func TestRolesCheck_SyntaxError(t *testing.T) {
	tmp := t.TempDir()
	path := role.RolesFilePath(tmp)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{not valid json`), 0644); err != nil {
		t.Fatal(err)
	}

	r, err := check.Run(tmp)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range r.Findings {
		if f.File == ".sdd/roles.json" && f.Rule == "roles-syntax" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected roles-syntax finding, got %+v", r.Findings)
	}
}

func TestRolesCheck_InvalidSchema(t *testing.T) {
	tmp := t.TempDir()
	path := role.RolesFilePath(tmp)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"schema": 42, "roles": {}}`), 0644); err != nil {
		t.Fatal(err)
	}

	r, err := check.Run(tmp)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range r.Findings {
		if f.File == ".sdd/roles.json" && f.Rule == "roles-schema" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected roles-schema finding, got %+v", r.Findings)
	}
}

func TestRolesCheck_EmptyRoles(t *testing.T) {
	tmp := t.TempDir()
	cfg := &role.RoleConfig{
		Schema: role.CurrentSchema,
		Roles:  map[string]role.RoleInfo{},
		Phases: map[string]role.PhaseMapping{},
	}
	if err := role.SaveRoles(tmp, cfg); err != nil {
		t.Fatal(err)
	}

	r, err := check.Run(tmp)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range r.Findings {
		if f.File == ".sdd/roles.json" && f.Rule == "roles-empty" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected roles-empty finding, got %+v", r.Findings)
	}
}

func TestRolesCheck_InvalidRoleName(t *testing.T) {
	tmp := t.TempDir()
	cfg := role.DefaultRoleConfig()
	cfg.Roles["Invalid_Name"] = role.RoleInfo{
		Skill:       "coordinator",
		Description: "Invalid name role",
	}
	if err := role.SaveRoles(tmp, cfg); err != nil {
		t.Fatal(err)
	}

	r, err := check.Run(tmp)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range r.Findings {
		if f.File == ".sdd/roles.json" && f.Rule == "roles-name" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected roles-name finding, got %+v", r.Findings)
	}
}

func TestRolesCheck_UnresolvedSkill(t *testing.T) {
	tmp := t.TempDir()
	cfg := role.DefaultRoleConfig()
	cfg.Roles["custom"] = role.RoleInfo{
		Skill:       "does-not-exist-anywhere",
		Description: "Unresolved skill test",
	}
	if err := role.SaveRoles(tmp, cfg); err != nil {
		t.Fatal(err)
	}

	r, err := check.Run(tmp)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range r.Findings {
		if f.File == ".sdd/roles.json" && f.Rule == "roles-skill-unresolved" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected roles-skill-unresolved finding, got %+v", r.Findings)
	}
}

func TestRolesCheck_InvalidSkillFrontmatter(t *testing.T) {
	tmp := t.TempDir()
	// Create project override skill with broken frontmatter
	skillDir := filepath.Join(tmp, "skills", "custom-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("not valid frontmatter"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := role.DefaultRoleConfig()
	cfg.Roles["custom-role"] = role.RoleInfo{
		Skill:       "custom-skill",
		Description: "Broken skill test",
	}
	if err := role.SaveRoles(tmp, cfg); err != nil {
		t.Fatal(err)
	}

	r, err := check.Run(tmp)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range r.Findings {
		if f.File == ".sdd/roles.json" && f.Rule == "roles-skill-invalid" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected roles-skill-invalid finding, got %+v", r.Findings)
	}
}

func TestRolesCheck_UnresolvedPhaseRole(t *testing.T) {
	tmp := t.TempDir()
	cfg := role.DefaultRoleConfig()
	cfg.Phases["phase-z"] = role.PhaseMapping{
		Planner: "non-existent-role",
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	path := role.RolesFilePath(tmp)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	r, err := check.Run(tmp)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range r.Findings {
		if f.File == ".sdd/roles.json" && f.Rule == "roles-phase-unresolved" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected roles-phase-unresolved finding, got %+v", r.Findings)
	}
}

func TestRolesCheck_InvalidSkillIdentifier(t *testing.T) {
	tmp := t.TempDir()
	cfg := role.DefaultRoleConfig()
	cfg.Roles["custom-role"] = role.RoleInfo{
		Skill:       "../../etc/passwd",
		Description: "Traversal skill test",
	}
	if err := role.SaveRoles(tmp, cfg); err != nil {
		t.Fatal(err)
	}

	r, err := check.Run(tmp)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range r.Findings {
		if f.File == ".sdd/roles.json" && f.Rule == "roles-skill-name" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected roles-skill-name finding, got %+v", r.Findings)
	}
}

func TestRolesCheck_DescriptionBounds(t *testing.T) {
	tmp := t.TempDir()
	cfg := role.DefaultRoleConfig()
	cfg.Roles["coordinator"] = role.RoleInfo{
		Skill:       "coordinator",
		Description: "   ",
	}
	if err := role.SaveRoles(tmp, cfg); err != nil {
		t.Fatal(err)
	}

	r, err := check.Run(tmp)
	if err != nil {
		t.Fatal(err)
	}

	foundEmpty := false
	for _, f := range r.Findings {
		if f.File == ".sdd/roles.json" && f.Rule == "roles-desc-empty" {
			foundEmpty = true
			break
		}
	}
	if !foundEmpty {
		t.Errorf("expected roles-desc-empty finding, got %+v", r.Findings)
	}
}

func TestRolesCheck_ExtendedPhaseMappingsUnresolved(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*role.RoleConfig)
		phase   string
		missing string
	}{
		{
			name: "unresolved implementer",
			mutate: func(cfg *role.RoleConfig) {
				cfg.Phases["phase-b"] = role.PhaseMapping{
					Implementer: "ghost-implementer",
				}
			},
			phase:   "phase-b",
			missing: "ghost-implementer",
		},
		{
			name: "unresolved qa",
			mutate: func(cfg *role.RoleConfig) {
				cfg.Phases["operations"] = role.PhaseMapping{
					QA: "ghost-qa",
				}
			},
			phase:   "operations",
			missing: "ghost-qa",
		},
		{
			name: "unresolved devops",
			mutate: func(cfg *role.RoleConfig) {
				cfg.Phases["operations"] = role.PhaseMapping{
					DevOps: "ghost-devops",
				}
			},
			phase:   "operations",
			missing: "ghost-devops",
		},
		{
			name: "unresolved analyst in phase-discovery",
			mutate: func(cfg *role.RoleConfig) {
				cfg.Phases["phase-discovery"] = role.PhaseMapping{
					Analyst: "ghost-analyst",
				}
			},
			phase:   "phase-discovery",
			missing: "ghost-analyst",
		},
		{
			name: "unresolved challenger in phase-discovery",
			mutate: func(cfg *role.RoleConfig) {
				cfg.Phases["phase-discovery"] = role.PhaseMapping{
					Analyst:    "codebase-analyst",
					Challenger: "ghost-challenger",
				}
			},
			phase:   "phase-discovery",
			missing: "ghost-challenger",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			cfg := role.DefaultRoleConfig()
			tc.mutate(cfg)

			if err := role.SaveRoles(tmp, cfg); err != nil {
				t.Fatal(err)
			}

			r, err := check.Run(tmp)
			if err != nil {
				t.Fatal(err)
			}

			found := false
			for _, f := range r.Findings {
				if f.File == ".sdd/roles.json" && f.Rule == "roles-phase-unresolved" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("[%s] expected roles-phase-unresolved finding for %s in %s, got: %+v", tc.name, tc.missing, tc.phase, r.Findings)
			}
		})
	}
}

func TestRolesCheck_LegacyMissingPhaseDiscovery(t *testing.T) {
	tmp := t.TempDir()
	cfg := role.DefaultRoleConfig()
	delete(cfg.Phases, "phase-discovery")

	if err := role.SaveRoles(tmp, cfg); err != nil {
		t.Fatal(err)
	}

	r, err := check.Run(tmp)
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range r.Findings {
		if f.File == ".sdd/roles.json" {
			t.Errorf("unexpected finding for legacy roles manifest missing phase-discovery: %+v", f)
		}
	}
}

func TestRolesCheck_ModelRouting(t *testing.T) {
	// 1. Valid default config passes cleanly
	tmp := t.TempDir()
	cfg := role.DefaultRoleConfig()
	if err := role.SaveRoles(tmp, cfg); err != nil {
		t.Fatal(err)
	}

	r, err := check.Run(tmp)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range r.Findings {
		if f.File == ".sdd/roles.json" {
			t.Errorf("unexpected finding for valid default roles: %+v", f)
		}
	}

	// 2. Invalid model_tier triggers roles-tier
	tmpTier := t.TempDir()
	rolesPath := role.RolesFilePath(tmpTier)
	if err := os.MkdirAll(filepath.Dir(rolesPath), 0755); err != nil {
		t.Fatal(err)
	}
	invalidTierJSON := `{
  "schema": 1,
  "roles": {
    "coordinator": {
      "skill": "coordinator",
      "description": "Coordinator",
      "model_tier": "hyper-fast",
      "reasoning_effort": "high",
      "recommended_family": "claude-sonnet"
    }
  },
  "phases": {}
}`
	if err := os.WriteFile(rolesPath, []byte(invalidTierJSON), 0644); err != nil {
		t.Fatal(err)
	}
	rTier, err := check.Run(tmpTier)
	if err != nil {
		t.Fatal(err)
	}
	hasTierErr := false
	for _, f := range rTier.Findings {
		if f.File == ".sdd/roles.json" && f.Rule == "roles-tier" {
			hasTierErr = true
			break
		}
	}
	if !hasTierErr {
		t.Errorf("expected roles-tier error for invalid model_tier, got findings: %+v", rTier.Findings)
	}

	// 3. Invalid reasoning_effort triggers roles-reasoning
	tmpEffort := t.TempDir()
	rolesPathEffort := role.RolesFilePath(tmpEffort)
	if err := os.MkdirAll(filepath.Dir(rolesPathEffort), 0755); err != nil {
		t.Fatal(err)
	}
	invalidEffortJSON := `{
  "schema": 1,
  "roles": {
    "coordinator": {
      "skill": "coordinator",
      "description": "Coordinator",
      "model_tier": "frontier",
      "reasoning_effort": "maximum",
      "recommended_family": "claude-sonnet"
    }
  },
  "phases": {}
}`
	if err := os.WriteFile(rolesPathEffort, []byte(invalidEffortJSON), 0644); err != nil {
		t.Fatal(err)
	}
	rEffort, err := check.Run(tmpEffort)
	if err != nil {
		t.Fatal(err)
	}
	hasEffortErr := false
	for _, f := range rEffort.Findings {
		if f.File == ".sdd/roles.json" && f.Rule == "roles-reasoning" {
			hasEffortErr = true
			break
		}
	}
	if !hasEffortErr {
		t.Errorf("expected roles-reasoning error for invalid reasoning_effort, got findings: %+v", rEffort.Findings)
	}

	// 4. Blank or multiline recommended_family triggers roles-family
	tmpFamily := t.TempDir()
	rolesPathFamily := role.RolesFilePath(tmpFamily)
	if err := os.MkdirAll(filepath.Dir(rolesPathFamily), 0755); err != nil {
		t.Fatal(err)
	}
	invalidFamilyJSON := `{
  "schema": 1,
  "roles": {
    "coordinator": {
      "skill": "coordinator",
      "description": "Coordinator",
      "model_tier": "frontier",
      "reasoning_effort": "high",
      "recommended_family": "claude\nsonnet"
    }
  },
  "phases": {}
}`
	if err := os.WriteFile(rolesPathFamily, []byte(invalidFamilyJSON), 0644); err != nil {
		t.Fatal(err)
	}
	rFamily, err := check.Run(tmpFamily)
	if err != nil {
		t.Fatal(err)
	}
	hasFamilyErr := false
	for _, f := range rFamily.Findings {
		if f.File == ".sdd/roles.json" && f.Rule == "roles-family" {
			hasFamilyErr = true
			break
		}
	}
	if !hasFamilyErr {
		t.Errorf("expected roles-family error for multiline recommended_family, got findings: %+v", rFamily.Findings)
	}

	// 5. Omitted fields in legacy roles pass cleanly
	tmpLegacy := t.TempDir()
	rolesPathLegacy := role.RolesFilePath(tmpLegacy)
	if err := os.MkdirAll(filepath.Dir(rolesPathLegacy), 0755); err != nil {
		t.Fatal(err)
	}
	legacyJSON := `{
  "schema": 1,
  "roles": {
    "coordinator": {
      "skill": "coordinator",
      "description": "Coordinator"
    }
  },
  "phases": {}
}`
	if err := os.WriteFile(rolesPathLegacy, []byte(legacyJSON), 0644); err != nil {
		t.Fatal(err)
	}
	rLegacy, err := check.Run(tmpLegacy)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range rLegacy.Findings {
		if f.File == ".sdd/roles.json" && (f.Rule == "roles-tier" || f.Rule == "roles-reasoning" || f.Rule == "roles-family") {
			t.Errorf("unexpected finding for legacy roles without model routing: %+v", f)
		}
	}
}
