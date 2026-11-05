# Validation — T-033 Ensemble code review standards and AppSec Specialist rubric

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Verified

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/skill -run TestCanonicalCodeReviewSkillValid` | Canonical skill contains Review Triad standard and passes validation | Passed (triad standard verified in skill body) | PASS |
| R-002 | test | `go test -v ./internal/cli -run TestPromptReviewLens` | `sdd prompt review <T-xxx> --lens <spec\|security\|pattern\|all>` generates tailored prompts | Passed (all 4 lens prompts generated and verified) | PASS |
| R-003 | test | `go test -v ./internal/cli -run TestPromptReviewLens` | `sdd prompt review <T-xxx> --lens appsec` aliases to security lens | Passed (appsec alias mapped to security) | PASS |
| R-004 | test | `go test -v ./internal/cli -run TestPromptReviewLens` | `sdd prompt review <T-xxx> --lens <lens> --json` includes lens metadata in JSON | Passed (lens and lensFocus present in parsed JSON) | PASS |
| R-005 | test | `go test -v ./internal/cli -run TestPromptReviewLens` | Invalid lens value exits with code 2 and helpful error | Passed (code 2 returned with valid choices help text) | PASS |
| R-006 | test | `go test -v ./internal/check` | Review templates include Review Triad checklist | Passed (all review templates checked and valid) | PASS |
| R-007 | test | `go test -v ./internal/cli -run TestSkillExport` | `sdd skill export --skill code-review` exports updated canonical skill | Passed (code-review skill exported cleanly) | PASS |
| R-008 | test | `go test -v ./internal/check -run TestCheckReviewDocument` | `sdd check` verifies review.md adherence | Passed (review check verifies rubric & sign-off) | PASS |
| R-009 | inspection | Inspect `SKILLS.md` §10 | `SKILLS.md` §10 updated with ensemble review triad | Inspected: Section 10 fully codifies Review Triad & 3-tier rubric | PASS |
| R-010 | test | `make check` and `sdd check --strict` | 0 errors and 0 warnings | 0 lint issues, all tests passed, 0 check errors/warnings | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | Passed (all unit & integration tests green) |
| `make check` | Passed (0 golangci-lint issues, all unit & integration tests green) |
| `sdd check --strict` | Passed (0 error(s), 0 warning(s), 1 note(s)) |

## 3. Manual checks
- `sdd prompt review T-033 --lens security` -> outputs AppSec Specialist focused prompt (input bounds, path confinement, secrets hygiene, command safety).
- `sdd prompt review T-033 --lens spec` -> outputs Spec Auditor focused prompt (requirements traceability, boundary enforcement, negative cases).
- `sdd prompt review T-033 --lens pattern` -> outputs Pattern Auditor focused prompt (decision conformance, determinism, language idioms, feedback rules).
- `sdd prompt review T-033 --lens all` -> outputs full ensemble prompt.

## 4. Negative and failure cases
- `sdd prompt review non-existent task ID` -> exits with code 1.
- `sdd prompt review T-033 --lens invalid` -> exits with code 2 (`sdd prompt review: invalid --lens "invalid" (expected: spec, security, pattern, or all)`).
- `sdd prompt review` (missing task ID) -> exits with code 2 (`Usage: sdd prompt review <T-xxx> [--lens <spec|security|pattern|all>] [--json]`).

## 5. Determinism check
Ran `./bin/sdd prompt review T-033 --lens security --json` twice; diff was byte-identical (0 diff).

## 6. Deviations, known limitations, and feedback-rule compliance
- Deviations: none.
- Feedback-rule compliance (SKILLS.md, "Human feedback rules"): none active.

## 7. Definition of done
- [x] Every requirement has evidence
- [x] Build, lint, and tests pass
- [x] Negative and failure cases tested
- [x] Determinism confirmed
- [x] No files changed outside the plan; no unapproved dependencies
- [x] Living docs updated
- [x] Compliant with every active feedback rule
- [x] Suggested commit message provided

## 8. Sign-off
Suggested commit message: `feat(review): codify ensemble review triad, AppSec Specialist rubric, and lens CLI options (T-033)`
Reviewed by: Vikash (human)   Date: 2026-11-05   Reply received: `COMMITTED T-033`
