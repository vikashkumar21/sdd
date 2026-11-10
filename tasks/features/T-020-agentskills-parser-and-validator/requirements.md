# Requirements — T-020 agentskills.io parser, validator, and sdd check integration

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-03)
Roadmap ref: `ROADMAP.md` T-020

## 1. Goal
Implement a pure Go standard-library parser and validator for the `agentskills.io` specification and integrate skill validation rules into `sdd check`.

## 2. Context and references
- Mission / principles: Principle 1 (Deterministic), Principle 4 (Small and shippable), Principle 5 (Never overwrite human work).
- Decisions this relies on: D-007 (Stdlib only, no third-party dependencies), D-024 (Approved - Agent Skills standard adoption).
- Depends on tasks: T-001 (Core check engine).

## 3. Scope
**In scope**
- Package `internal/skill`:
  - `Skill` data model representing an agent skill (name, description, license, compatibility, metadata, allowed-tools, markdown body).
  - Pure Go standard library YAML frontmatter parser (supports strings, multiline strings, string slices, and simple key-value maps without external dependencies).
  - Validation against the official `agentskills.io` standard:
    - Name: 1–64 characters, lowercase alphanumeric and single hyphens (`^[a-z0-9]+(?:-[a-z0-9]+)*$`).
    - Name-to-directory match: The skill `name` must match the parent folder name.
    - Description: 1–1024 characters, non-empty and non-whitespace.
    - Compatibility: max 500 characters when provided.
  - Skill directory discovery in `.sdd/skills/` and `skills/`.
- Integration into `internal/check` (`sdd check`):
  - Rule to scan project skill directories.
  - Report findings for missing frontmatter, invalid naming regex, mismatched directory names, missing/empty descriptions, and oversized fields.

**Out of scope**
- External YAML libraries (`gopkg.in/yaml.v3`) — preserving `D-007` stdlib purity.
- Running executable code inside `scripts/` (validation only).
- CLI scaffolding commands (`sdd skill new` is in T-021).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL parse YAML frontmatter delimited by `---` from `SKILL.md` files using only the Go standard library. | MUST | test |
| R-002 | The system SHALL validate that skill `name` matches `^[a-z0-9]+(?:-[a-z0-9]+)*$` and is 1 to 64 characters long. | MUST | test |
| R-003 | The system SHALL validate that skill `name` matches its parent directory name when loaded from disk. | MUST | test |
| R-004 | The system SHALL validate that `description` is non-empty and does not exceed 1024 characters. | MUST | test |
| R-005 | The system SHALL validate that `compatibility` does not exceed 500 characters when present. | MUST | test |
| R-006 | `sdd check` SHALL discover skills in `.sdd/skills/` and `skills/`, reporting actionable findings for malformed skills. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Parsing and validating skills SHALL produce byte-identical results and consistent finding ordering.
- **Fail-safe behaviour:** Corrupted, truncated, or invalid YAML frontmatter SHALL return descriptive parse errors without panicking.
- **Dependencies:** Standard library only (`D-007`). Zero external dependencies.
- **Performance:** Scanning and validating skill directories SHALL complete in under 50ms.

## 6. Constraints and assumptions
- Skills are stored in `.sdd/skills/<name>/SKILL.md` or `skills/<name>/SKILL.md`.
- Conforms to the Linux Foundation / Anthropic `agentskills.io` open standard specification.

## 7. Open questions
(None. Resolved via Decision D-024.)

## 8. Acceptance criteria
- Given a valid `skills/code-review/SKILL.md` adhering to `agentskills.io`, `skill.Parse` extracts metadata and body, and `sdd check` reports 0 errors.
- Given a skill with an invalid name (e.g. `Code_Review` or `code--review`), `sdd check` reports an error with line location.
- Given a skill whose `name` does not match its parent directory, `sdd check` reports an error.
- Given a skill with a missing or empty `description`, `sdd check` reports an error.
- Given a skill missing YAML frontmatter delimiters, `sdd check` reports an error.
