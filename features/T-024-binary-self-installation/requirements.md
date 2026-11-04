# Requirements — T-024 Cross-platform binary self-installation (sdd install and double-click setup)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-04)
Roadmap ref: `ROADMAP.md` T-024

## 1. Goal
Provide cross-platform self-installation (`sdd install`), uninstallation (`sdd uninstall`), and interactive double-click setup so users and AI agents can install and configure `sdd` in their system PATH on macOS, Linux, and Windows without requiring Go, root/administrator permissions, or manual shell editing.

## 2. Context and references
- Mission / principles touched: Principle 1 (Deterministic), Principle 2 (Fail safe), Principle 5 (Never overwrite human work).
- Decisions this relies on: D-007 (Stdlib only), D-025 (Approved - Cross-platform self-installation architecture).
- Depends on tasks: T-001 (Core CLI).

## 3. Scope
**In scope**
- Package `internal/install`:
  - Destination resolution:
    - macOS & Linux: `~/.local/bin/sdd` (fallback `/usr/local/bin/sdd` if requested).
    - Windows: `%LOCALAPPDATA%\Programs\sdd\sdd.exe`.
  - Self-copying: atomically copies the running binary to the destination with executable permissions (`0o755`).
  - Shell PATH integration (Unix):
    - Detects active shell config (`~/.zshrc` on macOS/zsh, `~/.bashrc` on Linux/bash).
    - Checks whether `~/.local/bin` is already on PATH.
    - Appends `export PATH="$HOME/.local/bin:$PATH"` cleanly without duplicating.
  - Windows PATH integration:
    - Sets user-level environment variable without requiring Administrator privileges.
  - Double-click detection:
    - When `sdd` is launched with zero arguments in an interactive terminal and is not running from the installed directory, prompts the user to install to PATH.
    - Prompts `Press Enter to exit...` on completion when executed in a detached/interactive terminal session so the window remains visible.
  - Uninstallation:
    - `sdd uninstall` removes the installed binary and cleans up shell profile PATH entries.
- CLI commands:
  - `sdd install [--force] [--dry-run]`
  - `sdd uninstall [--yes]`
- Updates to `README.md` and `SKILLS.md`.

**Out of scope**
- Downloading binary over the internet (that is `T-025` for `install.sh`/`install.ps1`).
- Writing to system-wide protected folders (`/usr/bin`, `C:\Program Files`) requiring root/admin.

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL provide `sdd install` to copy the running binary to the user-level binary path (`~/.local/bin/sdd` on Unix, `%LOCALAPPDATA%\Programs\sdd\sdd.exe` on Windows). | MUST | test |
| R-002 | The system SHALL configure the user's shell profile (`~/.zshrc`, `~/.bashrc`) or Windows user environment to include the binary path if not already present. | MUST | test |
| R-003 | The installation SHALL be idempotent: repeated runs SHALL not duplicate PATH entries or corrupt shell configuration. | MUST | test |
| R-004 | When executed with zero arguments in an interactive terminal outside the installed path, the system SHALL prompt the user to install `sdd` into PATH. | MUST | test |
| R-005 | When double-clicked or run in a detached terminal, the installer SHALL wait for user confirmation before exiting so output remains readable. | MUST | test |
| R-006 | The system SHALL provide `sdd uninstall` to remove the installed binary and clean up shell PATH additions. | MUST | test |

## 5. Non-functional requirements
- **Security & Privilege:** Must run with standard user privileges; SHALL NOT require `sudo` or Administrator rights.
- **Determinism:** Shell profile modifications SHALL be formatted identically with distinct `# Added by sdd installer` markers.
- **Fail-safe behaviour:** Inability to write or read destination SHALL return clear error messages and exit code 1.
- **Standard Library Only:** Zero external dependencies (`D-007`).

## 6. Constraints and assumptions
- Uses standard library `os`, `path/filepath`, `io`, and `bufio`.
- Shell profile detection targets `~/.zshrc`, `~/.bash_profile`, `~/.bashrc`.

## 7. Open questions
(None. Settled in approved Decision D-025.)

## 8. Acceptance criteria
- Running `sdd install` on macOS/Linux creates `~/.local/bin/sdd` with executable permissions and updates `~/.zshrc` / `~/.bashrc`.
- Running `sdd install` a second time reports that `sdd` is already installed without modifying shell profiles.
- Running `sdd uninstall --yes` deletes the binary and removes the PATH export from shell profile.
