---
status: active
task_id: T-048
last_reviewed: 2026-11-16
---
# Plan — T-048 Multi-agent coordinator contract and handover prompt template alignment

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved 2026-11-16
Roadmap ref: `ROADMAP.md` T-048

## 1. Approach summary
Update the embedded contract template (`internal/templates/data/base/contract.md.tmpl`) to codify the Task Coordinator operating model, explicitly detailing Phase A (`architect`, `plan-reviewer`), Phase B (`implementer`, `devops`), Phase C.1 (`qa`), Phase C.2 (`spec-auditor`, `appsec-specialist`, `pattern-auditor`), and Phase D (qualification/merge). Update `internal/templates/data/base/prompt.md.tmpl` to brief incoming agents that they act as Task Coordinator dispatching roles mapped in `.sdd/roles.json`. Bump `templates.Version` to `2.0.0` in `internal/templates/templates.go`, synchronize contract files (`AGENTS.md`, `CLAUDE.md`, `GEMINI.md`) and `.sdd/manifest.json`, verify `sdd upgrade` on older kits, and bump CLI version to `1.7.0`.

## 2. Decisions involved
| ID | Title | Options | Recommendation | Status |
|---|---|---|---|---|
| D-041 | Multi-agent coordinator contract and handover prompt template alignment | Option A (Codify Task Coordinator and multi-subagent dispatches in contract template Section 2 and 4, update handover prompt with coordinator mandate, bump template version to 2.0.0, and bump release version to 1.7.0) / Option B (Keep generic "What the agent does" wording in contract and only update prompt text) | Option A: Fully aligns embedded contract and handover prompt with autonomous multi-agent taxonomy (.sdd/roles.json), resolves sdd upgrade contract staleness in downstream projects, and advances template version to 2.0.0 per Rule 16. | **Approved** (pending GATE 1) |

## 3. Dependencies involved
- Standard library only (`D-007`). Zero external dependencies.

## 4. Feedback rules that apply
- **F-001:** Worker agents must resolve all findings (Blockers, Warnings, Minors) before GATE 2; any retained finding must be explicitly justified in `review.md`.

## 5. Step-by-step implementation plan

| # | Step | Files | Check |
|---|---|---|---|
| 1 | Update `internal/templates/data/base/contract.md.tmpl` Section 2 ("The workflow") and Section 4 ("Hard rules") to codify the Task Coordinator role, multi-subagent dispatches (`architect`, `plan-reviewer`, `implementer`, `qa`, `devops`, `spec-auditor`, `appsec-specialist`, `pattern-auditor`), and generalized `tasks/<typology>/` paths. Author `TestContractTemplateCoordinatorAndWorkflow` in `internal/templates/templates_test.go`. | `internal/templates/data/base/contract.md.tmpl`, `internal/templates/templates_test.go` | `go test -v ./internal/templates -run TestContractTemplateCoordinatorAndWorkflow` |
| 2 | Update `internal/templates/data/base/prompt.md.tmpl` to brief incoming handover agents as Task Coordinators, dispatching roles from `.sdd/roles.json`, updating paths from `features/` to `tasks/<typology>/`. Author test assertion in `internal/cli/cli_test.go`. | `internal/templates/data/base/prompt.md.tmpl`, `internal/cli/cli_test.go` | `go test -v ./internal/cli -run TestPromptCoordinatorBriefing` |
| 3 | Bump `templates.Version` to `2.0.0` in `internal/templates/templates.go`, update template version assertion in `internal/templates/templates_test.go`, and update `.sdd/manifest.json`. | `internal/templates/templates.go`, `internal/templates/templates_test.go`, `.sdd/manifest.json` | `go test -v ./internal/templates -run TestVersion` |
| 4 | Synchronize `AGENTS.md`, `CLAUDE.md`, and `GEMINI.md` in 100% lockstep with the updated contract text and header marker `<!-- sdd:begin contract templates=2.0.0 -->`. | `AGENTS.md`, `CLAUDE.md`, `GEMINI.md` | `go test -v ./internal/check -run "TestDetectsProblems/agent_contract_drift" && ./bin/sdd check --strict` |
| 5 | Add upgrade test in `internal/upgrade/upgrade_test.go` verifying that `sdd upgrade` upgrades a kit with `templates=1.9.0` contracts to `2.0.0`. | `internal/upgrade/upgrade_test.go` | `go test -v ./internal/upgrade` |
| 6 | Bump CLI release version to `1.7.0` in `internal/version/version.go`, `Makefile`, and `README.md`. | `internal/version/version.go`, `Makefile`, `README.md` | `make build && ./bin/sdd version` |
| 7 | Run verification suite: `make verify`, `make lint`, `./bin/sdd check --strict`, and Rule 16 fresh kit generation test in temporary directory. | (all) | `make verify && make lint && ./bin/sdd check --strict` |

## 6. Files to touch
| File | Action | Purpose |
|---|---|---|
| `internal/templates/data/base/contract.md.tmpl` | modify | Update Section 2 and 4 with Task Coordinator and role dispatches |
| `internal/templates/data/base/prompt.md.tmpl` | modify | Update handover prompt template with Task Coordinator mandate |
| `internal/templates/templates.go` | modify | Bump Version to 2.0.0 |
| `internal/templates/templates_test.go` | modify | Add contract template content assertions and update version assertion to 2.0.0 |
| `internal/cli/cli_test.go` | modify | Add assertion verifying prompt briefs agent as Task Coordinator |
| `internal/cli/prompt_context_test.go` | modify | Update expected prompt path assertion from legacy features/ to tasks/<typology>/ |
| `internal/upgrade/upgrade_test.go` | modify | Add test for upgrading 1.9.0 kit contracts to 2.0.0 |
| `internal/version/version.go` | modify | Bump CLI Version to 1.7.0 |
| `Makefile` | modify | Bump VERSION to 1.7.0 |
| `README.md` | modify | Update version status line to v1.7.0 (templates 2.0.0) |
| `ROADMAP.md` | modify | Update status to Awaiting review in Phase D, Done upon COMMITTED T-048; record in change log |
| `TECH_STACK.md` | modify | Record D-041 as Approved, add verified facts entry in §7 |
| `AGENTS.md` | modify | Synchronize contract to templates=2.0.0 |
| `CLAUDE.md` | modify | Synchronize contract to templates=2.0.0 |
| `GEMINI.md` | modify | Synchronize contract to templates=2.0.0 |
| `.sdd/manifest.json` | modify | Update templates version to 2.0.0 |

## 7. Living docs to update
- `ROADMAP.md`: update status to `Awaiting review` in Phase D, `Done` upon `COMMITTED T-048`; record in change log.
- `TECH_STACK.md`: record `D-041` as Approved, add verified facts entry in §7.
- `README.md`: update version status line to `v1.7.0 (templates 2.0.0)`.

## 8. Risks, mitigations, rollback
- **Risk:** Existing projects running `sdd upgrade` might receive proposals if they edited contract text outside managed blocks.
  - **Mitigation:** SDD's managed block architecture (`block.Locate`) ensures that only text between `<!-- sdd:begin contract ... -->` and `<!-- sdd:end contract -->` is merged; human edits outside the block are 100% preserved.
- **Rollback:** `git checkout` reverts template modifications.

## 9. Verification commands
```bash
make verify
make lint
./bin/sdd check --strict
tmpdir=$(mktemp -d) && ./bin/sdd init --dir "$tmpdir" --name testkit --lang go --yes >/dev/null && ./bin/sdd check --dir "$tmpdir" && rm -rf "$tmpdir"
```

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-16   Phrase received: `APPROVED T-048`
