# Plan — T-034 Unified tasks/ directory hierarchy and sdd upgrade auto-migration

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-10)

## 1. Approach summary
Relocate all task typologies (`features`, `bugfixes`, `hotfixes`, `spikes`) under a single `tasks/` top-level directory (`tasks/features/`, `tasks/bugfixes/`, etc.) across template data, scaffolding, and CLI discovery. Implement an automated migration step in `sdd upgrade` that seamlessly moves legacy root folders into `tasks/` and updates `.sdd/manifest.json`. Bump `templates.Version` to `1.4.0` per Hard Rule 16, synchronize agent contracts (`AGENTS.md`, `CLAUDE.md`, `GEMINI.md`), and migrate this repository's own task directories into `tasks/`.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-030 | Unified `tasks/` directory structure and `sdd upgrade` migration | Option A (tasks/<typology>/) vs Option B (flat tasks/) | Option A | Approved (2026-11-10) |
| D-027 | Work typology and specialized lifecycle artifacts | Option A (root directories) -> Updated by D-030 | Option A | Approved (2026-11-05) |

## 3. Design

### Typology Directory & Scaffolding (`internal/feature`, `internal/scaffold`)
- Update `Typology.Directory()`:
  - If project has legacy `<typology>` folder and not `tasks/`, support legacy path during transition.
  - Default / fresh kits return `tasks/features`, `tasks/bugfixes`, `tasks/hotfixes`, `tasks/spikes`.
- Reorganize embedded templates:
  - Move `internal/templates/data/features/` -> `internal/templates/data/tasks/features/`
  - Move `internal/templates/data/bugfixes/` -> `internal/templates/data/tasks/bugfixes/`
  - Move `internal/templates/data/hotfixes/` -> `internal/templates/data/tasks/hotfixes/`
  - Move `internal/templates/data/spikes/` -> `internal/templates/data/tasks/spikes/`
- Bump `templates.Version` to `1.4.0` in `internal/templates/templates.go`.

### Universal Task Discovery (`internal/worktree`, `internal/status`, `internal/cli`)
- Update task lookup helpers (`FindTaskLocation`, `findTaskFolder`, `findTaskDirectory`):
  - Check `tasks/<typology>/<taskID>-*` first.
  - Fall back to `<typology>/<taskID>-*` if not found in `tasks/`.
  - Return canonical relative path.

### Upgrade Migration Engine (`internal/upgrade`)
- When `sdd upgrade` executes:
  - Check if `tasks/` exists or if any legacy root folders (`features/`, `bugfixes/`, `hotfixes/`, `spikes/`) exist.
  - If legacy root folders exist:
    - Create `tasks/` directory.
    - Move `features/` -> `tasks/features/`, `bugfixes/` -> `tasks/bugfixes/`, etc.
    - Update entry paths in `.sdd/manifest.json` from `features/...` to `tasks/features/...` while preserving recorded file hashes.
    - If `--dry-run`, print planned folder migrations and manifest updates without touching disk.

### Consistency Checks (`internal/check`)
- Update check engine to validate `tasks/<typology>/` layout.
- Ensure `TestFreshKitHasNoErrors` verifies fresh kits generated with template version `1.4.0`.

### Agent Contract Sync & Self-Migration
- Update `AGENTS.md`, `CLAUDE.md`, `GEMINI.md` and `contract.md.tmpl` marker to `<!-- sdd:begin contract templates=1.4.0 -->`.
- Update workflow references in contracts: `tasks/features/T-xxx-<kebab-slug>/`.
- Move this repository's existing task directories into `tasks/`.

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Reorganize embedded template assets into `internal/templates/data/tasks/` and bump `templates.Version` to `1.4.0`. | `internal/templates/templates.go`, `internal/templates/data/` | `go test -v ./internal/templates` |
| 2 | Update `internal/feature` and `internal/scaffold` to generate and scaffold under `tasks/<typology>/`. | `internal/feature/feature.go`, `internal/feature/feature_test.go`, `internal/scaffold/` | `go test -v ./internal/feature -run TestNew` |
| 3 | Update universal task discovery in `worktree`, `status`, `deconflict`, `coordinator`, and `cli`. | `internal/worktree/`, `internal/status/`, `internal/cli/`, `internal/coordinator/` | `go test -v ./internal/worktree ./internal/status ./internal/cli` |
| 4 | Implement legacy directory migration and manifest rewriting in `internal/upgrade`. | `internal/upgrade/upgrade.go`, `internal/upgrade/upgrade_test.go` | `go test -v ./internal/upgrade` |
| 5 | Update `internal/check` to validate `tasks/<typology>/` kits and fresh kit generation. | `internal/check/`, `internal/check/check_test.go` | `go test -v ./internal/check` |
| 6 | Synchronize agent contracts (`AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `contract.md.tmpl`) and migrate repo's own task folders into `tasks/`. | Agent files, living docs, `tasks/` | `make check`, `sdd check --strict` |
| 7 | Run verification suite (`make verify`, `make check`), invoke independent subagent review, and prepare GATE 2. | `features/T-034-tasks-folder-hierarchy/` | `make check`, `review.md` sign-off |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/templates/templates.go` | modify | Bump templates.Version to 1.4.0 |
| `internal/templates/templates_test.go` | modify | Update version assertion |
| `internal/templates/data/tasks/` | create | Embedded template data relocated under tasks/ |
| `internal/templates/data/base/contract.md.tmpl` | modify | Update contract template with tasks/ paths and templates=1.4.0 |
| `internal/feature/feature.go` | modify | Update typology directory paths to tasks/<typology> |
| `internal/feature/feature_test.go` | modify | Test tasks/ typology paths |
| `internal/scaffold/plan.go` | modify | Scaffold manifest with tasks/ template paths |
| `internal/scaffold/scaffold_test.go` | modify | Test fresh kit generation under tasks/ |
| `internal/worktree/worktree.go` | modify | Universal task discovery in tasks/ |
| `internal/worktree/worktree_test.go` | modify | Test worktree discovery |
| `internal/status/status.go` | modify | Status command task discovery |
| `internal/status/status_test.go` | modify | Test status with tasks/ |
| `internal/cli/prompt.go` | modify | Review prompt task discovery |
| `internal/cli/task.go` | modify | Task command path builders |
| `internal/cli/cli_test.go` | modify | Test CLI commands with tasks/ |
| `internal/upgrade/upgrade.go` | modify | Auto-migration of legacy root folders to tasks/ |
| `internal/upgrade/upgrade_test.go` | modify | Test legacy migration and manifest updating |
| `internal/check/check.go` | modify | Update check engine for tasks/ structure |
| `internal/check/check_test.go` | modify | Test check engine with tasks/ |
| `AGENTS.md` | modify | Update contract with tasks/ paths and templates=1.4.0 |
| `CLAUDE.md` | modify | Update contract with tasks/ paths and templates=1.4.0 |
| `GEMINI.md` | modify | Update contract with tasks/ paths and templates=1.4.0 |
| `SKILLS.md` | modify | Update references to tasks/ |
| `README.md` | modify | Update documentation with tasks/ directory layout |
| `ROADMAP.md` | modify | Update T-034 status |
| `TECH_STACK.md` | modify | Record verified fact for T-034 |
| `tasks/` (migrated from `features/`, etc.) | move | Repository's own task directories migrated |
| `features/T-034-tasks-folder-hierarchy/requirements.md` | modify | Requirements spec |
| `features/T-034-tasks-folder-hierarchy/plan.md` | modify | Implementation plan |
| `features/T-034-tasks-folder-hierarchy/validation.md` | modify | Validation matrix |
| `features/T-034-tasks-folder-hierarchy/review.md` | modify | Subagent review audit |

## 6. Living docs to update

| File | Update when |
|---|---|
| `ROADMAP.md` | T-034 status transitions |
| `SKILLS.md` | References to tasks/ structure updated |
| `README.md` | Document tasks/ directory hierarchy |
| `TECH_STACK.md` | Verified facts log updated |
| `AGENTS.md`, `CLAUDE.md`, `GEMINI.md` | Workflow paths updated to tasks/ |

## 7. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Breaking existing projects that still have root `features/` | Multi-path fallback discovery: look in `tasks/` first, fall back to root `<typology>/` seamlessly |
| Data loss during `sdd upgrade` folder migration | Atomic directory moves, dry-run preview, and manifest validation |
| Git merge conflicts with active branches | Standardized branch names remain `feature/T-xxx` regardless of on-disk directory |

## 8. Rollback plan
Revert all file modifications with `git checkout -- .` and move back any migrated directories.

## 9. Approval record
Approved by: Vikash (human)   Date: 2026-11-10   Phrase received: `APPROVED T-034`
