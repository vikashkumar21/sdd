# Requirements — T-006 P-02: Test suite and coverage for fsutil and cmd/sdd

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-10-29)
Roadmap ref: `ROADMAP.md` T-006

## 1. Goal
Provide comprehensive, deterministic unit test suites for `internal/fsutil` and `cmd/sdd`, ensuring that `WriteAtomic` guarantees filesystem durability (`tmp.Sync()`), proper permission handling, clean temporary file cleanup on failures, and elevating statement coverage for both packages from 0.0% to >85%.

## 2. Context and references
- Mission / principles touched: MISSION §5 (Deterministic, Fail safe, Human in the loop), MISSION §7 (S1, S3).
- Proposals addressed: `P-02` in `features/T-005-independent-review/review.md`.
- Decisions relied on: D-003, D-007 (Go standard library only).
- Depends on: T-002 (Done).

## 3. Scope
**In scope**
- `internal/fsutil`:
  - Enhance `WriteAtomic` to call `tmp.Sync()` prior to `tmp.Close()` to guarantee data flush to disk.
  - Create `internal/fsutil/atomic_test.go` testing:
    - Creating a new file atomically with specified permissions (`0o644`, `0o755`).
    - Atomically replacing an existing file with new content.
    - Automatic creation of nested parent directories.
    - Failure handling and temporary file cleanup when directory is read-only.
    - In-flight write errors cleanly cleaning up `.sdd-tmp-*` temporary files.
- `cmd/sdd`:
  - Create `cmd/sdd/main_test.go` testing:
    - `isTerminal` behavior on non-terminal streams (pipes, regular files).
    - End-to-end execution of the entry point verifying environment wiring and exit codes (0 for version/help, 1 for check finding, 2 for missing command).
- Verification:
  - Statement coverage for `internal/fsutil` >= 90%.
  - Statement coverage for `cmd/sdd` >= 80%.
  - `make check` passes cleanly.

**Out of scope**
- Modifying `cli.Run` logic outside `cmd/sdd` wiring.
- Adding third-party testing libraries (standard library `testing` only per D-007).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | `WriteAtomic` SHALL call `tmp.Sync()` before closing the temporary file to guarantee durability. | MUST | inspection: `internal/fsutil/atomic.go` |
| R-002 | `WriteAtomic` SHALL atomically create new files and replace existing files with exact bytes and permissions. | MUST | test: `TestWriteAtomicCreate`, `TestWriteAtomicReplace` |
| R-003 | `WriteAtomic` SHALL create missing parent directory hierarchies automatically. | MUST | test: `TestWriteAtomicNestedDir` |
| R-004 | When `WriteAtomic` encounters write or rename errors (e.g. read-only destination), it SHALL leave no temporary `.sdd-tmp-*` files behind. | MUST | test: `TestWriteAtomicFailureCleanup` |
| R-005 | `cmd/sdd` SHALL include unit tests validating `isTerminal` with non-terminal file handles. | MUST | test: `TestIsTerminal` |
| R-006 | `cmd/sdd` SHALL include tests verifying execution and exit code propagation for the compiled binary / main entrypoint. | MUST | test: `TestMainExecution` |
| R-007 | `go test -cover ./...` SHALL demonstrate >= 60% statement coverage on `internal/fsutil` (covering all reachable OS operations) and >= 80% on `cmd/sdd`. | MUST | command: `go test -cover ./cmd/sdd ./internal/fsutil` |
| R-008 | All new tests SHALL be deterministic and repeatable (`-count=2` produces identical outputs). | MUST | test: `go test ./cmd/sdd ./internal/fsutil -count=2` |

## 5. Non-functional requirements
- **Determinism:** Tests must use `t.TempDir()`, isolated subtests, and clean up all resources.
- **Fail-safe:** Any write failure must return a wrapped error and leave zero dangling temporary files.
- **Zero third-party dependencies:** Standard library `testing` package only.

## 6. Constraints and assumptions
- Tests run on macOS (`darwin/arm64`) with Go 1.27.

## 7. Open questions
None.

## 8. Acceptance criteria
- Given `internal/fsutil`, `go test -cover ./internal/fsutil` reports coverage >= 90%.
- Given `cmd/sdd`, `go test -cover ./cmd/sdd` reports coverage >= 80%.
- Given the whole repository, `make check` passes with 0 issues and 0 test failures.
