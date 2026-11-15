---
status: completed
task_id: T-041
last_reviewed: 2026-11-15
---
# Requirements — T-041 Role-based agent skills: implementer, QA, and DevOps operations

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Completed
Roadmap ref: `ROADMAP.md` T-041

## 1. Goal
Implement canonical `agentskills.io` role skills for execution and operations (`implementer`, `qa`, `devops`), extend `.sdd/roles.json` with Phase B and operations phase mappings, introduce `sdd prompt implement` for turnkey execution handover, support conditional plan activation, and provide mechanical validation in `sdd check`.

## 2. Context and references
- Mission / principles touched:
  - Principle 1 (Deterministic): Deterministic phase-to-role mappings, explicit boundaries, and reproducible agent handover prompts.
  - Principle 2 (Fail safe): Read-only boundary enforcement on review/QA roles, strict YAML frontmatter validation, and schema checking.
  - Principle 3 (Human in the loop): All gates remain human-held; execution and operational roles operate strictly under Gate 1 approval and Gate 2 review.
  - Principle 5 (Never overwrite human work): Two-tier storage where project `skills/` overrides infrastructure `.sdd/skills/` without clobbering customizations.
- Decisions this relies on:
  - D-002: Spec-driven workflow with human gates.
  - D-007: Go standard library only, single static binary.
  - D-013: Template overrides and manifest.
  - D-024: Open Agent Skills standards adoption (`agentskills.io`).
  - D-026: Code review specification, rubric, and task artifact.
  - D-029: Two-tier git governance and Task Coordinator orchestration.
  - D-031: Pre-GATE 1 Plan Review Subagent and 3-Subagent Parallel Review Triad.
  - D-032: Human Feedback Rule F-001 (all review findings resolved before GATE 2).
  - D-036: Role skill taxonomy, phase-to-role mapping (`.sdd/roles.json`), and two-tier storage architecture.
  - D-037: Execution and operations role skills taxonomy, conditional plan activation, and roles manifest extension.
- Depends on tasks: T-040 (Done).

## 3. Scope
**In scope**
- Canonical `agentskills.io` skill authoring for 3 execution and operations roles:
  - `implementer`: Senior software engineer executing `plan.md` step-by-step; strictly modifies ONLY planned files in Section 5; table-driven tests; standard library purity (`D-007`); determinism (injected clocks, seeded IDs); zero scope creep.
  - `qa`: Senior QA automation engineer; deep testing; integration tests; negative path and boundary fuzzing; race detection (`-race`); determinism verification (`-count=2`); benchmark and regression checks. Activated when a plan names QA.
  - `devops`: Senior DevOps / SRE engineer; build automation; CI/CD recipes (GitHub Actions, Makefile targets); cross-compilation; distribution packaging; release tagging and self-update validation. Activated when a plan names DevOps.
- Embedded templates for all 3 skills in `internal/templates/data/skills/<role>/SKILL.md`.
- Phase-to-role mapping manifest extension in `.sdd/roles.json`:
  - Map `phase-b` to `implementer`.
  - Map `operations` to `qa` and `devops`.
- Go representations in `internal/role`:
  - Extend `PhaseMapping` with `Implementer`, `QA`, and `DevOps`.
  - Extend `CanonicalRoleNames()` to include `implementer`, `qa`, and `devops` (9 total).
- CLI command `sdd prompt implement <T-xxx> [--role <implementer|qa|devops>] [--json]`:
  - Turnkey handover prompt for Phase B implementation.
  - Injects `implementer` role skill, task requirements, approved plan, and strict file boundaries.
  - Supports `--role qa` and `--role devops` for specialized test hardening or release operations.
  - Supports conditional plan activation: detects if `plan.md` names QA or DevOps in its steps and enriches prompt accordingly.
- CLI skill export:
  - `sdd skill export --roles` exports all 9 canonical role skills.
  - `sdd skill export --role <name>` exports any named canonical role skill.
- Mechanical consistency check in `sdd check`:
  - Validate that extended `.sdd/roles.json` schema and role references resolve cleanly and pass `agentskills.io` validation.
- Scaffolding & Upgrades:
  - `internal/scaffold/plan.go` and `internal/upgrade/upgrade.go` seed updated default `.sdd/roles.json`.

**Out of scope**
- Remote CI agent runners or cloud deployments (standard library purity `D-007`; local CI recipes only).
- Modifying git contracts (no agent contract changes; QA and DevOps operate under existing human-held gates).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL author canonical `agentskills.io` compliant skills with valid YAML frontmatter for 3 execution and operations roles (`implementer`, `qa`, `devops`). | MUST | test / inspection |
| R-002 | The system SHALL embed all 3 execution and operations role skills in the binary under `internal/templates/data/skills/<role>/SKILL.md`. | MUST | test |
| R-003 | The system SHALL extend `internal/role/role.go` and `.sdd/roles.json` to map `phase-b` to `implementer` and `operations` to `qa` and `devops`. | MUST | test |
| R-004 | The system SHALL extend `CanonicalRoleNames()` in `internal/role/role.go` to include `implementer`, `qa`, and `devops` in sorted canonical order. | MUST | test |
| R-005 | The system SHALL update `sdd init` and `sdd upgrade` to seed and update `.sdd/roles.json` with the extended roles and phase mappings without overwriting customizations. | MUST | test |
| R-006 | The system SHALL validate in `sdd check` that extended `.sdd/roles.json` conforms to schema and all declared role skills exist and pass `agentskills.io` validation. | MUST | test |
| R-007 | The system SHALL support `sdd skill export --roles` to export all 9 canonical role skills and `sdd skill export --role <name>` for `implementer`, `qa`, and `devops`. | MUST | test |
| R-008 | The system SHALL implement `sdd prompt implement <T-xxx> [--role <implementer\|qa\|devops>] [--json]` to generate turnkey Phase B execution handover prompts enriched with role skills and plan boundaries. | MUST | test |
| R-009 | The system SHALL detect if `plan.md` conditionally names QA or DevOps in its steps and automatically include the respective role skill references in the implementation prompt. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Role skill loading, prompt generation, and phase mapping resolution are byte-reproducible across runs.
- **Fail-safe behaviour:** Unknown role names, invalid schemas, or missing files produce descriptive error diagnostics.
- **Standard library purity:** Zero external dependencies (pure Go stdlib per D-007).
- **Security & Confinement:** Path traversal rejection enforced on all role resolution paths; atomic writes via `fsutil.WriteAtomic`.

## 6. Constraints and assumptions
- All skills comply with the Linux Foundation / Anthropic `agentskills.io` open standard.
- Implementer operates strictly inside its isolated git worktree branch and modifies ONLY planned files.
- QA and DevOps operate under the existing SDD contract without changing core gates.

## 7. Open questions
(None. Section is empty for GATE 1.)

## 8. Acceptance criteria
- **AC-1 (Role Skills Standards Compliance):** Given the 3 embedded execution/ops role skills, all 3 parse cleanly with valid frontmatter (`name` matching directory, `description` non-empty and under 1024 characters).
- **AC-2 (Extended Phase Mapping Validation):** Given `.sdd/roles.json` referencing `implementer`, `qa`, and `devops`, `sdd check` verifies all roles exist and conform to standards.
- **AC-3 (Turnkey Implementation Prompt):** Given `sdd prompt implement T-xxx`, the generated prompt includes the `implementer` role skill instructions, Section 5 planned files boundary, and requirements.
- **AC-4 (Specialized Ops Prompting):** Given `sdd prompt implement T-xxx --role qa` or `--role devops`, the prompt enriches context with the specialized QA or DevOps role skill.
- **AC-5 (Conditional Plan Activation):** Given a `plan.md` mentioning QA or DevOps in its steps, `sdd prompt implement T-xxx` automatically detects and notes the activated operational skills.
- **AC-6 (Skill Export):** Given `sdd skill export --roles` or `sdd skill export --role implementer`, skills are cleanly exported to `skills/<role>/SKILL.md`.
