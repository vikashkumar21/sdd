---
status: draft
task_id: T-040
last_reviewed: 2026-11-14
---
# Requirements — T-040 Role-based agent skills: core governance, planning, and review triad

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Draft
Roadmap ref: `ROADMAP.md` T-040

## 1. Goal
Implement canonical role-based `agentskills.io` skills for core governance, planning, and review roles (`coordinator`, `architect`, `plan-reviewer`, `spec-auditor`, `appsec-specialist`, `pattern-auditor`), introduce `.sdd/roles.json` phase mapping, provide mechanical validation in `sdd check`, and integrate with `sdd prompt` and `sdd skill export`.

## 2. Context and references
- Mission / principles touched:
  - Principle 1 (Deterministic): Deterministic phase-to-role mappings, explicit boundaries, and reproducible agent handover prompts.
  - Principle 2 (Fail safe): Read-only boundary enforcement on review roles, strict YAML frontmatter validation, and schema checking.
  - Principle 3 (Human in the loop): All gates remain human-held; role skills formalize auditor and planning responsibilities without compromising human approvals.
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
- Depends on tasks: T-033 (Done), T-035 (Done), T-039 (Done).

## 3. Scope
**In scope**
- Canonical `agentskills.io` skill authoring for 6 governance and review roles:
  - `coordinator`: multi-agent workspace governance, worktree provisioning, deconfliction, merge qualification, and Gate 2 presentation.
  - `architect`: Phase A requirements synthesis, architectural decision formulation (`D-xxx`), boundary setting, and risk modeling.
  - `plan-reviewer`: Pre-GATE 1 independent plan audit, rubric checking, boundary precision, open questions check.
  - `spec-auditor`: Phase C.2 Review Triad (traceability of R-xxx, negative coverage, plan boundaries, stdlib purity).
  - `appsec-specialist`: Phase C.2 Review Triad (threat surface, bounded inputs, filesystem confinement, secrets hygiene, command safety).
  - `pattern-auditor`: Phase C.2 Review Triad (decision conformance, determinism, language idioms, Rule F-001).
- Embedded templates for all 6 skills in `internal/templates/data/skills/<role>/SKILL.md`.
- Phase-to-role mapping manifest `.sdd/roles.json` and Go representation in `internal/role`.
- Two-tier skill resolution: project root `skills/<role>/SKILL.md` overrides `.sdd/skills/<role>/SKILL.md`, falling back to embedded templates.
- Mechanical consistency check in `sdd check`:
  - Validate `.sdd/roles.json` schema and role references.
  - Validate that each role skill conforms to `agentskills.io` standards (regex naming, description length, progressive disclosure sections).
- CLI enhancements:
  - `sdd skill export --roles` exports all canonical governance role skills.
  - `sdd skill export --role <name>` exports a specific role skill.
  - `sdd prompt plan-review` automatically enriches prompt with `plan-reviewer` skill instructions.
  - `sdd prompt review --lens <lens>` automatically enriches prompt with the respective review lens skill (`spec-auditor`, `appsec-specialist`, or `pattern-auditor`).
  - `sdd init` and `sdd upgrade` seed/update `.sdd/roles.json`.

**Out of scope**
- Phase B execution and operational roles (`implementer`, `qa`, `devops`), which are sequenced for Part 2 (T-041).
- External network-based skill registries or remote downloading (D-007 stdlib offline purity).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL author canonical `agentskills.io` compliant skills with valid YAML frontmatter for 6 governance roles (`coordinator`, `architect`, `plan-reviewer`, `spec-auditor`, `appsec-specialist`, `pattern-auditor`). | MUST | test / inspection |
| R-002 | The system SHALL embed all 6 governance role skills in the binary under `internal/templates/data/skills/<role>/SKILL.md`. | MUST | test |
| R-003 | The system SHALL define a deterministic JSON schema for `.sdd/roles.json` mapping workflow phases (Phase A, Phase C.2, Phase D) to role identifiers. | MUST | test |
| R-004 | The system SHALL generate a baseline `.sdd/roles.json` during `sdd init` and `sdd upgrade` without overwriting existing customizations. | MUST | test |
| R-005 | The system SHALL resolve role skills using the two-tier hierarchy: project root `skills/<role>/SKILL.md` overrides `.sdd/skills/<role>/SKILL.md`, which falls back to embedded role templates. | MUST | test |
| R-006 | The system SHALL validate in `sdd check` that `.sdd/roles.json` conforms to schema and all declared role skills exist and pass `agentskills.io` validation. | MUST | test |
| R-007 | The system SHALL support `sdd skill export --roles` to export all canonical governance role skills to `skills/<role>/SKILL.md`. | MUST | test |
| R-008 | The system SHALL support `sdd skill export --role <name>` to export a specific canonical role skill. | MUST | test |
| R-009 | The system SHALL enrich `sdd prompt plan-review` by dynamically loading and referencing the `plan-reviewer` role skill. | MUST | test |
| R-010 | The system SHALL enrich `sdd prompt review --lens <lens>` by dynamically loading and referencing the respective review lens role skill (`spec-auditor`, `appsec-specialist`, or `pattern-auditor`). | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Role skill loading, phase mapping resolution, and prompt compilation are byte-reproducible across runs.
- **Fail-safe behaviour:** Missing or corrupt role skills declared in `.sdd/roles.json` produce clear error diagnostics in `sdd check`.
- **Standard library purity:** Zero external dependencies (pure Go stdlib per D-007).
- **Progressive disclosure:** Each role skill provides an overview and triggers section for compact prompting (~50 tokens), followed by detailed step-by-step instructions.

## 6. Constraints and assumptions
- All skills comply with the Linux Foundation / Anthropic `agentskills.io` open standard.
- Auditor roles are strictly read-only; no code editing permissions.

## 7. Open questions
(None. Section is empty for GATE 1.)

## 8. Acceptance criteria
- **AC-1 (Role Skills Standards Compliance):** Given the 6 embedded governance role skills, when validated with `internal/skill` parser, then all 6 parse cleanly with valid frontmatter (`name` matching directory, `description` non-empty and under 1024 characters).
- **AC-2 (Phase-to-Role Mapping Validation):** Given `.sdd/roles.json` referencing roles, when `sdd check` is run, then it verifies all referenced roles exist in project skills, `.sdd/skills/`, or embedded templates.
- **AC-3 (Skill Export):** Given `sdd skill export --roles` or `sdd skill export --role appsec-specialist`, the specified skills are cleanly written to `skills/<role>/SKILL.md` with standard permissions.
- **AC-4 (Prompt Enrichment):** Given `sdd prompt plan-review T-xxx` or `sdd prompt review T-xxx --lens <lens>`, the generated prompt references and includes the role skill instructions and boundaries.
- **AC-5 (Two-Tier Override Priority):** Given a customized skill in `skills/spec-auditor/SKILL.md`, `sdd prompt review --lens spec` reflects the project override rather than the embedded baseline.
