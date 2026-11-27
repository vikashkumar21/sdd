---
status: completed
---
# Plan — T-065 Align MISSION.md scope and contract git-write rules

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-27)
Roadmap ref: `ROADMAP.md` T-065

## 1. Approach summary
Update `MISSION.md` §4 so delivered v1.1–v1.11 capabilities are in scope and boundaries accurately distinguish offline kit operations from optional `sdd self-update`. Align `GATE 2` and `Hard rule 2` with `Hard rule 13` in `internal/templates/data/base/contract.md.tmpl` and the three root contracts (`AGENTS.md`, `CLAUDE.md`, `GEMINI.md`), bump `templates.Version` to `2.12.0` (updating `TestVersion` in `internal/templates/templates_test.go` and `.sdd/manifest.json`), and verify parity, fresh-kit regeneration (Hard Rule 16), `make verify`, and `make lint`.

## 2. Decisions (new or affected)
None. Relies on `D-007` and `D-029` (Approved), plus explicit human approval of `MISSION.md` and contract edits at GATE 1.

## 3. Design
- **`MISSION.md` §4:** Rename `**In scope (v1)**` to `**In scope**`; move `sdd upgrade` (`T-003`), `sdd adopt` (`T-004`), Windows support (`T-018`), and per-agent permission settings (`T-013`) plus shipped post-v1 kit capabilities (task state, worktree isolation, review triad, `sdd passport`, transcript token ingestion) into `**In scope**`. Under `**Out of scope / boundaries**`, retain codebase understanding (agent under gates, not binary) and AI/network exclusions (offline kit operations; optional `sdd self-update` uses HTTPS to GitHub Releases) plus Parking lot reference. Update `Last updated: 2026-11-26`.
- **Contract template & root contracts (`internal/templates/data/base/contract.md.tmpl`, `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`):**
  - `GATE 2` bullet 3:
    `- The agent never runs \`git add\`, \`commit\`, \`push\`, \`tag\`, \`reset\`, \`rebase\`, \`stash\`, \`checkout --\`, or any other command that writes git state on \`main\` (or outside an isolated task branch). Read-only git (\`status\`, \`diff\`, \`log\`, \`show\`) is allowed; workers may commit only on their isolated task branch per Rule 13, and merges to \`main\` require human approval via \`sdd task merge\` or human commit.`
  - `Hard rule 2`:
    `2. **Never commit, push, or merge directly on \`main\`.** Suggest; the human commits or approves \`sdd task merge\`. (Workers may commit only on their isolated task branch per Rule 13.)`
  - Contract block header version: `<!-- sdd:begin contract templates=2.12.0 -->`
- **Template version (`internal/templates/templates.go`, `internal/templates/templates_test.go`, `.sdd/manifest.json`):** `2.11.0` → `2.12.0`.

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Update `MISSION.md` §4 scope and `Last updated` date | `MISSION.md` | `make build && ./bin/sdd check --strict` |
| 2 | Align `GATE 2` and `Hard rule 2` in `base/contract.md.tmpl`, bump `templates.Version` to `2.12.0`, update `TestVersion` | `internal/templates/data/base/contract.md.tmpl`, `internal/templates/templates.go`, `internal/templates/templates_test.go` | `go test -v ./internal/templates` |
| 3 | Mirror contract edits and `2.12.0` block header across `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, and update `.sdd/manifest.json` | `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `.sdd/manifest.json` | `go test -v ./internal/check -run TestCheckContractsMatch` |
| 4 | Fix 4 pre-existing lint issues in `internal/hook/hook.go` (human-approved delta), run full verification (`make build`, `make verify`, `make lint`, `./bin/sdd check --strict`), and update `TECH_STACK.md`; transition `ROADMAP.md` to `Awaiting review` in Phase D after C.1/C.2 sign-off | `internal/hook/hook.go`, `TECH_STACK.md`, `ROADMAP.md`, `tasks/features/T-065-mission-and-contract-alignment/validation.md` | `make verify && make lint && make build && ./bin/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `MISSION.md` | modify | align §4 Scope with shipped features |
| `internal/templates/data/base/contract.md.tmpl` | modify | align GATE 2 & Rule 2 with Rule 13 |
| `internal/templates/templates.go` | modify | bump `Version` to `2.12.0` (Rule 16) |
| `internal/templates/templates_test.go` | modify | update `TestVersion` expectation to `2.12.0` |
| `AGENTS.md` | modify | mirror contract & `2.12.0` header |
| `CLAUDE.md` | modify | mirror contract & `2.12.0` header |
| `GEMINI.md` | modify | mirror contract & `2.12.0` header |
| `.sdd/manifest.json` | modify | update `templates.version` to `2.12.0` |
| `internal/hook/hook.go` | modify | fix 4 pre-existing lint issues (`QF1003`, `errcheck`) so repo-wide `make lint` passes (human-approved delta) |
| `ROADMAP.md` | modify | task status, changelog, and Parking lot note |
| `TECH_STACK.md` | modify | verified-facts log for `templates.Version = 2.12.0` |
| `tasks/features/T-065-mission-and-contract-alignment/requirements.md` | modify | task requirements |
| `tasks/features/T-065-mission-and-contract-alignment/plan.md` | modify | task plan |
| `tasks/features/T-065-mission-and-contract-alignment/validation.md` | modify | validation evidence |
| `tasks/features/T-065-mission-and-contract-alignment/review.md` | modify | review ledger and sign-off |
| `.sdd/state/T-065/state.json` | create | task token telemetry |
| `.sdd/state/T-065/events.jsonl` | create | task event journal |

## 6. Dependencies
None.

## 7. Living docs to update

| File | Change |
|---|---|
| `MISSION.md` | §4 Scope aligned with delivered capabilities |
| `AGENTS.md`, `CLAUDE.md`, `GEMINI.md` | GATE 2 & Rule 2 aligned with Rule 13; `templates=2.12.0` |
| `ROADMAP.md` | T-065 status & changelog; Parking lot note for `prompt.md.tmpl` & `sdd-protocol` git-rule summary alignment |
| `TECH_STACK.md` | verified fact for template version `2.12.0` |

## 8. Risks, mitigations, rollback
- Risk: `TestCheckContractsMatch` or `TestContractTemplateCoordinatorAndWorkflow` fails if a contract file drifts → Mitigation: mirror exact text across all 3 root contracts and `base/contract.md.tmpl`, and run `TestCheckContractsMatch`.
- Rollback: revert the merge commit on `main`.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- `F-001`: resolve all review findings before GATE 2.
- `F-002`: no personal paths.
- `F-003`: record measured session tokens via `sdd record-tokens T-065` at Phase D.

## 10. Approval record
Approved by: Vikash   Date: 2026-11-27   Phrase received: `APPROVED T-065`
