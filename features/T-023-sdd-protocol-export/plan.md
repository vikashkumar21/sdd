# Plan — T-023 SDD protocol export as native agent skill

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-03)
Roadmap ref: `ROADMAP.md` T-023

## 1. Approach summary
1. Implement canonical `sdd-protocol` skill and export engine in `internal/skill/export.go`:
   - Define embedded canonical `sdd-protocol` content adhering strictly to `agentskills.io` frontmatter and progressive disclosure sections.
   - Implement `ExportProtocolSkill(opts ExportOptions) (ExportResult, error)`:
     - Target path resolution: project `skills/` by default, custom `--target <dir>`, or global home directory (`--global`).
     - Collision detection: refuses overwrite without `--force`.
     - Atomic backup on `--force` into `.sdd/backup/`.
     - Atomic write via `fsutil.WriteAtomic`.
     - Self-validation using `skill.Parse` and `skill.Validate`.
2. Add unit tests in `internal/skill/export_test.go`:
   - Test project export, custom target export, global export, collision refusal, force backup, and validity.
3. Wire `sdd skill export` subcommand in `internal/cli/skill.go` and `internal/cli/cli.go`:
   - Handle flags `--target`, `--global`, `--force`, `--dir`.
   - Route `export` subaction.
4. Add CLI end-to-end tests in `internal/cli/cli_test.go`.
5. Update living docs:
   - `README.md`: Document `sdd skill export` and programmatic Agent SDK integration patterns.
   - `SKILLS.md`: Add programmatic Agent SDK skill ingestion guidance.
   - `ROADMAP.md`: Track T-023 status across gates.
6. Run full verification suite (`make check`, `sdd check --strict`).

## 2. Decisions (new or affected)
None. Settled in approved Decision D-024.

## 3. Design
- **Canonical Skill Definition (`sdd-protocol`)**:
  - Name: `sdd-protocol`
  - Description: `Spec-Driven Development (SDD) contract, deterministic gating protocol (GATE 1, GATE 2), decision protocols, and task execution rules for AI coding agents.`
  - Body:
    - `# Spec-Driven Development (SDD) Protocol`
    - `## Overview & Triggers` (when to invoke: taking over development, creating features, planning tasks)
    - `## The SDD Anthem & Hard Rules` (determinism, no git writes, Gate 1 approval before code, Gate 2 human commit)
    - `## The 4-Phase Workflow` (Specify -> Implement -> Verify -> Close)
    - `## Decision Protocol (D-xxx)`
    - `## Verification Standards & Tool Execution`
- **Export Destination Resolution**:
  - Default: `<projectRoot>/skills/sdd-protocol/SKILL.md`
  - Custom: `<targetDir>/sdd-protocol/SKILL.md`
  - Global: User global skill directory based on installed agent environment (`~/.gemini/antigravity/skills/`, `~/.claude/skills/`, or fallback `~/.sdd/skills/`).

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement canonical `sdd-protocol` skill and export engine | `internal/skill/export.go`, `internal/skill/export_test.go` | `go test -v ./internal/skill -run TestExport` |
| 2 | Add CLI `sdd skill export` subcommand | `internal/cli/skill.go`, `internal/cli/cli.go`, `internal/cli/cli_test.go` | `go test -v ./internal/cli -run TestSkillExport` |
| 3 | Update living docs (`README.md`, `SKILLS.md`, `ROADMAP.md`) | `README.md`, `SKILLS.md`, `ROADMAP.md` | `make check && go run ./cmd/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/skill/export.go` | create | Canonical SDD skill content and export logic |
| `internal/skill/export_test.go` | create | Unit tests for skill export, custom target, global, and collisions |
| `internal/cli/skill.go` | modify | Add `export` subaction handler |
| `internal/cli/cli.go` | modify | Add `export` command to root usage text |
| `internal/cli/cli_test.go` | modify | End-to-end CLI tests for `sdd skill export` |
| `README.md` | modify | Document `sdd skill export` and Agent SDK loading |
| `SKILLS.md` | modify | Document programmatic Agent SDK skill integration |
| `ROADMAP.md` | modify | Track T-023 status across gates |

## 6. Dependencies
None. Standard library only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `README.md` | Document `sdd skill export` and programmatic Agent SDK usage |
| `SKILLS.md` | Document programmatic Agent SDK loading techniques |
| `ROADMAP.md` | Update T-023 status across gates |

## 8. Risks, mitigations, rollback
- **Risk:** Exporting to global directories without permission.
  - *Mitigation:* Explicit path validation, creation of parent directories with standard permissions (0755), clear error reporting if home is unavailable.
- **Rollback:** `git restore .` reverts all changes.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active.

## 10. Approval record
Approved by: Vikash   Date: 2026-11-03   Phrase received: APPROVED T-023
