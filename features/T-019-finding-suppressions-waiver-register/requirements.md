# Requirements — T-019 Finding suppressions and baseline waiver register

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-019

## 1. Goal
Provide a finding suppression and baseline waiver register (`.sdd/suppressions.json`) that allows human waivers for specific check findings to pass as informational notes in `sdd check` without failing CI or strict mode, while surfacing them with `[SUPPRESSED]` context in `sdd check --fix` so they can remain waived or be repaired.

## 2. Context and references
- Mission / principles touched: Principles 1 (Deterministic), 2 (Fail safe), 3 (Human in the loop for the irreversible), Scope §4 (`sdd check`)
- Decisions this relies on: `D-003` (Go), `D-004` (Toolchain), `D-007` (Standard library only), `D-010` (Managed-block merge)
- Depends on tasks: `T-002` (Done), `T-008` (Done), `T-009` (Done), `T-012` (Done)

## 3. Scope
**In scope**
- Schema and file storage for finding waivers in `.sdd/suppressions.json`.
- Downgrading matching findings in `sdd check` to `INFO` with waiver reason so they do not fail standard runs or `--strict` mode.
- Surfacing suppressed findings in `sdd check --fix` diff previews and prompts with a clear `[SUPPRESSED]` badge.
- Interactive `[s]uppress` option in `sdd check --fix` to record new waivers directly from the repair prompt.
- Automatically clearing waivers when a suppressed item is approved and repaired in `sdd check --fix`.
- Inclusion of `suppressed: true` and reason in `sdd check --json` output.
- Confinement of all read/write operations to the project root via `os.Root` and atomic writes.

**Out of scope**
- Auto-suppressing findings without human confirmation or configuration.
- Suppressing critical filesystem I/O errors or corrupt JSON syntax that prevent CLI operation.

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL persist finding waivers in `.sdd/suppressions.json` containing file path, rule identifier, human reason, and date. | MUST | test |
| R-002 | The system SHALL downgrade suppressed findings in `sdd check` to `INFO` with the suppression reason, ensuring `--strict` does not exit with failure. | MUST | test |
| R-003 | The system SHALL display a `[SUPPRESSED]` indicator and reason in `sdd check --fix` diff previews for files with active waivers. | MUST | test |
| R-004 | The system SHALL offer a `[s]uppress` option in interactive `sdd check --fix` to create a waiver without modifying the file. | MUST | test |
| R-005 | The system SHALL automatically remove the corresponding suppression from `.sdd/suppressions.json` when a previously suppressed repair is applied. | MUST | test |
| R-006 | The system SHALL mark suppressed findings in `sdd check --json` output with `suppressed: true` and the waiver reason. | MUST | test |
| R-007 | The system SHALL confine all reads, updates, and writes of `.sdd/suppressions.json` within the project root using `os.Root` and atomic writes. | MUST | test |
| R-008 | The system SHALL detect stale waivers in `.sdd/suppressions.json` whose findings no longer trigger, reporting an `INFO` note in `sdd check` and offering interactive removal during `sdd check --fix`. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Finding matching, suppression filtering, and JSON serialization SHALL be strictly deterministic and reproducible.
- **Fail-safe behaviour:** A missing, empty, or malformed `.sdd/suppressions.json` file SHALL be handled gracefully without crashing.
- **Dependencies:** Standard library only (`D-007`).

## 6. Constraints and assumptions
- Suppressions must be explicit and human-approved.
- In `sdd check --fix`, repairs on suppressed files remain optional (answering `n` keeps the suppression intact).

## 7. Open questions
(None)

## 8. Acceptance criteria
- Given a project with contract drift on `GEMINI.md` and a registered waiver in `.sdd/suppressions.json`, when `sdd check --strict` runs, then it outputs `INFO` and exits with code 0.
- Given a project with a suppressed repair, when `sdd check --fix` runs, then the diff is marked `[SUPPRESSED]` and prompts whether to repair or leave suppressed.
- Given an unsuppressed repair in `sdd check --fix`, when the user types `s` and provides a reason, then the waiver is recorded in `.sdd/suppressions.json` and the file is untouched.
- Given a resolved finding with an existing waiver in `.sdd/suppressions.json`, when `sdd check` runs, it outputs an `INFO` note for the stale waiver; and when `sdd check --fix` runs, it offers to prune the stale entry.
