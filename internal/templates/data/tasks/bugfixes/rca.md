---
status: draft
---
# Root Cause Analysis — T-xxx <title>

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Draft | Approved (YYYY-MM-DD)
Roadmap ref: `ROADMAP.md` T-xxx

## 1. Underlying Root Cause
<Explain WHY the bug occurred. Was it a logic flaw, off-by-one boundary, state synchronization race, missing input sanitization, or unhandled error?>

## 2. Affected Code & Mechanism
- **Primary Defect Location:** `<file path and line numbers>`
- **Faulty Function / Struct:** `<symbol name>`
- **Mechanism of Failure:** <Walk through the exact execution path that leads to the faulty state>

## 3. Blast-Radius Assessment
<Identify every downstream package, caller, or user-facing feature that interacts with this code. What could break if this code is modified?>
- Direct Callers: `<list>`
- Downstream Packages / Modules: `<list>`
- Shared Data / State Boundaries: `<list>`

## 4. Regression Risk & Prevention Strategy
- **Risk Level:** High / Medium / Low
- **Prevention Strategy:** <How will we ensure this class of bug cannot recur? E.g., boundary validation, invariant assertion, table-driven test coverage>
