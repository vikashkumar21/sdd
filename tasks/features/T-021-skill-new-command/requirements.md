# Requirements — T-021 sdd skill new command with file and text ingestion

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-03)
Roadmap ref: `ROADMAP.md` T-021

## 1. Goal
Provide the `sdd skill new <name>` CLI command to scaffold or import standard-compliant agent skills (`agentskills.io`) from scratch, existing files, or raw text strings with collision detection, backup safety, and agent refinement scaffolding.

## 2. Context and references
- Mission / principles touched: Principle 1 (Deterministic), Principle 2 (Fail safe), Principle 5 (Never overwrite human work).
- Decisions this relies on: D-007 (Stdlib only), D-024 (Agent Skills standard adoption).
- Depends on tasks: T-020 (Parser, validator, and check integration).

## 3. Scope
**In scope**
- CLI command `sdd skill new <name>`:
  - Scaffolds `skills/<name>/SKILL.md` (or `.sdd/skills/<name>/SKILL.md` when `--internal` is provided).
  - Validates skill name against `agentskills.io` regex (`^[a-z0-9]+(?:-[a-z0-9]+)*$`) and max length 64 before touching disk.
  - Accepts `--description <text>` (1–1024 chars), defaulting deterministically if omitted.
  - Accepts `--from-file <path>` to import markdown/text from an existing file into the skill body.
  - Accepts `--from-text <string>` to import raw instructions directly into the skill body.
  - Generates structured progressive-disclosure sections (`## Overview & Triggers`, `## Step-by-Step Instructions`, `## Verification`, `## Troubleshooting`) and encapsulates raw ingested notes within an agent refinement block (`<!-- sdd:skill-refine:begin -->` ... `<!-- sdd:skill-refine:end -->`).
  - Accepts `--internal` to scaffold under `.sdd/skills/<name>/SKILL.md`.
  - Refuses to overwrite existing skills unless `--force` is specified.
  - On `--force`, creates an atomic timestamped backup in `.sdd/backup/` prior to replacement.
  - Automatically verifies generated skill with `internal/skill` validator before completion.
- Document the Agent Skill Authoring and Refinement Standard in `SKILLS.md`.

**Out of scope**
- Listing or indexing skills (`sdd skill list` is T-022).
- Exporting SDD contracts as a native agent skill (T-023).
- Calling external LLM models to summarize skill descriptions (pure deterministic stdlib logic).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL provide `sdd skill new <name>` to scaffold a standard `agentskills.io` skill in `skills/<name>/SKILL.md`. | MUST | test |
| R-002 | The system SHALL validate that `<name>` satisfies `^[a-z0-9]+(?:-[a-z0-9]+)*$` and is 1–64 characters before creating any files. | MUST | test |
| R-003 | The system SHALL populate the skill body from `--from-file <path>` or `--from-text <string>` when provided. | MUST | test |
| R-004 | The system SHALL require `--description <text>` or extract a default description if provided, validating that length is 1–1024 characters. | MUST | test |
| R-005 | The system SHALL support `--internal` to scaffold under `.sdd/skills/<name>/SKILL.md`. | MUST | test |
| R-006 | The system SHALL refuse to overwrite an existing skill directory or `SKILL.md` without `--force`. | MUST | test |
| R-007 | When `--force` is specified and the skill already exists, the system SHALL back up the existing skill to `.sdd/backup/` before overwriting. | MUST | test |
| R-008 | When ingesting raw content via `--from-file` or `--from-text`, the system SHALL wrap the content in an agent refinement block and scaffold standard progressive-disclosure sections (`Overview & Triggers`, `Step-by-Step Instructions`, `Verification`). | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Given identical inputs and clocks, generated skill files SHALL be byte-identical.
- **Fail-safe behaviour:** Invalid names, missing input files, non-existent directories, or duplicate skills without `--force` SHALL exit with code 1 and descriptive errors.
- **Standard Library Only:** Zero external dependencies (`D-007`).

## 6. Constraints and assumptions
- Skills conform to the `agentskills.io` standard validated in T-020.
- Default scaffold path is `<root>/skills/<name>/SKILL.md`.

## 7. Open questions
(None.)

## 8. Acceptance criteria
- Given `sdd skill new code-review --description "Review code for quality"`, `skills/code-review/SKILL.md` is created with valid YAML frontmatter and standard sections, passing `sdd check --strict`.
- Given an invalid name `sdd skill new Code_Review`, command fails immediately with an actionable regex validation error and no files are written.
- Given an existing `skills/code-review/SKILL.md`, running `sdd skill new code-review` fails with an "already exists" error.
- Given `sdd skill new code-review --force`, the previous version is backed up to `.sdd/backup/` and the new file is written.
- Given `sdd skill new deploy-helper --from-file deploy.txt --description "Deployment steps"`, the body of `deploy.txt` is wrapped with the refinement block and standard sections in `skills/deploy-helper/SKILL.md`.
