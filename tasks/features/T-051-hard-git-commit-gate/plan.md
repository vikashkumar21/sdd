---
status: completed
---
# Plan — T-051 Hard Git commit gate and pre-commit hook enforcement

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Completed

## 1. Approach summary

Implement repository-level Git pre-commit and pre-merge-commit enforcement via a dedicated `internal/hook` package and `sdd hook` CLI command group (`install`, `status`, `uninstall`, `check-gate1`):
1. **Hook Engine (`internal/hook/`):**
   - Provide `Install(root string, opts InstallOptions) (*InstallReport, error)`
   - Provide `Status(root string) (*StatusReport, error)`
   - Provide `Uninstall(root string, opts UninstallOptions) (*UninstallReport, error)`
   - Provide `CheckGate1(root, branch string) (*Gate1Report, error)`
   - Provide portable hook script template provisioning `.git/hooks/pre-commit` and `.git/hooks/pre-merge-commit` with:
     - Dynamic Git hook directory resolution (`git rev-parse --git-path hooks`) supporting Git worktrees.
     - Branch-aware gating (`main`/`master` vs task branches).
     - Reliable TTY check via `/dev/tty` availability and readability.
     - Token evaluation (`$SDD_HUMAN_APPROVED=1`).
     - Gate 1 discipline check via `sdd hook check-gate1` across `task/`, `feature/`, `bugfix/`, `hotfix/`.
     - Mechanical kit consistency audit (`sdd check --strict`).
   - Idempotent non-destructive chaining: back up foreign hooks to `*.sdd-orig` without overwriting on reinstall, and chain execution.
2. **CLI Commands (`internal/cli/hook.go`):**
   - Wire `sdd hook install`, `status`, `uninstall`, and `check-gate1` into CLI dispatch and help usage.
3. **Coordinator Integration (`internal/coordinator/git.go`):**
   - Ensure `ExecuteMerge` injects `SDD_HUMAN_APPROVED=1` when invoking git merge into `main`.
4. **Verification & Living Docs:**
   - Unit tests in `internal/hook/hook_test.go`, `internal/cli/hook_test.go`, and `internal/coordinator/git_test.go`.
   - Update `README.md`, `TECH_STACK.md` (D-043 Approved), `SKILLS.md`, and `ROADMAP.md`.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-043 | Hard git commit gate architecture and pre-commit hook token protocol | Option A: Native `sdd hook install/status/uninstall` command provisioning branch-aware `.git/hooks/pre-commit` and `pre-merge-commit` with TTY confirmation, `SDD_HUMAN_APPROVED=1` token bypass for `sdd task merge`, Gate 1 discipline check, and `sdd check` validation / Option B: Shell alias or external git wrapper script | Option A: Native Git hooks are repository-level, work across any IDE/terminal/agent harness (Claude Code, Cursor, Gemini, terminal, git CLI), and require zero external tools. | Approved |

## 3. Design

### 1. Hook Script Structure (`pre-commit` and `pre-merge-commit`)
```bash
#!/bin/sh
# sdd managed hook — do not edit manually

# 1. Bypass check
if [ "$SDD_SKIP_HOOK" = "1" ]; then
    exit 0
fi

# 2. Resolve current branch and hook directory (worktree compatible)
BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")
GIT_HOOKS_DIR=$(git rev-parse --git-path hooks 2>/dev/null || echo ".git/hooks")
HOOK_NAME=$(basename "$0")

# 3. Protected Branch Gate (main / master)
if [ "$BRANCH" = "main" ] || [ "$BRANCH" = "master" ]; then
    if [ "$SDD_HUMAN_APPROVED" != "1" ]; then
        if [ -r /dev/tty ] && [ -t 1 ] && [ -t 2 ]; then
            printf "\n[sdd hook] Direct commit/merge to %s requires human approval.\n" "$BRANCH" >/dev/tty
            printf "[sdd hook] Proceed with %s? (y/N): " "$HOOK_NAME" >/dev/tty
            read -r reply </dev/tty
            case "$reply" in
                [yY]|[yY][eE][sS]) ;;
                *) printf "[sdd hook] Action aborted by human.\n" >/dev/tty; exit 1 ;;
            esac
        else
            printf "\n[sdd hook] ERROR: Direct autonomous commit/merge to %s is blocked (Hard Rule 2).\n" "$BRANCH" >&2
            printf "[sdd hook] Only human commits or qualified merges (sdd task merge) are permitted.\n" >&2
            exit 1
        fi
    fi
fi

# 4. Gate 1 Discipline Check (Task, Feature, Bugfix, Hotfix branches)
if echo "$BRANCH" | grep -Eq '^(task/|feature/|bugfix/|hotfix/)T-[0-9]{3}'; then
    if command -v sdd >/dev/null 2>&1; then
        sdd hook check-gate1 --branch "$BRANCH" || exit 1
    elif [ -x "./bin/sdd" ]; then
        ./bin/sdd hook check-gate1 --branch "$BRANCH" || exit 1
    fi
fi

# 5. Pre-commit Kit Consistency Audit (commits only)
if [ "$HOOK_NAME" = "pre-commit" ] && [ "$SDD_SKIP_CHECK" != "1" ]; then
    if command -v sdd >/dev/null 2>&1; then
        sdd check --strict || exit 1
    elif [ -x "./bin/sdd" ]; then
        ./bin/sdd check --strict || exit 1
    fi
fi

# 6. Chain to original hook if present (worktree compatible)
ORIG_HOOK="$GIT_HOOKS_DIR/$HOOK_NAME.sdd-orig"
if [ -f "$ORIG_HOOK" ]; then
    exec "$ORIG_HOOK" "$@"
fi
exit 0
```

### 2. `internal/hook` Package Interfaces
```go
package hook

type InstallOptions struct {
    Force bool
}

type InstallReport struct {
    PreCommitPath      string `json:"pre_commit_path"`
    PreMergeCommitPath string `json:"pre_merge_commit_path"`
    Chained            bool   `json:"chained"`
    OriginalHooks      []string `json:"original_hooks,omitempty"`
}

type StatusReport struct {
    Installed          bool   `json:"installed"`
    PreCommitExists    bool   `json:"pre_commit_exists"`
    PreMergeExists     bool   `json:"pre_merge_exists"`
    IsSDDHook          bool   `json:"is_sdd_hook"`
    Executable         bool   `json:"executable"`
    Chained            bool   `json:"chained"`
    OriginalHooks      []string `json:"original_hooks,omitempty"`
}

type UninstallOptions struct {
    Force bool
}

type UninstallReport struct {
    RemovedHooks  []string `json:"removed_hooks"`
    RestoredHooks []string `json:"restored_hooks"`
}

type Gate1Report struct {
    Compliant bool   `json:"compliant"`
    TaskID    string `json:"task_id,omitempty"`
    Reason    string `json:"reason,omitempty"`
}

func Install(root string, opts InstallOptions) (*InstallReport, error)
func Status(root string) (*StatusReport, error)
func Uninstall(root string, opts UninstallOptions) (*UninstallReport, error)
func CheckGate1(root, branch string) (*Gate1Report, error)
```

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Create `internal/hook/hook.go` implementing `Install`, `Status`, `Uninstall`, `CheckGate1`, and hook script template with unit tests. | `internal/hook/hook.go`, `internal/hook/hook_test.go` | `go test -v ./internal/hook` |
| 2 | Create `internal/cli/hook.go` and wire `sdd hook` command group (`install`, `status`, `uninstall`, `check-gate1`) into `cli.Run` and command routing. | `internal/cli/hook.go`, `internal/cli/hook_test.go`, `internal/cli/commands.go`, `internal/cli/cli.go` | `go test -v ./internal/cli -run TestHook` |
| 3 | Update `internal/coordinator/git.go` (`ExecuteMerge`) to pass `SDD_HUMAN_APPROVED=1` environment variable during qualified `sdd task merge`. | `internal/coordinator/git.go`, `internal/coordinator/git_test.go` | `go test -v ./internal/coordinator -run TestExecuteMerge_TokenInjection` |
| 4 | Update living documents (`README.md`, `TECH_STACK.md`, `SKILLS.md`, `ROADMAP.md`). | Living docs | `sdd check --strict` |
| 5 | Run full verification suite (`make verify`, `sdd task audit T-051`), independent review triad audit, and Phase D close. | All | Full suite clean |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/hook/hook.go` | create | Hook install, status, uninstall, check-gate1 engine and script generation |
| `internal/hook/hook_test.go` | create | Unit tests for hook lifecycle, chaining, worktree paths, and execution |
| `internal/cli/hook.go` | create | CLI command handlers for `sdd hook` |
| `internal/cli/hook_test.go` | create | CLI integration tests for `sdd hook` commands |
| `internal/cli/commands.go` | modify | Route `sdd hook` command |
| `internal/cli/cli.go` | modify | Add `sdd hook` to top-level help text |
| `internal/coordinator/git.go` | modify | Inject SDD_HUMAN_APPROVED=1 in task merge runner |
| `internal/coordinator/git_test.go` | modify | Assert SDD_HUMAN_APPROVED=1 presence during merge |
| `README.md` | modify | Document `sdd hook` commands and commit gate |
| `TECH_STACK.md` | modify | Record Decision D-043 Approval and verified facts |
| `SKILLS.md` | modify | Add Section 14 documenting Hard Git Commit Gate standards |
| `ROADMAP.md` | modify | Milestone 12 and T-051 status tracking |
| `.sdd/state/T-051/**` | create | Task state and token telemetry |
| `tasks/features/T-051-hard-git-commit-gate/requirements.md` | modify | Task requirements |
| `tasks/features/T-051-hard-git-commit-gate/plan.md` | modify | Task plan |
| `tasks/features/T-051-hard-git-commit-gate/validation.md` | modify | Task validation |
| `tasks/features/T-051-hard-git-commit-gate/review.md` | modify | Task review |

## 6. Dependencies

None (100% standard library per `D-007`).

## 7. Living docs to update

- `ROADMAP.md`: update T-051 status (`Specifying` -> `Implementing` -> `Awaiting review` -> `Done`), Milestone 12 completion.
- `TECH_STACK.md`: record D-043 approval and verified facts entry.
- `SKILLS.md`: add Section 14 "Hard Git Commit Gate and Repository Hook Architecture".
- `README.md`: document `sdd hook install`, `status`, and `uninstall` commands.

## 8. Risks, mitigations, rollback

- **Risk:** Existing team git hooks (e.g. husky or pre-commit) clobbered.
  - **Mitigation:** Safe chaining: automatically detect foreign hooks, back up to `pre-commit.sdd-orig`, and chain execution.
- **Risk:** Non-interactive environments (CI or container builds) hung waiting for TTY.
  - **Mitigation:** The hook checks `/dev/tty` readability and streams (`[ -r /dev/tty ] && [ -t 1 ] && [ -t 2 ]`); if no interactive terminal is detected and no approval token is provided, it fails immediately with code 1 instead of blocking indefinitely.

### Rollback
- Run `sdd hook uninstall --yes` or remove `.git/hooks/pre-commit` and `.git/hooks/pre-merge-commit`.

## 9. Applicable human-feedback rules

- **F-001:** Resolve all review findings (Blockers, Warnings, Minors) across Review Triad before Gate 2 close.
- **F-002:** Zero personal absolute filesystem paths in generated code, scripts, or documentation.
- **F-003:** Track and record session token consumption via `sdd record-tokens T-051`, document metrics in `validation.md`, and report at Gate 2.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-24   Phrase received: `APPROVED T-051`
