# Requirements — T-001 Project skeleton

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Draft — blocked on D-030 (GitHub handle for the module path)
Roadmap ref: `ROADMAP.md` T-001

## 1. Goal
Create the Go module and a minimal, fully verifiable project skeleton, so every later task starts from a green `make check` and from agreed conventions.

## 2. Context and references
- Mission principles touched: MISSION §5 (deterministic, boring technology, small and shippable).
- Decisions relied on: D-003 (Go), D-012 (Go 1.27, Approved), D-030 (module path, **Open**), D-016 (logging convention), D-017 (lint and format toolchain), D-018 (commit convention), D-019 (test conventions).
- Verified facts used: `TECH_STACK.md` §12 (golangci-lint v2 configuration and `fmt` command).
- Depends on: T-000 (Done), T-000.1 (must be committed first).

## 3. Scope
**In scope**
- `go.mod` (module path per D-030, language version 1.27 per D-012).
- `cmd/dbguard` with a `version` subcommand only.
- `internal/version` (build-time version, default `dev`).
- A table-driven test for the CLI entry logic.
- `Makefile` with `build`, `test`, `vet`, `fmt`, `fmt-check`, `lint`, `check`, `clean`.
- `.golangci.yml` in golangci-lint v2 format.
- Recording D-016 to D-019 and D-030 as Approved in `TECH_STACK.md`, and updating the package layout and conventions there.
- README "Development" section update.

**Out of scope**
- Any dbguard feature code (parsing, classification, policy, MCP, Postgres, SQLite).
- Any third-party dependency: the module is standard-library-only in this task.
- CI configuration, docker-compose, release tooling (T-015, T-017).
- A CLI framework (D-015, T-014) and help output beyond the usage error.
- Logging code: there is nothing to log yet; D-016 only fixes the convention.

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | `go.mod` SHALL declare the module path decided in D-030 and Go language version 1.27. | MUST | inspection of `go.mod`; `go version` reports go1.27.x |
| R-002 | `go build ./...` SHALL succeed with no errors or warnings. | MUST | test: command |
| R-003 | `dbguard version` SHALL write exactly `dbguard <version>` and a newline to stdout, write nothing to stderr, and exit with code 0, where `<version>` defaults to `dev`. | MUST | test (unit) |
| R-004 | The version SHALL be overridable at build time with `-ldflags "-X <module>/internal/version.Version=<value>"`, and `make build VERSION=<value>` SHALL apply it. | MUST | demo: build with `VERSION=1.2.3`, run, observe |
| R-005 | `dbguard` with no arguments, with an unknown subcommand, or with extra arguments after `version`, SHALL write a usage message to stderr, write nothing to stdout, and exit with code 2. | MUST | test (unit) |
| R-006 | The CLI logic SHALL be a function `run(args []string, stdout, stderr io.Writer) int` that never calls `os.Exit`; `main` SHALL only call it and exit with its result. | MUST | inspection and test |
| R-007 | The Makefile SHALL provide the targets `build`, `test`, `vet`, `fmt`, `fmt-check`, `lint`, `check`, `clean`. `build` SHALL write `bin/dbguard`. `check` SHALL run `fmt-check`, `vet`, `lint`, `test` in that order and stop at the first failure. | MUST | test: commands |
| R-008 | `.golangci.yml` SHALL use `version: "2"`, enable the formatters `gofmt` and `goimports`, use the `standard` default linter set, and additionally enable `errorlint`, `gochecknoinits`, `gochecknoglobals`. | MUST | inspection and `make lint` |
| R-009 | `make lint` and `make fmt-check` SHALL pass on the skeleton, and `make fmt-check` SHALL fail on a deliberately mis-formatted file. | MUST | test: commands including the negative case |
| R-010 | `make lint` SHALL fail with a clear message, not a raw shell error, when `golangci-lint` is not installed. | SHOULD | test: negative case |
| R-011 | The module SHALL have no dependencies beyond the Go standard library (no `require` lines, no `go.sum`). | MUST | inspection |
| R-012 | `internal/version.Version` SHALL be the only package-level mutable variable in the repository and SHALL carry a `//nolint:gochecknoglobals` directive with a stated reason. | MUST | inspection and `make lint` |
| R-013 | `TECH_STACK.md` SHALL record D-016, D-017, D-018, D-019, D-030 as Approved with the chosen values, and its package layout (§4) and conventions (§10) SHALL match the skeleton. | MUST | inspection |
| R-014 | The README "Development" section SHALL list the prerequisites and the commands `make build` and `make check`. | SHOULD | inspection |

## 5. Non-functional requirements
- **Determinism:** `make test` run twice gives identical results; tests use no time, randomness, or environment state.
- **Fail closed:** an unknown subcommand is an error exit, never a default action.
- **stdout discipline (D-016):** only a successful `version` writes to stdout.
- **Offline:** no step requires network access (standard library only; no `go get`).
- **Security and privacy:** nothing sensitive is handled in this task.

## 6. Constraints and assumptions
- Go 1.27 is installed on the human's machine (human-stated; the agent has not verified it).
- `golangci-lint` v2 and `make` are installed locally. They are developer tools, not module dependencies; their versions are recorded in `validation.md`.
- Verification commands run on the human's machine, or by a local agent inside the repository. The agent's sandbox has no Go toolchain, so evidence is pasted into `validation.md`; anything not run stays **NOT VERIFIED**.
- All paths are relative to the repository root `/Users/vikash/projects/dbguard`.

## 7. Open questions
- **OQ-1 → DECISION NEEDED D-030:** the GitHub handle (user or organisation) for the module path `github.com/<handle>/dbguard`. See `plan.md` §2. This must be resolved before GATE 1.

## 8. Acceptance criteria
- Given a clean checkout, when `make check` runs, then it exits 0.
- Given `make build VERSION=1.2.3`, when `bin/dbguard version` runs, then stdout is `dbguard 1.2.3` and the exit code is 0.
- Given any unknown subcommand, when dbguard runs, then stdout is empty, stderr contains the usage text, and the exit code is 2.
- Given a deliberately mis-formatted Go file, when `make fmt-check` runs, then it exits non-zero.
