---
status: draft
---
# Requirements — T-055 Default task token telemetry and template integration

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

## 1. Goal

Integrate task token telemetry seamlessly into SDD by default for all future tasks and projects:
1. Provide standard `## Token Metrics` sections in all task starter templates (`features`, `bugfixes`, `hotfixes`, `spikes`).
2. Enforce token telemetry recording and Gate 2 reporting in the agent operating contract (`contract.md.tmpl`, `AGENTS.md`, `GEMINI.md`, `CLAUDE.md`) in compliance with Human Feedback Rule `F-003`.
3. Provide a turnkey top-level shorthand command `sdd record-tokens <T-xxx>` delegating to `sdd task state record-tokens`.
4. Surface task token telemetry automatically in `sdd task pr <T-xxx>` Pull Request markdown bodies and JSON bundles.
5. Fix frontmatter handling in `internal/check/review.go` so frontmatter in `review.md` does not trip false-positive status warnings.
6. Bump `templates.Version` from `2.1.0` to `2.2.0` and verify clean upgrades via `sdd upgrade`.

## 2. Scope & Non-Goals

### In Scope
- Task templates across all 4 typologies (`features`, `bugfixes`, `hotfixes`, `spikes`) in embedded `internal/templates/data/tasks/` and repository `tasks/*/_templates/`.
- Agent operating contracts (`internal/templates/data/base/contract.md.tmpl`, `AGENTS.md`, `GEMINI.md`, `CLAUDE.md`).
- Top-level CLI command `sdd record-tokens <T-xxx>` with flag, stdin, and json support.
- Pull request bundle enrichment in `internal/taskpr/taskpr.go`.
- Robust frontmatter skipping in `internal/check/review.go`.
- `templates.Version` bump to `2.2.0` with full regression and upgrade test coverage.

### Non-Goals
- Background network sniffing or proxying external LLM API traffic (SDD remains zero-network, local, and deterministic per Principle 2 and D-007).
- Modifying token telemetry persistence format in `.sdd/state/<T-xxx>/state.json` (format established in T-049 remains invariant).

## 3. Requirements

| ID | Statement | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL include a standard `## Token Metrics` table and a Definition of Done checklist item in `validation.md` / `findings.md` starter templates for all 4 typologies (`features`, `bugfixes`, `hotfixes`, `spikes`). | MUST | inspection |
| R-002 | The system SHALL update the agent contract template (`contract.md.tmpl`) and repository agent contracts (`AGENTS.md`, `GEMINI.md`, `CLAUDE.md`) in Phase D, Gate 2, Section 7 (Communication format), and Section 8 (Definition of Done) to mandate recording tokens via `sdd task state record-tokens` / `sdd record-tokens` and presenting metrics at Gate 2 per Rule F-003. | MUST | test |
| R-003 | The system SHALL provide a top-level command `sdd record-tokens <T-xxx> [--input <N>] [--cache <N>] [--output <N>] [--stdin] [--json]` as an ergonomic alias for `sdd task state record-tokens`. | MUST | test |
| R-004 | The system SHALL update `sdd task pr <T-xxx>` to read recorded token telemetry from `.sdd/state/<T-xxx>/state.json` and embed a formatted `## Token Telemetry` section in the generated PR body and structured JSON bundle (omitting the section when no tokens are recorded). | MUST | test |
| R-005 | The system SHALL enhance `checkReviewDocument` in `internal/check/review.go` to ignore YAML frontmatter (`--- ... ---`) when scanning for the markdown review status header. | MUST | test |
| R-006 | The system SHALL bump `templates.Version` to `2.2.0` in `internal/templates/templates.go`, update contract block markers, and verify that `sdd upgrade` and fresh kit generation pass strict check. | MUST | test |
| R-007 | The system SHALL maintain 100% standard library purity (`D-007`) and determinism (`D-003`). | MUST | test |

## 4. Acceptance Criteria

- **AC-1 (Template Defaults):** Running `sdd task new --type <feature|bugfix|hotfix|spike> T-003 sample` creates task documents where `validation.md` (or `findings.md`) contains `## 8. Token Metrics` (or equivalent) and Definition of Done references Rule F-003.
- **AC-2 (Contract Parity & Strict Check):** `AGENTS.md`, `GEMINI.md`, and `CLAUDE.md` maintain 100% identical rules, have `templates=2.2.0`, and pass `sdd check --strict`.
- **AC-3 (record-tokens Shorthand):** Running `./bin/sdd record-tokens T-055 --input 1000 --cache 5000 --output 200` outputs formatted token telemetry and persists into `.sdd/state/T-055/state.json`. Running with `--json` outputs JSON matching `task state record-tokens`.
- **AC-4 (PR Token Telemetry):** Running `sdd task pr T-055` includes `## Token Telemetry` in the PR markdown output when tokens exist for the task. When no tokens are recorded in `.sdd/state/<T-xxx>/state.json`, `PRBundle.Tokens` is `nil` and the `## Token Telemetry` section is omitted cleanly without error.
- **AC-5 (Review Frontmatter Immunity):** `sdd check` parses `review.md` files with YAML frontmatter without misidentifying `status: draft` as the markdown review status.
- **AC-6 (Upgrade Compatibility):** `internal/upgrade` tests succeed verifying upgrade of contracts and manifests to `2.2.0`.

## 5. Open Questions
(none)
