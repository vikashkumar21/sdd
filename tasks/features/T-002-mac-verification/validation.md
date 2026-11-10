# Validation — T-002 Verify on the maintainer's Mac: Go 1.27, golangci-lint v2, macOS binary; fix findings (P-01)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Complete

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | inspection | `grep "^go " go.mod` | `go 1.27` | `go 1.27` | PASS |
| R-002 | test | `golangci-lint run` | exit 0, 0 issues | `0 issues.` exit code 0 | PASS |
| R-003 | test | `go test -race ./... -count=1` | exit 0, all packages PASS | 9 packages passed with race detector | PASS |
| R-004 | test | `make check` | exit 0, clean check | `fmt-check`, `vet`, `lint`, `test` all pass, exit 0 | PASS |
| R-005 | command | `make dist` | produces dist/ binaries + SHA256SUMS | 4 binaries (darwin/arm64, darwin/amd64, linux/amd64, linux/arm64) and SHA256SUMS created | PASS |
| R-006 | demo | `./dist/sdd_darwin_arm64 version` | `sdd 0.1.0 (templates 1.0.0)` | `sdd 0.1.0 (templates 1.0.0)` exit 0 | PASS |
| R-007 | test | scratch directory test runner | all project types succeed or refuse cleanly | Greenfield (Go, Java, Python, TS), brownfield, handmade refusal, read-only failure all verified | PASS |
| R-008 | test | repeat tests twice | byte-identical outputs | Consecutive `make check` runs produced identical outputs | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make fmt-check` | Clean, exit 0 |
| `make vet` | Clean, exit 0 |
| `make lint` | `0 issues.`, exit 0 |
| `make test` | 9 packages tested, 0 failures, exit 0 |
| `make check` | Full pipeline passed, exit 0 |
| `make dist` | Produced 4 static binaries and SHA256SUMS, exit 0 |

## 3. Manual checks
- `which sdd`: confirmed not shadowing system PATH (`sdd not found`).
- Executed `./dist/sdd_darwin_arm64 version`: printed `sdd 0.1.0 (templates 1.0.0)`.

## 4. Negative and failure cases
- Read-only directory initialization failed cleanly with `permission denied` without leaving half-written files.
- Hand-made kit (`dbguard-kit`) without manifest was cleanly refused by `sdd init` with exit code 1.
- Running `sdd init` without `--yes` and with empty input safely aborted with exit code 1 without writing anything.

## 5. Determinism check
Ran `make check` twice consecutively; exit codes and test outputs were identical.

## 6. Deviations, known limitations, and feedback-rule compliance
- Deviations: none.
- Feedback-rule compliance: none active in `SKILLS.md`.

## 7. Definition of done
- [x] Every requirement has evidence
- [x] Build, lint, and tests pass (commands above)
- [x] Negative and failure cases tested
- [x] Determinism confirmed
- [x] No files changed outside the plan; no unapproved dependencies
- [x] Living docs updated (ROADMAP.md status set to Awaiting review, TECH_STACK.md D-004 approved)
- [x] Compliant with every active feedback rule
- [x] Suggested commit message provided

## 8. Sign-off
Suggested commit message: `fix(ci): update to go 1.27, resolve golangci-lint findings, and verify on macOS (T-002)`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-002`
