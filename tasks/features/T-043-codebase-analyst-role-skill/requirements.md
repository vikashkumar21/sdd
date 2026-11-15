---
status: active
task_id: T-043
last_reviewed: 2026-11-15
---
# Requirements — T-043 Role-based agent skills: codebase-analyst and discovery phase mapping

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-043

## 1. Goal
Provide a specialized canonical `codebase-analyst` role skill (`agentskills.io` standard) for forensic brownfield discovery (topology, UI/design systems/fonts/CSS, backend architectures, database schemas, and existing pitfalls), register `phase-discovery` in `.sdd/roles.json`, extend the canonical role taxonomy to 10 roles, and update mechanical validation and export CLI tools.

## 2. Context and references
- Mission / principles touched:
  - Principle 1 (Deterministic): Standardized forensic discovery procedures produce repeatable, objective specifications of existing codebases.
  - Principle 2 (Fail safe): Unresolved phase mappings or invalid role skills fail with clear diagnostics in `sdd check`.
  - Principle 3 (Human in the loop): Discovery outputs (living documents) are strictly submitted to the human for Gate 1 approval before any changes.
- Decisions this relies on:
  - D-002: Spec-driven workflow with human gates.
  - D-007: Go standard library only, single static binary.
  - D-024: Linux Foundation / Anthropic `agentskills.io` standard adoption (`skills/<name>/SKILL.md`).
  - D-036: Role skill taxonomy, phase-to-role mapping (`.sdd/roles.json`), and two-tier storage architecture.
  - D-037: Execution and operations role skills taxonomy and roles manifest extension.
  - D-039: Role-based agent skill: codebase-analyst and discovery phase mapping.
- Depends on tasks: T-040, T-041 (Done).

## 3. Scope
**In scope**
- Canonical role skill authoring:
  - `internal/templates/data/skills/codebase-analyst/SKILL.md` embedded template.
  - Frontmatter conforming to `agentskills.io` (`name: codebase-analyst`, `description: ...`).
  - Multi-tier progressive disclosure: High-Level Topology, Frontend/UI & Design System Extraction (CSS framework, design tokens, typography/fonts, component composition, state management), Backend & Data Modeling, Quality & Testing Conventions, and Pitfalls/Tech Debt Extraction (for `SKILLS.md` §7).
- Role configuration and taxonomy:
  - Add `"codebase-analyst"` to `CanonicalRoleNames()` in `internal/role/role.go` (alphabetical sort, now 10 canonical roles).
  - Add `Analyst string `json:"analyst,omitempty"`` to `PhaseMapping` in `internal/role/role.go`.
  - Add `"phase-discovery": { Analyst: "codebase-analyst", Planner: "architect" }` to `DefaultRoleConfig()` in `internal/role/role.go`.
- Scaffolding & upgrade configuration:
  - Update `role.DefaultRoleConfig()` in `internal/role/role.go` and this repository's `.sdd/roles.json` to include `"codebase-analyst"` role and `"phase-discovery"` phase mapping.
- Mechanical validation in `sdd check`:
  - Verify that `phase-discovery` mapping in `.sdd/roles.json` is validated in `checkRoles()` in `internal/check/roles.go`.
  - Validate that `codebase-analyst` resolves cleanly via the two-tier resolution engine.
- CLI Export:
  - Support `sdd skill export --role codebase-analyst`.
  - `sdd skill export --roles` exports all 10 canonical role skills to `skills/<role>/SKILL.md`.
- Scaffolding and upgrade preservation:
  - Verify that `sdd init` and `sdd upgrade` generate and preserve the extended `.sdd/roles.json`.

**Out of scope**
- Interactive prompt command `sdd prompt discovery` (scheduled for follow-up task T-044).
- Autonomous codebase rewriting or modifications (discovery is strictly read-only analysis).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL embed the canonical `codebase-analyst` role skill in `internal/templates/data/skills/codebase-analyst/SKILL.md` conforming to `agentskills.io` with YAML frontmatter, progressive disclosure, and UI/CSS/font/backend/pitfall extraction procedures. | MUST | test / inspection |
| R-002 | The system SHALL add `"codebase-analyst"` to `CanonicalRoleNames()` in `internal/role/role.go` in alphabetical sorted order, expanding the canonical set to 10 roles. | MUST | test |
| R-003 | The system SHALL extend `PhaseMapping` in `internal/role/role.go` with an `Analyst` field and add `"phase-discovery"` (`analyst: "codebase-analyst"`, `planner: "architect"`) to `DefaultRoleConfig()`. | MUST | test |
| R-004 | The system SHALL update `role.DefaultRoleConfig()` in `internal/role/role.go` and this repository's `.sdd/roles.json` to include `"codebase-analyst"` and `"phase-discovery"`. | MUST | inspection / check |
| R-005 | The system SHALL validate in `internal/check/roles.go` that `.sdd/roles.json` with `"phase-discovery"` mapping resolves `analyst` and `planner` roles to valid skills via the two-tier hierarchy. | MUST | test |
| R-006 | The system SHALL update `sdd skill export --roles` to export all 10 canonical roles, and `sdd skill export --role codebase-analyst` to export the `codebase-analyst` skill. | MUST | test / CLI |
| R-007 | The system SHALL verify that `sdd init` and `sdd upgrade` install and preserve `.sdd/roles.json` with `"phase-discovery"` and `"codebase-analyst"`. | MUST | test |

## 5. Non-functional requirements
- **Standard Library Purity:** Pure Go standard library per Decision D-007.
- **Determinism:** 100% deterministic role resolution and canonical list sorting.
- **Fail Safe:** Any unresolvable role or malformed frontmatter in `.sdd/roles.json` fails with line/key-precise errors in `sdd check`.

## 6. Constraints and assumptions
- Role identifiers are lower-kebab-case matching directory and frontmatter `name`.
- Backward compatibility: existing `.sdd/roles.json` files without `phase-discovery` continue to load cleanly.

## 7. Open questions
(None. Section is empty for GATE 1.)

## 8. Acceptance criteria
- **AC-1 (Canonical Skill):** `internal/templates/data/skills/codebase-analyst/SKILL.md` contains valid frontmatter (`name: codebase-analyst`) and explicit extraction steps for frontend CSS/tokens/fonts and backend architecture.
- **AC-2 (10 Canonical Roles):** `role.CanonicalRoleNames()` returns 10 sorted strings containing `codebase-analyst`.
- **AC-3 (Phase Discovery Mapping):** `role.DefaultRoleConfig().Phases["phase-discovery"]` maps `Analyst` to `"codebase-analyst"` and `Planner` to `"architect"`.
- **AC-4 (Mechanical Validation):** A `.sdd/roles.json` referencing a non-existent role in `phase-discovery` fails in `sdd check` with a descriptive error.
- **AC-5 (CLI Export):** `sdd skill export --roles` exports all 10 canonical skills; `sdd skill export --role codebase-analyst` exports only `skills/codebase-analyst/SKILL.md`.
- **AC-6 (Init & Upgrade):** `sdd init` and `sdd upgrade` render `.sdd/roles.json` with `phase-discovery`.
