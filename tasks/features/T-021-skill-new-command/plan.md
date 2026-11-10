# Plan — T-021 sdd skill new command with file and text ingestion

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-03)
Roadmap ref: `ROADMAP.md` T-021

## 1. Approach summary
1. Implement skill scaffolding engine in `internal/skill/create.go`:
   - `CreateOptions` struct carrying name, description, body/source, internal flag, force flag, and injected clock.
   - Name and description pre-validation using existing `internal/skill` rules.
   - Target path resolution: `skills/<name>/SKILL.md` (or `.sdd/skills/<name>/SKILL.md` if `Internal`).
   - Duplicate collision detection: refuses with an explicit error if the skill directory or `SKILL.md` exists unless `Force` is true.
   - Atomic backup of previous skill on `Force` using `.sdd/backup/`.
   - Scaffold template rendering:
     - Frontmatter with `name` and `description`.
     - Standard sections: `## Overview & Triggers`, `## Step-by-Step Instructions`, `## Verification`, `## Troubleshooting`.
     - When ingesting raw content via `FromFile` or `FromText`, embeds raw notes in `<!-- sdd:skill-refine:begin -->` ... `<!-- sdd:skill-refine:end -->` with clear instructions for AI agents to refine into the standard sections.
   - Atomic write via `fsutil.WriteAtomic`.
   - Post-creation self-validation via `skill.Parse`.
2. Add unit tests in `internal/skill/create_test.go`:
   - Basic creation, invalid name rejection, duplicate collision refusal, force overwrite with backup verification, `--from-file`, and `--from-text` refinement scaffolding.
3. Wire CLI command `sdd skill new` in `internal/cli/skill.go` and `internal/cli/cli.go`:
   - Handle flags `--description`, `--from-file`, `--from-text`, `--internal`, `--force`.
   - Dispatch to `skill.CreateSkill`.
4. Add CLI end-to-end tests in `internal/cli/cli_test.go`.
5. Update `SKILLS.md` with Agent Skill Authoring and Refinement Standard.
6. Run full verification suite (`make check`, `sdd check --strict`).

## 2. Decisions (new or affected)
None. Settled in approved Decision D-024.

## 3. Design
- **Engine Signature (`internal/skill/create.go`)**:
  ```go
  type CreateOptions struct {
      Name        string
      Description string
      FromFile    string
      FromText    string
      Internal    bool
      Force       bool
      Clock       func() time.Time
  }

  func CreateSkill(root string, opts CreateOptions) (string, error)
  ```
- **Collision and Duplicate Handling**:
  - Target directory: `filepath.Join(root, baseDir, opts.Name)` where `baseDir` is `skills` or `.sdd/skills`.
  - Target file: `filepath.Join(targetDir, "SKILL.md")`.
  - If `SKILL.md` or `targetDir` exists and `!opts.Force`:
    Returns `fmt.Errorf("skill %q already exists at %s (use --force to overwrite)", opts.Name, relPath)`.
  - If `opts.Force`:
    Backs up existing `SKILL.md` to `.sdd/backup/<timestamp>_skills_<name>_SKILL.md`.
- **Refinement Scaffolding Layout**:
  ```markdown
  ---
  name: <name>
  description: <description>
  ---

  # <name>

  <!-- sdd:skill-refine:begin -->
  > [!NOTE]
  > The notes below were ingested from raw context. When an agent takes over,
  > refine these into the standard sections below.

  ## Ingested Context
  <raw content>
  <!-- sdd:skill-refine:end -->

  ## Overview & Triggers

  ## Step-by-Step Instructions

  ## Verification

  ## Troubleshooting
  ```

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement skill creation engine with collision safety, refinement block, and backup | `internal/skill/create.go`, `internal/skill/create_test.go` | `go test -v ./internal/skill -run TestCreateSkill` |
| 2 | Add CLI dispatch and flags for `sdd skill new` | `internal/cli/skill.go`, `internal/cli/cli.go`, `internal/cli/cli_test.go` | `go test -v ./internal/cli -run TestSkillNew` |
| 3 | Update living docs (`SKILLS.md`, `README.md`, `ROADMAP.md`) | `SKILLS.md`, `README.md`, `ROADMAP.md` | `make check && go run ./cmd/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/skill/create.go` | create | Scaffolding and file/text ingestion logic for skills |
| `internal/skill/create_test.go` | create | Unit tests for skill creation, collisions, refinement block, and backups |
| `internal/cli/skill.go` | create | CLI subcommand handler for `sdd skill` |
| `internal/cli/cli.go` | modify | Route `skill` command dispatch |
| `internal/cli/cli_test.go` | modify | End-to-end CLI tests for `sdd skill new` |
| `SKILLS.md` | modify | Document the Agent Skill Authoring and Refinement Standard |
| `README.md` | modify | Document `sdd skill new` usage and flags |
| `ROADMAP.md` | modify | Track T-021 status |

## 6. Dependencies
None. Standard library only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `SKILLS.md` | Add Section 9: Agent Skill Authoring and Refinement Standard |
| `README.md` | Document `sdd skill new` usage, flags, and raw ingestion behaviour |
| `ROADMAP.md` | Update T-021 status across gates |

## 8. Risks, mitigations, rollback
- **Risk:** Existing user skill unintentionally overwritten when running `sdd skill new`.
  - *Mitigation:* Explicit collision check refusing overwrite without `--force`; atomic backup to `.sdd/backup/` when `--force` is used.
- **Rollback:** `git restore .` reverts all changes.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active.

## 10. Approval record
Approved by: Vikash   Date: 2026-11-03   Phrase received: APPROVED T-021
