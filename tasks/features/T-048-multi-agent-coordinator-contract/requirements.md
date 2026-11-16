---
status: active
task_id: T-048
last_reviewed: 2026-11-16
---
# Requirements — T-048 Multi-agent coordinator contract and handover prompt template alignment

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved 2026-11-16
Roadmap ref: `ROADMAP.md` T-048

## 1. Goal
Align the embedded contract template (`base/contract.md.tmpl`), agent contract files (`AGENTS.md`, `CLAUDE.md`, `GEMINI.md`), and handover prompt template (`base/prompt.md.tmpl`) with the full autonomous multi-agent coordinator operating model, bump `templates.Version` to `2.0.0`, and bump CLI version to `1.7.0` so that downstream projects automatically detect and upgrade contract files via `sdd upgrade`.

## 2. Context and references
- Mission / principles touched: Principles 1 (Deterministic), 3 (Human-in-the-loop), 5 (Never overwrite human work).
- Decisions this relies on: `D-029` (Two-tier git governance), `D-031` (Review triad), `D-036` (Role skill taxonomy & `.sdd/roles.json`), `D-037` (Execution/operations roles), `D-039` (Codebase analyst role), `D-041` (Multi-agent coordinator contract and handover alignment).
- Depends on tasks: `T-044` (Canonical protocol skills sync), `T-045` (Turnkey prompt discovery).

## 3. Scope
**In scope**
- Update Section 2 ("The workflow") and Section 4 ("Hard rules") of `internal/templates/data/base/contract.md.tmpl` to explicitly designate the session agent as the Task Coordinator, dispatching canonical roles from `.sdd/roles.json` across all lifecycle phases (Architect, Plan Reviewer, Implementer, QA, DevOps, Spec Auditor, AppSec Specialist, Pattern Auditor).
- Update `internal/templates/data/base/prompt.md.tmpl` to instruct the incoming handover agent that it acts as the Task Coordinator, dispatching canonical roles mapped in `.sdd/roles.json` and replacing legacy `features/` references with `tasks/<typology>/`.
- Synchronize `AGENTS.md`, `CLAUDE.md`, and `GEMINI.md` in lockstep with the updated contract text and marker `<!-- sdd:begin contract templates=2.0.0 -->`.
- Bump `templates.Version` to `2.0.0` in `internal/templates/templates.go` and update template tests.
- Bump CLI release version to `1.7.0` in `internal/version/version.go`, `Makefile`, and `README.md`.
- Update `.sdd/manifest.json` template version to `2.0.0`.
- Verify `sdd upgrade` detects contract changes in older kits (`templates=1.9.0`) and proposes updating them to `2.0.0`.
- Verify `sdd check --strict` passes with 0 errors and 0 warnings.
- Verify Hard Rule 16 by generating a fresh kit in a temporary directory with `sdd init` and verifying with `sdd check`.

**Out of scope**
- Modifying the CLI flags or schema of `sdd upgrade`.
- Changing external third-party dependencies (`D-007` standard library only).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL update `internal/templates/data/base/contract.md.tmpl` Section 2 ("The workflow") table to explicitly designate the primary session agent as the Task Coordinator, orchestrating specialized subagent roles across Phase A (`architect`, `plan-reviewer`), Phase B (`implementer`, optional `devops`), Phase C.1 (`qa`), Phase C.2 (`spec-auditor`, `appsec-specialist`, `pattern-auditor`), and Phase D (merge qualification), generalizing path references to `tasks/<typology>/` and `sdd task new --type <typology>`. | MUST | test / inspection |
| R-002 | The system SHALL update `internal/templates/data/base/contract.md.tmpl` Section 4 Rule 13 to mandate that the primary session agent operates as the Task Coordinator dispatching canonical roles from `.sdd/roles.json`, verified by a new unit test `TestContractTemplateCoordinatorAndWorkflow` in `internal/templates/templates_test.go`. | MUST | test / inspection |
| R-003 | The system SHALL update `internal/templates/data/base/prompt.md.tmpl` (`sdd prompt <agent>`) to brief incoming agents that they operate as the Task Coordinator, dispatching the canonical roles mapped in `.sdd/roles.json`, and updating path references from `features/` to `tasks/<typology>/`, verified by unit tests in `internal/cli/cli_test.go`. | MUST | test / inspection |
| R-004 | The system SHALL bump `templates.Version` to `2.0.0` in `internal/templates/templates.go`, update template tests in `internal/templates/templates_test.go`, and update `.sdd/manifest.json`. | MUST | test |
| R-005 | The system SHALL synchronize `AGENTS.md`, `CLAUDE.md`, and `GEMINI.md` in lockstep with the updated contract text and header marker `<!-- sdd:begin contract templates=2.0.0 -->` per Contract Rule 9 and Rule 16. | MUST | test |
| R-006 | The system SHALL verify in `internal/upgrade/upgrade_test.go` that running `sdd upgrade` in an existing project with `templates=1.9.0` contracts detects the contract diff and updates them to `2.0.0`. | MUST | test |
| R-007 | The system SHALL verify that regenerating a fresh kit in a temporary directory via `sdd init` produces `templates=2.0.0` contracts and passes `sdd check --strict` per Contract Rule 16. | MUST | test |
| R-008 | The system SHALL bump CLI version to `1.7.0` in `internal/version/version.go`, `Makefile`, and `README.md`. | MUST | test / inspection |

## 5. Non-functional requirements
- **Determinism:** Embedded templates and generated handover prompts must produce byte-identical output for identical inputs and timestamps.
- **Fail-safe:** No destructive overwrites during `sdd upgrade`; human edits outside managed blocks are preserved.
- **Standard Library Purity (`D-007`):** Zero external dependencies.

## 6. Constraints and assumptions
- Agent files (`AGENTS.md`, `CLAUDE.md`, `GEMINI.md`) must maintain 100% parity inside the managed block markers (`<!-- sdd:begin contract ... -->` ... `<!-- sdd:end contract -->`).
- Bumping template version to `2.0.0` reflects the major shift from single-agent workflow instructions to full multi-agent coordinator orchestration.

## 7. Open questions
(None. Settled in Decision D-041.)

## 8. Acceptance criteria
- **AC-1 (Contract Multi-Agent Workflow):** Given `base/contract.md.tmpl`, Section 2 explicitly assigns Coordinator and Subagent actions across Phase A (`architect`, `plan-reviewer`), Phase B (`implementer`), Phase C.1 (`qa`), Phase C.2 (`spec-auditor`, `appsec-specialist`, `pattern-auditor`), and Phase D.
- **AC-2 (Handover Prompt Briefing):** Given `sdd prompt <agent>`, the generated handover prompt explicitly designates the incoming agent as Task Coordinator and briefs it on dispatching `.sdd/roles.json` roles.
- **AC-3 (Template Version Bump & Contract Parity):** `templates.Version` is `2.0.0`. `AGENTS.md`, `CLAUDE.md`, and `GEMINI.md` contain `templates=2.0.0` and pass `sdd check --strict` parity checks.
- **AC-4 (Upgrade Detection):** Running `sdd upgrade` on a project with `templates=1.9.0` replaces or updates the managed contract block with the `2.0.0` coordinator workflow.
- **AC-5 (Rule 16 Kit Proof):** Regenerating a kit in a temporary directory with `sdd init` passes `sdd check --strict` with 0 errors.
- **AC-6 (CLI Version):** `sdd version` reports `sdd 1.7.0 (templates 2.0.0)`.
