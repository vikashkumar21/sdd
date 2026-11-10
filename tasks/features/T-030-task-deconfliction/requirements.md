# Requirements — T-030 Static task deconfliction and plan overlap analyzer

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-05)
Roadmap ref: `ROADMAP.md` T-030

## 1. Goal
Provide automated static file target analysis across task plans (`plan.md` Section 5) to detect file collision hazards, classify task pairs as concurrency-safe or sequential, generate parallel execution schedules, and surface overlap warnings in `sdd status` and `sdd check`.

## 2. Context and references
- Mission / principles touched: Principle 1 (Deterministic), Principle 4 (Small and shippable), Principle 5 (Never overwrite human work)
- Decisions this relies on:
  - D-002: Spec-driven workflow with human gates
  - D-007: Go, standard library only, single static binary
  - D-028: Git worktree concurrency and workspace isolation
  - D-029: Two-tier git governance and Task Coordinator orchestration
- Depends on tasks: T-029 (Done)

## 3. Scope
**In scope**
- Plan target extractor in `internal/deconflict/`:
  - Parses Section 5 ("Files to create or modify") in `plan.md` (or `patch.md` for hotfixes) across `features/`, `bugfixes/`, `hotfixes/`, `spikes/`.
  - Normalizes relative file paths (strips code fences, backticks, leading/trailing whitespace, forward-slash normalization).
- Static Overlap Analyzer:
  - Computes pairwise file set intersections across tasks.
  - Groups tasks into disjoint independent execution batches (safe to run concurrently in parallel worktrees) vs sequential conflict queues.
- CLI subcommand:
  - `sdd task deconflict [T-xxx ...]` (computes overlap for specified tasks, or all active in-flight tasks by default).
  - Flag `--json`: outputs machine-readable JSON structure for Task Coordinator agents and CI.
  - Flag `--all`: analyzes all tasks in `ROADMAP.md` that have task folders.
- Integration in `sdd status`:
  - Warns when multiple active tasks have overlapping planned files.
- Integration in `sdd check`:
  - Warns if multiple tasks simultaneously marked `Implementing` or `Awaiting review` share planned files.

**Out of scope**
- Pre-merge test qualification and automated git merge simulation (handled in T-031).
- Task Coordinator prompt and agent skill authoring (handled in T-032).
- Dynamic runtime file lock acquisition.

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL parse Section 5 ("Files to create or modify") of `plan.md` (or `patch.md`) across all 4 work typologies and extract planned file paths. | MUST | test |
| R-002 | The system SHALL provide `sdd task deconflict [T-xxx ...]` to analyze file overlaps across specified tasks or all active in-flight tasks. | MUST | test |
| R-003 | The system SHALL classify task pairs with disjoint planned file sets as concurrency-safe (`parallel`). | MUST | test |
| R-004 | The system SHALL classify task pairs with overlapping planned file sets as collision hazards (`sequential`), listing the exact conflicting file paths. | MUST | test |
| R-005 | The system SHALL schedule tasks into disjoint parallel execution batches and sequential queues. | MUST | test |
| R-006 | The system SHALL provide `sdd task deconflict --json` outputting structured JSON for Task Coordinator agents. | MUST | test |
| R-007 | `sdd status` SHALL detect if active in-progress tasks share planned target files and display an overlap warning. | MUST | test |
| R-008 | `sdd check` SHALL warn if multiple tasks marked `Implementing` or `Awaiting review` have overlapping planned target files. | MUST | test |
| R-009 | The system SHALL handle edge cases gracefully: missing plans, empty file tables, unparsed paths, and single-task queries. | MUST | test |
| R-010 | The system SHALL produce deterministic output regardless of map iteration order or filesystem directory order. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** File paths and task lists must be sorted canonically so comparisons and JSON reports are byte-identical across runs.
- **Fail-safe:** If a task folder or plan file is missing, the analyzer reports an actionable warning without crashing.
- **Standard library only:** All parsing and scheduling algorithms implemented with pure Go standard library (`D-007`).

## 6. Constraints and assumptions
- Plan files follow the standard SDD template table in Section 5 (`| Path | Action | Reason |`).

## 7. Open questions
(None - all decisions resolved in D-028 and D-029)

## 8. Acceptance criteria
- Given Task A planning `[internal/cli/cli.go, internal/cli/task.go]` and Task B planning `[internal/status/status.go]`, `sdd task deconflict TaskA TaskB` reports disjoint file sets and schedules them in Parallel Batch 1.
- Given Task A planning `[internal/cli/cli.go]` and Task C planning `[internal/cli/cli.go, README.md]`, `sdd task deconflict TaskA TaskC` reports a collision hazard on `internal/cli/cli.go` and schedules them sequentially.
- Given multiple active tasks with overlapping files, `sdd status` and `sdd check` emit an actionable warning.
