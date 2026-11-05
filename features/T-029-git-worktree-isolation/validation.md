# Validation — T-029 Git worktree isolation and concurrent workspace provisioning (sdd task start and clean)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Passed (2026-11-05)

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/worktree -run TestWorktreeStartAndList` | `sdd task start` provisions `.sdd/worktrees/<taskID>-<slug>` | PASS (provisions isolated worktree, verifies filesystem existence) | pass |
| R-002 | test | `go test -v ./internal/worktree -run TestBranchNamingTypologies` | Branch follows `<typology>/<taskID>-<slug>` for feature, bugfix, hotfix, spike | PASS (validates branch naming format across all 4 typologies) | pass |
| R-003 | test | `go test -v ./internal/worktree -run TestHotfixBaseRef` | Hotfixes branch from `--from-tag` or latest release tag | PASS (branches from latest release tag v1.0.0) | pass |
| R-004 | test | `go test -v ./internal/worktree -run TestGitignoreWorktrees` | `.sdd/worktrees/` automatically appended to `.gitignore` | PASS (appends rule and preserves existing content) | pass |
| R-005 | test | `go test -v ./internal/worktree -run TestWorktreeStartAndList` | `.sdd/worktrees.json` updated with active worktree metadata | PASS (persists and reads worktree registry entry) | pass |
| R-006 | test | `go test -v ./internal/cli -run TestWorktreeCommands` | `sdd task list` displays active worktrees (--json supported) | PASS (outputs formatted table and structured JSON array) | pass |
| R-007 | test | `go test -v ./internal/worktree -run TestWorktreeCleanDirtyBackup` | `sdd task clean` backs up dirty changes before removing | PASS (refuses without force, snapshots dirty files to `.sdd/backup/worktrees/`) | pass |
| R-008 | test | `go test -v ./internal/status -run TestStatusWorktrees` | `sdd status` surfaces worktree directory and branch | PASS (renders worktree directory and branch under active task) | pass |
| R-009 | test | `go test -v ./internal/check -run TestCheckWorktrees` | `sdd check` verifies branch naming rules | PASS (warns on invalid branch and errors on unknown task) | pass |
| R-010 | test | `go test -v ./internal/worktree -run TestFailSafeNonGit` | Non-git directories fail gracefully with clear error | PASS (errors cleanly: "not a git repository") | pass |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | PASS (gofmt, go vet, all package unit tests passed) |
| `make check` | PASS (golangci-lint 0 issues, full test suite passed) |
| `go run ./cmd/sdd check --strict` | PASS (0 errors, 0 warnings) |

## 3. Manual checks
- `sdd task start T-001` in test repository -> provisions worktree `.sdd/worktrees/T-001-worktree-cli` on `feature/T-001-worktree-cli`.
- `sdd task list` -> outputs active worktree table and `--json`.
- `sdd status` -> shows `Worktree: .sdd/worktrees/T-001-worktree-cli (branch: feature/T-001-worktree-cli)`.
- `sdd task clean T-001 --delete-branch` -> removes worktree, deletes branch, and prunes registry.

## 4. Negative and failure cases
- Start with non-existent task ID -> fails with `task ID not found in ROADMAP.md or task folders`.
- Clean non-existent worktree -> fails with `no active worktree found for T-xxx`.
- Clean dirty worktree -> refuses without `--force`, backs up dirty files to `.sdd/backup/worktrees/`.
- Run in non-git directory -> fails gracefully with `not a git repository: sdd worktree commands require an active git repository (run 'git init' first)`.
- Git binary missing on PATH -> fails with `git command not found on PATH: please install Git to use worktree isolation commands`.

## 5. Determinism check
Ran `sdd task list --json` and `sdd status --json` twice; outputs identical: byte-identical verified via `diff -u`.

## 6. Deviations, known limitations, and feedback-rule compliance
- Deviations: none.
- Feedback-rule compliance (SKILLS.md, "Human feedback rules"): none active.

## 7. Definition of done
- [x] Every requirement has evidence
- [x] Build, lint, and tests pass
- [x] Negative and failure cases tested
- [x] Determinism confirmed
- [x] No files changed outside the plan; no unapproved dependencies
- [x] Living docs updated
- [x] Compliant with every active feedback rule
- [x] Suggested commit message provided

## 8. Sign-off
Suggested commit message: `feat(worktree): add git worktree isolation and task lifecycle commands (T-029)`
Reviewed by: Vikash (human)   Date: 2026-11-05   Reply received: `COMMITTED T-029`
