# Requirements — T-022 sdd skill list and progressive disclosure handover prompt

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-03)
Roadmap ref: `ROADMAP.md` T-022

## 1. Goal
Provide the `sdd skill list` CLI command to inspect available project and internal skills, and enrich `sdd prompt` with a progressive disclosure catalog of active skills and refinement guidance.

## 2. Context and references
- Mission / principles touched: Principle 1 (Deterministic), Principle 4 (Small and shippable), Principle 5 (Never overwrite human work).
- Decisions this relies on: D-007 (Stdlib only), D-024 (Agent Skills standard adoption).
- Depends on tasks: T-020 (Parser, validator, and check integration), T-021 (Skill creation and ingestion).

## 3. Scope
**In scope**
- CLI command `sdd skill list`:
  - Discovers all skills in `skills/` and `.sdd/skills/`.
  - Displays a deterministic formatted table showing skill name, scope (`project` or `internal`), validity status (`valid` or validation error count), and description.
  - Supports `--json` for machine-readable JSON output.
  - Returns exit code 0 when successful (even if no skills exist, printing a clear notice).
- Handover prompt enrichment (`sdd prompt`):
  - Automatically queries available skills in the project.
  - Follows the Anthropic / Linux Foundation progressive disclosure standard: renders only skill name and trigger description (~50-100 tokens total) so the agent knows what capabilities exist without loading large bodies into context prematurely.
  - Detects if any skill contains raw ingested notes (`<!-- sdd:skill-refine:begin -->`) and instructs the agent to refine them into standard sections under active tasks.
- Updates to `README.md` documenting:
  - `sdd skill list` and `--json` usage.
  - `sdd prompt` progressive disclosure skill catalog and agent refinement workflow.

**Out of scope**
- Native agent skill export for SDD itself (T-023).
- Remote skill downloading or registry interactions (offline, local-only per D-007).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL provide `sdd skill list` to enumerate all skills in `skills/` and `.sdd/skills/`. | MUST | test |
| R-002 | `sdd skill list` SHALL display name, scope (`project` or `internal`), status (`valid` or `invalid`), and description. | MUST | test |
| R-003 | `sdd skill list --json` SHALL output a structured JSON array of discovered skills. | MUST | test |
| R-004 | `sdd prompt` SHALL include an `Available Agent Skills (agentskills.io)` section when skills are present in the project. | MUST | test |
| R-005 | When skills contain `<!-- sdd:skill-refine:begin -->`, `sdd prompt` SHALL output guidance alerting the agent to refine raw notes into standard sections. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Output rows and JSON fields SHALL be sorted deterministically by skill name.
- **Fail-safe behaviour:** Malformed skills SHALL be reported with `invalid` status and their validation error without crashing.
- **Performance:** Listing skills SHALL execute in under 30ms.
- **Standard Library Only:** Zero external dependencies (`D-007`).

## 6. Constraints and assumptions
- Skills are discovered using `internal/skill.Discover`.
- Conforms to progressive disclosure: prompt includes metadata only, not full Markdown bodies.

## 7. Open questions
(None.)

## 8. Acceptance criteria
- Given a project with `skills/code-review/SKILL.md`, running `sdd skill list` prints a table containing `code-review`, scope `project`, and its description.
- Given a project with no skills, `sdd skill list` prints "No skills found." and exits 0.
- Given `--json`, `sdd skill list --json` outputs valid JSON parseable into a skill slice.
- Given `sdd prompt gemini` on a project with skills, the output contains the `Available Agent Skills` table.
- Given a skill with `sdd:skill-refine:begin`, `sdd prompt` includes the refinement note.
