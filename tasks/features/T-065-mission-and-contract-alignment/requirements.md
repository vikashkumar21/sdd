---
status: completed
---
# Requirements — T-065 Align MISSION.md scope and contract git-write rules

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-27)
Roadmap ref: `ROADMAP.md` T-065

## 1. Goal
Resolve the two documented contradictions between `MISSION.md` vs. `README.md`/`ROADMAP.md` and between `GATE 2`/`Rule 2` vs. `Rule 13` in the agent contracts, and bump `templates.Version` to `2.12.0`.

## 2. Context and references
- Mission / principles touched: §1 Determinism, §6 Living documents (requires explicit human approval at GATE 1 to update `MISSION.md` and agent contracts).
- Decisions this relies on: `D-007` (stdlib only), `D-029` (two-tier git governance).
- Depends on tasks: `T-064` (Done).

## 3. Scope
**In scope**
- Update `MISSION.md` §4 (`Scope`):
  - Rename `**In scope (v1)**` to `**In scope**`.
  - Move delivered capabilities (`sdd upgrade` `T-003`, `sdd adopt` `T-004`, Windows support `T-018`, per-agent permission settings `T-013`) out of `Out of scope for now` into `**In scope**`, along with shipped post-v1 kit capabilities (task state, worktree isolation, review triad, `sdd passport`, transcript token ingestion).
  - Keep `Understanding a codebase (performed by the agent under the gates, not by the sdd binary itself)` and `Calling any AI model (sdd kit operations are offline and deterministic; only optional sdd self-update uses HTTPS to fetch GitHub Releases)` under `**Out of scope / boundaries**` (referencing the `ROADMAP.md` Parking lot). Update `Last updated: 2026-11-26`.
- Align `internal/templates/data/base/contract.md.tmpl`, `AGENTS.md`, `CLAUDE.md`, and `GEMINI.md` at two places:
  - `GATE 2` bullet 3: clarify that the agent never runs `git add`, `commit`, `push`, `tag`, `reset`, `rebase`, `stash`, `checkout --`, or any other command that writes git state on `main` (or outside an isolated task branch), while read-only git is allowed and workers may commit only on their isolated task branch per Rule 13 (merges to `main` require human approval via `sdd task merge` or human commit).
  - `Hard rule 2`: clarify `"**Never commit, push, or merge directly on \`main\`.** Suggest; the human commits or approves \`sdd task merge\`. (Workers may commit only on their isolated task branch per Rule 13.)"`
- Bump `templates.Version` from `2.11.0` to `2.12.0` in `internal/templates/templates.go`, update `TestVersion` in `internal/templates/templates_test.go`, and update `.sdd/manifest.json` (`templates.version`) and contract headers (`<!-- sdd:begin contract templates=2.12.0 -->` in `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`).
- Verify fresh kit generation (`sdd init --yes` + `sdd check`), `make verify`, `make lint` (including fixing 4 pre-existing lint issues in `internal/hook/hook.go` per human-approved plan delta), and `./bin/sdd check --strict`.

**Out of scope**
- Rewriting `MISSION.md` principles (§5) or non-contradictory contract sections.
- Modifying `prompt.md.tmpl` or `skills/` summaries of the git rule (tracked in `ROADMAP.md` Parking lot for a later template bump if desired).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL update `MISSION.md` §4 (`Scope`) so `**In scope**` includes `sdd upgrade`, `sdd adopt`, Windows support, per-agent permissions, and shipped post-v1 kit capabilities, and `**Out of scope / boundaries**` retains only non-binary/AI exclusions (qualifying `sdd self-update` network access) and the Parking lot reference. | MUST | Inspection + `make build && ./bin/sdd check --strict` |
| R-002 | The system SHALL align `GATE 2` and `Hard rule 2` with `Hard rule 13` in `internal/templates/data/base/contract.md.tmpl`, `AGENTS.md`, `CLAUDE.md`, and `GEMINI.md` so isolated task-branch worktree commits are explicitly permitted while writing, pushing, or merging directly on `main` remains strictly forbidden. | MUST | `go test -v ./internal/check -run TestCheckContractsMatch` |
| R-003 | The system SHALL bump `templates.Version` to `2.12.0` in `internal/templates/templates.go`, update `TestVersion`, update root contract block headers and `.sdd/manifest.json`, and verify fresh-kit generation per Hard Rule 16. | MUST | `go test -v ./internal/templates` and `go test -v ./internal/check -run TestFreshKitHasNoErrors` and `go test -v ./internal/cli -run TestInitYesGreenfieldThenCheckPasses` |
| R-004 | The system SHALL pass `make verify`, `make lint`, `make build && ./bin/sdd check --strict` (0 errors, 0 warnings), and `git diff -- go.mod` (0 new dependencies, `D-007`). | MUST | `make verify && make lint && make build && ./bin/sdd check --strict && git diff -- go.mod` |

## 5. Non-functional requirements
- **Contract parity (§9):** `CLAUDE.md`, `GEMINI.md`, `AGENTS.md`, and rendered `base/contract.md.tmpl` remain byte-identical inside their managed blocks.

## 6. Constraints and assumptions
- Editing `MISSION.md` and the agent contracts requires explicit human approval (`APPROVED T-065`).

## 7. Open questions
None.

## 8. Acceptance criteria
- Given `MISSION.md` and `README.md`, neither contradicts `ROADMAP.md` on `upgrade`, `adopt`, `permissions`, Windows support, or `sdd self-update`.
- Given `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, and `base/contract.md.tmpl`, `GATE 2`, `Rule 2`, and `Rule 13` agree without contradiction, and `TestCheckContractsMatch` passes.
- Given `templates.Version == "2.12.0"`, regenerating a kit in a temp dir and running `sdd check` passes with 0 errors.
