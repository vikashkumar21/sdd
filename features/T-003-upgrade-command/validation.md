# Validation — T-003 sdd upgrade: refresh generated files from newer templates without overwriting human edits

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Passed
Roadmap ref: `ROADMAP.md` T-003

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/upgrade -run TestUpgradeMissingManifest` | Refuses upgrade when manifest is missing | Refused missing manifest with exit 1 | PASS |
| R-002 | test | `go test -v ./internal/upgrade -run TestUpgradeAlreadyUpToDate` | Reports already up-to-date when versions match | Reports up-to-date without writing files | PASS |
| R-003 | test | `go test -v ./internal/upgrade -run TestUpgradeUntouchedFiles` | Untouched files replaced with new templates and backed up | Untouched MISSION.md updated, original backed up to .sdd/backup/ | PASS |
| R-004 | test | `go test -v ./internal/upgrade -run TestUpgradePreservesEditedFiles` | Edited files preserved, .sdd-new proposal written, diff generated | Edited MISSION.md untouched, MISSION.md.sdd-new created, diff generated | PASS |
| R-005 | test | `go test -v ./internal/upgrade -run TestUpgradeManagedBlocks` | Managed blocks upgraded while preserving human rules outside | Managed block in CLAUDE.md refreshed, custom rules outside preserved | PASS |
| R-006 | test | `go test -v ./internal/cli -run TestCLIUpgradeDryRun` | --dry-run previews changes without writing files or proposals | Plan and diff previewed, nothing written | PASS |
| R-007 | test | `go test -v ./internal/upgrade -run TestUpgradeForce` | --force replaces edited files after creating backups | Edited file replaced, original backed up | PASS |
| R-008 | test | `go test -v ./internal/upgrade -run TestUpgradeUntouchedFiles` | Manifest updated with new template version and file hashes | Manifest reflects new template version and replaced state | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | PASS (fmt-check, vet, all tests pass) |
| `make check` | PASS (0 issues from golangci-lint, all tests pass) |
| `go run ./cmd/sdd check --strict` | PASS (0 errors, 0 warnings, 1 info note) |

## 3. Manual checks
- Verified that running `sdd upgrade` on an edited kit produces clean unified diff output and creates `<file>.sdd-new`.
- Verified that running `sdd upgrade` on an untouched kit updates files and creates backup in `.sdd/backup/`.

## 4. Negative and failure cases
- Running in a project without `.sdd/manifest.json` returns exit code 1 with clear refusal.
- Running with corrupted `.sdd/manifest.json` returns exit code 1 with schema refusal.

## 5. Determinism check
Ran `go test ./internal/upgrade ./internal/cli` twice; outputs identical: yes. Evidence: 2 runs pass with identical test counts.

## 6. Deviations, known limitations, and feedback-rule compliance
None.

Feedback-rule compliance (SKILLS.md, "Human feedback rules"): none active.

## 7. Definition of done
- [x] Every requirement has evidence
- [x] Build, lint, and tests pass (commands above)
- [x] Negative and failure cases tested
- [x] Determinism confirmed
- [x] No files changed outside the plan; no unapproved dependencies
- [x] Living docs updated
- [x] Compliant with every active feedback rule
- [x] Suggested commit message provided

## 8. Sign-off
Suggested commit message: `feat(upgrade): add sdd upgrade command with hash checking and proposal generation (T-003)`
Reviewed by: Vikash   Date: 2026-11-02   Reply received: `COMMITTED T-003`
