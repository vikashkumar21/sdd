# Requirements — T-012 P-09: Safe interactive repair command (sdd check --fix)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-02)
Roadmap ref: `ROADMAP.md` T-012

## 1. Goal
Provide a safe, interactive repair command (`sdd check --fix`) that mechanically fixes agent contract drift and markdown table formatting, displays a unified diff preview, requires human confirmation, and backs up original files before applying changes.

## 2. Context and references
- Mission / principles touched: Principles 1 (Deterministic), 2 (Fail safe), 3 (Human in the loop for the irreversible), 5 (Never overwrite human work outside managed blocks)
- Decisions this relies on: `D-003` (Go), `D-004` (Toolchain), `D-007` (Standard library only), `D-010` (Managed-block merge), `D-017` (Managed-block markers)
- Depends on tasks: `T-002` (Done), `T-008` (Robust parser, Done), `T-009` (JSON output, Done)

## 3. Scope
**In scope**
- Interactive repair flag `sdd check --fix` with `--dry-run` and `--yes` support.
- Mechanical repair of agent contract drift across `CLAUDE.md`, `GEMINI.md`, and `AGENTS.md` using the canonical managed block, preserving text outside managed blocks.
- Mechanical repair of malformed markdown table delimiter rows and column alignment.
- Unified diff preview of proposed changes prior to execution.
- Mandatory backup of original files into `.sdd/backup/<timestamp>/` using atomic writes via `os.Root`.
- Unit and integration tests for contract repair, table repair, dry run, and backup creation.

**Out of scope**
- Auto-fixing semantic logic errors, unverified facts, or missing requirement files.
- Modifying human work outside marked blocks (guarantee S1).
- Non-deterministic or heuristic edits without exact mechanical rules.

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL support the `--fix` flag on `sdd check` to plan and execute safe mechanical repairs. | MUST | test |
| R-002 | The system SHALL synchronize managed agent contracts across `CLAUDE.md`, `GEMINI.md`, and `AGENTS.md` while preserving all content outside managed blocks. | MUST | test |
| R-003 | The system SHALL repair malformed markdown table delimiters and column counts to restore table validity. | MUST | test |
| R-004 | The system SHALL display a readable unified diff preview of all proposed file repairs prior to applying changes. | MUST | test |
| R-005 | The system SHALL support `--dry-run` to preview the diff without modifying files or creating backups. | MUST | test |
| R-006 | The system SHALL require interactive human confirmation before writing changes, unless `--yes` is specified. | MUST | test |
| R-007 | The system SHALL create backups of all modified files in `.sdd/backup/<timestamp>/` prior to writing modifications. | MUST | test |
| R-008 | The system SHALL perform all file reads, backups, and writes confined to the project root using `os.Root` and atomic writes. | MUST | test |
| R-009 | The system SHALL support selective interactive repair (`[y]es / [n]o / [p]er-file`), prompting individually for each file when `[p]` is chosen. | MUST | test |
| R-010 | The system SHALL support the `--file` flag on `sdd check --fix` to filter and apply repairs only to specified file paths. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Given the same corrupted input file, repair planning and diff generation SHALL produce identical outputs.
- **Fail-safe behaviour:** If an agent file has no managed block or is corrupt beyond mechanical repair, the system SHALL refuse to guess and skip the file.
- **Backup integrity:** Any failed repair write SHALL leave the original file intact and backups in `.sdd/backup/`.
- **Dependencies:** Standard library only (`D-007`).

## 6. Constraints and assumptions
- All writes must respect principle S1: never overwrite human work outside managed blocks.
- Non-interactive scripts in CI can pass `--yes` or `--dry-run`.

## 7. Open questions
(None)

## 8. Acceptance criteria
- Given drifted agent files where `GEMINI.md` has an outdated contract block, when `sdd check --fix --yes` runs, then `GEMINI.md` is updated to match `AGENTS.md`, text outside the block is preserved, and the original is backed up in `.sdd/backup/`.
- Given a markdown file with a table delimiter row missing a column, when `sdd check --fix --dry-run` runs, then a unified diff is displayed showing the corrected delimiter row and no files are written.
