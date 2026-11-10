# Plan — T-035 Pre-GATE 1 Plan Review Subagent and 3-Subagent Parallel Review Triad

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-10)

## 1. Approach summary
Implement the `sdd prompt plan-review <T-xxx>` CLI command (with `--json` export) for pre-GATE 1 plan review handover prompt generation, enhance `sdd prompt review <T-xxx> --lens <lens>` for specialized single-lens parallel reviewer dispatch, synchronize agent contracts (`AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `contract.md.tmpl`) to codify both the Pre-GATE 1 Plan Review Subagent in Phase A and the 3-Subagent Parallel Review Triad in Phase C.2, bump `templates.Version` to `1.5.0` per Hard Rule 16, and update canonical agent skills.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-031 | Pre-GATE 1 Plan Review Subagent and 3-Subagent Parallel Review Triad | Option A (Plan reviewer in Phase A + 3 parallel subagent reviewers in Phase C.2) vs Option B (Single subagent doing all passes) | Option A | Approved (2026-11-10) |
| D-026 | Code review specification, rubric, and task artifact (`review.md`) | Updated by D-031 to mandate 3 parallel subagents | Option A | Approved (2026-11-05) |
| D-029 | Two-tier git governance and Task Coordinator orchestration | Updated by D-031 to orchestrate plan review and parallel triad | Option A | Approved (2026-11-05) |

## 3. Design

### Pre-GATE 1 Plan Review Subagent Protocol (`internal/cli/prompt.go`)
- Command: `sdd prompt plan-review <T-xxx> [--json]`
- Reads task artifacts (`requirements.md`, `plan.md`, `validation.md`) from `tasks/<typology>/T-xxx-*`.
- Reads `TECH_STACK.md` decisions and `SKILLS.md` active feedback rules.
- Generates structured handover prompt:
  - Role: `Independent Plan Reviewer`
  - Rubric:
    1. Spec completeness: Atomic `R-xxx` requirements, clear acceptance criteria, zero vague adjectives.
    2. Plan soundness: Steps small enough to review in one sitting; each step names files and proving check.
    3. Boundary precision: Section 5 files list is exhaustive; zero extraneous files or unapproved dependencies.
    4. Risk & rollback: Realistic failure modes and clean recovery paths.
    5. Testability: Traceability matrix maps each requirement to a concrete test or command.
  - JSON output (`--json`): structured object containing `task_id`, `typology`, `folder`, `requirements`, `plan`, `validation`, and `rubric` for SDK dispatch.

### 3-Subagent Parallel Review Triad (`internal/cli/prompt.go`, `skills/code-review`)
- Single-lens prompt generation via `sdd prompt review <T-xxx> --lens <spec|security|pattern> [--json]`:
  - `spec`: Focused exclusively on `Spec Auditor` role (requirements traceability, negative tests, plan boundary adherence, zero dependencies).
  - `security`: Focused exclusively on `AppSec Specialist` role (threat surface, bounds, `os.Root` confinement, secrets hygiene, command safety).
  - `pattern`: Focused exclusively on `Pattern & Quality Auditor` role (decision compliance, determinism, Go idioms, active feedback rules).
- Review aggregation:
  - All three parallel subagents report findings into their respective lens sections of `review.md`.
  - Task status advances to `Awaiting review` only when all 3 lenses have 0 Blockers.

### Batch Remediation & Conflict Resolution Protocol
When multiple subagents report changes during review:
1. **Single Remediator Invariant:** Review subagents are strictly read-only auditors; only the primary worker agent executes remediation code changes.
2. **Batched Remediation:** The primary agent waits for all 3 lens reports, collects all Blockers and Warnings, and addresses them in a single unified fix commit. This consumes exactly 1 iteration of the 3-loop circuit breaker.
3. **Conflict Resolution Hierarchy:**
   - AppSec & safety invariants take precedence over feature convenience.
   - Determinism and standard library purity (`D-007`) take precedence over pattern shortcuts.
   - If two subagents issue fundamentally conflicting requirements that cannot be safely unified, the agent MUST STOP and ask the human via the Decision protocol (`DECISION NEEDED: D-xxx`). Never guess or silently pick a winner.
4. **Regression Sweep & Re-Review:** After applying the batched fix, the primary agent runs the full test suite (`make verify`, `make check`), then triggers re-review for all affected lenses.

### Contract & Template Version Bump
- Bump `templates.Version` to `1.5.0` in `internal/templates/templates.go`.
- Update `internal/templates/data/base/contract.md.tmpl` and agent files (`AGENTS.md`, `CLAUDE.md`, `GEMINI.md`):
  - Phase A table row and GATE 1 section: add mandatory Pre-GATE 1 Plan Review Subagent check.
  - Phase C.2 table row and Phase C.2 section: mandate 3-Subagent Parallel Review Triad.
  - Enforce 3-loop circuit breaker across both plan review and code review loops.

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement `sdd prompt plan-review <T-xxx>` CLI command with `--json` support and test coverage. | `internal/cli/prompt.go`, `internal/cli/cli.go`, `internal/cli/cli_test.go` | `go test -v ./internal/cli -run TestPromptPlanReview` |
| 2 | Enhance single-lens handover prompts in `sdd prompt review <T-xxx> --lens <lens>` with dedicated rubric specialization and test coverage. | `internal/cli/prompt.go`, `internal/cli/cli_test.go` | `go test -v ./internal/cli -run TestPromptReviewLens` |
| 3 | Enhance mechanical review validation in `internal/check/review.go` and add test coverage in `internal/check/check_test.go`. | `internal/check/review.go`, `internal/check/check_test.go` | `go test -v ./internal/check -run TestCheckReviewDocument` |
| 4 | Bump `templates.Version` to `1.5.0` in `internal/templates/templates.go` and update template tests. | `internal/templates/templates.go`, `internal/templates/templates_test.go` | `go test -v ./internal/templates -run TestVersion` |
| 5 | Synchronize agent contracts (`AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `contract.md.tmpl`) to codify Pre-GATE 1 Plan Review and 3-Subagent Parallel Review Triad. | `internal/templates/data/base/contract.md.tmpl`, `AGENTS.md`, `CLAUDE.md`, `GEMINI.md` | `go test -v ./internal/check -run TestDetectsProblems/agent_contract_drift` |
| 6 | Update canonical skills (`skills/code-review/SKILL.md`, `skills/task-coordinator/SKILL.md`) and living docs (`README.md`, `SKILLS.md`, `TECH_STACK.md`, `ROADMAP.md`). | `skills/`, living docs | `sdd check --strict`, `make build` |
| 7 | Run verification suite (`make verify`, `make check`), invoke 3-subagent parallel review triad on T-035, and present GATE 2. | `tasks/features/T-035-plan-review-triad/` | `make check`, `review.md` sign-off |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/cli/prompt.go` | modify | Implement `prompt plan-review` and refine single-lens `prompt review` |
| `internal/cli/commands.go` | modify | Route `prompt plan-review` subcommand dispatch |
| `internal/cli/cli.go` | modify | Register `prompt plan-review` subcommand and help text |
| `internal/cli/cli_test.go` | modify | Tests for `prompt plan-review` (markdown, json, lens flags) |
| `internal/check/review.go` | modify | Enhance multi-lens review structure verification |
| `internal/check/check_test.go` | modify | Tests for multi-lens review validation |
| `internal/templates/templates.go` | modify | Bump `templates.Version` to 1.5.0 |
| `internal/templates/templates_test.go` | modify | Update version assertion to 1.5.0 |
| `internal/templates/data/base/contract.md.tmpl` | modify | Contract template update with plan-review and review triad |
| `.sdd/manifest.json` | modify | Bump template version to 1.5.0 |
| `AGENTS.md` | modify | Synchronize contract with templates=1.5.0 |
| `CLAUDE.md` | modify | Synchronize contract with templates=1.5.0 |
| `GEMINI.md` | modify | Synchronize contract with templates=1.5.0 |
| `skills/code-review/SKILL.md` | modify | Document 3-subagent parallel review triad |
| `skills/task-coordinator/SKILL.md` | modify | Document parallel triad orchestration and pre-gate 1 plan review |
| `README.md` | modify | Document `sdd prompt plan-review` and parallel review triad |
| `ROADMAP.md` | modify | Update T-035 status |
| `TECH_STACK.md` | modify | Record verified fact for T-035 |
| `tasks/features/T-035-plan-review-triad/requirements.md` | modify | Requirements spec |
| `tasks/features/T-035-plan-review-triad/plan.md` | modify | Implementation plan |
| `tasks/features/T-035-plan-review-triad/validation.md` | modify | Validation matrix |
| `tasks/features/T-035-plan-review-triad/review.md` | modify | Multi-lens review artifact |

## 6. Living docs to update
- `ROADMAP.md`: Status to `Implementing`, then `Awaiting review`, then `Done`.
- `TECH_STACK.md`: Section 7 verified facts log entry for T-035.
- `SKILLS.md`: Note any new review learnings or patterns.
- `README.md`: Document `sdd prompt plan-review` and parallel review triad.
- Agent contracts: `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `contract.md.tmpl`.

## 7. Risks and rollback
- **Risk:** Prompt length for plan-review could exceed token context if task files are huge.
  - **Mitigation:** Rely on SDD 64KB context budget guard; `prompt plan-review` truncates or summarizes non-essential sections if necessary, though typical task files are 2-5KB.
- **Risk:** Subagent parallel execution concurrency limits or rate limiting when dispatching 3 reviewers concurrently.
  - **Mitigation:** The review triad protocol allows parallel or sequential execution across the three lenses without compromising review independence; each lens records findings in its designated section of `review.md`.
- **Rollback:** `git revert` cleanly restores previous CLI dispatch and contract text.
