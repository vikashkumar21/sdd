# Validation — T-031 Task Coordinator pre-merge qualification, simulation, and sdd task merge

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Complete
Date: 2026-11-05

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/coordinator -run TestQualifyReviewBlockers` | Rejects merge if review.md has unresolved 🔴 Blockers or status is not Approved | Correctly flagged 1 blocker and set ReadyToMerge=false | pass |
| R-002 | test | `go test -v ./internal/coordinator -run TestBlastRadiusComputation` | Computes downstream importing packages for modified source files | Cascaded modified package through forward and reverse AST dependencies | pass |
| R-003 | test | `go test -v ./internal/coordinator -run TestRegressionTestSweep` | Runs test sweep on modified packages + blast radius | Verified test runner received modified + blast radius packages | pass |
| R-004 | test | `go test -v ./internal/coordinator -run TestFlakeGuardDetection` | Flake guard runs tests twice; detects non-deterministic failure | Detected second-run test timeout and flagged FlakeDetected=true | pass |
| R-005 | test | `go test -v ./internal/coordinator -run TestMergeSimulationClean` | Simulates clean merge without modifying target branch state | Clean merge simulated with zero git state alteration | pass |
| R-006 | test | `go test -v ./internal/coordinator -run TestMergeSimulationConflict` | Detects merge conflicts during simulation and lists conflicting paths | Detected conflict on README.md without repository corruption | pass |
| R-007 | test | `go test -v ./internal/cli -run TestTaskMerge` | `sdd task merge --dry-run` performs full qualification without git mutations | Ran full dry-run qualification report and presentation output | pass |
| R-008 | test | `go test -v ./internal/coordinator -run TestRebaseTaskBranch` | Rebase helper safely rebases task branch onto target branch | Cleanly rebased task branch onto target branch | pass |
| R-009 | test | `go test -v ./internal/cli -run TestTaskMerge` | `sdd task merge --json` outputs structured qualification matrix | Unmarshaled JSON with task_id, ready_to_merge, and qualification fields | pass |
| R-010 | test | `go test -v ./internal/coordinator -run TestGate2PresentationFormatting` | Formats Task Coordinator GATE 2 presentation expecting `APPROVED MERGE T-xxx` | Formatted presentation with commit message and approval token | pass |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `go test -v ./internal/coordinator` | PASS (12 tests passed, 1.976s) |
| `go test -v ./internal/cli -run TestTaskMerge` | PASS (0.44s) |
| `go test -v ./internal/status -run TestStatusMergeReadiness` | PASS (0.402s) |
| `make verify` | PASS (vet clean, all tests passed across all packages) |
| `make check` | PASS (golangci-lint clean with 0 issues, all tests passed) |
| `./bin/sdd check --strict` | PASS (0 errors, 0 warnings, 1 info note) |

## 3. Manual checks
- `./bin/sdd task merge T-031 --dry-run` -> Computed modified packages (`internal/cli`, `internal/coordinator`, `internal/status`), blast radius (`cmd/sdd`), ran regression sweep and flake guard.
- `./bin/sdd status` -> Displays tasks with merge qualification status (`QUALIFIED` or `BLOCKED`).

## 4. Negative and failure cases
- Unresolved 🔴 Blocker in `review.md` -> Qualification fails with `1 unresolved 🔴 Blocker(s)` error.
- Non-deterministic test in suite -> Flake guard catches failure on run 2 and sets `FlakeDetected=true`.
- Git merge conflict during simulation -> Lists exact conflicting files without modifying git branch.
- Unknown task ID in arguments -> Exits with code 1.

## 5. Determinism check
Ran `TestDeterminism` in `internal/coordinator/qualify_test.go`: byte-identical qualification reports across identical runs.

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
Suggested commit message: `feat(coordinator): add pre-merge qualification engine and sdd task merge (T-031)`
Reviewed by: Vikash (human)   Date: 2026-11-05   Reply received: pending