# Requirements — T-032 Autonomous multi-agent hierarchy and Task Coordinator skill

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-05)
Roadmap ref: `ROADMAP.md` T-032

## 1. Goal
Codify the two-level autonomous multi-agent hierarchy (Task Coordinator vs Worker Agents) and mandatory independent subagent audit into the agent contracts (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`) and embedded templates, export the canonical `skills/task-coordinator/SKILL.md` skill, provide `sdd prompt review <T-xxx>` to generate subagent review context bundles across coding harnesses, and enforce independent review sign-off in `sdd check`.

## 2. Context and references
- Mission / principles touched:
  - Principle 1 (Deterministic): Deterministic two-tier role definitions and repeatable subagent audit protocol.
  - Principle 3 (Human in the loop for the irreversible): Merge and commit remain strictly human-gated (`APPROVED MERGE T-xxx` / `COMMITTED T-xxx`).
  - Principle 4 (Small and shippable): Isolated task branches prevent cross-task corruption.
- Decisions this relies on:
  - D-002: Spec-driven workflow with human gates.
  - D-007: Go, standard library only, single static binary.
  - D-024: Open Agent Skills standard (`agentskills.io`).
  - D-026: Code review specification, rubric, and task artifact (`review.md`).
  - D-028: Git worktree concurrency and workspace isolation.
  - D-029: Two-tier git governance and Task Coordinator orchestration.
- Depends on tasks: T-031 (Done).

## 3. Scope
**In scope**
- Agent Contract Updates (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md` and `internal/templates/data/base/*.tmpl`):
  - Codify Two-Tier Agent Roles:
    - Worker Agents: Operate strictly inside their isolated git worktree/branch; forbidden from mutating `main`.
    - Task Coordinator Agent: Orchestrates deconfliction, provisions worktrees, qualifies merges, and prepares GATE 2 merge presentations.
  - Codify Phase C.2 (Independent Subagent Audit):
    - After Phase C.1 (validation tests), the primary agent MUST spawn an independent subagent reviewer.
    - Subagent receives task context, diff, `review.md`, and `SKILLS.md`.
    - Primary agent is strictly forbidden from approving its own work.
  - Bump template version to `1.3.0`.
- Canonical Task Coordinator Skill:
  - Create `skills/task-coordinator/SKILL.md` conforming to `agentskills.io` standard.
  - Embed in binary template assets (`internal/templates/data/`) for generation in `sdd init` and export via `sdd skill export`.
- CLI Subagent Handover Generator:
  - `sdd prompt review <T-xxx>`: dynamically compiles and prints a self-contained subagent review prompt including task requirements, plan, changed files, active feedback rules, and 3-tier risk rubric.
  - Flag `--json`: structured review context for automated agent orchestration.
- Mechanical Verification in `sdd check`:
  - Enforce contract parity across all agent files for the new Phase C.2 rules.
  - Verify that `review.md` in tasks marked `Awaiting review` or `Done` has independent review sign-off.
- Template Version Bump Verification:
  - Test kit generation in temporary directory to verify 0 errors on fresh kits (Rule 14).

**Out of scope**
- Multi-lens ensemble review rubrics (Spec Auditor vs AppSec Specialist vs Pattern Auditor) (handled in T-033).
- Remote CI pipeline auto-repair loops (handled in M9).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL update `CLAUDE.md`, `GEMINI.md`, `AGENTS.md` and their embedded templates to codify two-tier agent governance and Phase C.2 Independent Subagent Audit. | MUST | test / inspection |
| R-002 | The agent contract SHALL explicitly state that primary agents are strictly forbidden from self-approving their own work in `review.md`. | MUST | inspection |
| R-003 | The system SHALL bump embedded template version to `1.3.0` across contracts, manifests, and `internal/templates`. | MUST | test |
| R-004 | The system SHALL create and embed canonical `skills/task-coordinator/SKILL.md` conforming to `agentskills.io`. | MUST | test |
| R-005 | `sdd skill export` SHALL support exporting `task-coordinator` to project and global skill directories. | MUST | test |
| R-006 | The system SHALL provide `sdd prompt review <T-xxx>` generating a self-contained subagent review handover bundle. | MUST | test |
| R-007 | `sdd prompt review <T-xxx> --json` SHALL output machine-readable review context for programmatic Agent SDKs. | MUST | test |
| R-008 | `sdd check` SHALL verify that agent contracts match across `CLAUDE.md`, `GEMINI.md`, and `AGENTS.md` including the new multi-agent clauses. | MUST | test |
| R-009 | `sdd check` SHALL warn if a task in `Awaiting review` has self-approved review or missing reviewer attribution in `review.md`. | MUST | test |
| R-010 | The system SHALL verify that `sdd init` generates internally consistent kits matching template version `1.3.0` with 0 errors. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Contract wording and prompt generation must be byte-identical across runs.
- **Fail-safe:** If an agent attempts to present GATE 2 without independent review sign-off, `sdd check` fails in `--strict` mode.
- **Universal compatibility:** Prompts generated by `sdd prompt review` must be compatible with Antigravity, Claude Code, Cursor, and any programmatic Agent SDK.

## 6. Constraints and assumptions
- All agent files (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`) maintain identical rules per §9.
- Skills follow the `agentskills.io` standard (`skills/<name>/SKILL.md`).

## 7. Open questions
(None - all architectural choices settled in D-024, D-026, D-028, and D-029)

## 8. Acceptance criteria
- Given any SDD project, `sdd prompt review T-xxx` prints a turnkey prompt containing task requirements, changed files, active feedback rules, and review instructions.
- Fresh kit generation with `sdd init` produces contracts with Phase C.2 and the `task-coordinator` skill with 0 `sdd check` errors.
- `sdd check --strict` passes cleanly on the repository itself.