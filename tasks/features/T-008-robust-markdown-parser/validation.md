# Validation — T-008 P-04: Robust markdown parser in check

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Complete

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | go test -v -run TestCodeFences ./internal/markdown | pass | PASS (0.00s) | PASS |
| R-002 | test | go test -v -run TestNestedCodeFences ./internal/markdown | pass | PASS (0.00s) | PASS |
| R-003 | test | go test -v -run TestInlineCodePipes ./internal/markdown | pass | PASS (0.00s) | PASS |
| R-004 | test | go test -v -run TestEscapedPipes ./internal/markdown | pass | PASS (0.00s) | PASS |
| R-005 | test | go test -v -run TestCheckTablesWithPipesInCodeAndEscaped ./internal/check | pass | PASS (0.06s) | PASS |
| R-006 | test | go test -v -run TestDetectsProblems ./internal/check | pass | PASS (0.76s) | PASS |
| R-007 | command | make check && go test -race ./... && GOOS=windows go vet ./... | exit 0 | All pkgs passed race detector, 0 lint issues, Windows vet clean | PASS |
| R-008 | test | go test ./internal/markdown ./internal/check -count=2 | pass | PASS (2.143s) | PASS |

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
| go test ./internal/markdown ./internal/check -count=2 | ok (exit 0) |
| go run ./cmd/sdd check | 0 error(s), 0 warning(s) |

## 3. Manual checks
- Verified nested 4-backtick code fences enclosing 3-backtick snippets parse cleanly without unbalanced fence errors.
- Verified tables containing escaped pipes or code spans with pipes count true structural columns.

## 4. Negative and failure cases
- `TestCodeFences/unclosed_backtick_fence`: unclosed code block caught.
- `TestDetectsProblems/broken_table`: column count mismatch caught.
- `TestDetectsProblems/unbalanced_fence`: unclosed fence caught.

## 5. Determinism check
Ran `go test ./internal/markdown ./internal/check -count=2` twice; outputs identical: yes.

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
Suggested commit message: feat(check): add robust markdown fence and table parser (T-008)
Reviewed by: Vikash (human)   Date: 2026-11-02   Reply received: `COMMITTED T-008`
