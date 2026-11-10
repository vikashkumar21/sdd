# Plan — T-030 Static task deconfliction and plan overlap analyzer

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-05)

## 1. Approach summary
Implement a static plan deconfliction engine in `internal/deconflict/` that parses planned file sets from Section 5 of `plan.md` across tasks, calculates pairwise intersections, detects collision hazards, and schedules tasks into disjoint parallel execution batches vs sequential queues. Expose `sdd task deconflict` (with `--json` and `--all`), and surface collision warnings in `sdd status` and `sdd check`.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-028 | Git worktree concurrency and workspace isolation | Option A | Option A | Approved (2026-11-05) |
| D-029 | Two-tier git governance and Task Coordinator orchestration | Option A | Option A | Approved (2026-11-05) |

## 3. Design
### Deconfliction Architecture (`internal/deconflict/`)
1. **Target Extraction:**
   - Scans task folders across `features/`, `bugfixes/`, `hotfixes/`, `spikes/`.
   - Locates `plan.md` (or `patch.md` for hotfixes).
   - Locates Section 5 ("Files to create or modify") table.
   - Extracts the first column (file path), strips surrounding backticks, trims whitespace, and normalizes file path separators.
2. **Pairwise Overlap Analysis:**
   - For every pair of tasks `(TaskA, TaskB)`:
     - Intersection: `Files(TaskA) ∩ Files(TaskB)`.
     - If empty: task pair is disjoint (`SafePairs`).
     - If non-empty: task pair is in collision hazard (`Conflicts`), recording the shared file list.
3. **Concurrency Scheduling:**
   - Tasks are partitioned into parallel batches:
     - Batch 1: Maximal set of pairwise disjoint tasks that can start concurrently in isolated worktrees.
     - Subsequent Batches: Tasks deferred sequentially until earlier conflicting tasks complete.
4. **Data Structures:**
   ```go
   type Overlap struct {
       TaskA       string   `json:"task_a"`
       TaskB       string   `json:"task_b"`
       SharedFiles []string `json:"shared_files"`
   }

   type Report struct {
       Tasks       []TaskPlan `json:"tasks"`
       Conflicts   []Overlap  `json:"conflicts"`
       SafePairs   [][2]string `json:"safe_pairs"`
       Batches     [][]string `json:"batches"`
       HasConflict bool       `json:"has_conflict"`
   }
   ```

### CLI Command (`internal/cli/task.go`)
- `sdd task deconflict [T-xxx ...]`
  - If task arguments provided: analyzes specified tasks.
  - If no task arguments: analyzes all currently active in-flight tasks (`specifying`, `implementing`, `awaiting review`).
  - Flag `--all`: analyzes all tasks in `ROADMAP.md` that have task folders.
  - Flag `--json`: outputs machine-readable JSON structure.

### Status and Check Integration
- `sdd status`: when multiple active tasks have planned file overlaps, adds an actionable warning in the status report.
- `sdd check`: warns if multiple tasks currently marked `Implementing` or `Awaiting review` have overlapping planned files.

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Create `internal/deconflict` package with target extractor, overlap analyzer, scheduler, and unit tests. | `internal/deconflict/deconflict.go`, `internal/deconflict/deconflict_test.go` | `go test -v ./internal/deconflict` |
| 2 | Add `sdd task deconflict` CLI command in `internal/cli/task.go` and `internal/cli/cli.go`. | `internal/cli/task.go`, `internal/cli/cli.go`, `internal/cli/cli_test.go` | `go test -v ./internal/cli -run TestTaskDeconflict` |
| 3 | Integrate file overlap warnings into `internal/status`. | `internal/status/status.go`, `internal/status/status_test.go` | `go test -v ./internal/status -run TestStatusTaskOverlap` |
| 4 | Add task file overlap consistency check in `internal/check`. | `internal/check/check.go`, `internal/check/check_test.go` | `go test -v ./internal/check -run TestCheckTaskOverlap` |
| 5 | Update `README.md`, `ROADMAP.md`, run full verification (`make verify`, `make check`, `sdd check --strict`). | `README.md`, `ROADMAP.md` | `make check`, `sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/deconflict/deconflict.go` | create | Plan target parser, overlap analysis, and batch scheduler |
| `internal/deconflict/deconflict_test.go` | create | Unit tests for target extraction, intersection, and scheduling |
| `internal/cli/task.go` | modify | Add `task deconflict` command handler |
| `internal/cli/cli.go` | modify | Update command usage text |
| `internal/cli/cli_test.go` | modify | Test `sdd task deconflict` CLI flags and output |
| `internal/status/status.go` | modify | Detect active task overlap hazards in status |
| `internal/status/status_test.go` | modify | Test status warning for overlapping tasks |
| `internal/check/check.go` | modify | Add `task-overlap` check rule |
| `internal/check/check_test.go` | modify | Test check rule for concurrent task collisions |
| `README.md` | modify | Document `sdd task deconflict` |
| `ROADMAP.md` | modify | Update T-030 status |

## 6. Dependencies
Standard library only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Update T-030 status and notes |
| `README.md` | Document `sdd task deconflict` command |

## 8. Risks, mitigations, rollback
- **Risk:** Variations in table formatting or missing Section 5 in incomplete plans.
  - **Mitigation:** Flexible markdown scanner searches for Section 5 header, skips non-table lines, and returns empty file list if unpopulated.
- **Rollback:** Revert git working tree changes.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- None active in §8 register.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-05   Phrase received: `APPROVED T-030`
