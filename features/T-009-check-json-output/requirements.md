# Requirements — T-009 P-05: Machine-readable JSON output for sdd check

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-009

## 1. Goal
Provide a `--json` flag for `sdd check` that outputs structured, machine-readable JSON containing check summary counts, status, and an itemized array of findings so CI/CD pipelines, pre-commit hooks, IDE plugins, and autonomous AI agents can parse check results deterministically without brittle regex scraping.

## 2. Context and references
- Mission / principles touched: MISSION §5 (Deterministic, Small and shippable), MISSION §7 (S1).
- Proposals addressed: `P-05` in `features/T-005-independent-review/review.md`.
- Decisions relied on: D-007 (Standard library only), D-015 (Approved: Go 1.27 toolchain).
- Depends on: T-002 (Done).

## 3. Scope
**In scope**
- `internal/check`:
  - Add JSON serialization support for `check.Finding` and `check.Report` with clean field tags:
    - `valid`: boolean (true if 0 errors and, under strict mode, 0 warnings).
    - `summary`: object with `errors`, `warnings`, `notes`.
    - `findings`: array of objects each with `severity`, `file`, `line`, `message`.
- `internal/cli`:
  - Add `--json` flag to `sdd check`.
  - When `--json` is set, suppress plain text output and emit indented JSON to `stdout` (`env.Out`).
  - Maintain identical exit codes (0 for pass, 1 for failures/findings under strict, 2 for usage errors).
- Unit and integration tests:
  - Test `sdd check --json` against a valid repository (passes, returns valid JSON, exit 0).
  - Test `sdd check --json` against a repository with findings (returns valid JSON with findings list, exit 1).
  - Test combining `--json` with `--strict`.

**Out of scope**
- Adding JSON output to other commands (such as `sdd init` or `sdd feature`).
- Adding third-party JSON libraries (standard library `encoding/json` only).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | `sdd check` SHALL support a `--json` boolean flag. | MUST | CLI flag check |
| R-002 | When `--json` is passed, `sdd check` SHALL output valid JSON to stdout matching schema: `valid`, `summary`, and `findings`. | MUST | test: `TestCheckJSONOutputValid` |
| R-003 | Each item in `findings` SHALL include `severity`, `file`, `line`, and `message`. | MUST | test: `TestCheckJSONFindingFields` |
| R-004 | When `--json` and `--strict` are both passed, `valid` SHALL be false if warnings are present, and exit code SHALL be 1. | MUST | test: `TestCheckJSONStrictMode` |
| R-005 | When `--json` is not passed, `sdd check` output SHALL remain identical to current human-readable plain text format. | MUST | test: `TestCheckHumanReadableDefault` |
| R-006 | `sdd check --json` SHALL exit 0 when checks pass and 1 when checks fail (preserving existing exit code semantics). | MUST | CLI exit code test |
| R-007 | `make check`, `go test -race ./...`, and `GOOS=windows go vet ./...` SHALL pass with zero issues. | MUST | command check |
| R-008 | All new tests SHALL be deterministic and repeatable across successive runs. | MUST | test: `go test ./internal/check ./internal/cli -count=2` |

## 5. Non-functional requirements
- **Determinism:** Findings in JSON output are sorted canonically by file, line number, and message.
- **Fail-safe:** If an unhandled filesystem error occurs during check, exit code is 1 and error is reported.

## 6. Constraints and assumptions
- Uses standard library `encoding/json` (D-007).

## 7. Open questions
None.

## 8. Acceptance criteria
- Running `sdd check --json` produces valid JSON that parses cleanly with `json.Unmarshal`.
- In CI scripts, `sdd check --json | jq .valid` evaluates to `true` or `false`.
