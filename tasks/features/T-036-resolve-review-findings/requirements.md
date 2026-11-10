# Requirements — T-036 Binding Human Feedback Rule F-001 (Resolve all review findings)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-10)
Roadmap ref: `ROADMAP.md` T-036

## 1. Goal
Codify human feedback as the first binding rule `F-001` in `SKILLS.md` §8, enforce it in agent contracts and prompt templates, update canonical skills, and mechanically verify in `sdd check` that all review findings across all severity levels (Blockers, Warnings, and Minors) must be resolved (or explicitly justified) before GATE 2 human review.

## 2. Context and references
- Mission / principles touched:
  - Principle 1 (Deterministic): Deterministic verification of finding resolution in `review.md`.
  - Principle 2 (Fail safe): Unresolved review findings halt progression to human review unless explicitly justified.
  - Principle 3 (Human in the loop): Human feedback rules (`F-xxx`) capture general human expectations and bind all future tasks.
- Decisions this relies on:
  - D-002: Spec-driven workflow with human gates.
  - D-007: Standard library only (no external dependencies).
  - D-026: Code review specification, rubric, and task artifact (`review.md`).
  - D-031: Pre-GATE 1 Plan Review Subagent and 3-Subagent Parallel Review Triad.
  - D-032: Human Feedback Rule F-001 (Resolve all review findings: Blockers, Warnings, and Minors).
- Depends on tasks: T-035 (Done).

## 3. Scope
**In scope**
- Human Feedback Register in `SKILLS.md` §8:
  - Register `F-001` as active binding rule requiring resolution of all review findings (Blockers, Warnings, Minors) before GATE 2.
- Agent Contracts & Templates:
  - Update `internal/templates/data/base/contract.md.tmpl` Phase C.2, Phase D, Definition of Done, and Hard Rule 12.
  - Synchronize `AGENTS.md`, `CLAUDE.md`, and `GEMINI.md` in lockstep.
  - Bump `templates.Version` to `1.6.0` per Hard Rule 16.
- Mechanical Verification (`internal/check/review.go`):
  - Enhance `checkReviewDocument` to check all rows in the findings table:
    - Unresolved 🔴 Blocker -> Error.
    - Unresolved 🟡 Warning or 🟢 Minor -> Warning (flagged under rule `review-findings / F-001`).
    - Statuses `Resolved`, `Justified`, `Waived`, `Accepted` are recognized as compliant.
- Prompt Generation (`internal/cli/prompt.go`):
  - Surface `F-001` in prompt handover context and lens instructions, emphasizing that subagents must catalog all levels and workers must resolve all levels.
- Canonical Skills (`skills/`):
  - Update `skills/code-review/SKILL.md` and `skills/task-coordinator/SKILL.md` reflecting Rule F-001.

**Out of scope**
- Document lifecycle YAML frontmatter metadata and `.ragignore` (scheduled in T-037).
- Native self-update CLI binary command (scheduled in T-038).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL register `F-001` in `SKILLS.md` §8 as an active binding human feedback rule mandating that all review findings (Blockers, Warnings, and Minors) must be resolved or explicitly justified before GATE 2 presentation. | MUST | inspection |
| R-002 | Agent contracts (`contract.md.tmpl`, `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`) SHALL be updated in lockstep to encode Rule F-001 into Phase C.2 and the Definition of Done (§8). | MUST | test / inspection |
| R-003 | `templates.Version` SHALL be bumped to `1.6.0` in `internal/templates/templates.go`, verified by unit tests and fresh kit regeneration per Hard Rule 16. | MUST | test |
| R-004 | `internal/check/review.go` SHALL mechanically verify all findings in `review.md` for tasks in `Awaiting review` or `Done`, emitting an Error for unresolved Blockers and a Warning for unresolved Warnings or Minors under Rule F-001. | MUST | test |
| R-005 | `sdd check` SHALL accept `Resolved`, `Justified`, `Waived`, and `Accepted` as valid resolved statuses in the `review.md` findings register. | MUST | test |
| R-006 | `sdd prompt review` and `sdd prompt plan-review` SHALL dynamically include `F-001` in prompt synthesis and active feedback rules. | MUST | test |
| R-007 | Canonical skills (`skills/code-review/SKILL.md`, `skills/task-coordinator/SKILL.md`) SHALL document Rule F-001 finding resolution standards. | MUST | inspection |
| R-008 | The entire test suite (`make verify`, `make check`) and mechanical validation (`sdd check --strict`) SHALL pass with 0 errors and 0 warnings. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Finding parsing and prompt generation are deterministic and order-independent.
- **Fail safe:** Any unresolved blocker or unverified warning in strict mode fails the check.
- **Purity:** Standard library only; no external dependencies (D-007).

## 6. Constraints and assumptions
- Review tables follow standard format: `| ID | Category | Severity | Finding | Suggested Action | Status |`.
- Status column values are parsed case-insensitively.

## 7. Open questions
(None — settled by approved Decision D-032).

## 8. Acceptance criteria
- `SKILLS.md` §8 registers `F-001` with complete metadata.
- `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, and `contract.md.tmpl` reflect `templates=1.6.0` with identical rule sets.
- `sdd check` detects unresolved Warnings and Minors in `review.md` as warnings, failing under `--strict`.
- `sdd check` passes when all findings in `review.md` are marked `Resolved` or `Justified`.
- `make check` and `sdd check --strict` pass with 0 errors and 0 warnings.
