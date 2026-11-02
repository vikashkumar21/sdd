# Plan — T-019 Finding suppressions and baseline waiver register

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-019

## 1. Approach summary
1. Create `internal/suppress` package to manage `.sdd/suppressions.json` (schema, load, save, match, add, remove, find stale).
2. Integrate suppression checking into `internal/check/check.go`: downgrade matching findings to `INFO` so they pass in standard and `--strict` checks, report stale waivers as `INFO`, and annotate structured JSON output.
3. Integrate suppression awareness into `internal/repair`: badge suppressed patches with `[SUPPRESSED]` in diff previews and prompt to repair or keep waived.
4. Support interactive `[s]uppress` in `runCheck` to record waivers directly from the repair prompt, and offer interactive pruning of stale waivers.
5. Automatically clear waivers from `.sdd/suppressions.json` when a previously waived repair is applied.
6. Add unit and integration tests across `suppress`, `check`, and `cli`.
7. Update living docs (`README.md`, `TECH_STACK.md`, `ROADMAP.md`).

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| (none) | No architectural decisions needed | n/a | Standard library JSON persistence in `.sdd/suppressions.json` | n/a |

## 3. Design
- **Package `internal/suppress`**:
  - `Entry`: `File string`, `Rule string`, `Reason string`, `Date string`
  - `Manifest`: `Schema int`, `Suppressions []Entry`
  - `Load(root string) (Manifest, bool, error)`: confined read via `os.OpenRoot`.
  - `Save(root string, m Manifest) error`: atomic write via `fsutil.WriteAtomicRoot`.
  - `Matches(e Entry, file string, rule string) bool`: normalized path and rule match.
- **Check Integration (`internal/check`)**:
  - Add fields to `Finding`: `Suppressed bool` and `SuppressReason string` (with `omitempty` in JSON).
  - When a finding matches an active suppression: change `Severity` to `Info` and suffix `Message` with `(suppressed: <reason>)`.
  - Check for stale waivers (waivers in `.sdd/suppressions.json` that matched zero findings) and report them as `Finding{Severity: Info, File: ".sdd/suppressions.json", Message: "stale waiver for ... is no longer triggered"}`.
- **Repair Integration (`internal/repair`)**:
  - Add `Suppressed bool` and `SuppressReason string` to `Patch`.
  - In `PlanRepairs`: check against `.sdd/suppressions.json` and populate suppression fields; also identify stale waiver entries.
  - In `Apply`: remove suppressions for successfully repaired files and save `.sdd/suppressions.json`.
- **CLI Workflow (`internal/cli`)**:
  - When displaying patches: render `[SUPPRESSED: <reason>]` in header.
  - Interactive prompt: `Apply repair to <file>? [y]es / [n]o / [s]uppress [y/n/s]: `
  - If `s`: prompt `Waiver reason: `, save to `.sdd/suppressions.json`, and leave file untouched.
  - In `sdd check --fix`, if stale waivers exist, prompt whether to remove them from `.sdd/suppressions.json`.

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Create internal/suppress package with load, save, match | `internal/suppress/` | `go test ./internal/suppress` |
| 2 | Integrate suppression checking and JSON metadata in check | `internal/check/` | `go test ./internal/check -run TestSuppressedFindings` |
| 3 | Integrate suppression awareness and clearing in repair | `internal/repair/` | `go test ./internal/repair -run TestRepairSuppression` |
| 4 | Add [s]uppress option and prompt flow in cli.runCheck | `internal/cli/commands.go` | `go test ./internal/cli -run TestCheckSuppress` |
| 5 | Verify entire suite, formatting, and update living docs | All modified files | `make check && go run ./cmd/sdd check` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/suppress/suppress.go` | create | Suppression manifest schema, load, save, match logic |
| `internal/suppress/suppress_test.go` | create | Unit tests for suppression manifest operations |
| `internal/check/check.go` | modify | Downgrade suppressed findings to Info and add JSON fields |
| `internal/check/check_test.go` | modify | Tests for suppressed findings in check and strict mode |
| `internal/repair/repair.go` | modify | Surface suppression in patches; clear on repair |
| `internal/repair/repair_test.go` | modify | Tests for patch suppression marking and waiver clearing |
| `internal/cli/commands.go` | modify | Add [s]uppress prompt and badge in check --fix |
| `internal/cli/check_fix_test.go` | modify | Integration tests for interactive suppression flow |
| `README.md` | modify | Document suppressions and waivers |
| `TECH_STACK.md` | modify | Add internal/suppress to architecture and verified facts |
| `ROADMAP.md` | modify | Update T-019 status across workflow gates |

## 6. Dependencies
None. Standard library only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `README.md` | Document `.sdd/suppressions.json` and waiver workflow |
| `TECH_STACK.md` | Add `internal/suppress` to architecture diagram; add verified fact |
| `ROADMAP.md` | Update T-019 status |

## 8. Risks, mitigations, rollback
- **Risk:** Malformed `.sdd/suppressions.json` could cause CLI crashes or failed builds.
  - *Mitigation:* Strict schema parsing; corrupt file is flagged as an error without crashing.
- **Rollback:** `git restore .` and `rm -rf internal/suppress` reverts all changes.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active.

## 10. Approval record
Approved by: Vikash   Date: 2026-11-02   Phrase received: `APPROVED T-019`
