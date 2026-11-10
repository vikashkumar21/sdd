# Requirements — T-037 Document lifecycle YAML frontmatter metadata and .ragignore generation

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-10)
Roadmap ref: `ROADMAP.md` T-037

## 1. Goal
Codify document lifecycle metadata via standard Markdown YAML frontmatter (`status`, `task_id`, `superseded_by`, `last_reviewed`), generate a project-level `.ragignore` file during `sdd init` and `sdd upgrade` to exclude ephemeral artifacts from RAG/vector indexing pipelines, and mechanically validate metadata integrity in `sdd check`.

## 2. Context and references
- Mission / principles touched:
  - Principle 1 (Deterministic): Deterministic lifecycle schema and RAG ignore patterns.
  - Principle 2 (Fail safe): Missing or invalid metadata reported cleanly by `sdd check`.
  - Principle 5 (Never overwrite): Existing user frontmatter and files respected; `.ragignore` generated non-destructively.
- Decisions this relies on:
  - D-002: Spec-driven workflow with human gates.
  - D-007: Standard library only (no external dependencies).
  - D-017: Manifest schema 1 and proposal format.
  - D-030: Unified `tasks/` directory structure.
  - D-033: Lifecycle YAML frontmatter metadata and `.ragignore` for RAG filtering.
- Depends on tasks: T-034 (Done), T-035 (Done), T-036 (Done).

## 3. Scope
**In scope**
- Document Lifecycle YAML Frontmatter Specification:
  - Fields: `status` (`active`, `draft`, `completed`, `superseded`, `deprecated`), `task_id` (optional), `superseded_by` (relative path to replacement doc), `last_reviewed` (`YYYY-MM-DD`).
  - Distinction from non-lifecycle frontmatter: Frontmatter containing no lifecycle fields (such as `skills/*/SKILL.md` with `name` and `description`) is ignored by the lifecycle validator.
- Core Parser & Validator (`internal/doc`):
  - Pure Go standard library frontmatter parser extracting metadata, frontmatter presence flag, and markdown body.
  - Validator checking status enum, date formatting (`YYYY-MM-DD`), and ensuring `superseded_by` exists when `status` is `superseded` (checked relative to repo root, falling back to document directory).
- `.ragignore` Generation:
  - Embedded template `base/.ragignore.tmpl` containing canonical ignore rules (`.sdd/`, `bin/`, `dist/`, `node_modules/`, `vendor/`, `*.tmp`, `*.bak`).
  - `sdd init` generates `.ragignore` in project root.
  - `sdd upgrade` detects missing `.ragignore` and installs it.
- Mechanical Verification (`internal/check`):
  - Rule `ragignore-missing`: Check that `.ragignore` exists in project root (warning if missing).
  - Rule `lifecycle-status`: Validate status enum values (`active`, `draft`, `completed`, `superseded`, `deprecated`).
  - Rule `lifecycle-date`: Validate `last_reviewed` matches `YYYY-MM-DD`.
  - Rule `lifecycle-superseded`: If `status: superseded`, verify target file in `superseded_by` is non-empty and exists.
- Template & Contract Synchronization:
  - Add YAML frontmatter to base templates (`base/MISSION.md.tmpl`, `base/TECH_STACK.md.tmpl`, `base/ROADMAP.md.tmpl`, `base/SKILLS.md.tmpl`) and task templates (`tasks/features/`, `tasks/bugfixes/`, etc.).
  - Bump `templates.Version` to `1.7.0` in `internal/templates/templates.go`.
  - Synchronize `contract.md.tmpl`, `AGENTS.md`, `CLAUDE.md`, and `GEMINI.md` in lockstep.
  - Update `.sdd/manifest.json`.

**Out of scope**
- Native sdd self-update CLI binary updater (scheduled in T-038).
- Calling external vector databases or embedding models (strictly local file generation per D-007 and Mission Non-goals).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL define a standard Markdown YAML frontmatter schema supporting `status` (`active`, `draft`, `completed`, `superseded`, `deprecated`), `task_id`, `superseded_by`, and `last_reviewed`. | MUST | test / inspection |
| R-002 | `internal/doc` SHALL provide pure Go standard library functions to parse and validate document lifecycle frontmatter without third-party dependencies, ignoring non-lifecycle frontmatter. | MUST | test |
| R-003 | `sdd init` SHALL generate a root `.ragignore` file containing canonical exclude patterns for internal and ephemeral files (`.sdd/`, `bin/`, `dist/`, `node_modules/`, `vendor/`, `*.tmp`, `*.bak`). | MUST | test |
| R-004 | `sdd upgrade` SHALL detect missing `.ragignore` in existing SDD projects and generate it automatically. | MUST | test |
| R-005 | `sdd check` SHALL check for the existence of `.ragignore` in project root, emitting a Warning under rule `ragignore-missing` if missing. | MUST | test |
| R-006 | `sdd check` SHALL validate lifecycle frontmatter when present in Markdown files, reporting warnings for invalid status values (`lifecycle-status`), malformed dates (`lifecycle-date`), or nonexistent `superseded_by` targets (`lifecycle-superseded`). | MUST | test |
| R-007 | Base document templates and task templates SHALL be updated with lifecycle frontmatter and `.ragignore.tmpl`, bumping `templates.Version` to `1.7.0`. | MUST | test |
| R-008 | Agent contracts (`contract.md.tmpl`, `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`) SHALL be synchronized in lockstep to `templates=1.7.0`. | MUST | test |
| R-009 | The entire test suite (`make verify`, `make check`), Hard Rule 16 fresh kit generation, and mechanical check (`sdd check --strict`) SHALL pass with 0 errors and 0 warnings. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Frontmatter parsing and `.ragignore` generation are byte-identical across repeat runs.
- **Fail safe:** Missing or invalid frontmatter is flagged without crashing; unknown fields are preserved or rejected deterministically.
- **Standard library purity:** 100% standard library (D-007). Zero new dependencies.

## 6. Constraints and assumptions
- Frontmatter follows standard YAML delimiters `---` at the beginning of the file.
- Markdown documents without frontmatter continue to be supported (frontmatter is optional on arbitrary docs, but validated whenever present).
- `superseded_by` paths are resolved relative to the repository root first, falling back to the document directory.

## 7. Open questions
(None — settled by approved Decision D-033).

## 8. Acceptance criteria
- Running `sdd init` creates `.ragignore` in project root with canonical ignore patterns.
- Running `sdd upgrade` on a project missing `.ragignore` adds it.
- `sdd check` reports a warning if `.ragignore` is absent from project root.
- `sdd check` reports a warning if frontmatter specifies an invalid `status` or nonexistent `superseded_by` target.
- `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, and `contract.md.tmpl` reflect `templates=1.7.0` with identical rule sets.
- `make check` and `sdd check --strict` pass with 0 errors and 0 warnings.
