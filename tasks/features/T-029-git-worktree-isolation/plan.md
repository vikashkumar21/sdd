# Plan — T-029 Git worktree isolation and concurrent workspace provisioning (sdd task start and clean)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-05)

## 1. Approach summary
Implement a robust Git worktree management engine in `internal/worktree/` to provision isolated workspaces for tasks in `.sdd/worktrees/T-xxx-<slug>/` with standardized branch naming (`feature/`, `bugfix/`, `hotfix/`, `spike/`). Wire CLI subcommands (`sdd task start`, `sdd task clean`, `sdd task list`), add fail-safe backup for uncommitted worktree changes, and integrate worktree status into `sdd status` and `sdd check`.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-028 | Git worktree concurrency and workspace isolation | Option A (.sdd/worktrees/ isolation, standardized branch naming, fail-safe cleanup) | Option A | Approved (2026-11-05) |

## 3. Design
### Worktree Architecture (`internal/worktree/`)
1. **Directory & Branch Conventions:**
   - Worktree path: `.sdd/worktrees/<taskID>-<slug>/`
   - Branch: `<typology>/<taskID>-<slug>` where `<typology>` is `feature`, `bugfix`, `hotfix`, or `spike`.
   - `.gitignore`: automatically appends `.sdd/worktrees/` if not present.
2. **Registry (`.sdd/worktrees.json`):**
   - Stores active worktrees:
     ```json
     {
       "schema": 1,
       "worktrees": [
         {
           "task_id": "T-029",
           "typology": "feature",
           "slug": "git-worktree-isolation",
           "branch": "feature/T-029-git-worktree-isolation",
           "path": ".sdd/worktrees/T-029-git-worktree-isolation",
           "base_ref": "HEAD",
           "created_at": "2026-11-05T17:00:00Z"
         }
       ]
     }
     ```
3. **Fail-Safe Lifecycle Operations:**
   - `Start`:
     - Checks git availability and repo root.
     - Locates task in `features/`, `bugfixes/`, `hotfixes/`, `spikes/`.
     - For `hotfix`: derives base from `--from-tag <tag>` or `git describe --tags --abbrev=0` (fallback to `HEAD`).
     - Executes `git worktree add -b <branch> <path> <baseRef>`.
     - Updates `.sdd/worktrees.json`.
   - `Clean`:
     - Checks `git status --porcelain` inside the worktree.
     - If dirty: copies files to `.sdd/backup/worktrees/<taskID>-<timestamp>/` before teardown.
     - Executes `git worktree remove --force <path>`.
     - Optionally deletes branch if `--delete-branch` is passed.
     - Prunes entry from `.sdd/worktrees.json`.
   - `List`:
     - Discovers active worktrees from `.sdd/worktrees.json` and checks if directory exists.

### CLI Integration (`internal/cli/task.go`)
- `sdd task start <T-xxx>` (`--base <ref>`, `--from-tag <tag>`, `--force`)
- `sdd task clean <T-xxx>` (`--force`, `--delete-branch`)
- `sdd task list` (`--json`)

### Status & Check Integration
- `sdd status`: displays `Worktree: .sdd/worktrees/T-xxx-<slug> (branch: <branch>)` for active tasks.
- `sdd check`: warns if a registered worktree branch does not match `^(feature|bugfix|hotfix|spike)/T-\d{3}-[a-z0-9-]+$`.

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Create `internal/worktree` package with Registry, Manager, Git runner, and unit tests. | `internal/worktree/worktree.go`, `internal/worktree/worktree_test.go` | `go test -v ./internal/worktree` |
| 2 | Wire CLI commands (`task start`, `task clean`, `task list`) in `internal/cli/task.go`. | `internal/cli/task.go`, `internal/cli/cli.go`, `internal/cli/cli_test.go` | `go test -v ./internal/cli -run TestWorktreeCommands` |
| 3 | Integrate worktree awareness into `sdd status`. | `internal/status/status.go`, `internal/status/status_test.go` | `go test -v ./internal/status -run TestStatusWorktrees` |
| 4 | Add worktree consistency checks in `internal/check`. | `internal/check/check.go`, `internal/check/check_test.go` | `go test -v ./internal/check -run TestCheckWorktrees` |
| 5 | Update `README.md`, `ROADMAP.md`, run full verification (`make verify`, `make check`, `sdd check --strict`). | `README.md`, `ROADMAP.md` | `make check`, `sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/worktree/worktree.go` | create | Core Worktree manager, registry, and git operations |
| `internal/worktree/worktree_test.go` | create | Unit tests for worktree lifecycle, backup, and error handling |
| `internal/cli/task.go` | modify | Add `task start`, `task clean`, `task list` handlers |
| `internal/cli/cli.go` | modify | Update help and command dispatch for task subcommands |
| `internal/cli/cli_test.go` | modify | Tests for worktree CLI dispatch |
| `internal/status/status.go` | modify | Surface worktree info in status output |
| `internal/status/status_test.go` | modify | Test status rendering with worktrees |
| `internal/check/check.go` | modify | Add worktree branch naming and path checks |
| `internal/check/check_test.go` | modify | Test worktree check rules |
| `README.md` | modify | Document `sdd task start`, `sdd task clean`, `sdd task list` |
| `ROADMAP.md` | modify | Update T-029 status |

## 6. Dependencies
Standard library only (`D-007`). Calls host `git` CLI via `os/exec`.

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Update T-029 status and notes |
| `README.md` | Document `sdd task start`, `clean`, `list` commands |

## 8. Risks, mitigations, rollback
- **Risk:** Host does not have `git` installed or user runs outside a git repo.
  - **Mitigation:** Pre-flight check verifies `git --version` and `git rev-parse --is-inside-work-tree` with actionable error messages.
- **Risk:** Agent leaves uncommitted changes in worktree before `task clean`.
  - **Mitigation:** `task clean` checks `git status --porcelain`; automatically snapshots dirty files into `.sdd/backup/worktrees/` before deleting.
- **Rollback:** `git worktree remove` or delete branch; restore working tree from backups.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- None active in §8 register.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-05   Phrase received: `APPROVED T-029`
