---
status: completed
task_id: T-040
last_reviewed: 2026-11-15
---
# Plan — T-040 Role-based agent skills: core governance, planning, and review triad

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved

## 1. Approach summary
Author canonical `agentskills.io` compliant skills for the 6 core governance and review roles (`coordinator`, `architect`, `plan-reviewer`, `spec-auditor`, `appsec-specialist`, `pattern-auditor`), introduce a deterministic phase-to-role mapping manifest (`.sdd/roles.json`), implement mechanical consistency validation in `sdd check`, enrich `sdd prompt plan-review` and `sdd prompt review --lens` with role skills, and extend `sdd skill export` with `--roles` and `--role <name>`.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-036 | Role skill taxonomy, phase-to-role mapping (.sdd/roles.json), and two-tier storage architecture | Option A (Two-tier storage with .sdd/skills/ infrastructure baseline and skills/ project override, .sdd/roles.json manifest, pure Go stdlib parser, and turnkey sdd prompt/export integration) / Option B (Hardcoded prompt templates without standalone skills) | Option A: Adopts Linux Foundation/Anthropic agentskills.io standard, enables clean SDK integration for autonomous agents, avoids repository root clutter, and supports user customization. | Approved (2026-11-14) |

## 3. Design

### Data Structures & Manifest (`internal/role/role.go`)
```go
package role

type RoleInfo struct {
    Skill       string `json:"skill"`
    Description string `json:"description"`
}

type PhaseMapping struct {
    Planner     string   `json:"planner,omitempty"`
    Reviewer    string   `json:"reviewer,omitempty"`
    Reviewers   []string `json:"reviewers,omitempty"`
    Coordinator string   `json:"coordinator,omitempty"`
}

type RoleConfig struct {
    Schema int                     `json:"schema"`
    Roles  map[string]RoleInfo     `json:"roles"`
    Phases map[string]PhaseMapping `json:"phases"`
}
```

### Two-Tier Skill Resolution
```go
// ResolveSkill looks for a skill in:
// 1. <root>/skills/<name>/SKILL.md (project override)
// 2. <root>/.sdd/skills/<name>/SKILL.md (infrastructure baseline)
// 3. Embedded templates (internal/templates/data/skills/<name>/SKILL.md)
func ResolveSkill(root, skillName string, set *templates.Set) (string, error)
```

### Canonical Role Skills
1. `coordinator`: Task Coordinator orchestration, git worktree provisioning, deconfliction analysis, merge simulation, and Gate 2 presentation.
2. `architect`: Phase A requirements extraction, system architecture, architectural decision formulation (`D-xxx`), boundary setting.
3. `plan-reviewer`: Pre-GATE 1 independent audit (rubric, boundary precision, risk/rollback, open questions check).
4. `spec-auditor`: Phase C.2 Review Triad (traceability of R-xxx, negative coverage, plan boundaries, stdlib purity).
5. `appsec-specialist`: Phase C.2 Review Triad (threat surface, bounded inputs, filesystem confinement, secrets hygiene, command safety).
6. `pattern-auditor`: Phase C.2 Review Triad (decision conformance, determinism, language idioms, Rule F-001).

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement `internal/role` package with `.sdd/roles.json` loader, schema validation, and two-tier skill resolver | `internal/role/role.go`, `internal/role/role_test.go` | `go test -v ./internal/role` |
| 2 | Author 6 canonical governance skills in `internal/templates/data/skills/` and wire embedded templates | `internal/templates/data/skills/*`, `internal/templates/templates_test.go` | `go test -v ./internal/templates` |
| 3 | Add `sdd check` validation rules for `.sdd/roles.json` and role skill conformance | `internal/check/roles.go`, `internal/check/roles_test.go`, `internal/check/check.go` | `go test -v ./internal/check -run TestRoles` |
| 4 | Enrich `sdd prompt plan-review` and `sdd prompt review --lens` with role skill instructions, and extend `sdd skill export` with `--roles` and `--role` | `internal/cli/prompt.go`, `internal/cli/skill.go`, `internal/cli/skill_test.go` | `go test -v ./internal/cli -run "TestSkillExport_Roles|TestPrompt"` |
| 5 | Seed `.sdd/roles.json` in `internal/scaffold/plan.go` and `internal/upgrade/upgrade.go`, create project `.sdd/roles.json`, update living docs, and run full verification | `internal/scaffold/plan.go`, `internal/scaffold/scaffold_test.go`, `internal/upgrade/upgrade.go`, `internal/upgrade/upgrade_test.go`, `.sdd/roles.json`, `README.md`, `ROADMAP.md`, `TECH_STACK.md` | `make verify && make check && ./bin/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `.sdd/roles.json` | create | Canonical project phase-to-role mapping manifest |
| `internal/role/role.go` | create | Role configuration, phase mapping, and two-tier skill resolver |
| `internal/role/role_test.go` | create | Unit tests for role loader, schema validation, and two-tier resolution |
| `internal/templates/data/skills/coordinator/SKILL.md` | create | Canonical Task Coordinator agentskills.io skill |
| `internal/templates/data/skills/architect/SKILL.md` | create | Canonical Architect / Planner agentskills.io skill |
| `internal/templates/data/skills/plan-reviewer/SKILL.md` | create | Canonical Plan Reviewer agentskills.io skill |
| `internal/templates/data/skills/spec-auditor/SKILL.md` | create | Canonical Spec Auditor Review Triad skill |
| `internal/templates/data/skills/appsec-specialist/SKILL.md` | create | Canonical AppSec Specialist Review Triad skill |
| `internal/templates/data/skills/pattern-auditor/SKILL.md` | create | Canonical Pattern and Quality Auditor Review Triad skill |
| `internal/templates/templates_test.go` | modify | Tests for embedded role skills |
| `internal/check/roles.go` | create | Mechanical consistency check for roles manifest and role skills |
| `internal/check/roles_test.go` | create | Unit tests for roles manifest and skill check rules |
| `internal/check/check.go` | modify | Wire roles check into sdd check |
| `internal/cli/prompt.go` | modify | Enrich plan-review and review prompts with dynamic role skill instructions |
| `internal/cli/skill.go` | modify | Add --roles and --role flags to sdd skill export |
| `internal/cli/skill_test.go` | create | Tests for sdd skill export --roles and --role |
| `internal/scaffold/plan.go` | modify | Seed default .sdd/roles.json on sdd init |
| `internal/scaffold/scaffold_test.go` | modify | Tests for .sdd/roles.json scaffolding on init |
| `internal/upgrade/upgrade.go` | modify | Seed default .sdd/roles.json on sdd upgrade |
| `internal/upgrade/upgrade_test.go` | modify | Tests for .sdd/roles.json upgrading |
| `README.md` | modify | Document role skills, .sdd/roles.json, and sdd skill export --roles |
| `ROADMAP.md` | modify | Track milestone and task lifecycle status |
| `TECH_STACK.md` | modify | Record Decision D-036 and verified facts |
| `tasks/features/T-040-governance-role-skills/plan.md` | modify | Architecture plan and implementation steps |
| `tasks/features/T-040-governance-role-skills/requirements.md` | modify | Specify requirements R-001 to R-010 |
| `tasks/features/T-040-governance-role-skills/validation.md` | modify | Traceability matrix and verification evidence |
| `tasks/features/T-040-governance-role-skills/review.md` | modify | Review Triad evaluation and sign-off |

## 6. Dependencies
None. Pure Go standard library (`os`, `path/filepath`, `encoding/json`, `errors`, `fmt`, `strings`, `sort`) per Decision D-007.

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Update T-040 status across lifecycle (`Specifying` -> `Implementing` -> `Awaiting review` -> `Done`) |
| `TECH_STACK.md` | Approve Decision D-036 upon human approval and record role skill architecture |
| `README.md` | Add documentation for role skills, `.sdd/roles.json`, and `sdd skill export --roles` |

## 8. Risks, mitigations, rollback
- **Risk:** Duplicate rules across 6 role skills causing rule drift.
  - *Mitigation:* Role skills focus strictly on role-specific boundaries, inputs/outputs, and rubrics, referencing `sdd-protocol` for general gating rules.
- **Risk:** Missing `.sdd/roles.json` in brownfield kits failing `sdd check`.
  - *Mitigation:* `sdd upgrade` automatically seeds `.sdd/roles.json`, and `LoadRoles` provides fallback defaults when absent.
- **Rollback:** Remove `internal/role/`, revert CLI wiring and check rules.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- **Rule F-001 (Resolve all review findings: Blockers, Warnings, and Minors):** Worker agent must resolve all findings reported across the 3 subagent review lenses before presenting to human at GATE 2.

## 10. Approval record
Approved by: Vikash   Date: 2026-11-14   Phrase received: `APPROVED T-040`
