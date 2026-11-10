# Bugfix Validation — T-xxx <title>

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Not run | In progress | Complete

## 1. Reproduction Proof & Verification

| Check | Command / Test | Before Patch (Unpatched) | After Patch (Fixed) | Status |
|---|---|---|---|---|
| Step 0 Failing Test | `<command>` | FAILED (verified in reproduction.md) | PASSED | PASS / FAIL |

## 2. Regression Suite & Blast Radius Checks

| Target / Package | Command | Expected | Actual | Result |
|---|---|---|---|---|
| Modified Package | `go test -v ./<path>` | pass | <actual> | PASS / FAIL |
| Downstream Blast Radius | `go test -v ./<downstream>` | pass | <actual> | PASS / FAIL |
| Full Repository Suite | `make verify` | pass | <actual> | PASS / FAIL |

## 3. Negative and Boundary Cases
- <boundary condition 1 → observed result>
- <boundary condition 2 → observed result>

## 4. Determinism Check
Ran test suite twice; outputs identical: yes / no.

## 5. Definition of Done
- [ ] Reproduction test failed before patch and passes after patch
- [ ] Full regression suite passes with 0 failures
- [ ] No files modified outside approved plan
- [ ] Living docs updated
- [ ] Suggested commit message provided

## 6. Sign-off
Suggested commit message: `fix(<scope>): <summary> (T-xxx)`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-xxx`
