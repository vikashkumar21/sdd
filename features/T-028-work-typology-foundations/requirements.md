# Requirements — T-028 Work typology foundations: bugfixes, hotfixes, spikes, and task templates

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-05)
Roadmap ref: `ROADMAP.md` T-028

## 1. Goal
Implement production-grade work typology foundations supporting `features/`, `bugfixes/`, `hotfixes/`, and `spikes/`, with specialized lifecycle templates (mandatory failing reproduction test for bugs, post-mortem for hotfixes, non-merging spike prototypes), a 64KB specification context budget guard in `sdd check`, CLI scaffolding commands, and template version bump to `1.2.0`.

## 2. Context and references
- Mission / principles touched: Principle 1 (Deterministic), Principle 4 (Small and shippable), Principle 5 (Never overwrite human work)
- Decisions this relies on:
  - D-002: Spec-driven workflow with human gates
  - D-007: Go, standard library only, single static binary
  - D-026: Code review specification, 3-tier risk grading, and `review.md`
  - D-027: Work typology and specialized lifecycle artifacts
- Depends on tasks: T-027 (Done)

## 3. Scope
**In scope**
- Canonical work typology definitions: `feature`, `bugfix`, `hotfix`, `spike`.
- Directory structure: `features/T-xxx-<slug>/`, `bugfixes/T-xxx-<slug>/`, `hotfixes/T-xxx-<slug>/`, `spikes/T-xxx-<slug>/`.
- Embedded templates for all typologies in `internal/templates/data/`:
  - `bugfixes/`: `reproduction.md` (Step 0 minimal failing test), `rca.md` (root cause & blast radius), `plan.md`, `validation.md`, `review.md`.
  - `hotfixes/`: `incident.md` (severity, symptom, user impact), `patch.md` (minimal diff, no speculative refactor), `validation.md`, `post_mortem.md`.
  - `spikes/`: `hypotheses.md` (questions to answer, timebox), `findings.md` (learned facts, disposable prototype reference), `plan.md`.
  - `features/`: existing requirements, plan, validation, review.
- CLI subcommands:
  - `sdd bugfix new <T-xxx> <slug>`
  - `sdd hotfix new <T-xxx> <slug>`
  - `sdd spike new <T-xxx> <slug>`
  - `sdd task new --type <feature|bugfix|hotfix|spike> <T-xxx> <slug>`
  - `sdd feature new <T-xxx> <slug>` (maintained with full backward compatibility).
- Specification Context Budget Guard:
  - `sdd check` warns if any task document in `features/`, `bugfixes/`, `hotfixes/`, or `spikes/` exceeds 64KB (65,536 bytes).
- Mechanical validation in `sdd check` for bugfixes (verifying reproduction evidence), hotfixes, and spikes.
- Upgrading existing kits: `sdd upgrade` adds `bugfixes/_templates/`, `hotfixes/_templates/`, `spikes/_templates/`.
- Bump `templates.Version` to `1.2.0` in `internal/templates/templates.go`, updating contract headers in `CLAUDE.md`, `GEMINI.md`, `AGENTS.md` per Rule 14.
- Update `sdd status` to discover all 4 work directories.

**Out of scope**
- Git worktree provisioning (handled in T-029).
- Static plan file collision analysis (handled in T-030).
- Remote CI integration (Milestone 9).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL support four work typologies: `feature`, `bugfix`, `hotfix`, and `spike`, mapped to `features/`, `bugfixes/`, `hotfixes/`, and `spikes/` respectively. | MUST | test |
| R-002 | The system SHALL provide specialized embedded templates for: (a) `bugfixes/` (`reproduction.md`, `rca.md`, `plan.md`, `validation.md`, `review.md`); (b) `hotfixes/` (`incident.md`, `patch.md`, `validation.md`, `post_mortem.md`); (c) `spikes/` (`hypotheses.md`, `findings.md`, `plan.md`); (d) `features/` (`requirements.md`, `plan.md`, `validation.md`, `review.md`). | MUST | test / inspection |
| R-003 | The system SHALL mandate a Step 0 failing test case in `bugfixes/` `reproduction.md`, requiring proof that the test fails on unpatched code. | MUST | test / inspection |
| R-004 | The system SHALL provide CLI subcommands: `sdd bugfix new <T-xxx> <slug>`, `sdd hotfix new <T-xxx> <slug>`, `sdd spike new <T-xxx> <slug>`, and `sdd task new --type <type> <T-xxx> <slug>`, with alias `sdd feature new`. | MUST | test |
| R-005 | The system SHALL update `ROADMAP.md` task status to `Specifying` upon scaffolding any work typology task folder. | MUST | test |
| R-006 | `sdd check` SHALL enforce the 64KB context budget guard: if any task document in `features/`, `bugfixes/`, `hotfixes/`, or `spikes/` exceeds 65,536 bytes, it SHALL emit a `WARN` finding advising task decomposition or staging via `sdd ingest`. | MUST | test |
| R-007 | `sdd check` SHALL validate task folder completeness and status alignment across all 4 work typologies. | MUST | test |
| R-008 | `sdd upgrade` SHALL install `bugfixes/_templates/`, `hotfixes/_templates/`, `spikes/_templates/` into existing kits while preserving human edits. | MUST | test |
| R-009 | The system SHALL bump `templates.Version` from `1.1.0` to `1.2.0`, updating contract headers in `CLAUDE.md`, `GEMINI.md`, `AGENTS.md` per Rule 14. | MUST | test / inspection |
| R-010 | `sdd status` SHALL discover task folders in `features/`, `bugfixes/`, `hotfixes/`, and `spikes/` and indicate the work typology. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Scaffolding and checking must produce byte-identical results regardless of platform, newline convention, or clock.
- **Fail-safe:** If target directory exists, refuse without `--force`. If unknown typology is requested, exit 2 with actionable error.
- **Standard library only:** All parsers, templates, and checks implemented with Go standard library (`D-007`).
- **Rule 14 verification:** Generating a fresh kit in a temporary directory and running `sdd check --strict` on it passes with 0 errors.

## 6. Constraints and assumptions
- Task IDs remain sequential `T-xxx`.
- `ROADMAP.md` rows keep standard pipe-delimited format.

## 7. Open questions
(None - all decisions resolved in D-027)

## 8. Acceptance criteria
- Given `sdd bugfix new T-029 memory-leak`, then `bugfixes/T-029-memory-leak/` is created containing `reproduction.md`, `rca.md`, `plan.md`, `validation.md`, `review.md`, and `ROADMAP.md` marks T-029 as `Specifying`.
- Given a task markdown document exceeding 64KB, when `sdd check` runs, it outputs a warning highlighting the context budget overrun.
- Given an existing kit with only `features/_templates/`, running `sdd upgrade` populates `bugfixes/_templates/`, `hotfixes/_templates/`, and `spikes/_templates/`.
- Given a project with tasks in `bugfixes/` and `hotfixes/`, `sdd status` and `sdd check` recognize them seamlessly.
