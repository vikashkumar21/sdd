# Plan — T-013 P-10: Agent permission manifest generation

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-013

## 1. Approach summary
1. Create `internal/permissions` package to generate and merge configuration files that deny git write commands for Claude Code, Cursor, and Gemini/Antigravity.
2. Implement non-destructive JSON merging for `.claude/settings.json` so existing user settings are preserved.
3. Wire `sdd permissions` into CLI dispatch with `--dir`, `--agents`, `--dry-run`, and `--force`.
4. Add optional `--permissions` flag to `sdd init`.
5. Add validation in `internal/check/check.go` to ensure generated permission manifests are valid.
6. Verify entire suite with tests and update living docs.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-022 | Agent permission manifest targets and merge behaviour | A) Multi-agent (Claude Code, Cursor, Gemini) with JSON merge<br>B) Claude Code only<br>C) Git hook wrapper only | Option A: Native multi-agent manifests with non-destructive merge | **Approved** 2026-11-02 |

## 3. Design
- **Package `internal/permissions`**:
  - `DeniedGitCommands`: `["git add", "git commit", "git push", "git tag", "git reset", "git rebase", "git stash", "git checkout --", "git merge", "git cherry-pick"]`
  - `Plan`: holds list of planned files with paths, action (`create`, `merge`, `skip`), original content, and modified content.
  - `BuildPlan(root string, targets []string) (Plan, error)`: inspects existing files, prepares byte-identical outputs, detects existing JSON to merge.
  - `Apply(root string, plan Plan) error`: executes writes using `fsutil.WriteAtomicRoot`.
- **Target formats**:
  - `.claude/settings.json`:
    ```json
    {
      "permissions": {
        "deny": [
          "git add *",
          "git checkout -- *",
          "git cherry-pick *",
          "git commit *",
          "git merge *",
          "git push *",
          "git rebase *",
          "git reset *",
          "git stash *",
          "git tag *"
        ]
      }
    }
    ```
  - `.cursor/rules/sdd-git-protection.mdc`: MDC rule file with YAML frontmatter `alwaysApply: true`.
  - `.gemini/rules/git-protection.md`: Markdown instruction file forbidding git writes.
- **CLI Commands (`internal/cli`)**:
  - `sdd permissions [--dir <path>] [--agents <list>] [--dry-run] [--force]`
  - `sdd init --permissions`

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Create internal/permissions package with Plan, BuildPlan, Apply, and JSON merge | `internal/permissions/` | `go test ./internal/permissions` |
| 2 | Add sdd permissions CLI command with --dry-run and --agents flags | `internal/cli/` | `go test ./internal/cli -run TestPermissionsCommand` |
| 3 | Add --permissions flag to sdd init | `internal/cli/init.go` | `go test ./internal/cli -run TestInitWithPermissions` |
| 4 | Add validation check for permission files in internal/check | `internal/check/` | `go test ./internal/check -run TestCheckPermissions` |
| 5 | Verify full suite, linting, formatting, and update living docs | All modified files | `make check && go run ./cmd/sdd check` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/permissions/permissions.go` | create | Permission manifest generator and JSON merger |
| `internal/permissions/permissions_test.go` | create | Unit tests for manifest generation and merging |
| `internal/cli/permissions.go` | create | CLI handler for sdd permissions |
| `internal/cli/permissions_test.go` | create | CLI tests for sdd permissions |
| `internal/cli/cli.go` | modify | Command dispatch and usage text |
| `internal/cli/init.go` | modify | Support --permissions flag in init |
| `internal/check/check.go` | modify | Verify syntax of permission manifests if present |
| `internal/check/check_test.go` | modify | Test permission manifest validation |
| `README.md` | modify | Document sdd permissions and init --permissions |
| `TECH_STACK.md` | modify | Record Decision D-022 and update architecture |
| `ROADMAP.md` | modify | Update T-013 status across gates |

## 6. Dependencies
None. Standard library only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `README.md` | Document `sdd permissions` command and `--permissions` flag |
| `TECH_STACK.md` | Record Decision D-022, add `internal/permissions` to architecture |
| `ROADMAP.md` | Update T-013 status |

## 8. Risks, mitigations, rollback
- **Risk:** Modifying an existing `.claude/settings.json` could erase custom user settings.
  - *Mitigation:* Parse existing JSON into `map[string]any`, merge `permissions.deny` without touching other keys, and write atomically.
- **Rollback:** Delete generated `.claude/settings.json`, `.cursor/`, and `.gemini/rules/`.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active.

## 10. Approval record
Approved by: Vikash   Date: 2026-11-02   Phrase received: `APPROVED T-013`
