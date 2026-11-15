---
status: active
task_id: T-042
last_reviewed: 2026-11-15
---
# Plan — T-042 Cross-project and service dependency register in TECH_STACK.md §8

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved

## 1. Approach summary
Extend the `TECH_STACK.md.tmpl` template to partition Section 8 into package dependencies (§8.1) and cross-project/service dependencies (§8.2). Implement mechanical consistency checks in `internal/check` to validate decision references, directional descriptors, and status values without network probing. Bump `templates.Version` to `1.8.0` and adopt in this repository's `TECH_STACK.md`.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-038 | Cross-project and service dependency register in TECH_STACK.md §8 and sdd check validation | Option A (Partition §8 into §8.1 packages and §8.2 cross-project dependencies with 7 columns, mechanical sdd check validation for D-xxx links, direction and status enums, and template version bump to 1.8.0) / Option B (Unstructured free-form text comments in §8 without mechanical checking) | Option A: Preserves deterministic structure, provides AI agents and developers with explicit integration blast-radius awareness, enables mechanical validation without external network calls, and aligns with Contract Rule 16. | **Approved** 2026-11-15 |

## 3. Design

### Template Schema (`internal/templates/data/base/TECH_STACK.md.tmpl`)
```markdown
## 8. Dependency register

### 8.1 Package and library dependencies
{{if .Detected.Manifests}}
Dependency manifests *detected* (unverified): {{join .Detected.Manifests ", "}}. List the approved dependencies below as they are verified.
{{end}}
| Dependency | Purpose | Status |
|---|---|---|
| | (none recorded yet) | |

### 8.2 Cross-project and service dependencies
List external repositories, microservices, APIs, and sibling projects that this project depends on or that consume this project:

| Name | Repo / URL | Direction | Contract / Version | Owner | Decision | Status |
|---|---|---|---|---|---|---|
| | | | | | | |
```
*(Note: Base template uses generic placeholders without project-specific decision references like D-007 to ensure newly initialized kits pass `sdd check` cleanly. Project-specific D-007 entry is adopted solely in this repository's own `TECH_STACK.md`.)*

### Direction, Status & Decision Validations (`internal/check/check.go`)
- **Direction:** Must match case-insensitively one of: `Inbound`, `Outbound`, `Bidirectional`, `Inbound / Outbound`, `Producer`, `Consumer`, `None`, or empty/placeholder (`-`, `n/a`, `(none)`). Emits `techstack-dep-direction` warning if unrecognized.
- **Status:** Must match case-insensitively one of: `Approved`, `Proposed`, `Deprecated`, `Active`, `Draft`, `Retired`, or empty/placeholder (`-`, `n/a`, `(none)`). Emits `techstack-dep-status` warning if unrecognized.
- **Decision:** Must contain either a valid decision reference (`D-xxx`), placeholder (`-`, `n/a`, `(none)`), or empty. If non-empty text does not contain a recognizable decision reference pattern, emits `techstack-dep-decision`. If a `D-xxx` reference is present, resolution against §6 is verified by the existing `checkReferences` engine (which already generates line-precise `unresolved-decision` errors), avoiding duplicate diagnostic findings.

### Contract Bump (`internal/templates/templates.go`)
- `Version = "1.8.0"`
- Contract begin markers updated to `<!-- sdd:begin contract templates=1.8.0 -->` in `CLAUDE.md`, `GEMINI.md`, `AGENTS.md`.

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Update `TECH_STACK.md.tmpl` with §8.1 and §8.2, bump `templates.Version` to `1.8.0` in `internal/templates/templates.go`, update contract headers in `CLAUDE.md`, `GEMINI.md`, `AGENTS.md`, sync `.sdd/manifest.json`, and update `internal/templates/templates_test.go` | `internal/templates/data/base/TECH_STACK.md.tmpl`, `internal/templates/templates.go`, `internal/templates/templates_test.go`, `CLAUDE.md`, `GEMINI.md`, `AGENTS.md`, `.sdd/manifest.json` | `go test -v ./internal/templates` |
| 2 | Implement mechanical validation in `internal/check/check.go` for §8.2 (D-xxx decision resolution, direction and status enums) and add unit tests in `internal/check/check_test.go` | `internal/check/check.go`, `internal/check/check_test.go` | `go test -v ./internal/check -run TestCrossProjectDependencies` |
| 3 | Update `internal/scaffold/scaffold_test.go` and `internal/upgrade/upgrade_test.go` to assert new template rendering and safe upgrade preservation | `internal/scaffold/scaffold_test.go`, `internal/upgrade/upgrade_test.go` | `go test -v ./internal/scaffold && go test -v ./internal/upgrade -run TestUpgrade` |
| 4 | Adopt §8.1 and §8.2 in this repo's `TECH_STACK.md` (registering `sdd-agent`), update `README.md`, update `ROADMAP.md`, and run full verification suite | `TECH_STACK.md`, `README.md`, `ROADMAP.md`, `tasks/features/T-042-cross-project-dependencies/*` | `make verify && make check && ./bin/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `.sdd/manifest.json` | modify | Update templates version to 1.8.0 per Rule 16 |
| `AGENTS.md` | modify | Update contract version to 1.8.0 per Rule 16 |
| `CLAUDE.md` | modify | Update contract version to 1.8.0 per Rule 16 |
| `GEMINI.md` | modify | Update contract version to 1.8.0 per Rule 16 |
| `README.md` | modify | Document cross-project dependency register |
| `ROADMAP.md` | modify | Track milestone and task lifecycle status |
| `TECH_STACK.md` | modify | Adopt §8.1 and §8.2, register sdd-agent, record Decision D-038 |
| `internal/check/check.go` | modify | Implement checkTechStackDependencies validation rules |
| `internal/check/check_test.go` | modify | Unit tests for cross-project dependency validation |
| `internal/scaffold/scaffold_test.go` | modify | Test kit generation with updated template |
| `internal/templates/data/base/TECH_STACK.md.tmpl` | modify | Add §8.1 and §8.2 dependency tables |
| `internal/templates/templates.go` | modify | Bump Version to 1.8.0 |
| `internal/templates/templates_test.go` | modify | Verify Version 1.8.0 and template rendering |
| `internal/upgrade/upgrade_test.go` | modify | Test upgrade preservation of dependencies tables |
| `tasks/features/T-041-execution-ops-skills/plan.md` | modify | T-041 close sign-off and completion status |
| `tasks/features/T-041-execution-ops-skills/requirements.md` | modify | T-041 close completion status |
| `tasks/features/T-041-execution-ops-skills/validation.md` | modify | T-041 close sign-off and completed verification |
| `tasks/features/T-042-cross-project-dependencies/plan.md` | modify | Architecture plan and implementation steps |
| `tasks/features/T-042-cross-project-dependencies/requirements.md` | modify | Specify requirements R-001 to R-008 |
| `tasks/features/T-042-cross-project-dependencies/review.md` | modify | Review Triad evaluation and sign-off |
| `tasks/features/T-042-cross-project-dependencies/validation.md` | modify | Traceability matrix and verification evidence |

## 6. Dependencies
None. Pure Go standard library (`os`, `path/filepath`, `regexp`, `strings`) per Decision D-007.

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Update T-042 status across lifecycle (`Specifying` -> `Implementing` -> `Awaiting review` -> `Done`) |
| `TECH_STACK.md` | Adopt §8.1 and §8.2, register sdd-agent, record Decision D-038 |
| `README.md` | Mention cross-project dependency register in kit documentation |

## 8. Risks, mitigations, rollback
- **Risk:** Existing repositories failing `sdd check` if they lack Section 8.2.
  - *Mitigation:* `sdd check` treats Section 8.2 as optional for backward compatibility; it validates §8.2 rows only when the table is present.
- **Risk:** Network calls or external path checking causing flakiness or slowdowns.
  - *Mitigation:* Strictly zero external network access; validation is 100% local, mechanical, and deterministic.
- **Rollback:** Revert template and contract bump; restore single-table §8.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- **Rule F-001 (Resolve all review findings: Blockers, Warnings, and Minors):** Worker agent must resolve all findings reported across the 3 subagent review lenses before presenting to human at GATE 2.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-15   Phrase received: `APPROVED T-042`
