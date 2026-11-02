# Plan — T-008 P-04: Robust markdown parser in check

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved

## 1. Approach summary
Implement a dedicated standard-library package `internal/markdown` providing a deterministic scanner for markdown documents, replacing fragile string counting in `internal/check`:
1. In `internal/markdown`:
   - Create `parser.go` with a scanner that tracks code fence state (delimiter character backtick or tilde, length >= 3).
   - Implement `CountTableColumns(line string) (int, bool)` which scans a table line, skipping escaped pipes and pipes inside inline code spans, returning the true structural column count.
2. In `internal/check/check.go`:
   - Refactor `checkFences` to use `internal/markdown` to ensure all opened code fences have matching closing fences.
   - Refactor `checkTables` to use `internal/markdown` so table rows with escaped pipes or pipes inside code spans are counted accurately against the header row.
3. Unit tests:
   - `internal/markdown/parser_test.go`: test fence nesting, inline code with pipes, escaped pipes, and indented tables.
   - `internal/check/check_test.go`: test table checking with real markdown containing escaped and code-span pipes.
4. Verify:
   - `make check`, `go test -race ./...`, `GOOS=windows go vet ./...`.

## 2. Decisions (new or affected)
None. (Relies on D-007 standard library purity and D-015 Go 1.27 toolchain).

## 3. Design
- **`internal/markdown/parser.go`**:
  ```go
  type FenceState struct {
      InFence   bool
      Char      rune // '`' or '~'
      Length    int  // e.g. 3, 4, 5
  }

  // CountTableColumns counts structural column pipes in line, ignoring pipes
  // escaped with backslash or inside inline code spans.
  func CountTableColumns(line string) (int, bool)

  // ScanFences validates that all code fences in text are properly opened and closed.
  func ScanFences(text string) []FenceFinding
  ```

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Create `internal/markdown` package with fence and table column scanners | `internal/markdown/parser.go` | Unit tests in `parser_test.go` |
| 2 | Refactor `checkFences` and `checkTables` in `internal/check/check.go` | `internal/check/check.go` | Tests in `check_test.go` |
| 3 | Add table and fence edge case tests | `internal/check/check_test.go` | `go test -v ./internal/check` |
| 4 | Run linter, race detector, Windows cross-vet, and consistency check | Project root | `make check`, `go test -race ./...`, `go run ./cmd/sdd check` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/markdown/parser.go` | create | Deterministic markdown fence and table scanner |
| `internal/markdown/parser_test.go` | create | Unit tests for markdown scanner |
| `internal/check/check.go` | modify | Refactor checkFences and checkTables |
| `internal/check/check_test.go` | modify | Add test cases for tables with pipes in code/escaped |
| `features/T-008-robust-markdown-parser/requirements.md` | modify/review | Requirements specification |
| `features/T-008-robust-markdown-parser/plan.md` | modify/review | Implementation plan |
| `features/T-008-robust-markdown-parser/validation.md` | modify/review | Validation evidence |

## 6. Dependencies
None. Standard library only (D-007).

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Mark T-008 `Awaiting review` then `Done`; update notes |
| `TECH_STACK.md` | Note `internal/markdown` in architecture diagram and package table |
| `SKILLS.md` | Update table formatting note to reflect that escaped pipes and code pipes are now safely parsed |

## 8. Risks, mitigations, rollback
- **Risk:** Edge cases in obscure CommonMark syntax.
  - *Mitigation:* Comprehensive unit tests against standard table and fence formats.
- **Rollback:** `git restore .` reverts to earlier check logic.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active yet.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-02   Phrase received: `APPROVED T-008`
