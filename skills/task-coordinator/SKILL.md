---
name: task-coordinator
description: Orchestrate multi-agent task execution, concurrency deconfliction, independent subagent reviews, and pre-merge qualification in SDD.
---

# SDD Task Coordinator Standard

> **ANTHEM: ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**

## Overview & Triggers
Activate this skill whenever:
- Operating as an orchestrator or coordinator agent supervising one or more worker agents.
- Managing concurrent tasks across isolated git worktrees or branches.
- Handing off completed implementation to an independent subagent reviewer before GATE 2.
- Running pre-merge qualification, blast-radius regression checks, or presenting merges for human review.

## Two-Tier Agent Roles & Boundaries
1. **Worker Agents:**
   - Operate strictly inside their isolated git worktree / task branch (`task/<T-xxx>`).
   - Autonomous commits are permitted on the isolated task branch.
   - Strictly forbidden from writing to, rebasing, or merging into `main`.
   - Cannot self-approve their own `review.md`.
2. **Task Coordinator Agent:**
   - Supervises task provisioning, scheduling, and deconfliction (`sdd task start`, `sdd task deconflict`).
   - Dispatches Pre-GATE 1 Plan Review Subagent (`sdd prompt plan-review <T-xxx>`) before presenting to the human.
   - Dispatches 3-Subagent Parallel Review Triad (`sdd prompt review <T-xxx> --lens <spec|security|pattern>`).
   - Executes pre-merge qualification simulations and regression sweeps (`sdd task merge`).
   - Presents qualified merges and suggested commit messages to the human for GATE 2 sign-off (`APPROVED MERGE T-xxx` / `COMMITTED T-xxx`).

## Phase A Pre-GATE 1 Plan Review Protocol
- When task artifacts (`requirements.md`, `plan.md`, `validation.md`, `review.md`) are drafted, an independent Plan Review Subagent MUST be dispatched before presenting GATE 1 to the human.
- The handover prompt is compiled using `sdd prompt plan-review <T-xxx>` (or `--json` for programmatic agent harnesses).
- Reviewer audits:
  1. Spec completeness: Atomic `R-xxx` requirements, clear acceptance criteria, zero vague assertions.
  2. Plan soundness: Steps reviewable in one sitting; each step names files and proving check.
  3. Boundary precision: Section 5 files list is exhaustive; zero extraneous files or unapproved dependencies.
  4. Risk & rollback: Failure modes and clean recovery paths.
  5. Testability: Traceability matrix maps every requirement to concrete tests.
- All plan findings must be resolved before presenting to the human at GATE 1. Bounded by a 3-loop circuit breaker: max 3 plan review-fix iterations before escalating to the human.

## Phase C.2 3-Subagent Parallel Review Triad Protocol
- When a worker agent completes Phase C.1 validation checks, three independent subagents MUST be spawned in parallel across specialized lenses:
  1. 🔎 `Spec Auditor`: requirements traceability, negative test cases, plan boundary enforcement, standard library purity.
  2. 🛡️ `AppSec Specialist`: threat surface, input validation, filesystem confinement (`os.Root`), secrets hygiene, command safety.
  3. 📐 `Pattern & Quality Auditor`: architectural decision conformance (`D-xxx`), determinism, language idioms, active feedback rules (`F-xxx`).
- The handover prompts are compiled using `sdd prompt review <T-xxx> --lens <spec|security|pattern>` (or `--json`).
- **Batch Remediation Protocol:**
  - When multiple subagents report changes, the primary worker agent consolidates all remediations into a single fix pass.
  - Review subagents are strictly read-only auditors; only the primary worker agent modifies code.
  - Resolving a batch of findings across multiple lenses consumes exactly 1 iteration in the 3-loop circuit breaker.
  - Per Rule F-001, the remediation pass must address all findings (🔴 Blockers, 🟡 Warnings, and 🟢 Minors); any retained finding requires explicit justification.
- **Conflict Resolution:** Safety, security, and determinism take precedence over convenience. If review recommendations fundamentally conflict, the agent halts and issues `DECISION NEEDED: D-xxx` to the human.
- **3-Loop Circuit Breaker:**
  - Maximum 3 iterations (1 initial review + up to 2 fix attempts).
  - If Blockers persist after attempt 3, the Task Coordinator halts, marks the task `Blocked`, and escalates directly to the human.

## Pre-Merge Qualification Workflow
Before any merge into `main`:
1. Check task status: must be `Awaiting review`.
2. Check independent review sign-off: `review.md` must have status `Approved` with 0 Blockers, and all findings (Blockers, Warnings, Minors) resolved or justified per Rule F-001.
3. Simulate git merge in an isolated throwaway worktree: `sdd task merge --simulate <T-xxx>`.
4. Run full blast-radius regression test suite on the simulated merge commit.
5. Present merge qualification summary to the human for approval.
