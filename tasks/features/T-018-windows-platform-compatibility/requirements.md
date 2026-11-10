# Requirements — T-018 Full Windows platform compatibility

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-018

## 1. Goal
Ensure full cross-platform compatibility for Windows: reject unsafe drive roots (`C:\`), reject reserved device names and drive letters in paths, ensure deterministic CRLF normalisation across all tools and checksums, produce `.exe` binaries in `make dist`, and re-verify all facts due at `T-018`.

## 2. Context and references
- Mission / principles touched: Principles 1 (Deterministic), 2 (Fail safe), Known limits §6 (Windows platform compatibility)
- Decisions this relies on: `D-003` (Go), `D-004` (Toolchain), `D-007` (Standard library only), `D-015` (Go 1.27), `D-021` (Windows compatibility scope and sequencing)
- Depends on tasks: `T-007` (Done)

## 3. Scope
**In scope**
- Unsafe drive root rejection in `guardRoot` (`C:\`, `D:/`, `c:`, network share roots).
- Hardening `fsutil.ValidateLocalRelPath` against drive letters, colons (NTFS streams), and reserved Windows device names (`CON`, `PRN`, `AUX`, `NUL`, `COM0..9`, `LPT0..9`).
- CRLF line ending tolerance and normalisation across markdown tables, code fences, contract parity checks, and manifest checksums.
- Cross-compilation of Windows static binaries (`sdd_windows_amd64.exe`, `sdd_windows_arm64.exe`) with SHA256 checksums in `make dist`.
- Re-verification and advancement of verified facts in `TECH_STACK.md` §7 scheduled for `T-018`.

**Out of scope**
- Running a Windows kernel or VM inside the local macOS developer environment.

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL refuse `sdd init` when targeting any Windows drive root (e.g. C:\, D:/, c:, \\server\share). | MUST | test |
| R-002 | The system SHALL reject relative paths with drive letters, colons, or Windows reserved device names in `fsutil.ValidateLocalRelPath`. | MUST | test |
| R-003 | The system SHALL parse and check markdown tables, code fences, contract parity, and manifest hashes deterministically regardless of whether line endings are CRLF (`\r\n`) or LF (`\n`). | MUST | test |
| R-004 | The system SHALL cross-compile static Windows binaries (`windows/amd64` and `windows/arm64`) with `.exe` extensions in `make dist`. | MUST | test |
| R-005 | The system SHALL re-verify or advance all verified facts in `TECH_STACK.md` §7 that are due at `T-018`. | MUST | inspection |

## 5. Non-functional requirements
- **Determinism:** CRLF and LF files must yield identical SHA256 hashes and identical check reports.
- **Fail-safe behaviour:** When in doubt about whether a path is a root or reserved device name, fail safe and refuse.
- **Dependencies:** Standard library only (`D-007`).

## 6. Constraints and assumptions
- Cross-compilation uses standard Go toolchain (`CGO_ENABLED=0 GOOS=windows GOARCH=...`).

## 7. Open questions
(None - settled in Decision D-021)

## 8. Acceptance criteria
- Given a drive root path such as `C:\`, when `guardRoot` runs, then it returns an error refusing to run in the filesystem root.
- Given a path containing colons or reserved device names (`CON.txt`, `NUL`, `COM1`), when `ValidateLocalRelPath` runs, then it returns an error.
- Given files with CRLF line endings, when `sdd check` runs, then contracts, tables, and manifest integrity verify without errors.
- Given `make dist`, when executed, then `dist/sdd_windows_amd64.exe` and `dist/sdd_windows_arm64.exe` are created and listed in `dist/SHA256SUMS`.
