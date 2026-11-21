package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vikashkumar21/sdd/internal/role"
	"github.com/vikashkumar21/sdd/internal/skill"
)

func (c *checker) checkRoles() {
	relPath := ".sdd/roles.json"
	fullPath := filepath.Join(c.root, filepath.FromSlash(relPath))

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Backward compatibility: kits without .sdd/roles.json pass check cleanly
			return
		}
		c.add(Error, relPath, 0, "roles-read", "cannot read roles manifest: %v", err)
		return
	}

	var cfg role.RoleConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		c.add(Error, relPath, 0, "roles-syntax", "invalid JSON syntax in %s: %v", relPath, err)
		return
	}

	if cfg.Schema != role.CurrentSchema {
		c.add(Error, relPath, 0, "roles-schema", "%s has unsupported schema %d (expected %d)", relPath, cfg.Schema, role.CurrentSchema)
	}

	if len(cfg.Roles) == 0 {
		c.add(Warn, relPath, 0, "roles-empty", "%s defines no roles", relPath)
	}

	// Deterministic validation of roles
	roleNames := make([]string, 0, len(cfg.Roles))
	for name := range cfg.Roles {
		roleNames = append(roleNames, name)
	}
	sort.Strings(roleNames)

	for _, name := range roleNames {
		info := cfg.Roles[name]
		if !skill.NameRegex.MatchString(name) {
			c.add(Error, relPath, 0, "roles-name", "role %q has invalid identifier (must be 1-64 lowercase alphanumeric chars and hyphens)", name)
		}
		if info.Skill == "" {
			c.add(Error, relPath, 0, "roles-skill-missing", "role %q has no skill configured", name)
			continue
		}
		if err := skill.ValidateName(info.Skill); err != nil {
			c.add(Error, relPath, 0, "roles-skill-name", "role %q has invalid skill identifier %q: %v", name, info.Skill, err)
			continue
		}

		descTrim := strings.TrimSpace(info.Description)
		if descTrim == "" {
			c.add(Error, relPath, 0, "roles-desc-empty", "role %q has empty description", name)
		} else if len(descTrim) > 1024 {
			c.add(Error, relPath, 0, "roles-desc-length", "role %q description exceeds maximum 1024 characters (%d)", name, len(descTrim))
		}

		// Resolve role skill across project, .sdd, and embedded templates
		skillContent, err := role.ResolveSkill(c.root, info.Skill, nil)
		if err != nil {
			c.add(Error, relPath, 0, "roles-skill-unresolved", "role %q references unresolved skill %q: %v", name, info.Skill, err)
			continue
		}

		// Validate skill conforms to agentskills.io standard
		parsedSkill, err := skill.Parse([]byte(skillContent))
		if err != nil {
			c.add(Error, relPath, 0, "roles-skill-invalid", "role %q skill %q failed frontmatter parsing: %v", name, info.Skill, err)
			continue
		}

		valErrs := skill.Validate(parsedSkill, info.Skill)
		for _, vErr := range valErrs {
			c.add(Error, relPath, 0, "roles-skill-invalid", "role %q skill %q validation error: %v", name, info.Skill, vErr)
		}
	}

	// Deterministic validation of phase mappings
	phaseNames := make([]string, 0, len(cfg.Phases))
	for name := range cfg.Phases {
		phaseNames = append(phaseNames, name)
	}
	sort.Strings(phaseNames)

	for _, pName := range phaseNames {
		m := cfg.Phases[pName]
		if m.Planner != "" {
			if _, ok := cfg.Roles[m.Planner]; !ok {
				c.add(Error, relPath, 0, "roles-phase-unresolved", "phase %q references undefined planner role %q", pName, m.Planner)
			}
		}
		if m.Reviewer != "" {
			if _, ok := cfg.Roles[m.Reviewer]; !ok {
				c.add(Error, relPath, 0, "roles-phase-unresolved", "phase %q references undefined reviewer role %q", pName, m.Reviewer)
			}
		}
		for _, r := range m.Reviewers {
			if _, ok := cfg.Roles[r]; !ok {
				c.add(Error, relPath, 0, "roles-phase-unresolved", "phase %q references undefined reviewer role %q", pName, r)
			}
		}
		if m.Coordinator != "" {
			if _, ok := cfg.Roles[m.Coordinator]; !ok {
				c.add(Error, relPath, 0, "roles-phase-unresolved", "phase %q references undefined coordinator role %q", pName, m.Coordinator)
			}
		}
		if m.Implementer != "" {
			if _, ok := cfg.Roles[m.Implementer]; !ok {
				c.add(Error, relPath, 0, "roles-phase-unresolved", "phase %q references undefined implementer role %q", pName, m.Implementer)
			}
		}
		if m.QA != "" {
			if _, ok := cfg.Roles[m.QA]; !ok {
				c.add(Error, relPath, 0, "roles-phase-unresolved", "phase %q references undefined qa role %q", pName, m.QA)
			}
		}
		if m.DevOps != "" {
			if _, ok := cfg.Roles[m.DevOps]; !ok {
				c.add(Error, relPath, 0, "roles-phase-unresolved", "phase %q references undefined devops role %q", pName, m.DevOps)
			}
		}
		if m.Analyst != "" {
			if _, ok := cfg.Roles[m.Analyst]; !ok {
				c.add(Error, relPath, 0, "roles-phase-unresolved", "phase %q references undefined analyst role %q", pName, m.Analyst)
			}
		}
		if m.Challenger != "" {
			if _, ok := cfg.Roles[m.Challenger]; !ok {
				c.add(Error, relPath, 0, "roles-phase-unresolved", "phase %q references undefined challenger role %q", pName, m.Challenger)
			}
		}
	}
}
