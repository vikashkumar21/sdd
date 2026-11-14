---
status: draft
task_id: T-039
last_reviewed: 2026-11-14
---
# Requirements — T-039 Deterministic task state persistence and event journaling (sdd task state)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-14)
Roadmap ref: `ROADMAP.md` T-039

## 1. Goal
Implement the `sdd task state` command group to provide deterministic, data-plane state persistence and audit logging for autonomous agents (e.g. `sdd-agent`). All commands must support structured JSON output via `--json` and POSIX atomic filesystem guarantees.

## 2. Context and references
- Mission / principles touched:
  - Principle 1 (Deterministic): Deterministic state snapshots, strictly ordered chronological event journals, and JSON IPC contracts.
  - Principle 2 (Fail safe): Path traversal rejection, JSON payload validation before write, and atomic temp-file rename preventing corruption.
  - Principle 3 (Human in the loop): Confirmation prompt on purge unless `--force` or `--json` is supplied.
  - Principle 5 (Never overwrite human work): Atomic swap and crash-safe fsync guarantees.
- Decisions this relies on:
  - D-002: Spec-driven workflow with human gates.
  - D-003: Go language.
  - D-007: Standard library only (no external dependencies, pure Go).
  - D-028: Git worktree isolation and concurrent workspace provisioning.
  - D-029: Two-tier git governance and Task Coordinator orchestration.
  - D-035: Task state persistence, event journaling, and IPC architecture (`sdd task state`).
- Depends on tasks: T-034 (Done), T-029 (Done).

## 3. Scope
**In scope**
- Core state store package `internal/state`:
  - Data directory layout: `<repo-root>/.sdd/state/<TASK_ID>/state.json` and `events.jsonl`.
  - Task ID validation regex (`^[A-Za-z]+-\d+$`) and path traversal confinement (rejection of `..`, `/`, `\`).
  - Atomic snapshot persistence via temporary file (`state.json.tmp.<pid>.<nanos>`), `file.Sync()`, and `os.Rename`.
  - Append-only event journaling with `os.O_CREATE | os.O_WRONLY | os.O_APPEND`, `\n` line termination, and `file.Sync()`.
  - Event scanning using `bufio.Scanner` bounded by a 1MB buffer with corrupt line detection.
  - State retrieval and git worktree cross-checking (`git worktree list`).
  - Safe task state purging via `os.RemoveAll`.
- CLI commands under `sdd task state`:
  - `sdd task state show <task-id> [--json]`
  - `sdd task state save <task-id> [--state <json-string> | --stdin] [--json]`
  - `sdd task state append-event <task-id> [--event <json-string> | --stdin] [--json]`
  - `sdd task state purge <task-id> [--force] [--json]`
- CLI integration:
  - `sdd task clean <task-id> --purge-state`
- Structured JSON output schemas matching the IPC contract.
- Pure Go standard library table-driven test suite.

**Out of scope**
- Network-based remote state replication or cloud databases.
- AI model calling or autonomous planning logic inside `sdd` (handled externally by `sdd-agent`).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL store task state in `<repo-root>/.sdd/state/<TASK_ID>/` containing `state.json` (state snapshot) and `events.jsonl` (chronological append-only event log). | MUST | test |
| R-002 | The system SHALL validate all task IDs against regex `^[A-Za-z]+-\d+$`, canonicalize to uppercase, and immediately reject any ID containing path separators (slash, backslash) or traversal tokens (".."). | MUST | test |
| R-003 | The system SHALL provide `sdd task state show <task-id> [--json]` to read `state.json` and `events.jsonl`, cross-check active git worktrees, and return structured JSON or human-readable table (when state is not found, human CLI exits with code 1 and error message, while `--json` exits with code 0 and emits `{"task_id":"...","found":false}`). | MUST | test |
| R-004 | The system SHALL provide `sdd task state save <task-id> [--state <json> or --stdin] [--json]` to atomically write `state.json` using a temporary file in the same directory (`state.json.tmp.<pid>.<nanos>`), flush, `file.Sync()`, and `os.Rename` (enforcing a 5MB bounded reader on `--stdin`). | MUST | test |
| R-005 | The system SHALL validate that state input to `save` is valid JSON prior to writing, rejecting empty or malformed inputs. | MUST | test |
| R-006 | The system SHALL provide `sdd task state append-event <task-id> [--event <json> or --stdin] [--json]` to append validated JSON event lines to `events.jsonl` with `os.O_CREATE or os.O_WRONLY or os.O_APPEND`, single trailing `\n`, and `file.Sync()` (enforcing a 1MB bounded reader on `--stdin`). | MUST | test |
| R-007 | The system SHALL provide `sdd task state purge <task-id> [--force] [--json]` to delete `.sdd/state/<task-id>/`, prompting for confirmation unless `--force` or `--json` is supplied. | MUST | test |
| R-008 | The system SHALL support `--purge-state` flag on `sdd task clean <task-id>` to remove both the git worktree and the task state store. | MUST | test |
| R-009 | The system SHALL read `events.jsonl` using `bufio.Scanner` with a 1MB buffer (`scanner.Buffer(make([]byte, 64*1024), 1024*1024)`), reporting errors on malformed lines. | MUST | test |
| R-010 | The system SHALL implement all functionality using the Go standard library only with zero external dependencies per Decision D-007. | MUST | test / inspection |

## 5. Non-functional requirements
- **Determinism:** Snapshots are written atomically; readers never observe partial writes; event journals strictly preserve append order.
- **Fail-safe behaviour:** Path traversal attempts fail immediately with descriptive errors; malformed JSON payloads are rejected before mutating disk state.
- **Crash resilience:** `file.Sync()` executed on every snapshot replacement and event append guarantees persistence across crashes.
- **Confinement:** State operations are strictly confined beneath `.sdd/state/<TASK_ID>/`.

## 6. Constraints and assumptions
- POSIX atomic rename semantics (`os.Rename`) on the local filesystem within the same directory.
- Standard library purity (D-007).

## 7. Open questions
(None. Section is empty for GATE 1.)

## 8. Acceptance criteria
- **AC-1 (State Snapshot Roundtrip):** Given a valid task ID and JSON state payload, when `sdd task state save <task-id> --state '<json>' --json` is executed, then `state.json` is atomically created and `sdd task state show <task-id> --json` returns the snapshot with `found: true`.
- **AC-2 (Append-Only Event Journal):** Given multiple events appended sequentially via `sdd task state append-event <task-id> --event '<json>'`, then `events.jsonl` contains each event in chronological order with single trailing newlines and `sdd task state show` returns the complete event list.
- **AC-3 (Worktree Cross-check):** Given an active git worktree for the task, when `sdd task state show <task-id> --json` is run, then `worktree_active: true` and `worktree_path` are correctly reflected in the JSON response.
- **AC-4 (Path Traversal Protection):** Given an invalid task ID containing `..` or path separators, when any `sdd task state` command is executed, then the command fails immediately with an error and exit code 1 without touching the filesystem.
- **AC-5 (State Purge & Task Clean):** Given a task with stored state and an active worktree, when `sdd task clean <task-id> --purge-state` is executed, both the worktree and `.sdd/state/<task-id>/` are cleanly removed.
