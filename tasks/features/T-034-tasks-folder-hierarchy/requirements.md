# Requirements — T-034 Unified tasks/ directory hierarchy and sdd upgrade auto-migration

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-10)
Roadmap ref: `ROADMAP.md` T-034

## 1. Goal
Unify all task typology directories (`features/`, `bugfixes/`, `hotfixes/`, `spikes/`) under a single top-level `tasks/` folder (`tasks/features/`, `tasks/bugfixes/`, `tasks/hotfixes/`, `tasks/spikes/`) to eliminate root project clutter, update `sdd init` and task scaffolding commands, and provide seamless automated migration of existing projects in `sdd upgrade`.

## 2. Context and references
- Mission / principles touched:
  - Principle 1 (Deterministic): Deterministic task discovery and migration without loss of hashes or git status.
  - Principle 4 (Small and shippable): Clear vertical slice delivering uncluttered project ergonomics.
  - Principle 5 (Never overwrite human work): Non-destructive migration; existing task files, requirements, plans, and validation results are preserved exactly.
- Decisions this relies on:
  - D-002: Spec-driven workflow with human gates.
  - D-007: Standard library only (no external dependencies).
  - D-017: Manifest schema 1 and proposal format.
  - D-027: Work typology and specialized lifecycle artifacts (Features, Bugfixes, Hotfixes, Spikes).
  - D-030: Unified `tasks/` directory structure and `sdd upgrade` migration.
- Depends on tasks: T-033 (Done).

## 3. Scope
**In scope**
- Typology directory architecture (`internal/feature`, `internal/scaffold`, `internal/templates`):
  - Change default directory paths to `tasks/features`, `tasks/bugfixes`, `tasks/hotfixes`, `tasks/spikes`.
  - Update embedded templates in `internal/templates/data/tasks/`.
  - Bump `templates.Version` to `1.4.0` per Hard Rule 16.
- Task Scaffolding (`sdd feature new`, `sdd bugfix new`, `sdd hotfix new`, `sdd spike new`, `sdd task new`):
  - Create new task directories under `tasks/<typology>/T-xxx-<slug>/`.
- Universal Task Discovery (`internal/worktree`, `internal/coordinator`, `internal/deconflict`, `internal/status`, `internal/cli`):
  - Search `tasks/<typology>/` first; fall back gracefully to legacy root `<typology>/` for backward compatibility.
- Automated Upgrade Migration (`internal/upgrade`):
  - Detect legacy root directories (`features/`, `bugfixes/`, `hotfixes/`, `spikes/`) during `sdd upgrade`.
  - Migrate folders into `tasks/<typology>/`.
  - Update paths in `.sdd/manifest.json` preserving hashes.
  - Preview folder moves in `sdd upgrade --dry-run`.
- Mechanical Consistency Checks (`internal/check`):
  - Verify kits with `tasks/` layout, including template checks, review checks, and 64KB context limits.
- Agent Contracts & Documentation:
  - Update `AGENTS.md`, `CLAUDE.md`, `GEMINI.md` to reference `tasks/<typology>/` and `templates=1.4.0`.
  - Update `README.md`, `SKILLS.md`, and `TECH_STACK.md`.
- Self-Migration:
  - Migrate this repository's own task directories into `tasks/`.

**Out of scope**
- Pre-Gate 1 Plan Review Subagent (scheduled in T-035).
- 3-Subagent Parallel Review Triad (scheduled in T-035).
- YAML Frontmatter metadata filtering and `.ragignore` (scheduled in T-037).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | `sdd init` SHALL scaffold fresh project kits with all typology templates located in `tasks/<typology>/_templates/` (`tasks/features/`, `tasks/bugfixes/`, `tasks/hotfixes/`, `tasks/spikes/`). | MUST | test / inspection |
| R-002 | `sdd feature new`, `sdd bugfix new`, `sdd hotfix new`, `sdd spike new`, and `sdd task new` SHALL scaffold task folders under `tasks/<typology>/T-xxx-<slug>/` when run in projects using the unified layout. | MUST | test |
| R-003 | Task discovery across all commands (`sdd check`, `sdd status`, `sdd prompt review`, `sdd task start`, `sdd task deconflict`, `sdd task merge`) SHALL search `tasks/<typology>/` as primary and support legacy root `<typology>/` as fallback. | MUST | test |
| R-004 | `sdd upgrade` SHALL detect legacy root typology folders (`features/`, `bugfixes/`, `hotfixes/`, `spikes/`), automatically migrate them into `tasks/<typology>/`, and update `.sdd/manifest.json` path mappings. | MUST | test |
| R-005 | `sdd upgrade --dry-run` SHALL report planned folder migrations without writing or moving filesystem paths. | MUST | test |
| R-006 | `sdd check` SHALL validate task lifecycles, review status, and context budgets under `tasks/<typology>/` without regressions. | MUST | test |
| R-007 | Embedded template assets in `internal/templates/data/` SHALL be organized under `tasks/`, and `templates.Version` SHALL be bumped to `1.4.0`. | MUST | test |
| R-008 | Agent contracts (`AGENTS.md`, `CLAUDE.md`, `GEMINI.md`) SHALL be updated in lockstep to reference `tasks/` paths and `templates=1.4.0`. | MUST | test / inspection |
| R-009 | This repository's own tasks SHALL be migrated to `tasks/features/`, `tasks/bugfixes/`, etc., leaving the root clean. | MUST | inspection |
| R-010 | The entire test suite (`make verify`, `make check`) and mechanical validation (`sdd check --strict`) SHALL pass with 0 errors and 0 warnings. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Folder migration and manifest updating must be byte-identical and order-independent across runs.
- **Zero data loss:** Legacy task files and human customizations are never deleted or truncated during migration.
- **Standard library purity:** Standard library only; no external dependencies (D-007).

## 6. Constraints and assumptions
- All existing task IDs (T-001 through T-034) remain intact and keep their sequential naming.
- `tasks/` becomes the single source of truth for work items.

## 7. Open questions
(None — settled by approved Decision D-030)

## 8. Acceptance criteria
- Running `sdd init` in a fresh directory produces `tasks/features/_templates/`, `tasks/bugfixes/_templates/`, etc.
- Running `sdd feature new T-035 plan-review-subagent` creates `tasks/features/T-035-plan-review-subagent/`.
- Running `sdd upgrade` in a legacy kit moves `features/` into `tasks/features/` and updates manifest paths seamlessly.
- `make check` and `sdd check --strict` pass with 0 errors and 0 warnings.
