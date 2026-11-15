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
1. **No Code Before GATE 1:** Only task documentation in `tasks/<typology>/T-xxx-<slug>/` (`requirements.md`, `plan.md`, `validation.md`, `review.md`) may be written during Phase A.
2. **Never Commit or Push:** The human always reviews the diff and runs git commit. Agents only suggest commit messages.
3. **Fail Safe:** When input, state, or intent is unknown or ambiguous, choose the most restrictive outcome or stop and ask.
4. **Deterministic by Construction:** Same input gives same output. No hidden clocks, randomness, or map-iteration order in decision logic.
5. **Standard Library Only:** Zero external third-party dependencies without an approved human decision.
6. **No Scope Creep:** Unplanned findings go to `ROADMAP.md` Parking lot, never into the current task.
7. **Rule F-001 (Resolve All Findings):** Worker agents must resolve all findings (Blockers, Warnings, Minors) before presenting to the human at GATE 2.

## The 4-Phase Workflow
Every task follows this sequence without skipping or reordering:

| Phase | What the agent does | Gate |
|---|---|---|
| A. Specify | Create `tasks/<typology>/T-xxx-<slug>/` and draft `requirements.md`, `plan.md`, `validation.md`, `review.md`. Invoke independent Plan Review Subagent (`sdd prompt plan-review`). Remediate findings until 0 Blockers. | GATE 1 (`APPROVED T-xxx`) |
| B. Implement | Execute `plan.md` step by step. Touch only planned files. | none |
| C.1 Verify | Run all checks in `validation.md` and record real commands/results. | none |
| C.2 Review Triad | Spawn 3 parallel subagents: Spec Auditor, AppSec Specialist, Pattern Auditor (`sdd prompt review --lens <lens>`). Remediate findings in unified fix pass (max 3 loops). | 0 Blockers across all lenses |
| D. Close | Update living docs, set status to `Awaiting review`, present diff and commit message. | GATE 2 (`COMMITTED T-xxx`) |

## Decision Protocol (D-xxx)
When a choice is not settled in `TECH_STACK.md` or an approved plan, STOP and emit:
```
DECISION NEEDED: D-xxx — <title>
Context:        <why this must be decided now>
Options:        A) …   B) …   C) …
Recommendation: <option + reason>
Blocks:         <task or step>
Reply with:     DECIDE D-xxx: <A|B|C|other>
```

## Progressive Disclosure & SDK Agent Ingestion
- Programmatic SDK agents (Antigravity Python SDK, Anthropic Agent SDK, LangGraph) can load this skill directory or inject this contract into their system prompts.
- All actions taken by SDK agents must respect GATE 1 (waiting for human approval string) before executing code-modifying tools.
