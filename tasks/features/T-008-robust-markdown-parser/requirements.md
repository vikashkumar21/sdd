# Requirements — T-008 P-04: Robust markdown parser in check

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-008

## 1. Goal
Replace simplistic line-based string counting in `sdd check` with a deterministic, standard-library state machine parser that accurately handles nested code fences (e.g. 4+ backticks, tildes `~~~`), escaped table pipes (`\|`), inline code spans with pipes (` `|` `), and indented tables, preventing false-positive table formatting errors.

## 2. Context and references
- Mission / principles touched: MISSION §5 (Deterministic, Fail safe, Small and shippable), MISSION §7 (S1).
- Proposals addressed: `P-04` in `features/T-005-independent-review/review.md`.
- Decisions relied on: D-007 (Standard library only), D-015 (Approved: Go 1.27 toolchain).
- Depends on: T-002 (Done).

## 3. Scope
**In scope**
- `internal/markdown` package:
  - Deterministic state machine / scanner for Markdown documents.
  - Tracking code fences: supports backticks (``` or ````+) and tildes (~~~ or ~~~~+), matching opening fence length to closing fence length.
  - Tracking inline code spans: ignores pipes inside single/double backticks (`|` or ``|``).
  - Tracking escaped characters: recognizes `\|` as an escaped literal pipe rather than a column delimiter.
  - Table parsing: supports optional leading whitespace (indented tables up to 3 spaces or inside lists), header row, delimiter row (`|---|---|`), and data rows.
- `internal/check`:
  - Refactor `checkFences` and `checkTables` to use the robust `internal/markdown` scanner.
  - Accurately count unescaped structural table columns while permitting literal pipes inside backticks or escaped with a backslash.
- Unit and integration tests:
  - Tests covering nested code fences (e.g. carousel snippets with 4 backticks containing 3-backtick blocks).
  - Tests covering tables containing `\|` and `|` inside code spans.
  - Tests ensuring unbalanced fences and genuine malformed tables are still correctly flagged as errors.

**Out of scope**
- Third-party markdown parsing libraries (goldmark, blackfriday) — standard library only (D-007).
- Full CommonMark AST generation (we only need structural tokenization for fences, inline code, and table boundaries).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | `internal/markdown` SHALL parse code fences with varying lengths (3, 4, 5+ backticks or tildes) and require matching closing delimiters. | MUST | test: `TestCodeFences` |
| R-002 | `internal/markdown` SHALL support nested code fences where an outer fence of length N contains inner fences of length < N. | MUST | test: `TestNestedCodeFences` |
| R-003 | `internal/markdown` SHALL recognize inline code spans (single or multi-backtick) outside code fences and ignore pipe characters within them. | MUST | test: `TestInlineCodePipes` |
| R-004 | `internal/markdown` SHALL recognize escaped pipes (backslash-pipe) as non-delimiter literal characters. | MUST | test: `TestEscapedPipes` |
| R-005 | `internal/check` SHALL use `internal/markdown` to validate table column counts without false positives on inline code pipes or escaped pipes. | MUST | test: `TestCheckTablesWithPipesInCode` |
| R-006 | `internal/check` SHALL continue to report errors for genuine column mismatches and unbalanced fences. | MUST | test: `TestCheckDetectsGenuineErrors` |
| R-007 | `make check`, `go test -race ./...`, and `GOOS=windows go vet ./...` SHALL pass with zero issues. | MUST | command check |
| R-008 | All new tests SHALL be deterministic and repeatable across successive runs. | MUST | test: `go test ./internal/markdown ./internal/check -count=2` |

## 5. Non-functional requirements
- **Determinism:** State machine is 100% deterministic with single-pass line/character scanning.
- **Fail-safe:** Malformed markdown never causes panics or infinite loops.
- **Performance:** Processes documents linearly in $O(N)$ time with minimal allocations.

## 6. Constraints and assumptions
- Uses Go standard library only (D-007).

## 7. Open questions
None.

## 8. Acceptance criteria
- Given a markdown file with a table cell containing ` `grep | wc -l` ` or `\|`, `sdd check` passes without reporting a pipe mismatch.
- Given a markdown file with a 4-backtick code block enclosing a 3-backtick markdown snippet, `sdd check` passes without reporting an unbalanced fence.
- Given genuine syntax errors, `sdd check` reports them accurately with line numbers.
