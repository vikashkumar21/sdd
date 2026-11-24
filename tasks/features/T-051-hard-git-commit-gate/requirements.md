---
status: completed
---
# Requirements — T-051 Hard Git commit gate and pre-commit hook enforcement

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-051
Decision ref: `TECH_STACK.md` D-043 (Approved 2026-11-24)

## 1. Goal

Transform SDD's "Human Only Commits" rule (Hard Rule 2) from a prompt-level instruction into an ironclad mechanical lock by provisioning repository-level Git hooks (`sdd hook install`) that strictly block unauthorized agent commits to `main`, require human TTY confirmation or token authorization, enforce Gate 1 discipline on task branches, and enable qualified coordinator merges (`sdd task merge`).

## 2. Context and references

- Mission / principles: Principle 1 (Deterministic by construction), Principle 2 (Fail safe), Principle 3 (Human in the loop for the irreversible), Principle 5 (Never overwrite human work).
- Decisions: D-003 / D-007 (Standard library only), D-031 / D-036 (Two-tier coordinator and worker roles), D-043 (Hard git commit gate architecture).
- Hard Rules: Hard Rule 2 (Never commit or push; human commits), Hard Rule 13 (Two-tier hierarchy: autonomous commits allowed on task branches; forbidden on main).

## 3. Scope

### In scope
1. **`sdd hook` CLI Command Suite (`R-001`):**
   - `sdd hook install [--dir <project>] [--force]`: provisions branch-aware SDD `pre-commit` and `pre-merge-commit` hooks into Git's hooks directory.
   - `sdd hook status [--dir <project>] [--json]`: reports hook installation state, target paths, execution permissions, and chaining status.
   - `sdd hook uninstall [--dir <project>] [--yes] [--force]`: safely removes SDD hooks and restores any pre-existing chained hooks.
   - `sdd hook check-gate1 [--dir <project>]`: mechanical CLI helper invoked by hooks to verify Gate 1 discipline across platforms without shell brittleness.
2. **Deterministic Cross-Platform Hook Script (`R-002`):**
   - Provide a portable, POSIX-compliant shell script template installed with `0o755` permissions in both `.git/hooks/pre-commit` and `.git/hooks/pre-merge-commit`.
   - Resolve Git hooks path dynamically via `git rev-parse --git-path hooks` to support isolated Git worktrees natively.
3. **Branch-Aware Commit & Merge Policy (`R-003`):**
   - **Protected Branches (`main`, `master`):**
     - Autonomous agent direct commits and unverified direct merges are **blocked**.
     - Commits and merges require interactive human confirmation via `/dev/tty` (`[sdd hook] Direct commit/merge to main requires human approval. Proceed? (y/N)`), OR explicit authorization token `SDD_HUMAN_APPROVED=1`.
     - Non-interactive autonomous executions attempting to commit or merge to `main` without the token exit with code 1.
   - **Task / Feature / Bugfix / Hotfix Branches (`task/*`, `feature/*`, `bugfix/*`, `hotfix/*`, worktrees):**
     - Autonomous worker agent commits are **permitted** per Hard Rule 13 to allow iterative code and test saves.
4. **Gate 1 Discipline Check (`R-004`):**
   - When committing on a task branch (`task/*`, `feature/*`, `bugfix/*`, `hotfix/*`):
     - If staged changes include application source files outside `tasks/` (e.g. `*.go`, `*.py`, `*.java`, `*.ts`, `cmd/`, `internal/`), `sdd hook check-gate1` verifies that the corresponding task's `plan.md` has an approved record (`Status: Approved`, `Status: Implementing`, or approved frontmatter).
     - If an agent attempts to commit application code before Gate 1 plan approval, the hook blocks the commit with an explanatory Gate 1 violation error.
5. **Pre-Commit Kit Consistency Audit (`R-005`):**
   - On commit, the hook runs `sdd check --strict`:
     - Blocks commit if mechanical consistency checks fail.
     - Supports emergency human bypass via `git commit --no-verify` or `SDD_SKIP_CHECK=1`.
6. **Chaining & Non-Destructive Hook Lifecycle (`R-006`):**
   - If an existing foreign pre-commit or pre-merge-commit hook is present:
     - Backs it up to `*.sdd-orig` and chains execution to it.
     - `sdd hook install` is strictly idempotent: re-running does NOT overwrite existing `*.sdd-orig` backups.
     - On `sdd hook uninstall`, restores the original hooks cleanly; refuses to uninstall non-SDD foreign hooks unless `--force`.
7. **Coordinator `sdd task merge` Integration (`R-007`):**
   - Update `internal/coordinator/git.go` (`ExecuteMerge`) so that qualified pre-merge passes `SDD_HUMAN_APPROVED=1` in the execution environment when executing `git merge` into `main`, enabling turnkey automated merge once 0 review blockers and passing tests are verified.
8. **Standard Library Purity (`R-008`):**
   - 100% Go standard library implementation (`D-007`); zero external dependencies.

### Out of scope
- Server-side pre-receive or GitHub branch protection bypass (server repositories enforce their own branch protection).
- Modifying global git templates in `~/.git-templates`.

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL provide `sdd hook install`, `sdd hook status`, `sdd hook uninstall`, and `sdd hook check-gate1` CLI commands with `--dir`, `--force`, `--yes`, and `--json` support. | MUST | test / demo |
| R-002 | The system SHALL generate portable, POSIX-compliant `.git/hooks/pre-commit` and `.git/hooks/pre-merge-commit` scripts with `0o755` permissions, dynamically resolving Git hook directories via `git rev-parse --git-path hooks` to support Git worktrees. | MUST | test |
| R-003 | The system SHALL enforce branch-aware gating in pre-commit and pre-merge-commit hooks: direct commits and merges to `main`/`master` SHALL be blocked unless running in an interactive human session via `/dev/tty` or authorized with `SDD_HUMAN_APPROVED=1`, while task branches SHALL allow autonomous commits. | MUST | test |
| R-004 | The system SHALL enforce Gate 1 discipline across all typologies (`task/`, `feature/`, `bugfix/`, `hotfix/`), blocking commits containing application code outside `tasks/` if `plan.md` has not been approved. | MUST | test |
| R-005 | The system SHALL run `sdd check --strict` in the pre-commit hook, aborting the commit if mechanical consistency checks fail. | MUST | test |
| R-006 | The system SHALL preserve foreign hooks by backing them up to `*.sdd-orig`, chaining their execution, maintaining idempotency, and restoring them upon `sdd hook uninstall`. | MUST | test |
| R-007 | The system SHALL inject `SDD_HUMAN_APPROVED=1` in `internal/coordinator/git.go` when executing qualified task branch merges into `main`. | MUST | test |
| R-008 | The system SHALL implement all hook logic using only the Go standard library with zero third-party dependencies (`D-007`). | MUST | test |

## 5. Non-functional requirements

- **Determinism:** Hook template rendering and status reporting are 100% deterministic.
- **Fail-safe:** If hook state or branch name cannot be determined, fail restrictive and prompt the human.
- **Performance budget:** Hook execution overhead < 150ms on clean repositories.
- **Portability:** Works across macOS, Linux, and Windows (under Git Bash, WSL, and MSYS).

## 6. Constraints and assumptions

- The target project must be a Git repository (`.git` directory or pointer file exists).
- Git hook execution honors standard Git convention (`--no-verify` bypass).

## 7. Open questions

(none — Decision D-043 submitted for human approval)

## 8. Acceptance criteria

- **AC-1 (Installation & Status):** Running `sdd hook install` creates `.git/hooks/pre-commit` and `.git/hooks/pre-merge-commit` with mode `0o755`. `sdd hook status --json` outputs `installed: true`.
- **AC-2 (Main Branch Protection):** Running `git commit` or `git merge` non-interactively on `main` without `SDD_HUMAN_APPROVED=1` exits with non-zero code and blocks the action.
- **AC-3 (Task Branch Autonomous Commits):** Running `git commit` on branch `task/T-001-demo` with approved `plan.md` succeeds.
- **AC-4 (Gate 1 Enforcement):** Committing application code on a task/bugfix/hotfix branch whose `plan.md` is unapproved fails with Gate 1 violation message.
- **AC-5 (Worktree & Chaining Support):** Hook execution resolves hooks in worktrees via `git rev-parse --git-path hooks`. Pre-existing hooks are preserved in `*.sdd-orig` without recursion, and restored cleanly upon uninstall.
- **AC-6 (Coordinator Merge):** Running `sdd task merge` passes `SDD_HUMAN_APPROVED=1` and merges cleanly into `main` through `pre-merge-commit`.
