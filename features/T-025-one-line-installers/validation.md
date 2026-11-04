# Validation — T-025 Universal one-line installer scripts (install.sh and install.ps1)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Passed
Roadmap ref: `ROADMAP.md` T-025

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `sh -n scripts/install.sh` | Valid POSIX syntax | PASS: syntax check passed with 0 errors | PASS |
| R-002 | test | `go test -v ./internal/install -run TestInstallScriptPOSIX` | Script executes under `/bin/sh` without bashisms | PASS: executes cleanly under POSIX sh | PASS |
| R-003 | test | `go test -v ./internal/install -run TestInstallScriptEndToEnd/successful_install` | Downloads correct asset for detected platform | PASS: downloads matching sdd_${OS}_${ARCH} asset | PASS |
| R-004 | test | `go test -v ./internal/install -run TestInstallScriptEndToEnd/successful_install` | Verifies SHA256 checksum against SHA256SUMS | PASS: "✓ Verified SHA256 checksum" verified | PASS |
| R-005 | test | `go test -v ./internal/install -run TestInstallScriptEndToEnd/successful_install` | Invokes downloaded binary's `install` command | PASS: mock installer executed successfully | PASS |
| R-006 | inspection | Manual review of `scripts/install.ps1` | PowerShell syntax and architecture resolution valid | PASS: standard PowerShell 5.1+ syntax, 64-bit arm64/amd64 detection and Get-FileHash verification | PASS |
| R-007 | test | `go test -v ./internal/install -run TestInstallScriptEndToEnd` | Temporary download directories removed upon exit | PASS: trap cleanup removes temporary directories on exit | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | PASS (gofmt clean, go vet clean, all tests pass) |
| `make check` | PASS (go vet clean, golangci-lint clean with 0 issues, all tests pass) |
| `go run ./cmd/sdd check --strict` | PASS (0 errors, 0 warnings, 1 info) |

## 3. Manual checks
- Verified `scripts/install.sh` syntax and permissions (`0o755`).
- Verified `scripts/install.ps1` structure and SHA256 verification block.

## 4. Negative and failure cases
- Tested corrupt checksum failure (`TestInstallScriptEndToEnd/checksum_mismatch_failure`): script detects mismatch and aborts without executing binary.
- Unsupported OS and architecture handling in switch statements.
- Handling when both `curl` and `wget` are absent.

## 5. Determinism check
Ran tests twice; outputs identical:
- `go test ./internal/install -count=1` (Run 1: PASS, 0.183s)
- `go test ./internal/install -count=1` (Run 2: PASS, 0.193s)

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
Suggested commit message: `feat(scripts): add universal one-line installer scripts (T-025)`
Reviewed by: Vikash   Date: 2026-11-04   Reply received: `COMMITTED T-025`
