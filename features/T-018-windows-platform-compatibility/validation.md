# Validation — T-018 Full Windows platform compatibility

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Passed
Roadmap ref: `ROADMAP.md` T-018

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/cli -run TestInitDriveRootRefusals` | Windows drive roots refused by guardRoot | Refused "/", "\\", "C:\", "D:/", "c:", "\\server\share" | PASS |
| R-002 | test | `go test -v ./internal/fsutil -run TestValidateLocalRelPathWindows` | Colons, drive prefixes, and device names rejected | Rejection of C:file, D:/path, colons, ADS, and CON/PRN/AUX/NUL/COM/LPT verified | PASS |
| R-003 | test | `go test -v ./internal/check -run TestCheckCRLFInvariance` | CRLF files pass check identically to LF files | Asymmetric CRLF contract parity preserved; full CRLF kit produces 0 errors and identical findings | PASS |
| R-004 | test | `make dist` | dist/ contains sdd_windows_amd64.exe and sdd_windows_arm64.exe | Windows binaries compiled and hashed in dist/SHA256SUMS | PASS |
| R-005 | inspection | `go run ./cmd/sdd check --strict` | Verified facts due at T-018 re-verified without warnings | 0 errors, 0 warnings; all due facts advanced to M4 | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | PASS (fmt-check, vet, all tests pass) |
| `make check` | PASS (0 issues from golangci-lint, all tests pass) |
| `make dist` | PASS (builds darwin, linux, and windows static binaries with .exe) |
| `go run ./cmd/sdd check --strict` | PASS (0 errors, 0 warnings, 1 info note) |

## 3. Manual checks
- Inspected `dist/SHA256SUMS`: verified `sdd_windows_amd64.exe` and `sdd_windows_arm64.exe` are generated and hashed.
- Ran `sdd check --strict`: verified facts in `TECH_STACK.md` §7 pass expiry checks without warnings.

## 4. Negative and failure cases
- Drive roots (`/`, `\`, `C:\`, `D:/`, `c:`, `\\server\share`) rejected with "refusing to run in the filesystem root".
- Paths with colons (`C:file.txt`, `D:/path`, `foo:stream`, `sub/dir:ads`) rejected.
- Reserved Windows device names (`CON`, `PRN`, `AUX`, `NUL`, `COM1..9`, `LPT1..9`) rejected.

## 5. Determinism check
Ran `go test ./internal/cli ./internal/fsutil ./internal/check` twice; outputs identical: yes. Evidence: 2 runs pass with identical test counts.

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
Suggested commit message: `feat(windows): add full Windows platform compatibility and dist cross-compilation (T-018)`
Reviewed by: Vikash   Date: 2026-11-02   Reply received: `COMMITTED T-018`
