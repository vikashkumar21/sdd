# Validation — T-020 agentskills.io parser, validator, and sdd check integration

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Passed
Roadmap ref: `ROADMAP.md` T-020

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/skill -run TestParseSkill` | Extracts YAML frontmatter without external dependencies | PASS: parsed standard and block scalars | PASS |
| R-002 | test | `go test -v ./internal/skill -run TestValidateSkillName` | Validates regex `^[a-z0-9]+(?:-[a-z0-9]+)*$` and max length 64 | PASS: valid names accepted, invalid rejected | PASS |
| R-003 | test | `go test -v ./internal/skill -run TestValidateSkillDirectoryMatch` | Rejects skill name that mismatches parent folder | PASS: mismatch rejected | PASS |
| R-004 | test | `go test -v ./internal/skill -run TestValidateSkillDescription` | Rejects empty description or description > 1024 chars | PASS: empty and >1024 rejected | PASS |
| R-005 | test | `go test -v ./internal/skill -run TestValidateSkillCompatibility` | Rejects compatibility field > 500 chars | PASS: >500 rejected | PASS |
| R-006 | test | `go test -v ./internal/check -run TestCheckSkills` | sdd check flags invalid skills and passes valid skills | PASS: valid passed, mismatch and syntax errors flagged | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | PASS (gofmt clean, vet clean, all packages passed) |
| `make check` | PASS (golangci-lint 0 issues, all tests passed) |
| `go run ./cmd/sdd check --strict` | PASS (0 errors, 0 warnings, 1 info note) |

## 3. Manual checks
- Verified `skills/` and `.sdd/skills/` scanning with unit test in `internal/check/check_test.go` (`TestCheckSkills`).
- Verified that valid skills produce 0 errors and malformed skills generate actionable check findings with accurate line numbers and field names.

## 4. Negative and failure cases
- Tested missing frontmatter delimiters (`---`) -> rejected with clear error.
- Tested invalid characters in name (uppercase, underscores, consecutive hyphens, leading/trailing hyphens) -> rejected.
- Tested name length > 64 chars -> rejected.
- Tested mismatch between `name` and parent folder name -> rejected.
- Tested missing or empty `description` -> rejected.
- Tested `description` > 1024 chars -> rejected.
- Tested `compatibility` > 500 chars -> rejected.

## 5. Determinism check
Ran `go test ./internal/skill ./internal/check -count=1` twice; outputs identical: yes.
- Run 1: `ok github.com/vikashkumar21/sdd/internal/skill 0.066s`, `ok github.com/vikashkumar21/sdd/internal/check 1.804s`
- Run 2: `ok github.com/vikashkumar21/sdd/internal/skill 0.063s`, `ok github.com/vikashkumar21/sdd/internal/check 1.898s`

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
Suggested commit message: `feat(skill): add agentskills.io parser, validator, and sdd check integration (T-020)`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-020`
