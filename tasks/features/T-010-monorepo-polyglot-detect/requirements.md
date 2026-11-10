# Requirements — T-010 P-07: Monorepo and polyglot detection enhancement

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-02)
Roadmap ref: `ROADMAP.md` T-010

## 1. Goal
Enhance project detection for monorepos and polyglot repositories by expanding manifest search depth to 4 levels, pruning high-volume cache/build directories, and generating deterministic multi-language command suggestions for build, test, lint, and format.

## 2. Context and references
- Mission / principles touched: Principles 1 (Deterministic), 2 (Fail safe), 4 (Small and shippable), Scope §4 (`sdd init` detection)
- Decisions this relies on: `D-003` (Go), `D-004` (Toolchain), `D-007` (Standard library only), `D-011` (Language packs)
- Depends on tasks: `T-002` (Done)

## 3. Scope
**In scope**
- Increasing manifest search depth from 2 to 4 (`maxMarkDepth = 4`) to discover nested manifests (`go.mod`, `pom.xml`, `package.json`, `tsconfig.json`, `pyproject.toml`, etc.).
- Inspecting nested `package.json` files for `typescript` dependencies and npm scripts (`build`, `test`, `lint`, `format`).
- Expanding directory prune list (`ignoredDirs`) to skip common heavy build/cache/tooling folders (`.turbo`, `.pnpm-store`, `bower_components`, `.cache`, `.terraform`, `.yarn`).
- Supporting multi-language command suggestions combined via ` && ` in canonical language order when no unified Makefile target is present.
- Unit and integration tests covering deep monorepo structures, polyglot Go+TypeScript / Go+Python combinations, and nested directory traversal.

**Out of scope**
- Arbitrary script execution or network queries to inspect remote workspaces.
- Upgrading or mutating existing user manifests.
- Adding third-party manifest parsers (standard library `encoding/json` and string scanning only).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL discover language manifest files up to a depth of 4 directories below the project root. | MUST | test |
| R-002 | The system SHALL inspect nested package.json files to detect TypeScript dependencies and npm scripts. | MUST | test |
| R-003 | The system SHALL prune common build and cache directories (.turbo, .pnpm-store, bower_components, .cache, .terraform, .yarn) during directory walks. | MUST | test |
| R-004 | The system SHALL generate multi-language command suggestions combined with ` && ` in canonical language order for build, test, lint, and format when multiple languages are detected. | MUST | test |
| R-005 | The system SHALL prioritize Makefile targets over language-specific commands when Makefile targets exist. | MUST | test |
| R-006 | The system SHALL ensure all generated command suggestions are single-line strings without pipe or backtick characters that would violate markdown table syntax. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Language detection, manifest lists, and combined command ordering SHALL be strictly sorted and reproducible across identical repository trees.
- **Fail-safe behaviour:** Unreadable directories, malformed JSON manifests, or missing scripts SHALL be ignored gracefully without aborting detection.
- **Performance budget:** Walking deep directory trees SHALL skip pruned directories immediately via `filepath.SkipDir` and enforce the 20,000 file visit ceiling (`maxVisited`).
- **Dependencies:** Standard library only (no new dependencies).

## 6. Constraints and assumptions
- Commands combined with ` && ` must not contain newline, backtick, or pipe characters, complying with `Answers.Normalize()`.
- Top-level `Makefile` targets continue to take precedence for any target defined in the Makefile.

## 7. Open questions
(None)

## 8. Acceptance criteria
- Given a monorepo with `services/backend/go.mod` and `apps/web/package.json` at depth 2, when `detect.Detect` runs, then both `go` and `typescript` are detected in `res.Languages` and both manifests are recorded in `res.Manifests`.
- Given a polyglot project with Go and TypeScript, when `detect.Detect` runs without a Makefile, then `res.Build`, `res.Test`, `res.Lint`, and `res.Format` combine the commands in canonical order using ` && `.
- Given a directory named `.pnpm-store` or `.turbo`, when `detect.Detect` traverses the directory tree, then the directory is skipped without traversing its descendants.
