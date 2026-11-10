# Validation — T-036 Binding Human Feedback Rule F-001 (Resolve all review findings)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Passed
Date: 2026-11-10

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | inspection | `git diff SKILLS.md` | Rule F-001 registered in SKILLS.md §8 with complete metadata | Registered as Active binding rule in Section 8 Register table | PASS |
| R-002 | test / inspection | `git diff AGENTS.md CLAUDE.md GEMINI.md` | Contracts codify Rule F-001 in Phase C.2 and Definition of Done | Codified across all contracts in Phase C.2, Rule 12, Definition of Done | PASS |
| R-003 | test | `go test -v ./internal/templates -run TestVersion` and contract drift check | Template version 1.6.0 and contracts in 100% parity | Version 1.6.0 verified, drift check PASS (0.63s) | PASS |
| R-004 | test | `go test -v ./internal/check -run TestCheckReviewDocument` | `sdd check` verifies unresolved Warnings/Minors emit warnings and Blockers emit errors under F-001 | PASS (0.18s, TestCheckReviewDocument cases 1 & 6 verify) | PASS |
| R-005 | test | `go test -v ./internal/check -run TestCheckReviewDocument` | `Resolved`, `Justified`, `Waived`, `Accepted` statuses pass check | PASS (case 7 verifies zero review-findings for resolved/justified) | PASS |
| R-006 | test | `go test -v ./internal/cli -run TestPromptReview` | Handover prompts include F-001 and mandate resolving all finding tiers | PASS (0.08s, TestPromptReview & TestPromptReviewLens pass) | PASS |
| R-007 | inspection | `git diff skills/` | Canonical skills document Rule F-001 resolution requirement | Updated in skills/code-review/SKILL.md and skills/task-coordinator/SKILL.md | PASS |
| R-008 | test | `make check` and `./bin/sdd check --strict` | 0 errors and 0 warnings | golangci-lint clean (0 issues), all unit tests pass, sdd check 0 errors 0 warnings | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | `go vet ./...` clean, all package tests pass (exit 0) |
| `make check` | `golangci-lint run` 0 issues, all tests pass (exit 0) |
| `make build && ./bin/sdd check --strict` | Built `bin/sdd` cleanly; 0 error(s), 0 warning(s), 1 note(s) (exit 0) |
| `tmpdir=$(mktemp -d) && ./bin/sdd init --dir "$tmpdir" --name testkit --lang go --yes && ./bin/sdd check --dir "$tmpdir"` | Hard Rule 16 verified: regenerated fresh kit with templates 1.6.0, `sdd check` clean (exit 0) |
| `go test -v ./internal/check -run TestCheckReviewDocument` | Pass |
| `go test -v ./internal/templates -run TestVersion` | Pass |
| `go test -v ./internal/cli -run "TestPromptReview|TestPromptPlanReview"` | Pass |

## 3. Manual checks
- Verified `SKILLS.md` §8 table has `F-001` with status `Active`.
- Verified `sdd check` detects unresolved Warnings and Minors in test review documents.
- Verified `sdd check` passes when all findings are marked `Resolved` or `Justified`.

## 4. Negative and failure cases
- `review.md` containing unresolved 🔴 Blocker -> `sdd check` fails with Error (verified in `TestCheckReviewDocument` case 1).
- `review.md` containing unresolved 🟡 Warning -> `sdd check --strict` fails with Warning (code 1; verified in `TestCheckReviewDocument` case 6).
- `review.md` containing unresolved 🟢 Minor -> `sdd check --strict` fails with Warning (code 1; verified in `TestCheckReviewDocument` case 6).

## 5. Determinism check
- Running `./bin/sdd prompt review T-036 --json` twice produced byte-identical files (`diff -u` exit code 0). Determinism confirmed.

## 6. Deviations, known limitations, and feedback-rule compliance
- Deviations: none.
- Feedback-rule compliance (SKILLS.md, "Human feedback rules"): F-001 complied with; all findings will be resolved before GATE 2 presentation.

## 7. Definition of done
- [x] Every requirement has evidence
- [x] Build, lint, and tests pass
- [x] Negative and failure cases tested
- [x] Determinism confirmed
- [x] No files changed outside the plan; no unapproved dependencies
- [x] Independent subagent audit completed in review.md with 0 Blockers
- [x] Living docs updated
- [x] Compliant with every active feedback rule
- [x] Suggested commit message provided

## 8. Sign-off
Suggested commit message: `feat(rules): bind human feedback rule F-001 resolving all review findings (T-036)`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-036`
