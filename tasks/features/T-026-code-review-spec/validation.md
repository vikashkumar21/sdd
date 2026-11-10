# Validation — T-026 Official code-review skill and task review template (review.md)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Passed
Roadmap ref: `ROADMAP.md` T-026

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/scaffold -run TestGreenfieldCreatesTheWholeKit` | Embedded templates include `review.md` and scaffold creates it | PASS: features/_templates/review.md generated | PASS |
| R-002 | inspection | Manual review of `features/_templates/review.md` | Contains 3-tier risk register, scope checklist, feedback synthesis | PASS: standard sections and rubric verified | PASS |
| R-003 | test | `go test -v ./internal/feature -run TestNewCreatesFolderFromProjectTemplates` | `sdd feature new` creates `review.md` | PASS: 4 files created including review.md | PASS |
| R-004 | test | `go test -v ./internal/skill -run TestCanonicalCodeReviewSkillValid` | Canonical skill satisfies agentskills.io standard | PASS: frontmatter valid, name=code-review, description and rubric intact | PASS |
| R-005 | test | `go test -v ./internal/cli -run TestSkillExport` | `sdd skill export` defaults to all, exports code-review and sdd-protocol | PASS: both skills exported with verification | PASS |
| R-006 | test | `go test -v ./internal/check -run TestCheckReviewDocument` | `sdd check` flags unresolved 🔴 Blocker when task is Awaiting review or Done | PASS: Error generated for unresolved blocker | PASS |
| R-007 | test | `go test -v ./internal/check -run TestCheckReviewDocument` | `sdd check` validates review status header | PASS: Warning generated for invalid review status | PASS |
| R-008 | test | `go test -v ./internal/upgrade -run TestUpgradeCreatesMissingReviewTemplate` | `sdd upgrade` creates `features/_templates/review.md` on legacy kits | PASS: ActionCreate for review.md on upgrade | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | PASS (gofmt clean, vet clean, all tests pass) |
| `make lint` | PASS (golangci-lint clean, 0 issues) |
| `make check` | PASS (full check passes) |
| `go run ./cmd/sdd check --strict` | PASS (0 errors, 0 warnings, 1 info note) |
| Rule 14 test (fresh kit generation in temp directory) | PASS (13 files created including review.md, sdd check 0 errors) |

## 3. Manual checks
- Verified `sdd feature new` scaffolds `review.md`.
- Verified `sdd skill export` exports both canonical skills (`sdd-protocol` and `code-review`).
- Verified `sdd prompt gemini` displays both skills under available agent skills table.

## 4. Negative and failure cases
- Tested legacy kit without `features/_templates/review.md` continues to scaffold 3 files without failing (`TestNewWithLegacy3TemplatesOnly`).
- Tested `review.md` with unknown status produces warning (`TestCheckReviewDocument`).
- Tested `review.md` with unresolved 🔴 Blocker fails check (`TestCheckReviewDocument`).

## 5. Determinism check
Ran tests twice; outputs identical:
- `go test ./internal/skill ./internal/feature ./internal/check ./internal/upgrade -count=1` (Run 1: PASS)
- `go test ./internal/skill ./internal/feature ./internal/check ./internal/upgrade -count=1` (Run 2: PASS)

## 6. Deviations, known limitations, and feedback-rule compliance
None.

Feedback-rule compliance (SKILLS.md, "Human feedback rules"): none active.

## 7. Definition of done
- [x] Every requirement has evidence
- [x] Build, lint, and tests pass (commands above)
- [x] Negative and failure cases tested
- [x] Determinism confirmed
- [x] No files changed outside the plan; no unapproved dependencies
- [x] Living docs updated
- [x] Compliant with every active feedback rule
- [x] Suggested commit message provided

## 8. Sign-off
Suggested commit message: `feat(review): add official code-review skill and review.md task template (T-026)`
Reviewed by: Vikash   Date: 2026-11-05   Reply received: `COMMITTED T-026`
