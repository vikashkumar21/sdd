---
status: active
task_id: T-041
last_reviewed: 2026-11-15
---
# Plan — T-041 Role-based agent skills: implementer, QA, and DevOps operations

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved

## 1. Approach summary
Author canonical `agentskills.io` compliant skills for execution and operations roles (`implementer`, `qa`, `devops`), extend `.sdd/roles.json` with Phase B and operations phase mappings, introduce `sdd prompt implement` for turnkey execution handover with conditional QA/DevOps plan detection, extend `sdd skill export`, and implement mechanical validation in `sdd check`.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-037 | Execution and operations role skills taxonomy, conditional plan activation, and roles manifest extension | Option A (Canonical implementer, qa, and devops role skills, phase-b and operations mappings in .sdd/roles.json, conditional plan detection in sdd prompt implement, and turnkey export integration) / Option B (Ad-hoc unstructured prompt instructions without standard skills) | Option A: Adopts Linux Foundation/Anthropic agentskills.io standard, establishes clear senior/staff engineer personas, enforces strict file boundaries during implementation, and activates QA/DevOps conditionally without altering core gates. | Open |

## 3. Design

### Data Structures & Extended Manifest (`internal/role/role.go`)
```go
package role

type PhaseMapping struct {
    Planner     string   `json:"planner,omitempty"`
    Reviewer    string   `json:"reviewer,omitempty"`
    Reviewers   []string `json:"reviewers,omitempty"`
    Coordinator string   `json:"coordinator,omitempty"`
    Implementer string   `json:"implementer,omitempty"`
    QA          string   `json:"qa,omitempty"`
    DevOps      string   `json:"devops,omitempty"`
}
```

### Canonical Role Taxonomy (9 Roles)
1. `coordinator`: Multi-agent workspace governance, task decomposition, state tracking, and merge qualification.
2. `architect`: Phase A requirements synthesis, architectural decision formulation (`D-xxx`), boundary setting, and risk modeling.
3. `plan-reviewer`: Pre-GATE 1 independent plan audit, rubric checking, boundary precision, and open questions verification.
4. `implementer`: Phase B step-by-step implementation, strict plan boundary enforcement (Section 5 only), table-driven testing, and determinism.
5. `spec-auditor`: Phase C.2 Review Triad lens: requirements traceability, negative test coverage, plan boundaries, and stdlib purity.
6. `appsec-specialist`: Phase C.2 Review Triad lens: threat surface, input validation, filesystem confinement, secrets hygiene, and command safety.
7. `pattern-auditor`: Phase C.2 Review Triad lens: decision conformance, determinism, language idioms, and Rule F-001.
8. `qa`: Senior QA automation engineer; integration testing; negative coverage; race detection (`-race`); determinism (`-count=2`); fuzzing. Activated when a plan names QA.
9. `devops`: Senior DevOps / SRE engineer; build automation; CI/CD recipes; packaging; distribution. Activated when a plan names DevOps.

### Turnkey Execution Prompting (`sdd prompt implement <T-xxx>`)
- Flags: `--role <implementer|qa|devops>`, `--dir <project>`, `--json`.
- Handover Context:
  - Injects task requirements (`requirements.md`), approved plan (`plan.md`), validation matrix (`validation.md`), and planned file boundaries.
  - Injects target role skill instructions (`implementer` by default, or `qa` / `devops` when specified).
  - Conditionally detects if `plan.md` mentions QA or DevOps in its steps (case-insensitive regex and keyword scanning of Section 4 plan steps for `\b(qa|quality assurance|fuzz|race detector|benchmark)\b` and `\b(devops|sre|ci/cd|docker|container|packaging|release pipeline)\b`) and flags activated operational skills in the prompt overview.

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Extend `internal/role/role.go` with 3 new roles, `PhaseMapping` fields, updated `DefaultRoleConfig()`, and 9 canonical role names; update `internal/role/role_test.go` | `internal/role/role.go`, `internal/role/role_test.go` | `go test -v ./internal/role` |
| 2 | Author canonical skills in `internal/templates/data/skills/` (`implementer`, `qa`, `devops`) and update `internal/templates/templates_test.go` | `internal/templates/data/skills/implementer/SKILL.md`, `internal/templates/data/skills/qa/SKILL.md`, `internal/templates/data/skills/devops/SKILL.md`, `internal/templates/templates_test.go` | `go test -v ./internal/templates` |
| 3 | Implement `sdd prompt implement` subcommand with `--role` flag and conditional plan detection in `internal/cli/prompt.go`, wire dispatch in `internal/cli/commands.go` and `internal/cli/cli.go`, and update `internal/cli/skill_test.go` | `internal/cli/prompt.go`, `internal/cli/commands.go`, `internal/cli/cli.go`, `internal/cli/skill_test.go` | `go test -v ./internal/cli -run "TestPromptImplement"` |
| 4 | Update `internal/check/roles.go` to validate new phase mappings (`Implementer`, `QA`, `DevOps`) and update `internal/check/roles_test.go` | `internal/check/roles.go`, `internal/check/roles_test.go` | `go test -v ./internal/check -run TestRoles` |
| 5 | Update `.sdd/roles.json`, update scaffolding/upgrade in `internal/scaffold/plan.go`, `internal/scaffold/scaffold_test.go`, `internal/upgrade/upgrade.go`, `internal/upgrade/upgrade_test.go`, update living docs, and run full verification | `.sdd/roles.json`, `internal/scaffold/plan.go`, `internal/scaffold/scaffold_test.go`, `internal/upgrade/upgrade.go`, `internal/upgrade/upgrade_test.go`, `README.md`, `ROADMAP.md`, `TECH_STACK.md`, `tasks/features/T-041-execution-ops-skills/*` | `make verify && make check && ./bin/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `.sdd/roles.json` | modify | Add implementer, qa, and devops roles and phase-b/operations mappings |
| `internal/cli/cli.go` | modify | Document sdd prompt implement in usageText |
| `internal/cli/commands.go` | modify | Dispatch sdd prompt implement subcommand in runPrompt |
| `internal/cli/prompt.go` | modify | Implement sdd prompt implement with --role flag and conditional plan detection |
| `internal/cli/skill_test.go` | modify | Unit tests for sdd prompt implement and exporting new roles |
| `internal/check/roles.go` | modify | Validate Implementer, QA, and DevOps phase mappings in sdd check |
| `internal/check/roles_test.go` | modify | Unit tests for extended phase mappings validation |
| `internal/role/role.go` | modify | Add PhaseMapping fields, 9 canonical role names, and extended default config |
| `internal/role/role_test.go` | modify | Unit tests for extended role mappings and canonical names |
| `internal/scaffold/plan.go` | modify | Seed extended .sdd/roles.json on init |
| `internal/scaffold/scaffold_test.go` | modify | Tests for extended .sdd/roles.json scaffolding |
| `internal/templates/data/skills/implementer/SKILL.md` | create | Canonical Implementer agentskills.io skill |
| `internal/templates/data/skills/qa/SKILL.md` | create | Canonical QA automation agentskills.io skill |
| `internal/templates/data/skills/devops/SKILL.md` | create | Canonical DevOps / SRE agentskills.io skill |
| `internal/templates/templates_test.go` | modify | Tests for 9 embedded role skills |
| `internal/upgrade/upgrade.go` | modify | Upgrade/seed extended .sdd/roles.json on upgrade |
| `internal/upgrade/upgrade_test.go` | modify | Tests for extended .sdd/roles.json upgrading |
| `README.md` | modify | Document sdd prompt implement and new execution/ops role skills |
| `ROADMAP.md` | modify | Track milestone and task lifecycle status |
| `TECH_STACK.md` | modify | Record Decision D-037 and verified facts |
| `tasks/features/T-040-governance-role-skills/plan.md` | modify | Align status to approved for completed task |
| `tasks/features/T-041-execution-ops-skills/plan.md` | modify | Architecture plan and implementation steps |
| `tasks/features/T-041-execution-ops-skills/requirements.md` | modify | Specify requirements R-001 to R-009 |
| `tasks/features/T-041-execution-ops-skills/review.md` | modify | Review Triad evaluation and sign-off |
| `tasks/features/T-041-execution-ops-skills/validation.md` | modify | Traceability matrix and verification evidence |

## 6. Dependencies
None. Pure Go standard library (`os`, `path/filepath`, `encoding/json`, `errors`, `fmt`, `strings`, `sort`) per Decision D-007.

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Update T-041 status across lifecycle (`Specifying` -> `Implementing` -> `Awaiting review` -> `Done`) |
| `TECH_STACK.md` | Record and approve Decision D-037; record verified facts |
| `README.md` | Add documentation for `sdd prompt implement` and execution/ops role skills |

## 8. Risks, mitigations, rollback
- **Risk:** Agents overstepping boundaries during Phase B implementation.
  - *Mitigation:* The `implementer` skill explicitly enforces that modifying any file outside `plan.md` Section 5 is a hard violation requiring an immediate halt.
- **Risk:** QA or DevOps skills being confused with core gating requirements.
  - *Mitigation:* QA and DevOps skills are explicitly documented as conditional operational roles activated only when a plan specifies them, preserving existing Gate 1 and Gate 2 rules.
- **Rollback:** Revert CLI prompt addition and restore previous 6-role manifest in `.sdd/roles.json`.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- **Rule F-001 (Resolve all review findings: Blockers, Warnings, and Minors):** Worker agent must resolve all findings reported across the 3 subagent review lenses before presenting to human at GATE 2.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-15   Phrase received: `APPROVED T-041`
