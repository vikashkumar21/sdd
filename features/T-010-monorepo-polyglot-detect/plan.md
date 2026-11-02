# Plan — T-010 P-07: Monorepo and polyglot detection enhancement

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-02)
Roadmap ref: `ROADMAP.md` T-010

## 1. Approach summary
1. Increase `maxMarkDepth` from 2 to 4 in `internal/detect/detect.go` to scan nested monorepo packages.
2. Expand `ignoredDirs` to prune high-volume directories (`.turbo`, `.pnpm-store`, `bower_components`, `.cache`, `.terraform`, `.yarn`) immediately.
3. Enhance `package.json` scanning to inspect nested packages for TypeScript dependencies and npm scripts (`build`, `test`, `lint`, `format`).
4. Update `suggest()` to collect suggested commands per detected language and join them deterministically with ` && ` in canonical language order (`go`, `java`, `python`, `typescript`), while preserving top-level `Makefile` precedence.
5. Add unit and fixture tests for monorepo depth (depth 1 to 4), directory pruning, and polyglot command composition in `internal/detect/detect_test.go`.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| (none) | No new architectural decisions needed | n/a | Proceed with standard library implementation | n/a |

## 3. Design
- **Constants and Pruning:**
  - `maxMarkDepth = 4` (increased from 2).
  - Add to `ignoredDirs`: `".turbo"`, `".pnpm-store"`, `"bower_components"`, `".cache"`, `".terraform"`, `".yarn"`.
- **Nested Package Scanning:**
  - Track `package.json` files found up to depth 4.
  - Parse nested `package.json` files (or root if present) to detect TypeScript and scripts.
- **Polyglot Command Suggestion:**
  - Collect command candidates per detected language for `build`, `test`, `lint`, and `format`.
  - For each category, if `Makefile` defined a target (`make build`, `make test`, etc.), use the Makefile target.
  - Otherwise, collect non-empty command suggestions across all detected languages in canonical order (`go`, `java`, `python`, `typescript`) and join them with ` && `.
  - Ensure all combined strings contain no newline, backtick, or pipe characters.

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Increase manifest depth to 4 and expand ignoredDirs | `internal/detect/detect.go` | `go test ./internal/detect -run TestDetectLanguages` |
| 2 | Enhance nested package.json inspection and script extraction | `internal/detect/detect.go` | `go test ./internal/detect -run TestDetectNestedPackageJSON` |
| 3 | Refactor suggest() to combine multi-language commands deterministically | `internal/detect/detect.go` | `go test ./internal/detect -run TestDetectPolyglotCommands` |
| 4 | Add tests for monorepo depth 4, directory pruning, and polyglot combos | `internal/detect/detect_test.go` | `go test -v ./internal/detect` |
| 5 | Verify entire suite, formatting, and sdd check | All modified files | `make verify && go run ./cmd/sdd check` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/detect/detect.go` | modify | Increase depth limit, prune cache dirs, inspect nested packages, multi-language suggestions |
| `internal/detect/detect_test.go` | modify | Tests for depth 4, polyglot commands, and directory pruning |
| `ROADMAP.md` | modify | Update T-010 status across workflow gates |
| `TECH_STACK.md` | modify | Record verified fact for monorepo detection |

## 6. Dependencies
None. Standard library only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Update T-010 status |
| `TECH_STACK.md` | Add verified fact for monorepo scan depth and polyglot detection |

## 8. Risks, mitigations, rollback
- **Risk:** Traversing deeper directory trees (depth 4) could degrade performance on very large repos.
  - *Mitigation:* Expanded `ignoredDirs` immediately prunes heavy directories; the `maxVisited = 20000` ceiling remains strictly enforced.
- **Risk:** Joining commands with ` && ` could introduce pipes or invalid characters if script names are malformed.
  - *Mitigation:* Validate command strings before concatenation; reject any containing illegal characters (`|`, backticks, newlines).
- **Rollback:** `git restore internal/detect/` reverts all detection changes.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active.

## 10. Approval record
Approved by: Vikash   Date: 2026-11-02   Phrase received: `APPROVED T-010`
