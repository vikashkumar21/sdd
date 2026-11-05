# Requirements — T-027 sdd status: summarise roadmap, task progress, and gate status

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-05)
Roadmap ref: `ROADMAP.md` T-027

## 1. Goal
Provide a fast, read-only `sdd status` CLI command (with optional `--json` output) that scans `ROADMAP.md`, `TECH_STACK.md`, and `features/` to present an at-a-glance summary of active tasks, roadmap milestone progress, pending human gates, and risk warnings.

## 2. Context and references
- Mission / principles touched: Principles 1 (Deterministic), 3 (Human in the loop), 4 (Small and shippable)
- Decisions this relies on:
  - D-002: Spec-driven workflow with human gates (GATE 1, GATE 2)
  - D-007: Go, standard library only, single static binary
  - D-026: Code review specification, 3-tier risk grading, and `review.md`
- Depends on tasks: T-026 (Done)

## 3. Scope
**In scope**
- `sdd status` CLI command with human-readable terminal output.
- `--dir <path>` flag to inspect arbitrary project root (defaults to `.`).
- `--json` flag to emit machine-readable structured JSON for agent orchestration and CI/CD.
- Identification of active tasks (tasks not `Done` and not `Not started`) and their corresponding SDD phase.
- Identification of pending gates (`GATE 1` for `Specifying`/`Awaiting plan approval`, `GATE 2` for `Awaiting review`) with exact reply guidance.
- Roadmap milestone progress breakdown (total tasks, done tasks, completion percentage).
- Aggregate task metrics (total, done, in progress, blocked, not started).
- Detection and surfacing of warnings: multiple active tasks, unresolved 🔴 review blockers in active feature directories, and past-due verified facts.
- Help text updates in `cli.go` and usage documentation in `README.md`.

**Out of scope**
- Interactive prompts or file mutations (status is strictly read-only).
- Calling external services or network calls.
- Modifying task status (status changes remain driven by `sdd feature new`, plan approval, and human commits).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL provide a CLI command `sdd status` that reads project roadmap and feature state and prints a human-readable summary. | MUST | test / inspection |
| R-002 | The system SHALL support `--dir <path>` to target a project root; if `ROADMAP.md` is missing, it SHALL report an error and exit with code 1. | MUST | test |
| R-003 | The system SHALL identify active tasks (status neither `Done` nor `Not started`), detailing ID, title, milestone, current status, SDD phase, and feature folder path. | MUST | test |
| R-004 | The system SHALL identify pending gates for active tasks: `GATE 1` (Plan approval) when in `Specifying` or `Awaiting plan approval` with reply `APPROVED T-xxx`; `GATE 2` (Review & commit) when in `Awaiting review` with action `COMMITTED T-xxx`; and `None` when in `Implementing`. If no task is active, it SHALL suggest picking up the next `Not started` task via `sdd feature new`. | MUST | test |
| R-005 | The system SHALL parse all milestones from `ROADMAP.md`, computing task counts (total, done) and progress status (`Done`, `In progress`, `Not started`). | MUST | test |
| R-006 | The system SHALL compute an aggregate task summary across the entire roadmap: Total, Done, In Progress, Blocked, and Not Started. | MUST | test |
| R-007 | The system SHALL support `--json` to output a structured JSON payload (`ProjectStatus`) with 2-space indentation. | MUST | test |
| R-008 | The system SHALL surface warnings if: (a) more than one task is active simultaneously; (b) an active task has unresolved 🔴 Blockers in `review.md`; or (c) verified facts in `TECH_STACK.md` are past due for re-verification. | MUST | test |
| R-009 | The command SHALL be strictly read-only, deterministic (independent of clock or map iteration order), and exit 0 on success, 1 on missing/malformed files, and 2 on invalid flags. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Milestones and tasks are processed in canonical roadmap file order. Map keys are sorted. Output is identical across repeated runs on unchanged repositories.
- **Fail-safe behaviour:** Missing or unreadable `ROADMAP.md` exits 1 with an actionable error. Malformed rows are skipped or handled gracefully without panic.
- **Standard library only:** Implemented exclusively with standard Go packages (`fmt`, `io`, `os`, `path/filepath`, `strings`, `regexp`, `encoding/json`, `flag`).
- **Performance budget:** Must execute in under 50 milliseconds for typical repositories.
- **Zero side-effects:** Does not write, modify, or delete any files.

## 6. Constraints and assumptions
- Follows the existing `internal/roadmap` parsing conventions.
- Relies on standard SDD status strings (`Not started`, `Specifying`, `Awaiting plan approval`, `Implementing`, `Awaiting review`, `Done`, `Blocked`).

## 7. Open questions
(None - all requirements settled)

## 8. Acceptance criteria
- Given a valid SDD project with `T-027` in `Specifying`, when running `sdd status`, then it outputs `T-027` as active with pending `GATE 1`, along with milestone counts and summary metrics.
- Given `--json`, when running `sdd status --json`, then it outputs valid JSON matching the `ProjectStatus` schema with `valid: true`.
- Given a project where `ROADMAP.md` does not exist, when running `sdd status`, then it prints an error to stderr and exits with code 1.
- Given an active task with an unresolved 🔴 Blocker in `review.md`, when running `sdd status`, then the blocker is surfaced under warnings in both text and JSON outputs.
