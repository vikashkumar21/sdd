---
status: draft
---
# Requirements — T-050 Path hygiene validator and zero-personal-path enforcement

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-21)
Roadmap ref: `ROADMAP.md` T-050

## 1. Goal
Implement mechanical path hygiene verification in `sdd check` under rule `path-hygiene` to enforce binding Human Feedback Rule `F-002`, flagging developer-specific local absolute home directory paths in markdown documents and templates while permitting standard placeholders, CI runner paths, and rule definitions.

## 2. Context and references
- Mission / principles touched: Fail safe, Deterministic, Living documents.
- Decisions this relies on: `D-004` (Toolchain & check), `D-007` (Go stdlib only).
- Active feedback rules: `F-001` (Resolve all review findings), `F-002` (Zero developer-specific local absolute paths in public docs/templates).
- Depends on tasks: `T-048` (Multi-agent coordinator contract).

## 3. Scope
**In scope**
- Implement `checkPathHygiene(rel, text string)` in `internal/check/paths.go` integrated into `sdd check`.
- Detect macOS (`/Users/<name>`), Linux (`/home/<name>`), and Windows (`[A-Za-z]:\Users\<name>`) developer home directories.
- Whitelist standard generic placeholders (`username`, `example`, `user`, `yourname`, `runner`, `admin`, `default`, ellipsis `...`) and rule self-references.
- Surface findings as `WARN` with rule `path-hygiene` (enforced under `sdd check --strict`).
- Support waivers in `.sdd/suppressions.json`.
- Sweep and sanitize historical personal path in `tasks/features/T-005-independent-review/review.md`.
- Update `deconflict.ExtractPlanFiles` table header parser to evaluate column 1 specifically, preventing false exclusion of plan files containing "path" in filename or description.
- Comprehensive unit tests in `internal/check/paths_test.go` and `internal/deconflict/deconflict_test.go`.

**Out of scope**
- Arbitrary non-home absolute paths (e.g. `/tmp/`, `/var/`, `/usr/`, `/etc/`).
- Codebase-wide AST scanning for Go string literals (checked by existing linters/code review).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL scan all markdown files returned by `c.markdownFiles()` during `sdd check` for hardcoded local developer home directory paths. | MUST | test |
| R-002 | The system SHALL match macOS home paths (`/Users/<name>`), Linux home paths (`/home/<name>`), and Windows home paths (`[A-Za-z]:\Users\<name>`). | MUST | test |
| R-003 | The system SHALL exempt generic placeholders (`username`, `example`, `user`, `yourname`, `runner`, `admin`, `default`, and paths ending with `...`) and lines referencing Rule `F-002` or `path-hygiene`. | MUST | test |
| R-004 | The system SHALL report violations as `WARN` findings with Rule `path-hygiene` citing the offending path, line number, and filename. | MUST | test |
| R-005 | The system SHALL support suppression of path hygiene findings via `.sdd/suppressions.json` matching file and rule `path-hygiene`. | MUST | test |
| R-006 | The system SHALL sanitize historical task review artifact `tasks/features/T-005-independent-review/review.md` to eliminate personal local machine path. | MUST | inspection / test |
| R-007 | The system SHALL pass `sdd check --strict` with zero path hygiene warnings across the entire repository. | MUST | test |
| R-008 | The system SHALL parse plan target files containing "path" in filename or description without false header exclusion in `deconflict.ExtractPlanFiles`. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Regex inspection is byte-deterministic, order-preserving, and injected-clock independent.
- **Performance:** Linear scan over lines; adds < 5ms to `sdd check`.
- **Standard Library Only:** Zero external dependencies per `D-007`.

## 6. Constraints and assumptions
- Paths in `.git/`, `.sdd/worktrees/`, `.sdd/backup/`, and binary files are already excluded by `markdownFiles()`.
- CI runner home paths (e.g. `/home/runner` on GitHub Actions) are recognized as valid infrastructure paths.

## 7. Open questions
(None — all requirements settled by Rule F-002 and user confirmation)

## 8. Acceptance criteria
- **AC-1:** Given a markdown file containing `/Users/alice/projects/acme`, when `sdd check` runs, then a `WARN` finding with rule `path-hygiene` is reported on that exact line.
- **AC-2:** Given a markdown file containing `/home/runner/work/repo`, when `sdd check` runs, then no warning is reported.
- **AC-3:** Given `sdd check --strict` on the sdd repository, 0 `path-hygiene` findings are reported.
