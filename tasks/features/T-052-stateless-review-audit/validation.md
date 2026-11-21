---
status: draft
---
# Validation — T-052 Stateless review context isolation and outside validation audit runner

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Complete

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/audit -run TestStagePlanBundle` | PASS: Stages spec.md, rubric.md, manifest.json in .sdd/review/<TASK_ID>/plan/ | PASS (0.02s) | Pass |
| R-002 | test | `go test -v ./internal/audit -run TestStageReviewBundle` | PASS: Stages spec.md, diff.patch, rubric.md, manifest.json in .sdd/review/<TASK_ID>/code/ | PASS (0.12s) | Pass |
| R-003 | test | `go test -v ./internal/audit -run TestStage` | PASS: Generates tabula-rasa prompt with zero conversation baggage for all 4 reviewer roles | PASS (0.14s) | Pass |
| R-004 | test | `go test -v ./internal/cli -run TestPromptReview_StatelessBundle` | PASS: Outputs valid JSON bundle metadata with --json | PASS (0.16s) | Pass |
| R-005 | test | `go test -v ./internal/audit -run TestExtractValidationCommands` | PASS: Parses commands from Section 1 and 2 of validation.md | PASS (0.00s) | Pass |
| R-006 | test | `go test -v ./internal/audit -run TestIsCommandSafe` | PASS: Whitelists safe verification commands; rejects unapproved commands | PASS (0.00s) | Pass |
| R-007 | test | `go test -v ./internal/audit -run TestRunAudit` | PASS: Mechanically executes verification commands and records exit code & duration | PASS (0.04s) | Pass |
| R-008 | test | `go test -v ./internal/cli -run TestReviewAudit_Alias` | PASS: sdd review audit functions identically to sdd task audit | PASS (0.04s) | Pass |
| R-009 | test | `go test -v ./internal/cli -run TestTaskAudit` | PASS: Outputs structured audit report with --json | PASS (0.08s) | Pass |
| R-010 | test | `./bin/sdd check --strict` | PASS: 0 error(s), 0 warning(s) | PASS: 0 error(s), 0 warning(s), 1 note(s) | Pass |

## 2. Commands run

| Command | Result |
|---|---|
| `make build` | PASS: binary built at bin/sdd with version 1.8.0 |
| `make test` | PASS: all packages passed across repository |
| `make vet` | PASS: clean |
| `make lint` | PASS: golangci-lint clean (0 issues) |
| `make check` | PASS: vet, lint, and all tests pass |
| `go test -race ./internal/audit ./internal/cli` | PASS: 0 data races detected |
| `go test -count=2 ./internal/audit ./internal/cli` | PASS: identical test runs confirmed |
| `./bin/sdd check --strict` | PASS: 0 error(s), 0 warning(s), 1 note(s) |

## 3. Manual checks
- Run `./bin/sdd prompt plan-review T-050`: verified frozen bundle staged in `.sdd/review/T-050/plan/` with `spec.md`, `rubric.md`, `manifest.json`.
- Run `./bin/sdd prompt review T-050 --lens spec`: verified frozen bundle staged in `.sdd/review/T-050/code/` with `spec.md`, `diff.patch`, `rubric.md`, `manifest.json`.
- Run `./bin/sdd task audit T-050`: verified 15/15 validation commands executed mechanically outside agent narrative with exit code 0.
- Run `./bin/sdd review audit T-050`: verified command alias functions identically to `sdd task audit`.

## 4. Negative and failure cases
- Command safety violation (`rm -rf ...`, `curl ...`): rejected with exit code 126 in unit test `TestIsCommandSafe` and `TestTaskAudit_SafetyRejection`.
- Shell injection / pipe attempt (`|`, `>`, `;`): rejected with error "contains unapproved shell character".
- Non-existent task ID: rejected cleanly with exit code 1 in `TestTaskAudit_NegativeCases`.
- Missing task ID: rejected with usage error exit code 2 in `TestTaskAudit_NegativeCases`.

## 5. Determinism check
Ran `go test -count=2 ./internal/audit ./internal/cli` twice; outputs identical: yes.
- Run 1: `internal/audit` 0.473s, `internal/cli` 14.653s (all PASS)
- Run 2: `internal/audit` 0.481s, `internal/cli` 14.710s (all PASS)

## 6. Deviations, known limitations, and feedback-rule compliance
- Compliance with Rule F-001 (Resolve all review findings): Compliant. All 6 review findings (5 Warnings, 1 Minor) remediated in unified batch and certified with 0 Blockers in review.md.
- Compliance with Rule F-002 (Zero personal local paths): Compliant. All bundle paths, prompts, and tests use repository-relative paths (`.sdd/review/...`).

## 7. Definition of done
- [x] Every requirement has evidence
- [x] Build, lint, and tests pass (commands above)
- [x] Negative and failure cases tested
- [x] Determinism confirmed
- [x] No files changed outside the plan; no unapproved dependencies
- [x] Living docs updated
- [x] Compliant with every active feedback rule (F-001, F-002)
- [x] Suggested commit message provided

## 8. Sign-off
Suggested commit message: `feat(audit): implement stateless reviewer isolation and outside validation runner (T-052)`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-052`
