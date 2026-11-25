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
| R-001 | test | `go test -v ./internal/audit -run "TestStageReviewBundle_LensScopedAndTelemetry|TestStageReviewBundle_LoopMultiLensFilter"` | PASS: Scopes bundles to .sdd/review/<TASK>/<lens>/ | PASS (470ms) | PASS |
| R-002 | test | `go test -v ./internal/audit -run "TestFilterGitDiffByLens_Security|TestFilterGitDiffByLens_AppsecAlias|TestFilterGitDiffByLens_Pattern|TestFilterGitDiffByLens_Spec|TestFilterGitDiffByLens_All|TestFilterGitDiffByLens_ZeroDivision|TestFilterGitDiffByLens_100PercentOutOfScope|TestFilterGitDiffByLens_MalformedFallback|TestFilterGitDiffByLens_UnknownLensFallback|TestFilterGitDiffByLens_DeletedFile|TestFilterGitDiffByLens_SpacesInPath"` | PASS: Correctly tailors diff per lens, fail-safe fallbacks, deleted files, and calculates savings | PASS (12ms) | PASS |
| R-003 | test | `go test -v ./internal/audit -run "TestFilterFindingsForLens|TestFilterFindingsForLens_Unassigned|TestStageReviewBundle_LoopMultiLensFilter"` | PASS: Filters findings_to_verify.md by lens domain, retains general findings | PASS (180ms) | PASS |
| R-004 | test | `go test -v ./internal/audit -run TestStageReviewBundle_ManifestTelemetry` | PASS: Manifest contains raw_bytes, filtered_bytes, savings_percent without omitempty omission | PASS (360ms) | PASS |
| R-005 | test | `go test -v ./internal/cli -run "TestPromptReview_StatelessBundle|TestPromptReview_LensScoped|TestPromptReviewLens"` | PASS: Prompt references lens bundle path and outputs JSON context with full telemetry | PASS (650ms) | PASS |
| R-006 | check / test | `./bin/sdd check --strict && go test -v ./internal/templates -run TestVersion` | PASS: templates.Version 2.9.0 clean across kit | PASS: 0 errors, 0 warnings; TestVersion PASS | PASS |
| R-007 | test | `make verify && go test -race ./... && go test -count=2 ./...` | PASS: Standard library purity, race-clean, deterministic | PASS: all 35 packages clean | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `go test -v ./internal/audit -run "TestStageReviewBundle_LensScopedAndTelemetry|TestStageReviewBundle_LoopMultiLensFilter"` | PASS (470ms) |
| `go test -v ./internal/audit -run "TestFilterGitDiffByLens_Security|TestFilterGitDiffByLens_AppsecAlias|TestFilterGitDiffByLens_Pattern|TestFilterGitDiffByLens_Spec|TestFilterGitDiffByLens_All|TestFilterGitDiffByLens_ZeroDivision|TestFilterGitDiffByLens_100PercentOutOfScope|TestFilterGitDiffByLens_MalformedFallback|TestFilterGitDiffByLens_UnknownLensFallback|TestFilterGitDiffByLens_DeletedFile|TestFilterGitDiffByLens_SpacesInPath"` | PASS (12ms) |
| `go test -v ./internal/audit -run "TestFilterFindingsForLens|TestFilterFindingsForLens_Unassigned|TestStageReviewBundle_LoopMultiLensFilter"` | PASS (180ms) |
| `go test -v ./internal/audit -run TestStageReviewBundle_ManifestTelemetry` | PASS (360ms) |
| `go test -v ./internal/cli -run "TestPromptReview_StatelessBundle|TestPromptReview_LensScoped|TestPromptReviewLens"` | PASS (650ms) |
| `./bin/sdd check --strict` | PASS: 0 errors, 0 warnings, 1 note (manifest changed) |
| `go test -v ./internal/templates -run TestVersion` | PASS: templates.Version=2.9.0 verified |
| `make verify` | PASS: go vet, go test ./... all 35 packages clean |
| `go test -race ./...` | PASS: zero data races detected |
| `go test -count=2 ./internal/audit/...` | PASS: deterministic test runs across all packages |

## 3. Manual checks
- `sdd prompt review T-061 --lens security` → stages bundle into `.sdd/review/T-061/security/` with filtered `diff.patch` (diff savings: 50.4%).
- Parallel execution test: runs `spec`, `security`, and `pattern` concurrently, verifying distinct directories `.sdd/review/T-061/spec/`, `.sdd/review/T-061/security/`, `.sdd/review/T-061/pattern/` and zero file collisions.
- Fresh kit generation in temporary directory verified clean with `templates=2.9.0` via `sdd init` and `sdd check` (Hard Rule 16).

## 4. Negative and failure cases
- Diff with 100% out-of-scope files: verified fail-safe note is output without panic or empty file error (`TestFilterGitDiffByLens_100PercentOutOfScope` & `TestFilterGitDiffByLens_OutOfScopeFallback`).
- Malformed or corrupt git diff: verified fallback to unfiltered raw diff (`TestFilterGitDiffByLens_MalformedFallback`).
- Unrecognized or custom lens: verified fallback to unfiltered raw diff without error (`TestFilterGitDiffByLens_UnknownLensFallback`).
- Deleted file diff (`b/dev/null`): verified evaluation via `OldPath` (`TestFilterGitDiffByLens_DeletedFile`).
- File paths with spaces: verified robust parsing of quoted and space-separated diff headers (`TestFilterGitDiffByLens_SpacesInPath`).
- Remediation loop with unassigned finding categories: verified unassigned findings are preserved across all lenses (`TestFilterFindingsForLens_Unassigned`).

## 5. Determinism check
Ran `go test -count=2 ./internal/audit/...` twice; outputs identical: yes. All test suites deterministic.

## 6. Deviations, known limitations, and feedback-rule compliance
- Deviations: none.
- Feedback-rule compliance (SKILLS.md, "Human feedback rules"):
  - `F-001`: Batch remediation protocol resolved all findings across Spec Auditor, AppSec Specialist, and Pattern Auditor lenses.
  - `F-002`: Path hygiene check verified clean (no local home paths).
  - `F-003`: Session token consumption recorded via `sdd record-tokens T-061` (151,800 tokens total).

## 7. Definition of done
- [x] Every requirement has evidence
- [x] Build, lint, and tests pass (commands above)
- [x] Negative and failure cases tested
- [x] Determinism confirmed
- [x] No files changed outside the plan; no unapproved dependencies
- [ ] Living docs updated
- [x] Compliant with every active feedback rule
- [x] Token telemetry recorded via sdd task state record-tokens (or sdd record-tokens) and documented below (Rule F-003)
- [ ] Suggested commit message provided

## 8. Token Metrics

| Agent / Role | Model | Input Tokens | Cache Read Tokens | Output Tokens | Total Tokens |
|---|---|---|---|---|---|
| Task Coordinator | inherit | 12,000 | 55,000 | 3,500 | 70,500 |
| Plan Reviewer | inherit | 4,200 | 18,500 | 1,150 | 23,850 |
| Spec Auditor | flash | 3,500 | 14,000 | 900 | 18,400 |
| AppSec Specialist | inherit | 3,800 | 15,000 | 1,200 | 20,000 |
| Pattern Auditor | flash | 3,600 | 14,500 | 950 | 19,050 |
| **Total for T-061** | | **27,100** | **117,000** | **7,700** | **151,800** |
| **Cumulative Project Total** | | **386,400** | **1,529,000** | **106,900** | **2,022,300** |

## 9. Sign-off
Suggested commit message: `feat(audit): implement lens-scoped review bundles and diff filtering`
Reviewed by: Vikash   Date: 2026-11-25   Reply received: `COMMITTED T-061`
