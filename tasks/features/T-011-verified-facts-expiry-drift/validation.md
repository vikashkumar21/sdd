# Validation — T-011 P-08: Verified-facts expiry and roadmap drift checker

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Complete
Roadmap ref: `ROADMAP.md` T-011

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/check -run TestCheckVerifiedFactsExpiry` | Fact summary and Re-verify at targets parsed | `PASS: TestCheckVerifiedFactsExpiry (0.05s)` | PASS |
| R-002 | test | `go test -v ./internal/check -run TestCheckVerifiedFactsExpiry` | Warning emitted when Re-verify at task is Done | `PASS: TestCheckVerifiedFactsExpiry (0.05s)` | PASS |
| R-003 | test | `go test -v ./internal/check -run TestCheckVerifiedFactsMilestoneExpiry` | Warning emitted when Re-verify at milestone is completed | `PASS: TestCheckVerifiedFactsMilestoneExpiry (0.05s)` | PASS |
| R-004 | test | `go test -v ./internal/check -run TestCheckDependencyDrift` | Warning emitted when active task has incomplete prerequisite | `PASS: TestCheckDependencyDrift (0.05s)` | PASS |
| R-005 | test | `go test -v ./internal/check -run TestCheckFeatureStatusDrift` | Warning emitted when task is Done but doc status is incomplete | `PASS: TestCheckFeatureStatusDrift (0.05s)` | PASS |
| R-006 | test | `go test -v ./internal/check -run TestCheckJSONIncludesExpiryWarnings` | Warnings present in JSON findings output | `PASS: TestCheckJSONIncludesExpiryWarnings (0.05s)` | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make fmt` | Formatted internal/check/check.go |
| `make verify` | PASS (all packages pass) |
| `make check` | PASS (golangci-lint 0 issues, all tests pass) |
| `go run ./cmd/sdd check` | PASS (0 error(s), 0 warning(s)) |

## 3. Manual checks
- Verified `sdd check` caught all 8 past-due verified facts in `TECH_STACK.md` before targets were aligned with future task `T-018`.
- Verified `sdd check` produces 0 errors and 0 warnings after target alignment.

## 4. Negative and failure cases
- Active tasks (`Not started`) referenced in `Re-verify at` do not trigger warnings.
- Non-task markers (`permanent`, `n/a`) do not trigger warnings.
- Prerequisite tasks marked `none` do not trigger dependency drift warnings.

## 5. Determinism check
Ran `go test ./internal/check -run "TestCheckVerifiedFacts|TestCheckDependency|TestCheckFeatureStatus" -count=1` twice:
- Pass 1: `ok github.com/vikashkumar21/sdd/internal/check 0.303s`
- Pass 2: `ok github.com/vikashkumar21/sdd/internal/check 0.290s`
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
Suggested commit message: `feat(check): add verified-facts expiry and roadmap drift checks (T-011)`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-011`
