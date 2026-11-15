// Package role manages role-based agent skills, phase-to-role mappings,
// and two-tier skill resolution for autonomous multi-agent SDD workflows.
package role

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vikashkumar21/sdd/internal/fsutil"
	"github.com/vikashkumar21/sdd/internal/skill"
	"github.com/vikashkumar21/sdd/internal/templates"
)

// CurrentSchema defines the schema version for .sdd/roles.json.
const CurrentSchema = 1

// RoleInfo defines metadata and the corresponding skill name for an agent role.
type RoleInfo struct {
	Skill       string `json:"skill"`
	Description string `json:"description"`
}

// PhaseMapping defines role assignments for SDD workflow phases.
type PhaseMapping struct {
	Planner     string   `json:"planner,omitempty"`
	Reviewer    string   `json:"reviewer,omitempty"`
	Reviewers   []string `json:"reviewers,omitempty"`
	Coordinator string   `json:"coordinator,omitempty"`
}

// RoleConfig represents the configuration stored in .sdd/roles.json.
type RoleConfig struct {
	Schema int                     `json:"schema"`
	Roles  map[string]RoleInfo     `json:"roles"`
	Phases map[string]PhaseMapping `json:"phases"`
}

// CanonicalRoleNames returns the sorted list of canonical governance role identifiers.
func CanonicalRoleNames() []string {
	return []string{
		"appsec-specialist",
		"architect",
		"coordinator",
		"pattern-auditor",
		"plan-reviewer",
		"spec-auditor",
	}
}

// DefaultRoleConfig returns a fresh RoleConfig seeded with standard SDD governance roles and phases.
func DefaultRoleConfig() *RoleConfig {
	return &RoleConfig{
		Schema: CurrentSchema,
		Roles: map[string]RoleInfo{
			"coordinator": {
				Skill:       "coordinator",
				Description: "Multi-agent workspace governance, task decomposition, state tracking, and merge qualification",
			},
			"architect": {
				Skill:       "architect",
				Description: "Phase A requirements synthesis, architectural decision formulation (D-xxx), boundary setting, and risk modeling",
			},
			"plan-reviewer": {
				Skill:       "plan-reviewer",
				Description: "Pre-GATE 1 independent plan audit, rubric checking, boundary precision, and open questions verification",
			},
			"spec-auditor": {
				Skill:       "spec-auditor",
				Description: "Phase C.2 Review Triad lens: requirements traceability, negative test coverage, plan boundaries, and stdlib purity",
			},
			"appsec-specialist": {
				Skill:       "appsec-specialist",
				Description: "Phase C.2 Review Triad lens: threat surface, input validation, filesystem confinement, secrets hygiene, and command safety",
			},
			"pattern-auditor": {
				Skill:       "pattern-auditor",
				Description: "Phase C.2 Review Triad lens: decision conformance, determinism, language idioms, and Rule F-001",
			},
		},
		Phases: map[string]PhaseMapping{
			"phase-a": {
				Planner:  "architect",
				Reviewer: "plan-reviewer",
			},
			"phase-c.2": {
				Reviewers: []string{
					"spec-auditor",
					"appsec-specialist",
					"pattern-auditor",
				},
			},
			"phase-d": {
				Coordinator: "coordinator",
			},
			"governance": {
				Coordinator: "coordinator",
			},
		},
	}
}

// RolesFilePath returns the expected path to .sdd/roles.json in root.
func RolesFilePath(root string) string {
	return filepath.Join(root, ".sdd", "roles.json")
}

// LoadRoles loads and parses .sdd/roles.json from root. If the file does not exist,
// it returns the default configuration and a nil error.
func LoadRoles(root string) (*RoleConfig, error) {
	path := RolesFilePath(root)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultRoleConfig(), nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var cfg RoleConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	if cfg.Schema != CurrentSchema {
		return nil, fmt.Errorf("%s has invalid schema version %d (expected %d)", path, cfg.Schema, CurrentSchema)
	}

	if cfg.Roles == nil {
		cfg.Roles = make(map[string]RoleInfo)
	}
	if cfg.Phases == nil {
		cfg.Phases = make(map[string]PhaseMapping)
	}

	return &cfg, nil
}

// SaveRoles atomically writes cfg to .sdd/roles.json in root.
func SaveRoles(root string, cfg *RoleConfig) error {
	if cfg == nil {
		return errors.New("cannot save nil role config")
	}
	path := RolesFilePath(root)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal roles json: %w", err)
	}
	data = append(data, '\n')

	if err := fsutil.WriteAtomic(path, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// RoleNames returns a sorted slice of all configured role names.
func (c *RoleConfig) RoleNames() []string {
	if c == nil || len(c.Roles) == 0 {
		return nil
	}
	names := make([]string, 0, len(c.Roles))
	for name := range c.Roles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ResolveSkill resolves a skill in two tiers:
// 1. <root>/skills/<name>/SKILL.md (project override)
// 2. <root>/.sdd/skills/<name>/SKILL.md (infrastructure baseline)
// 3. Embedded templates (skills/<name>/SKILL.md)
//
// If set is nil, it automatically instantiates templates.New("").
func ResolveSkill(root, skillName string, set *templates.Set) (string, error) {
	if skillName == "" {
		return "", errors.New("skill name cannot be empty")
	}
	if !skill.NameRegex.MatchString(skillName) || strings.Contains(skillName, "..") || strings.ContainsAny(skillName, `/\:`) {
		return "", fmt.Errorf("invalid skill name %q: must match agentskills.io format and contain no path separators", skillName)
	}

	// 1. Project override: <root>/skills/<name>/SKILL.md
	if root != "" {
		projPath := filepath.Join(root, "skills", skillName, "SKILL.md")
		if data, err := os.ReadFile(projPath); err == nil {
			return string(data), nil
		}

		// 2. Infrastructure baseline: <root>/.sdd/skills/<name>/SKILL.md
		infraPath := filepath.Join(root, ".sdd", "skills", skillName, "SKILL.md")
		if data, err := os.ReadFile(infraPath); err == nil {
			return string(data), nil
		}
	}

	// 3. Embedded template
	if set == nil {
		var err error
		set, err = templates.New("")
		if err != nil {
			return "", fmt.Errorf("init templates: %w", err)
		}
	}

	templatePath := "skills/" + skillName + "/SKILL.md"
	content, err := set.Raw(templatePath)
	if err == nil && content != "" {
		return content, nil
	}

	return "", fmt.Errorf("role skill %q not found in project, .sdd, or embedded templates", skillName)
}
