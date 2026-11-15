---
status: active
task_id: T-044
last_reviewed: 2026-11-15
---
# Requirements — T-044 Canonical protocol skills synchronization and taxonomy alignment

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-044

## 1. Goal
Synchronize canonical protocol skills (`sdd-protocol`, `code-review`, and `task-coordinator` / `coordinator`) across project-level `skills/`, embedded `internal/templates/data/skills/`, and Go export strings in `internal/skill/export.go` to eliminate legacy path references (`features/`), incorporate the Pre-GATE 1 Plan Review Subagent, 3-Subagent Parallel Review Triad, and Rule F-001, and establish clean role-skill aliasing and frontmatter alignment between `coordinator` and `task-coordinator`.

## 2. Context and references
- Mission / principles touched:
  - Principle 1 (Deterministic): Standardized protocols ensure identical agent workflows across IDEs, Agent SDKs, and CLI execution.
  - Principle 2 (Fail safe): Stale paths or contradictory review rules in protocol skills mislead AI agents into violating gates or touching invalid paths.
  - Principle 5 (Never overwrite human work): Exports preserve existing files or back them up atomically.
- Decisions this relies on:
  - D-002: Spec-driven workflow with human gates.
  - D-007: Standard library only.
  - D-024: Linux Foundation / Anthropic `agentskills.io` standard adoption (`skills/<name>/SKILL.md`).
  - D-026: Code review specification, rubric, and task artifact (`review.md`).
  - D-029: Two-tier git governance and Task Coordinator orchestration (`skills/task-coordinator/SKILL.md`).
  - D-030: Unified `tasks/` directory hierarchy (`tasks/features/`, `tasks/bugfixes/`, `tasks/hotfixes/`, `tasks/spikes/`).
  - D-031: Pre-GATE 1 Plan Review Subagent and 3-Subagent Parallel Review Triad.
  - D-032: Binding Human Feedback Rule F-001 (Resolve all review findings).
  - D-035: Task state persistence and event journaling (`sdd task state`).
  - D-036: Role skill taxonomy and phase-to-role mapping (`.sdd/roles.json`).
- Depends on tasks: T-043 (Done).

## 3. Scope
**In scope**
- Canonical `sdd-protocol` skill synchronization:
  - Update `skills/sdd-protocol/SKILL.md` and `CanonicalSDDProtocolSkill` in `internal/skill/export.go`.
  - Replace all legacy `features/T-xxx-<slug>/` references with `tasks/<typology>/T-xxx-<slug>/`.
  - Include `review.md` in task documentation quartet (`requirements.md`, `plan.md`, `validation.md`, `review.md`).
  - Document Pre-GATE 1 Plan Review Subagent in Phase A and 3-Subagent Parallel Review Triad in Phase C.2.
  - Reference Rule F-001 resolution mandate.
- Canonical `code-review` skill synchronization:
  - Update `CanonicalCodeReviewSkill` in `internal/skill/export.go` to match `skills/code-review/SKILL.md`.
  - Update 3-tier risk rubric to enforce Rule F-001: 🟡 Warnings and 🟢 Minors MUST be resolved or explicitly justified before GATE 2 (replacing legacy "address or explicitly accept").
  - Update paths to `tasks/<typology>/T-xxx-<slug>/review.md`.
- Canonical `coordinator` / `task-coordinator` taxonomy alignment:
  - Synchronize `skills/task-coordinator/SKILL.md`, `internal/templates/data/skills/coordinator/SKILL.md`, `internal/templates/data/skills/task-coordinator/SKILL.md`, and `CanonicalTaskCoordinatorSkill` in `internal/skill/export.go`.
  - Synthesize the comprehensive governance workflows (Pre-GATE 1 Plan Review, Review Triad, Rule F-001, and PR size guardrails) with the state store persistence (`sdd task state`) and deconfliction protocols.
  - In `internal/role/role.go`, allow `ResolveSkill` to resolve `"task-coordinator"` as an alias to `"coordinator"` (and vice versa) with frontmatter `name:` dynamically matching the requested identifier to pass `agentskills.io` directory validation.
- Verification and test suite updates:
  - Update tests in `internal/skill/export_test.go`, `internal/cli/skill_test.go`, `internal/role/role_test.go`, and `internal/templates/templates_test.go`.

**Out of scope**
- Modifying role schema or phase mapping in `.sdd/roles.json`.
- Modifying other role skills (`architect`, `codebase-analyst`, `implementer`, `qa`, etc.).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL update `skills/sdd-protocol/SKILL.md` and `CanonicalSDDProtocolSkill` in `internal/skill/export.go` to use unified `tasks/<typology>/` paths, include `review.md`, and document Pre-GATE 1 Plan Review and Phase C.2 Review Triad workflows. | MUST | test / inspection |
| R-002 | The system SHALL synchronize `CanonicalCodeReviewSkill` in `internal/skill/export.go` with `skills/code-review/SKILL.md` enforcing Rule F-001 finding resolution (🟡 Warnings and 🟢 Minors must be resolved or justified) and `tasks/<typology>/` paths. | MUST | test / inspection |
| R-003 | The system SHALL align `task-coordinator` and `coordinator` skill definitions across templates, project skills, and export constants, and ensure `role.ResolveSkill` resolves both `"coordinator"` and `"task-coordinator"` with matching frontmatter names. | MUST | test |
| R-004 | The system SHALL verify that `sdd skill export` (default protocol skills) and `sdd skill export --roles` export synchronized, valid `agentskills.io` skills that pass `sdd check --strict`. | MUST | test / check |
| R-005 | The system SHALL verify all skill exports and content invariants in automated unit tests across `internal/skill`, `internal/role`, `internal/templates`, and `internal/cli`. | MUST | test |

## 5. Non-functional requirements
- **Standard Library Purity:** Pure Go standard library per Decision D-007.
- **Determinism:** Exported skill content and frontmatter are byte-deterministic across identical runs.
- **Fail Safe:** Any unresolved skill name or invalid frontmatter continues to fail with exit code 1 and descriptive diagnostics.

## 6. Constraints and assumptions
- All skills strictly conform to the Linux Foundation / Anthropic `agentskills.io` specification (`name`, `description` <= 1024 chars, valid YAML frontmatter).
- Backward compatibility: existing calls to `sdd skill export --skill task-coordinator` and `sdd skill export --role coordinator` both succeed.

## 7. Open questions
(None. Section is empty for GATE 1.)

## 8. Acceptance criteria
- **AC-1 (`sdd-protocol` Modernization):** `skills/sdd-protocol/SKILL.md` and `CanonicalSDDProtocolSkill` contain zero references to bare `features/T-xxx` and explicitly list `review.md`, Plan Review Subagent, and Review Triad.
- **AC-2 (`code-review` Rule F-001 Alignment):** `CanonicalCodeReviewSkill` in `internal/skill/export.go` mandates resolving 🟡 Warnings and 🟢 Minors per Rule F-001, matching `skills/code-review/SKILL.md`.
- **AC-3 (`coordinator` Aliasing & Frontmatter):** `role.ResolveSkill(root, "task-coordinator", nil)` returns content with `name: task-coordinator`, and `role.ResolveSkill(root, "coordinator", nil)` returns content with `name: coordinator`, both without error and satisfying directory validation.
- **AC-4 (Mechanical Verification):** All unit tests in `internal/skill`, `internal/role`, `internal/templates`, and `internal/cli` pass; `make check` and `./bin/sdd check --strict` pass with 0 errors and 0 warnings.
