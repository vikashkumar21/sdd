# Validation — T-024 Cross-platform binary self-installation (sdd install and double-click setup)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Passed
Roadmap ref: `ROADMAP.md` T-024

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/install -run TestInstallUnix` | Copies binary to destination directory (~/.local/bin/sdd) with permissions 0755 | PASS: target binary created with exact source content and executable permissions | PASS |
| R-002 | test | `go test -v ./internal/install -run TestInstallUnix` | Appends marked PATH block to shell profile (.zshrc, .bashrc) | PASS: .zshrc updated with marked # sdd:begin path block preserving existing aliases | PASS |
| R-003 | test | `go test -v ./internal/install -run TestInstallUnix` | Repeated runs do not duplicate PATH block | PASS: PathUpdated is false on second run, exactly 1 marker in file | PASS |
| R-004 | test | `go test -v ./internal/cli -run TestInteractiveZeroArgInstallPrompt` | Prompts for installation when executed without args outside PATH | PASS: prompts user with target location and proceeds when confirmed | PASS |
| R-005 | test | `go test -v ./internal/cli -run TestInteractiveZeroArgInstallPrompt` | Installer waits for user confirmation before exiting in interactive mode | PASS: prompts "Press Enter to exit..." before terminating | PASS |
| R-006 | test | `go test -v ./internal/install -run TestUninstall` | `sdd uninstall` removes binary and cleans shell profile | PASS: binary deleted and marked block cleanly removed while preserving user config | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | PASS (gofmt clean, go vet clean, all tests pass) |
| `make check` | PASS (go vet clean, golangci-lint clean with 0 issues, all tests pass) |
| `go run ./cmd/sdd check --strict` | PASS (0 errors, 0 warnings, 1 info) |

## 3. Manual checks
- Verified `sdd install` installs to `~/.local/bin/sdd`.
- Verified `sdd install` detects when already installed.
- Verified `sdd uninstall --yes` removes `~/.local/bin/sdd` cleanly and removes `# sdd:begin path` block.

## 4. Negative and failure cases
- Tested user declining interactive install prompt (exits code 2 with usage).
- Tested non-interactive zero-argument execution (immediately exits code 2 with usage).
- Tested uninstallation when binary was already removed or missing.
- Tested resolve target with empty home directory (returns error).

## 5. Determinism check
Ran tests twice; outputs identical:
- `go test ./internal/install ./internal/cli -count=1` (Run 1: PASS)
- `go test ./internal/install ./internal/cli -count=1` (Run 2: PASS)

## 6. Deviations, known limitations, and feedback-rule compliance
None.

Feedback-rule compliance (SKILLS.md, "Human feedback rules"): none active.

## 7. Definition of done
- [x] Every requirement has evidence
- [x] Build, lint, and tests pass (commands above)
- [x] Negative and failure cases tested
- [x] Determinism confirmed
- [x] No files changed outside the plan; no unapproved dependencies
- [x] Living docs updated
- [x] Compliant with every active feedback rule
- [x] Suggested commit message provided

## 8. Sign-off
Suggested commit message: `feat(install): add cross-platform binary self-installation and double-click setup (T-024)`
Reviewed by: Vikash   Date: 2026-11-04   Reply received: `COMMITTED T-024`
