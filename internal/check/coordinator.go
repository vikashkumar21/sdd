package check

import (
	"strings"
)

// checkCoordinatorTierDispatch verifies that Task Coordinator skills and agent contracts
// enforce tier-calibrated subagent model dispatch, routing routine auditor roles (spec-auditor,
// pattern-auditor, qa, devops) to standard/flash models rather than inheriting expensive frontier models,
// and ensuring all Review Triad roles (spec-auditor, appsec-specialist, pattern-auditor) are declared with explicit tier mappings.
func (c *checker) checkCoordinatorTierDispatch(agents []string) {
	// 1. Check task-coordinator skill if present.
	// Gracefully skip if absent for backward compatibility with legacy and greenfield kits.
	coordSkillPath := "skills/task-coordinator/SKILL.md"
	if text, ok := c.load(coordSkillPath); ok {
		hasSpecAuditor := strings.Contains(text, "spec-auditor")
		hasPatternAuditor := strings.Contains(text, "pattern-auditor")
		hasAppSec := strings.Contains(text, "appsec-specialist")
		hasFlash := strings.Contains(text, `Model="flash"`) || strings.Contains(text, `Model: "flash"`)
		hasInheritOrPro := strings.Contains(text, `Model="inherit"`) || strings.Contains(text, `Model: "inherit"`) || strings.Contains(text, `Model="pro"`) || strings.Contains(text, `Model: "pro"`)

		if !hasSpecAuditor || !hasPatternAuditor || !hasAppSec {
			c.add(Error, coordSkillPath, 0, "coordinator-tier-dispatch",
				"task coordinator skill must declare all Review Triad roles: spec-auditor, appsec-specialist, and pattern-auditor")
		} else if !hasFlash || !hasInheritOrPro {
			c.add(Error, coordSkillPath, 0, "coordinator-tier-dispatch",
				"task coordinator skill must mandate explicit model tier dispatch (Model=\"flash\" for standard roles and Model=\"inherit\"/pro for frontier roles)")
		}
	}

	// 2. Check agent contracts if present.
	for _, agentFile := range agents {
		text, ok := c.load(agentFile)
		if !ok {
			continue
		}
		// Only check agent contracts that define SDD workflow (contain Phase C.2 or Review Triad)
		if strings.Contains(text, "Phase C.2") {
			hasSpec := strings.Contains(text, "spec-auditor")
			hasSec := strings.Contains(text, "appsec-specialist")
			hasPattern := strings.Contains(text, "pattern-auditor")
			hasFlash := strings.Contains(text, "flash") || strings.Contains(text, "standard model tier")
			hasTierDispatch := strings.Contains(text, "Tier-Calibrated Subagent Dispatch") || strings.Contains(text, "calibrated model tier")

			if !hasSpec || !hasSec || !hasPattern {
				c.add(Error, agentFile, 0, "coordinator-tier-dispatch",
					"agent contract %s must declare all Review Triad roles (spec-auditor, appsec-specialist, pattern-auditor) in Phase C.2", agentFile)
			} else if !hasFlash || !hasTierDispatch {
				c.add(Error, agentFile, 0, "coordinator-tier-dispatch",
					"agent contract %s must specify tier-calibrated subagent dispatch for routine review roles", agentFile)
			}
		}
	}
}
