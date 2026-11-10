# Plan — T-007 P-03: Filesystem confinement via os.Root

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved

## 1. Approach summary
Leverage modern Go standard-library `os.Root` (enabled by Go 1.27) to guarantee that all filesystem write, read, and backup operations in `sdd` are strictly confined to the project directory:
1. In `internal/fsutil/atomic.go`, implement `WriteAtomicRoot(root *os.Root, relPath string, data []byte, perm fs.FileMode) error` which validates paths via `filepath.IsLocal`, uniformly rejects Windows reserved device names, and performs directory creation, temporary file creation, write, fsync, chmod, and atomic rename strictly through `root`.
2. Refactor `WriteAtomic(path string, data []byte, perm fs.FileMode) error` to safely delegate to `WriteAtomicRoot` using `os.OpenRoot(filepath.Dir(path))`.
3. In `internal/scaffold/apply.go`, refactor `Apply` to open `os.OpenRoot(p.Root)` and execute all reads, writes, backups, and manifest generation through the root handle, and update `safeRel` to use `filepath.IsLocal`.
4. In `internal/feature/feature.go`, refactor `feature.New` to open `os.OpenRoot(root)` and execute all template reads and file writes through the root handle.
5. Add unit and integration tests verifying that paths escaping the root (e.g. `../outside.txt`), symlinks pointing outside the root, and Windows reserved device names are rejected.
6. Verify with `make check`, race detector, and cross-platform verification via `GOOS=windows go vet ./...` and `GOOS=windows go build ./cmd/sdd`.

## 2. Decisions (new or affected)
- D-007 (Standard library only)
- D-015 (Approved: Go 1.27 toolchain)
- D-021 (Approved: Option A, cross-platform os.Root determinism, device name rejection, and Windows cross-build; T-018 scheduled for full Windows support)

## 3. Design
- **`internal/fsutil/atomic.go`**:
  ```go
  // IsReservedDeviceName reports whether a path component is a Windows reserved device name.
  func IsReservedDeviceName(name string) bool { ... }

  // WriteAtomicRoot writes data atomically to relPath beneath root.
  func WriteAtomicRoot(root *os.Root, relPath string, data []byte, perm fs.FileMode) error {
      if !filepath.IsLocal(relPath) {
          return fmt.Errorf("path %q is not a local relative path", relPath)
      }
      // Reject Windows reserved device names uniformly
      if isReservedPath(relPath) {
          return fmt.Errorf("path %q uses reserved device name", relPath)
      }
      dir := filepath.Dir(relPath)
      if dir != "." && dir != "" {
          if err := root.MkdirAll(dir, 0o755); err != nil {
              return fmt.Errorf("create dir %s: %w", dir, err)
          }
      }
      // Generate temporary file name inside dir beneath root
      tmpRel := filepath.Join(dir, fmt.Sprintf(".sdd-tmp-%d", time.Now().UnixNano()))
      tmp, err := root.OpenFile(tmpRel, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
      if err != nil {
          return fmt.Errorf("temp file in %s: %w", dir, err)
      }
      cleanup := func() { _ = root.Remove(tmpRel) }
      if _, err := tmp.Write(data); err != nil {
          _ = tmp.Close()
          cleanup()
          return fmt.Errorf("write %s: %w", relPath, err)
      }
      if err := tmp.Sync(); err != nil {
          _ = tmp.Close()
          cleanup()
          return fmt.Errorf("sync %s: %w", relPath, err)
      }
      if err := tmp.Close(); err != nil {
          cleanup()
          return fmt.Errorf("close %s: %w", relPath, err)
      }
      if err := root.Chmod(tmpRel, perm); err != nil {
          cleanup()
          return fmt.Errorf("chmod %s: %w", relPath, err)
      }
      if err := root.Rename(tmpRel, relPath); err != nil {
          cleanup()
          return fmt.Errorf("rename to %s: %w", relPath, err)
      }
      return nil
  }
  ```
- **`internal/scaffold/apply.go`**:
  ```go
  rootHandle, err := os.OpenRoot(p.Root)
  if err != nil {
      return rep, fmt.Errorf("scaffold: open project root %s: %w", p.Root, err)
  }
  defer rootHandle.Close()
  ```
  All reads, backups, and writes delegate through `rootHandle`.
- **`internal/feature/feature.go`**:
  ```go
  rootHandle, err := os.OpenRoot(root)
  if err != nil {
      return Result{}, fmt.Errorf("feature: open root %s: %w", root, err)
  }
  defer rootHandle.Close()
  ```

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement `WriteAtomicRoot` with `filepath.IsLocal` and device name rejection | `internal/fsutil/atomic.go` | Unit tests in `atomic_test.go` |
| 2 | Add confinement, symlink escape, and reserved name tests | `internal/fsutil/atomic_test.go` | `go test -v ./internal/fsutil` |
| 3 | Refactor `scaffold.Apply` and `safeRel` to use `os.Root` and `filepath.IsLocal` | `internal/scaffold/apply.go` | `go test -v ./internal/scaffold` |
| 4 | Refactor `feature.New` to use `os.Root` | `internal/feature/feature.go` | `go test -v ./internal/feature` |
| 5 | Verify entire test suite, race detector, and Windows cross-compilation | Project root | `make check`, `go test -race ./...`, `GOOS=windows go vet ./...` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/fsutil/atomic.go` | modify | Implement `WriteAtomicRoot` and update `WriteAtomic` |
| `internal/fsutil/atomic_test.go` | modify | Add `WriteAtomicRoot` tests, root escapes, and reserved names |
| `internal/scaffold/apply.go` | modify | Integrate `os.Root` confinement in `Apply` and update `safeRel` |
| `internal/scaffold/scaffold_test.go` | modify | Add test asserting confinement in scaffold |
| `internal/feature/feature.go` | modify | Integrate `os.Root` confinement in `feature.New` |
| `features/T-007-os-root-confinement/requirements.md` | modify/review | Requirements specification |
| `features/T-007-os-root-confinement/plan.md` | modify/review | Step-by-step implementation plan |
| `features/T-007-os-root-confinement/validation.md` | modify/review | Validation evidence and execution logs |

## 6. Dependencies
None. Standard library only (`os.Root` in Go 1.27, D-007).

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Mark T-007 `Awaiting review` then `Done`; schedule T-018 for full Windows support |
| `TECH_STACK.md` | Record D-021; update architecture note describing `os.Root` filesystem confinement |

## 8. Risks, mitigations, rollback
- **Risk:** `os.Root` behaves differently on Windows device names (`NUL`, `COM1`).
  - *Mitigation:* Explicitly rejected on all operating systems in `fsutil` and `safeRel` per D-021; full Windows support scheduled as T-018.
- **Risk:** Existing symlinked project roots might be affected.
  - *Mitigation:* `os.OpenRoot` follows symlinks to the directory root itself, but confines internal operations to not escape. Tested in T-002 verification.
- **Rollback:** `git restore .` reverts to string-based path checking.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active yet.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-10-30   Phrase received: `APPROVED T-007`
