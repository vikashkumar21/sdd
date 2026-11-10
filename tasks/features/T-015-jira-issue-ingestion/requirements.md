# Requirements — T-015 Jira issue ingestion (JSON, Markdown, and text)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-02)
Roadmap ref: `ROADMAP.md` T-015

## 1. Goal
Parse Jira issue specifications from JSON, Markdown, and plain text formats and import their key, summary, description, and acceptance criteria into task templates (`requirements.md`) during `sdd feature new` and stage them as structured context artifacts via `sdd ingest`.

## 2. Context and references
- Mission / principles: Principle 1 (Deterministic), Principle 4 (Small and shippable), Principle 5 (Never overwrite human work).
- Decisions this relies on: D-018 (Approved - Context staging), D-019 (Approved - Jira task integration method Option A).
- Depends on tasks: T-014 (Context staging pipeline).

## 3. Scope
**In scope**
- `internal/jira` parser supporting Jira issue extraction from:
  - Jira REST API / export JSON (including `fields.summary`, `fields.description`, custom fields or arrays for acceptance criteria).
  - Markdown exports (headers, bullet points, `Acceptance Criteria` sections).
  - Structured plain text (Key, Summary, Description, Acceptance Criteria).
- Integration with `sdd feature new`:
  - `--from-jira <file|string>` flag.
  - Automatically seeds `requirements.md` Goal from the Jira summary.
  - Adds Jira issue key and type to `requirements.md` Context and references.
  - Generates initial rows in Functional requirements table from acceptance criteria.
  - Populates Acceptance criteria section.
  - Automatically stages the source Jira file into `.sdd/context/<task>/` as a `jira` format artifact.
- Integration with `sdd ingest`:
  - `--jira` flag to explicitly parse and stage Jira issue content with format `jira`.

**Out of scope**
- Live Jira REST API network requests or OAuth credentials (per D-019 Option A; stdlib offline purity).
- Bidirectional synchronization back to Jira.

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL parse Jira issues from JSON exports, extracting key, summary, description, and acceptance criteria. | MUST | test |
| R-002 | The system SHALL parse Jira issues from Markdown and structured plain text. | MUST | test |
| R-003 | The system SHALL support `--from-jira <file>` in `sdd feature new`, populating `requirements.md` with Goal, Context, Functional requirements, and Acceptance criteria. | MUST | test |
| R-004 | The system SHALL stage the imported Jira issue into `.sdd/context/<task>/` with format `jira` upon `sdd feature new --from-jira`. | MUST | test |
| R-005 | The system SHALL support `--jira` in `sdd ingest` to validate and stage Jira issue content. | MUST | test |
| R-006 | The system SHALL fail safe with an explanatory error when Jira input cannot be parsed or lacks summary information. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Parsing the same Jira artifact SHALL always produce identical `requirements.md` content and staged hashes.
- **Fail-safe behaviour:** Malformed JSON or unknown text SHALL report clear errors without panicking.
- **Security & privacy:** Offline processing only; zero network calls or credential storage (`D-019`).
- **Dependencies:** Standard library only (`D-007`).

## 6. Constraints and assumptions
- Jira issues contain at least an issue key (e.g. `PROJ-123`) or a summary line.
- The project follows the standard `features/_templates/requirements.md` template layout.

## 7. Open questions
(None. Resolved via Decision D-019.)

## 8. Acceptance criteria
- Given a Jira JSON file with summary and 2 acceptance criteria, when `sdd feature new T-002 token-refresh --from-jira issue.json` runs, then `features/T-002-token-refresh/requirements.md` is populated with the summary as Goal and both criteria in the requirements table and acceptance criteria section, and the issue is staged in `.sdd/context/T-002/`.
- Given a Jira Markdown export, when parsed by `jira.Parse`, then key, summary, description, and acceptance criteria are correctly extracted.
- Given an invalid file, when passed to `--from-jira`, then `sdd feature new` exits 1 with a descriptive error.
