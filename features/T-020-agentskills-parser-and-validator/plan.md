# Plan — T-020 agentskills.io parser, validator, and sdd check integration

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-020

## 1. Approach summary
1. Implement `internal/skill/skill.go`:
   - `Skill` struct with standard `agentskills.io` metadata and markdown body.
   - Pure Go stdlib YAML frontmatter parser handling single/double-quoted strings, unquoted text, multiline block scalars (`|`, `>`), and key-value maps.
   - Validation against `agentskills.io` rules:
     - Name regex `^[a-z0-9]+(?:-[a-z0-9]+)*$`, length 1–64.
     - Folder name matching.
     - Description length 1–1024, non-empty.
     - Compatibility length max 500.
   - Discovery helper `Discover(root string) ([]DiscoveredSkill, error)` to find skills in `.sdd/skills/` and `skills/`.
2. Implement unit tests in `internal/skill/skill_test.go` covering full/minimal frontmatter, multiline values, and all negative validation cases.
3. Integrate skill validation into `internal/check`:
   - Create `internal/check/skills.go` with rule `checkSkills`.
   - Wire into `internal/check/check.go`.
   - Add test cases in `internal/check/check_test.go`.
4. Run verification and update living docs (`ROADMAP.md`).

## 2. Decisions (new or affected)
None. Settled in approved Decision D-024 (Option A: Adopt `agentskills.io` standard with pure Go stdlib parser).

## 3. Design
- **Data Model (`internal/skill/skill.go`)**:
  ```go
  type Skill struct {
      Name         string            `json:"name"`
      Description  string            `json:"description"`
      License      string            `json:"license,omitempty"`
      Compatibility string           `json:"compatibility,omitempty"`
      AllowedTools string            `json:"allowed_tools,omitempty"`
      Metadata     map[string]string `json:"metadata,omitempty"`
      Body         string            `json:"body"`
      Path         string            `json:"path,omitempty"`
  }
  ```
- **Validation Rules**:
  - `name`: Must match `^[a-z0-9]+(?:-[a-z0-9]+)*$` and `len <= 64`.
  - Directory parity: `filepath.Base(dir) == skill.Name`.
  - `description`: `1 <= len(trimmed) <= 1024`.
  - `compatibility`: `len(trimmed) <= 500`.
- **`sdd check` Integration**:
  - Scans `skills/` and `.sdd/skills/` beneath project root.
  - Adds `check.Error` for frontmatter syntax errors, naming violations, and missing descriptions.

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement standard library agentskills.io parser and validator | `internal/skill/skill.go`, `internal/skill/skill_test.go` | `go test ./internal/skill` |
| 2 | Integrate skill checks into sdd check | `internal/check/skills.go`, `internal/check/check.go`, `internal/check/check_test.go` | `go test ./internal/check -run TestCheckSkills` |
| 3 | Verify entire test suite and strict checks | `ROADMAP.md` | `make check && go run ./cmd/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/skill/skill.go` | create | Pure Go agentskills.io YAML frontmatter parser and validator |
| `internal/skill/skill_test.go` | create | Unit tests for skill parsing and validation |
| `internal/check/skills.go` | create | Skill consistency checker for sdd check |
| `internal/check/check.go` | modify | Wire checkSkills into check report engine |
| `internal/check/check_test.go` | modify | Tests for skill validation rules in sdd check |
| `ROADMAP.md` | modify | Track T-020 status |

## 6. Dependencies
None. Standard library only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Update T-020 status across gates |

## 8. Risks, mitigations, rollback
- **Risk:** Complex YAML constructs (nested lists, anchors) might be present in user skills.
  - *Mitigation:* The `agentskills.io` standard explicitly specifies a flat/simple frontmatter schema (strings and simple maps); parser will support standard scalar types, quotes, and block literals.
- **Rollback:** `git restore .` reverts all changes.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active.

## 10. Approval record
Approved by: Vikash   Date: 2026-11-03   Phrase received: APPROVED T-020
