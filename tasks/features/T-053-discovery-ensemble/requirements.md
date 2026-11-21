---
status: draft
---
# Requirements — T-053 Adversarial multi-agent brownfield discovery ensemble

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved 2026-11-21
Roadmap ref: `ROADMAP.md` T-053

## 1. Goal
Deliver an adversarial multi-agent brownfield discovery ensemble protocol featuring a canonical `discovery-challenger` role skill, `phase-discovery` mapping in `.sdd/roles.json`, stateless discovery bundle staging in `.sdd/review/<TASK_ID>/discovery/`, turnkey `sdd prompt discovery --ensemble` CLI flag with structured JSON context, and a 7-vector blind-spot reconciliation protocol to ensure brownfield codebase discovery never overlooks critical components, implicit flows, or architectural debt.

## 2. Context and references
- Mission / principles touched: `MISSION.md` §1 (One-liner), §4 (Scope: brownfield discovery), §5 (Deterministic, Fail safe, Small and shippable), §8 (H1)
- Decisions this relies on:
  - `D-007` (Go stdlib only, single static binary)
  - `D-024` (agentskills.io open standard specification)
  - `D-029` (Two-tier agent hierarchy and role dispatch)
  - `D-036` (Role-based governance skills and phase mapping)
  - `D-039` (Canonical codebase-analyst role skill and discovery phase)
  - `D-044` (Stateless reviewer context isolation)
  - `D-045` (Adversarial discovery ensemble protocol and challenger role)
- Depends on tasks: `T-043`, `T-044`, `T-045`, `T-052` (Done)

## 3. Scope
**In scope**
1. Canonical `discovery-challenger` role skill in `internal/templates/data/skills/discovery-challenger/SKILL.md` conforming to agentskills.io specification.
2. Role registry enhancements in `internal/role/role.go`:
   - Add `discovery-challenger` to `CanonicalRoleNames()`.
   - Add `discovery-challenger` role info to `DefaultRoleConfig()`.
   - Add `Challenger` field to `PhaseMapping` struct with JSON tag `challenger,omitempty`.
   - Map `phase-discovery.challenger = "discovery-challenger"` in default phase mappings.
3. Skill CLI and export alignment:
   - Update `internal/cli/skill.go` `--roles` help text to list `discovery-challenger`.
   - Update `internal/cli/skill_test.go` canonical role count assertion from 10 to 11.
   - Update `internal/templates/templates_test.go` embedded role skills test table.
   - Update repository `.sdd/roles.json` and `internal/scaffold/scaffold_test.go` to recognize `discovery-challenger`.
4. Stateless discovery review bundle staging in `internal/audit`:
   - Implement `StageDiscoveryBundle` in `internal/audit/bundle.go` staging `.sdd/review/<TASK_ID>/discovery/` containing `findings.md`, `rubric.md`, and `manifest.json`.
   - Unit tests in `internal/audit/bundle_test.go`.
5. Turnkey CLI support via `sdd prompt discovery [T-xxx] --ensemble`:
   - Add `--ensemble` flag to `sdd prompt discovery`.
   - Update usage and dispatch in `internal/cli/`.
   - Two-tier skill resolution for both `codebase-analyst` and `discovery-challenger`.
   - Automatic staging of stateless discovery bundle in `.sdd/review/<TASK_ID>/discovery/` when task ID is present.
   - Render multi-stage ensemble prompt with primary extraction brief, 7-vector challenger audit rubric, stateless bundle reference, and deterministic reconciliation protocol.
   - Structured JSON export (`--json`) populating `DiscoveryPromptContext` with `ensemble: true`, `challengerRoleSkill`, `bundleDir`, `blindSpotVectors`, and `reconciliationChecklist` (using `omitempty` for backward compatibility).
6. Role configuration validation and upgrade migration:
   - Update `internal/check` to validate `discovery-challenger` against agentskills.io rules and ensure `phase-discovery.challenger` points to a registered role when specified, while maintaining backward compatibility with legacy manifests.
   - Update `internal/upgrade` to non-destructively seed `discovery-challenger` in `Roles` and populate `phase-discovery.challenger` in existing `.sdd/roles.json` without clobbering custom roles or descriptions.
7. Comprehensive unit, integration, and golden CLI tests.

**Out of scope**
- Generating deep domain flow documentation and visual Mermaid diagrams (`T-054`).
- Direct network calls or LLM runtime invocations (Task Coordinator orchestrates subagents per `D-007` and `D-029`).
- Modifying standard single-analyst output when `--ensemble` is omitted (100% backward compatible).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL define a canonical `discovery-challenger` role skill in `internal/templates/data/skills/discovery-challenger/SKILL.md` complying with the agentskills.io YAML frontmatter specification. | MUST | test / inspection |
| R-002 | The system SHALL register `discovery-challenger` in `CanonicalRoleNames()`, include its description in `DefaultRoleConfig()`, add `Challenger` to `PhaseMapping`, and map `phase-discovery.challenger = "discovery-challenger"`. | MUST | test |
| R-003 | The system SHALL validate in `sdd check` that `discovery-challenger` (when present) satisfies agentskills.io schema rules and that `phase-discovery.challenger` (when non-empty) references a valid registered role, while remaining backward-compatible with pre-T-053 manifests. | MUST | test |
| R-004 | The system SHALL update `sdd upgrade` to non-destructively append `discovery-challenger` to `Roles` and populate `phase-discovery.challenger` in existing `.sdd/roles.json` files without overwriting user customizations. | MUST | test |
| R-005 | The system SHALL accept an `--ensemble` boolean flag in `sdd prompt discovery`, parsing cleanly alongside existing positional task arguments, `--task`, `--dir`, and `--json`. | MUST | test |
| R-006 | When `--ensemble` is active, `sdd prompt discovery` SHALL render a multi-stage handover prompt comprising: (1) Primary Analyst extraction brief, (2) Adversarial Challenger audit rubric evaluating 7 blind-spot vectors, and (3) a deterministic Reconciliation Protocol before Gate 1 approval. | MUST | test |
| R-007 | When both `--ensemble` and `--json` are active, `sdd prompt discovery` SHALL emit a structured JSON context bundle containing `ensemble: true`, both role skills, detected project metadata, the 7 blind-spot vectors, and the reconciliation checklist. | MUST | test |
| R-008 | When `--ensemble` is not passed, `sdd prompt discovery` SHALL preserve existing single-agent behavior and output format without regression, omitting ensemble-only fields in JSON. | MUST | test |
| R-009 | The system SHALL stage an isolated stateless discovery bundle in `.sdd/review/<TASK_ID>/discovery/` (containing `findings.md`, `rubric.md`, and bundle metadata `manifest.json`) via `audit.StageDiscoveryBundle` when `sdd prompt discovery --ensemble` runs for an identified task. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** All generated prompt text, JSON keys, vector lists, and checklists are sorted canonically and invariant across repeated runs.
- **Fail-safe:** Unknown or conflicting CLI arguments exit with code 1 or 2 and display clear usage instructions.
- **Zero Third-Party Dependencies:** Implemented strictly using Go standard library (`D-007`).
- **Path Hygiene:** No developer-specific local paths in templates, prompts, or test fixtures per Rule `F-002`.

## 6. Constraints and assumptions
- Role skills are managed via two-tier resolution: project-specific `.sdd/skills/<name>/SKILL.md` or embedded fallback `templates.Data`.
- Discovery challenger operates strictly in read-only audit mode on application code, modifying only discovery specifications or task review documents.

## 7. Open questions
(None. Section is strictly empty before GATE 1.)

## 8. Acceptance criteria
- Given a project with or without an active discovery task, when `sdd prompt discovery --ensemble` is run, then it outputs the two-agent ensemble brief with the 7 blind-spot vectors and reconciliation protocol.
- Given an identified discovery task, when `sdd prompt discovery <TASK_ID> --ensemble` is run, then it stages the stateless review bundle at `.sdd/review/<TASK_ID>/discovery/` with `findings.md`, `rubric.md`, and `manifest.json`.
- Given `sdd prompt discovery --ensemble --json`, when executed, then it outputs valid JSON containing both `roleSkill` and `challengerRoleSkill` along with `blindSpotVectors`, `bundleDir`, and `reconciliationChecklist`.
- Given `sdd check`, when `.sdd/roles.json` contains `discovery-challenger` and `phase-discovery.challenger`, then check passes with 0 errors.
- Given an existing project initialized prior to T-053, when `sdd upgrade` is run, then `.sdd/roles.json` is updated to include `discovery-challenger` without overwriting custom roles.
- Given `sdd prompt discovery` without `--ensemble`, when run, then the single-agent prompt text matches legacy behavior.
