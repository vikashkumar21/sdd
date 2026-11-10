# Validation — T-034 Unified tasks/ directory hierarchy and sdd upgrade auto-migration

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Verified (2026-11-10)

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/scaffold -run TestGreenfieldCreatesTheWholeKit` | Fresh kit generates `tasks/<typology>/_templates/` | `PASS: TestGreenfieldCreatesTheWholeKit (0.10s)` | pass |
| R-002 | test | `go test -v ./internal/feature -run TestNew` | Task scaffolding creates `tasks/<typology>/T-xxx-<slug>/` | `PASS: TestNewAllTypologies (0.16s)` | pass |
| R-003 | test | `go test -v ./internal/cli -run TestPromptReview` | Universal task discovery resolves tasks under `tasks/<typology>/` and legacy paths | `PASS: TestPromptReview (0.00s)` | pass |
| R-004 | test | `go test -v ./internal/upgrade -run TestUpgradeLegacyFolderMigration` | `sdd upgrade` migrates legacy root folders to `tasks/` and rewrites manifest | `PASS: TestUpgradeLegacyFolderMigration (0.12s)` | pass |
| R-005 | test | `go test -v ./internal/upgrade -run TestUpgradeDryRunMigration` | `sdd upgrade --dry-run` previews folder migration without disk changes | `PASS: TestUpgradeDryRunMigration (0.13s)` | pass |
| R-006 | test | `go test -v ./internal/check -run "TestTypologyChecks|TestFreshKitHasNoErrors"` | `sdd check` verifies `tasks/<typology>/` layout | `PASS: TestTypologyChecks (0.13s)` | pass |
| R-007 | test | `go test -v ./internal/templates -run TestVersion` | Template version is 1.4.0 and embedded data located in `tasks/` | `PASS: TestVersion (0.00s)` | pass |
| R-008 | test | `go test -v ./internal/check -run "TestDetectsProblems/agent_contract_drift"` | Agent contracts synchronized with `tasks/` and `templates=1.4.0` | `PASS: agent_contract_drift (0.11s)` | pass |
| R-009 | inspection | `ls tasks` in repository root | Repository's own task directories located under `tasks/` | `bugfixes  features  hotfixes  spikes` (root clean) | pass |
| R-010 | test | `make check` and `./bin/sdd check --strict` | 0 errors and 0 warnings | `0 issues; 24 packages passed; 0 error(s), 0 warning(s)` | pass |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | PASS (`go vet`, `go test ./... -count=1` clean across all 24 packages) |
| `make check` | PASS (`golangci-lint run` 0 issues, tests pass) |
| `./bin/sdd check --strict` | PASS (`0 error(s), 0 warning(s), 1 note(s)`) |
| `./bin/sdd status` | PASS (correctly detects active task `tasks/features/T-034-tasks-folder-hierarchy`) |

## 3. Manual checks
- `sdd feature new T-035 plan-review-subagent` in a test kit -> scaffolds under `tasks/features/T-035-plan-review-subagent/`.
- `sdd upgrade` on a legacy test project with root `features/` -> moves to `tasks/features/` and updates `.sdd/manifest.json`.

## 4. Negative and failure cases
- Attempting migration when destination directory `tasks/<typology>/` already exists with colliding files -> safely stops and reports conflict without overwriting (`TestUpgradeTypologies`).
- `sdd prompt review non-existent task ID` -> exits with code 1 (`TestPromptReview`).

## 5. Determinism check
Running migration on the same repository fixture twice produces byte-identical manifest and directory tree: verified by `TestUpgradeMigrationIsDeterministic` (PASS 0.22s).

## 6. Deviations, known limitations, and feedback-rule compliance
- Deviations: none.
- Feedback-rule compliance (SKILLS.md, "Human feedback rules"): none active.

## 7. Definition of done
- [x] Every requirement has evidence
- [x] Build, lint, and tests pass
- [x] Negative and failure cases tested
- [x] Determinism confirmed
- [x] No files changed outside the plan; no unapproved dependencies
- [x] Independent subagent audit completed in review.md with 0 Blockers
- [x] Living docs updated
- [x] Compliant with every active feedback rule
- [x] Suggested commit message provided

## 8. Sign-off
Suggested commit message: `feat(typology): unify tasks under tasks/ folder hierarchy and auto-migrate legacy projects (T-034)`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-034`
