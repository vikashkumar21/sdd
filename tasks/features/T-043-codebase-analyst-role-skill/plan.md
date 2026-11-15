---
status: active
task_id: T-043
last_reviewed: 2026-11-15
---
# Plan — T-043 Role-based agent skills: codebase-analyst and discovery phase mapping

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved

## 1. Approach summary
Author the canonical `codebase-analyst` role skill in `internal/templates/data/skills/codebase-analyst/SKILL.md` adhering to the `agentskills.io` standard. Expand `internal/role` to recognize `codebase-analyst` as the 10th canonical role, extend `PhaseMapping` with `Analyst`, map `phase-discovery` in `DefaultRoleConfig()` (which seeds `.sdd/roles.json` during `sdd init` and `sdd upgrade`), update this repository's `.sdd/roles.json`, extend mechanical `sdd check` validation in `internal/check/roles.go`, and verify 10-role export and upgrade preservation.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-039 | Role-based agent skill: codebase-analyst and discovery phase mapping | Option A (Dedicated canonical codebase-analyst role skill, phase-discovery mapping in .sdd/roles.json, 10-role export, and sdd check validation) / Option B (Rely on generic architect role without specialized discovery extraction instructions) | Option A: Standardizes brownfield reverse-engineering, ensures frontend CSS/fonts/design tokens and backend architecture patterns are extracted consistently, and enables turnkey discovery prompts. | **Approved** 2026-11-15 |

## 3. Design

### Canonical Role Skill Structure (`skills/codebase-analyst/SKILL.md`)
```markdown
---
name: codebase-analyst
description: Perform read-only forensic analysis on an existing brownfield codebase to discover directory topology, design systems/CSS/fonts, backend architectures, database schemas, and existing debt/pitfalls.
---

# SDD Codebase Analyst Role Standard

## Overview & Triggers
Activate this skill whenever:
- Operating in Phase A of brownfield task T-001 Discovery.
- Analyzing an existing codebase to synthesize ground truth for MISSION.md, TECH_STACK.md, SKILLS.md, or ROADMAP.md.

## Forensic Discovery Procedures
1. Phase 1: High-Level Topology & Toolchain Mapping
2. Phase 2: Design System, UI & Frontend Extraction (CSS framework, design tokens, typography/fonts, component hierarchy, state management)
3. Phase 3: Backend, Data & API Extraction (layering, routing, database/ORM, transaction boundaries)
4. Phase 4: Quality, Testing & Pitfalls Extraction (test runners, mocking conventions, tech debt, legacy gotchas for SKILLS.md §7)
5. Phase 5: Synthesis into Living Documents
```

### Schema & Phase Mapping Extension (`internal/role/role.go`)
```go
type PhaseMapping struct {
    Planner     string   `json:"planner,omitempty"`
    Reviewer    string   `json:"reviewer,omitempty"`
    Reviewers   []string `json:"reviewers,omitempty"`
    Coordinator string   `json:"coordinator,omitempty"`
    Implementer string   `json:"implementer,omitempty"`
    QA          string   `json:"qa,omitempty"`
    DevOps      string   `json:"devops,omitempty"`
    Analyst     string   `json:"analyst,omitempty"`
}

// CanonicalRoleNames includes 10 roles in sorted order:
// appsec-specialist, architect, codebase-analyst, coordinator, devops, implementer, pattern-auditor, plan-reviewer, qa, spec-auditor
```

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Author canonical `codebase-analyst` role skill template and update embedded skill tests | `internal/templates/data/skills/codebase-analyst/SKILL.md`, `internal/templates/templates_test.go` | `go test -v ./internal/templates -run TestEmbeddedRoleSkills` |
| 2 | Update `internal/role/role.go` with `codebase-analyst` role, `Analyst` in `PhaseMapping`, `phase-discovery` in `DefaultRoleConfig`, and unit tests | `internal/role/role.go`, `internal/role/role_test.go` | `go test -v ./internal/role` |
| 3 | Update `.sdd/roles.json` and `internal/check/roles.go` (`checkRoles` validating `Analyst` in `phase-discovery`) with unit tests in `internal/check/roles_test.go` | `.sdd/roles.json`, `internal/check/roles.go`, `internal/check/roles_test.go` | `go test -v ./internal/check -run TestRolesCheck` |
| 4 | Update `internal/cli/skill.go` (help text and export loops) and `internal/cli/skill_test.go` to support 10-role export and `--role codebase-analyst`; update scaffold and upgrade tests | `internal/cli/skill.go`, `internal/cli/skill_test.go`, `internal/scaffold/scaffold_test.go`, `internal/upgrade/upgrade_test.go` | `go test -v ./internal/cli -run TestSkillExport && go test -v ./internal/scaffold && go test -v ./internal/upgrade` |
| 5 | Update living docs (`TECH_STACK.md`, `ROADMAP.md`, `README.md`) and run full verification suite | `TECH_STACK.md`, `ROADMAP.md`, `README.md`, `tasks/features/T-043-codebase-analyst-role-skill/*` | `make verify && make check && ./bin/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `.sdd/roles.json` | modify | Adopt codebase-analyst role and phase-discovery mapping |
| `README.md` | modify | Document codebase-analyst role and 10 canonical skills |
| `ROADMAP.md` | modify | Track milestone and task lifecycle status |
| `TECH_STACK.md` | modify | Record Decision D-039 and verified facts |
| `internal/check/roles.go` | modify | Extend checkRoles to validate Analyst field in phase-discovery |
| `internal/check/roles_test.go` | modify | Unit tests for phase-discovery Analyst validation |
| `internal/cli/skill.go` | modify | Support exporting all 10 canonical roles and --role codebase-analyst |
| `internal/cli/skill_test.go` | modify | Test CLI export of codebase-analyst and 10 canonical roles |
| `internal/role/role.go` | modify | Add codebase-analyst, Analyst in PhaseMapping, and phase-discovery |
| `internal/role/role_test.go` | modify | Unit tests for 10 canonical roles and phase-discovery mapping |
| `internal/scaffold/scaffold_test.go` | modify | Test kit generation with updated roles.json |
| `internal/templates/data/skills/codebase-analyst/SKILL.md` | create | Canonical codebase-analyst role skill template |
| `internal/templates/templates_test.go` | modify | Assert 10 embedded role skills |
| `internal/upgrade/upgrade_test.go` | modify | Test upgrade preservation and migration of phase-discovery |
| `tasks/features/T-043-codebase-analyst-role-skill/plan.md` | modify | Task implementation plan |
| `tasks/features/T-043-codebase-analyst-role-skill/requirements.md` | modify | Task functional and non-functional requirements |
| `tasks/features/T-043-codebase-analyst-role-skill/review.md` | modify | Review Triad sign-off |
| `tasks/features/T-043-codebase-analyst-role-skill/validation.md` | modify | Traceability matrix and verification evidence |

## 6. Dependencies
None. Pure Go standard library per Decision D-007.

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Update T-043 status across lifecycle (`Specifying` -> `Implementing` -> `Awaiting review` -> `Done`) |
| `TECH_STACK.md` | Record Decision D-039 resolution and verified facts |
| `README.md` | Mention 10 canonical role skills and codebase-analyst discovery role |

## 8. Risks, mitigations, rollback
- **Risk:** Older `.sdd/roles.json` files missing `phase-discovery`.
  - *Mitigation:* `sdd check` treats `phase-discovery` as optional if absent; when present, it strictly validates the mapped roles.
- **Rollback:** Revert `internal/role` and templates back to 9 canonical roles.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- **Rule F-001 (Resolve all review findings: Blockers, Warnings, and Minors):** Worker agent must resolve all findings reported across the 3 subagent review lenses before presenting to human at GATE 2.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-15   Phrase received: `APPROVED T-043`
