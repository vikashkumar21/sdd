# Validation — T-022 sdd skill list and progressive disclosure handover prompt

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Passed
Roadmap ref: `ROADMAP.md` T-022

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/cli -run TestSkillList` | `sdd skill list` enumerates skills across `skills/` and `.sdd/skills/` | PASS: both project and internal skills discovered | PASS |
| R-002 | test | `go test -v ./internal/cli -run TestSkillList` | Displays aligned columns: name, scope, status, description | PASS: aligned table output with headers and status | PASS |
| R-003 | test | `go test -v ./internal/cli -run TestSkillList` | `sdd skill list --json` outputs structured JSON | PASS: valid unmarshalled JSON array | PASS |
| R-004 | test | `go test -v ./internal/cli -run TestPromptWithSkills` | `sdd prompt` includes `Available Agent Skills` catalog | PASS: progressive disclosure table rendered in prompt | PASS |
| R-005 | test | `go test -v ./internal/cli -run TestPromptWithRefinementSkills` | `sdd prompt` outputs refinement alert for unrefined skills | PASS: alert for `<!-- sdd:skill-refine:begin -->` rendered | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | PASS (gofmt clean, vet clean, all tests passed) |
| `make check` | PASS (golangci-lint 0 issues, all tests passed) |
| `go run ./cmd/sdd check --strict` | PASS (0 errors, 0 warnings, 1 info note) |

## 3. Manual checks
- Verified `sdd skill list` on empty project outputs `No skills found.` and exits 0.
- Verified deterministic ordering: skills sorted alphabetically by name.
- Verified `sdd prompt <agent>` progressive disclosure table outputs only name and trigger description (~50 tokens total).
- Verified `sdd prompt <agent>` refinement note triggers when raw ingested notes are present.

## 4. Negative and failure cases
- Tested unexpected arguments to `sdd skill list` (e.g. `sdd skill list extra`) -> exits 2 with usage message.
- Tested `sdd skill unknown` -> exits 2 with unknown command error.

## 5. Determinism check
Ran `go test ./internal/cli -count=1` twice; outputs identical: yes.
- Run 1: `ok github.com/vikashkumar21/sdd/internal/cli 2.052s`
- Run 2: `ok github.com/vikashkumar21/sdd/internal/cli 2.071s`

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
Suggested commit message: `feat(skill): add sdd skill list and progressive disclosure handover prompt (T-022)`
Reviewed by: Vikash   Date: 2026-11-03   Reply received: COMMITTED T-022
