# Validation — T-001 Project skeleton

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Not run

## 0. Environment (fill in when running)

| Item | Value |
|---|---|
| OS | |
| `go version` | |
| `golangci-lint --version` | |
| `make --version` (first line) | |
| Run by (human / local agent) | |

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | inspection | `cat go.mod`; `go version`; `go list -m` | module path equals the D-030 value; `go 1.27`; go1.27.x | | NOT RUN |
| R-002 | test | `go build ./...` | exit 0, no output | | NOT RUN |
| R-003 | test | `go test ./cmd/dbguard -run TestRun -count=1` (case: `version`) | stdout `dbguard <version>\n`; stderr empty; code 0 | | NOT RUN |
| R-004 | demo | `make build VERSION=1.2.3 && bin/dbguard version` | stdout `dbguard 1.2.3` | | NOT RUN |
| R-005 | test | `go test ./cmd/dbguard -run TestRun -count=1` (cases: no args, unknown, `version extra`) | stdout empty; stderr has usage; code 2 | | NOT RUN |
| R-006 | inspection and test | read `main.go`; the tests call `run` directly | `main` only calls `run` and exits | | NOT RUN |
| R-007 | test | `make build test vet fmt-check lint check clean` (each) | each target exists and exits 0; `check` order is fmt-check, vet, lint, test | | NOT RUN |
| R-008 | inspection | read `.golangci.yml` | matches `plan.md` §3 | | NOT RUN |
| R-009 | test | `make lint`; `make fmt-check`; then a temporary mis-formatted file and `make fmt-check` | first two exit 0; the third exits non-zero | | NOT RUN |
| R-010 | test | `PATH=/usr/bin:/bin make lint` | clear "golangci-lint not installed" style message, non-zero exit | | NOT RUN |
| R-011 | inspection | `cat go.mod`; `ls go.sum` | no `require` lines; no `go.sum` | | NOT RUN |
| R-012 | inspection | `grep -rn "^var " --include=*.go .` | only `internal/version.Version`, with a reasoned `//nolint:gochecknoglobals` | | NOT RUN |
| R-013 | inspection | read `TECH_STACK.md` §4, §10, §11 | D-016 to D-019 and D-030 Approved with values; layout matches | | NOT RUN |
| R-014 | inspection | read README "Development" | prerequisites and `make build`, `make check` listed | | NOT RUN |

## 2. Commands run
Record the exact command and its outcome.

| Command | Result |
|---|---|
| `go build ./...` | |
| `go vet ./...` | |
| `go test ./... -count=1` (run 1) | |
| `go test ./... -count=1` (run 2) | |
| `make check` | |

## 3. Manual checks
- <step → observed result>

## 4. Negative and fail-closed cases
- Unknown subcommand: stdout empty, exit 2.
- `version` with an extra argument: exit 2.
- Mis-formatted file: `make fmt-check` fails.
- `golangci-lint` missing: `make lint` prints a clear message.

## 5. Determinism check
Ran `make test` twice; outputs identical: yes / no. Evidence: <paste>

## 6. Deviations, known limitations, and feedback-rule compliance
<None, or list. Any deviation from the approved plan must have been re-approved.>

Feedback-rule compliance (SKILLS.md §12): none active.

## 7. Definition of done
- [ ] Every requirement has evidence
- [ ] Build, vet, lint, tests pass (commands above)
- [ ] Negative and fail-closed cases tested
- [ ] Determinism confirmed
- [ ] No files changed outside the plan; no unapproved dependencies
- [ ] Living docs updated
- [ ] Compliant with every active SKILLS.md §12 rule
- [ ] Suggested commit message provided

## 8. Sign-off
Suggested commit message: `chore: add Go module, minimal CLI skeleton, Makefile, and lint config`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-001`
