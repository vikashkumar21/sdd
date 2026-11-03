# Validation — T-021 sdd skill new command with file and text ingestion

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Passed
Roadmap ref: `ROADMAP.md` T-021

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/skill -run TestCreateSkillBasic` | Creates standard skill at `skills/<name>/SKILL.md` | PASS: created valid skill with frontmatter and sections | PASS |
| R-002 | test | `go test -v ./internal/skill -run TestCreateSkillInvalidName` | Rejects invalid skill name before writing | PASS: invalid regex/lengths rejected | PASS |
| R-003 | test | `go test -v ./internal/skill -run TestCreateSkillIngestion` | Ingests body from file and raw text | PASS: file and text ingested | PASS |
| R-004 | test | `go test -v ./internal/skill -run TestCreateSkillDescription` | Validates custom and default description | PASS: custom and default accepted, >1024 rejected | PASS |
| R-005 | test | `go test -v ./internal/skill -run TestCreateSkillInternal` | Creates skill under `.sdd/skills/<name>/` with `--internal` | PASS: created in `.sdd/skills/` | PASS |
| R-006 | test | `go test -v ./internal/skill -run TestCreateSkillCollisionRefusal` | Refuses to overwrite existing skill without `--force` | PASS: collision error returned | PASS |
| R-007 | test | `go test -v ./internal/skill -run TestCreateSkillForceBackup` | Backs up existing skill to `.sdd/backup/` with `--force` | PASS: backed up and overwritten | PASS |
| R-008 | test | `go test -v ./internal/skill -run TestCreateSkillRefinementBlock` | Scaffolds refinement block and standard sections for raw input | PASS: refinement markers and sections present | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | PASS (gofmt clean, vet clean, all tests passed) |
| `make check` | PASS (golangci-lint 0 issues, all tests passed) |
| `go run ./cmd/sdd check --strict` | PASS (0 errors, 0 warnings, 1 info note) |

## 3. Manual checks
- Verified `sdd skill new` via CLI integration tests in `internal/cli/cli_test.go` (`TestSkillNew`):
  - Verified argument validation and usage errors.
  - Verified creation of `skills/deploy-checker/SKILL.md` with valid YAML frontmatter.
  - Verified collision refusal when re-running without `--force`.
  - Verified `--force` creates `.sdd/backup/` timestamped backup and updates content.
  - Verified `--from-file` ingestion creates the refinement block (`<!-- sdd:skill-refine:begin -->`).

## 4. Negative and failure cases
- Tested invalid skill names (`CodeReview`, `code_review`, `-review`, `review-`, `code--review`, empty name) -> all rejected before file creation.
- Tested collision without `--force` -> returns error without modifying file.
- Tested mutually exclusive flags (`--from-file` and `--from-text` together) -> returns error.
- Tested non-existent `--from-file` path -> returns descriptive file read error.
- Tested oversized description (>1024 chars) -> returns error.

## 5. Determinism check
Ran `go test ./internal/skill ./internal/cli -count=1` twice; outputs identical: yes.
- Run 1: `ok github.com/vikashkumar21/sdd/internal/skill 0.116s`, `ok github.com/vikashkumar21/sdd/internal/cli 1.935s`
- Run 2: `ok github.com/vikashkumar21/sdd/internal/skill 0.114s`, `ok github.com/vikashkumar21/sdd/internal/cli 1.974s`

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
Suggested commit message: `feat(skill): add sdd skill new command with file and text ingestion (T-021)`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-021`
