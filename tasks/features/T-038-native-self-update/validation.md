# Validation — T-038 Native sdd self-update CLI binary updater

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Complete

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/update -run TestAssetName` | pass | PASS (0.00s) | PASS |
| R-002 | test | `go test -v ./internal/update -run TestCompareSemver` | pass | PASS (0.00s) | PASS |
| R-003 | test | `go test -v ./internal/update -run "TestParseChecksum\|TestDownloadAndVerify"` | pass | PASS (0.00s) | PASS |
| R-004 | test | `go test -v ./internal/update -run TestReplaceExecutable` | pass | PASS (0.00s) | PASS |
| R-005 | test | `go test -v ./internal/cli -run "TestSelfUpdate_CheckOnly\|TestSelfUpdate_CheckOnly_JSON"` | pass | PASS (0.00s) | PASS |
| R-006 | test | `go test -v ./internal/cli -run TestSelfUpdate_DryRun` | pass | PASS (0.00s) | PASS |
| R-007 | test | `go test -v ./internal/cli -run TestSelfUpdate_TargetVersion` | pass | PASS (0.00s) | PASS |
| R-008 | test | `go test -v ./internal/cli -run TestSelfUpdate_InteractiveAndYes` | pass | PASS (0.00s) | PASS |
| R-009 | test | `go test -v ./internal/update` | pass | PASS (0.118s) | PASS |
| R-010 | test | `make verify` && `make check` && `./bin/sdd check --strict` | pass | PASS (0 errors, 0 warnings) | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make build` | PASS (compiled bin/sdd with version 1.4.0) |
| `make test` | PASS (all unit tests passed across 26 packages) |
| `make lint` | PASS (golangci-lint run: 0 issues) |
| `make verify` | PASS (go vet ./... && go test ./... -count=1 passed) |
| `make check` | PASS (go vet, golangci-lint, go test passed) |
| `./bin/sdd check --strict` | PASS (0 errors, 0 warnings, 1 note) |

## 3. Manual checks
- `./bin/sdd self-update -h`: verified flag definitions (`-check`, `-dry-run`, `-force`, `-json`, `-version`, `-y`, `-yes`).
- `./bin/sdd update -h`: verified alias works identically.
- `./bin/sdd self-update --check --json`: verified JSON output schema handling.

## 4. Negative and failure cases
- Corrupted binary download fails checksum verification and leaves executable unchanged: verified by `TestSelfUpdate_ChecksumMismatch` and `TestDownloadAndVerify/checksum_mismatch_detection`.
- Missing asset in release for current OS/arch returns explicit error: verified by `TestParseChecksum/missing_asset` and `TestCheck/release_not_found`.
- HTTP 404 error returns clean failure without partial file writes: verified by `TestCheck/release_not_found`.
- Windows atomic swap and rollback on rename failure: verified by `TestReplaceExecutable/windows_atomic_swap_with_old_backup`.
- Cancellation on prompt aborts without touching binary: verified by `TestSelfUpdate_InteractiveAndYes/interactive_prompt_cancelled`.

## 5. Determinism check
Ran `go test -v ./internal/update` twice; outputs identical: yes.
Evidence: Run 1 passed in 0.118s; Run 2 passed identically. All tests use synthetic in-memory HTTP transports and deterministic semver logic.

## 6. Deviations, known limitations, and feedback-rule compliance
Deviations: none.

Feedback-rule compliance (SKILLS.md, "Human feedback rules"):
- `F-001`: Active (binding). Resolved all Plan Review findings in Phase A. Subagent Review Triad will audit Phase C.2 and all findings will be resolved before GATE 2 close.

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
Suggested commit message: `feat(update): add native sdd self-update CLI command with SHA256 verification and atomic swap`
Reviewed by: Vikash (human) & Review Triad   Date: 2026-11-10   Reply received: COMMITTED T-038
