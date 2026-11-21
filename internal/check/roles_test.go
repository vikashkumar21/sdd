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
