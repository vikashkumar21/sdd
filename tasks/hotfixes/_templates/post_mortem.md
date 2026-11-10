# Hotfix Post-Mortem — T-xxx <title>

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Draft | Complete

## 1. Timeline
- `<YYYY-MM-DD HH:MM>` — Defect introduced in commit `<hash>`
- `<YYYY-MM-DD HH:MM>` — Incident detected / alerted
- `<YYYY-MM-DD HH:MM>` — Hotfix branched, verified, and deployed
- `<YYYY-MM-DD HH:MM>` — Incident mitigated

## 2. Root Cause Analysis
<Technical description of why the failure occurred in production.>

## 3. Detection Gap
<Why did automated unit tests, integration tests, or CI fail to detect this before release?>

## 4. Preventative Follow-Up Items
| Action Item | Owner | Task ID / Ref |
|---|---|---|
| Add regression test to CI matrix | Agent / Human | <T-xxx> |
| Update lint rule or feedback rule | Agent / Human | <F-xxx> |
