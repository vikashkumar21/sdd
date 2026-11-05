# Plan — T-028 Work typology foundations: bugfixes, hotfixes, spikes, and task templates

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-05)

## 1. Approach summary
Establish first-class support for four work typologies (`features`, `bugfixes`, `hotfixes`, `spikes`) across templates, scaffolding engine, CLI, and mechanical checks. Embed specialized lifecycle templates (failing reproduction test for bugs, post-mortem for hotfixes, non-merging spike prototypes), add a 64KB context budget guard in `sdd check`, bump `templates.Version` to `1.2.0`, and upgrade `sdd status` and `sdd upgrade`.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-027 | Work typology and specialized lifecycle artifacts | Option A (features/, bugfixes/, hotfixes/, spikes/) | Option A | Approved (2026-11-05) |

## 3. Design
### Work Typologies & File Sets
1. **`features/T-xxx-<slug>/`**:
   - `requirements.md`, `plan.md`, `validation.md`, `review.md`
2. **`bugfixes/T-xxx-<slug>/`**:
   - `reproduction.md` (Step 0 minimal failing reproduction test case)
   - `rca.md` (Root cause analysis and blast-radius assessment)
   - `plan.md` (Minimal targeted fix)
   - `validation.md` (Proof test failed before fix and passes after fix; zero regression)
   - `review.md` (4 pillars, 🔴/🟡/🟢 risk audit, feedback capture)
3. **`hotfixes/T-xxx-<slug>/`**:
   - `incident.md` (Severity, production symptom, rollback)
   - `patch.md` (Minimal diff; no speculative refactoring)
   - `validation.md` (Verification against incident symptom)
   - `post_mortem.md` (Root cause, why tests missed it, follow-up preventative items)
4. **`spikes/T-xxx-<slug>/`**:
   - `hypotheses.md` (Questions to answer, timebox)
   - `findings.md` (Verified facts, recommendations, prototype reference)
   - `plan.md` (Investigation plan)
   - *Rule:* Prototype code in `spikes/T-xxx/spike/` is strictly non-merging; markdown is merged.

### CLI Dispatch
- `sdd task new --type <feature|bugfix|hotfix|spike> <T-xxx> <slug>`
- Convenience subcommands:
  - `sdd feature new <T-xxx> <slug>`
  - `sdd bugfix new <T-xxx> <slug>`
  - `sdd hotfix new <T-xxx> <slug>`
  - `sdd spike new <T-xxx> <slug>`

### Check Engine Extensions (`internal/check`)
- Scans `features/`, `bugfixes/`, `hotfixes/`, `spikes/`.
- **64KB Context Budget Guard:** Emits `WARN` if any task markdown file exceeds 65,536 bytes.
- Validates required files per typology.

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Create embedded templates for bugfixes, hotfixes, spikes, and bump `templates.Version` to `1.2.0`. | `internal/templates/data/bugfixes/`, `hotfixes/`, `spikes/`, `internal/templates/templates.go`, agent contract headers | `go test -v ./internal/templates` |
| 2 | Extend scaffolding engine to support all 4 work typologies. | `internal/feature/feature.go`, `internal/feature/feature_test.go` | `go test -v ./internal/feature` |
| 3 | Add CLI subcommands (`bugfix new`, `hotfix new`, `spike new`, `task new`). | `internal/cli/task.go`, `internal/cli/cli.go`, `internal/cli/cli_test.go` | `go test -v ./internal/cli -run TestTaskScaffolding` |
| 4 | Extend `sdd check` with 64KB context budget guard and multi-typology folder validation. | `internal/check/check.go`, `internal/check/check_test.go` | `go test -v ./internal/check` |
| 5 | Update `sdd upgrade` and `sdd status` for multi-typology awareness. | `internal/upgrade/upgrade.go`, `internal/status/status.go`, tests | `go test -v ./internal/upgrade ./internal/status` |
| 6 | Copy templates to project `bugfixes/_templates/`, `hotfixes/_templates/`, `spikes/_templates/`, update living docs, and verify Rule 14. | `README.md`, `ROADMAP.md`, `bugfixes/_templates/`, etc. | `make check`, `sdd check --strict`, Rule 14 temp kit check |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/templates/data/bugfixes/reproduction.md` | create | Bugfix reproduction template with failing Step 0 test |
| `internal/templates/data/bugfixes/rca.md` | create | Root cause analysis & blast radius template |
| `internal/templates/data/bugfixes/plan.md` | create | Bugfix plan template |
| `internal/templates/data/bugfixes/validation.md` | create | Bugfix validation template |
| `internal/templates/data/bugfixes/review.md` | create | Bugfix review template |
| `internal/templates/data/hotfixes/incident.md` | create | Hotfix incident template |
| `internal/templates/data/hotfixes/patch.md` | create | Hotfix patch template |
| `internal/templates/data/hotfixes/validation.md` | create | Hotfix validation template |
| `internal/templates/data/hotfixes/post_mortem.md` | create | Hotfix post-mortem template |
| `internal/templates/data/spikes/hypotheses.md` | create | Spike hypotheses template |
| `internal/templates/data/spikes/findings.md` | create | Spike findings template |
| `internal/templates/data/spikes/plan.md` | create | Spike plan template |
| `bugfixes/_templates/` | create | Project template overrides |
| `hotfixes/_templates/` | create | Project template overrides |
| `spikes/_templates/` | create | Project template overrides |
| `internal/templates/templates.go` | modify | Bump `templates.Version` to `1.2.0` |
| `CLAUDE.md`, `GEMINI.md`, `AGENTS.md` | modify | Update contract block version to `1.2.0` |
| `internal/feature/feature.go` | modify | Support typologies, directory routing, and file generation |
| `internal/feature/feature_test.go` | modify | Test scaffolding for all typologies |
| `internal/cli/task.go` | create | CLI handlers for `task new`, `bugfix new`, `hotfix new`, `spike new` |
| `internal/cli/cli.go` | modify | Wire new commands and update usage help |
| `internal/cli/cli_test.go` | modify | CLI tests for new typology commands |
| `internal/check/check.go` | modify | 64KB context budget check & multi-typology validation |
| `internal/check/check_test.go` | modify | Tests for 64KB guard and typology checking |
| `internal/upgrade/upgrade.go` | modify | Upgrade kit templates for new typologies |
| `internal/status/status.go` | modify | Scan `bugfixes/`, `hotfixes/`, `spikes/` |
| `README.md` | modify | Document new typology commands and templates |
| `ROADMAP.md` | modify | Update T-028 status |

## 6. Dependencies
Standard library only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Update T-028 status and notes |
| `README.md` | Add documentation for `sdd bugfix new`, `sdd hotfix new`, `sdd spike new`, `sdd task new` |
| `CLAUDE.md`, `GEMINI.md`, `AGENTS.md` | Bump contract header version to `1.2.0` |

## 8. Risks, mitigations, rollback
- **Risk:** Backward compatibility with existing repositories that only have `features/`.
  - **Mitigation:** Existing `features/` projects remain 100% valid; `sdd upgrade` adds new template folders without touching existing files.
- **Rollback:** Revert git working tree changes.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- None active in §8 register.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-05   Phrase received: `APPROVED T-028`
