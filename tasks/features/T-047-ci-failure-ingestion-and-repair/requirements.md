---
status: active
task_id: T-047
last_reviewed: 2026-11-19
---
# Requirements — T-047 Remote CI failure ingestion and diagnostic auto-repair loop

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-19)
Roadmap ref: `ROADMAP.md` T-047

## 1. Goal
Provide remote CI log ingestion via `sdd ingest --ci-log` with robust ANSI stripping, multi-vector credential and secret sanitization, and automated diagnostic extraction, paired with bugfix task reproduction seeding (`sdd bugfix new --from-ci` and `--seed-bugfix`) and turnkey auto-repair prompt generation (`sdd prompt repair <T-xxx>`) to close the feedback loop between remote CI failures and deterministic local repairs.

## 2. Context and references
- Mission / principles touched: Principles 1 (Deterministic), 2 (Fail safe), 3 (Human-in-the-loop for the irreversible), 4 (Small and shippable), 5 (Never overwrite human work).
- Decisions this relies on: `D-007` (Standard library only), `D-018` (Context ingestion architecture), `D-027` (Typology lifecycle & bugfix `reproduction.md`), `D-035` (Task state persistence), `D-036` (Role skill taxonomy), `D-040` (Universal CI workflows, automated PR packaging, and CI log ingestion).
- Depends on tasks: `T-046` (Universal CI workflows and automated PR packaging).

## 3. Scope
**In scope**
- **CI Log Ingestion Engine (`internal/cilog`):**
  - ANSI Escape Sequence Stripping:
    - Strips all ANSI color codes, cursor controls, and escape sequences (`\x1b\[[0-9;]*[a-zA-Z]`, `\x1b\(B`, etc.).
  - Multi-Vector Secret and Credential Sanitization:
    - Authorization headers and Bearer tokens (`Bearer [A-Za-z0-9_\-\.]+` -> `Bearer [REDACTED]`).
    - AWS credentials (`AKIA[0-9A-Z]{16}` -> `[AWS_KEY_REDACTED]`, secret access keys).
    - GitHub tokens (`ghp_[A-Za-z0-9]{36}`, `github_pat_[A-Za-z0-9_]{82}` -> `ghp_[REDACTED]`).
    - Common environment variable and key-value secret patterns (`password=...`, `token: ...`, `api_key=...`, `private_key=...`).
    - Preserves CI-native masking (`***`).
  - Failure Diagnostic Analysis & Extraction:
    - Infers CI platform from runner markers (GitHub Actions, GitLab CI, Bitbucket, Azure DevOps, Cloud Build, CodeBuild, Generic).
    - Extracts failed step/command (e.g. `go test -v ./...`, `make verify`, `npm test`, `mvn test`, `sdd check --strict`).
    - Extracts primary error headline (e.g. failed test name, panic message, lint failure, exit status).
    - Extracts list of failing tests or compilation errors.
    - Captures a bounded failure snippet / stack trace window around the failure point.
- **CLI Integration (`sdd ingest --ci-log`):**
  - Flags:
    - `--ci-log`: activates CI log ingestion pipeline.
    - `--file <path>`: file path to raw CI log (repeatable).
    - `--name <label>`: custom context entry name.
    - `--task <T-xxx>`: associate context entry with existing task.
    - `--seed-bugfix`: automatically seed a bugfix task folder from the extracted failure diagnostics (requires `--task <T-xxx>` existing in `ROADMAP.md`; slug defaults to `--name` or kebab-cased failing test name).
    - Supports reading directly from piped standard input (e.g. `gh run view --log-failed | sdd ingest --ci-log`).
  - Stages cleaned log into `.sdd/context/` with format `ci-log`.
- **Bugfix Reproduction Seeding (`sdd bugfix new --from-ci` & `sdd task new --type bugfix --from-ci`):**
  - Injects parsed diagnostics directly into `tasks/bugfixes/T-xxx-<slug>/reproduction.md`:
    - Section 1: Problem Summary (Observed behavior: failure headline & command).
    - Section 2: Environment & Preconditions (Runner OS & runtime versions from log).
    - Section 3: Reproduction Steps (Extracted execution command).
    - Section 4: Minimal Failing Test Case (Failed test name & sanitized failure output).
- **Turnkey Auto-Repair Agent Prompt (`sdd prompt repair <T-xxx>`):**
  - Emits focused auto-repair prompt for the `implementer` or `architect` role:
    - Mandates Step 0: author minimal failing test case reproducing defect.
    - Presents sanitized CI trace, failed assertions, and reproduction command.
    - Outlines minimal repair boundary and verification steps.
    - Supports `--max-loops <N>` (configurable loop ceiling, default: `3`, allowed range: `1` to `5`).
    - Integrates with task state tracking (`.sdd/state/<T-xxx>/`), displaying the current loop attempt (e.g. `Attempt: 1 of 3`).
    - Enforces circuit breaker halt: when the attempt counter reaches `--max-loops`, the prompt instructs the agent to halt immediately, mark the task `Blocked`, and escalate to the human with diagnostic findings.
    - Supports `--json` for programmatic Agent SDK consumption.

**Out of scope**
- Direct network API integration to download remote CI logs (logs are ingested via CLI file or stdin from tools like `gh run view`).
- Autonomous code mutation without Gate 1 approval.
- Modifying remote CI service configurations.

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL strip all ANSI escape sequences, color codes, and terminal formatting characters from ingested CI logs. | MUST | test / inspection |
| R-002 | The system SHALL sanitize all detected secrets, AWS access keys, GitHub tokens, Bearer tokens, passwords, and private keys, replacing them with `[REDACTED]` markers. | MUST | test |
| R-003 | The system SHALL extract failure diagnostics from sanitized CI logs, identifying the failed step, primary error headline, failed test list, and failure snippet across Go, Node, Python, Java, and SDD checks. | MUST | test |
| R-004 | The system SHALL support `sdd ingest --ci-log` via `--file`, `--text`, or piped stdin, staging the sanitized log in `.sdd/context/` with format `ci-log` and updated `manifest.json`. | MUST | test |
| R-005 | The system SHALL support seeding a bugfix task via `sdd ingest --ci-log --seed-bugfix`, `sdd bugfix new <T-xxx> <slug> --from-ci <file|staged-id>`, or `sdd task new --type bugfix <T-xxx> <slug> --from-ci <file|staged-id>`, populating `reproduction.md` with problem summary, reproduction command, and sanitized failure trace. | MUST | test |
| R-006 | The system SHALL provide `sdd prompt repair <T-xxx> [--json] [--max-loops <N>]` to generate a turnkey auto-repair prompt for coding agents with Step 0 reproduction mandates, failure diagnostics, configurable loop limit (default: 3, range: 1–5), task state loop tracking, and circuit breaker halt escalation. | MUST | test / inspection |
| R-007 | The system SHALL reject oversized CI logs exceeding `context.MaxInputBytes` (10MB) or malformed inputs with descriptive, fail-safe error messages. | MUST | test |
| R-008 | The system SHALL document `sdd ingest --ci-log`, `sdd bugfix new --from-ci`, and `sdd prompt repair` in CLI help text and `README.md`. | MUST | inspection |

## 5. Non-functional requirements
- **Determinism:** Ingesting the same CI log produces byte-identical sanitized output, diagnostics, and seeded `reproduction.md`.
- **Fail-Safe Security:** Any unrecognized token pattern resembling a credential or sensitive variable is redacted; ambiguous failure formats fallback to safe raw snippet extraction.
- **Bounded Memory:** Scanner and sanitization buffer sizes are bounded; maximum input size strictly enforced (`MaxInputBytes = 10MB`).
- **Standard Library Purity (`D-007`):** Pure Go standard library; zero third-party dependencies.

## 6. Constraints and assumptions
- CI logs can come from diverse CI systems (GitHub Actions, GitLab, Bitbucket, Azure, Cloud Build, CodeBuild, or local runner output).
- CI logs may contain multi-megabyte outputs with mixed ANSI colors, timestamps, and interleaved process streams.
- The agent does not directly call remote cloud APIs; the human or caller pipes log output or specifies a log file.

## 7. Open questions
(None — all architectural choices resolved in Decision `D-040`).

## 8. Acceptance criteria
- **AC-1 (ANSI & Secret Sanitization):** Given a CI log containing colored test output, ANSI cursor codes, an AWS access key `AKIAIOSFODNN7EXAMPLE`, a GitHub token `ghp_123456789012345678901234567890123456`, and a Bearer token, when processed by `cilog.Parse()`, then all ANSI sequences are removed and all credentials are replaced with `[REDACTED]` or `[AWS_KEY_REDACTED]`.
- **AC-2 (Diagnostic Extraction):** Given a failing Go test log (`--- FAIL: TestFoo`), when parsed, then `FailedStep` contains the test command, `PrimaryError` identifies `TestFoo`, and `FailureSnippet` contains the failing assertion.
- **AC-3 (CLI Ingestion):** Given a CI failure log, when running `sdd ingest --ci-log --file test.log --name ci-run-1`, then `.sdd/context/` contains the sanitized log and `manifest.json` records `format: ci-log`.
- **AC-4 (Bugfix Seeding):** Given a staged CI log, when running `sdd bugfix new T-xxx fix-test --from-ci test.log`, then `tasks/bugfixes/T-xxx-fix-test/reproduction.md` is initialized with Problem Summary, Reproduction Steps, and the Minimal Failing Test Case section filled from the log.
- **AC-5 (Repair Prompt):** Given a bugfix task with seeded CI diagnostics, when running `sdd prompt repair T-xxx`, then a structured handover prompt is emitted detailing the failure and mandating Step 0 minimal failing test verification.
