# Plan — T-037 Document lifecycle YAML frontmatter metadata and .ragignore generation

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-10)

## 1. Approach summary
Implement pure Go document lifecycle YAML frontmatter parsing and validation in `internal/doc`, generate project root `.ragignore` files during `sdd init` and `sdd upgrade` from embedded template `base/.ragignore.tmpl`, enhance `internal/check` to mechanically check `.ragignore` existence (`ragignore-missing`) and validate frontmatter integrity (`lifecycle-status`, `lifecycle-date`, `lifecycle-superseded`) across Markdown files, add lifecycle frontmatter to base and task templates, bump `templates.Version` to `1.7.0` per Hard Rule 16, synchronize agent contracts (`AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `contract.md.tmpl`), and create `.ragignore` in this repository.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-033 | Lifecycle YAML frontmatter metadata and `.ragignore` for RAG filtering | Option A (Standard YAML frontmatter: status, task_id, superseded_by, last_reviewed, and .ragignore generation) | Option A | Approved (2026-11-10) |
| D-017 | Manifest schema 1 and proposal format | Updated to track `.ragignore` in manifest | Option A | Approved (2026-11-02) |

## 3. Design

### Lifecycle Frontmatter Schema & Parser (`internal/doc/lifecycle.go`)
- Schema:
  ```yaml
  ---
  status: active # active | draft | completed | superseded | deprecated
  task_id: T-037 # optional
  last_reviewed: 2026-11-10
  superseded_by: docs/architecture/v2-storage.md # required when status is superseded
  ---
  ```
- Go Types:
  ```go
  type LifecycleMetadata struct {
      Status       string `json:"status"`
      TaskID       string `json:"task_id,omitempty"`
      LastReviewed string `json:"last_reviewed,omitempty"`
      SupersededBy string `json:"superseded_by,omitempty"`
  }
  ```
- Pure Go Parser & Validator:
  - `ParseFrontmatter(data []byte) (meta LifecycleMetadata, hasFrontmatter bool, isLifecycle bool, body []byte, err error)`
  - Distinguishes lifecycle docs from other frontmatter (e.g. `skills/*/SKILL.md` with `name` and `description`): if frontmatter contains none of `status`, `task_id`, `superseded_by`, or `last_reviewed`, `isLifecycle` evaluates to `false` and validation is bypassed.
  - `ValidateLifecycle(meta LifecycleMetadata, docPath string, fileExists func(string) bool) []ValidationError`
  - Validates:
    - Status enum: `active`, `draft`, `completed`, `superseded`, `deprecated` (case-insensitive). Rule: `lifecycle-status`.
    - Date format: `YYYY-MM-DD` (regex `^\d{4}-\d{2}-\d{2}$`). Optional (`omitempty`), but if non-empty, must match. Rule: `lifecycle-date`.
    - If `status == "superseded"`: `superseded_by` must not be empty, and the referenced file must exist.
      Path resolution order: checks relative to repository `root` first (canonical convention); if not found, checks relative to `filepath.Dir(docPath)`. Rule: `lifecycle-superseded`.

### `.ragignore` Generation (`internal/scaffold/plan.go`, `internal/upgrade/upgrade.go`)
- Embedded Template: `internal/templates/data/base/.ragignore.tmpl`
  ```
  # SDD .ragignore - Exclude ephemeral files and build artifacts from RAG/Vector indexing
  .sdd/
  dist/
  bin/
  node_modules/
  vendor/
  *.tmp
  *.bak
  ```
- `scaffold.BuildPlan`: Adds `.ragignore` to base kit generation during `sdd init`.
- `upgrade.Run`: Detects if `.ragignore` is absent in target project and plans creation.

### Mechanical Verification (`internal/check/check.go`)
- In `internal/check/check.go`:
  - Rule `ragignore-missing`: Verify `.ragignore` exists in project root (warns if missing: `missing root .ragignore file; run sdd upgrade or create one for RAG filtering`).
  - Scan all Markdown files in project:
    - If lifecycle frontmatter is present: validate with `doc.ValidateLifecycle`.
    - Emit Warn findings for `lifecycle-status`, `lifecycle-date`, or `lifecycle-superseded`.

### Template & Contract Version Bump
- Add lifecycle frontmatter to:
  - Base templates: `base/MISSION.md.tmpl` (`status: active`), `base/TECH_STACK.md.tmpl` (`status: active`), `base/ROADMAP.md.tmpl` (`status: active`), `base/SKILLS.md.tmpl` (`status: active`).
  - Task templates: `tasks/features/requirements.md` (`status: draft`), `tasks/features/plan.md` (`status: draft`), `tasks/features/validation.md` (`status: draft`), `tasks/features/review.md` (`status: draft`).
  - Same for `tasks/bugfixes/`, `tasks/hotfixes/`, `tasks/spikes/`.
- Bump `templates.Version` to `1.7.0` in `internal/templates/templates.go`.
- Synchronize `internal/templates/data/base/contract.md.tmpl`, `AGENTS.md`, `CLAUDE.md`, and `GEMINI.md` in lockstep.
- Update `.sdd/manifest.json`.

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement YAML frontmatter parser and lifecycle validator in `internal/doc/lifecycle.go` with table-driven tests. | `internal/doc/lifecycle.go`, `internal/doc/lifecycle_test.go` | `go test -v ./internal/doc -run TestLifecycle` |
| 2 | Create `internal/templates/data/base/.ragignore.tmpl`, wire `.ragignore` into `scaffold.BuildPlan` and `upgrade.Run`, and update tests. | `internal/templates/data/base/.ragignore.tmpl`, `internal/scaffold/plan.go`, `internal/scaffold/scaffold_test.go`, `internal/upgrade/upgrade.go`, `internal/upgrade/upgrade_test.go` | `go test -v ./internal/scaffold ./internal/upgrade` |
| 3 | Enhance `internal/check/` to verify `.ragignore` existence (`ragignore-missing`) and validate lifecycle frontmatter (`lifecycle-status`, `lifecycle-date`, `lifecycle-superseded`). | `internal/check/check.go`, `internal/check/check_test.go` | `go test -v ./internal/check -run TestRAGIgnoreAndLifecycle` |
| 4 | Add lifecycle frontmatter to base templates and task templates in `internal/templates/data/`, and update feature test assertions for frontmatter. | `internal/templates/data/base/`, `internal/templates/data/tasks/`, `internal/feature/feature_test.go` | `go test -v ./internal/templates -run TestTemplates`, `go test -v ./internal/feature` |
| 5 | Bump `templates.Version` to `1.7.0` in `internal/templates/templates.go` and update template assertions. | `internal/templates/templates.go`, `internal/templates/templates_test.go` | `go test -v ./internal/templates -run TestVersion` |
| 6 | Synchronize agent contracts (`contract.md.tmpl`, `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`), create root `.ragignore`, and update `.sdd/manifest.json`. | `contract.md.tmpl`, `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `.ragignore`, `.sdd/manifest.json` | `go test -v ./internal/check -run TestDetectsProblems/agent_contract_drift` |
| 7 | Update living docs (`README.md`, `TECH_STACK.md`, `ROADMAP.md`). | `README.md`, `TECH_STACK.md`, `ROADMAP.md` | `sdd check --strict`, `make build` |
| 8 | Run verification suite (`make verify`, `make check`, Hard Rule 16 fresh kit regeneration), invoke 3-subagent parallel review triad on T-037, and present GATE 2. | `tasks/features/T-037-lifecycle-metadata-ragignore/` | `make check`, `review.md` sign-off |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/doc/lifecycle.go` | create | Pure Go YAML frontmatter parser and lifecycle metadata validator |
| `internal/doc/lifecycle_test.go` | create | Tests for lifecycle frontmatter parsing and validation |
| `internal/templates/data/base/.ragignore.tmpl` | create | Embedded base template for .ragignore |
| `internal/templates/data/base/MISSION.md.tmpl` | modify | Add lifecycle frontmatter (`status: active`) |
| `internal/templates/data/base/TECH_STACK.md.tmpl` | modify | Add lifecycle frontmatter (`status: active`) |
| `internal/templates/data/base/ROADMAP.md.tmpl` | modify | Add lifecycle frontmatter (`status: active`) |
| `internal/templates/data/base/SKILLS.md.tmpl` | modify | Add lifecycle frontmatter (`status: active`) |
| `internal/templates/data/tasks/features/requirements.md` | modify | Add lifecycle frontmatter (`status: draft`) |
| `internal/templates/data/tasks/features/plan.md` | modify | Add lifecycle frontmatter (`status: draft`) |
| `internal/templates/data/tasks/features/validation.md` | modify | Add lifecycle frontmatter (`status: draft`) |
| `internal/templates/data/tasks/features/review.md` | modify | Add lifecycle frontmatter (`status: draft`) |
| `internal/templates/data/tasks/bugfixes/plan.md` | modify | Add lifecycle frontmatter (`status: draft`) |
| `internal/templates/data/tasks/bugfixes/rca.md` | modify | Add lifecycle frontmatter (`status: draft`) |
| `internal/templates/data/tasks/bugfixes/reproduction.md` | modify | Add lifecycle frontmatter (`status: draft`) |
| `internal/templates/data/tasks/bugfixes/review.md` | modify | Add lifecycle frontmatter (`status: draft`) |
| `internal/templates/data/tasks/bugfixes/validation.md` | modify | Add lifecycle frontmatter (`status: draft`) |
| `internal/templates/data/tasks/hotfixes/incident.md` | modify | Add lifecycle frontmatter (`status: draft`) |
| `internal/templates/data/tasks/hotfixes/patch.md` | modify | Add lifecycle frontmatter (`status: draft`) |
| `internal/templates/data/tasks/hotfixes/post_mortem.md` | modify | Add lifecycle frontmatter (`status: draft`) |
| `internal/templates/data/tasks/hotfixes/validation.md` | modify | Add lifecycle frontmatter (`status: draft`) |
| `internal/templates/data/tasks/spikes/findings.md` | modify | Add lifecycle frontmatter (`status: draft`) |
| `internal/templates/data/tasks/spikes/hypotheses.md` | modify | Add lifecycle frontmatter (`status: draft`) |
| `internal/templates/data/tasks/spikes/plan.md` | modify | Add lifecycle frontmatter (`status: draft`) |
| `internal/feature/feature_test.go` | modify | Update title assertion in feature test to handle frontmatter preceding markdown heading |
| `internal/scaffold/plan.go` | modify | Plan .ragignore during sdd init |
| `internal/scaffold/scaffold_test.go` | modify | Test .ragignore included in init plan |
| `internal/upgrade/upgrade.go` | modify | Add missing .ragignore during sdd upgrade |
| `internal/upgrade/upgrade_test.go` | modify | Test upgrade generates .ragignore when absent |
| `internal/check/check.go` | modify | Check .ragignore existence and validate markdown frontmatter |
| `internal/check/check_test.go` | modify | Tests for .ragignore check and lifecycle validation |
| `internal/templates/templates.go` | modify | Bump `templates.Version` to 1.7.0 |
| `internal/templates/templates_test.go` | modify | Assert template version is 1.7.0 |
| `internal/templates/data/base/contract.md.tmpl` | modify | Update contract template for 1.7.0 |
| `AGENTS.md` | modify | Synchronize contract to templates=1.7.0 |
| `CLAUDE.md` | modify | Synchronize contract to templates=1.7.0 |
| `GEMINI.md` | modify | Synchronize contract to templates=1.7.0 |
| `.ragignore` | create | Project root RAG ignore file |
| `.sdd/manifest.json` | modify | Update template version to 1.7.0 and add .ragignore |
| `README.md` | modify | Document .ragignore and lifecycle metadata |
| `ROADMAP.md` | modify | Track T-037 progress |
| `TECH_STACK.md` | modify | Record verified fact for T-037 |
| `tasks/features/T-037-lifecycle-metadata-ragignore/requirements.md` | modify | Requirements spec |
| `tasks/features/T-037-lifecycle-metadata-ragignore/plan.md` | modify | Implementation plan |
| `tasks/features/T-037-lifecycle-metadata-ragignore/validation.md` | modify | Validation matrix |
| `tasks/features/T-037-lifecycle-metadata-ragignore/review.md` | modify | Multi-lens review artifact |

## 6. Dependencies
Standard library only (`D-007`). Zero new dependencies.

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Status to `Implementing`, then `Awaiting review`, then `Done` |
| `TECH_STACK.md` | Record verified fact for T-037 in Section 7 |
| `README.md` | Document .ragignore and lifecycle frontmatter filtering |

## 8. Risks, mitigations, rollback
- **Risk:** Existing Markdown documents without frontmatter might produce false positives in `sdd check`.
  - **Mitigation:** Frontmatter is strictly optional on arbitrary docs. `internal/doc` only validates lifecycle metadata if the file begins with the `---` YAML delimiter and contains at least one lifecycle field (`status`, `task_id`, `superseded_by`, `last_reviewed`).
- **Rollback:** `git restore` on planned files if needed.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- `F-001`: Active (binding). All review findings (Blockers, Warnings, Minors) will be resolved before GATE 2 close.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-10   Phrase received: `APPROVED T-037`
