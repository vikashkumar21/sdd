# Plan — T-002 Verify on the maintainer's Mac: Go 1.27, golangci-lint v2, macOS binary; fix findings (P-01)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Implemented

## 1. Approach summary
Execute Step 10 and Proposal P-01 from the senior Go review:
1. Update `go.mod` directive from `go 1.22` to `go 1.27` in accordance with approved Decision D-015.
2. Resolve all 12 `golangci-lint` issues:
   - `internal/check/check_test.go`: Fix line 106 identical expressions comparison bug (`dump(run(t, root)) != dump(run(t, root))`) by storing results of consecutive runs in separate variables and asserting equality.
   - `internal/detect/detect_test.go`: Simplify lines 69 and 72 boolean conditions per De Morgan's law (`!a || !b`).
   - `internal/cli/cli.go` and `internal/cli/commands.go`: Check or handle errors from `fmt.Fprint*` on `env.Out` and `env.Err` (or write helper `fprint(w, text)`).
3. Validate `make check` (`fmt-check`, `vet`, `lint`, `test`) locally on macOS.
4. Run `make dist`, verify the multi-platform static binary artifacts and SHA256SUMS, and execute `dist/sdd_darwin_arm64 version`.
5. Execute the full matrix of verification scenarios in temporary throwaway directories (`os.TempDir()`).
6. Update living documentation (`TECH_STACK.md` D-004 status, `ROADMAP.md`).

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-004 | Build, test, lint, and format toolchain | A) Approve current Makefile + golangci-lint v2 toolchain; B) Change toolchain | Option A (Approve Makefile targets `build`, `test`, `vet`, `fmt`, `lint`, `verify`, `check`, `dist`) | Proposed (Approve at GATE 2 upon successful `make check`) |
| D-015 | Go language version in `go.mod` | A) 1.22; B) 1.24; C) 1.27 | Option C (1.27) | **Approved** 2026-10-29 |

## 3. Design
- **`internal/check/check_test.go`**:
  Refactor test `TestCheckIsDeterministic`:
  ```go
  first := dump(run(t, root))
  second := dump(run(t, root))
  if first != second {
      t.Fatalf("check output not deterministic:\n--- first ---\n%s\n--- second ---\n%s", first, second)
  }
  ```
- **`internal/detect/detect_test.go`**:
  Apply De Morgan's laws to simplify length and equality checks on lines 69 and 72.
- **`internal/cli` Error Handling**:
  Wrap or check `fmt.Fprintf` and `fmt.Fprint` calls to satisfy `errcheck`, ensuring all I/O errors to stdout/stderr are either ignored explicitly via `_ = ...` or handled.

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Update `go.mod` directive to 1.27 | `go.mod` | `go version -m` or `head -n 3 go.mod` |
| 2 | Fix test bugs and simplifications | `internal/check/check_test.go`, `internal/detect/detect_test.go` | `go test ./internal/check ./internal/detect` |
| 3 | Resolve `errcheck` findings in CLI | `internal/cli/cli.go`, `internal/cli/commands.go` | `golangci-lint run` |
| 4 | Run complete local verification suite | Project root | `make check` (must exit 0) |
| 5 | Build distribution artifacts | `Makefile`, `dist/` | `make dist`, `./dist/sdd_darwin_arm64 version` |
| 6 | Execute project-type matrix in scratch dirs | Temporary scratch directory | Automated runner script verifying greenfield/brownfield in Go, Java, Python, TS |
| 7 | Update living documentation | `TECH_STACK.md`, `ROADMAP.md` | `sdd check` passes |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `go.mod` | modify | Update language version to 1.27 (D-015) |
| `internal/check/check_test.go` | modify | Fix SA4000 identical expressions bug in determinism test |
| `internal/detect/detect_test.go` | modify | Fix QF1001 boolean simplification warnings |
| `internal/cli/cli.go` | modify | Fix errcheck warnings on usage and version output |
| `internal/cli/commands.go` | modify | Fix errcheck warnings on error messages and prompt output |
| `features/T-002-mac-verification/requirements.md` | modify/review | Requirement specifications |
| `features/T-002-mac-verification/plan.md` | modify/review | Step-by-step implementation plan |
| `features/T-002-mac-verification/validation.md` | modify/review | Validation evidence and execution logs |

## 6. Dependencies
None. Standard library only (D-007).

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Mark T-002 `Awaiting review` then `Done`; update notes |
| `TECH_STACK.md` | Approve D-004 upon clean `make check` |
| `README.md` | Update status note to reflect verified macOS / Go 1.27 support |

## 8. Risks, mitigations, rollback
- **Risk:** Updating `go.mod` to 1.27 breaks older Go environments (<1.27).
  - *Mitigation:* Explicitly decided and approved by the maintainer in D-015 (matches host environment `go1.27.3`).
- **Risk:** Silencing `errcheck` on stdout/stderr obscures write errors.
  - *Mitigation:* CLI exit codes already communicate command success or failure; checking or cleanly ignoring pipe write errors to stdout/stderr is standard CLI practice in Go.
- **Rollback:** Revert git working tree changes with `git restore .`.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active yet.

## 10. Approval record
Approved by: Vikash   Date: 2026-10-29   Phrase received: `APPROVED T-002`
