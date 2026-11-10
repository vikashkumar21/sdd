# Validation — T-012 P-09: Safe interactive repair command (sdd check --fix)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Complete
Roadmap ref: `ROADMAP.md` T-012

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/cli -run TestCheckFixNoRepairsNeeded` | --fix flag accepted and triggers repair workflow | `PASS: TestCheckFixNoRepairsNeeded (0.00s)` | PASS |
| R-002 | test | `go test -v ./internal/repair -run TestRepairContractParity` | Managed blocks synced across agent files, outside text preserved | `PASS: TestRepairContractParity (0.02s)` | PASS |
| R-003 | test | `go test -v ./internal/repair -run TestRepairTableDelimiters` | Broken table delimiters repaired to match header columns | `PASS: TestRepairTableDelimiters (0.01s)` | PASS |
| R-004 | test | `go test -v ./internal/repair -run TestDiff` | Unified diff preview correctly generated | `PASS: TestDiff (0.00s)` | PASS |
| R-005 | test | `go test -v ./internal/cli -run TestCheckFixDryRun` | --dry-run prints diff without writing files or backups | `PASS: TestCheckFixDryRun (0.00s)` | PASS |
| R-006 | test | `go test -v ./internal/cli -run TestCheckFixPromptAbort` | User rejection aborts without modifying files | `PASS: TestCheckFixPromptAbort (0.00s)` | PASS |
| R-007 | test | `go test -v ./internal/cli -run TestCheckFixYes` | Backups created under .sdd/backup/ before applying fixes | `PASS: TestCheckFixYes (0.02s)` | PASS |
| R-008 | test | `go test -v ./internal/repair -run TestRepairContractParity` | Operations confined via os.Root | `PASS: TestRepairContractParity (0.02s)` | PASS |
| R-009 | test | `go test -v ./internal/cli -run TestCheckFixPerFilePrompt` | Interactive per-file selection applies only chosen file | `PASS: TestCheckFixPerFilePrompt (0.01s)` | PASS |
| R-010 | test | `go test -v ./internal/cli -run TestCheckFixFileFlag` | --file flag restricts repairs to specified files | `PASS: TestCheckFixFileFlag (0.01s)` | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make fmt` | Formatted internal/cli/check_fix_test.go |
| `make verify` | PASS (all packages pass) |
| `make check` | PASS (golangci-lint 0 issues, all tests pass) |
| `go run ./cmd/sdd check` | PASS (0 error(s), 0 warning(s)) |

## 3. Manual checks
- Tested interactive selective confirmation: answering `p` prompts for each file, typing `y` repairs that file and `n` preserves the other.
- Tested `sdd check --fix --dry-run` on a drifted kit: diff printed, 0 files modified.

## 4. Negative and failure cases
- Files without managed blocks not matching agent format are skipped safely.
- User selecting `n` at confirmation aborts without writing or creating backups.
- Filtering by a non-existent file produces "No mechanical repairs needed." and exits 0.

## 5. Determinism check
Ran `go test ./internal/repair ./internal/cli -run "TestCheckFix|TestDiff|TestRepair" -count=1` twice:
- Pass 1: `ok github.com/vikashkumar21/sdd/internal/repair 0.095s`, `ok github.com/vikashkumar21/sdd/internal/cli 0.119s`
- Pass 2: `ok github.com/vikashkumar21/sdd/internal/repair 0.093s`, `ok github.com/vikashkumar21/sdd/internal/cli 0.115s`
Outputs are identical. Determinism confirmed.

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
Suggested commit message: `feat(repair): add safe interactive repair command sdd check --fix (T-012)`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-012`
