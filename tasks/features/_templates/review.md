---
status: draft
---
# Code Review — T-xxx <title>

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Pending | Changes requested | Approved
Reviewer: <Agent Name / Human>
Commit / Diff Ref: <HEAD / diff summary>

## 1. Multi-Lens Ensemble Review

### 🔎 Spec Auditor Lens
- [ ] Diff implements all requirements (R-001 ... R-xxx) from requirements.md
- [ ] No scope creep: diff modifies ONLY files listed in plan.md Section 5
- [ ] Negative and edge cases are explicitly tested and verified in validation.md
- [ ] Zero unapproved third-party dependencies added to package manifests

### 🛡️ AppSec Specialist Lens
- [ ] Input validation: bound sizes on all external inputs; fail-safe on ambiguity
- [ ] Confinement: filesystem operations confined (os.Root / clean paths; no traversal)
- [ ] Secrets hygiene: zero credentials, tokens, or PII exposed; synthetic fixtures only
- [ ] Execution safety: commands use separate arguments; no string-concatenated shell execution

### 📐 Pattern & Quality Auditor Lens
- [ ] Complies with approved decisions in TECH_STACK.md (D-xxx)
- [ ] Complies with active feedback rules in SKILLS.md §8 (F-xxx)
- [ ] Deterministic by construction (injected clocks, fixed random seeds, canonical map order)
- [ ] Language standards: Effective Go conventions, context.Context first, clean linting (make lint)

## 2. Risk & Findings Register

| ID | Category | Severity | Finding | Suggested Action | Status |
|---|---|---|---|---|---|
| F-001 | Correctness | 🔴 Blocker | <critical defect or spec violation> | <actionable fix> | Open |

### Severity Guidelines:
- 🔴 **Blocker (Developer Attention Needed):** Spec violation, security hazard, test gap, plan drift. MUST be resolved before GATE 2 commit.
- 🟡 **Warning (Attention Suggested):** Sub-optimal pattern, edge case risk, missing doc or comment.
- 🟢 **Minor (Safe / Informational):** Style, minor naming or micro-optimization.

## 3. Reusable Knowledge & Feedback Synthesis
- [ ] Did any review finding reveal a general pattern?
  - If general: record an F-xxx rule in SKILLS.md §8 (Human feedback rules) or scaffold a skill via `sdd skill new`.
  - If task-specific: resolved in this change set.

## 4. Review Sign-off
Outcome: Approved | Changes requested
Notes: <summary of review evaluation>
