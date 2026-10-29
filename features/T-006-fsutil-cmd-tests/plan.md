# Plan — T-006 P-02: Test suite and coverage for fsutil and cmd/sdd

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Implemented

## 1. Approach summary
Implement comprehensive unit tests for `internal/fsutil` and `cmd/sdd` (both currently at 0.0% coverage):
1. In `internal/fsutil/atomic.go`, add `tmp.Sync()` before `tmp.Close()` to ensure data durability on disk as identified during review.
2. Author `internal/fsutil/atomic_test.go` covering creation, atomic replacement, directory creation, read-only permissions, and temporary file cleanup.
3. Author `cmd/sdd/main_test.go` testing `isTerminal` on pipes and files, plus end-to-end command execution and exit code verification.
4. Verify statement coverage meets targets (>= 90% for `fsutil`, >= 80% for `cmd/sdd`).
5. Verify `make check` and determinism.

## 2. Decisions (new or affected)
None. Standard library only (D-007).

## 3. Design
- **`internal/fsutil/atomic.go`**:
  ```go
  if err := tmp.Sync(); err != nil {
      _ = tmp.Close()
      cleanup()
      return fmt.Errorf("sync %s: %w", path, err)
  }
  ```
- **`internal/fsutil/atomic_test.go`**:
  - `TestWriteAtomicCreate`: verify writing new file with `0o644` and `0o755`.
  - `TestWriteAtomicReplace`: verify replacing existing content.
  - `TestWriteAtomicNestedDir`: verify creation of multi-level non-existent directories.
  - `TestWriteAtomicReadOnlyDir`: verify failure when destination directory is `0o555`, checking that no `.sdd-tmp-*` files linger.
  - `TestWriteAtomicBadPerm`: verify error handling when chmod fails or path is invalid.
- **`cmd/sdd/main_test.go`**:
  - `TestIsTerminal`: test on `os.Pipe()` reader/writer and a regular temp file (must return false).
  - `TestMainExitCodes`: execute `cli.Run` / subprocess with:
    - empty args -> exit 2
    - `version` -> exit 0
    - `unknown-command` -> exit 2

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Add fsync to `WriteAtomic` | `internal/fsutil/atomic.go` | Code inspection |
| 2 | Create `atomic_test.go` | `internal/fsutil/atomic_test.go` | `go test -v -cover ./internal/fsutil` |
| 3 | Create `main_test.go` | `cmd/sdd/main_test.go` | `go test -v -cover ./cmd/sdd` |
| 4 | Run coverage check | `cmd/sdd`, `internal/fsutil` | `go test -cover ./cmd/sdd ./internal/fsutil` (>= 85% and >= 80%) |
| 5 | Run repository verification | Project root | `make check` passes with 0 issues |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/fsutil/atomic.go` | modify | Add `tmp.Sync()` for durability before close |
| `internal/fsutil/atomic_test.go` | create | Comprehensive unit tests for atomic file writes |
| `cmd/sdd/main_test.go` | create | Unit tests for isTerminal and main entrypoint wiring |
| `features/T-006-fsutil-cmd-tests/requirements.md` | modify/review | Requirement specifications |
| `features/T-006-fsutil-cmd-tests/plan.md` | modify/review | Step-by-step implementation plan |
| `features/T-006-fsutil-cmd-tests/validation.md` | modify/review | Validation evidence and execution logs |

## 6. Dependencies
None. Standard library only (D-007).

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Mark T-006 `Awaiting review` then `Done`; update notes |

## 8. Risks, mitigations, rollback
- **Risk:** `tmp.Sync()` adds disk I/O overhead.
  - *Mitigation:* `sdd` writes only a handful of markdown files; durability guarantees far outweigh a few microseconds.
- **Rollback:** Revert `git restore .` if tests fail.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active yet.

## 10. Approval record
Approved by: Vikash   Date: 2026-10-29   Phrase received: `APPROVED T-006`
