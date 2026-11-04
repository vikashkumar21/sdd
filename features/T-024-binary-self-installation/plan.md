# Plan — T-024 Cross-platform binary self-installation (sdd install and double-click setup)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-04)
Roadmap ref: `ROADMAP.md` T-024

## 1. Approach summary
1. Implement `internal/install/install.go`:
   - Target path resolution:
     - Unix: `$HOME/.local/bin/sdd`
     - Windows: `%LOCALAPPDATA%\Programs\sdd\sdd.exe`
   - Self-copying: read executable binary and write atomically to destination directory with permissions `0o755`.
   - Shell PATH configuration with marked block:
     - Appends managed block:
       ```sh
       # sdd:begin path
       export PATH="$HOME/.local/bin:$PATH"
       # sdd:end path
       ```
     - Checks whether `$HOME/.local/bin` is already in `$PATH` or in profile before modifying.
     - Idempotent: checks for marked block; never duplicates entries.
   - Uninstallation:
     - Removes destination binary.
     - Strips `# sdd:begin path` block from shell profiles.
2. Implement unit tests in `internal/install/install_test.go`:
   - Test fresh install on Unix (creates binary and updates `~/.zshrc`).
   - Test idempotency (repeated install does not modify profile).
   - Test uninstall (removes binary and restores profile).
   - Test already-installed detection.
3. Wire CLI commands in `internal/cli/install.go` and `internal/cli/cli.go`:
   - `sdd install [--dry-run] [--force]`
   - `sdd uninstall [--yes]`
   - Zero-argument interactive launcher: when launched in an interactive terminal outside the installed path, asks the user whether to install to PATH, and waits on completion if run interactively.
4. Add CLI end-to-end tests in `internal/cli/cli_test.go`.
5. Update living docs:
   - `README.md`: Document `sdd install`, `sdd uninstall`, and double-click setup.
   - `ROADMAP.md`: Track T-024 status across gates.
6. Run full verification suite (`make check`, `sdd check --strict`).

## 2. Decisions (new or affected)
None. Settled in approved Decision D-025 (Option A: User-level self-installing binary with automatic PATH setup).

## 3. Design
- **Path Managed Block Markers**:
  ```sh
  # sdd:begin path
  export PATH="$HOME/.local/bin:$PATH"
  # sdd:end path
  ```
  This guarantees that `sdd uninstall` or `sdd install` can cleanly identify and manage its own lines without touching user aliases or configuration.

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement self-installation and uninstallation engine | `internal/install/install.go`, `internal/install/install_test.go` | `go test -v ./internal/install` |
| 2 | Add CLI `sdd install`, `sdd uninstall`, and interactive zero-arg setup | `internal/cli/install.go`, `internal/cli/cli.go`, `internal/cli/cli_test.go` | `go test -v ./internal/cli -run TestInstall` |
| 3 | Update living docs (`README.md`, `ROADMAP.md`) | `README.md`, `ROADMAP.md` | `make check && go run ./cmd/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/install/install.go` | create | Core self-installation, shell profile modification, and uninstall logic |
| `internal/install/install_test.go` | create | Unit tests for installation, idempotency, and uninstallation |
| `internal/cli/install.go` | create | CLI subcommand handlers for `sdd install` and `sdd uninstall` |
| `internal/cli/cli.go` | modify | Route `install` / `uninstall` commands and interactive setup |
| `internal/cli/cli_test.go` | modify | CLI tests for `sdd install` and `sdd uninstall` |
| `README.md` | modify | Document `sdd install` and double-click setup |
| `ROADMAP.md` | modify | Track T-024 status across gates |

## 6. Dependencies
None. Standard library only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `README.md` | Document `sdd install`, `sdd uninstall`, and double-click setup |
| `ROADMAP.md` | Update T-024 status across gates |

## 8. Risks, mitigations, rollback
- **Risk:** Shell configuration file corrupted if write fails midway.
  - *Mitigation:* Uses atomic file writes (`fsutil.WriteAtomic`) and marked block isolation (`# sdd:begin path`).
- **Rollback:** `git restore .` reverts all changes.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active.

## 10. Approval record
Approved by: Vikash   Date: 2026-11-04   Phrase received: APPROVED T-024
