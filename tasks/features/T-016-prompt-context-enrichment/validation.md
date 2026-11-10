# Validation — T-016 Handover prompt enrichment for staged context

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Complete

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | go test -v -run TestPromptWithStagedContext ./internal/cli | pass | PASS (0.05s) | PASS |
| R-002 | test | go test -v -run TestPromptWithStagedContext ./internal/cli | pass | PASS (0.05s) | PASS |
| R-003 | test | go test -v -run TestPromptWithoutContext ./internal/cli | pass | PASS (0.06s) | PASS |
| R-004 | test | go test -v -run TestPromptWithTaskContext ./internal/cli | pass | PASS (0.05s) | PASS |
| R-005 | inspection | inspect internal/templates/data/base/prompt.md.tmpl | instructions present | instructions for MISSION, TECH_STACK, ROADMAP, SKILLS | PASS |
| R-006 | inspection | inspect internal/templates/data/base/prompt.md.tmpl | Gate 1 strictness reinforced | GATE 1 IS STRICT emphasized | PASS |
| R-007 | test | go test -v -run TestPromptReadOnly ./internal/cli | pass | PASS (prompt leaves repo clean) | PASS |
| R-008 | command | make check && go test -race ./... && GOOS=windows go vet ./... | exit 0 | All pkgs passed race detector, 0 lint issues, Windows vet clean | PASS |
| R-009 | test | go test ./internal/cli -count=2 | pass | PASS (2.009s) | PASS |

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
| go test ./internal/cli -count=2 | ok (exit 0) |
| go run ./cmd/sdd check | 0 error(s), 0 warning(s) |

## 3. Manual checks
- Verified `sdd prompt gemini` displays staged context table when `.sdd/context/manifest.json` exists.
- Verified `sdd prompt` cleanly omits context section when `.sdd/context/` has no staged artifacts.
- Verified task-specific context isolation when `--task T-001` matches an entry.

## 4. Negative and failure cases
- Tested unknown focus task rejected with exit code 1.
- Tested prompt generation on repository without context cleanly omits section without template leftovers.

## 5. Determinism check
Ran `go test ./internal/cli -count=2` twice; outputs identical: yes.

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
Suggested commit message: feat(cli): enrich prompt handover with staged context guidance (T-016)
Reviewed by: Vikash (human)   Date: 2026-11-02   Reply received: `COMMITTED T-016`
