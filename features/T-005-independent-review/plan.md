# Plan — T-005 Independent review, adaptive-design proposals, and improvement plan (docs only until approved)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Implemented

## 1. Approach summary
Conduct an adversarial, evidence-grounded review of `sdd` across four distinct dimensions:
1. Historical context & competitive analysis (genesis, D-001..D-017, web comparison with GitHub Spec Kit and OpenSpec, resolution of H2).
2. Codebase integrity & adversarial testing (package-by-package analysis, coverage gaps, `golangci-lint` audit, regex fragility, edge cases).
3. Modern Go architecture & standard library modernisation (evaluation against Go 1.27.3, stdlib layout, modern stdlib candidates such as `os.Root`, `slices`, `maps`, `errors.Join`).
4. Deterministic adaptive design & proposals catalogue (concrete designs for self-check, self-heal, self-analyse, self-improve; proposals `P-01`..`Pn` with size, risk, dependencies, test plans).

All work is strictly documentation and temporary scratch experiments; no production code outside `features/T-005-independent-review/` will be touched.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-015 | `go` directive in `go.mod` | A) Keep 1.22 (broadest legacy compatibility); B) Raise to 1.24 (unlocks `os.Root` for safe path confinement); C) Raise to 1.27 (aligns with current toolchain) | B (Raise to 1.24 to enable standard-library `os.Root` for filesystem safety while remaining within 2-release support window) | Proposed (Decision block presented below) |

## 3. Design
The review findings and proposals are structured into a standalone document:
- `features/T-005-independent-review/review.md` covering:
  - §1: Project Genesis & Decision Log Verification (D-001 through D-017).
  - §2: Comparative Analysis & Hypothesis H2 (GitHub Spec Kit, OpenSpec, `sdd` differentiation).
  - §3: Adversarial Code Audit & Correctness Analysis (package-by-package breakdown, linter issues, test gaps).
  - §4: Coverage of Project Types (Go, Java, Python, TypeScript, monorepos, brownfield, multi-agent).
  - §5: Language Pack Review (standards, contracts, skillmaps).
  - §6: Deterministic Adaptive Architecture (Self-Check, Self-Heal, Self-Analyse, Self-Improve, AI Adaptation).
  - §7: Modern Go Review & Toolchain Audit (Go 1.27.3, `os.Root`, `slices`, `maps`, `errors.Join`, `golangci-lint` v2).
  - §8: Prioritised Proposals Catalogue (`P-01` through `Pn`).
  - §9: Verification Evidence & Scratch Experiment Results.

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Toolchain and environment verification | temporary scratch dir | `go version`, `golangci-lint run`, `make verify`, `go test -race ./...` |
| 2 | Draft requirements, plan, validation, and review framework | `features/T-005-independent-review/` | Document inspection, `sdd check --dir .` |
| 3 | Present GATE 1 to human maintainer | Output to chat | Human reply `APPROVED T-005` |
| 4 | Expand `review.md` in full (sections 7–10) upon approval | `features/T-005-independent-review/review.md` | Evidence verification, review completeness |
| 5 | Present GATE 2 for review and proposals catalogue | Output to chat | Human reply `SELECT P-xx, ...` or `COMMITTED T-005` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `features/T-005-independent-review/requirements.md` | create | Requirements for T-005 |
| `features/T-005-independent-review/plan.md` | create | Execution plan for T-005 |
| `features/T-005-independent-review/validation.md` | create | Traceability and verification records |
| `features/T-005-independent-review/review.md` | create | Comprehensive review report and proposals catalogue |

*(No files outside `features/T-005-independent-review/` are created or modified during this task.)*

## 6. Dependencies
None. Standard library only (D-007).

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Update status of T-005 upon progression; add new proposal tasks `T-xxx` when selected by maintainer |
| `TECH_STACK.md` | Record resolution of D-015 if decided; log newly verified facts |
| `SKILLS.md` | Record any general feedback rules (F-xxx) or pitfalls identified |
| `README.md` | None (review is internal; user-facing behavior unchanged) |

## 8. Risks, mitigations, rollback
- **Risk:** Scope creep during review distracting from core determinism.
  - *Mitigation:* Frame all actionable ideas as strictly gated, independent proposals (`P-01`..`Pn`) requiring explicit human selection.
- **Risk:** Premature code changes before proposals are approved.
  - *Mitigation:* Hard constraint: zero production code edits in T-005. Only documentation in `features/T-005-independent-review/`.
- **Rollback:** `rm -rf features/T-005-independent-review/` and restore `ROADMAP.md` row to `Not started`.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active yet.

## 10. Approval record
Approved by: Vikash   Date: 2026-10-29   Phrase received: `APPROVED T-005`
