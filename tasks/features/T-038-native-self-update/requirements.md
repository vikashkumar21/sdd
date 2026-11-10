---
status: completed
task_id: T-038
---
# Requirements — T-038 Native sdd self-update CLI binary updater

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Completed
Roadmap ref: `ROADMAP.md` T-038

## 1. Goal
Provide an in-place CLI self-update command (`sdd self-update` with `sdd update` alias) that discovers release versions from GitHub Releases, verifies binary integrity against `SHA256SUMS`, and safely swaps the running executable across Linux, macOS, and Windows with atomic replacement and rollback.

## 2. Context and references
- Mission / principles touched:
  - Principle 1 (Deterministic): Deterministic asset selection, checksum verification, and version comparison.
  - Principle 2 (Fail safe): Checksum mismatches, network errors, or non-writable paths cleanly abort without corrupting the running binary.
  - Principle 3 (Human in the loop): Interactive confirmation previewing the upgrade version before replacing the executable unless `--yes` is passed.
- Decisions this relies on:
  - D-002: Spec-driven workflow with human gates.
  - D-007: Standard library only (no external dependencies, pure Go).
  - D-025: Cross-platform self-installation architecture and locations.
  - D-034: Native binary self-update architecture and verification.
- Depends on tasks: T-034 (Done), T-037 (Done).

## 3. Scope
**In scope**
- Core Self-Update Engine (`internal/update`):
  - GitHub Releases API integration (`https://api.github.com/repos/vikashkumar21/sdd/releases/latest` and `/releases/tags/<tag>`).
  - Semver comparison between current `version.Version` and release tag.
  - Asset matching for OS and Architecture (`runtime.GOOS` and `runtime.GOARCH`) matching `dist/sdd_<os>_<arch>[.exe]`.
  - Checksum verification against `SHA256SUMS` from release assets before modifying executable.
  - Atomic executable replacement using temporary staging file and rollback capability on failure (handling Windows file locks via `.old` renaming).
  - Injectable `http.Client` and API base URL for offline deterministic testing with `httptest.Server`.
- CLI Command (`sdd self-update` and `sdd update`):
  - Flags:
    - `--check`: check if a newer release exists without downloading or modifying executable.
    - `--dry-run`: check and download/verify without replacing binary.
    - `--version <tag>`: target a specific release version (upgrade or downgrade).
    - `--force`: allow reinstalling or overwriting the same version.
    - `--yes`: proceed without interactive confirmation.
    - `--json`: machine-readable output for update checks.
- Living Docs & Tests:
  - Document `sdd self-update` in `README.md`.
  - Comprehensive unit and integration tests with synthetic releases and mock servers.

**Out of scope**
- Background telemetry or automatic background update checkers (explicit user-invoked only).
- Package manager updates (Homebrew, winget, etc.; handled externally by their respective maintainers).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL provide `sdd self-update` (with alias `sdd update`) to query release versions and download platform-matching binary assets. | MUST | test / inspection |
| R-002 | `internal/update` SHALL compare the latest release version against current `version.Version`, reporting if already up to date unless `--force` or `--version` is passed. | MUST | test |
| R-003 | The system SHALL download `SHA256SUMS` and verify that the downloaded binary's SHA256 checksum strictly matches the published digest, refusing execution on mismatch. | MUST | test |
| R-004 | The system SHALL locate the currently running binary via `os.Executable()`, stage the new binary in a temporary file with executable permissions (`0755`), and perform atomic replacement with rollback on failure. | MUST | test |
| R-005 | The system SHALL support `--check` to report update availability, current version, and latest version (in human-readable format or structured JSON via `--json`) without downloading binary assets or modifying files. | MUST | test |
| R-006 | The system SHALL support `--dry-run` to preview the update without modifying the running binary. | MUST | test |
| R-007 | The system SHALL support `--version <tag>` to target a specific release tag (e.g. `v1.5.0` or `1.5.0`). | MUST | test |
| R-008 | The system SHALL support `--yes` for non-interactive execution and prompt for user confirmation in interactive terminal sessions. | MUST | test |
| R-009 | The implementation SHALL use pure Go standard library only (`net/http`, `crypto/sha256`, `os`, `path/filepath`, `runtime`), enabling 100% offline deterministic unit tests with `httptest.Server`. | MUST | test |
| R-010 | The entire test suite (`make verify`, `make check`), `sdd check --strict`, and determinism checks SHALL pass with 0 errors and 0 warnings. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Fixed asset naming and checksum matching; offline tests use synthetic fixtures.
- **Fail safe:** Network errors, corrupt downloads, non-writable permissions, or checksum mismatches leave the existing binary untouched.
- **Standard library purity:** 100% Go standard library (D-007). Zero new third-party dependencies.

## 6. Constraints and assumptions
- The binary name matches the asset schema produced by `make dist`: `sdd_<os>_<arch>` (or `sdd_<os>_<arch>.exe` on Windows).
- Release assets are published to GitHub repository `vikashkumar21/sdd`.
- Custom base URL can be injected via options/environment for test harnesses and air-gapped environments.

## 7. Open questions
(None — settled by Decision D-034).

## 8. Acceptance criteria
- `sdd self-update --check` prints whether an update is available without modifying disk.
- `sdd self-update` detects when the binary is already up-to-date and exits cleanly.
- When an update is available, `sdd self-update` downloads the binary, verifies SHA256 against `SHA256SUMS`, and swaps the executable.
- A corrupted binary or invalid checksum aborts the update and leaves the original executable functional.
- Unit tests cover all scenarios using `httptest.Server` with zero external network access.
