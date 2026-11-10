# Plan — T-014 Context staging pipeline and sdd ingest command

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved

## 1. Approach summary
Implement an offline, OS-confined context ingestion pipeline to stage external files, raw text, and piped input into `.sdd/context/`:
1. Create new package `internal/context` with data types `Manifest`, `Entry`, and function `Stage(root string, item Item, now time.Time) (Entry, error)`:
   - Manages `.sdd/context/manifest.json`.
   - Stages files into `.sdd/context/` with deterministic timestamps and slugs.
   - Enforces a 10 MB per-item limit.
   - Uses `fsutil.WriteAtomicRoot` for atomic writes confined within project root.
2. In `internal/cli`:
   - Implement `runIngest(args []string, env IO) int` supporting `--file`, `--text`, `--name`, `--task`, and piped stdin.
   - Extend `runInit` to support `--context-file` and `--context-text`.
   - Update `usageText` in `cli.go`.
3. In `internal/check`:
   - Add a rule in `check.Run` to validate `.sdd/context/manifest.json` (if present) ensuring all referenced files exist and hashes match.
4. Unit and integration tests:
   - `internal/context/context_test.go`: test staging text, files, size limits, reserved names, and manifest updates.
   - `internal/cli/ingest_test.go`: test CLI dispatch, flags, stdin pipes, and task associations.
   - `internal/check/check_test.go`: test context manifest validation.
5. Verification:
   - `make check`, `go test -race ./...`, and `GOOS=windows go vet ./...`.

## 2. Decisions (new or affected)
- D-007 (Standard library only)
- D-015 (Approved: Go 1.27 toolchain)
- D-018 (Approved: Ingestion architecture via `.sdd/context/` staging)
- D-020 (Approved: Text & file inputs staged for agent reading)
- D-021 (Approved: Path safety and Windows cross-platform determinism)

## 3. Design
- **`internal/context/manifest.go`**:
  ```go
  type Entry struct {
      ID        string `json:"id"`
      Source    string `json:"source"`    // "raw-text", "file", "stdin"
      Origin    string `json:"origin"`    // original filename or label
      Path      string `json:"path"`      // relative path inside .sdd/context/
      Task      string `json:"task,omitempty"` // e.g. "T-001"
      Format    string `json:"format"`    // "text", "markdown", "json", "pdf", etc.
      Bytes     int64  `json:"bytes"`
      SHA256    string `json:"sha256"`
      CreatedAt string `json:"created_at"`
  }

  type Manifest struct {
      Schema  int     `json:"schema"`
      Entries []Entry `json:"entries"`
  }
  ```
- **`internal/context/stage.go`**:
  ```go
  type Item struct {
      Source    string // "raw-text", "file", "stdin"
      Origin    string
      Name      string
      Task      string
      Format    string
      Data      []byte
  }

  func Stage(root string, item Item, now time.Time) (Entry, error)
  func ReadManifest(root string) (Manifest, bool, error)
  ```

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Create `internal/context` package with `Stage` and `Manifest` | `internal/context/manifest.go`, `internal/context/stage.go` | Unit tests in `context_test.go` |
| 2 | Add `sdd ingest` command implementation | `internal/cli/ingest.go`, `internal/cli/cli.go` | CLI tests in `ingest_test.go` |
| 3 | Add `--context-file` and `--context-text` to `sdd init` | `internal/cli/init.go` | CLI tests in `init_test.go` |
| 4 | Add context manifest checking in `internal/check` | `internal/check/check.go` | Tests in `check_test.go` |
| 5 | Verify full test suite, race detector, and Windows cross-compilation | Project root | `make check`, `go test -race ./...`, `GOOS=windows go vet ./...` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/context/manifest.go` | create | Data structures and JSON serialization for context manifest |
| `internal/context/stage.go` | create | Logic for staging items and atomic writing into `.sdd/context/` |
| `internal/context/context_test.go` | create | Unit tests for context staging and manifest tracking |
| `internal/cli/ingest.go` | create | Implementation of `sdd ingest` command |
| `internal/cli/ingest_test.go` | create | Unit tests for `sdd ingest` CLI flags and pipes |
| `internal/cli/cli.go` | modify | Register `ingest` command and update usage text |
| `internal/cli/init.go` | modify | Support `--context-file` and `--context-text` in `sdd init` |
| `internal/check/check.go` | modify | Validate integrity of `.sdd/context/manifest.json` |
| `internal/check/check_test.go` | modify | Tests for context manifest validation |
| `features/T-014-ingest-staging-pipeline/requirements.md` | modify/review | Requirements specification |
| `features/T-014-ingest-staging-pipeline/plan.md` | modify/review | Implementation plan |
| `features/T-014-ingest-staging-pipeline/validation.md` | modify/review | Validation evidence |

## 6. Dependencies
None. Standard library only (D-007).

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Mark T-014 `Awaiting review` then `Done`; update notes |
| `TECH_STACK.md` | Record architecture update for `internal/context` staging pipeline |
| `README.md` | Document `sdd ingest` and `--context` options for `sdd init` |

## 8. Risks, mitigations, rollback
- **Risk:** Large files could cause memory spikes.
  - *Mitigation:* Bounded at 10 MB fail-safe limit.
- **Risk:** Symlink or directory traversal when specifying `--file` or staging names.
  - *Mitigation:* Uses `filepath.IsLocal` and `fsutil.WriteAtomicRoot` with `os.OpenRoot`.
- **Rollback:** `git restore .` reverts the changes.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active yet.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-10-30   Phrase received: `APPROVED T-014`
