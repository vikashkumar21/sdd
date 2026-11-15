# Validation — T-040 Role-based agent skills: core governance, planning, and review triad

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Verified

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/role -run "TestRoleConfigSchema\|TestCanonicalRoleNames"` | pass | PASS (0.00s) | VERIFIED |
| R-002 | test | `go test -v ./internal/templates -run TestEmbeddedRoleSkills` | pass | PASS (0.00s) | VERIFIED |
| R-003 | test | `go test -v ./internal/role -run TestRoleConfigSchema` | pass | PASS (0.00s) | VERIFIED |
| R-004 | test | `go test -v ./internal/scaffold -run TestInitRolesJson` && `go test -v ./internal/upgrade -run TestUpgradeInstallsMissingRolesJson` | pass | PASS (0.10s, 0.14s) | VERIFIED |
| R-005 | test | `go test -v ./internal/role -run TestResolveSkillTwoTier` | pass | PASS (0.00s) | VERIFIED |
| R-006 | test | `go test -v ./internal/check -run TestRolesCheck` | pass | PASS (0.11s, 9 subtests) | VERIFIED |
| R-007 | test | `go test -v ./internal/cli -run TestSkillExport_Roles` | pass | PASS (0.03s) | VERIFIED |
| R-008 | test | `go test -v ./internal/cli -run TestSkillExport_SingleRole` | pass | PASS (0.00s) | VERIFIED |
| R-009 | test | `go test -v ./internal/cli -run TestPromptPlanReview_RoleSkill` | pass | PASS (0.00s) | VERIFIED |
| R-010 | test | `go test -v ./internal/cli -run TestPromptReview_RoleSkillLens` | pass | PASS (0.00s) | VERIFIED |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make build` | pass |
| `make test` | pass (all packages ok) |
| `make lint` | pass (golangci-lint 0 issues) |
| `make verify` | pass (go vet clean, all tests pass) |
| `make check` | pass (golangci-lint + vet + tests clean) |
| `./bin/sdd check --strict` | pass (0 error(s), 0 warning(s), 1 note(s)) |

## 3. Manual checks
- `sdd skill export --roles` exports the 6 governance role skills to `skills/<role>/SKILL.md`: verified via CLI execution.
- `sdd skill export --role appsec-specialist` exports only the `appsec-specialist` skill: verified via CLI execution.
- `sdd prompt plan-review T-040` includes the `plan-reviewer` skill instructions in the generated prompt: verified via CLI execution.
- `sdd prompt review T-040 --lens spec` includes the `spec-auditor` skill instructions in the generated prompt: verified via CLI execution.
- `sdd check --strict` validates `.sdd/roles.json` and role skills: verified (0 errors).

## 4. Negative and failure cases
- Invalid `.sdd/roles.json` (malformed JSON, invalid schema version) fails in `sdd check`: verified in `TestRolesCheck_SyntaxError` and `TestRolesCheck_InvalidSchema`.
- `.sdd/roles.json` referencing a non-existent role skill fails in `sdd check`: verified in `TestRolesCheck_UnresolvedSkill`.
- Role skill with invalid `agentskills.io` frontmatter fails in `sdd check`: verified in `TestRolesCheck_InvalidSkillFrontmatter`.
- `sdd skill export --role non-existent` fails with descriptive error: verified in `TestSkillExport_InvalidRole`.

## 5. Determinism check
Ran `go test ./internal/role ./internal/templates ./internal/scaffold ./internal/check ./internal/cli -count=2`: pass, identical results across both runs.

## 6. Deviations, known limitations, and feedback-rule compliance
None.

Feedback-rule compliance (SKILLS.md, "Human feedback rules"):
- Rule F-001 (Resolve all review findings: Blockers, Warnings, and Minors): Review Triad audit initiated; all findings will be remediated before GATE 2 close.

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
Suggested commit message: `feat(skill): add role-based governance skills and roles manifest`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-040`
