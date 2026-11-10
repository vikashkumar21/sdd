# Requirements — T-035 Pre-GATE 1 Plan Review Subagent and 3-Subagent Parallel Review Triad

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-10)
Roadmap ref: `ROADMAP.md` T-035

## 1. Goal
Codify and tool the Pre-GATE 1 Plan Review Subagent protocol in Phase A (subagent audits plan soundness before human presentation) and the 3-Subagent Parallel Review Triad in Phase C.2 (three parallel subagents each auditing through a single specialized lens: Spec Auditor, AppSec Specialist, Pattern & Quality Auditor), supported by `sdd prompt plan-review` CLI commands and contract updates.

## 2. Context and references
- Mission / principles touched:
  - Principle 1 (Deterministic): Deterministic review rubrics with explicit lens boundaries and machine-readable exports.
  - Principle 2 (Fail safe): 0 Blockers required across all three review lenses before human GATE 2 sign-off.
  - Principle 3 (Human in the loop): Plan review subagent audits plan *before* human GATE 1, giving the human higher-quality, pre-screened plans.
  - Principle 4 (Small and shippable): Clean modular CLI extensions and contract synchronization.
- Decisions this relies on:
  - D-002: Spec-driven workflow with human gates.
  - D-007: Standard library only (no external dependencies).
  - D-026: Code review specification, rubric, and task artifact (`review.md`).
  - D-029: Two-tier git governance and Task Coordinator orchestration.
  - D-030: Unified `tasks/` directory structure.
  - D-031: Pre-GATE 1 Plan Review Subagent and 3-Subagent Parallel Review Triad.
- Depends on tasks: T-034 (Done).

## 3. Scope
**In scope**
- Pre-GATE 1 Plan Review Subagent:
  - Define the Phase A plan review subagent rubric (requirements completeness, step sizing, file boundaries, risk mitigation, testability).
  - Add CLI command `sdd prompt plan-review <T-xxx>` to generate turnkey handover prompt for the plan reviewer subagent.
  - Support `--json` flag on `sdd prompt plan-review` for programmatic agent harnesses and Agent SDKs.
- 3-Subagent Parallel Review Triad in Phase C.2:
  - Formulate distinct parallel reviewer roles for the three lenses:
    1. 🔎 `Spec Auditor`: requirements traceability, negative cases, plan boundary strictness, zero added dependencies.
    2. 🛡️ `AppSec Specialist`: threat surface, input validation, filesystem confinement (`os.Root`), secrets hygiene, command injection safety.
    3. 📐 `Pattern & Quality Auditor`: architectural decisions (`D-xxx`), language idioms (Effective Go), determinism, active feedback rules (`F-xxx`).
  - Update `sdd prompt review <T-xxx> --lens <spec|security|pattern>` to generate targeted single-lens review handover prompts.
  - Update review aggregation protocol: all three parallel lenses record evaluation in `review.md`, and all three must report 0 Blockers before Phase D close.
- Mechanical Validation (`internal/check`):
  - Enhance `sdd check` to verify multi-lens review integrity in `review.md`.
- Agent Contract & Template Synchronization:
  - Update `internal/templates/data/base/contract.md.tmpl`, `AGENTS.md`, `CLAUDE.md`, `GEMINI.md` in lockstep.
  - Bump `templates.Version` to `1.5.0` per Hard Rule 16.
- Canonical Skills Update:
  - Update `skills/code-review/SKILL.md` and `skills/task-coordinator/SKILL.md` to document the 3-subagent parallel review triad orchestration and pre-GATE 1 plan review protocol.

**Out of scope**
- Human Feedback Rule F-001 binding enforcement (scheduled in T-036).
- Document lifecycle YAML frontmatter metadata and `.ragignore` (scheduled in T-037).
- Native self-update CLI binary command (scheduled in T-038).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL specify a Pre-GATE 1 Plan Review protocol in Phase A: after drafting task docs, an independent subagent reviews the plan for soundness, step sizing, file boundaries, risks, and testability before GATE 1 presentation to the human. | MUST | test / inspection |
| R-002 | `sdd prompt plan-review <T-xxx>` SHALL generate a turnkey handover prompt containing task requirements, plan, validation matrix, and plan review rubric. | MUST | test |
| R-003 | `sdd prompt plan-review <T-xxx> --json` SHALL export structured machine-readable JSON containing task metadata, doc contents, and rubric guidelines for programmatic Agent SDK dispatch. | MUST | test |
| R-004 | The system SHALL codify the 3-Subagent Parallel Review Triad in Phase C.2, where three separate subagents execute in parallel, each auditing through exactly one specialized lens: Spec Auditor, AppSec Specialist, and Pattern & Quality Auditor. | MUST | test / inspection |
| R-005 | `sdd prompt review <T-xxx> --lens <spec\|security\|pattern>` SHALL generate dedicated, focused prompts tailored specifically for each specialized lens subagent. | MUST | test |
| R-006 | The review coordinator / primary agent SHALL aggregate evaluations from all three parallel lens subagents in `review.md`, requiring 0 Blockers across all three lenses to qualify for Phase D close, and `sdd check` SHALL verify multi-lens section presence and 0 unresolved Blockers. | MUST | test |
| R-007 | The 3-loop circuit breaker SHALL govern both plan review (max 3 revisions before escalating to human) and code review (max 3 review-fix iterations). | MUST | inspection |
| R-008 | Agent contracts (`AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `contract.md.tmpl`) SHALL be updated in lockstep to mandate the Pre-GATE 1 Plan Review and the 3-Subagent Parallel Review Triad, bumping `templates.Version` to `1.5.0`. | MUST | test / inspection |
| R-009 | Canonical skills (`skills/code-review/SKILL.md` and `skills/task-coordinator/SKILL.md`) SHALL be updated to describe the parallel triad orchestration and pre-GATE 1 plan review workflow. | MUST | inspection |
| R-010 | The entire test suite (`make verify`, `make check`) and mechanical validation (`sdd check --strict`) SHALL pass with 0 errors and 0 warnings. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Handover prompt generation and JSON output must produce byte-identical content given identical inputs.
- **Independence:** Author agents cannot approve their own plans or code reviews.
- **Fail safe:** Any unresolved blocker in plan review or code review halts progression and escalates to the human.
- **Standard library purity:** Standard library only; no external dependencies (D-007).

## 6. Constraints and assumptions
- Subagents are invoked via harness tools (`invoke_subagent`, `Task` tool, or SDK dispatch).
- Lens reviewers operate concurrently; each inspects the same task git diff and documentation.

## 7. Open questions
(None — settled by approved Decision D-031)

## 8. Acceptance criteria
- `sdd prompt plan-review T-035` outputs a comprehensive, structured plan review handover prompt.
- `sdd prompt plan-review T-035 --json` outputs valid structured JSON.
- `sdd prompt review T-035 --lens spec`, `--lens security`, and `--lens pattern` generate distinct, focused prompts for each respective lens.
- `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, and `contract.md.tmpl` reflect `templates=1.5.0` with identical rule sets.
- `make check` and `sdd check --strict` pass with 0 errors and 0 warnings.
