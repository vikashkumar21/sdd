# Plan — T-022 sdd skill list and progressive disclosure handover prompt

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-03)
Roadmap ref: `ROADMAP.md` T-022

## 1. Approach summary
1. Extend `internal/cli/skill.go`:
   - Implement `runSkillList(args []string, env IO) int` handling `sdd skill list [--json] [--dir <path>]`.
   - Use `skill.Discover(root)` to find all skills.
   - Sort skills deterministically by Name.
   - For table output: print aligned columns (`NAME`, `SCOPE`, `STATUS`, `DESCRIPTION`).
   - For JSON output: serialize a structured list with path, scope, validity, and metadata.
2. Wire `list` subcommand into `runSkill` in `internal/cli/skill.go`.
3. Enrich `sdd prompt` in `internal/cli/commands.go`:
   - Add `skillView` struct:
     ```go
     type skillView struct {
         Name        string
         Scope       string
         Description string
         Path        string
         Valid       bool
     }
     ```
   - Populate `Skills []skillView` and `HasRefinementSkills bool` in `promptData`.
   - Update `internal/templates/data/base/prompt.md.tmpl` to render the progressive disclosure table and refinement guidance when `.Skills` is non-empty.
4. Add test coverage in `internal/cli/cli_test.go` and `internal/cli/prompt_context_test.go`:
   - Test `sdd skill list` with empty project.
   - Test `sdd skill list` with valid and invalid skills.
   - Test `sdd skill list --json`.
   - Test `sdd prompt` output with active skills and refinement notes.
5. Update living documents (`README.md`, `ROADMAP.md`).
6. Run full verification suite (`make check`, `sdd check --strict`).

## 2. Decisions (new or affected)
None. Settled in approved Decision D-024.

## 3. Design
- **Progressive Disclosure Prompt Pattern**:
  Only `name`, `scope`, and `description` are loaded into prompt context upfront:
  ```markdown
  ### Available Agent Skills (agentskills.io)

  The following skills are available in this project:
  | Skill Name | Scope | Description |
  |---|---|---|
  | `code-review` | project | Review pull requests for security and style |
  ```
  The agent only opens the full `SKILL.md` when the task actively calls for it.
- **Skill Scope Resolution**:
  - Starts with `.sdd/skills/` -> `internal`
  - Starts with `skills/` -> `project`

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement `sdd skill list` command and `--json` support | `internal/cli/skill.go`, `internal/cli/cli_test.go` | `go test -v ./internal/cli -run TestSkillList` |
| 2 | Enrich `sdd prompt` with skill progressive disclosure catalog | `internal/cli/commands.go`, `internal/templates/data/base/prompt.md.tmpl`, `internal/cli/prompt_context_test.go` | `go test -v ./internal/cli -run TestPromptWithSkills` |
| 3 | Update living docs (`README.md`, `ROADMAP.md`) | `README.md`, `ROADMAP.md` | `make check && go run ./cmd/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/cli/skill.go` | modify | Implement `runSkillList` and route `sdd skill list` |
| `internal/cli/commands.go` | modify | Populate `Skills` and `HasRefinementSkills` in `promptData` |
| `internal/templates/data/base/prompt.md.tmpl` | modify | Render skill catalog and refinement notice in handover prompt |
| `internal/cli/cli_test.go` | modify | Tests for `sdd skill list` and `--json` |
| `internal/cli/prompt_context_test.go` | modify | Tests for `sdd prompt` with skills |
| `README.md` | modify | Document `sdd skill list` usage |
| `ROADMAP.md` | modify | Track T-022 status across gates |

## 6. Dependencies
None. Standard library only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `README.md` | Document `sdd skill list`, `--json`, and `sdd prompt` progressive disclosure handover behaviour |
| `ROADMAP.md` | Update T-022 status across gates |

## 8. Risks, mitigations, rollback
- **Risk:** Prompt token inflation if skill descriptions are long.
  - *Mitigation:* Descriptions are bounded to 1024 characters by validator; only metadata is included in the prompt, leaving bodies on disk.
- **Rollback:** `git restore .` reverts all changes.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active.

## 10. Approval record
Approved by: Vikash   Date: 2026-11-03   Phrase received: APPROVED T-022
