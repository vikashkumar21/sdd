# Plan — T-012 P-09: Safe interactive repair command (sdd check --fix)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-02)
Roadmap ref: `ROADMAP.md` T-012

## 1. Approach summary
1. Create `internal/repair` package implementing:
   - Contract block synchronization across `CLAUDE.md`, `GEMINI.md`, and `AGENTS.md` using `internal/block`.
   - Table delimiter and column alignment repair using `internal/markdown`.
   - Unified diff generator for previewing planned changes.
   - Atomic file applicator creating backups in `.sdd/backup/<timestamp>/` using `os.OpenRoot` and `fsutil.WriteAtomicRoot`.
2. Integrate `--fix`, `--dry-run`, and `--yes` flags into `runCheck` in `internal/cli/commands.go`.
3. Add unit and integration tests covering contract parity repairs, table fixes, dry runs, user confirmation aborts, and backup preservation.
4. Update documentation in `README.md` and record verified facts in `TECH_STACK.md`.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| (none) | No architectural decisions needed | n/a | Standard library implementation | n/a |

## 3. Design
- **Package `internal/repair`**:
  - `Plan(root string) (RepairPlan, error)`: inspects agent files and markdown tables; returns list of file patches.
  - `Diff(original, modified, path string) string`: returns unified diff with `--- a/file` and `+++ b/file`.
  - `Apply(root string, plan RepairPlan, now time.Time) (Result, error)`: creates backup under `.sdd/backup/<timestamp>/` and writes modified files via `fsutil.WriteAtomicRoot`.
- **Contract Repair**:
  - Locate canonical managed block (prefer `AGENTS.md`, then `CLAUDE.md`, then `GEMINI.md`).
  - For each agent file whose managed block differs or is missing, replace/append the managed block while keeping all lines outside the block intact.
- **Table Delimiter Repair**:
  - For tables with mismatched column counts on delimiter lines (`|---|`), generate a normalized delimiter row matching the header column count.
- **CLI Flow**:
  - If `*fix`:
    - Generate `repair.Plan(root, *fileFilter)`.
    - If empty, print `"No mechanical repairs needed."` and proceed with normal check report.
    - Print unified diff of planned patches.
    - If `*dryRun`: print summary and exit 0.
    - If not `*yes`:
      - If multiple files are planned: prompt `Apply repairs? [y]es to all / [n]o / [p]er-file [y/n/p]: `
      - If `p`: iterate over each patch, display diff, and prompt `Apply repair to <file>? [y/N]: `
      - If single file: prompt `Apply repair to <file>? [y/N]: `
      - If denied or none selected: print `"Repairs aborted. No changes written."` and exit 0.
    - Call `repair.Apply` for approved patches. Print confirmation with backup path.

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Create internal/repair package with diff, plan, and apply | `internal/repair/` | `go test ./internal/repair` |
| 2 | Add contract synchronization and table repair logic | `internal/repair/repair.go` | `go test ./internal/repair -run TestRepairContractParity` |
| 3 | Wire --fix, --dry-run, --yes flags in cli.runCheck | `internal/cli/commands.go` | `go test ./internal/cli -run TestCheckFix` |
| 4 | Add integration tests for backup creation and confirmation | `internal/cli/check_fix_test.go` | `go test ./internal/cli -run TestCheckFix` |
| 5 | Update living docs (README, TECH_STACK, ROADMAP) | `README.md`, `TECH_STACK.md`, `ROADMAP.md` | `make check && go run ./cmd/sdd check` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/repair/repair.go` | create | Repair planner and execution engine |
| `internal/repair/diff.go` | create | Unified diff preview generator |
| `internal/repair/repair_test.go` | create | Unit tests for contract and table repairs |
| `internal/cli/commands.go` | modify | Add --fix, --dry-run, --yes flags and dispatch |
| `internal/cli/check_fix_test.go` | create | CLI integration tests for sdd check --fix |
| `README.md` | modify | Document sdd check --fix |
| `TECH_STACK.md` | modify | Add internal/repair to architecture and verified facts |
| `ROADMAP.md` | modify | Update T-012 status across workflow gates |

## 6. Dependencies
None. Standard library only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `README.md` | Document `sdd check --fix`, `--dry-run`, and `--yes` |
| `TECH_STACK.md` | Add `internal/repair` to architecture diagram; add verified fact |
| `ROADMAP.md` | Update T-012 status |

## 8. Risks, mitigations, rollback
- **Risk:** Overwriting human text during repair (violating S1).
  - *Mitigation:* Repairs operate strictly inside marked contract blocks or strictly on table delimiter formatting; mandatory backups created in `.sdd/backup/` before every write.
- **Rollback:** `git restore internal/` and `rm -rf internal/repair` reverts all changes.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active.

## 10. Approval record
Approved by: Vikash   Date: 2026-11-02   Phrase received: `APPROVED T-012`
