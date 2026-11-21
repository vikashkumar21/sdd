---
status: draft
---
# Plan — T-055 Default task token telemetry and template integration

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

## 1. Summary

Incorporate task token telemetry into SDD defaults:
1. Update starter templates across all 4 typologies (`features`, `bugfixes`, `hotfixes`, `spikes`) to include `## Token Metrics` tables and Rule F-003 compliance items in the Definition of Done.
2. Update agent operating contracts (`contract.md.tmpl`, `AGENTS.md`, `GEMINI.md`, `CLAUDE.md`) in Section 2 (Phase D & Gate 2), Section 7 (Communication format), and Section 8 (Definition of Done) to mandate recording session token consumption and presenting it at Gate 2 per Rule F-003.
3. Add top-level shorthand command `sdd record-tokens <T-xxx>` in `internal/cli/cli.go` routing to `runTaskStateRecordTokens`.
4. Update `internal/taskpr/taskpr.go` to surface token telemetry in PR bodies and JSON bundles.
5. Fix frontmatter handling in `internal/check/review.go` to ignore `--- ... ---` YAML blocks when detecting markdown review status.
6. Bump `templates.Version` from `2.1.0` to `2.2.0` in `internal/templates/templates.go`, update test fixtures and `.sdd/manifest.json`, and verify with `sdd check --strict` and `sdd upgrade` tests.

## 2. Architecture & Design Principles

- **Standard Library Only (`D-007`):** Zero external dependencies.
- **Fail Safe & Deterministic (`D-003`):** Unambiguous CLI parsing, sorted outputs, deterministic PR markdown formatting.
- **Contract Parity (§9):** Every change to `contract.md.tmpl` is mirrored exactly across `AGENTS.md`, `GEMINI.md`, and `CLAUDE.md`.
- **Template Versioning (Rule 16):** Any template modification bumps `templates.Version` and must be validated via clean upgrade simulation.

## 3. Decisions

### D-047: Default task token telemetry, record-tokens command, and template integration
- **Context:** Human feedback rule F-003 requires tracking session token consumption and reporting at Gate 2. Currently, templates and contracts do not mandate or scaffold token metrics by default.
- **Option A (Recommended):** Integrate `## Token Metrics` directly into starter templates, mandate token recording and Gate 2 reporting in agent contracts, provide `sdd record-tokens` shorthand CLI command, enrich `sdd task pr` with token telemetry, and bump `templates.Version` to `2.2.0`.
- **Option B:** Leave templates and contracts untouched; rely on manual recollection and custom table authoring.
- **Recommendation:** Option A. Seamless ergonomics, consistent metrics across all future tasks, and native contract enforcement.

## 4. Step-by-Step Implementation

| Step | Action | Files | Verification |
|---|---|---|---|
| 1 | Update task starter templates in `internal/templates/data/tasks/` (`features`, `bugfixes`, `hotfixes`, `spikes`) and repository `tasks/*/_templates/` with standard `## Token Metrics` tables and Rule F-003 in Definition of Done. | `internal/templates/data/tasks/**`, `tasks/*/_templates/**` | `git diff internal/templates/data/tasks tasks/` |
| 2 | Update contract template `internal/templates/data/base/contract.md.tmpl` and agent files `AGENTS.md`, `GEMINI.md`, `CLAUDE.md` in Sections 2, 7, and 8 to mandate `record-tokens` and Gate 2 presentation. | `internal/templates/data/base/contract.md.tmpl`, `AGENTS.md`, `GEMINI.md`, `CLAUDE.md` | `./bin/sdd check --strict` |
| 3 | Add top-level command `sdd record-tokens` in `internal/cli/cli.go` and `internal/cli/task_state.go`. Add unit tests in `internal/cli/task_state_test.go` and `internal/cli/cli_test.go`. | `internal/cli/cli.go`, `internal/cli/task_state.go`, `internal/cli/task_state_test.go`, `internal/cli/cli_test.go`, `README.md` | `go test -v ./internal/cli -run TestRecordTokens` |
| 4 | Enrich `internal/taskpr/taskpr.go` to load `.sdd/state/<T-xxx>/state.json` token telemetry, include `## Token Telemetry` section in `FormatPRMarkdown`, and add `Tokens *state.TokenTelemetry` to `PRBundle`. Author tests in `internal/taskpr/taskpr_test.go`. | `internal/taskpr/taskpr.go`, `internal/taskpr/taskpr_test.go` | `go test -v ./internal/taskpr` |
| 5 | Enhance `internal/check/review.go` to ignore YAML frontmatter when parsing `review.md`. Add test case in `internal/check/review_test.go` or `check_test.go`. | `internal/check/review.go`, `internal/check/review_test.go` | `go test -v ./internal/check -run TestReview` |
| 6 | Bump `templates.Version` to `2.2.0` in `internal/templates/templates.go`. Update `internal/templates/templates_test.go`, `.sdd/manifest.json`, and contract headers to `templates=2.2.0`. Verify upgrade tests in `internal/upgrade/upgrade_test.go`. | `internal/templates/templates.go`, `internal/templates/templates_test.go`, `.sdd/manifest.json`, `internal/upgrade/upgrade_test.go` | `go test -v ./internal/templates ./internal/upgrade` |
| 7 | Run verification suite: `make fmt`, `make lint`, `make test`, and `sdd check --strict`. | Repository files | `make verify && ./bin/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/templates/templates.go` | modify | Bump templates.Version to 2.2.0 |
| `internal/templates/templates_test.go` | modify | Update Version test assertion to 2.2.0 |
| `internal/templates/data/base/contract.md.tmpl` | modify | Add token telemetry recording and Gate 2 reporting |
| `internal/templates/data/tasks/features/validation.md` | modify | Add Token Metrics table and F-003 DoD item |
| `internal/templates/data/tasks/bugfixes/validation.md` | modify | Add Token Metrics table and F-003 DoD item |
| `internal/templates/data/tasks/hotfixes/validation.md` | modify | Add Token Metrics table and F-003 DoD item |
| `internal/templates/data/tasks/spikes/findings.md` | modify | Add Token Metrics table and F-003 DoD item |
| `tasks/features/_templates/validation.md` | modify | Add Token Metrics table and F-003 DoD item in repo starter template |
| `tasks/bugfixes/_templates/validation.md` | modify | Add Token Metrics table and F-003 DoD item in repo starter template |
| `tasks/hotfixes/_templates/validation.md` | modify | Add Token Metrics table and F-003 DoD item in repo starter template |
| `tasks/spikes/_templates/findings.md` | modify | Add Token Metrics table and F-003 DoD item in repo starter template |
| `internal/cli/cli.go` | modify | Add sdd record-tokens top-level command dispatch and help |
| `internal/cli/cli_test.go` | modify | Add CLI test for top-level sdd record-tokens dispatch |
| `internal/cli/task_state.go` | modify | Expose runTaskStateRecordTokens for top-level shorthand |
| `internal/cli/task_state_test.go` | modify | CLI tests for top-level sdd record-tokens |
| `internal/taskpr/taskpr.go` | modify | Read token telemetry and embed ## Token Telemetry section in PR markdown and JSON |
| `internal/taskpr/taskpr_test.go` | modify | Test PR bundle with token telemetry |
| `internal/check/review.go` | modify | Skip YAML frontmatter when checking review.md status |
| `internal/check/review_test.go` | create | Test review check with YAML frontmatter |
| `AGENTS.md` | modify | Update contract parity and templates=2.2.0 |
| `GEMINI.md` | modify | Update contract parity and templates=2.2.0 |
| `CLAUDE.md` | modify | Update contract parity and templates=2.2.0 |
| `.sdd/manifest.json` | modify | Update version to 2.2.0 |
| `README.md` | modify | Document sdd record-tokens shorthand |
| `ROADMAP.md` | modify | Update T-055 status and progress |
| `SKILLS.md` | modify | Register Human Feedback Rule F-003 |
| `TECH_STACK.md` | modify | Record D-047 as Approved and update verified facts |
| `tasks/features/T-053-discovery-ensemble/validation.md` | modify | Retrospective token metrics table and sign-off for T-053 |
| `tasks/features/T-055-default-token-telemetry/requirements.md` | modify | Task requirements document |
| `tasks/features/T-055-default-token-telemetry/plan.md` | modify | Task plan document |
| `tasks/features/T-055-default-token-telemetry/validation.md` | modify | Task validation document |
| `tasks/features/T-055-default-token-telemetry/review.md` | modify | Task review document |

## 6. Living Documents to Update

- `ROADMAP.md`: mark T-055 `Implementing`, then `Awaiting review`, update Milestone 12 progress.
- `SKILLS.md`: register Human Feedback Rule F-003 (token telemetry tracking and reporting).
- `TECH_STACK.md`: record D-047 as Approved, update verified facts log.
- `README.md`: document `sdd record-tokens <T-xxx>` top-level shorthand.
- `AGENTS.md`, `GEMINI.md`, `CLAUDE.md`: contract parity update for token telemetry and template version 2.2.0.

## 7. Applicable Human Feedback Rules

- **F-001:** Resolve all review findings (Blockers, Warnings, Minors) across Review Triad before Gate 2 close.
- **F-002:** Zero personal absolute filesystem paths in templates, docs, or tests.
- **F-003:** Record token telemetry via `sdd task state record-tokens` / `sdd record-tokens`, document in `validation.md`, and report at Gate 2.

## 8. Risks & Mitigations

- **Risk:** Existing projects running `sdd upgrade` see conflicts in `validation.md` if customized.
  - **Mitigation:** `sdd upgrade` compares manifest hashes. Untouched templates are updated cleanly; customized templates receive side-by-side proposals (`.sdd-new`) without overwriting human work.
- **Risk:** Agent operating contracts become desynchronized.
  - **Mitigation:** `sdd check --strict` validates byte-level contract parity across `AGENTS.md`, `GEMINI.md`, and `CLAUDE.md`.

### Rollback
- Discard task worktree via `sdd task clean T-055` if worktree was provisioned.
- In case of pre-merge failure or human rejection, discard task branch; `main` branch remains untouched and pristine.
- Revert uncommitted changes via `git checkout -- .` on the isolated task branch.
