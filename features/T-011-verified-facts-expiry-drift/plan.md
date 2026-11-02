# Plan — T-011 P-08: Verified-facts expiry and roadmap drift checker

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-02)
Roadmap ref: `ROADMAP.md` T-011

## 1. Approach summary
1. Extend `internal/roadmap/roadmap.go` to parse the `Depends on` cell into `Task.DependsOn []string`.
2. Add prerequisite dependency drift validation to `checkRoadmap()` in `internal/check/check.go` to detect tasks started or completed ahead of prerequisites.
3. Add `checkVerifiedFactsExpiry()` in `internal/check/check.go` to parse the Verified facts table in `TECH_STACK.md` and warn if `Re-verify at` references a completed task or fully completed milestone.
4. Add feature document status drift validation in `checkGateDiscipline()` to detect discrepancies between roadmap `Done` state and feature `plan.md` / `validation.md` status lines.
5. Add unit test suites in `internal/roadmap/roadmap_test.go` and `internal/check/check_test.go`.
6. Update living docs: `TECH_STACK.md` verified facts log and `ROADMAP.md` task state.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| (none) | No architectural decisions needed | n/a | Standard library implementation | n/a |

## 3. Design
- **Roadmap Prerequisite Extraction:**
  - In `roadmap.Parse`: when task rows have >= 4 cells, parse `cells[3]` with `taskRefRe` into `task.DependsOn`.
- **Dependency Drift Check:**
  - If a task has status `implementing`, `awaiting review`, or `done`:
    For each prerequisite ID in `task.DependsOn`, verify prerequisite exists and its status is `done` (or at least `implementing` if mutually dependent). If prerequisite is `not started`, `specifying`, or `blocked`, emit warning.
- **Verified Facts Expiry Check:**
  - Locate `## <n>. Verified facts log` table in `TECH_STACK.md`.
  - For each row, extract `Fact` and `Re-verify at`.
  - If `Re-verify at` matches task IDs (e.g. `T-002`) and that task is `Done`, emit warning:
    `verified fact %q is past due for re-verification (%s is Done)`
  - If `Re-verify at` matches a milestone (e.g. `M1`) and all tasks in that milestone are `Done`, emit warning:
    `verified fact %q is past due for re-verification (milestone %s is completed)`
- **Feature Document Status Drift:**
  - If task is `done` in `ROADMAP.md`:
    - Check `validation.md`: warn if status header is `Not run` or `In progress`.
    - Check `plan.md`: warn if status header is `Draft` or `Awaiting approval`.

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Add DependsOn parsing to roadmap.Task | `internal/roadmap/roadmap.go` | `go test ./internal/roadmap -run TestParseDependsOn` |
| 2 | Add dependency prerequisite drift checker | `internal/check/check.go` | `go test ./internal/check -run TestCheckDependencyDrift` |
| 3 | Add verified facts expiry checker | `internal/check/check.go` | `go test ./internal/check -run TestCheckVerifiedFactsExpiry` |
| 4 | Add feature document status drift checker | `internal/check/check.go` | `go test ./internal/check -run TestCheckFeatureStatusDrift` |
| 5 | Verify full check suite and refresh TECH_STACK facts | `TECH_STACK.md`, `ROADMAP.md` | `make check && go run ./cmd/sdd check` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/roadmap/roadmap.go` | modify | Extract prerequisite task IDs into Task.DependsOn |
| `internal/roadmap/roadmap_test.go` | modify | Tests for prerequisite parsing |
| `internal/check/check.go` | modify | Implement facts expiry, dependency drift, and doc drift checks |
| `internal/check/check_test.go` | modify | Tests for verified facts expiry and roadmap drift rules |
| `TECH_STACK.md` | modify | Align verified facts re-verify milestones with future targets |
| `ROADMAP.md` | modify | Update T-011 status across workflow gates |

## 6. Dependencies
None. Standard library only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Update T-011 status |
| `TECH_STACK.md` | Align verified-facts re-verify milestones with future tasks; add verified fact |

## 8. Risks, mitigations, rollback
- **Risk:** Existing verified facts tables in repositories might produce unexpected warnings if milestones are outdated.
  - *Mitigation:* Warnings are informational reminders (severity `Warn`); they do not break default non-strict `sdd check`.
- **Rollback:** `git restore internal/roadmap/ internal/check/` reverts all drift and expiry checks.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active.

## 10. Approval record
Approved by: Vikash   Date: 2026-11-02   Phrase received: `APPROVED T-011`
