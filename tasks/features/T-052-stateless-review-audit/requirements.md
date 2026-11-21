---
status: draft
---
# Requirements — T-052 Stateless review context isolation and outside validation audit runner

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-21)
Roadmap ref: `ROADMAP.md` T-052

## 1. Goal
Implement stateless review context isolation by default across all four SDD reviewer roles (Plan Reviewer, Spec Auditor, AppSec Specialist, Pattern Auditor) and an outside verification audit runner (`sdd task audit`), ensuring fresh tabula-rasa context without conversational bias and mechanical validation execution.

## 2. Context and references
- Mission / principles touched: Fail safe, Deterministic, Living documents.
- Decisions this relies on: `D-004` (Toolchain & check), `D-007` (Go stdlib only), `D-026` (Code review standard), `D-029` (Two-tier coordinator hierarchy), `D-031` (Review Triad), `D-044` (Stateless reviewer context isolation & mechanical audit runner).
- Active feedback rules: `F-001` (Resolve all review findings), `F-002` (Zero personal local paths).
- Depends on tasks: `T-048` (Multi-agent coordinator contract), `T-050` (Path hygiene validator).

## 3. Scope
**In scope**
- Implement `internal/audit` package supporting reviewer bundle staging, tabula-rasa prompt generation, and outside validation auditing.
- Make all four SDD reviewers stateless and isolated by default without requiring specialized opt-in flags:
  - `sdd prompt plan-review <TASK_ID>`: stages frozen specification and plan-reviewer rubric in `.sdd/review/<TASK_ID>/plan/`, generating a tabula-rasa review prompt.
  - `sdd prompt review <TASK_ID> [--lens <lens>]`: stages frozen specification, git diff patch (`diff.patch`), and lens rubric in `.sdd/review/<TASK_ID>/code/`, generating a tabula-rasa review prompt.
  - Prompts instruct reviewers to evaluate strictly the frozen artifacts with zero historical conversational baggage or anchoring bias.
  - Ensure `.sdd/review/` is ignored by `.gitignore` and `.ragignore`.
- `sdd task audit <TASK_ID>` (and alias `sdd review audit <TASK_ID>`):
  - Parse `validation.md` from the task folder to extract verification commands from backtick code spans.
  - Support compound command execution (`cmd1 && cmd2`) by splitting into discrete execution steps.
  - Whitelist execution of safe verification commands (`go test`, `make`, `sdd check`, `./bin/sdd`, `bin/sdd`, `sdd status`, `git grep`, etc.).
  - Mechanically execute verification commands outside agent narrative, recording true exit codes, execution durations, and output streams.
  - Validate determinism by executing determinism checks and confirming identical outcomes.
  - Support machine-readable `--json` output for CI and Coordinator orchestration.
- Unit and integration tests in `internal/audit/audit_test.go`, `internal/cli/audit_test.go`, and `internal/cli/prompt_test.go`.

**Out of scope**
- LLM API calling or automated subagent spawning (the host / coordinator spawns agents; SDD generates the isolated bundles and runs mechanical verification).
- Arbitrary destructive command execution (auditor enforces strict command safety whitelisting).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL stage an immutable, isolated review bundle in `.sdd/review/<TASK_ID>/plan/` containing `spec.md`, `rubric.md`, and `manifest.json` by default when `sdd prompt plan-review` runs. | MUST | test |
| R-002 | The system SHALL stage an immutable, isolated review bundle in `.sdd/review/<TASK_ID>/code/` containing `spec.md` (frozen requirements.md and plan.md), `diff.patch`, `rubric.md`, and `manifest.json` by default when `sdd prompt review` runs. | MUST | test |
| R-003 | The system SHALL format tabula-rasa review prompts by default with zero conversational history for all four reviewer roles (Plan Reviewer, Spec Auditor, AppSec Specialist, Pattern Auditor). | MUST | test |
| R-004 | The system SHALL output structured JSON bundle metadata when `--json` is supplied to `sdd prompt plan-review` or `sdd prompt review`. | MUST | test |
| R-005 | The system SHALL parse verification commands enclosed in backtick code spans from Section 1 and Section 2 of a task's `validation.md` document in `sdd task audit <TASK_ID>`, supporting compound `&&` command chains. | MUST | test |
| R-006 | The system SHALL enforce strict command safety in `sdd task audit`, allowing only whitelisted build, test, lint, and verification commands (`go test`, `make`, `sdd check`, `./bin/sdd`, `bin/sdd`, `sdd status`, `git grep`, `git diff`, `git status`) and rejecting unapproved commands or shell redirects. | MUST | test |
| R-007 | The system SHALL mechanically execute extracted verification commands, measuring exit codes, execution duration, and capturing output streams. | MUST | test |
| R-008 | The system SHALL support `sdd review audit` as a backwards-compatible alias for `sdd task audit`. | MUST | test |
| R-009 | The system SHALL support `--json` output in `sdd task audit` for integration with Task Coordinator and CI pipelines. | MUST | test |
| R-010 | The system SHALL pass full test suite, linting, race detection, and `sdd check --strict` with zero warnings. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Bundle staging is idempotent and byte-deterministic for identical repository states.
- **Security & Confinement:** Execution is restricted to whitelisted safe commands; no arbitrary shell injection or string-concatenated shell invocation.
- **Standard Library Only:** Standard Go library only (`os`, `os/exec`, `path/filepath`, `strings`, `regexp`, `encoding/json`, `time`) per Decision `D-007`.

## 6. Constraints and assumptions
- Task must exist in one of the 4 typology folders (`tasks/features/`, `tasks/bugfixes/`, `tasks/hotfixes/`, `tasks/spikes/`).
- Staged `.sdd/review/` bundles are transient task infrastructure and ignored by `.gitignore` and `.ragignore`.

## 7. Open questions
(None — settled by Decision D-044 proposal)

## 8. Acceptance criteria
- **AC-1:** Given task `T-050`, when `sdd prompt plan-review T-050` runs, then `.sdd/review/T-050/plan/spec.md`, `rubric.md`, and `manifest.json` are staged by default with a tabula-rasa prompt.
- **AC-2:** Given task `T-050`, when `sdd prompt review T-050 --lens spec` runs, then `.sdd/review/T-050/code/spec.md`, `diff.patch`, `rubric.md`, and `manifest.json` are staged by default with a tabula-rasa prompt.
- **AC-3:** Given task `T-050` with completed `validation.md`, when `sdd task audit T-050` runs, then all verification commands are executed mechanically, returning overall `PASS` with 0 failures.
- **AC-4:** Given an unapproved shell command (e.g. `rm -rf /` or `curl ...`), when `sdd task audit` parses `validation.md`, then it refuses execution and reports a safety rejection.
