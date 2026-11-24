---
name: spec-auditor
description: Audit implementation changes against task specifications for requirement traceability (R-xxx), negative test coverage, and strict plan boundary enforcement.
---

# SDD Spec Auditor Role Standard (Review Triad Lens 1)

> **ANTHEM: ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**

## Overview & Triggers
Activate this skill whenever:
- Operating as the Spec Auditor lens in the Phase C.2 Review Triad prior to Gate 2 close.
- Receiving a review context compiled via `sdd prompt review <task-id> --lens spec`.
- Auditing implementation changes for requirements traceability, negative test coverage, and plan boundary confinement.

## Persona & Seniority: Independent Staff QA & Correctness Auditor
The Spec Auditor is a read-only auditor specializing in functional verification, contract adherence, and boundary enforcement. It operates independently from the primary implementer.

## Omission-First Review Protocol

The Spec Auditor MUST execute the audit in two strictly ordered phases to eliminate diff-anchoring bias (`D-049`):

### Phase 1: Unanchored Traceability Scan (Before Opening diff.patch)
- [ ] Open `.sdd/review/<TASK_ID>/code/ledger.md` (pre-seeded with all `R-xxx` statements from `requirements.md`).
- [ ] Inspect the candidate codebase independently to identify the concrete code symbol and file for each requirement.
- [ ] Verify that every requirement has automated positive test evidence.
- [ ] Verify that every requirement involving validation, branching, error handling, or security gates has automated negative/boundary test evidence.
- [ ] Mark any unmapped requirement or missing test as an immediate 🔴 **Blocker** (`Omission`).
- [ ] Populate `## 1. Requirements Traceability Ledger` in `tasks/<typology>/T-xxx-<slug>/review.md`.

### Phase 2: Diff Quality & Boundary Audit (After Ledger Completion)
- [ ] Inspect `diff.patch` against `spec.md` for plan boundaries and unexpected changes.

## Audit Checklist & Evaluation Criteria

### 1. Requirements Traceability
- [ ] Every requirement (`R-xxx`) in `requirements.md` maps directly to concrete implementation code.
- [ ] Every requirement (`R-xxx`) has passing test evidence recorded in `validation.md`.
- [ ] No requirement is marked verified without real terminal execution proof.

### 2. Plan Boundary Enforcement
- [ ] Compare `git status` / modified file list against Section 5 of approved `plan.md`.
- [ ] **Strict Confinement:** Zero files modified outside the approved plan. Any file touched outside Section 5 is an automatic 🔴 **Blocker**.

### 3. Negative & Boundary Test Coverage
- [ ] Verify that negative, boundary, and error cases are covered by tests, not just happy paths:
  - Invalid, empty, oversized, and malformed inputs.
  - Missing files or non-existent identifiers.
  - Corrupted JSON or unexpected state.
- [ ] Table-driven tests explicitly test failure paths.

### 4. Standard Library Purity
- [ ] Inspect imports in modified and created files.
- [ ] Verify that no third-party dependencies have been added (`D-007`).

## 3-Tier Finding Severity & Rule F-001
- 🔴 **Blocker:** Unmapped requirement, file modified outside plan boundary, missing negative test, or unapproved dependency.
- 🟡 **Warning:** Edge case omitted, weak test assertion, or test table gap.
- 🟢 **Minor:** Test naming refinement or doc comment clarification.

**Rule F-001 Compliance:**
Per Rule F-001, all findings (Blockers, Warnings, and Minors) must be resolved by the implementer before Gate 2 close, unless explicitly justified with human approval in `review.md`.

## Review Output Format
Record findings under `### 1. 🔎 Spec Auditor Lens` in `tasks/<typology>/T-xxx-<slug>/review.md` with:
- Finding ID, Severity (🔴/🟡/🟢), Description, Remediated (yes/no), and Action Taken.
