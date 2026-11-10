# Plan — T-032 Autonomous multi-agent hierarchy and Task Coordinator skill

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-05)

## 1. Approach summary
Update the SDD agent contracts (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`) and their embedded templates to codify two-tier agent governance (Task Coordinator vs Worker Agents) and mandatory Phase C.2 Independent Subagent Audit. Create and embed the canonical `skills/task-coordinator/SKILL.md` skill, implement `sdd prompt review <T-xxx>` to generate turnkey subagent review context bundles across coding harnesses, bump template version to `1.3.0`, and enforce independent review sign-off in `sdd check`.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-029 | Two-tier git governance and Task Coordinator orchestration | Option A | Option A | Approved (2026-11-05) |

## 3. Design

### Agent Contract Updates
- **Section 2 ("The workflow"):**
  - Split Phase C into:
    - **C.1 Verify:** Primary agent executes checks in `validation.md` and records real commands/results.
    - **C.2 Independent Audit:** Primary agent invokes an independent subagent reviewer (via Antigravity `invoke_subagent`, Claude Code `Task` tool, or `sdd prompt review`). Subagent receives task context, diff, `review.md`, and `SKILLS.md`. **Primary agent is strictly forbidden from self-approving its own work.**
    - **Phase D Close:** Only when subagent review confirms 0 Blockers does agent present GATE 2.
- **Section 4 ("Hard rules"):**
  - Add Rule: "Worker agents operate strictly inside their isolated task worktree/branch. Autonomous commits are permitted on the task branch; workers are forbidden from writing to `main`."
  - Add Rule: "Independent review required before GATE 2; an agent cannot approve its own work."
- Synchronized across:
  - `CLAUDE.md`
  - `GEMINI.md`
  - `AGENTS.md`
  - `internal/templates/data/base/CLAUDE.md.tmpl`
  - `internal/templates/data/base/GEMINI.md.tmpl`
  - `internal/templates/data/base/AGENTS.md.tmpl`
  - Header marker bumped: `<!-- sdd:begin contract templates=1.3.0 -->`

### Canonical Task Coordinator Skill (`skills/task-coordinator/SKILL.md`)
- `agentskills.io` standard format:
  - Frontmatter:
    ```yaml
    ---
    name: task-coordinator
    description: "Orchestrate multi-agent task execution, concurrency deconfliction, independent subagent reviews, and pre-merge qualification in SDD."
    ---
    ```
  - Sections:
    - Overview & Triggers
    - Worker Provisioning & Concurrency Scheduling (`sdd task start`, `sdd task deconflict`)
    - Subagent Review Packaging (`sdd prompt review`)
    - Pre-Merge Qualification & Merge Presentation (`sdd task merge`)
  - Embedded in binary template assets (`internal/templates/data/skills/task-coordinator/SKILL.md`).
  - Supported in `sdd skill export`.

### CLI Subagent Handover Generator (`internal/cli/prompt.go`)
- `sdd prompt review <T-xxx>`:
  - Finds task folder across typologies (`features/`, `bugfixes/`, `hotfixes/`, `spikes/`).
  - Reads `requirements.md`, `plan.md`, `validation.md`, `review.md`.
  - Resolves changed files on task branch (via git diff or plan Section 5).
  - Extracts active feedback rules from `SKILLS.md` §8.
  - Generates turnkey markdown prompt for the subagent reviewer.
  - Flag `--json`: outputs structured JSON review bundle for programmatic Agent SDKs.

### Mechanical Verification in `sdd check`
- `internal/check`:
  - Enforce identical contract parity for the updated Phase C.2 rules.
  - In `checkReviewDocument`: if task is in `Awaiting review` or `Done`, warn if `Reviewer` field is missing or self-signed.
  - Verify template version `1.3.0` kit generation in `check_test.go`.

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Update `CLAUDE.md`, `GEMINI.md`, `AGENTS.md` and their embedded templates with Phase C.2 and two-tier rules. | `CLAUDE.md`, `GEMINI.md`, `AGENTS.md`, `internal/templates/data/base/*.tmpl` | `go test -v ./internal/check -run TestManagedBlockParity` |
| 2 | Bump `templates.Version` to `1.3.0` in `internal/templates/templates.go`. | `internal/templates/templates.go` | `go test -v ./internal/templates` |
| 3 | Create canonical `skills/task-coordinator/SKILL.md`, embed in templates, and support in `sdd skill export`. | `skills/task-coordinator/SKILL.md`, `internal/templates/data/skills/task-coordinator/SKILL.md`, `internal/skill/skill.go` | `go test -v ./internal/skill` |
| 4 | Implement `sdd prompt review <T-xxx>` CLI command and JSON output. | `internal/cli/prompt.go`, `internal/cli/cli.go`, `internal/cli/cli_test.go` | `go test -v ./internal/cli -run TestPromptReview` |
| 5 | Update `internal/check` to verify independent review sign-off and bump test fixtures. | `internal/check/review.go`, `internal/check/check_test.go` | `go test -v ./internal/check` |
| 6 | Regenerate fresh kit in temporary directory to verify Rule 14, run `make check` and `sdd check --strict`, and subagent audit. | `README.md`, `ROADMAP.md` | `make check`, `sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `CLAUDE.md` | modify | Add Phase C.2 Independent Subagent Audit and two-tier rules |
| `GEMINI.md` | modify | Add Phase C.2 Independent Subagent Audit and two-tier rules |
| `AGENTS.md` | modify | Add Phase C.2 Independent Subagent Audit and two-tier rules |
| `internal/templates/data/base/CLAUDE.md.tmpl` | modify | Template contract update |
| `internal/templates/data/base/GEMINI.md.tmpl` | modify | Template contract update |
| `internal/templates/data/base/AGENTS.md.tmpl` | modify | Template contract update |
| `internal/templates/templates.go` | modify | Bump template version to 1.3.0 |
| `skills/task-coordinator/SKILL.md` | create | Canonical Task Coordinator skill |
| `internal/templates/data/skills/task-coordinator/SKILL.md` | create | Embedded Task Coordinator skill template |
| `internal/skill/skill.go` | modify | Support task-coordinator in skill export |
| `internal/cli/prompt.go` | modify | Add `sdd prompt review` subcommand |
| `internal/cli/cli.go` | modify | Update command usage text |
| `internal/cli/cli_test.go` | modify | Test `sdd prompt review` text and JSON output |
| `internal/check/review.go` | modify | Check independent reviewer sign-off |
| `internal/check/check_test.go` | modify | Update tests for template version 1.3.0 |
| `README.md` | modify | Document `sdd prompt review` and `task-coordinator` skill |
| `ROADMAP.md` | modify | Update T-032 status |

## 6. Dependencies
Standard library only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Update T-032 status and progress |
| `README.md` | Document `sdd prompt review` and `task-coordinator` skill |
| Agent files | Updated with Phase C.2 and two-tier governance rules |

## 8. Risks, mitigations, rollback
- **Risk:** Contract divergence between `CLAUDE.md`, `GEMINI.md`, `AGENTS.md` and their templates.
  - **Mitigation:** Strict test `TestManagedBlockParity` enforces 100% byte-for-byte parity across all files.
- **Rollback:** Revert git changes.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- None active in §8 register.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-05   Phrase received: `APPROVED T-032`