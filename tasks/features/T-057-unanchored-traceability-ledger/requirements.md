---
status: active
---
# Requirements — T-057 Unanchored Requirements Traceability Ledger and omission-first review protocol

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Active
Roadmap ref: `ROADMAP.md` T-057
Decision ref: `TECH_STACK.md` D-049 (Approved 2026-11-24)

## 1. Goal

Eliminate the LLM "Diff Framing" anchoring bias and omission blindness in the Review Triad by establishing the **Unanchored Requirements Traceability Ledger** protocol, auto-generating unanchored ledger bundles in `internal/audit`, requiring mandatory negative-path test evidence, embedding the ledger in task review templates (templates version 2.5.0), and enforcing mechanical ledger completeness in `sdd check`.

## 2. Context and references

- Mission / principles: Principle 1 (Deterministic by construction), Principle 2 (Fail safe), Principle 3 (Human in the loop for the irreversible).
- Decisions: D-003 / D-007 (Standard library only), D-026 / D-032 / D-033 (Code review standard, Rule F-001, Review Triad), D-044 (Stateless review context isolation), D-048 (Forensic brownfield coverage ledger), D-049 (Cognitive bias immunity & unanchored auditing).
- Active Feedback Rules: F-001 (Resolve all findings before Gate 2), F-002 (Zero personal absolute paths), F-003 (Token telemetry).

## 3. Scope

### In scope
1. **Canonical Requirements Traceability Ledger Schema (`R-001`):**
   - Standardize `## Requirements Traceability Ledger` in `review.md`:
     `| Req ID | Requirement Statement | Code Symbol & File | Positive Test Evidence | Negative / Boundary Test Evidence | Status |`
   - Every requirement `R-xxx` from `requirements.md` must be itemized.
   - Status must be `Verified` with concrete evidence or `Omission` / `Missing` (automatic 🔴 Blocker).
2. **Automated Ledger Bundle Generation (`R-002`):**
   - In `internal/audit/bundle.go`, implement deterministic `ExtractRequirements(content string) []RequirementItem`.
   - In `StageReviewBundle`, generate `.sdd/review/<TASK_ID>/code/ledger.md` pre-populated with all `R-xxx` IDs and requirement statements, with unanchored empty evidence columns for the reviewer to fill in *before* opening `diff.patch`.
3. **Omission-First Review Protocol (`R-003`):**
   - Update `skills/code-review/SKILL.md` and `internal/templates/data/skills/spec-auditor/SKILL.md` (and role standards):
     - **Step 1: Unanchored Traceability Scan:** The reviewer extracts/fills the ledger directly against the candidate code *before* reading `diff.patch`. Any unmapped requirement or missing test is an immediate 🔴 Blocker.
     - **Step 2: Incremental Diff Quality Audit:** Review `diff.patch` for security, pattern, and style only after all `R-xxx` requirements are accounted for.
4. **Mandatory Negative-Path Test Evidence (`R-004`):**
   - For all branching logic, access gates, input validation, and error states, the ledger must cite an automated test proving that the failure/negative path executes and aborts cleanly.
5. **Mechanical Validation in `sdd check` (`R-005`):**
   - Implement `traceability-ledger` rule in `internal/check/review.go`:
     - Flags error if `## Requirements Traceability Ledger` is missing in completed `review.md`.
     - Flags error if any `R-xxx` from `requirements.md` is omitted from the ledger.
     - Flags error if any cell is empty, placeholder (`<...>`, `TODO`, `n/a`), or marked `Omission`.
6. **Task Review Templates & Version Bump (`R-006`):**
   - Update review starter templates across applicable typologies (`features` and `bugfixes`) in `internal/templates/data/tasks/features/review.md`, `internal/templates/data/tasks/bugfixes/review.md`, `tasks/features/_templates/review.md`, and `tasks/bugfixes/_templates/review.md` to embed `## Requirements Traceability Ledger`.
   - Bump `templates.Version` to `2.5.0` and update contract version markers (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`) per Hard Rule 16.
7. **Standard Library Purity (`R-007`):**
   - 100% Go standard library implementation (`D-007`); zero external dependencies.

### Out of scope
- Dynamic model routing and provider dispatch (addressed in T-059).
- Ephemeral subagent runner lifecycle orchestration (addressed in T-058).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL define and standardize the `## Requirements Traceability Ledger` table schema in `review.md` requiring every `R-xxx` to have code symbol, positive test, and negative test evidence. | MUST | test / inspection |
| R-002 | The system SHALL implement `ExtractRequirements` and stage an unanchored `ledger.md` in `.sdd/review/<TASK_ID>/code/` pre-seeded with all task requirement statements and unanchored evidence columns. | MUST | test |
| R-003 | The system SHALL document and enforce the Omission-First Review Protocol in `skills/code-review/SKILL.md` and `internal/templates/data/skills/spec-auditor/SKILL.md` role standards, mandating ledger population prior to reading `diff.patch`. | MUST | inspection / check |
| R-004 | The system SHALL enforce mandatory negative-path test evidence in the ledger for branching, validation, and gating requirements. | MUST | test |
| R-005 | The system SHALL implement the mechanical `traceability-ledger` rule in `sdd check`, reporting errors for missing ledgers, omitted `R-xxx` entries, or placeholder/unverified status. | MUST | test |
| R-006 | The system SHALL update task review starter templates across applicable typologies (`features` and `bugfixes`), bump `templates.Version` to `2.5.0`, and update contract markers per Hard Rule 16. | MUST | test |
| R-007 | The system SHALL implement all ledger extraction and verification logic using only the Go standard library with zero third-party dependencies (`D-007`). | MUST | test |

## 5. Non-functional requirements

- **Determinism:** `ExtractRequirements` produces deterministic, sorted output matching requirement declaration order.
- **Fail-safe:** If `requirements.md` is malformed or unreadable, `sdd check` fails restrictive with an explicit syntax error.
- **Performance budget:** Traceability ledger validation adds < 20ms to `sdd check`.

## 6. Constraints and assumptions

- Task requirements follow standard SDD `R-xxx` table convention in `requirements.md`.
- Historical completed tasks prior to `v1.9.0` retain their existing review format; `traceability-ledger` rule applies to active/new tasks.

## 7. Open questions

(none — Decision D-049 Approved on 2026-11-24)

## 8. Acceptance criteria

- **AC-1 (Ledger Staging):** Running `sdd prompt review T-057` stages `.sdd/review/T-057/code/ledger.md` containing all `R-xxx` requirement statements ready for unanchored review.
- **AC-2 (Omission Detection):** An approved `review.md` missing any `R-xxx` declared in `requirements.md` fails `sdd check --strict` under rule `traceability-ledger`.
- **AC-3 (Evidence Completeness):** An approved `review.md` containing placeholder tokens (`<...>`, `TODO`) in code or test columns fails `sdd check --strict`.
- **AC-4 (Template Version Bump):** `templates.Version` is bumped to `2.5.0`, contract headers match `templates=2.5.0`, and regenerating a test kit in a temporary directory passes `sdd check --strict`.
- **AC-5 (Standard Library Purity):** Zero external dependencies added to `go.mod`.

