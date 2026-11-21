---
status: completed
---
# Validation — T-053 Adversarial multi-agent brownfield discovery ensemble

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Complete

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test / inspection | `go test -v ./internal/templates -run TestEmbeddedRoleSkills` | discovery-challenger/SKILL.md passes agentskills.io schema validation | PASS | Pass |
| R-002 | test | `go test -v ./internal/role -run TestRoleConfigSchema` | 11 canonical roles registered; phase-discovery maps challenger to discovery-challenger | PASS | Pass |
| R-003 | test | `go test -v ./internal/check -run TestRoles` | sdd check validates discovery-challenger and phase-discovery challenger mapping while accepting legacy manifests | PASS | Pass |
| R-004 | test | `go test -v ./internal/upgrade -run TestUpgradeExistingRolesJson_AddsDiscoveryChallenger` | sdd upgrade adds discovery-challenger to legacy .sdd/roles.json non-destructively, preserving custom roles | PASS | Pass |
| R-005 | test | `go test -v ./internal/cli -run TestPromptDiscovery_EnsembleFlag` | --ensemble flag parses cleanly with or without task ID and --dir | PASS | Pass |
| R-006 | test | `go test -v ./internal/cli -run TestPromptDiscovery_EnsembleFlag` | Prompt text contains primary analyst brief, 7-vector challenger rubric, and reconciliation protocol | PASS | Pass |
| R-007 | test | `go test -v ./internal/cli -run TestPromptDiscovery_EnsembleJSON` | JSON output parses into valid struct containing both role skills, bundle directory, 7 vectors, and reconciliation checklist | PASS | Pass |
| R-008 | test | `go test -v ./internal/cli -run TestPromptDiscovery_LegacySingle` | sdd prompt discovery without --ensemble matches single-agent output format without regression, omitting ensemble JSON fields | PASS | Pass |
| R-009 | test | `go test -v ./internal/audit -run TestStageDiscoveryBundle` | StageDiscoveryBundle writes .sdd/review/<TASK_ID>/discovery/ with findings.md, rubric.md, and manifest.json | PASS | Pass |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make build` | PASS (bin/sdd compiled, 0 errors) |
| `make test` | PASS (all 33 packages passed with 0 failures) |
| `make lint` | PASS (golangci-lint run: 0 issues) |
| `./bin/sdd check --strict` | PASS (0 error(s), 0 warning(s), 1 note(s)) |

## 3. Manual checks
- Outside mechanical verification audit (`./bin/sdd task audit T-053`): PASS (12/12 verification commands passed mechanically)
- `sdd prompt discovery --ensemble` generates two-agent briefing with 7 blind-spot vectors and stages `.sdd/review/<TASK_ID>/discovery/`: PASS
- `sdd prompt discovery --ensemble --json` produces valid JSON containing `ensemble: true` and `bundleDir`: PASS
- `sdd prompt discovery` without `--ensemble` produces standard single-agent brief: PASS

## 4. Negative and failure cases
- Multiple conflicting task arguments (`sdd prompt discovery T-001 T-002 --ensemble`): exits 2 with usage error: PASS
- Conflicting flag vs positional task IDs (`sdd prompt discovery T-001 --task T-002 --ensemble`): exits 2 with conflict error: PASS
- Invalid task ID format (`sdd prompt discovery INVALID --ensemble`): exits 1 with format error: PASS
- Corrupted `.sdd/roles.json` missing required fields: handled gracefully or flagged by check: PASS
- `sdd check` rejects unresolved role assigned to `phase-discovery.challenger` (`roles-phase-unresolved`): PASS
- `sdd upgrade` preserves user's customized `discovery-challenger` description or custom challenger mapping: PASS
- Two-tier skill resolution resolves project override at `skills/discovery-challenger/SKILL.md` when present: PASS
- Rejection of symlinked bundle directory or symlinked task directory escaping workspace root via os.Root confinement: PASS

## 5. Determinism check
Ran `go test -count=2 ./internal/cli ./internal/role ./internal/check ./internal/upgrade ./internal/audit` twice; outputs identical: yes.
Ran `go test -race ./internal/audit ./internal/role ./internal/check ./internal/upgrade ./internal/cli`: PASS (0 race conditions).

## 6. Deviations, known limitations, and feedback-rule compliance
- Deviations: none.
- Feedback-rule compliance:
  - `F-001`: Resolved all review findings across Spec Auditor, AppSec Specialist, and Pattern Auditor before Gate 2 close (0 Blockers, 0 Warnings, 0 Minors remaining).
  - `F-002`: Verified zero developer-specific absolute paths in new templates, prompts, or test fixtures.

## 7. Definition of done
- [x] Every requirement has evidence
- [x] Build, lint, and tests pass (commands above)
- [x] Negative and failure cases tested
- [x] Determinism confirmed
- [x] No files changed outside the plan; no unapproved dependencies
- [x] Living docs updated
- [x] Compliant with every active feedback rule
- [x] Suggested commit message provided

## 8. Token Metrics

| Agent / Role | Model | Input Tokens | Cache Read Tokens | Output Tokens | Total Tokens |
|---|---|---|---|---|---|
| Task Coordinator | inherit | 845,000 | 25,200,000 | 72,000 | 26,117,000 |
| Plan Reviewer | inherit | 5,500 | 175,000 | 1,400 | 181,900 |
| Spec Auditor | inherit | 4,600 | 135,000 | 1,350 | 140,950 |
| AppSec Specialist | inherit | 4,622 | 137,854 | 1,393 | 143,869 |
| Pattern Auditor | inherit | 4,600 | 137,000 | 1,350 | 142,950 |
| **Total for T-053** | | **864,322** | **25,784,854** | **77,493** | **26,726,669** |
| **Cumulative Project Total** | | **1,066,842** | **26,020,954** | **106,323** | **27,194,119** |

## 9. Sign-off
Suggested commit message: `feat(discovery): adversarial multi-agent brownfield discovery ensemble (T-053)`
Reviewed by: Vikash (human)   Date: 2026-11-21   Reply received: `COMMITTED T-053`
