# Validation — T-009 P-05: Machine-readable JSON output for sdd check

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Complete

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | go test -v -run TestCheckJSONFlag ./internal/cli | pass | PASS (0.07s) | PASS |
| R-002 | test | go test -v -run TestCheckJSONOutput ./internal/check | pass | PASS (0.06s) | PASS |
| R-003 | test | go test -v -run TestCheckJSONOutput ./internal/check | pass | severity, file, line, message fields verified | PASS |
| R-004 | test | go test -v -run TestCheckJSONStrictMode ./internal/cli | pass | PASS (0.05s, valid=false, exit 1) | PASS |
| R-005 | test | go test -v -run TestCheckHumanReadableDefault ./internal/cli | pass | PASS (0.05s) | PASS |
| R-006 | test | go test -v -run TestCheckJSON ./internal/cli | pass | exit 0 on clean, exit 1 on errors | PASS |
| R-007 | command | make check && go test -race ./... && GOOS=windows go vet ./... | exit 0 | All pkgs passed race detector, 0 lint issues, Windows vet clean | PASS |
| R-008 | test | go test ./internal/check ./internal/cli -count=2 | pass | PASS (7.933s) | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| gofmt -s -l cmd internal | clean (exit 0) |
| go vet ./... | clean (exit 0) |
| golangci-lint run | 0 issues (exit 0) |
| go test ./... -count=1 | ok all packages (exit 0) |
| go test -race ./... | ok all packages (exit 0) |
| GOOS=windows go vet ./... | clean (exit 0) |
| go test ./internal/check ./internal/cli -count=2 | ok (exit 0) |
| go run ./cmd/sdd check | 0 error(s), 0 warning(s) |
| go run ./cmd/sdd check --json | valid JSON output, exit 0 |

## 3. Manual checks
- Ran `sdd check --json` against current repo, confirmed valid indented JSON with `valid: true`, `summary`, and `findings`.

## 4. Negative and failure cases
- Tested `TestCheckJSONOutput` with missing file, confirmed `valid: false` and `errors: 1`.
- Tested `TestCheckJSONStrictMode` with warning, confirmed exit code 1 and `valid: false`.

## 5. Determinism check
Ran `go test ./internal/check ./internal/cli -count=2` twice; outputs identical: yes.

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
Suggested commit message: feat(check): add machine-readable JSON output to sdd check (T-009)
Reviewed by: Vikash (human)   Date: 2026-11-02   Reply received: `COMMITTED T-009`
