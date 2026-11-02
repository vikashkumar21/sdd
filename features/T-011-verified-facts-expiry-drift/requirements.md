# Requirements — T-011 P-08: Verified-facts expiry and roadmap drift checker

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-02)
Roadmap ref: `ROADMAP.md` T-011

## 1. Goal
Add automated consistency checks in `sdd check` that warn on past-due verified facts in `TECH_STACK.md` and detect drift between `ROADMAP.md` task states, prerequisite dependencies, and feature documents.

## 2. Context and references
- Mission / principles touched: Principles 1 (Deterministic), 2 (Fail safe), Scope §4 (`sdd check` consistency rules)
- Decisions this relies on: `D-003` (Go), `D-004` (Toolchain), `D-007` (Standard library only)
- Depends on tasks: `T-002` (Done), `T-008` (Robust parser, Done), `T-009` (JSON output, Done)

## 3. Scope
**In scope**
- Parsing the Verified facts log table in `TECH_STACK.md` to extract each fact's `Re-verify at` milestone or task reference.
- Emitting a warning when a fact's `Re-verify at` task or milestone is already completed (`Done`) in `ROADMAP.md`.
- Parsing `Depends on` prerequisite references in `ROADMAP.md` and warning if an active or completed task has incomplete prerequisites (`Not started`, `Specifying`, or `Blocked`).
- Checking feature document status headers (`validation.md`, `plan.md`) against `ROADMAP.md` to warn on status drift.
- Integrating findings into both plain text and JSON (`--json`) output modes.
- Unit tests covering active facts, expired facts, dependency drift, and document status drift.

**Out of scope**
- Automatically re-verifying external facts (human or agent verification responsibility).
- Altering existing roadmap task statuses automatically (human-driven gates).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL parse the Verified facts log table in TECH_STACK.md and extract the Fact summary and Re-verify at targets. | MUST | test |
| R-002 | The system SHALL emit a warning in sdd check for any verified fact whose Re-verify at task ID is marked Done in ROADMAP.md. | MUST | test |
| R-003 | The system SHALL emit a warning in sdd check for any verified fact whose Re-verify at milestone (e.g. M1) has all tasks marked Done in ROADMAP.md. | MUST | test |
| R-004 | The system SHALL emit a warning in sdd check when a task is Implementing, Awaiting review, or Done, but a prerequisite task in its Depends on column is incomplete. | MUST | test |
| R-005 | The system SHALL emit a warning in sdd check when a task is marked Done in ROADMAP.md, but its feature folder validation.md or plan.md status indicates incomplete state. | MUST | test |
| R-006 | The system SHALL include all verified-facts expiry and roadmap drift warnings in sdd check --json findings. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Finding messages, line numbers, and order SHALL be strictly deterministic across repeated runs.
- **Fail-safe behaviour:** Tables without verified facts, unrecognised non-task markers (e.g. `n/a`, `permanent`, `-`), or missing columns SHALL be handled gracefully without crashing.
- **Performance budget:** Fact table parsing and roadmap dependency checking SHALL execute in memory within milliseconds.
- **Dependencies:** Standard library only (`D-007`).

## 6. Constraints and assumptions
- Warnings notify the team of decaying facts or out-of-order execution without blocking normal non-strict check runs; `--strict` elevates warnings to exit 1.

## 7. Open questions
(None)

## 8. Acceptance criteria
- Given a `TECH_STACK.md` containing a fact with `Re-verify at: T-001`, when `T-001` is `Done` in `ROADMAP.md`, then `sdd check` emits a warning identifying the expired fact.
- Given a task `T-005` marked `Done` in `ROADMAP.md` that depends on `T-004` which is `Not started`, when `sdd check` runs, then a warning identifies the dependency prerequisite drift.
- Given a task `T-002` marked `Done` in `ROADMAP.md` but `features/T-002-.../validation.md` has `Status: Not run`, when `sdd check` runs, then a warning identifies the feature status drift.
