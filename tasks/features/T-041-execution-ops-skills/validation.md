# Validation — T-041 Role-based agent skills: implementer, QA, and DevOps operations

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Completed

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/templates -run TestExtendedEmbeddedRoleSkills` && `go test -v ./internal/role -run "TestRoleConfigSchema\|TestCanonicalRoleNames"` | pass | PASS (TestExtendedEmbeddedRoleSkills, TestRoleConfigSchema, TestCanonicalRoleNames) | PASS |
| R-002 | test | `go test -v ./internal/templates -run TestExtendedEmbeddedRoleSkills` | pass | PASS (TestExtendedEmbeddedRoleSkills) | PASS |
| R-003 | test | `go test -v ./internal/role -run TestRoleConfigSchema` | pass | PASS (TestRoleConfigSchema: phase-b & operations verified) | PASS |
| R-004 | test | `go test -v ./internal/role -run TestCanonicalRoleNames` | pass | PASS (9 canonical roles in sorted order) | PASS |
| R-005 | test | `go test -v ./internal/scaffold -run TestInitExtendedRolesJson` && `go test -v ./internal/upgrade -run TestUpgradeExtendedRolesJson` | pass | PASS (TestInitExtendedRolesJson & TestUpgradeExtendedRolesJson) | PASS |
| R-006 | test | `go test -v ./internal/check -run TestRolesCheck` | pass | PASS (13/13 tests pass including extended unresolved phase mappings) | PASS |
| R-007 | test | `go test -v ./internal/cli -run "TestSkillExport_AllNineRoles\|TestSkillExport_ImplementerOps"` | pass | PASS (all 9 canonical skills and individual ops skills exported) | PASS |
| R-008 | test | `go test -v ./internal/cli -run TestPromptImplement_TurnkeyPrompt` | pass | PASS (turnkey prompt, QA/DevOps roles, JSON schema, invalid role/ID) | PASS |
| R-009 | test | `go test -v ./internal/cli -run TestPromptImplement_ConditionalPlanActivation` | pass | PASS (QA and DevOps detected from plan steps) | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make build` | pass: `go build -trimpath -ldflags ... -o bin/sdd ./cmd/sdd` |
| `make test` | pass: `go test ./... -count=1` all packages ok |
| `make lint` | pass: `golangci-lint run` 0 issues |
| `make verify` | pass: `go vet ./...` & `go test ./...` ok |
| `./bin/sdd check --strict` | pass: 0 error(s), 0 warning(s), 1 note(s) |

## 3. Manual checks
- `sdd skill export --roles` exports all 9 canonical skills to `skills/<role>/SKILL.md` (verified in `TestSkillExport_AllNineRoles` and manual CLI run).
- `sdd skill export --role implementer` exports only the `implementer` skill (verified in `TestSkillExport_ImplementerOps`).
- `sdd prompt implement T-041` outputs the Phase B implementation handover prompt including the `implementer` role skill instructions and Section 5 file boundary (verified).
- `sdd prompt implement T-041 --role qa` enriches context with the `qa` role skill instructions (verified).
- `sdd prompt implement T-041 --role devops` enriches context with the `devops` role skill instructions (verified).
- `sdd check --strict` validates the extended `.sdd/roles.json` and all 9 role skills (verified with 0 errors).

## 4. Negative and failure cases
- `sdd prompt implement T-041 --role unknown-role` fails with exit code 1 and descriptive error (verified in `TestPromptImplement_TurnkeyPrompt`).
- `.sdd/roles.json` with unresolvable `Implementer`, `QA`, or `DevOps` phase mappings fails in `sdd check` (verified in `TestRolesCheck_ExtendedPhaseMappingsUnresolved`).
- Attempting to resolve a skill with path traversal characters (`..`) is rejected in `role.ResolveSkill` (verified in `TestResolveSkillTwoTier` and `TestRolesCheck_InvalidSkillIdentifier`).

## 5. Determinism check
Ran tests twice (`go test ./internal/role ./internal/templates ./internal/cli ./internal/check -count=2`); outputs identical and repeatable across runs: pass.

## 6. Deviations, known limitations, and feedback-rule compliance
None.

Feedback-rule compliance (SKILLS.md, "Human feedback rules"):
- Rule F-001 (Resolve all review findings: Blockers, Warnings, and Minors): Phase C.2 Review Triad audit initiated.

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
Suggested commit message: `feat(skill): add implementer, QA, and DevOps role skills and prompt implement command`
Reviewed by: Vikash (human)   Date: 2026-11-15   Reply received: `COMMITTED T-041`
