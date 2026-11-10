# Requirements — T-004 sdd adopt: write a manifest for kits created before sdd (P-06, for example dbguard)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-02)
Roadmap ref: `ROADMAP.md` T-004

## 1. Goal
Inspect an existing hand-crafted or pre-sdd spec-driven development kit (such as `dbguard`), detect its structure, core documents, and agent contracts without altering any existing files, and write a valid `.sdd/manifest.json` (schema 1) so the kit passes `sdd check` and is enabled for future `sdd upgrade` workflows.

## 2. Context and references
- Mission / principles: Principle 1 (Deterministic), Principle 3 (Human in the loop), Principle 5 (Never overwrite human work).
- Decisions this relies on: D-017 (Approved - Manifest schema 1 and managed-block markers).
- Depends on tasks: T-002, T-007 (os.Root confinement).

## 3. Scope
**In scope**
- `sdd adopt` CLI command accepting `--dir`, `--dry-run`, `--yes`, `--force`, `--name`, `--maintainer`.
- Validation that the target directory contains a spec-driven kit (`MISSION.md`, `TECH_STACK.md`, `ROADMAP.md`, `SKILLS.md`).
- Detection of existing agent contracts (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`) and task templates (`features/_templates/*`).
- Calculation of SHA256 hashes for all discovered kit files without modifying their contents.
- Detection of managed blocks versus whole-file contracts.
- Atomic creation of `.sdd/manifest.json` confined by `os.Root`.
- Support for `--dry-run` to inspect the generated manifest JSON before writing.
- Refusal when `.sdd/manifest.json` already exists (unless `--force` is set, with backup).
- Verification that `sdd check` passes cleanly on adopted kits (including `testdata/dbguard-kit`).

**Out of scope**
- Modifying, reformatting, or regenerating any markdown files in the adopted kit.
- Upgrading templates during adopt (that is `sdd upgrade`, T-003).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL refuse `sdd adopt` if core documents (MISSION.md, TECH_STACK.md, ROADMAP.md, SKILLS.md) are missing. | MUST | test |
| R-002 | The system SHALL refuse `sdd adopt` if `.sdd/manifest.json` already exists, unless `--force` is specified. | MUST | test |
| R-003 | The system SHALL detect existing agent contracts (CLAUDE.md, GEMINI.md, AGENTS.md) and task templates and record them in the manifest. | MUST | test |
| R-004 | The system SHALL compute SHA256 hashes for all discovered kit files and mark their state as `existing` in the manifest. | MUST | test |
| R-005 | The system SHALL write ONLY `.sdd/manifest.json` and SHALL NOT modify or mutate any existing markdown or code files. | MUST | test |
| R-006 | The system SHALL support `--dry-run` to preview the planned manifest JSON to stdout without creating any files on disk. | MUST | test |
| R-007 | The system SHALL ensure that running `sdd check` on an adopted kit immediately after adoption produces zero errors. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Adopting the same kit multiple times SHALL produce identical manifest JSON.
- **Fail-safe behaviour:** Refuse cleanly if filesystem root or home directory is targeted.
- **Zero mutation:** Existing documentation and contract files remain byte-identical before and after adoption.
- **Dependencies:** Standard library only (`D-007`).

## 6. Constraints and assumptions
- The target repository is already a valid spec-driven kit containing the four core documents.

## 7. Open questions
(None.)

## 8. Acceptance criteria
- Given a hand-crafted kit like `testdata/dbguard-kit` with no manifest, when `sdd adopt` runs, then `.sdd/manifest.json` is generated, no existing files are changed, and `sdd check` passes with 0 errors.
- Given a directory lacking core documents, when `sdd adopt` runs, then it refuses with exit code 1 and an explanatory message.
- Given a project already having `.sdd/manifest.json`, when `sdd adopt` runs without `--force`, then it refuses with exit code 1.
