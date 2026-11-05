# Code Review — T-xxx <title>

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Pending | Changes requested | Approved
Reviewer: <Agent Name / Human>
Commit / Diff Ref: <HEAD / diff summary>

## 1. Multi-Lens Ensemble Review

### 🔎 Spec Auditor Lens
- [ ] Patch resolves the defect without introducing side effects
- [ ] No speculative refactoring: diff modifies ONLY files required for the fix
- [ ] Mandatory Step 0 reproduction test passes after patch

### 🛡️ AppSec Specialist Lens
- [ ] Root cause fix does not introduce new attack surfaces or security regressions
- [ ] Input validation, size bounds, and path confinement verified
- [ ] Secrets & privacy hygiene verified (synthetic fixtures only)

### 📐 Pattern & Quality Auditor Lens
- [ ] Complies with approved decisions in TECH_STACK.md (D-xxx)
- [ ] Complies with active feedback rules in SKILLS.md §8 (F-xxx)
- [ ] Concurrency & memory safety verified (no data races or leaks)
- [ ] Clean linting and language idioms preserved

## 2. Risk & Findings Register

| ID | Category | Severity | Finding | Suggested Action | Status |
|---|---|---|---|---|---|
| (none) | | | | | |

### Severity Guidelines:
- 🔴 **Blocker (Developer Attention Needed):** Regression, unverified fix, plan drift. MUST be resolved before GATE 2 commit.
- 🟡 **Warning (Attention Suggested):** Sub-optimal pattern, edge case risk, missing doc or comment.
- 🟢 **Minor (Safe / Informational):** Style, minor naming or micro-optimization.

## 3. Reusable Knowledge & Feedback Synthesis
- [ ] Did this defect reveal a broader recurring pattern?
  - If general: record an F-xxx rule in SKILLS.md §8 (Human feedback rules) or scaffold a skill via `sdd skill new`.
  - If bug-specific: resolved in this change set.

## 4. Review Sign-off
Outcome: Approved | Changes requested
Notes: <summary of review evaluation>
