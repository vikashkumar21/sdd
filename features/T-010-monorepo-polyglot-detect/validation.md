# Validation — T-010 P-07: Monorepo and polyglot detection enhancement

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Complete
Roadmap ref: `ROADMAP.md` T-010

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/detect -run TestDetectMonorepoDepth` | Manifests at depth 1-4 discovered; depth > 4 ignored | `PASS: TestDetectMonorepoDepth (0.00s)` | PASS |
| R-002 | test | `go test -v ./internal/detect -run TestDetectNestedPackageJSON` | Nested TypeScript dependencies and npm scripts detected | `PASS: TestDetectNestedPackageJSON (0.00s)` | PASS |
| R-003 | test | `go test -v ./internal/detect -run TestDetectPrunesHeavyDirs` | Pruned dirs skipped via SkipDir | `PASS: TestDetectPrunesHeavyDirs (0.00s)` | PASS |
| R-004 | test | `go test -v ./internal/detect -run TestDetectPolyglotCommands` | Multi-language commands joined by ` && ` in canonical order | `PASS: TestDetectPolyglotCommands (0.00s)` | PASS |
| R-005 | test | `go test -v ./internal/detect -run TestDetectMakefilePrecedence` | Makefile targets take precedence over language commands | `PASS: TestDetectMakefilePrecedence (0.00s)` | PASS |
| R-006 | test | `go test -v ./internal/detect -run TestDetectCommandTableSafety` | No pipe or backtick characters present in combined commands | `PASS: TestDetectCommandTableSafety (0.00s)` | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | PASS (gofmt clean, vet clean, all packages passed) |
| `make check` | PASS (golangci-lint 0 issues, all tests passed) |
| `go run ./cmd/sdd check` | PASS (0 error(s), 0 warning(s)) |

## 3. Manual checks
- Ran `TestDetectPolyglotCommands`: combined `go build ./... && npm --prefix apps/web run build` and `go test ./... && pytest && npm --prefix apps/web test`.
- Ran `TestDetectPrunesHeavyDirs`: verified `.turbo`, `.pnpm-store`, `bower_components`, `.cache`, `.terraform`, `.yarn` are skipped without counting files.

## 4. Negative and failure cases
- Depth 5 manifests (`a/b/c/d/e/Cargo.toml`) ignored (verified by `TestDetectMonorepoDepth`).
- Unsafe script values containing `|` or backticks rejected (verified by `TestDetectCommandTableSafety`).
- Non-directory paths passed to `Detect` return error (verified by `TestDetectNotADirectory`).

## 5. Determinism check
Ran `go test -v ./internal/detect -count=1` twice:
- Pass 1: `ok github.com/vikashkumar21/sdd/internal/detect 0.098s`
- Pass 2: `ok github.com/vikashkumar21/sdd/internal/detect 0.095s`
Outputs are identical. Determinism confirmed.

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
Suggested commit message: `feat(detect): enhance monorepo manifest depth and polyglot command suggestions (T-010)`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-010`
