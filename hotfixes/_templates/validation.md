# Hotfix Validation — T-xxx <title>

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Not run | Complete

## 1. Incident Symptom Verification
- Command / Test: `<command to verify incident is resolved>`
- Expected: Incident symptom eliminated
- Actual: <paste result>
- Result: PASS / FAIL

## 2. Blast-Radius Verification
- Full test suite: `make verify` -> PASS / FAIL

## 3. Emergency Release & Cherry-Pick Plan
- [ ] Tag hotfix release (e.g. `v1.3.1`)
- [ ] Cherry-pick / merge back into `main` branch to prevent regression

## 4. Sign-off
Suggested commit message: `fix(hotfix): <summary> (T-xxx)`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-xxx`
