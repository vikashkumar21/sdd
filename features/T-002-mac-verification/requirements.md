# Requirements — T-002 Verify on the maintainer's Mac: Go 1.27, golangci-lint v2, macOS binary; fix findings (P-01)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-10-29)
Roadmap ref: `ROADMAP.md` T-002

## 1. Goal
Upgrade the Go toolchain directive in `go.mod` to Go 1.27 per approved Decision D-015, fix all 12 `golangci-lint` issues (Proposal P-01), verify that `make check` passes cleanly on macOS (`darwin/arm64`), test cross-compilation and local execution of the macOS binary, and record empirical evidence for the full verification suite in temporary scratch directories.

## 2. Context and references
- Mission principles touched: MISSION §5 (Deterministic, Fail safe, Small and shippable), MISSION §7 (S1–S5).
- Decisions relied on: D-004 (Build, test, lint, format toolchain), D-007 (Standard library only), D-015 (Approved: Go 1.27).
- Depends on: T-001 (Done / committed), T-005 (Done).

## 3. Scope
**In scope**
- Update `go.mod` directive from `go 1.22` to `go 1.27` per D-015.
- Fix all 12 `golangci-lint` findings (P-01):
  - `internal/check/check_test.go`: Fix `SA4000` identical expressions comparison bug.
  - `internal/detect/detect_test.go`: Fix `QF1001` boolean logic warnings.
  - `internal/cli/cli.go` & `internal/cli/commands.go`: Handle or silence `errcheck` warnings for standard I/O prints.
- Verify that `make check` (`fmt-check`, `vet`, `lint`, `test`) passes locally with exit code 0.
- Verify `make dist` generates static binaries for `darwin/arm64`, `darwin/amd64`, `linux/amd64`, `linux/arm64` and valid `SHA256SUMS`.
- Execute the compiled macOS native binary (`dist/sdd_darwin_arm64`) and verify version output.
- Check `which sdd` to confirm no shadowing binary.
- Run the full verification suite in temporary scratch directories (greenfield/brownfield matrix across Go, Java, Python, TypeScript, refuels, dry-runs, permission failures, symlinks, spaces, CRLF).
- Confirm determinism by running tests and scenario generation twice with byte-for-byte identical output.

**Out of scope**
- Adding any third-party dependency (standard library only per D-007).
- Modifying project code unrelated to linter findings and Go 1.27 directive.
- Implementing later proposals (P-02 through P-10).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | `go.mod` SHALL declare `go 1.27` in accordance with approved Decision D-015. | MUST | inspection: `go.mod` |
| R-002 | The codebase SHALL pass `golangci-lint run` (v2.15.0) with zero errors and zero warnings, resolving all 12 issues from P-01. | MUST | command: `golangci-lint run` |
| R-003 | The full test suite SHALL pass with race detector enabled under Go 1.27 (`go test -race ./...`). | MUST | command: `go test -race ./...` |
| R-004 | `make check` SHALL pass with exit code 0 on the maintainer's Mac. | MUST | command: `make check` |
| R-005 | `make dist` SHALL successfully produce static binaries for darwin/arm64, darwin/amd64, linux/amd64, linux/arm64 and a SHA256SUMS file. | MUST | command: `make dist`, inspection of `dist/` |
| R-006 | The compiled native macOS binary `dist/sdd_darwin_arm64` SHALL execute and output version information matching `version.Version`. | MUST | command: `./dist/sdd_darwin_arm64 version` |
| R-007 | All CLI operations tested across project types (greenfield/brownfield in Go, Java, Python, TS, hand-made refusal, dry-run, force, template override, spaces, symlinks, read-only dirs) SHALL execute correctly in scratch temporary directories. | MUST | test & demo: recorded commands in `validation.md` |
| R-008 | All tests and scratch generation scenarios SHALL be deterministic (identical output across two successive runs). | MUST | test: comparison of successive runs in `validation.md` |

## 5. Non-functional requirements
- **Determinism:** Repeatable results across repeated executions.
- **Fail-safe:** Clear failure on invalid inputs or missing permissions; no partial file writes.
- **Security:** Standard library only; no external network access during build or test.

## 6. Constraints and assumptions
- Host is macOS (`darwin/arm64`) running Apple silicon.
- Host toolchains verified: `go version go1.27.3 darwin/arm64`, `golangci-lint 2.15.0`.

## 7. Open questions
None. (D-015 is Approved; D-004 resolution is proposed in `plan.md`.)

## 8. Acceptance criteria
- Given the repository on macOS, when `make check` runs, it exits 0 with no lint, vet, formatting, or test failures.
- Given `go.mod`, it specifies `go 1.27`.
- Given the built native binary `dist/sdd_darwin_arm64`, running `./dist/sdd_darwin_arm64 version` exits 0 and prints `sdd 0.1.0 (templates 1.0.0)`.
