# Code Review — T-xxx <title>

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Pending | Changes requested | Approved
Reviewer: <Agent Name / Human>
Commit / Diff Ref: <HEAD / diff summary>

## 1. Scope & Defect Alignment
- [ ] Patch resolves the defect without introducing side effects
- [ ] No speculative refactoring: diff modifies ONLY files required for the fix
- [ ] Mandatory Step 0 reproduction test passes after patch

## 2. Architecture & Standards Conformance
- [ ] Complies with approved decisions in TECH_STACK.md (D-xxx)
- [ ] Complies with active feedback rules in SKILLS.md (F-xxx)
- [ ] Concurrency & memory safety verified (no data races or leaks)
- [ ] Security & secrets hygiene verified

## 3. Risk & Findings Register

| ID | Category | Severity | Finding | Suggested Action | Status |
|---|---|---|---|---|---|
| (none) | | | | | |

### Severity Guidelines:
- 🔴 **Blocker (Developer Attention Needed):** Regression, unverified fix, plan drift. MUST be resolved before GATE 2 commit.
- 🟡 **Warning (Attention Suggested):** Sub-optimal pattern, missing boundary test, doc omission.
- 🟢 **Minor (Safe / Informational):** Style, minor naming or micro-optimization.

## 4. Reusable Knowledge & Feedback Synthesis
- [ ] Did this defect reveal a broader recurring pattern?
  - If general: record an F-xxx rule in SKILLS.md §8 (Human feedback rules) or scaffold a skill via `sdd skill new`.
  - If bug-specific: resolved in this change set.

## 5. Review Sign-off
Outcome: Approved | Changes requested
Notes: <summary of review evaluation>
