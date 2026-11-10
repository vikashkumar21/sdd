# Plan — T-026 Official code-review skill and task review template (review.md)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-05)
Roadmap ref: `ROADMAP.md` T-026

## 1. Approach summary
1. **Templates & Versioning (Rule 14):**
   - Create embedded template `internal/templates/data/features/review.md`.
   - Copy to project repository `features/_templates/review.md`.
   - Bump `templates.Version` in `internal/templates/version.go` from `1.0.0` to `1.1.0`.
   - Mirror version bump in agent contracts (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`) managed block header: `templates=1.1.0`.
2. **Feature Scaffolding (`internal/feature`):**
   - Update `internal/feature/feature.go` to inspect available templates in `features/_templates/`. If `review.md` is present, scaffold it alongside `requirements.md`, `plan.md`, `validation.md`.
   - If absent (legacy kit), continue scaffolding the original three files without error.
3. **Canonical Agent Skill (`internal/skill`):**
   - Define `CanonicalCodeReviewSkill` in `internal/skill/export.go` conforming to `agentskills.io` standard.
   - Enhance `sdd skill export` to support `--skill <name>` (defaulting to `sdd-protocol`, supporting `code-review` and `all`).
4. **Mechanical Checks (`internal/check`):**
   - Add `checkReviewDocument` in `internal/check/`:
     - Checks that `review.md` status is one of `Pending`, `Changes requested`, or `Approved`.
     - When a task in `ROADMAP.md` has status `Awaiting review` or `Done`, verifies there are zero unresolved 🔴 Blocker rows in `review.md`.
5. **Verification & Living Docs:**
   - Unit tests across `internal/feature`, `internal/skill`, `internal/check`, `internal/cli`.
   - Update `README.md`, `ROADMAP.md`, `SKILLS.md`.
   - Rule 14 verification: regenerate kit in a temporary directory and verify `sdd check` passes.

## 2. Decisions (new or affected)
- `D-026` (Approved 2026-11-05): Code review specification, rubric, and task artifact (`review.md`).

## 3. Design
- **Review Document Format (`features/_templates/review.md`):**
  - Status line: `Status: Pending | Changes requested | Approved`
  - Reviewer: `<Agent Name / Human>`
  - Scope & Spec Alignment checklist
  - Architectural & Decision Conformance checklist
  - 3-Tier Risk & Findings Register table:
    - 🔴 **Blocker (Developer Attention Needed)**
    - 🟡 **Warning (Attention Suggested)**
    - 🟢 **Minor (Safe / Informational)**
  - Reusable Knowledge & Feedback Synthesis checklist

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Create `review.md` template and bump `templates.Version` to 1.1.0 | `internal/templates/data/features/review.md`, `features/_templates/review.md`, `internal/templates/version.go`, `CLAUDE.md`, `GEMINI.md`, `AGENTS.md` | `go test -v ./internal/templates` |
| 2 | Update `internal/feature` and `internal/upgrade` to support `review.md` | `internal/feature/feature.go`, `internal/feature/feature_test.go`, `internal/upgrade/upgrade.go`, `internal/upgrade/upgrade_test.go` | `go test -v ./internal/feature ./internal/upgrade` |
| 3 | Add canonical `code-review` skill and update `sdd skill export` | `internal/skill/export.go`, `internal/skill/export_test.go`, `internal/cli/skill.go`, `internal/cli/cli_test.go` | `go test -v ./internal/skill ./internal/cli -run TestSkillExport` |
| 4 | Add `sdd check` rules for `review.md` status and blockers | `internal/check/review.go`, `internal/check/check.go`, `internal/check/check_test.go` | `go test -v ./internal/check -run TestCheckReview` |
| 5 | Verify Rule 14 and update living docs | `README.md`, `ROADMAP.md`, `SKILLS.md` | `make check && go run ./cmd/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/templates/data/features/review.md` | create | Embedded template for code review artifact |
| `features/_templates/review.md` | create | Project template for task code review artifact |
| `internal/templates/version.go` | modify | Bump `templates.Version` to 1.1.0 (Rule 14) |
| `CLAUDE.md`, `GEMINI.md`, `AGENTS.md` | modify | Update template version in contract header to 1.1.0 |
| `internal/feature/feature.go` | modify | Scaffold `review.md` when template is present |
| `internal/feature/feature_test.go` | modify | Unit tests for 4-file task scaffolding |
| `internal/upgrade/upgrade.go` | modify | Include `review.md` in task templates upgrade loop |
| `internal/upgrade/upgrade_test.go` | modify | Test upgrading older kit creates `features/_templates/review.md` |
| `internal/skill/export.go` | modify | Add `CanonicalCodeReviewSkill` and skill target resolution |
| `internal/skill/export_test.go` | modify | Test exporting `code-review` skill |
| `internal/cli/skill.go` | modify | Support `--skill` flag for `sdd skill export` |
| `internal/cli/cli_test.go` | modify | CLI tests for skill export |
| `internal/check/review.go` | create | Consistency check for task `review.md` |
| `internal/check/check.go` | modify | Invoke review check in `Run` |
| `internal/check/check_test.go` | modify | Unit tests for review findings and status verification |
| `README.md` | modify | Document `review.md` and `code-review` skill |
| `ROADMAP.md` | modify | Track T-026 status across gates |

## 6. Dependencies
None. Standard library only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `README.md` | Document `review.md` artifact and `sdd skill export --skill code-review` |
| `ROADMAP.md` | Update T-026 status across gates |
| `SKILLS.md` | Add Code Review Standard to section 9 |

## 8. Risks, mitigations, rollback
- **Risk:** Existing projects lacking `features/_templates/review.md` fail on `sdd feature new`.
  - *Mitigation:* `internal/feature` checks whether `review.md` exists before attempting to read and render it; continues cleanly with 3 documents if absent.
- **Rollback:** `git restore .` reverts all changes.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active.

## 10. Approval record
Approved by: Vikash   Date: 2026-11-05   Phrase received: APPROVED T-026
