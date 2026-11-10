# Plan — T-036 Binding Human Feedback Rule F-001 (Resolve all review findings)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-10)

## 1. Approach summary
Register the first binding human feedback rule `F-001` in `SKILLS.md` §8, codify it across all agent contracts (`contract.md.tmpl`, `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`) in Phase C.2 and the Definition of Done, bump `templates.Version` to `1.6.0` per Hard Rule 16, enhance `internal/check/review.go` to mechanically verify that all findings in `review.md` (Blockers, Warnings, and Minors) are resolved or explicitly justified before GATE 2 human review, and update canonical agent skills.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-032 | Human Feedback Rule F-001 (Resolve all review findings: Blockers, Warnings, and Minors) | Option A (Enact binding Rule F-001 in SKILLS.md §8 and contracts: worker agents must resolve all Blockers, Warnings, and Minors before GATE 2; any retained item must be explicitly justified) | Option A | Approved (2026-11-10) |
| D-026 | Code review specification, rubric, and task artifact (`review.md`) | Updated by D-032 to enforce resolution of all finding tiers | Option A | Approved (2026-11-05) |

## 3. Design

### Human Feedback Rule Register (`SKILLS.md` §8)
- Register `F-001`:
  - **ID:** `F-001`
  - **Date:** `2026-11-10`
  - **Rule:** Worker agents must resolve all findings (🔴 Blockers, 🟡 Warnings, and 🟢 Minors) reported during review before presenting to the human at GATE 2. Any retained finding not remediated must be explicitly justified in `review.md`.
  - **Applies to:** All tasks (Phase C.2 review triad, batch remediation, Phase D close, GATE 2).
  - **Source:** Human feedback during Milestone 9 review (Decision `D-032`).
  - **Verified by:** `sdd check` mechanical review validation, subagent review triad, GATE 2 checklist.
  - **Status:** `Active`.

### Mechanical Check Verification (`internal/check/review.go`)
- In `checkReviewDocument(rel string, taskStatus string)`:
  - Parse every table row in the findings register: `| ID | Category | Severity | Finding | Suggested Action | Status |`.
  - Extract `Severity` and `Status` columns.
  - Classify status:
    - Resolved: `resolved`, `justified`, `waived`, `accepted`.
    - Unresolved: `open`, `pending`, `""`, or template placeholder `<...>`.
  - Enforcement rules for tasks in `Awaiting review` or `Done`:
    - Unresolved 🔴 Blocker: Emit `Error` (blocks progression, exit 1).
    - Unresolved 🟡 Warning or 🟢 Minor: Emit `Warn` with message:
      `task is %s but review.md contains unresolved %s finding %q (Rule F-001 mandates resolving all findings or explicit justification)`
  - Under `sdd check --strict`, unresolved Warnings and Minors produce exit code 1, guaranteeing strict compliance.

### Contract and Templates Update (`templates.Version = 1.6.0`)
- Bump `templates.Version` to `1.6.0` in `internal/templates/templates.go`.
- Update `internal/templates/data/base/contract.md.tmpl`:
  - Phase C.2: Mandate resolving all review findings (Blockers, Warnings, Minors) in the batched remediation pass per Rule F-001.
  - Definition of Done (§8): Add checklist item:
    `- [ ] All review findings (Blockers, Warnings, and Minors) resolved or explicitly justified per Rule F-001`
  - Hard Rule 12: Note that active binding rule `F-001` is codified and enforced.
- Synchronize `AGENTS.md`, `CLAUDE.md`, and `GEMINI.md` in lockstep.
- Update `.sdd/manifest.json`.

### Prompt Synthesis (`internal/cli/prompt.go`)
- Ensure `extractFeedbackRules` reads active rules from `SKILLS.md` and surfaces `F-001` with clear resolution requirements.
- Add guidance in review prompt to verify that all severity tiers (Blockers, Warnings, Minors) are addressed.

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Register `F-001` in `SKILLS.md` §8 register table. | `SKILLS.md` | Inspection / git diff |
| 2 | Enhance `internal/check/review.go` to verify all findings rows (Blockers, Warnings, Minors) against F-001 resolution rules, and add test coverage. | `internal/check/review.go`, `internal/check/check_test.go` | `go test -v ./internal/check -run TestCheckReviewDocument` |
| 3 | Bump `templates.Version` to `1.6.0` and update template tests. | `internal/templates/templates.go`, `internal/templates/templates_test.go` | `go test -v ./internal/templates -run TestVersion` |
| 4 | Update `contract.md.tmpl`, synchronize `AGENTS.md`, `CLAUDE.md`, `GEMINI.md` in lockstep (`templates=1.6.0`), and update `.sdd/manifest.json`. | `internal/templates/data/base/contract.md.tmpl`, `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `.sdd/manifest.json` | `go test -v ./internal/check -run TestDetectsProblems/agent_contract_drift` |
| 5 | Update prompt synthesis and tests in `internal/cli/` to surface `F-001` and mandate resolving all finding tiers. | `internal/cli/prompt.go`, `internal/cli/cli_test.go` | `go test -v ./internal/cli -run TestPromptReview` |
| 6 | Update canonical skills (`skills/code-review/SKILL.md`, `skills/task-coordinator/SKILL.md`) and living docs (`README.md`, `TECH_STACK.md`, `ROADMAP.md`). | `skills/`, living docs | `sdd check --strict`, `make build` |
| 7 | Run verification suite (`make verify`, `make check`, Hard Rule 16 temp kit regeneration), invoke 3-subagent parallel review triad on T-036, and present GATE 2. | `tasks/features/T-036-resolve-review-findings/` | `make check`, `review.md` sign-off |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `SKILLS.md` | modify | Register `F-001` in §8 Human feedback rules table |
| `internal/check/review.go` | modify | Verify resolution of all findings (Blockers, Warnings, Minors) per F-001 |
| `internal/check/check_test.go` | modify | Test cases for unresolved vs resolved Warnings and Minors in review.md |
| `internal/templates/templates.go` | modify | Bump `templates.Version` to 1.6.0 |
| `internal/templates/templates_test.go` | modify | Assert `templates.Version` is 1.6.0 |
| `internal/templates/data/base/contract.md.tmpl` | modify | Add Rule F-001 to Phase C.2 and Definition of Done |
| `AGENTS.md` | modify | Synchronize contract to templates=1.6.0 |
| `CLAUDE.md` | modify | Synchronize contract to templates=1.6.0 |
| `GEMINI.md` | modify | Synchronize contract to templates=1.6.0 |
| `.sdd/manifest.json` | modify | Bump template version to 1.6.0 |
| `internal/cli/prompt.go` | modify | Surface F-001 resolution mandate in review prompt context |
| `internal/cli/cli_test.go` | modify | Test prompt generation reflects F-001 |
| `skills/code-review/SKILL.md` | modify | Document Rule F-001 resolution requirement across all tiers |
| `skills/task-coordinator/SKILL.md` | modify | Document Task Coordinator verification of Rule F-001 before merge |
| `README.md` | modify | Document Rule F-001 in review standards |
| `ROADMAP.md` | modify | Track T-036 progress |
| `TECH_STACK.md` | modify | Record verified fact for T-036 |
| `tasks/features/T-035-plan-review-triad/plan.md` | modify | Update status to Approved |
| `tasks/features/T-035-plan-review-triad/requirements.md` | modify | Update status to Approved |
| `tasks/features/T-036-resolve-review-findings/requirements.md` | modify | Requirements spec |
| `tasks/features/T-036-resolve-review-findings/plan.md` | modify | Implementation plan |
| `tasks/features/T-036-resolve-review-findings/validation.md` | modify | Validation matrix |
| `tasks/features/T-036-resolve-review-findings/review.md` | modify | Multi-lens review artifact |

## 6. Dependencies
Standard library only (`D-007`). Zero new dependencies.

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Status to `Implementing`, then `Awaiting review`, then `Done` |
| `TECH_STACK.md` | Record verified fact for T-036 in Section 7 |
| `SKILLS.md` | Register `F-001` in Section 8 |
| `README.md` | Update review standards overview |

## 8. Risks, mitigations, rollback
- **Risk:** Existing legacy tasks in completed milestones might have unresolved template placeholders in `review.md`.
  - **Mitigation:** In `internal/check/review.go`, only validate tasks that have a `review.md` file and check their actual finding rows. Tasks prior to M7 (`T-000` to `T-025`) do not have `review.md`. Tasks `T-026` onwards have clean, resolved findings.
- **Rollback:** `git restore` on modified files if needed.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- `F-001`: Active (being codified and enacted in this task). All findings in `review.md` for this task will be resolved before GATE 2.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-10   Phrase received: `APPROVED T-036`
