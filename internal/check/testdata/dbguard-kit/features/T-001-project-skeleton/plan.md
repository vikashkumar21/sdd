# Plan — T-001 Project skeleton

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Draft — blocked on D-030

## 1. Approach summary
A standard-library-only skeleton in three small pieces: a testable `run` function behind `main`, an `internal/version` package for build-time injection, and tooling (Makefile plus a golangci-lint v2 configuration) that makes `make check` the single quality gate. No features and no dependencies.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-012 | Go language version | 1.27 | 1.27 (human-stated) | **Approved** |
| D-030 | Go module path | A) `github.com/<handle>/dbguard`; B) `github.com/<org>/dbguard` | A, with the handle you will actually host the repository under | **Open — needs your reply** |
| D-016 | Logging convention | A) `log/slog` to stderr or a file; stdout reserved for the protocol (no code in T-001); B) decide later | A | Proposed → approve at GATE 1 |
| D-017 | Lint and format toolchain | A) golangci-lint v2: formatters gofmt and goimports, `standard` linters plus `errorlint`, `gochecknoinits`, `gochecknoglobals`; B) plain `gofmt` and `go vet` only | A (config schema verified against the golangci-lint docs; see `TECH_STACK.md` §12) | Proposed → approve at GATE 1 |
| D-018 | Commit convention | A) Conventional Commits; B) free-form | A | Proposed → approve at GATE 1 |
| D-019 | Test conventions | A) stdlib `testing`, table-driven, `integration` build tag later, `go-cmp` deferred until a task needs it; B) add `go-cmp` now | A (keeps T-001 dependency-free) | Proposed → approve at GATE 1 |

```
DECISION NEEDED: D-030 — Go module path
Context:        go.mod needs its final module path; changing it later touches every import.
Options:        A) github.com/<your-github-handle>/dbguard   B) github.com/<org>/dbguard
Recommendation: A, using the handle you will actually host the repository under.
Blocks:         T-001 GATE 1 (R-001, R-004, R-008 local-prefixes).
Reply with:     DECIDE D-030: <exact module path>
```

## 3. Design (signatures and behaviour only)

```
cmd/dbguard/main.go
    package main
    func main()                                                  // os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
    func run(args []string, stdout, stderr io.Writer) int

internal/version/version.go
    package version
    var Version = "dev"   //nolint:gochecknoglobals // set at build time via -ldflags
```

**Behaviour of `run`**

| Arguments | stdout | stderr | Exit code |
|---|---|---|---|
| `version` | `dbguard <version>` and newline | empty | 0 |
| none | empty | usage | 2 |
| unknown command | empty | `dbguard: unknown command "<x>"`, blank line, usage | 2 |
| `version` plus any extra argument | empty | usage | 2 |

Usage text:

```
usage: dbguard <command>

commands:
  version    print the dbguard version
```

**Makefile semantics** (targets are `.PHONY`; `.NOTPARALLEL` guarantees the order of `check`)

| Target | Does |
|---|---|
| `build` | `go build -trimpath -ldflags "-X $(MODULE)/internal/version.Version=$(VERSION)" -o bin/dbguard ./cmd/dbguard`; `VERSION` defaults to `dev` |
| `test` | `go test ./...` |
| `vet` | `go vet ./...` |
| `fmt` | `golangci-lint fmt` |
| `fmt-check` | `golangci-lint fmt --diff` (non-zero when code is not formatted) |
| `lint` | `golangci-lint run`, after a guard that prints a clear message if `golangci-lint` is missing |
| `check` | `fmt-check`, `vet`, `lint`, `test`, in that order |
| `clean` | `rm -rf bin` |

**`.golangci.yml`**

```yaml
version: "2"
linters:
  default: standard
  enable:
    - errorlint
    - gochecknoinits
    - gochecknoglobals
formatters:
  enable:
    - gofmt
    - goimports
  settings:
    goimports:
      local-prefixes:
        - <module path from D-030>
```

## 4. Implementation steps
Each step is small enough to review in one sitting.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Create the module with `go mod init <D-030 path>`; make sure the `go` directive is 1.27 | `go.mod` | `cat go.mod`; `go version`; `go list -m` |
| 2 | Add the version package | `internal/version/version.go` | `go build ./...` |
| 3 | Add `main` and `run` | `cmd/dbguard/main.go` | `go build ./...`; `go run ./cmd/dbguard version` |
| 4 | Add table-driven tests for every row of the behaviour table, asserting stdout, stderr, and exit code (expected version string read from `version.Version`, never hard-coded) | `cmd/dbguard/main_test.go` | `go test ./... -count=1`, run twice |
| 5 | Add the Makefile | `Makefile` | `make build`; `bin/dbguard version`; `make build VERSION=1.2.3` then `bin/dbguard version` |
| 6 | Add the lint configuration | `.golangci.yml` | `make lint` |
| 7 | Run the negative checks: a temporary mis-formatted file must fail `make fmt-check`; with `golangci-lint` off the `PATH`, `make lint` must print the clear message; remove the temporary file afterwards | none (temporary) | `make fmt-check` exit code; `PATH=/usr/bin:/bin make lint` output |
| 8 | Update the living docs (§7) | listed in §5 | inspection |
| 9 | Run `make check` twice; paste the evidence into `validation.md` | `validation.md` | `make check` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `go.mod` | create | R-001, R-011 |
| `internal/version/version.go` | create | R-004, R-012 |
| `cmd/dbguard/main.go` | create | R-003, R-005, R-006 |
| `cmd/dbguard/main_test.go` | create | R-003, R-005, R-006 |
| `Makefile` | create | R-004, R-007, R-009, R-010 |
| `.golangci.yml` | create | R-008, R-009, R-012 |
| `TECH_STACK.md` | modify | R-013 |
| `ROADMAP.md` | modify | status and notes |
| `README.md` | modify | R-014 |
| `SKILLS.md` | modify | pitfalls and conventions learned |
| `features/T-001-project-skeleton/*.md` | modify | fill in evidence |

`.gitignore` already ignores `bin/`; no change expected.

## 6. Dependencies
None in the module (standard library only). Developer tools, not module dependencies: Go 1.27, `make`, golangci-lint v2 (installed version recorded in `validation.md`).

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | T-001 status; actual hours; note any re-estimate |
| `TECH_STACK.md` | D-016, D-017, D-018, D-019, D-030 to Approved with values; §4 layout; §10 conventions; record the installed golangci-lint version in §12 |
| `SKILLS.md` | any pitfall found (for example linter-version behaviour) |
| `README.md` | Development prerequisites and commands |

## 8. Risks, mitigations, rollback
- **golangci-lint version drift** (config schema or linter names differ by version). Mitigation: record the installed version; fail loudly; pinning is deferred to the CI task (T-017).
- **golangci-lint built with an older Go than the module's `go` directive** is known to refuse to run (from memory; verify). Mitigation: install a build made with Go 1.27 or newer, and note the version in `validation.md`.
- **`gochecknoglobals` may flag future legitimate patterns.** Mitigation: any exception needs a narrow `//nolint` with a reason and human approval; otherwise revisit D-017.
- **Go 1.27 is not verified by the agent.** Mitigation: R-001 evidence is the output of `go version`.
- **The sandbox has no Go toolchain.** Mitigation: verification runs on the human's machine; unrun checks stay **NOT VERIFIED**.
- **Rollback:** `git revert` the commit, or delete the files in §5. There are no external side effects.

## 9. Applicable human-feedback rules (SKILLS.md §12)
None active yet (the register is empty).

## 10. Approval record
Approved by: ______   Date: ______   Phrase received: `APPROVED T-001`
