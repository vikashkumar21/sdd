---
status: draft
---
# Validation — T-061 Lens-scoped review bundles and diff filtering

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Not run | In progress | Complete

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/audit -run TestStageReviewBundle_LensScoped` | PASS: Scopes bundles to .sdd/review/<TASK>/<lens>/ | pending | NOT VERIFIED |
| R-002 | test | `go test -v ./internal/audit -run TestFilterGitDiffByLens` | PASS: Correctly tailors diff per lens and calculates savings | pending | NOT VERIFIED |
| R-003 | test | `go test -v ./internal/audit -run TestFilterFindingsForLens` | PASS: Filters findings_to_verify.md by lens domain | pending | NOT VERIFIED |
| R-004 | test | `go test -v ./internal/audit -run TestStageReviewBundle_ManifestTelemetry` | PASS: Manifest contains raw_bytes, filtered_bytes, savings_percent | pending | NOT VERIFIED |
| R-005 | test | `go test -v ./internal/cli -run TestPromptReview_LensScoped` | PASS: Prompt references lens bundle path and outputs JSON context | pending | NOT VERIFIED |
| R-006 | check / test | `./bin/sdd check --strict && go test -v ./internal/templates -run TestVersion` | PASS: templates.Version 2.9.0 clean across kit | pending | NOT VERIFIED |
| R-007 | test | `make verify && go test -race ./... && go test -count=2 ./...` | PASS: Standard library purity, race-clean, deterministic | pending | NOT VERIFIED |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | pending |
| `go test -race ./...` | pending |
| `go test -count=2 ./...` | pending |
| `./bin/sdd check --strict` | pending |
| `./bin/sdd task audit T-061` | pending |

## 3. Manual checks
- `sdd prompt review T-061 --lens security` → stages bundle into `.sdd/review/T-061/security/` with filtered `diff.patch`.
- Parallel execution test: runs `spec`, `security`, and `pattern` concurrently, verifying distinct directories and zero file collisions.

## 4. Negative and failure cases
- Diff with 100% out-of-scope files: verifies fail-safe note is output without panic or empty file error.
- Malformed or corrupt git diff: verifies fallback to unfiltered raw diff.
- Remediation loop with unassigned finding categories: verifies unassigned findings are preserved across all lenses.

## 5. Determinism check
Ran `go test -count=2 ./internal/audit/...` twice; outputs identical: yes / no. Evidence: pending.

## 6. Deviations, known limitations, and feedback-rule compliance
- Deviations: none.
- Feedback-rule compliance (SKILLS.md, "Human feedback rules"):
  - `F-001`: Pending independent Review Triad sign-off.
  - `F-002`: Path hygiene check verified clean (no local home paths).
  - `F-003`: Session token consumption recorded via `sdd record-tokens T-061`.

## 7. Definition of done
- [ ] Every requirement has evidence
- [ ] Build, lint, and tests pass (commands above)
- [ ] Negative and failure cases tested
- [ ] Determinism confirmed
- [ ] No files changed outside the plan; no unapproved dependencies
- [ ] Living docs updated
- [ ] Compliant with every active feedback rule
- [ ] Token telemetry recorded via sdd task state record-tokens (or sdd record-tokens) and documented below (Rule F-003)
- [ ] Suggested commit message provided

## 8. Token Metrics

| Agent / Role | Model | Input Tokens | Cache Read Tokens | Output Tokens | Total Tokens |
|---|---|---|---|---|---|
| Task Coordinator | inherit | pending | pending | pending | pending |
| Plan Reviewer | inherit | pending | pending | pending | pending |
| **Total for T-061** | | **pending** | **pending** | **pending** | **pending** |
| **Cumulative Project Total** | | **pending** | **pending** | **pending** | **pending** |

## 9. Sign-off
Suggested commit message: `feat(audit): implement lens-scoped review bundles and diff filtering`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-061`
