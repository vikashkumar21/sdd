# Plan — T-027 sdd status: summarise roadmap, task progress, and gate status

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-05)

## 1. Approach summary
Implement a dedicated `internal/status` package that reads and analyses `ROADMAP.md`, `TECH_STACK.md`, and `features/` folders to compute project state, milestone progress, active tasks, pending human gates, and risk warnings. Wire this package to the CLI command `sdd status` (supporting `--dir` and `--json`) and document it in `cli.go` and `README.md`.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| (none) | No new architectural decisions needed; uses stdlib (D-007) and existing workflow (D-002, D-026). | n/a | n/a | Approved |

## 3. Design
### Types in `internal/status/status.go`
```go
type MilestoneInfo struct {
    ID         string `json:"id"`
    Title      string `json:"title"`
    TasksTotal int    `json:"tasks_total"`
    TasksDone  int    `json:"tasks_done"`
    Percent    int    `json:"percent"`
    Status     string `json:"status"` // "Done", "In progress", "Not started"
}

type GateInfo struct {
    TaskID string `json:"task_id"`
    Gate   string `json:"gate"`   // "GATE 1", "GATE 2", or ""
    Name   string `json:"name"`   // "Approve requirements and plan", "Review and commit"
    Status string `json:"status"` // "Pending approval", "Awaiting review", "In progress"
    Action string `json:"action"` // e.g. "Reply APPROVED T-027"
}

type TaskInfo struct {
    ID             string    `json:"id"`
    Title          string    `json:"title"`
    Milestone      string    `json:"milestone"`
    Status         string    `json:"status"`
    Phase          string    `json:"phase"`
    Folder         string    `json:"folder,omitempty"`
    Gate           *GateInfo `json:"gate,omitempty"`
    ReviewBlockers int       `json:"review_blockers,omitempty"`
}

type TaskSummary struct {
    Total      int `json:"total"`
    Done       int `json:"done"`
    InProgress int `json:"in_progress"`
    Blocked    int `json:"blocked"`
    NotStarted int `json:"not_started"`
}

type ProjectStatus struct {
    Root            string          `json:"root"`
    ActiveMilestone *MilestoneInfo  `json:"active_milestone,omitempty"`
    Milestones      []MilestoneInfo `json:"milestones"`
    TaskSummary     TaskSummary     `json:"task_summary"`
    ActiveTasks     []TaskInfo      `json:"active_tasks"`
    NextTask        *TaskInfo       `json:"next_task,omitempty"`
    NextGate        *GateInfo       `json:"next_gate,omitempty"`
    Warnings        []string        `json:"warnings,omitempty"`
}
```

### Core Functions
- `Scan(root string) (*ProjectStatus, error)`: Parses `ROADMAP.md` and related context, resolving milestones, task counts, active tasks, pending gates, and warnings.
- `RenderText(ps *ProjectStatus) string`: Formats the status for clean, visual terminal display.
- `RenderJSON(ps *ProjectStatus) ([]byte, error)`: Formats the status as 2-space indented JSON.

### CLI Handler in `internal/cli/status.go`
- `runStatus(args []string, env IO) int`: Parses flags (`--dir`, `--json`), invokes `status.Scan`, and writes text or JSON to `env.Out`.

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement `internal/status` package with parser, status aggregator, warnings checker, and renderers. | `internal/status/status.go`, `internal/status/status_test.go` | `go test -v ./internal/status` |
| 2 | Implement CLI status command handler. | `internal/cli/status.go` | `go test -v ./internal/cli -run TestStatus` |
| 3 | Wire `sdd status` into CLI router, usage text, and add CLI tests. | `internal/cli/cli.go`, `internal/cli/cli_test.go` | `go test -v ./internal/cli` |
| 4 | Update user documentation and roadmap. | `README.md`, `ROADMAP.md` | `sdd check --strict` and `make verify` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/status/status.go` | create | Core status scanner, milestone analyzer, gate resolver, and renderers |
| `internal/status/status_test.go` | create | Unit tests for status scanning, JSON serialization, and gate detection |
| `internal/cli/status.go` | create | CLI command handler `runStatus` |
| `internal/cli/cli.go` | modify | Register `status` command and update usage help text |
| `internal/cli/cli_test.go` | modify | Tests for `sdd status` and `sdd status --json` |
| `README.md` | modify | Document `sdd status` and `sdd status --json` |
| `ROADMAP.md` | modify | Update T-027 status |

## 6. Dependencies
Standard library only (`fmt`, `io`, `os`, `path/filepath`, `strings`, `regexp`, `encoding/json`, `flag`). Complies with D-007.

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Update T-027 status and notes |
| `README.md` | Add `sdd status` CLI command documentation and examples |

## 8. Risks, mitigations, rollback
- **Risk:** Incomplete or unclosed markdown tables in malformed `ROADMAP.md`.
  - **Mitigation:** Use defensive parsing with nil/boundary checks; fall back to safe zero values.
- **Rollback:** Delete `internal/status/`, `internal/cli/status.go`, and revert edits to `cli.go` and `cli_test.go`.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- None active in §8 register.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-05   Phrase received: `APPROVED T-027`
