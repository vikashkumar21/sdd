# Plan — T-025 Universal one-line installer scripts (install.sh and install.ps1)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-04)
Roadmap ref: `ROADMAP.md` T-025

## 1. Approach summary
1. Create `scripts/install.sh`:
   - POSIX `/bin/sh` shell script with `set -eu`.
   - Platform detection:
     - `uname -s` -> Darwin -> `darwin`, Linux -> `linux`
     - `uname -m` -> `x86_64` / `amd64` -> `amd64`, `arm64` / `aarch64` -> `arm64`
   - Download prebuilt asset and `SHA256SUMS` from GitHub Releases.
   - Verify SHA256 integrity using `shasum -a 256` or `sha256sum`.
   - Clear macOS quarantine via `xattr -d com.apple.quarantine` if present.
   - Execute `"$TMP_BIN" install` (delegating user-level binary copying and shell profile management to `sdd install`).
   - Trap-based cleanup ensuring temporary files are removed regardless of exit reason.
2. Create `scripts/install.ps1`:
   - PowerShell script for Windows.
   - Architecture detection (`amd64` vs `arm64`).
   - Download `sdd_windows_<arch>.exe` and `SHA256SUMS`.
   - Verify hash using `Get-FileHash`.
   - Execute `& $tempBin install`.
   - Cleanup temporary directory.
3. Automated test script / harness in `scripts/test_install.sh` or a Go test:
   - Validate POSIX compliance (`sh -n scripts/install.sh`).
   - Run end-to-end test using local mock HTTP file server serving test binary and SHA256SUMS.
4. Update living docs:
   - `README.md`: Document `curl -fsSL https://raw.githubusercontent.com/vikashkumar21/sdd/main/scripts/install.sh | sh` and Windows PowerShell one-liner.
   - `ROADMAP.md`: Track T-025 status across gates.
5. Verification:
   - `make verify`, `make check`, `go run ./cmd/sdd check --strict`.

## 2. Decisions (new or affected)
None. Settled in approved Decision D-025 (Cross-platform self-installation architecture).

## 3. Design
- **Single Source of Truth for Installation:**
  The installer scripts do not reinvent the logic for updating shell profiles (`~/.zshrc`, `~/.bashrc`) or creating user directories. They download the binary and invoke `sdd install`, leveraging the exact same deterministic marked-block logic built and tested in `T-024`.
- **POSIX Safety:**
  `scripts/install.sh` avoids all bashisms (no `[[`, no `${VAR//}`, no `function` keyword, no arrays) so it runs identically under `/bin/sh` on Debian/Ubuntu (`dash`), macOS (`zsh/bash`), and Alpine (`busybox ash`).

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Create `scripts/install.sh` and `scripts/install.ps1` | `scripts/install.sh`, `scripts/install.ps1` | `sh -n scripts/install.sh` |
| 2 | Create end-to-end installer tests | `internal/install/script_test.go` | `go test -v ./internal/install -run TestInstallScript` |
| 3 | Update living docs (`README.md`, `ROADMAP.md`) | `README.md`, `ROADMAP.md` | `make check && go run ./cmd/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `scripts/install.sh` | create | POSIX shell installer script for macOS and Linux |
| `scripts/install.ps1` | create | PowerShell installer script for Windows |
| `internal/install/script_test.go` | create | End-to-end tests for `install.sh` using local test server |
| `README.md` | modify | Document one-line web installation instructions |
| `ROADMAP.md` | modify | Track T-025 status across gates |

## 6. Dependencies
None. Standard library and standard OS shells only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `README.md` | Document 1-line curl/powershell web installers |
| `ROADMAP.md` | Update T-025 status across gates |

## 8. Risks, mitigations, rollback
- **Risk:** Shell scripts failing on minimalist environments lacking `curl`.
  - *Mitigation:* `install.sh` checks for `curl` and falls back to `wget`. If neither is found, it reports an informative error.
- **Rollback:** `git restore .` reverts all changes.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active.

## 10. Approval record
Approved by: Vikash   Date: 2026-11-04   Phrase received: APPROVED T-025
