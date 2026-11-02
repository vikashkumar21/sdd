# Plan — T-009 P-05: Machine-readable JSON output for sdd check

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved

## 1. Approach summary
Implement `--json` output for `sdd check`:
1. In `internal/check/check.go`:
   - Add struct tags (`json:"..."`) to `Finding` and define a JSON representation `JSONReport`:
     ```go
     type Finding struct {
         Severity Severity `json:"severity"`
         File     string   `json:"file"`
         Line     int      `json:"line,omitempty"`
         Message  string   `json:"message"`
     }

     type Summary struct {
         Errors   int `json:"errors"`
         Warnings int `json:"warnings"`
         Notes    int `json:"notes"`
     }

     type JSONReport struct {
         Valid    bool      `json:"valid"`
         Summary  Summary   `json:"summary"`
         Findings []Finding `json:"findings"`
     }

     func (r Report) JSON(strict bool) ([]byte, error)
     ```
2. In `internal/cli/commands.go`:
   - Add `--json` flag to `runCheck`.
   - When `--json` is set, render `rep.JSON(*strict)` and print to `env.Out`.
   - Update `usageText` in `internal/cli/cli.go` to mention `--json`.
3. In `internal/cli/cli_test.go`:
   - Add unit tests verifying `sdd check --json` output structure, exit codes, and `--strict` interactions.
4. Verify:
   - `make check`, `go test -race ./...`, `GOOS=windows go vet ./...`.

## 2. Decisions (new or affected)
None. (Relies on D-007 standard library purity and D-015 Go 1.27 toolchain).

## 3. Design
- **`internal/check/check.go`**:
  ```go
  func (r Report) JSON(strict bool) ([]byte, error) {
      jr := JSONReport{
          Valid: !r.Failed(strict),
          Summary: Summary{
              Errors:   r.Count(Error),
              Warnings: r.Count(Warn),
              Notes:    r.Count(Info),
          },
          Findings: r.Findings,
      }
      return json.MarshalIndent(jr, "", "  ")
  }
  ```
- **`internal/cli/commands.go`**:
  ```go
  jsonOut := fs.Bool("json", false, "output findings as structured JSON")
  ...
  if *jsonOut {
      data, err := rep.JSON(*strict)
      if err != nil {
          fprintf(env.Err, "sdd check: %v\n", err)
          return 1
      }
      fprintln(env.Out, string(data))
      if rep.Failed(*strict) {
          return 1
      }
      return 0
  }
  ```

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Add `Finding` JSON tags, `Summary`, `JSONReport`, and `Report.JSON` | `internal/check/check.go` | Unit test in `check_test.go` |
| 2 | Add `--json` flag and handler to `runCheck` | `internal/cli/commands.go`, `internal/cli/cli.go` | CLI tests in `cli_test.go` |
| 3 | Add CLI integration tests for JSON output | `internal/cli/cli_test.go` | `go test -v -run TestCheckJSON ./internal/cli` |
| 4 | Run linter, race detector, Windows cross-vet, and consistency check | Project root | `make check`, `go test -race ./...`, `go run ./cmd/sdd check` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/check/check.go` | modify | Add JSON tags and Report.JSON method |
| `internal/check/check_test.go` | modify | Test JSON serialization of check reports |
| `internal/cli/commands.go` | modify | Support --json flag in runCheck |
| `internal/cli/cli.go` | modify | Update check command usage documentation |
| `internal/cli/cli_test.go` | modify | Add tests for sdd check --json |
| `features/T-009-check-json-output/requirements.md` | modify/review | Requirements specification |
| `features/T-009-check-json-output/plan.md` | modify/review | Implementation plan |
| `features/T-009-check-json-output/validation.md` | modify/review | Validation evidence |

## 6. Dependencies
None. Standard library only (`encoding/json`, D-007).

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Mark T-009 `Awaiting review` then `Done`; update notes |
| `README.md` | Document `sdd check --json` under CLI usage |

## 8. Risks, mitigations, rollback
- **Risk:** Existing plain text output format changes.
  - *Mitigation:* Plain text output path is completely untouched unless `--json` is explicitly passed.
- **Rollback:** `git restore .` reverts to plain text only.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active yet.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-02   Phrase received: `APPROVED T-009`
