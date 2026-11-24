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

// ModelTier represents an abstract model capability class for heterogeneous routing.
type ModelTier string

const (
	TierFrontier ModelTier = "frontier"
	TierAdvanced ModelTier = "advanced"
	TierStandard ModelTier = "standard"
	TierFast     ModelTier = "fast"
)

// AllowedModelTiers lists all valid model tier identifiers in canonical order.
func AllowedModelTiers() []string {
	return []string{
		string(TierFrontier),
		string(TierAdvanced),
		string(TierStandard),
		string(TierFast),
	}
}

// ValidateModelTier verifies whether tier is a valid ModelTier.
func ValidateModelTier(tier string) error {
	t := strings.ToLower(strings.TrimSpace(tier))
	if t == "" {
		return errors.New("model tier cannot be empty")
	}
	switch ModelTier(t) {
	case TierFrontier, TierAdvanced, TierStandard, TierFast:
		return nil
	default:
		return fmt.Errorf("invalid model tier %q (expected one of: %s)", tier, strings.Join(AllowedModelTiers(), ", "))
	}
}

// ReasoningEffort represents the recommended reasoning effort for a role.
type ReasoningEffort string

const (
	ReasoningHigh   ReasoningEffort = "high"
	ReasoningMedium ReasoningEffort = "medium"
	ReasoningLow    ReasoningEffort = "low"
	ReasoningNone   ReasoningEffort = "none"
)

// AllowedReasoningEfforts lists all valid reasoning effort identifiers in canonical order.
func AllowedReasoningEfforts() []string {
	return []string{
		string(ReasoningHigh),
		string(ReasoningMedium),
		string(ReasoningLow),
		string(ReasoningNone),
	}
}

// ValidateReasoningEffort verifies whether effort is a valid ReasoningEffort.
func ValidateReasoningEffort(effort string) error {
	e := strings.ToLower(strings.TrimSpace(effort))
	if e == "" {
		return errors.New("reasoning effort cannot be empty")
	}
	switch ReasoningEffort(e) {
	case ReasoningHigh, ReasoningMedium, ReasoningLow, ReasoningNone:
		return nil
	default:
		return fmt.Errorf("invalid reasoning effort %q (expected one of: %s)", effort, strings.Join(AllowedReasoningEfforts(), ", "))
	}
}

// ModelRouting captures model routing metadata for an agent role.
type ModelRouting struct {
	Tier              string `json:"model_tier,omitempty"`
	ReasoningEffort   string `json:"reasoning_effort,omitempty"`
	RecommendedFamily string `json:"recommended_family,omitempty"`
}

// RoleInfo defines metadata and the corresponding skill name for an agent role.
type RoleInfo struct {
	Skill             string `json:"skill"`
	Description       string `json:"description"`
	ModelTier         string `json:"model_tier,omitempty"`
	ReasoningEffort   string `json:"reasoning_effort,omitempty"`
	RecommendedFamily string `json:"recommended_family,omitempty"`
}

// Routing returns the ModelRouting representation for r.
func (r RoleInfo) Routing() ModelRouting {
	return ModelRouting{
		Tier:              r.ModelTier,
		ReasoningEffort:   r.ReasoningEffort,
		RecommendedFamily: r.RecommendedFamily,
	}
}

// PhaseMapping defines role assignments for SDD workflow phases.
type PhaseMapping struct {
	Planner     string   `json:"planner,omitempty"`
	Reviewer    string   `json:"reviewer,omitempty"`
	Reviewers   []string `json:"reviewers,omitempty"`
	Coordinator string   `json:"coordinator,omitempty"`
	Implementer string   `json:"implementer,omitempty"`
	QA          string   `json:"qa,omitempty"`
	DevOps      string   `json:"devops,omitempty"`
	Analyst     string   `json:"analyst,omitempty"`
	Challenger  string   `json:"challenger,omitempty"`
}

// RoleConfig represents the configuration stored in .sdd/roles.json.
type RoleConfig struct {
	Schema int                     `json:"schema"`
	Roles  map[string]RoleInfo     `json:"roles"`
	Phases map[string]PhaseMapping `json:"phases"`
}

// CanonicalRoleNames returns the sorted list of canonical role identifiers.
func CanonicalRoleNames() []string {
	return []string{
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
}

// DefaultRoleConfig returns a fresh RoleConfig seeded with standard SDD governance roles and phases.
func DefaultRoleConfig() *RoleConfig {
	return &RoleConfig{
		Schema: CurrentSchema,
		Roles: map[string]RoleInfo{
			"coordinator": {
				Skill:             "coordinator",
				Description:       "Multi-agent workspace governance, task decomposition, state tracking, and merge qualification",
				ModelTier:         string(TierFrontier),
				ReasoningEffort:   string(ReasoningMedium),
				RecommendedFamily: "claude-sonnet / gemini-pro / gpt-4o",
			},
			"architect": {
				Skill:             "architect",
				Description:       "Phase A requirements synthesis, architectural decision formulation (D-xxx), boundary setting, and risk modeling",
				ModelTier:         string(TierFrontier),
				ReasoningEffort:   string(ReasoningHigh),
				RecommendedFamily: "claude-opus / gemini-pro / o1",
			},
			"codebase-analyst": {
				Skill:             "codebase-analyst",
				Description:       "Forensic codebase analysis, design system & CSS/font extraction, backend architecture mapping, schema discovery, and tech debt auditing",
				ModelTier:         string(TierFrontier),
				ReasoningEffort:   string(ReasoningHigh),
				RecommendedFamily: "claude-opus / gemini-pro / o1",
			},
			"discovery-challenger": {
				Skill:             "discovery-challenger",
				Description:       "Adversarial codebase challenger: audits discovery findings, uncovers hidden entry points, implicit fallbacks, orphan code, and unreferenced architecture",
				ModelTier:         string(TierFrontier),
				ReasoningEffort:   string(ReasoningHigh),
				RecommendedFamily: "claude-opus / gemini-pro / o1",
			},
			"plan-reviewer": {
				Skill:             "plan-reviewer",
				Description:       "Pre-GATE 1 independent plan audit, rubric checking, boundary precision, and open questions verification",
				ModelTier:         string(TierAdvanced),
				ReasoningEffort:   string(ReasoningHigh),
				RecommendedFamily: "claude-sonnet / gemini-pro / o3-mini",
			},
			"spec-auditor": {
				Skill:             "spec-auditor",
				Description:       "Phase C.2 Review Triad lens: requirements traceability, negative test coverage, plan boundaries, and stdlib purity",
				ModelTier:         string(TierStandard),
				ReasoningEffort:   string(ReasoningMedium),
				RecommendedFamily: "claude-sonnet / gemini-flash / gpt-4o",
			},
			"appsec-specialist": {
				Skill:             "appsec-specialist",
				Description:       "Phase C.2 Review Triad lens: threat surface, input validation, filesystem confinement, secrets hygiene, and command safety",
				ModelTier:         string(TierFrontier),
				ReasoningEffort:   string(ReasoningHigh),
				RecommendedFamily: "claude-opus / gemini-pro / o1",
			},
			"pattern-auditor": {
				Skill:             "pattern-auditor",
				Description:       "Phase C.2 Review Triad lens: decision conformance, determinism, language idioms, and Rule F-001",
				ModelTier:         string(TierStandard),
				ReasoningEffort:   string(ReasoningMedium),
				RecommendedFamily: "claude-sonnet / gemini-flash / gpt-4o",
			},
			"implementer": {
				Skill:             "implementer",
				Description:       "Phase B step-by-step implementation, strict plan boundary enforcement, and determinism",
				ModelTier:         string(TierAdvanced),
				ReasoningEffort:   string(ReasoningMedium),
				RecommendedFamily: "claude-sonnet / gemini-flash / gpt-4o",
			},
			"qa": {
				Skill:             "qa",
				Description:       "Senior QA automation engineer; integration testing; negative coverage; race detection; determinism; fuzzing",
				ModelTier:         string(TierStandard),
				ReasoningEffort:   string(ReasoningLow),
				RecommendedFamily: "claude-haiku / gemini-flash / gpt-4o-mini",
			},
			"devops": {
				Skill:             "devops",
				Description:       "Senior DevOps / SRE engineer; build automation; CI/CD recipes; packaging; distribution",
				ModelTier:         string(TierStandard),
				ReasoningEffort:   string(ReasoningLow),
				RecommendedFamily: "claude-haiku / gemini-flash / gpt-4o-mini",
			},
		},
		Phases: map[string]PhaseMapping{
			"phase-discovery": {
				Analyst:    "codebase-analyst",
				Challenger: "discovery-challenger",
				Planner:    "architect",
			},
			"phase-a": {
				Planner:  "architect",
				Reviewer: "plan-reviewer",
			},
			"phase-b": {
				Implementer: "implementer",
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
			"operations": {
				QA:     "qa",
				DevOps: "devops",
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

// ResolveRoleRouting resolves model routing metadata for roleName.
// It checks cfg.Roles[roleName], falling back to DefaultRoleConfig() defaults for any missing fields,
// or generic defaults (TierFrontier, ReasoningMedium, "claude-sonnet / gemini-pro / gpt-4o") if unconfigured.
func ResolveRoleRouting(cfg *RoleConfig, roleName string) ModelRouting {
	defCfg := DefaultRoleConfig()
	var defInfo RoleInfo
	if defCfg != nil && defCfg.Roles != nil {
		defInfo = defCfg.Roles[roleName]
	}

	var configured RoleInfo
	if cfg != nil && cfg.Roles != nil {
		configured = cfg.Roles[roleName]
	}

	tier := configured.ModelTier
	if tier == "" || ValidateModelTier(tier) != nil {
		tier = defInfo.ModelTier
	}
	// Custom or unconfigured roles without valid routing safely default to Frontier governance routing.
	if tier == "" || ValidateModelTier(tier) != nil {
		tier = string(TierFrontier)
	}

	effort := configured.ReasoningEffort
	if effort == "" || ValidateReasoningEffort(effort) != nil {
		effort = defInfo.ReasoningEffort
	}
	// Custom or unconfigured roles without valid reasoning effort safely default to Medium effort.
	if effort == "" || ValidateReasoningEffort(effort) != nil {
		effort = string(ReasoningMedium)
	}

	family := strings.TrimSpace(configured.RecommendedFamily)
	if family == "" {
		family = defInfo.RecommendedFamily
	}
	if family == "" {
		family = "claude-sonnet / gemini-pro / gpt-4o"
	}

	return ModelRouting{
		Tier:              tier,
		ReasoningEffort:   effort,
		RecommendedFamily: family,
	}
}

// ResolveSkill resolves a skill in two tiers:
// 1. <root>/skills/<name>/SKILL.md (project override)
// 2. <root>/.sdd/skills/<name>/SKILL.md (infrastructure baseline)
// 3. Embedded templates (skills/<name>/SKILL.md)
//
// If not found directly, it falls back to symmetrical aliasing (e.g. coordinator <-> task-coordinator)
// with frontmatter name normalization matching the requested skill name.
//
// If set is nil, it automatically instantiates templates.New("").
func ResolveSkill(root, skillName string, set *templates.Set) (string, error) {
	if err := skill.ValidateName(skillName); err != nil {
		return "", fmt.Errorf("invalid skill name %q: %w", skillName, err)
	}
	if strings.Contains(skillName, "..") || strings.ContainsAny(skillName, `/\:`) {
		return "", fmt.Errorf("invalid skill name %q: must contain no path separators", skillName)
	}

	var alias string
	switch skillName {
	case "coordinator":
		alias = "task-coordinator"
	case "task-coordinator":
		alias = "coordinator"
	}

	// 1. Project override: <root>/skills/<name>/SKILL.md
	if root != "" {
		projPath := filepath.Join(root, "skills", skillName, "SKILL.md")
		if data, err := os.ReadFile(projPath); err == nil {
			return string(data), nil
		}
		if alias != "" {
			aliasProjPath := filepath.Join(root, "skills", alias, "SKILL.md")
			if data, err := os.ReadFile(aliasProjPath); err == nil {
				return normalizeFrontmatterName(string(data), alias, skillName), nil
			}
		}

		// 2. Infrastructure baseline: <root>/.sdd/skills/<name>/SKILL.md
		infraPath := filepath.Join(root, ".sdd", "skills", skillName, "SKILL.md")
		if data, err := os.ReadFile(infraPath); err == nil {
			return string(data), nil
		}
		if alias != "" {
			aliasInfraPath := filepath.Join(root, ".sdd", "skills", alias, "SKILL.md")
			if data, err := os.ReadFile(aliasInfraPath); err == nil {
				return normalizeFrontmatterName(string(data), alias, skillName), nil
			}
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

	if alias != "" {
		aliasTemplatePath := "skills/" + alias + "/SKILL.md"
		aliasContent, err := set.Raw(aliasTemplatePath)
		if err == nil && aliasContent != "" {
			return normalizeFrontmatterName(aliasContent, alias, skillName), nil
		}
	}

	return "", fmt.Errorf("role skill %q not found in project, .sdd, or embedded templates", skillName)
}

func normalizeFrontmatterName(content, fromName, toName string) string {
	fromHeader := "---\nname: " + fromName + "\n"
	toHeader := "---\nname: " + toName + "\n"
	if strings.HasPrefix(content, fromHeader) {
		return strings.Replace(content, fromHeader, toHeader, 1)
	}
	fromHeaderCRLF := "---\r\nname: " + fromName + "\r\n"
	toHeaderCRLF := "---\r\nname: " + toName + "\r\n"
	if strings.HasPrefix(content, fromHeaderCRLF) {
		return strings.Replace(content, fromHeaderCRLF, toHeaderCRLF, 1)
	}
	return content
}
