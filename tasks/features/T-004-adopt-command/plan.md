# Plan — T-004 sdd adopt: write a manifest for kits created before sdd (P-06, for example dbguard)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-004

## 1. Approach summary
1. Implement core adoption engine in `internal/adopt/adopt.go`:
   - Verify core documents (`MISSION.md`, `TECH_STACK.md`, `ROADMAP.md`, `SKILLS.md`) exist.
   - Guard against running on filesystem roots or when `.sdd/manifest.json` already exists (unless `--force`).
   - Detect project name, maintainer, mode, languages, active agent contracts, and task templates.
   - Compute SHA256 hashes of all discovered files and construct a schema 1 `scaffold.Manifest`.
   - Write `.sdd/manifest.json` atomically under `os.Root` without altering any existing project files.
2. Implement CLI command in `internal/cli/adopt.go` and register `adopt` in `internal/cli/cli.go`.
3. Add unit and integration tests in `internal/adopt/adopt_test.go` and `internal/cli/adopt_test.go`, verifying zero file mutation and `sdd check` compliance using `testdata/dbguard-kit`.
4. Update living documents (`README.md`, `ROADMAP.md`).

## 2. Decisions (new or affected)
None. Relies on approved Decision D-017 (Manifest schema 1).

## 3. Design
- **Verification of Existing Kit**:
  - `IsKit(root string) bool`: checks presence of all four core documents. If any missing, refuse with clear error.
- **Manifest Generation**:
  - `Plan(root string, opts Options) (*scaffold.Manifest, error)`:
    - Answers: Project name extracted from `MISSION.md` header or directory name; Maintainer from `MISSION.md` or env; Mode: `brownfield`; Languages: detected from source or `TECH_STACK.md`.
    - Discovered Files:
      - Agent files (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`): check for managed block via `block.Inner`. If present, `Kind: "block"`, `SHA256: sum(inner)`. If absent, `Kind: "file"`, `SHA256: sum(content)`.
      - Core docs (`MISSION.md`, `TECH_STACK.md`, `ROADMAP.md`, `SKILLS.md`, `README.md`): `Kind: "file"`, `SHA256: sum(content)`.
      - Task templates (`features/_templates/*`): `Kind: "file"`, `SHA256: sum(content)`.
      - All states set to `"existing"`.
- **Zero Mutation Guarantee**:
  - `Adopt` writes only `.sdd/manifest.json` (and `.sdd/backup/` when overwriting an existing manifest with `--force`). No markdown, code, or template files are touched.

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement core adopt engine (kit detection, hash extraction, manifest writing) | `internal/adopt/adopt.go`, `internal/adopt/adopt_test.go` | `go test ./internal/adopt` |
| 2 | Implement `sdd adopt` CLI command with flags (--dir, --dry-run, --yes, --force, --name, --maintainer) | `internal/cli/adopt.go`, `internal/cli/cli.go`, `internal/cli/adopt_test.go` | `go test ./internal/cli -run TestCLIAdopt` |
| 3 | Verify adopt on dbguard-kit and run full check suite | `internal/adopt/adopt_test.go` | `go run ./cmd/sdd check --strict` |
| 4 | Update living docs (README.md, ROADMAP.md) | `README.md`, `ROADMAP.md` | `go run ./cmd/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/adopt/adopt.go` | create | Core adoption planning and manifest generation |
| `internal/adopt/adopt_test.go` | create | Unit tests for kit detection, hashing, and zero-mutation |
| `internal/cli/adopt.go` | create | CLI handler for `sdd adopt` command |
| `internal/cli/adopt_test.go` | create | End-to-end CLI integration tests for adopt |
| `internal/cli/cli.go` | modify | Register `adopt` command in CLI router and usage |
| `README.md` | modify | Document `sdd adopt` usage and flags |
| `ROADMAP.md` | modify | Track T-004 status |

## 6. Dependencies
None. Standard library only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `README.md` | Document `sdd adopt` command and usage |
| `ROADMAP.md` | Update T-004 status across gates |

## 8. Risks, mitigations, rollback
- **Risk:** Accidental mutation of existing documents during adoption.
  - *Mitigation:* The adopt package only ever writes `.sdd/manifest.json`. Tests verify SHA256 hashes of all project files before and after adoption.
- **Rollback:** `git restore .` reverts all changes.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active.

## 10. Approval record
Approved by: Vikash   Date: 2026-11-02   Phrase received: `APPROVED T-004`
