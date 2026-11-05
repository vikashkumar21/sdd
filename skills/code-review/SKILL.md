---
name: code-review
description: Deterministic code review rubric, progressive context ingestion, 3-tier risk grading (Blocker, Warning, Minor), and multi-lens ensemble review standards (Spec Auditor, AppSec Specialist, Pattern Auditor) for AI coding agents.
---

# SDD Code Review Standard

> **ANTHEM: ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**

## Overview & Triggers
Activate this skill whenever:
- Performing code review on a feature branch, staged change set, or pull request.
- Verifying code against approved specifications before Phase D / GATE 2 commit.
- Inspecting code diffs for spec alignment, architectural drift, security hazards, or unhandled errors.

## The Ensemble Review Triad (Specialized Lenses)
Every review applies the multi-lens Review Triad to ensure comprehensive coverage:

1. **🔎 Spec Auditor Lens (Scope & Correctness):**
   - **Traceability:** Every requirement (`R-xxx`) in `requirements.md` maps to concrete implementation and test evidence in `validation.md`.
   - **Boundary Enforcement:** Touch ONLY files listed in `plan.md` Section 5. Zero unapproved file modifications or scope creep.
   - **Negative Coverage:** Negative, boundary, and failure cases are explicitly tested and verified.

2. **🛡️ AppSec Specialist Lens (Security & Safety):**
   - **Threat Surface & Input Validation:** Bound sizes on all external inputs; parse strictly; treat untrusted input as hostile.
   - **Confinement & Traversal:** Verify filesystem confinement via `os.Root` or clean paths; reject path traversal and escape sequences.
   - **Secrets & Privacy Hygiene:** Zero credentials, tokens, connection strings, or personal data in logs, tests, docs, or code. Synthetic fixtures only.
   - **Command Safety:** Never construct shell commands via string concatenation; maintain separate arguments.
   - **Fail Safe:** Ambiguous or unknown state fails to the most restrictive outcome.

3. **📐 Pattern & Quality Auditor Lens (Architecture & Standards):**
   - **Decision Conformance:** Respects all approved decisions in `TECH_STACK.md` (`D-xxx`).
   - **Standard Library Purity:** Zero unapproved third-party dependencies (`D-007`).
   - **Determinism:** Inject clocks, random sources, and ID generators; avoid map iteration order dependencies.
   - **Code Quality:** Idiomatic Go (*Effective Go*, *Go Code Review Comments*); `context.Context` first for blocking/IO; `%w` error wrapping; clean linting (`make lint`).
   - **Continuous Learning:** Active feedback rules in `SKILLS.md` §8 (`F-xxx`) complied with; recurring patterns captured.

## 3-Tier Risk & Finding Rubric

| Severity | Definition | Action Required |
|---|---|---|
| 🔴 **Blocker** | Spec violation, security hazard, test gap, plan deviation, unapproved dependency. | MUST be resolved before GATE 2 commit. |
| 🟡 **Warning** | Sub-optimal pattern, edge case risk, missing doc or comment. | Developer attention suggested; address or explicitly accept. |
| 🟢 **Minor** | Style, naming refinement, or non-critical micro-optimization. | Informational suggestion; non-blocking. |

## 3-Loop Circuit Breaker
- Maximum 3 review-fix iterations (1 initial audit + up to 2 fix attempts) between author agent and reviewer subagent.
- If Blockers persist after attempt 3, halt immediately, mark the task `Blocked`, and escalate to the human.

## Review Output Format
Record all findings in `features/T-xxx-<slug>/review.md` (or typology equivalent) following the standard template. If any 🔴 Blocker remains, the review outcome is "Changes requested" and the task CANNOT proceed to GATE 2.
