# Requirements — T-029 Git worktree isolation and concurrent workspace provisioning (sdd task start and clean)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-05)
Roadmap ref: `ROADMAP.md` T-029

## 1. Goal
Provide mechanical, fail-safe Git worktree workspace isolation for autonomous agent tasks via `sdd task start <T-xxx>`, `sdd task clean <T-xxx>`, and `sdd task list`, standardizing branch naming (`feature/`, `bugfix/`, `hotfix/`, `spike/`), preventing git collision on `main`, backing up uncommitted changes on teardown, and integrating active worktree tracking into `sdd status` and `sdd check`.

## 2. Context and references
- Mission / principles touched: Principle 1 (Deterministic), Principle 2 (Fail safe), Principle 3 (Human in the loop for the irreversible), Principle 5 (Never overwrite human work)
- Decisions this relies on:
  - D-002: Spec-driven workflow with human gates
  - D-007: Go, standard library only, single static binary
  - D-027: Work typology and specialized lifecycle artifacts
  - D-028: Git worktree concurrency and workspace isolation
  - D-029: Two-tier git governance and Task Coordinator orchestration
- Depends on tasks: T-028 (Done)

## 3. Scope
**In scope**
- Worktree Manager in `internal/worktree/`:
  - Worktree creation at `.sdd/worktrees/<taskID>-<slug>/` via `git worktree add -b <branch> <path> <baseRef>`.
  - Automatic typology discovery from task directories (`features/`, `bugfixes/`, `hotfixes/`, `spikes/`).
  - Standardized branch naming:
    - Feature: `feature/T-xxx-<slug>`
    - Bugfix: `bugfix/T-xxx-<slug>`
    - Hotfix: `hotfix/T-xxx-<slug>` (base from `--from-tag` or latest release tag)
    - Spike: `spike/T-xxx-<slug>`
  - Automatic `.gitignore` management ensuring `.sdd/worktrees/` is ignored.
  - Worktree registry in `.sdd/worktrees.json`.
  - Worktree cleanup via `sdd task clean <T-xxx>` with dirty state detection, backup in `.sdd/backup/worktrees/`, and `git worktree remove`.
  - Worktree listing via `sdd task list` (`--json` supported).
- CLI subcommands:
  - `sdd task start <T-xxx>` (with `--base`, `--from-tag`, `--force`)
  - `sdd task clean <T-xxx>` (with `--force`, `--delete-branch`)
  - `sdd task list` (with `--json`)
- Integration with `sdd status`:
  - Surfaces active worktree path and branch under active task information.
- Integration with `sdd check`:
  - Validates active worktree branch names and paths against registered tasks.
- Fail-safe git validation:
  - Verifies git availability and repository presence before executing git operations.

**Out of scope**
- Static plan file collision analysis across multiple tasks (handled in T-030).
- Task Coordinator automated merge and blast radius evaluation (handled in T-031).
- Autonomous worker agent commit governance and prompt skills (handled in T-032).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL provide `sdd task start <T-xxx>` to provision an isolated git worktree at `.sdd/worktrees/<taskID>-<slug>/` with dedicated branch `<typology>/<taskID>-<slug>`. | MUST | test |
| R-002 | The system SHALL detect the work typology from the task folder location (`features/`, `bugfixes/`, `hotfixes/`, `spikes/`) and format the branch name according to D-028 conventions (`feature/`, `bugfix/`, `hotfix/`, `spike/`). | MUST | test |
| R-003 | For `hotfix` tasks, `sdd task start` SHALL support `--from-tag <tag>` (or discover the latest git release tag) as the branch base; other typologies SHALL default to `HEAD`. | MUST | test |
| R-004 | The system SHALL automatically ensure `.sdd/worktrees/` is listed in `.gitignore` upon worktree creation if `.gitignore` exists. | MUST | test |
| R-005 | The system SHALL maintain a worktree registry in `.sdd/worktrees.json` tracking active task worktrees, branches, base references, paths, and creation timestamps. | MUST | test |
| R-006 | The system SHALL provide `sdd task list` (with `--json` support) to inspect active task worktrees, branches, typologies, and paths. | MUST | test |
| R-007 | The system SHALL provide `sdd task clean <T-xxx>` to safely teardown the worktree via `git worktree remove`, backing up uncommitted changes to `.sdd/backup/worktrees/` if dirty. | MUST | test |
| R-008 | `sdd status` SHALL discover active task worktrees from `.sdd/worktrees.json` and display the worktree path and branch in active task summaries. | MUST | test |
| R-009 | `sdd check` SHALL verify that any registered worktree branch adheres to the pattern `^(feature|bugfix|hotfix|spike)/T-\d{3}-[a-z0-9-]+$`. | MUST | test |
| R-010 | The system SHALL fail safe: if git is not installed or the directory is not a git repository, operations fail with exit code 1 and actionable errors. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Branch naming, registry serialization, and path computation must be deterministic across platforms.
- **Fail-safe:** Never delete uncommitted work in a worktree without an atomic backup. Never leave git in a corrupted worktree state.
- **Standard library only:** All git operations invoked via `os/exec` standard library (`D-007`); no third-party libraries.
- **Platform compatibility:** Paths in `.sdd/worktrees.json` use normalized forward slashes; git commands execute identically on macOS, Linux, and Windows.

## 6. Constraints and assumptions
- Host environment has `git` installed and on `PATH` for worktree commands.
- The project is inside a git repository for `task start`/`clean` operations.

## 7. Open questions
(None - all decisions resolved in D-028)

## 8. Acceptance criteria
- Given a task `T-029` in `features/T-029-git-worktree-isolation/`, running `sdd task start T-029` creates `.sdd/worktrees/T-029-git-worktree-isolation/` on branch `feature/T-029-git-worktree-isolation`, adds `.sdd/worktrees/` to `.gitignore`, and registers it in `.sdd/worktrees.json`.
- Given an active worktree, running `sdd task list --json` outputs structured JSON listing the task, branch, typology, and path.
- Given an active worktree with dirty uncommitted edits, running `sdd task clean T-029` creates a backup in `.sdd/backup/worktrees/`, removes the git worktree, and unregisters it.
- Given an active worktree, running `sdd status` displays the worktree directory and branch under active task information.
