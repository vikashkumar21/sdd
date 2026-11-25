---
status: active
---
# Requirements — T-061 Lens-scoped review bundles and diff filtering

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved 2026-11-25
Roadmap ref: `ROADMAP.md` T-061

## 1. Goal
Produce tailored, lens-scoped diff and specification slices for each reviewer lens (`spec`, `security`, `pattern`) in isolated review directories, reducing redundant context tokens across the Review Triad by 30–70% while eliminating directory collisions during parallel reviewer execution.

## 2. Context and references
- Mission / principles touched: `MISSION.md` §5 Principles (Deterministic, Fail safe, Small and shippable); `SKILLS.md` §10 (Code Review and Verification Standard), §16 (Ephemeral Tabula-Rasa Reviewer Lifecycle), §18 (Tier-Calibrated Subagent Dispatch).
- Decisions this relies on: `D-007` (Go stdlib only), `D-026` (Review rubric & review.md), `D-031` (3-Subagent Review Triad), `D-044` (Stateless Review Isolation), `D-049` (Cognitive Bias Immunity & Heterogeneous Model Routing), `D-050` (Token Consumption Mitigation & Provider Caching).
- Depends on tasks: `T-060` (Subagent model routing & tier enforcement; Done).

## 3. Scope
**In scope**
- Directory isolation per reviewer lens:
  - `.sdd/review/<TASK_ID>/spec/` for `--lens spec`
  - `.sdd/review/<TASK_ID>/security/` for `--lens security`
  - `.sdd/review/<TASK_ID>/pattern/` for `--lens pattern`
  - `.sdd/review/<TASK_ID>/code/` for `--lens all` (preserving backward compatibility)
- Deterministic diff filtering engine (`FilterGitDiffByLens`):
  - `security`: focuses on executable application code, scripts, configurations, workflows, and dependency manifests; filters out non-executable task markdown documents and living doc changelogs.
  - `pattern`: focuses on code, tests, architecture decisions, coding standards, and active feedback rules; filters out task validation evidence and roadmap changelogs.
  - `spec`: focuses on requirements traceability, plan boundary enforcement, and positive/negative test coverage; retains full file modification list for boundary verification.
  - `all`: retains complete, unfiltered diff (fallback).
- Lens-scoped remediation finding filtering (`FilterFindingsForLens` and `FormatFindingsToVerifyForLens`) for Loop > 1, ensuring reviewers only receive defects applicable to their domain.
- Bundle manifest telemetry in `manifest.json`:
  - `raw_diff_bytes`, `filtered_diff_bytes`, `diff_savings_percent`, `raw_files_count`, `filtered_files_count`.
- Turnkey CLI integration in `sdd prompt review` and `ReviewPromptContext` JSON export.
- Synchronization of `skills/code-review/SKILL.md`, `skills/task-coordinator/SKILL.md`, and embedded templates.
- Bump `templates.Version` from `2.8.0` to `2.9.0` per Hard Rule 16.

**Out of scope**
- Prompt cache prefix alignment and dynamic suffix restructuring (scheduled for `T-062`).
- Living document micro-index and progressive disclosure (scheduled for `T-063`).
- Native transcript ingestion in `sdd record-tokens` (scheduled for `T-064`).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL stage review bundles in isolated directories per lens: `.sdd/review/<TASK_ID>/spec/` for `spec`, `.sdd/review/<TASK_ID>/security/` for `security`, `.sdd/review/<TASK_ID>/pattern/` for `pattern`, and `.sdd/review/<TASK_ID>/code/` for `all`, preventing filesystem race conditions during parallel Review Triad execution. | MUST | test |
| R-002 | The system SHALL implement a deterministic diff filtering engine (`FilterGitDiffByLens`) that tailors `diff.patch` contents to the specific audit focus of each lens (`spec`, `security`, `pattern`, `all`), eliminating out-of-scope diff hunks while preserving header integrity and fail-safe notices when all changes are out-of-scope. | MUST | test |
| R-003 | The system SHALL filter remediation findings in `findings_to_verify.md` by lens domain (`FilterFindingsForLens`) during multi-loop reviews (Loop > 1), presenting only findings pertinent to the auditor's specific lens or general unassigned findings. | MUST | test |
| R-004 | The system SHALL record diff filtering telemetry in `manifest.json` including `raw_diff_bytes`, `filtered_diff_bytes`, `diff_savings_percent`, `raw_files_count`, and `filtered_files_count`. | MUST | test |
| R-005 | The system SHALL update `sdd prompt review` to output prompts and JSON context (`ReviewPromptContext`) pointing to the lens-scoped bundle directory, bundle files, and filtering telemetry. | MUST | test / demo |
| R-006 | The system SHALL update `skills/code-review/SKILL.md`, `skills/task-coordinator/SKILL.md`, and embedded templates to document lens-scoped review bundles, and bump `templates.Version` to `2.9.0` with verified kit migration. | MUST | check / test |
| R-007 | The system SHALL maintain 100% Go standard library purity (`D-007`), execute race-clean (`-race`), and pass determinism verification (`-count=2`). | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Given identical git repository state and task files, `StageReviewBundleWithLoop` produces byte-identical review bundles and manifest metrics.
- **Fail-Safe Behavior:** If diff filtering encounters malformed diff hunks or unrecognized lens values, it safely falls back to the full unfiltered diff without panicking.
- **Performance Budget:** Diff parsing and filtering operates strictly in-memory using buffered scanners, adding less than 20ms of overhead on 10,000-line diffs.
- **Security & Confinement:** All bundle file writes remain confined within `os.Root` under `.sdd/review/<TASK_ID>/<lens>/`. Untracked file traversal checks and size limits are strictly preserved.

## 6. Constraints and assumptions
- Standard library purity: no third-party diff parser or AST libraries (`D-007`).
- Retain `.sdd/review/<TASK_ID>/code/` as the bundle location when `--lens all` is specified to ensure backwards compatibility with external scripts and tooling.
- Untracked files and git diff generation remain compatible with POSIX and Windows environments.

## 7. Open questions
(None. All design choices settled under D-050.)

## 8. Acceptance criteria
- **AC-1:** Given a repository with both Go code changes and documentation changes, when `sdd prompt review T-061 --lens security` is executed, then `.sdd/review/T-061/security/diff.patch` is created containing Go code changes and omitting non-executable documentation diffs, and `manifest.json` records `diff_savings_percent > 0`.
- **AC-2:** Given parallel execution of `sdd prompt review T-061 --lens spec`, `--lens security`, and `--lens pattern`, each reviewer's bundle is written to its dedicated lens directory (`spec/`, `security/`, `pattern/`) without file clobbering or write conflicts.
- **AC-3:** Given a task on remediation Loop 2 with a `review.md` containing `SPEC-001` (Category: `Spec`), `SEC-001` (Category: `Security`), and `PAT-001` (Category: `Pattern`), when `sdd prompt review T-061 --lens security --loop 2` runs, then `.sdd/review/T-061/security/findings_to_verify.md` contains only `SEC-001` and any uncategorized general findings.
- **AC-4:** `make verify` and `./bin/sdd check --strict` report 0 errors and 0 warnings.
