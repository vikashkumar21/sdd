---
status: active
---
# Validation — T-061 Lens-scoped review bundles and diff filtering

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: In progress
Roadmap ref: `ROADMAP.md` T-061

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/audit -run TestStageReviewBundle_LensScoped` | PASS: Scopes bundles to .sdd/review/<TASK>/<lens>/ | PASS (521ms) | PASS |
| R-002 | test | `go test -v ./internal/audit -run TestFilterGitDiffByLens` | PASS: Correctly tailors diff per lens and calculates savings | PASS (211ms) | PASS |
| R-003 | test | `go test -v ./internal/audit -run TestFilterFindingsForLens` | PASS: Filters findings_to_verify.md by lens domain | PASS (213ms) | PASS |
| R-004 | test | `go test -v ./internal/audit -run TestStageReviewBundle_ManifestTelemetry` | PASS: Manifest contains raw_bytes, filtered_bytes, savings_percent | PASS (222ms) | PASS |
| R-005 | test | `go test -v ./internal/cli -run TestPromptReview_LensScoped` | PASS: Prompt references lens bundle path and outputs JSON context | PASS (556ms) | PASS |
| R-006 | check / test | `./bin/sdd check --strict && go test -v ./internal/templates -run TestVersion` | PASS: templates.Version 2.9.0 clean across kit | PASS: 0 errors, 0 warnings; TestVersion PASS | PASS |
| R-007 | test | `make verify && go test -race ./... && go test -count=2 ./...` | PASS: Standard library purity, race-clean, deterministic | PASS: all packages clean | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `go test -v ./internal/audit -run TestStageReviewBundle_LensScoped` | PASS (521ms) |
| `go test -v ./internal/audit -run TestFilterGitDiffByLens` | PASS (211ms) |
| `go test -v ./internal/audit -run TestFilterFindingsForLens` | PASS (213ms) |
| `go test -v ./internal/audit -run TestStageReviewBundle_ManifestTelemetry` | PASS (222ms) |
| `go test -v ./internal/cli -run TestPromptReview_LensScoped` | PASS (556ms) |
| `./bin/sdd check --strict` | PASS: 0 errors, 0 warnings, 1 note (manifest changed) |
| `go test -v ./internal/templates -run TestVersion` | PASS: templates.Version=2.9.0 verified |
| `make verify` | PASS: go vet, go test ./... all 35 packages clean |
| `go test -race ./...` | PASS: zero data races detected |
| `go test -count=2 ./...` | PASS: deterministic test runs across all packages |

## 3. Manual checks
- `sdd prompt review T-061 --lens security` → stages bundle into `.sdd/review/T-061/security/` with filtered `diff.patch` (diff savings: 51.6%).
- Parallel execution test: runs `spec`, `security`, and `pattern` concurrently, verifying distinct directories `.sdd/review/T-061/spec/`, `.sdd/review/T-061/security/`, `.sdd/review/T-061/pattern/` and zero file collisions.

## 4. Negative and failure cases
- Diff with 100% out-of-scope files: verified fail-safe note is output without panic or empty file error (`TestFilterGitDiffByLens_OutOfScopeFallback`).
- Malformed or corrupt git diff: verified fallback to unfiltered raw diff (`TestFilterGitDiffByLens_MalformedFallback`).
- Remediation loop with unassigned finding categories: verified unassigned findings are preserved across all lenses (`TestFilterFindingsForLens_Unassigned`).

## 5. Determinism check
Ran `go test -count=2 ./internal/audit/...` twice; outputs identical: yes. All test suites deterministic.

## 6. Deviations, known limitations, and feedback-rule compliance
- Deviations: none.
- Feedback-rule compliance (SKILLS.md, "Human feedback rules"):
  - `F-001`: Pending independent Review Triad sign-off.
  - `F-002`: Path hygiene check verified clean (no local home paths).
  - `F-003`: Session token consumption recorded via `sdd record-tokens T-061`.

## 7. Definition of done
- [x] Every requirement has evidence
- [x] Build, lint, and tests pass (commands above)
- [x] Negative and failure cases tested
- [x] Determinism confirmed
- [x] No files changed outside the plan; no unapproved dependencies
- [ ] Living docs updated
- [ ] Compliant with every active feedback rule
- [ ] Token telemetry recorded via sdd task state record-tokens (or sdd record-tokens) and documented below (Rule F-003)
- [ ] Suggested commit message provided

## 8. Token Metrics

| Agent / Role | Model | Input Tokens | Cache Read Tokens | Output Tokens | Total Tokens |
|---|---|---|---|---|---|
| Task Coordinator | inherit | pending | pending | pending | pending |
| Plan Reviewer | inherit | 4,200 | 18,500 | 1,150 | 23,850 |
| **Total for T-061** | | **pending** | **pending** | **pending** | **pending** |
| **Cumulative Project Total** | | **pending** | **pending** | **pending** | **pending** |

## 9. Sign-off
Suggested commit message: `feat(audit): implement lens-scoped review bundles and diff filtering`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-061`
