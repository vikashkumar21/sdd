# Requirements — T-025 Universal one-line installer scripts (install.sh and install.ps1)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-04)
Roadmap ref: `ROADMAP.md` T-025

## 1. Goal
Provide production-grade, universal single-command web installers (`scripts/install.sh` for macOS/Linux and `scripts/install.ps1` for Windows) that automatically detect platform architecture, download the appropriate prebuilt static binary release, verify integrity, and invoke `sdd install` without requiring Go, root/administrator permissions, or manual setup.

## 2. Context and references
- Mission / principles touched: Principle 1 (Deterministic), Principle 2 (Fail safe), Principle 5 (Never overwrite human work).
- Decisions this relies on: D-007 (Stdlib only), D-025 (Approved - Cross-platform self-installation architecture).
- Depends on tasks: T-024 (Cross-platform binary self-installation).

## 3. Scope
**In scope**
- `scripts/install.sh`:
  - POSIX-compliant (`/bin/sh`), strict mode (`set -eu`).
  - Automatic OS detection: macOS (`darwin`) and Linux (`linux`).
  - Automatic architecture detection: `arm64` / `aarch64` and `amd64` / `x86_64`.
  - Fallback and error reporting for unsupported OS or architecture.
  - Download binary using `curl` (fallback to `wget`).
  - Checksum verification using `sha256sum` or `shasum -a 256` against release `SHA256SUMS`.
  - Quarantine flag clearing on macOS (`xattr -d com.apple.quarantine`).
  - Delegation to downloaded binary's `sdd install` command for atomic user-level directory and shell profile PATH setup.
  - Trap-based cleanup of temporary directories.
  - Support for `SDD_VERSION`, `SDD_REPO`, and `SDD_DRY_RUN` environment variables.
- `scripts/install.ps1`:
  - PowerShell 5.1+ and PowerShell Core/7+ compatible.
  - Automatic architecture detection (`arm64` vs `amd64`).
  - Download via `Invoke-WebRequest` / `Invoke-RestMethod`.
  - SHA256 checksum verification via `Get-FileHash`.
  - Delegation to downloaded binary's `sdd install` command.
  - Temporary file cleanup.
- Testing:
  - Test script covering POSIX compatibility, architecture mapping, local test download, and installation delegation.
- Documentation:
  - Update `README.md` with 1-line installation commands:
    - macOS / Linux: `curl -fsSL https://raw.githubusercontent.com/vikashkumar21/sdd/main/scripts/install.sh | sh`
    - Windows: `irm https://raw.githubusercontent.com/vikashkumar21/sdd/main/scripts/install.ps1 | iex`

**Out of scope**
- Package managers (Homebrew, Chocolatey, Scoop, Winget) — future milestone.
- System-wide root installation (`/usr/local/bin` requiring `sudo`).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL provide `scripts/install.sh` supporting macOS and Linux on `arm64` and `amd64`. | MUST | test / demo |
| R-002 | `scripts/install.sh` SHALL be POSIX `/bin/sh` compliant with zero bashisms and run under `dash` and `sh`. | MUST | test |
| R-003 | `scripts/install.sh` SHALL download the appropriate release binary from GitHub Releases for the detected platform. | MUST | test |
| R-004 | `scripts/install.sh` SHALL verify the downloaded binary's SHA256 checksum against `SHA256SUMS` when available. | MUST | test |
| R-005 | `scripts/install.sh` SHALL invoke the downloaded binary's `install` command, ensuring atomic user directory placement and marked shell PATH setup. | MUST | test |
| R-006 | The system SHALL provide `scripts/install.ps1` supporting Windows on `amd64` and `arm64` using native PowerShell. | MUST | inspection / test |
| R-007 | `scripts/install.sh` and `scripts/install.ps1` SHALL clean up all temporary download files upon completion or abort. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Downloaded binary matches checksum byte-for-byte; installation modifies profile identically.
- **Fail-safe behaviour:** Inability to detect architecture, download file, or verify checksum terminates immediately with exit code 1 and an actionable error message.
- **Security & Privilege:** Runs strictly as unprivileged user; never prompts for `sudo`.
- **Zero Dependencies:** Requires only standard POSIX tools (`curl` or `wget`, `shasum` or `sha256sum`) on Unix, and standard PowerShell on Windows.

## 6. Constraints and assumptions
- GitHub repository is `vikashkumar21/sdd`.
- Releases publish assets with naming convention:
  `sdd_darwin_arm64`, `sdd_darwin_amd64`, `sdd_linux_arm64`, `sdd_linux_amd64`, `sdd_windows_arm64.exe`, `sdd_windows_amd64.exe`, and `SHA256SUMS`.

## 7. Open questions
(None. Settled in approved Decision D-025.)

## 8. Acceptance criteria
- Executing `sh scripts/install.sh` with a local mock server or release URL detects platform, downloads binary, verifies checksum, and installs `sdd` to `~/.local/bin/sdd`.
- Running `sh scripts/install.sh` on unsupported architectures safely exits with an informative error.
- Temporary files are reliably removed upon success or failure.
