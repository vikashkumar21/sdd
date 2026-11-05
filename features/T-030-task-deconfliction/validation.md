# Validation — T-030 Static task deconfliction and plan overlap analyzer

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Verified
Date: 2026-11-05

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/deconflict -run TestExtractPlanFiles` | Parses Section 5 of `plan.md` across typologies and extracts file paths | Extracted 4 canonical paths ignoring markdown formatting and `./` prefixes | pass |
| R-002 | test | `go test -v ./internal/cli -run TestTaskDeconflict` | `sdd task deconflict` analyzes specified or active tasks | Ran both positional pair and active task auto-discovery cleanly | pass |
| R-003 | test | `go test -v ./internal/deconflict -run TestDisjointTasksParallel` | Disjoint tasks classified as safe for parallel execution | Classified as SAFE with 0 conflicts and placed in Batch 1 | pass |
| R-004 | test | `go test -v ./internal/deconflict -run TestOverlappingTasksSequential` | Overlapping tasks detected with exact shared files | Detected collision hazard on `internal/cli/cli.go` and scheduled sequential waves | pass |
| R-005 | test | `go test -v ./internal/deconflict -run TestConcurrencyBatchScheduling` | Partitions tasks into parallel execution batches | Scheduled into Wave 1 (parallel batch) and Wave 2 (deferred queue) | pass |
| R-006 | test | `go test -v ./internal/cli -run TestTaskDeconflictJSON` | Outputs structured JSON via `--json` | Valid JSON unmarshaled with `has_conflict`, `conflicts`, and `batches` | pass |
| R-007 | test | `go test -v ./internal/status -run TestStatusTaskOverlap` | `sdd status` warns on active task file collisions | Emitted warning for overlapping active in-progress tasks | pass |
| R-008 | test | `go test -v ./internal/check -run TestCheckTaskOverlap` | `sdd check` warns if concurrent tasks share planned files | Emitted `task-overlap` warning when active tasks share planned files | pass |
| R-009 | test | `go test -v ./internal/deconflict -run TestEdgeCases` | Handles missing plans, empty tables, and single tasks gracefully | Handled missing plans, empty tables, and single-task queries safely | pass |
| R-010 | test | `go test -v ./internal/deconflict -run TestDeterminism` | Output is strictly deterministic across runs | Identical output regardless of plan ordering | pass |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `go test -v ./internal/deconflict` | PASS (8 tests passed, 0.057s) |
| `go test -v ./internal/cli -run TestTaskDeconflict` | PASS (TestTaskDeconflict and TestTaskDeconflictJSON passed, 0.424s) |
| `go test -v ./internal/status -run TestStatusTaskOverlap` | PASS (passed, 0.077s) |
| `go test -v ./internal/check -run TestCheckTaskOverlap` | PASS (passed, 0.11s) |
| `make verify` | PASS (vet clean, all tests passed across all packages) |
| `make check` | PASS (vet clean, golangci-lint 0 issues, all tests passed) |
| `./bin/sdd check --strict` | PASS (0 errors, 0 warnings, 1 info note) |

## 3. Manual checks
- `./bin/sdd task deconflict T-029 T-030` -> Successfully detected COLLISION HAZARD on 8 shared files (`README.md`, `ROADMAP.md`, `internal/check/check_test.go`, `internal/cli/cli.go`, `internal/cli/cli_test.go`, `internal/cli/task.go`, `internal/status/status.go`, `internal/status/status_test.go`) and scheduled Wave 1 (T-029) and Wave 2 (T-030).
- `./bin/sdd status` -> Displays active task T-030 cleanly in Phase B.

## 4. Negative and failure cases
- `sdd task deconflict T-INVALID-1 T-INVALID-2` -> Exits with code 1: `task T-INVALID-1 not found in ROADMAP.md or task folders`.
- `sdd task deconflict T-030` -> Exits with code 2: `specify at least 2 tasks to analyze for conflicts, or use --all to analyze all roadmap tasks`.
- Missing `plan.md` in task directory -> Returns empty file list without error.

## 5. Determinism check
Ran `./bin/sdd task deconflict --all --json` twice; diffed outputs: byte-identical (0 diff).

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
Suggested commit message: `feat(deconflict): add static plan overlap analyzer and concurrency scheduler (T-030)`
Reviewed by: Vikash (human)   Date: 2026-11-05   Reply received: `COMMITTED T-030`
