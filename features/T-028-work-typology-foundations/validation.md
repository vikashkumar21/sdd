# Validation — T-028 Work typology foundations: bugfixes, hotfixes, spikes, and task templates

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Passed (2026-11-05)

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/feature -run TestNewAllTypologies` | Directory routing to `features/`, `bugfixes/`, `hotfixes/`, `spikes/` | PASS (routes correctly to all 4 typologies) | pass |
| R-002 | test | `go test -v ./internal/feature -run TestNewAllTypologies` | Templates generated per typology with correct files | PASS (all lifecycle templates verified) | pass |
| R-003 | test | `go test -v ./internal/check -run TestTypologyChecks` | Bugfix template has Step 0 failing test section and check detects unverified Step 0 | PASS (flags unverified Step 0 and passes when verified) | pass |
| R-004 | test | `go test -v ./internal/cli -run TestTaskScaffolding` | CLI supports `bugfix new`, `hotfix new`, `spike new`, `task new` | PASS (all subcommands scaffold corresponding typology) | pass |
| R-005 | test | `go test -v ./internal/feature -run TestNewAllTypologies` | `ROADMAP.md` status updated to `Specifying` | PASS (status transition confirmed) | pass |
| R-006 | test | `go test -v ./internal/check -run TestContextBudgetGuard` | `sdd check` warns when file > 64KB | PASS (emits WARN for 70KB specification document) | pass |
| R-007 | test | `go test -v ./internal/check -run TestTypologyChecks` | `sdd check` validates all 4 typologies | PASS (scans and validates features, bugfixes, hotfixes, spikes) | pass |
| R-008 | test | `go test -v ./internal/upgrade -run TestUpgradeTypologies` | `sdd upgrade` adds new `_templates/` | PASS (creates bugfixes, hotfixes, spikes templates) | pass |
| R-009 | test | `go test -v ./internal/templates` | `templates.Version` is `1.2.0` and matches contract headers | PASS (embedded templates compile, headers match) | pass |
| R-010 | test | `go test -v ./internal/status -run TestStatusTypologies` | `sdd status` detects tasks in all 4 directories | PASS (displays typology and correct gate action) | pass |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | PASS (format check, go vet, all package tests passed) |
| `make check` | PASS (golangci-lint 0 issues, full test suite passed) |
| `go run ./cmd/sdd check --strict` | PASS (0 errors, 0 warnings) |
| Rule 14 temp kit generation & check | PASS (generated 25 files in temp dir; sdd check passed with 0 errors) |

## 3. Manual checks
- `sdd bugfix new T-001 test-bug` -> scaffolds `bugfixes/T-001-test-bug/` with `reproduction.md`, `rca.md`, `plan.md`, `validation.md`, `review.md`.
- `sdd status` -> shows active task with `Typology: feature` and `Folder: features/T-028-work-typology-foundations`.
- File > 64KB in task folder -> `sdd check` displays 64KB `context-budget` warning.

## 4. Negative and failure cases
- Target folder already exists -> exits with error `refusing to overwrite without --force`.
- Unknown typology (`--type unknown`) -> exits 2 with `unknown typology "unknown": must be feature, bugfix, hotfix, or spike`.
- Incomplete task folder or unverified Step 0 reproduction -> `sdd check` flags `bugfix-reproduction` error when `Awaiting review`.

## 5. Determinism check
Ran `sdd status --json` and `sdd check --json` twice; outputs identical: byte-identical verified via `diff -u`.

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
Suggested commit message: `feat(typology): add bugfixes, hotfixes, spikes, and 64KB context guard (T-028)`
Reviewed by: Vikash (human)   Date: 2026-11-05   Reply received: `COMMITTED T-028`
