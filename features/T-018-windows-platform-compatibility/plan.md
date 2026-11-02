# Plan — T-018 Full Windows platform compatibility

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-018

## 1. Approach summary
1. Implement `isDriveRoot` in `internal/cli/init.go` to reject Windows drive roots (`C:\`, `D:/`, `c:`, UNC roots) in `guardRoot`.
2. Harden `fsutil.ValidateLocalRelPath` in `internal/fsutil/atomic.go` to reject colons (drive prefixes and NTFS streams) and verify reserved Windows device names (`CON`, `PRN`, `AUX`, `NUL`, etc.).
3. Ensure CRLF tolerance in `internal/check/check.go` (`contractText` and line ending checks) and add comprehensive CRLF test cases.
4. Update `Makefile` `dist` target to cross-compile static binaries for `windows/amd64` and `windows/arm64` with `.exe` extensions.
5. Re-verify all facts due at `T-018` in `TECH_STACK.md` §7 and update living docs.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| (none) | Settled in Decision D-021 | n/a | D-021: Full Windows platform compatibility in M3 | Approved |

## 3. Design
- **Drive Root Detection (`internal/cli/init.go`)**:
  - `isDriveRoot(p string) bool`: detects `/`, `\`, `p == filepath.Dir(p)`, `filepath.VolumeName(p)`, and drive patterns like `C:\`, `C:/`, `c:`.
- **Path Sanitization (`internal/fsutil/atomic.go`)**:
  - `ValidateLocalRelPath(relPath string)`: rejects paths containing colons (`strings.Contains(cleaned, ":")`) and reserved device names.
- **CRLF Invariance (`internal/check/check.go`)**:
  - Verify that `c.load` CRLF stripping and `contractText` trimming ensure identical findings on CRLF vs LF.
- **Cross-compilation (`Makefile`)**:
  - Build targets: `darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64 windows/arm64`.
  - Appends `.exe` for `windows` OS builds.

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Add isDriveRoot check in cli.guardRoot and test drive root refusals | `internal/cli/init.go`, `internal/cli/cli_test.go` | `go test ./internal/cli -run TestInitDriveRootRefusals` |
| 2 | Harden fsutil.ValidateLocalRelPath against colons and verify device names | `internal/fsutil/atomic.go`, `internal/fsutil/atomic_test.go` | `go test ./internal/fsutil -run TestValidateLocalRelPathWindows` |
| 3 | Add CRLF parity and drift tests in internal/check | `internal/check/check.go`, `internal/check/check_test.go` | `go test ./internal/check -run TestCheckCRLFInvariance` |
| 4 | Update Makefile dist target for windows/amd64 and windows/arm64 with .exe | `Makefile` | `make dist` |
| 5 | Re-verify facts due at T-018 in TECH_STACK.md and update living docs | `TECH_STACK.md`, `README.md`, `ROADMAP.md` | `go run ./cmd/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/cli/init.go` | modify | Add isDriveRoot to guardRoot |
| `internal/cli/cli_test.go` | modify | Test drive root refusals |
| `internal/fsutil/atomic.go` | modify | Reject colons in ValidateLocalRelPath |
| `internal/fsutil/atomic_test.go` | modify | Test Windows drive letter and colon rejection |
| `internal/check/check.go` | modify | Ensure CRLF trimming in contractText |
| `internal/check/check_test.go` | modify | Tests for CRLF invariance in check |
| `Makefile` | modify | Cross-compile Windows binaries in make dist |
| `README.md` | modify | Update platform compatibility notes |
| `TECH_STACK.md` | modify | Re-verify facts due at T-018 |
| `ROADMAP.md` | modify | Update T-018 status |

## 6. Dependencies
None. Standard library only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `README.md` | Note Windows platform compatibility and prebuilt Windows binaries |
| `TECH_STACK.md` | Re-verify facts in §7 and advance re-verification milestones |
| `ROADMAP.md` | Update T-018 status across gates |

## 8. Risks, mitigations, rollback
- **Risk:** Cross-compilation may fail if OS-specific syscalls or imports were introduced.
  - *Mitigation:* We use pure Go standard library without cgo (`CGO_ENABLED=0`).
- **Rollback:** `git restore .` reverts all changes.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active.

## 10. Approval record
Approved by: Vikash   Date: 2026-11-02   Phrase received: `APPROVED T-018`
