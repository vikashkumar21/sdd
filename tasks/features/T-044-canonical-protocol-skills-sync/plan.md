---
status: active
task_id: T-044
last_reviewed: 2026-11-15
---
# Plan — T-044 Canonical protocol skills synchronization and taxonomy alignment

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved

## 1. Approach summary
Synchronize canonical protocol skills (`sdd-protocol`, `code-review`, and `task-coordinator` / `coordinator`) across project-level `skills/`, embedded `internal/templates/data/skills/`, and string constants in `internal/skill/export.go`. Update all legacy `features/` paths to modern `tasks/<typology>/` paths, incorporate the 4th task artifact `review.md`, Pre-GATE 1 Plan Review Subagent, Phase C.2 3-Subagent Parallel Review Triad, and Rule F-001 finding resolution requirements. Synthesize `coordinator` and `task-coordinator` skills into a unified comprehensive standard, and add symmetrical aliasing with frontmatter name normalization in `internal/role/role.go` so both names resolve deterministically and pass `agentskills.io` directory basename validation.

## 2. Decisions (new or affected)
None. Conforms to approved decisions D-002, D-007, D-024, D-026, D-029, D-030, D-031, D-032, and D-036.

## 3. Design

### Updated `sdd-protocol` Contract Content
```markdown
---
name: sdd-protocol
description: Spec-Driven Development (SDD) contract, deterministic gating protocol (GATE 1, GATE 2), decision protocols, and task execution rules for AI coding agents.
---

# Spec-Driven Development (SDD) Protocol

> **ANTHEM: ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**

## Overview & Triggers
Activate this skill whenever:
- Taking over an existing repository or bootstrapping a greenfield project.
- Specifying, planning, implementing, or reviewing any task across work typologies (features, bugfixes, hotfixes, spikes).
- A human requires strict gating, deterministic verification, and non-negotiable review checkpoints before modifying code or living documents.

## The SDD Anthem & Hard Rules
1. **No Code Before GATE 1:** Only task documentation in tasks/<typology>/T-xxx-<slug>/ (requirements.md, plan.md, validation.md, review.md) may be written during Phase A.
2. **Never Commit or Push:** The human always reviews the diff and runs git commit. Agents only suggest commit messages.
3. **Fail Safe:** When input, state, or intent is unknown or ambiguous, choose the most restrictive outcome or stop and ask.
4. **Deterministic by Construction:** Same input gives same output. No hidden clocks, randomness, or map-iteration order in decision logic.
5. **Standard Library Only:** Zero external third-party dependencies without an approved human decision.
6. **No Scope Creep:** Unplanned findings go to ROADMAP.md Parking lot, never into the current task.
7. **Rule F-001 (Resolve All Findings):** Worker agents must resolve all findings (Blockers, Warnings, Minors) before presenting to the human at GATE 2.

## The 4-Phase Workflow
Every task follows this sequence without skipping or reordering:

| Phase | What the agent does | Gate |
|---|---|---|
| A. Specify | Create tasks/<typology>/T-xxx-<slug>/ and draft requirements.md, plan.md, validation.md, review.md. Invoke independent Plan Review Subagent (sdd prompt plan-review). Remediate findings until 0 Blockers. | GATE 1 (APPROVED T-xxx) |
| B. Implement | Execute plan.md step by step. Touch only planned files. | none |
| C.1 Verify | Run all checks in validation.md and record real commands/results. | none |
| C.2 Review Triad | Spawn 3 parallel subagents: Spec Auditor, AppSec Specialist, Pattern Auditor (sdd prompt review --lens <lens>). Remediate findings in unified fix pass (max 3 loops). | 0 Blockers across all lenses |
| D. Close | Update living docs, set status to Awaiting review, present diff and commit message. | GATE 2 (COMMITTED T-xxx) |
```

### Coordinator / Task-Coordinator Synthesis & Aliasing
- **Unified Synthesis:** Synthesize the operational orchestration directives from `coordinator/SKILL.md` (task decomposition guardrails $\le 10$–$15$ files, deterministic `.sdd/state/` state persistence and event journaling, worktree isolation, collision deconfliction) with the procedural governance workflows from `task-coordinator/SKILL.md` (Pre-GATE 1 Plan Review subagent, Phase C.2 Review Triad, Rule F-001, and pre-merge qualification).
- **Frontmatter Name Normalization in `ResolveSkill`:**
  When `ResolveSkill(root, skillIdentifier, set)` resolves via alias (e.g. `skillIdentifier == "task-coordinator"` falling back to `coordinator`, or vice versa), `ResolveSkill` replaces `name: <source>` with `name: <requested>` in the YAML frontmatter. This ensures that any caller exporting the skill to `skills/<requested>/SKILL.md` passes `skill.Validate` directory basename checks without error.

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Update `CanonicalSDDProtocolSkill`, `CanonicalCodeReviewSkill`, and `CanonicalTaskCoordinatorSkill` in `internal/skill/export.go` and `skills/sdd-protocol/SKILL.md` to reflect unified `tasks/` hierarchy, `review.md`, Plan Review Subagent, Review Triad, and Rule F-001 | `internal/skill/export.go`, `skills/sdd-protocol/SKILL.md` | `go test -v ./internal/skill -run "TestCanonicalSDDProtocolSkillValid\|TestCanonicalCodeReviewSkillValid\|TestCanonicalTaskCoordinatorSkillValid"` |
| 2 | Align `skills/task-coordinator/SKILL.md`, `internal/templates/data/skills/coordinator/SKILL.md`, and `internal/templates/data/skills/task-coordinator/SKILL.md`, and implement symmetrical aliasing with frontmatter name normalization in `internal/role/role.go:ResolveSkill` | `internal/templates/data/skills/coordinator/SKILL.md`, `internal/templates/data/skills/task-coordinator/SKILL.md`, `skills/task-coordinator/SKILL.md`, `internal/role/role.go`, `internal/role/role_test.go` | `go test -v ./internal/role` |
| 3 | Update export and CLI tests to assert synchronized skill contents, modern path references, and coordinator aliasing | `internal/skill/export_test.go`, `internal/cli/skill_test.go`, `internal/templates/templates_test.go` | `go test -v ./internal/skill && go test -v ./internal/cli -run TestSkillExport` |
| 4 | Update living docs (`TECH_STACK.md`, `ROADMAP.md`) and run full verification suite | `TECH_STACK.md`, `ROADMAP.md`, `tasks/features/T-044-canonical-protocol-skills-sync/*` | `make verify && make check && ./bin/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `ROADMAP.md` | modify | Track milestone and task lifecycle status |
| `TECH_STACK.md` | modify | Record verified fact for protocol skills synchronization |
| `internal/cli/skill_test.go` | modify | Verify CLI export of synchronized protocol and role skills |
| `internal/role/role.go` | modify | Symmetrical aliasing between coordinator and task-coordinator in ResolveSkill with frontmatter name normalization |
| `internal/role/role_test.go` | modify | Unit tests for coordinator/task-coordinator aliasing and frontmatter names |
| `internal/skill/export.go` | modify | Update CanonicalSDDProtocolSkill, CanonicalCodeReviewSkill, and CanonicalTaskCoordinatorSkill constants |
| `internal/skill/export_test.go` | modify | Unit tests verifying updated paths and Rule F-001 in exported skills |
| `internal/templates/data/skills/coordinator/SKILL.md` | modify | Synchronize coordinator role skill with unified orchestration standard |
| `internal/templates/data/skills/task-coordinator/SKILL.md` | modify | Synchronize embedded task-coordinator skill template |
| `internal/templates/templates_test.go` | modify | Assert embedded skills validity |
| `skills/sdd-protocol/SKILL.md` | modify | Update project-level sdd-protocol skill to modern SDD standards |
| `skills/task-coordinator/SKILL.md` | modify | Synchronize task-coordinator skill with canonical coordinator template |
| `tasks/features/T-044-canonical-protocol-skills-sync/plan.md` | modify | Task implementation plan |
| `tasks/features/T-044-canonical-protocol-skills-sync/requirements.md` | modify | Task functional and non-functional requirements |
| `tasks/features/T-044-canonical-protocol-skills-sync/review.md` | modify | Review Triad sign-off |
| `tasks/features/T-044-canonical-protocol-skills-sync/validation.md` | modify | Traceability matrix and verification evidence |

## 6. Dependencies
None. Pure Go standard library per Decision D-007.

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Update T-044 status across lifecycle (`Specifying` -> `Implementing` -> `Awaiting review` -> `Done`) |
| `TECH_STACK.md` | Record verified fact for protocol skills synchronization and aliasing |

## 8. Risks, mitigations, rollback
- **Risk:** Existing tests expecting exact match on legacy `CanonicalSDDProtocolSkill` or `CanonicalCodeReviewSkill` text.
  - *Mitigation:* Swept `internal/skill/export_test.go` and `internal/cli/cli_test.go`; tests assert substrings like `name: sdd-protocol` rather than entire byte blocks.
- **Rollback:** Revert `internal/skill/export.go`, `internal/role/role.go`, `internal/templates/data/skills/`, and `skills/`.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- **Rule F-001 (Resolve all review findings: Blockers, Warnings, and Minors):** Worker agent must resolve all findings reported across the 3 subagent review lenses before presenting to human at GATE 2.

## 10. Approval record
Plan Review Subagent: Approved (0 Blockers) 2026-11-15 (conv: 12d8fecd-af6a-4adc-a865-db5aadb3abe4)
Approved by: Vikash (human)   Date: 2026-11-15   Phrase received: `APPROVED T-044`
