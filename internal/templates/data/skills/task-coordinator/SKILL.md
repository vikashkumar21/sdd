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
   - Dispatches independent subagent reviewers using turnkey review contexts (`sdd prompt review`).
   - Executes pre-merge qualification simulations and regression sweeps (`sdd task merge`).
   - Presents qualified merges and suggested commit messages to the human for GATE 2 sign-off (`APPROVED MERGE T-xxx` / `COMMITTED T-xxx`).

## Phase C.2 Independent Subagent Audit Protocol
- When a worker agent completes Phase C.1 validation checks, an independent subagent reviewer MUST be spawned.
- The handover prompt is compiled using `sdd prompt review <T-xxx>` (or `--json` for programmatic agent harnesses).
- Reviewer audits:
  1. Correctness & spec traceability (`requirements.md` to `validation.md`).
  2. Plan boundary compliance (no file touched outside `plan.md`).
  3. Security & quality hygiene (no leaked secrets, stdlib purity, lint clean).
  4. Active feedback rules in `SKILLS.md` §8.
- **3-Loop Circuit Breaker:**
  - Maximum 3 iterations (1 initial review + up to 2 fix attempts).
  - If Blockers persist after attempt 3, the Task Coordinator halts, marks the task `Blocked`, and escalates directly to the human.

## Pre-Merge Qualification Workflow
Before any merge into `main`:
1. Check task status: must be `Awaiting review`.
2. Check independent review sign-off: `review.md` must have status `Approved` with 0 Blockers.
3. Simulate git merge in an isolated throwaway worktree: `sdd task merge --simulate <T-xxx>`.
4. Run full blast-radius regression test suite on the simulated merge commit.
5. Present merge qualification summary to the human for approval.
