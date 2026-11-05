# Validation — T-032 Autonomous multi-agent hierarchy and Task Coordinator skill

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Verified

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/check -run TestManagedBlockParity` | Contracts updated with two-tier roles and Phase C.2; parity verified | PASS (0.50s) | pass |
| R-002 | inspection | Inspect Section 2 and Section 4 in `AGENTS.md`, `CLAUDE.md`, `GEMINI.md` | Primary agent explicitly prohibited from self-approving review.md | Verified in contract Section 2 Phase C.2 & Rule 14 | pass |
| R-003 | test | `go test -v ./internal/templates -run TestVersion` | Template version bumped to 1.3.0 across templates package | PASS (Version == "1.3.0") | pass |
| R-004 | test | `go test -v ./internal/skill -run TestCanonicalTaskCoordinatorSkillValid` | `task-coordinator` skill passes agentskills.io validation | PASS (parsed & validated 0 errors) | pass |
| R-005 | test | `go test -v ./internal/cli -run TestSkillExport` | `sdd skill export` exports canonical task-coordinator skill | PASS (0.48s, exported task-coordinator verified) | pass |
| R-006 | test | `go test -v ./internal/cli -run TestPromptReview` | `sdd prompt review <T-xxx>` generates turnkey subagent review prompt | PASS (prompt generated with rubric & files) | pass |
| R-007 | test | `go test -v ./internal/cli -run TestPromptReviewJSON` | `sdd prompt review <T-xxx> --json` outputs structured review bundle | PASS (valid JSON with requirements, plan, files) | pass |
| R-008 | test | `go test -v ./internal/check -run TestManagedBlockParity` | Agent contracts identical across CLAUDE.md, GEMINI.md, AGENTS.md | PASS (contracts identical within managed block) | pass |
| R-009 | test | `go test -v ./internal/check -run TestCheckReviewDocument` | `sdd check` warns if review.md has missing or self-approved reviewer | PASS (Warn emitted for placeholder/self-approved) | pass |
| R-010 | test | `go test -v ./internal/check -run TestFreshKitHasNoErrors` | Fresh kit generation with version 1.3.0 passes with 0 errors | PASS (fresh kit 0 errors) | pass |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | PASS (vet, unit tests pass across all packages) |
| `make check` | PASS (gofmt check, vet, golangci-lint 0 issues, all tests pass) |
| `./bin/sdd check --strict` | PASS (0 error(s), 0 warning(s), 1 note(s)) |

## 3. Manual checks
- `./bin/sdd prompt review T-032` -> generated turnkey markdown prompt containing requirements, plan, changed files, active feedback rules, and 3-tier severity rubric.
- `./bin/sdd prompt review T-032 --json` -> produced structured JSON context bundle for programmatic agent harnesses.
- `./bin/sdd skill export --skill task-coordinator` -> exported canonical task coordinator skill to `skills/task-coordinator/SKILL.md`.

## 4. Negative and failure cases
- `sdd prompt review non-existent task ID` -> exited with code 1 (`task folder not found across features/, bugfixes/, hotfixes/, or spikes/`).
- `sdd prompt review` (missing task ID) -> exited with code 2 (`usage: sdd prompt review <T-xxx> [--dir <project>] [--json]`).
- `sdd prompt review invalid_task` -> exited with code 1 (`invalid task ID format`).
- Contract parity divergence -> `sdd check` fails with `contract-parity` error (verified in `TestManagedBlockParity`).
- Task in `Awaiting review` with self-approved reviewer in `review.md` -> `sdd check` emits warning (verified in `TestCheckReviewDocument`).

## 5. Determinism check
Ran `./bin/sdd prompt review T-032 --json` twice; diff was byte-identical (0 diff).

## 6. Deviations, known limitations, and feedback-rule compliance
- Deviations: none.
- Feedback-rule compliance (SKILLS.md, "Human feedback rules"): none active.

## 7. Definition of done
- [x] Every requirement has evidence
- [x] Build, lint, and tests pass
- [x] Negative and failure cases tested
- [x] Determinism confirmed
- [x] No files changed outside the plan; no unapproved dependencies
- [x] Living docs updated
- [x] Compliant with every active feedback rule
- [x] Suggested commit message provided

## 8. Sign-off
Suggested commit message: `feat(coordinator): codify two-tier agent hierarchy, Phase C.2 audit, and task-coordinator skill (T-032)`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-032`
