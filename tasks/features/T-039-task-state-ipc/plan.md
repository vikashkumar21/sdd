---
status: draft
task_id: T-039
last_reviewed: 2026-11-14
---
# Plan — T-039 Deterministic task state persistence and event journaling (sdd task state)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-14)

## 1. Approach summary
Implement a dedicated `internal/state` package providing crash-resilient, POSIX atomic state snapshot storage (`state.json`) and append-only lifecycle event journaling (`events.jsonl`) under `.sdd/state/<TASK_ID>/`. Wire the `sdd task state` CLI command group (`show`, `save`, `append-event`, `purge`) and `--purge-state` flag on `sdd task clean` into `internal/cli`, supporting structured JSON IPC output schemas and 100% standard library implementation.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-035 | Task state persistence, event journaling, and IPC architecture (`sdd task state`) | Option A (Native stdlib state store in `.sdd/state/<TASK_ID>/`, atomic `state.json` snapshot via temp file & rename, append-only `events.jsonl` with 1MB buffered scanner, task ID regex validation, worktree cross-check, and `--json` output) / Option B (External database SQLite/embedded key-value engine) | Option A: Native stdlib architecture preserves D-007 standard library purity, guarantees zero external dependencies, provides native POSIX atomic filesystem semantics, and seamlessly integrates with git worktrees and agent IPC. | Approved 2026-11-14 |

## 3. Design

### Data Structures & Types (`internal/state/state.go`)
```go
package state

type TaskState map[string]any // or structured task state representation with raw JSON preservation

type Event map[string]any // chronological event record with timestamp and kind

type CombinedState struct {
    TaskID         string         `json:"task_id"`
    Found          bool           `json:"found"`
    State          map[string]any `json:"state,omitempty"`
    Events         []any          `json:"events,omitempty"`
    WorktreeActive bool           `json:"worktree_active"`
    WorktreePath   string         `json:"worktree_path,omitempty"`
}

type SaveResult struct {
    TaskID    string `json:"task_id"`
    Saved     bool   `json:"saved"`
    Path      string `json:"path"`
    UpdatedAt string `json:"updated_at"`
}

type AppendResult struct {
    TaskID     string `json:"task_id"`
    Appended   bool   `json:"appended"`
    EventCount int    `json:"event_count"`
}

type PurgeResult struct {
    TaskID  string `json:"task_id"`
    Purged  bool   `json:"purged"`
    Path    string `json:"path"`
}
```

### Store Interface & Operations (`internal/state/state.go`)
- `ValidateTaskID(id string) (string, error)`: Validates against `^[A-Za-z]+-\d+$`, canonicalizes to uppercase, rejects traversal tokens (`..`, `/`, `\`).
- `StateDir(root, taskID string) string`: Returns filepath for `.sdd/state/<TASK_ID>`.
- `Save(root, taskID string, payload []byte, now time.Time) (SaveResult, error)`: Validates JSON, writes to `state.json.tmp.<pid>.<nanos>`, flushes, calls `file.Sync()`, and executes `os.Rename` to `state.json`.
- `AppendEvent(root, taskID string, payload []byte) (AppendResult, error)`: Validates JSON, opens `events.jsonl` with `O_CREATE|O_WRONLY|O_APPEND`, writes single trailing `\n`, calls `file.Sync()`.
- `Show(root, taskID string, activeWorktrees map[string]string) (CombinedState, error)`: Reads `state.json` and scans `events.jsonl` using a 1MB buffered `bufio.Scanner`, checks active worktrees, returns `CombinedState`.
- `Purge(root, taskID string) (PurgeResult, error)`: Removes directory via `os.RemoveAll`.

### CLI Command Group Wiring (`internal/cli/task_state.go`)
- Subcommands under `sdd task state`:
  - `sdd task state show <task-id> [--json]`: On missing state, CLI mode outputs error and exits with code 1; `--json` exits with code 0 and emits `{"task_id":"...","found":false}` for clean IPC piping.
  - `sdd task state save <task-id> [--state <json-string> | --stdin] [--json]`: Bounded reader (max 5MB) on `--stdin` preventing memory exhaustion.
  - `sdd task state append-event <task-id> [--event <json-string> | --stdin] [--json]`: Bounded reader (max 1MB) on `--stdin` preventing memory exhaustion.
  - `sdd task state purge <task-id> [--force] [--json]`
- Hook into `sdd task clean <task-id> --purge-state` in `internal/cli/task.go`.

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement `internal/state/state.go` with task ID validation, atomic snapshot save, append-only event logging, 1MB buffered scanner, and state purge | `internal/state/state.go`, `internal/state/state_test.go` | `go test -v ./internal/state` |
| 2 | Implement CLI commands in `internal/cli/task_state.go` and wire into `internal/cli/task.go` | `internal/cli/task_state.go`, `internal/cli/task.go`, `internal/cli/task_state_test.go` | `go test -v ./internal/cli -run TestTaskState` |
| 3 | Add `--purge-state` flag to `sdd task clean` in `internal/cli/task.go` and unit test | `internal/cli/task.go`, `internal/cli/task_test.go` | `go test -v ./internal/cli -run TestTaskClean_PurgeState` |
| 4 | Update living docs (`README.md`, `ROADMAP.md`, `TECH_STACK.md`), run verification and linter | `README.md`, `ROADMAP.md`, `TECH_STACK.md`, `tasks/features/T-039-task-state-ipc/*` | `make verify && make check && ./bin/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/state/state.go` | create | Core task state persistence, event journaling, and retrieval engine |
| `internal/state/state_test.go` | create | Unit tests for atomic saves, append events, scanner bounds, and directory traversal |
| `internal/cli/task_state.go` | create | CLI handlers for `sdd task state` subcommands |
| `internal/cli/task_state_test.go` | create | CLI tests for `show`, `save`, `append-event`, `purge`, `--stdin`, `--json`, and `clean --purge-state` |
| `internal/cli/task.go` | modify | Dispatch `state` subcommand and add `--purge-state` to `sdd task clean` |
| `README.md` | modify | Document `sdd task state` commands and `--purge-state` flag |
| `ROADMAP.md` | modify | Track milestone and task lifecycle status |
| `TECH_STACK.md` | modify | Record Decision D-035 and verified facts |
| `tasks/features/T-039-task-state-ipc/requirements.md` | modify | Specify requirements R-001 to R-010 |
| `tasks/features/T-039-task-state-ipc/plan.md` | modify | Architecture plan and implementation steps |
| `tasks/features/T-039-task-state-ipc/validation.md` | modify | Traceability matrix and verification evidence |
| `tasks/features/T-039-task-state-ipc/review.md` | modify | Review Triad evaluation and sign-off |

## 6. Dependencies
None. Pure Go standard library (`os`, `path/filepath`, `encoding/json`, `bufio`, `errors`, `fmt`, `regexp`, `strings`, `time`) per Decision D-007.

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Update T-039 status across lifecycle (`Specifying` -> `Implementing` -> `Awaiting review` -> `Done`) |
| `TECH_STACK.md` | Approve Decision D-035 upon human approval and record state store architecture |
| `README.md` | Add CLI usage examples for `sdd task state` and `sdd task clean --purge-state` |

## 8. Risks, mitigations, rollback
- **Risk:** Race conditions during concurrent `save` or `append-event` calls by multiple agents.
  - *Mitigation:* `save` uses unique temp files (`state.json.tmp.<pid>.<nanos>`) and atomic `os.Rename`. `append-event` uses `os.O_APPEND` which guarantees atomic POSIX appends on compliant filesystems.
- **Risk:** Malicious task IDs attempting directory traversal (e.g. `../../etc`).
  - *Mitigation:* Strict regex validation (`^[A-Za-z]+-\d+$`) rejects any ID containing separators or dots before filesystem access.
- **Risk:** Corrupted JSON payloads breaking downstream agent consumers.
  - *Mitigation:* Payloads are parsed with `json.Valid` or `json.Unmarshal` prior to writing or appending to disk.
- **Rollback:** Delete `internal/state/`, revert CLI wiring in `internal/cli/task.go`.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- **Rule F-001 (Resolve all review findings: Blockers, Warnings, and Minors):** Worker agent must resolve all findings reported across the 3 subagent review lenses before presenting to human at GATE 2.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-14   Phrase received: `APPROVED T-039`
