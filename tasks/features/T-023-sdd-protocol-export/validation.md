# Validation — T-023 SDD protocol export as native agent skill

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Passed
Roadmap ref: `ROADMAP.md` T-023

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/skill -run TestExportProtocolSkillBasic` | Exports valid `sdd-protocol/SKILL.md` into project `skills/` | PASS: exported skill with frontmatter and sections | PASS |
| R-002 | test | `go test -v ./internal/skill -run TestExportProtocolSkillValid` | Validates that canonical skill satisfies `agentskills.io` standard | PASS: passes `skill.Validate` with 0 errors | PASS |
| R-003 | test | `go test -v ./internal/skill -run TestExportProtocolSkillTarget` | Exports skill to arbitrary `--target` directory | PASS: written to target directory | PASS |
| R-004 | test | `go test -v ./internal/skill -run TestExportProtocolSkillGlobal` | Exports skill to `--global` user directory | PASS: written to global agent skill directory | PASS |
| R-005 | test | `go test -v ./internal/skill -run TestExportProtocolSkillCollisionRefusal` | Refuses to overwrite existing export without `--force` | PASS: collision error returned | PASS |
| R-006 | test | `go test -v ./internal/skill -run TestExportProtocolSkillForceBackup` | Creates atomic backup in `.sdd/backup/` when overwriting | PASS: backed up and overwritten | PASS |
| R-007 | test / inspection | `go test -v ./internal/cli -run TestSkillExport` | CLI end-to-end command exports skill cleanly for SDKs | PASS: CLI exports cleanly and safely | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | PASS (gofmt clean, vet clean, all tests passed) |
| `make check` | PASS (golangci-lint 0 issues, all tests passed) |
| `go run ./cmd/sdd check --strict` | PASS (0 errors, 0 warnings, 1 info note) |

## 3. Manual checks
- Verified `sdd skill export` generates `skills/sdd-protocol/SKILL.md` conforming to `agentskills.io`.
- Verified `sdd skill export --target <dir>` enables programmatic Agent SDK (Python/TypeScript) applications to mount the skill directory.
- Verified duplicate collision safety: re-running `sdd skill export` without `--force` safely refuses.

## 4. Negative and failure cases
- Attempted export over existing skill without `--force` -> returns collision error.
- Attempted export with unexpected extra positional arguments -> exits 2.
- Verified `--global` behavior when home directory is simulated.

## 5. Determinism check
Ran `go test ./internal/skill ./internal/cli -count=1` twice; outputs identical: yes.
- Run 1: `ok github.com/vikashkumar21/sdd/internal/skill 0.162s`, `ok github.com/vikashkumar21/sdd/internal/cli 2.100s`
- Run 2: `ok github.com/vikashkumar21/sdd/internal/skill 0.150s`, `ok github.com/vikashkumar21/sdd/internal/cli 2.136s`

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
Suggested commit message: `feat(skill): add SDD protocol export as native agent skill (T-023)`
Reviewed by: Vikash   Date: 2026-11-03   Reply received: COMMITTED T-023
