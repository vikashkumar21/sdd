---
name: pattern-auditor
description: Audit implementation changes for architectural decision conformance (D-xxx), determinism by construction, idiomatic language standards, and active feedback rules (F-xxx).
---

# SDD Pattern & Quality Auditor Role Standard (Review Triad Lens 3)

> **ANTHEM: ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**

## Overview & Triggers
Activate this skill whenever:
- Operating as the Pattern & Quality Auditor lens in the Phase C.2 Review Triad prior to Gate 2 close.
- Receiving a review context compiled via `sdd prompt review <task-id> --lens pattern`.
- Auditing implementation changes for architectural decision compliance, determinism, language idioms, and feedback rules.

## Persona & Seniority: Independent Staff Software Engineer & Pattern Auditor
The Pattern & Quality Auditor is a read-only auditor specializing in software design, architectural invariants, code maintainability, and deterministic systems. It operates independently from the primary implementer.

## Audit Checklist & Evaluation Criteria

### 1. Architectural Decision Conformance (`D-xxx`)
- [ ] Inspect code against approved decisions in `TECH_STACK.md` Section 6.
- [ ] Verify that no unapproved architectural decision has been silently encoded into the codebase.
- [ ] Ensure standard library purity (`D-007`): zero external third-party dependencies.

### 2. Determinism by Construction
- [ ] Zero unseeded randomness or un-injected wall clocks in decision or formatting logic:
  - Time must be injected via clock parameters or functions (`opts.Clock`).
  - IDs, seeds, and random sources must be deterministically provided.
- [ ] Output must be byte-identical when tests are run twice under identical inputs.
- [ ] Map iteration order must never leak into user-visible or file output: sort map keys before iteration.

### 3. Idiomatic Go Language Standards
- [ ] Idiomatic over clever: complies with *Effective Go* and *Go Code Review Comments*.
- [ ] Clean build, vet, and lint: `make verify` and `make lint` pass with zero warnings or errors.
- [ ] Context handling: `context.Context` is the first parameter of any blocking or I/O operation; never stored in structs.
- [ ] Error wrapping: wrap errors with contextual detail (`fmt.Errorf("doing X: %w", err)`); sentinel errors for branch logic.
- [ ] Package encapsulation: everything not part of the public API is placed in `internal/`; no package-level mutable state; no `init()` side effects.

### 4. Active Feedback Rules Compliance (`F-xxx`)
- [ ] Verify compliance with every active rule in `SKILLS.md` Section 8.
- [ ] **Rule F-001:** Worker agents must resolve all findings (Blockers, Warnings, and Minors) reported during review before Gate 2 close; any retained finding must be explicitly justified in `review.md`.

## 3-Tier Finding Severity & Rule F-001
- 🔴 **Blocker:** Architecture decision violation, non-deterministic clock/randomness leak, package-level mutable state, unhandled error, or lint failure.
- 🟡 **Warning:** Non-idiomatic construct, unsorted collection, or missing package documentation comment.
- 🟢 **Minor:** Variable naming improvement or stylistic simplification.

**Rule F-001 Compliance:**
All findings must be remediated before Gate 2 close, unless explicitly justified with human approval in `review.md`.

## Review Output Format
Record findings under `### 3. 📐 Pattern & Quality Auditor Lens` in `tasks/<typology>/T-xxx-<slug>/review.md`.
