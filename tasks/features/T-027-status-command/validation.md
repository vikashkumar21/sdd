# Validation — T-027 sdd status: summarise roadmap, task progress, and gate status

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Complete

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/cli -run TestStatusCLI` | Prints human-readable summary of active tasks and milestones | PASS: TestStatusCLI (0.00s) | PASS |
| R-002 | test | `go test -v ./internal/status -run TestScanMissingRoadmap` | Exits 1 with error when ROADMAP.md is missing | PASS: TestScanMissingRoadmap (0.00s) | PASS |
| R-003 | test | `go test -v ./internal/status -run TestScanActiveTasksAndGates` | Identifies active tasks, titles, statuses, phases, and folders | PASS: TestScanActiveTasksAndGates (0.00s) | PASS |
| R-004 | test | `go test -v ./internal/status -run TestScanGate2WhenAwaitingReview` | Correctly resolves GATE 1, GATE 2, and next task suggestions | PASS: TestScanGate2WhenAwaitingReview (0.00s) | PASS |
| R-005 | test | `go test -v ./internal/status -run TestScanActiveTasksAndGates` | Computes tasks done, total, percent, and milestone status | PASS: TestScanActiveTasksAndGates (0.00s) | PASS |
| R-006 | test | `go test -v ./internal/status -run TestScanActiveTasksAndGates` | Accurately aggregates total, done, in-progress, blocked, not started | PASS: TestScanActiveTasksAndGates (0.00s) | PASS |
| R-007 | test | `go test -v ./internal/cli -run TestStatusCLI` | Outputs valid 2-space indented JSON matching ProjectStatus schema | PASS: TestStatusCLI (0.00s) | PASS |
| R-008 | test | `go test -v ./internal/status -run TestScanReviewBlockersWarning` | Surfaces warnings for multiple active tasks, review blockers, and fact expiry | PASS: TestScanReviewBlockersWarning (0.00s) | PASS |
| R-009 | test | `go test -v ./internal/status -run TestDeterminism` | Running status scan twice produces byte-identical JSON | PASS: TestDeterminism (0.00s) | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | PASS (`go vet ./...` clean, all 23 packages pass) |
| `make check` | PASS (`golangci-lint` clean with 0 issues, all tests pass) |
| `go run ./cmd/sdd check --strict` | PASS (0 errors, 0 warnings, 1 note) |
| `go run ./cmd/sdd status` | PASS (renders clean terminal overview of M7 and T-027) |
| `go run ./cmd/sdd status --json` | PASS (renders indented JSON conforming to `ProjectStatus`) |

## 3. Manual checks
- `go run ./cmd/sdd status` in repository root -> outputs clear summary of T-027 in Implementing, M7 at 50%, and all milestone completion indicators.
- `go run ./cmd/sdd status --json` -> outputs valid JSON with active milestone, active task, and task summary.

## 4. Negative and failure cases
- Missing `ROADMAP.md` -> tested in `TestScanMissingRoadmap` and `TestStatusCLI`; reports error and exits 1.
- Unexpected positional arguments (`sdd status unexpected`) -> tested in `TestStatusCLI`; exits 2.
- Unknown flags (`sdd status --unknown-flag`) -> tested in `TestStatusCLI`; exits 2.
- Unresolved 🔴 Blockers in `review.md` -> tested in `TestScanReviewBlockersWarning`; surfaced under warnings.
- Expired verified facts in `TECH_STACK.md` -> tested in `TestScanFactExpiryWarning`; surfaced under warnings.
- Multiple active tasks -> tested in `TestMultipleActiveTasksWarning`; surfaced under warnings.

## 5. Determinism check
Ran `go run ./cmd/sdd status --json` twice; outputs identical: yes.
Evidence: `diff -u /tmp/sdd_status_1.json /tmp/sdd_status_2.json` produced zero diff.

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
Suggested commit message: `feat(status): add sdd status command with milestone and gate summary (T-027)`
Reviewed by: Vikash (human)   Date: 2026-11-05   Reply received: `COMMITTED T-027`
