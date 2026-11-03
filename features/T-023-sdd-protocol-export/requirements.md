# Requirements — T-023 SDD protocol export as native agent skill

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-03)
Roadmap ref: `ROADMAP.md` T-023

## 1. Goal
Provide the `sdd skill export` CLI command to export the canonical Spec-Driven Development (SDD) operating protocol as a standard `agentskills.io` agent skill (`sdd-protocol`) for use in IDEs, global agent skill configurations, and programmatic Agent SDK applications.

## 2. Context and references
- Mission / principles touched: Principle 1 (Deterministic), Principle 4 (Small and shippable), Principle 5 (Never overwrite human work).
- Decisions this relies on: D-007 (Stdlib only), D-024 (Agent Skills standard adoption).
- Depends on tasks: T-020 (Parser and validator), T-021 (Skill creation engine).

## 3. Scope
**In scope**
- Canonical `sdd-protocol` skill content:
  - Formatted strictly according to `agentskills.io` specifications (`internal/skill`).
  - Name: `sdd-protocol`.
  - Description explaining *when* and *why* an agent should trigger SDD (strict gates, deterministic coding, human approvals).
  - Contains the non-negotiable Anthem, Hard Rules (no git writes, gate approval before code, human commits), 4-phase workflow, decision protocol, and verification standards.
  - Designed for universal agent consumption: usable in Antigravity, Claude Code, Cursor, and programmatic Agent SDKs (Python/TypeScript/Go).
- CLI command `sdd skill export`:
  - Scaffolds `sdd-protocol/SKILL.md` into `<project>/skills/sdd-protocol/SKILL.md` by default.
  - Accepts `--target <dir>` to export to custom paths (such as SDK skill directories).
  - Accepts `--global` to export to the user's global agent skill directory (`~/.gemini/antigravity/skills/` or `~/.claude/skills/`).
  - Refuses to overwrite an existing skill without `--force` (backs up previous versions to `.sdd/backup/`).
  - Validates generated skill with `internal/skill` prior to completion.
- Updates to `README.md` and `SKILLS.md` documenting `sdd skill export` and SDK agent integration.

**Out of scope**
- Uploading skills to public internet registries (offline, local-first per D-007).
- Generating programming-language-specific SDK wrappers (Go/Python code generation).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL provide `sdd skill export` to generate a standard `agentskills.io` skill for the SDD protocol. | MUST | test |
| R-002 | The exported skill SHALL be named `sdd-protocol` and conform to all validation rules in `internal/skill`. | MUST | test |
| R-003 | The system SHALL support `--target <dir>` to write `sdd-protocol/SKILL.md` into an arbitrary directory. | MUST | test |
| R-004 | The system SHALL support `--global` to export into the user's home agent skill directory. | MUST | test |
| R-005 | The system SHALL refuse to overwrite an existing skill without `--force`. | MUST | test |
| R-006 | When `--force` is specified and the skill exists, the system SHALL create a backup before overwriting. | MUST | test |
| R-007 | The exported `sdd-protocol` skill SHALL be self-contained and usable by programmatic Agent SDKs. | MUST | test / inspection |

## 5. Non-functional requirements
- **Determinism:** Two consecutive exports to the same target SHALL produce byte-identical files.
- **Fail-safe behaviour:** Invalid targets, non-existent directories without permission, or collisions without `--force` SHALL return descriptive errors and exit code 1.
- **Standard Library Only:** Zero external dependencies (`D-007`).

## 6. Constraints and assumptions
- Target directory defaults to `<project-root>/skills`.
- Conforms to `agentskills.io` standard.

## 7. Open questions
(None.)

## 8. Acceptance criteria
- Given `sdd skill export` in a project, `skills/sdd-protocol/SKILL.md` is created and passes `sdd check --strict`.
- Given `sdd skill export --target ./sdk-skills`, `./sdk-skills/sdd-protocol/SKILL.md` is created with valid YAML frontmatter.
- Given an existing `skills/sdd-protocol/SKILL.md`, running `sdd skill export` without `--force` fails with an "already exists" error.
- Given `sdd skill export --force`, existing files are backed up to `.sdd/backup/`.
