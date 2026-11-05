# Code Review — T-xxx <title>

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Pending | Changes requested | Approved
Reviewer: <Agent Name / Human>
Commit / Diff Ref: <HEAD / diff summary>

## 1. Scope & Spec Alignment
- [ ] Diff implements all requirements (R-001 ... R-xxx) from requirements.md
- [ ] No scope creep: diff modifies ONLY files listed in plan.md
- [ ] No unapproved third-party dependencies added to go.mod / package manifests

## 2. Architecture & Standards Conformance
- [ ] Complies with approved decisions in TECH_STACK.md (D-xxx)
- [ ] Complies with active feedback rules in SKILLS.md (F-xxx)
- [ ] Deterministic by construction (no hidden clocks, random sources, or map order dependency)
- [ ] Security & secrets hygiene: no credentials, tokens, or personal data exposed

## 3. Risk & Findings Register

| ID | Category | Severity | Finding | Suggested Action | Status |
|---|---|---|---|---|---|
| F-001 | Correctness | 🔴 Blocker | <critical defect or spec violation> | <actionable fix> | Open |

### Severity Guidelines:
- 🔴 **Blocker (Developer Attention Needed):** Spec violation, security hazard, test gap, plan drift. MUST be resolved before GATE 2 commit.
- 🟡 **Warning (Attention Suggested):** Sub-optimal pattern, edge case risk, missing doc or comment.
- 🟢 **Minor (Safe / Informational):** Style, minor naming or micro-optimization.

## 4. Reusable Knowledge & Feedback Synthesis
- [ ] Did any review finding reveal a general pattern?
  - If general: record an F-xxx rule in SKILLS.md §8 (Human feedback rules) or scaffold a skill via `sdd skill new`.
  - If task-specific: resolved in this change set.

## 5. Review Sign-off
Outcome: Approved | Changes requested
Notes: <summary of review evaluation>
