---
status: draft
---
# Validation — T-055 Default task token telemetry and template integration

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Complete

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | inspection | `git diff internal/templates/data/tasks tasks/` | Starter templates for features, bugfixes, hotfixes, spikes contain standard Token Metrics table and F-003 checklist item | All 8 starter templates across 4 typologies updated with Token Metrics table and F-003 DoD item | PASS |
| R-002 | test | `./bin/sdd check --strict` | Agent contracts match across AGENTS.md, GEMINI.md, CLAUDE.md and contract.md.tmpl with token telemetry requirements | Contract parity clean, strict check 0 errors | PASS |
| R-003 | test | `go test -v ./internal/cli -run TestTopLevelRecordTokens` | `sdd record-tokens <T-xxx>` CLI shorthand executes cleanly with flags, stdin, and json | Top-level command and task state alias pass with 0 errors | PASS |
| R-004 | test | `go test -v ./internal/taskpr -run TestPackageTaskPR` | `sdd task pr` embeds token telemetry section when state has recorded tokens | Token telemetry section embedded in PR markdown when present; omitted when absent | PASS |
| R-005 | test | `go test -v ./internal/check -run TestCheckReviewDocument` | `checkReviewDocument` ignores YAML frontmatter without throwing false-positive status errors | Frontmatter skipped cleanly, 0 false-positive findings | PASS |
| R-006 | test | `go test -v ./internal/templates ./internal/upgrade` | templates.Version is 2.2.0, kit generation and sdd upgrade pass strict validation | templates.Version=2.2.0 verified, all upgrade migrations pass | PASS |
| R-007 | test | `make verify` | 100% standard library purity, race detection clean, tests pass | make verify clean, 0 race conditions, 0 issues | PASS |

## 2. Commands run

| Command | Result |
|---|---|
| `make build` | PASS (bin/sdd compiled, templates 2.2.0) |
| `make test` | PASS (all 33 packages passed with 0 failures) |
| `make lint` | PASS (golangci-lint run: 0 issues) |
| `./bin/sdd check --strict` | PASS (0 error(s), 0 warning(s), 1 note(s)) |

## 3. Manual checks
- Verified `./bin/sdd record-tokens T-055 --input 100 --cache 200 --output 50` outputs: `Recorded tokens for T-055: input=+100, cache=+200, output=+50 (totals: input=100, cache=200, output=50, total=350)`.
- Verified `./bin/sdd task pr T-055 --dry-run` includes `## Token Telemetry` with exact numbers in generated markdown.
- Verified running `./bin/sdd feature new T-003 sample` generates `validation.md` with `## 8. Token Metrics` table and F-003 DoD item.
- Outside mechanical verification audit (`./bin/sdd task audit T-055`): PASS.

## 4. Negative and failure cases
- `sdd record-tokens` with negative token values fails with code 1 (`token values cannot be negative`).
- `sdd record-tokens` without task ID fails with usage error (code 2: `usage: sdd record-tokens <T-xxx>`).
- `sdd record-tokens` with nonexistent task directory fails with clear error.
- `review.md` with `status: draft` in frontmatter and `Status: Pending` in body correctly parses status as `pending` (verified in `TestCheckReviewDocument_FrontmatterHandling`).

## 5. Determinism check
- Ran `go test -count=2 ./internal/taskpr ./internal/check ./internal/templates ./internal/upgrade`: PASS (identical outputs across runs).
- Ran `go test -race ./internal/taskpr ./internal/templates ./internal/cli -run "TestRecordTokens|TestTopLevelRecordTokens|TestTaskState"`: PASS (0 race conditions).
- Ran `go test -race ./internal/taskpr ./internal/templates ./internal/check -run "TestPackageTaskPR|TestCheckReviewDocument"`: PASS (0 race conditions).

## 6. Deviations, known limitations, and feedback-rule compliance
- Deviations: none.
- Feedback-rule compliance:
  - `F-001`: Resolve all findings across Spec Auditor, AppSec Specialist, and Pattern Auditor before Gate 2 close.
  - `F-002`: Verified zero developer-specific absolute paths in new templates, tests, or documentation.
  - `F-003`: Track and record session token consumption via `sdd record-tokens T-055`, document metrics below, and report at Gate 2.

## 7. Definition of done
- [x] Every requirement has evidence
- [x] Build, lint, and tests pass (commands above)
- [x] Negative and failure cases tested
- [x] Determinism confirmed
- [x] No files changed outside the plan; no unapproved dependencies
- [x] Living docs updated
- [x] Compliant with every active feedback rule (F-001, F-002, F-003)
- [x] Token telemetry recorded via `sdd record-tokens` and documented below
- [x] Suggested commit message provided

## 8. Token Metrics

| Agent / Role | Model | Input Tokens | Cache Read Tokens | Output Tokens | Total Tokens |
|---|---|---|---|---|---|
| Task Coordinator | inherit | 1,581,425 | 29,481,879 | 74,740 | 31,138,044 |
| **Total for T-055** | | **1,581,425** | **29,481,879** | **74,740** | **31,138,044** |
| **Cumulative Project Total** | | **2,648,267** | **55,502,833** | **181,063** | **58,332,163** |

## 9. Sign-off
Suggested commit message: `feat(telemetry): default task token telemetry, record-tokens shorthand, and template version 2.2.0 (T-055)`
Reviewed by: Vikash (human)   Date: 2026-11-22   Reply received: `COMMITTED T-055`
