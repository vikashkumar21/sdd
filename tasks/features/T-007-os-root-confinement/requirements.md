# Requirements — T-007 P-03: Filesystem confinement via os.Root

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-007

## 1. Goal
Harden `sdd` against directory traversal and symlink escape vulnerabilities by refactoring filesystem write operations in `internal/fsutil`, `internal/scaffold`, and `internal/feature` to use standard-library `os.Root` (`os.OpenRoot`), enforcing OS-level containment within the target project root.

## 2. Context and references
- Mission / principles touched: MISSION §5 (Deterministic, Fail safe, Human in the loop), MISSION §7 (S1).
- Proposals addressed: `P-03` in `features/T-005-independent-review/review.md`.
- Decisions relied on: D-007 (Standard library only), D-015 (Approved: Go 1.27, enabling `os.Root` from Go 1.24+), D-021 (Approved: Option A, cross-platform os.Root determinism, device name rejection, and Windows cross-build verification).
- Depends on: T-002 (Done).

## 3. Scope
**In scope**
- `internal/fsutil`:
  - Implement `WriteAtomicRoot(root *os.Root, relPath string, data []byte, perm fs.FileMode) error` which confines file writes, directory creation, sync, and atomic rename strictly within `root`.
  - Validate that relative paths satisfy `filepath.IsLocal(relPath)` and uniformly reject Windows reserved device names (`CON`, `PRN`, `AUX`, `NUL`, `COM1`–`COM9`, `LPT1`–`LPT9`) on all platforms.
  - Refactor `WriteAtomic(path string, data []byte, perm fs.FileMode) error` to safely delegate using `os.OpenRoot` or root-confined operations.
  - Add tests validating that `WriteAtomicRoot` rejects paths escaping the root, symlinks referencing targets outside the root, and Windows reserved device names.
- `internal/scaffold`:
  - Refactor `scaffold.Apply` to open `p.Root` via `os.OpenRoot(p.Root)` and perform writes, merges, replacements, and backups through the root handle.
  - Update `safeRel` to enforce `filepath.IsLocal` and reject Windows reserved device names uniformly.
- `internal/feature`:
  - Refactor `feature.New` to open `root` via `os.OpenRoot(root)` and write task files and roadmap updates through the root handle.
- Unit and integration tests:
  - Add explicit test cases in `fsutil_test.go`, `scaffold_test.go`, and `feature_test.go` attempting root escapes, symlink traversal, and reserved device names.
  - Verify `GOOS=windows` compilation and `go vet` pass with zero issues.

**Out of scope**
- Full Windows platform translation (CRLF normalization, drive roots, .exe in dist) which is scheduled under T-018 per D-021.
- Changing CLI flags or user-facing commands.
- Adding third-party dependencies (standard library `os.Root` only).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | `fsutil` SHALL provide `WriteAtomicRoot(root *os.Root, relPath string, data []byte, perm fs.FileMode) error` that confines all operations within `root`. | MUST | inspection: `internal/fsutil/atomic.go` |
| R-002 | `WriteAtomicRoot` SHALL create parent directories inside `root` and atomically write, fsync, and rename files. | MUST | test: `TestWriteAtomicRoot` |
| R-003 | `WriteAtomicRoot` SHALL reject paths referencing locations outside `root` or symlinks pointing outside `root`. | MUST | test: `TestWriteAtomicRootEscape`, `TestWriteAtomicRootSymlinkEscape` |
| R-004 | `scaffold.Apply` SHALL execute all file creation, merge, replace, backup, and manifest operations through an `*os.Root` opened at `p.Root`. | MUST | test: `scaffold_test.go` suite |
| R-005 | `feature.New` SHALL execute all task template and roadmap writes through an `*os.Root` opened at `root`. | MUST | test: `feature_test.go` suite |
| R-006 | Existing convenience function `fsutil.WriteAtomic` SHALL remain backward-compatible and functional. | MUST | test: `atomic_test.go` suite |
| R-007 | `make check` SHALL pass with zero issues and all test packages passing under race detector. | MUST | command: `make check`, `go test -race ./...` |
| R-008 | All new tests SHALL be deterministic and repeatable across successive runs. | MUST | test: `go test ./internal/fsutil ./internal/scaffold ./internal/feature -count=2` |
| R-009 | `WriteAtomicRoot` and `safeRel` SHALL enforce `filepath.IsLocal` and reject Windows reserved device names uniformly across all platforms. | MUST | test: `TestWriteAtomicRootReservedNames` |
| R-010 | The entire codebase SHALL compile and pass `go vet` cleanly under `GOOS=windows GOARCH=amd64`. | MUST | command: `GOOS=windows go vet ./...`, `GOOS=windows go build ./cmd/sdd` |

## 5. Non-functional requirements
- **Security:** Hardware/OS-enforced confinement prevents writing outside designated project root under any path or symlink configuration.
- **Fail-safe:** Any path escaping root, using reserved device names, or failing validation returns an error immediately and leaves no partial files.
- **Determinism:** File writes remain atomic, deterministic, and idempotent.

## 6. Constraints and assumptions
- Uses standard library `os.Root` available in Go 1.27 (`go.mod` specifies `go 1.27`).

## 7. Open questions
None.

## 8. Acceptance criteria
- Given a path attempting to escape root (e.g. `../escape.txt`), a symlink pointing outside root, or a reserved device name (e.g. `nul.txt`, `com1`), `WriteAtomicRoot` returns an error and creates no files outside root.
- Given `sdd init` or `sdd feature new`, all files are created atomically within the project root.
- Given `GOOS=windows`, the project passes build and vet.
- Given the whole repository, `make check` exits 0 with 0 issues.
