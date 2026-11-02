# Plan — T-003 sdd upgrade: refresh generated files from newer templates without overwriting human edits

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-003

## 1. Approach summary
1. Resolve D-017 (Manifest schema 1, managed blocks, and side-by-side proposal format).
2. Implement core upgrade planning and execution engine in `internal/upgrade/upgrade.go`:
   - Read `.sdd/manifest.json` and verify schema 1.
   - Re-render templates using saved manifest answers.
   - Compare current file hashes against manifest hashes to separate untouched vs. human-edited files.
   - Plan operations: `unchanged`, `update` (untouched), `proposal` (edited), `replace` (with `--force`).
   - Execute updates atomically under `os.Root`, writing backups to `.sdd/backup/<timestamp>/`, writing `<file>.sdd-new` proposals, and updating `.sdd/manifest.json`.
3. Integrate `sdd upgrade` CLI command in `internal/cli/upgrade.go` and `internal/cli/commands.go`.
4. Provide comprehensive unit and CLI integration tests in `internal/upgrade/upgrade_test.go` and `internal/cli/upgrade_test.go`.
5. Update living documents (`README.md`, `TECH_STACK.md`, `ROADMAP.md`).

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-017 | Manifest schema 1, managed-block markers, and side-by-side proposal format | A) Confirm schema 1, `<!-- sdd:begin contract templates=<v> -->`, and `<file>.sdd-new` proposal files with diff previews as permanent compatibility surface<br>B) Introduce schema 2 | Option A: Confirm schema 1 and proposal format | Proposed |

## 3. Design
- **Manifest & Template Inspection**:
  - `ReadManifest(root)` extracts previous answers and file hashes.
  - `templates.New(override)` loads the new template set.
  - If template version has not changed, override is identical, and `--force` is false, report up-to-date and exit 0.
- **Change Detection**:
  - For each generated file:
    - If file matches manifest hash: mark as **untouched** -> safe to update in place.
    - If file matches new template content: mark as **unchanged** -> no action.
    - If file differs from manifest hash and differs from new template: mark as **edited** -> do not overwrite; write `<file>.sdd-new` proposal and generate unified diff preview via `repair.Diff`.
    - If `--force` is active: treat edited files as update candidates after mandatory backup.
  - Managed blocks:
    - For agent contract files, extract inner text via `block.Inner`. If inner hash matches manifest hash, update only the managed block, preserving human rules outside the block.
- **Execution & Safety**:
  - All file reads and writes use `os.Root` confinement.
  - Overwritten files are backed up to `.sdd/backup/<timestamp>/<path>`.
  - Manifest is updated with new template version, tool version, and updated hashes.

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement core upgrade logic (plan, hash comparison, managed block handling, backup, apply) | `internal/upgrade/upgrade.go`, `internal/upgrade/upgrade_test.go` | `go test ./internal/upgrade` |
| 2 | Implement `sdd upgrade` CLI command with flags (--dir, --dry-run, --yes, --force, --templates) | `internal/cli/upgrade.go`, `internal/cli/commands.go`, `internal/cli/upgrade_test.go` | `go test ./internal/cli -run TestCLIUpgrade` |
| 3 | Update living docs (README, TECH_STACK, ROADMAP) | `README.md`, `TECH_STACK.md`, `ROADMAP.md` | `go run ./cmd/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/upgrade/upgrade.go` | create | Core upgrade planning and application engine |
| `internal/upgrade/upgrade_test.go` | create | Unit tests for hash checking, proposals, and backups |
| `internal/cli/upgrade.go` | create | CLI flags and command handler for `sdd upgrade` |
| `internal/cli/upgrade_test.go` | create | End-to-end CLI tests for `sdd upgrade` |
| `internal/cli/commands.go` | modify | Register `upgrade` in top-level CLI dispatcher |
| `README.md` | modify | Document `sdd upgrade` usage and flags |
| `TECH_STACK.md` | modify | Record decision D-017 |
| `ROADMAP.md` | modify | Update T-003 status |

## 6. Dependencies
None. Standard library only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `README.md` | Add `sdd upgrade` command documentation |
| `TECH_STACK.md` | Approve D-017 in Decision register |
| `ROADMAP.md` | Update T-003 status across gates |

## 8. Risks, mitigations, rollback
- **Risk:** Overwriting human edits in files that were partially customized.
  - *Mitigation:* Strict hash matching against manifest SHA256; write `<file>.sdd-new` side-by-side proposals and make backups before any writes.
- **Rollback:** `git restore .` reverts all changes.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active.

## 10. Approval record
Approved by: Vikash   Date: 2026-11-02   Phrase received: `APPROVED T-003`
