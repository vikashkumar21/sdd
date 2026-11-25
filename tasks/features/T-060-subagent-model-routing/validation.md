---
status: completed
---
# Validation — T-060 Subagent model routing, tier enforcement, and calibrated reasoning dispatch

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Passed

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | inspection | `git grep -n frontier skills/task-coordinator/SKILL.md` | Model capability tier mappings defined for frontier, advanced, standard, and fast | frontier, advanced, standard, and fast mapped in Section 5 | PASS |
| R-002 | inspection | `git grep -n Model: skills/task-coordinator/SKILL.md` | Coordinator skill mandates Model="flash" for spec-auditor, pattern-auditor, qa, devops and Model="inherit" for appsec-specialist and plan-reviewer | Model="flash" mandated for standard roles, Model="inherit" for frontier/advanced | PASS |
| R-003 | test | `./bin/sdd check --strict` | Agent contracts match across AGENTS.md, GEMINI.md, CLAUDE.md and contract.md.tmpl with tier-calibrated dispatch rules | 0 errors, 0 warnings, contract parity verified across all files | PASS |
| R-004 | test | `go test -v ./internal/cli -run TestPromptReview_ModelRouting` | `sdd prompt review`, `sdd prompt plan-review`, and `sdd prompt implement` output executable `invoke_subagent` snippets with Model parameter; fallback boundaries tested | TestPromptReview_ModelRouting, TestPromptPlanReview_ModelRouting, TestPromptImplement_ModelRouting, TestResolveSubagentModelParam, TestRoleTitleFor pass | PASS |
| R-005 | test | `go test -v ./internal/check -run TestCheckCoordinatorTierDispatch` | Mechanical rule `coordinator-tier-dispatch` validates coordinator skills and contracts declare model tier dispatch for all Review Triad roles | 6/6 tests pass in 0.413s; missing roles or Model parameters flag errors; legacy absence passes cleanly | PASS |
| R-006 | test | `go test -v ./internal/templates ./internal/upgrade` | templates.Version is 2.8.0, kit generation and upgrade migrations pass | TestVersion confirms 2.8.0, TestFreshKitHasNoErrors, TestUpgradeAlreadyUpToDate, and TestUpgradeDryRunMigration pass | PASS |
| R-007 | test | `make verify` | 100% standard library purity, race detection clean, tests pass | All 35 packages pass vet and tests; race detector clean; zero external dependencies | PASS |

## 2. Commands run

| Command | Result |
|---|---|
| `make build` | PASS (compiled bin/sdd cleanly) |
| `make test` | PASS (all unit and integration tests pass in 19.298s) |
| `go vet ./...` | PASS (clean) |
| `./bin/sdd check --strict` | PASS (0 errors, 0 warnings, 1 note) |
| `go test -race ./internal/check ./internal/cli ./internal/templates ./internal/upgrade` | PASS (0 data races detected) |

Note on static analysis: package-scoped `golangci-lint run ./internal/check/... ./internal/cli/... ./internal/templates/...` passed with 0 issues. Full-repo `make lint` fails on 4 pre-existing static analysis findings in `internal/hook/hook.go` (from T-051), detailed in Section 6.

## 3. Manual checks
- Verify running `./bin/sdd prompt review T-060 --lens spec` displays `invoke_subagent(Role="Independent Spec Auditor", Model="flash", ...)`: PASS (confirmed).
- Verify running `./bin/sdd prompt review T-060 --lens security` displays `invoke_subagent(Role="Independent AppSec Specialist", Model="inherit", ...)`: PASS (confirmed).
- Outside mechanical verification audit (`./bin/sdd task audit T-060`): PASS (11/11 verification commands passed in 43.855s).

## 4. Negative and failure cases
- Coordinator skill omitting `Model` for standard tier roles triggers `coordinator-tier-dispatch` warning/error in `sdd check`: PASS (`TestCheckCoordinatorTierDispatch_SkillMissingFlash`).
- Coordinator skill omitting any Review Triad role triggers `coordinator-tier-dispatch` error: PASS (`TestCheckCoordinatorTierDispatch_SkillMissingRole`).
- Contract omitting any Review Triad role triggers `coordinator-tier-dispatch` error: PASS (`TestCheckCoordinatorTierDispatch_ContractMissingRole`).
- Contract omitting tier-calibrated dispatch triggers `coordinator-tier-dispatch` error: PASS (`TestCheckCoordinatorTierDispatch_ContractMissingTierDispatch`).
- Kits lacking `skills/task-coordinator/SKILL.md` (e.g. `dbguard-kit`) pass `sdd check` cleanly without spurious `coordinator-tier-dispatch` errors: PASS (`TestCheckCoordinatorTierDispatch_MissingSkill_PassesCleanly`).
- Unrecognized or malformed tier string in `resolveSubagentModelParam` falls back safely to `"inherit"`: PASS (`TestResolveSubagentModelParam`).
- Empty or whitespace role in `roleTitleFor` falls back safely to `"Subagent"`: PASS (`TestRoleTitleFor`).

## 5. Determinism check
- Ran `go test -count=2 ./internal/check ./internal/cli ./internal/templates` twice; outputs identical: PASS (all tests pass deterministically).

## 6. Deviations, known limitations, and feedback-rule compliance
- Deviations: None. Repo-wide `make lint` fails on 4 pre-existing static analysis findings in `internal/hook/hook.go` (lines 233, 268, 477, 488) introduced in T-051; package-scoped linting for all T-060 packages (`./internal/check/...`, `./internal/cli/...`, `./internal/templates/...`) passes with 0 issues.
- Feedback-rule compliance:
  - `F-001`: Resolve all review findings (Blockers, Warnings, Minors) across Review Triad before Gate 2 close.
  - `F-002`: Zero personal absolute filesystem paths in generated code, templates, or documentation.
  - `F-003`: Track and record session token consumption via `sdd record-tokens T-060`, document metrics below, and report at Gate 2.

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
| Task Coordinator | inherit | tracked | tracked | tracked | tracked |
| Plan Reviewer (Loops 1 & 2) | inherit | 245,044 | 3,963,539 | 31,731 | 4,240,314 |
| Spec Auditor (Loops 1 & 2) | flash | 453,320 | 6,468,432 | 35,819 | 6,957,571 |
| AppSec Specialist (Loops 1 & 2) | inherit | 401,490 | 6,402,910 | 28,237 | 6,832,637 |
| Pattern Auditor (Loops 1 & 2) | flash | 322,095 | 3,964,841 | 25,886 | 4,312,822 |
| **Total Subagents for T-060** | | **1,421,949** | **20,799,722** | **121,673** | **22,343,344** |
| **Recorded Task Telemetry** | | **1,421,949** | **20,799,722** | **121,673** | **22,343,344** |

## 9. Sign-off
Suggested commit message: `feat(orchestration): subagent model routing, tier enforcement, and calibrated reasoning dispatch (T-060)`
Reviewed by: Vikash (human)   Date: 2026-11-25   Reply received: `COMMITTED T-060`
