# Validation — T-xxx <title>

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Not run | In progress | Complete

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `<command>` | pass | <paste> | PASS / FAIL / NOT VERIFIED (reason) |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| <build command from the Decision register> | |
| <test command from the Decision register> | |
| <lint command from the Decision register> | |

## 3. Manual checks
- <step → observed result>

## 4. Negative and failure cases
- <invalid input → …>
- <unknown input → …>

## 5. Determinism check
Ran <command> twice; outputs identical: yes / no. Evidence: <…>

## 6. Deviations, known limitations, and feedback-rule compliance
<None, or list. Any deviation from the approved plan must have been re-approved.>

Feedback-rule compliance (SKILLS.md, "Human feedback rules"): <F-xxx: pass / fail / n.a. per active rule, or "none active">

## 7. Definition of done
- [ ] Every requirement has evidence
- [ ] Build, lint, and tests pass (commands above)
- [ ] Negative and failure cases tested
- [ ] Determinism confirmed
- [ ] No files changed outside the plan; no unapproved dependencies
- [ ] Living docs updated
- [ ] Compliant with every active feedback rule
- [ ] Suggested commit message provided

## 8. Sign-off
Suggested commit message: `<type>(<scope>): <summary>`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-xxx`
