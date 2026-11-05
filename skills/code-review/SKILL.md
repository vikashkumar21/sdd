---
name: code-review
description: Deterministic code review rubric, progressive context ingestion, 3-tier risk grading (Blocker, Warning, Minor), and task review artifact guidelines for AI coding agents.
---

# SDD Code Review Standard

> **ANTHEM: ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**

## Overview & Triggers
Activate this skill whenever:
- Performing code review on a feature branch, staged change set, or pull request.
- Verifying code against approved specifications before Phase D / GATE 2 commit.
- Inspecting code diffs for spec alignment, architectural drift, security hazards, or unhandled errors.

## The 4 Review Pillars
Every code review evaluates the changes against four mandatory pillars:

1. **Correctness & Spec Alignment:**
   - Every requirement (R-xxx) in requirements.md maps to concrete implementation and test evidence in validation.md.
   - Touch ONLY files listed in plan.md. Zero scope creep.
   - Negative and failure cases are explicitly tested.

2. **Architectural & Decision Conformance:**
   - Respects all approved decisions in TECH_STACK.md (D-xxx).
   - Zero unapproved third-party dependencies (D-007 standard library purity).
   - Complies with active human feedback rules in SKILLS.md §8 (F-xxx).
   - Deterministic by construction: inject clocks, random sources, and avoid map iteration order dependencies.

3. **Security & Quality Hygiene:**
   - No credentials, tokens, connection strings, or personal data in logs, tests, or code.
   - Bound sizes on all external inputs; fail safe on ambiguity.
   - Clean linting (make lint) and formatting (gofmt -s).

4. **Continuous Learning & Feedback Synthesis:**
   - If a finding reveals a recurring or general pattern, formulate an F-xxx rule for SKILLS.md §8 or scaffold a new skill via sdd skill new.

## 3-Tier Risk & Finding Rubric

| Severity | Definition | Action Required |
|---|---|---|
| 🔴 **Blocker** | Spec violation, security hazard, test gap, plan deviation, unapproved dependency. | MUST be resolved before GATE 2 commit. |
| 🟡 **Warning** | Sub-optimal pattern, edge case risk, missing doc or comment. | Developer attention suggested; address or explicitly accept. |
| 🟢 **Minor** | Style, naming refinement, or non-critical micro-optimization. | Informational suggestion; non-blocking. |

## Review Output Format
Record all findings in features/T-xxx-<slug>/review.md following the standard template. If any 🔴 Blocker remains, the review outcome is "Changes requested" and the task CANNOT proceed to GATE 2.
