---
status: draft
---
# Bugfix Plan — T-xxx <title>

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Draft | Awaiting approval | Approved (YYYY-MM-DD) | Implemented

## 1. Approach Summary
<Minimal patch strategy. Speculative refactorings or out-of-scope cleanups are STRICTLY FORBIDDEN in bugfixes.>

## 2. Implementation Steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Apply minimal patch to resolve root cause | <path> | <reproduction test passes> |
| 2 | Add comprehensive regression tests (edge cases, boundary values) | <path> | <test command> |

## 3. Files to Create or Modify

| Path | Action | Reason |
|---|---|---|
| <path> | modify / create | <reason> |

## 4. Dependencies
<Standard library only (D-007) unless explicitly approved.>

## 5. Living Docs to Update

| File | Change |
|---|---|
| `ROADMAP.md` | Status update |
| `SKILLS.md` | Pitfalls log or F-rule if a general pattern was discovered |

## 6. Risks, Mitigations, Rollback
- Risk: <downstream blast radius> → Mitigation: <comprehensive test matrix>
- Rollback: Revert commit / patch.

## 7. Applicable Human-Feedback Rules (SKILLS.md §8)
<List active F-rules that apply, or "none active">

## 8. Approval Record
Approved by: ______   Date: ______   Phrase received: `APPROVED T-xxx`
