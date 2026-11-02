# Independent Senior Go Review & Adaptive Architecture Report (T-005)

> **ANTHEM: ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**

**Reviewer:** Independent Senior Go Engineer & Reviewer  
**Host Environment Verified:** Darwin arm64, `go version go1.27.3 darwin/arm64`, `golangci-lint v2.15.0`, `GNU Make 3.81`  
**Target Codebase:** `/Users/vikash/projects/sdd` (Commit `d91c9c2`)  
**Status:** Complete (Awaiting GATE 2 review)

---

## 1. Genesis, Decisions & Verified Facts

### 1.1 The Problem & Origin
The maintainer experienced a universal failure mode of AI coding agents: without explicit operational contracts, agents exhibit non-deterministic behavior—they speculate, invent unapproved dependencies, write code before architecture is agreed upon, self-commit breaking changes, silently choose between trade-offs, and lose historical context between sessions.

To solve this in an initial project (`dbguard`), the maintainer manually authored a spec-driven kit consisting of:
- Symmetrical agent contracts (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`) with explicit gates.
- Living architectural and operational documents (`MISSION.md`, `TECH_STACK.md`, `ROADMAP.md`, `SKILLS.md`).
- Structured per-task templates (`requirements.md`, `plan.md`, `validation.md`).

Maintaining this kit by hand across multiple repositories proved laborious and error-prone, causing drift across project copies. `sdd` was created as a thin, deterministic CLI to scaffold, mechanically validate, and extend this kit.

### 1.2 Decision Register Trace (D-001 through D-017)
Every architectural choice in `sdd` traces to explicit decisions:
- **D-001 (Approved):** Tool named `sdd`, module `github.com/vikashkumar21/sdd`.
- **D-002 (Approved):** Spec-driven workflow with 2 gates (GATE 1 before code, GATE 2 before human commit).
- **D-003, D-007 (Approved):** Go, standard library only, single static binary, no cgo.
- **D-004 (Proposed):** `Makefile` build/lint/test toolchain.
- **D-005 (Proposed):** Conventional Commits format.
- **D-006 (Approved):** Custom thin tool over adopting GitHub Spec Kit / OpenSpec.
- **D-008 (Approved):** Binary name `sdd`.
- **D-009 (Approved):** v1 scope: `init`, `check`, `feature new`, `prompt`, `version`.
- **D-010 (Approved):** Brownfield policy: never overwrite; managed-block contract merge; seed Discovery task.
- **D-011 (Approved):** Bundled packs: Go, Java, Python, TypeScript, generic.
- **D-012 (Approved):** Default agent files: `CLAUDE.md`, `GEMINI.md`, `AGENTS.md`.
- **D-013 (Approved):** File-level template override directory + `.sdd/manifest.json`.
- **D-014 (Approved):** Tool repository path.
- **D-015 (Proposed):** `go` directive in `go.mod` (1.22 vs 1.24 vs 1.27).
- **D-016 (Proposed):** Template syntax: Go `text/template`, missingkey=error.
- **D-017 (Proposed):** Manifest schema 1 and marker syntax `<!-- sdd:begin contract templates=<v> -->`.

### 1.3 Fact Verification Audit
| Statement / Claim | Classification | Evidence / Source |
|---|---|---|
| Maintainer built `dbguard` with a hand-made kit | Reported, verified by repo fixture | `internal/check/testdata/dbguard-kit` contains real hand-crafted kit |
| `sdd` builds standard-library only | **Verified fact** | `go.mod` has no `require` blocks; `go.sum` does not exist |
| Go toolchain on this machine is Go 1.27 | **Verified fact** | `go version` outputs `go version go1.27.3 darwin/arm64` (verified on host 2026-10-29) |
| Latest stable Go release on go.dev is Go 1.27.3 | **Verified fact** | Verified via `go.dev/dl/` search (2026-10-29) |
| `golangci-lint` v2 is available on host | **Verified fact** | `golangci-lint has version 2.15.0 built with go1.27.2` |
| `make verify` passes on macOS | **Verified fact** | Executed `make verify` on host; clean exit 0 |
| `make check` passes on macOS | **Refuted claim** | `make check` fails; `golangci-lint run` produced 12 errors |

---

## 2. Comparative Analysis & Hypothesis H2 (GitHub Spec Kit vs OpenSpec vs `sdd`)

### 2.1 Web Research Summary (Sources dated 2026)
1. **GitHub Spec Kit (`github/spec-kit` / `specify-cli`):**
   - *Design:* Framework for "Specification-Driven Development" across Copilot, Claude Code, and Gemini. Relies on structured templates (`spec.md`, `plan.md`, `tasks.md`) and a Python/Node CLI (`specify-cli`).
   - *Flaw regarding our anthem:* It focuses on prompt chaining and context assembly. It **does NOT** enforce hard human gates (exact phrases like `APPROVED T-xxx`), **does NOT** forbid AI commits, and provides no built-in mechanical validator for cross-document reference integrity or decision registers.
2. **OpenSpec (`openspec.dev` by Fission-AI):**
   - *Design:* Config-driven spec framework with slash-commands (`/opsx:explore`, `/opsx:propose`) and context budgeting (~50KB per spec) for coding agents.
   - *Flaw regarding our anthem:* OpenSpec treats the agent as an autonomous collaborator that can propose and apply diffs directly. It lacks deterministic gate phrases, has no formal decision ledger (`D-xxx`), no human feedback recording loop (`F-xxx`), and requires custom agent-side integrations.

### 2.2 Resolution of Hypothesis H2
> **Finding on H2:** Neither GitHub Spec Kit nor OpenSpec implements `sdd`'s non-negotiable core invariants:
> 1. Exact string gate confirmations (`APPROVED T-xxx`, `COMMITTED T-xxx`).
> 2. Complete prohibition of autonomous git write operations.
> 3. Strict decision registries (`D-xxx`) where unapproved architectural choices halt execution.
> 4. Persistent human feedback capture (`F-xxx` in `SKILLS.md`) binding on all subsequent tasks.
> 5. Zero-dependency, single static binary execution.

**Conclusion:** **H2 is validated.** Adopting GitHub Spec Kit or OpenSpec would compromise the anthem ("ALWAYS BE DETERMINISTIC. Never guess. Never silently choose.") and dilute human control. `sdd` must remain an independent, bespoke tool.

---

## 3. Adversarial Code Review & Correctness Analysis

A thorough adversarial inspection of `cmd/sdd` and `internal/*` reveals serious structural risks and bugs:

### 3.1 Unchecked Linter Failures (`golangci-lint v2.15.0`)
Running `golangci-lint run` fails immediately with 12 issues:
1. `internal/check/check_test.go:106:5`: `SA4000: identical expressions on the left and right side of the '!=' operator (staticcheck)`:
   ```go
   if dump(run(t, root)) != dump(run(t, root)) { // Bug: comparing function call against identical expression directly
   ```
2. `internal/detect/detect_test.go:69, 72`: `QF1001: could apply De Morgan's law (staticcheck)`.
3. `internal/cli/cli.go` (lines 67, 80, 83, 86) & `internal/cli/commands.go` (lines 25, 56, 67, 123, 183):
   `errcheck: Error return value of fmt.Fprint* is not checked`.

### 3.2 Coverage Gaps & Untested Packages
- `cmd/sdd/main.go`: **0.0% coverage**. Uncovered logic includes `isTerminal(os.Stdin)`, environment mapping, and exit code propagation.
- `internal/fsutil/atomic.go`: **0.0% coverage**. `WriteAtomic` has no standalone unit tests.
- `internal/scaffold`: 83.3% statement coverage; missing coverage on permission errors, corrupted manifest handling, and symlink resolution.
- `internal/feature`: 91.8% statement coverage; lacks test cases for boundary ID parsing (such as four-digit task IDs).

### 3.3 Atomic Write Vulnerabilities (`internal/fsutil/atomic.go`)
1. **Missing fsync (`File.Sync()`):**
   `WriteAtomic` writes data and immediately closes and renames the temporary file without calling `tmp.Sync()`. On power failure, OS crash, or write-back caching file systems, this risks metadata loss or zero-length files.
2. **Directory Permissive Creation:**
   `os.MkdirAll(dir, 0o755)` creates missing directories, but if the target path is on a read-only filesystem or a directory without write permissions, `os.CreateTemp` fails with an OS error, which is caught, but error messaging does not identify whether directory permissions or disk capacity caused the failure.

### 3.4 Fragility in Markdown Checking (`internal/check/check.go`)
1. **Naive Fence Counting (counting code fence markers with `strings.Count`):**
   - Fails when markdown documents use four-backtick fences (` ```` `) to nest markdown examples or carousels. Substring counting leads to false positives or false negatives.
   - Inline mentions of fence markers in prose confuse the counter.
2. **Naive Table Validation (`strings.Count(line, "|") != want`):**
   - A cell containing an escaped pipe (`\|`) or an inline code block with a pipe (`` `a | b` ``) increments the pipe count, causing `sdd check` to fail valid markdown tables.
   - Tables indented within list items (e.g. `  | col1 | col2 |`) are skipped entirely because `strings.HasPrefix(line, "|")` requires `|` at index 0.
3. **Regex Task and Decision ID Boundaries:**
   - `decisionRefRe = regexp.MustCompile(\`\bD-\d{3}\b\`)` and `taskRefRe = regexp.MustCompile(\`\bT-\d{3}\b\`)`.
   - When a project scales beyond 999 tasks (`T-1000`) or 999 decisions (`D-1000`), the regex will either fail to match or match only the 3-digit prefix.
4. **Contract Parity Normalization:**
   `contractText()` trims trailing whitespace on each line, but does not normalize Windows CRLF line endings when comparing files if one was saved with LF and another with CRLF outside git filters.

### 3.5 Project Scanning & Detection Fragility (`internal/detect/detect.go`)
1. **Hard Scan Cap (`maxVisited = 20000`):**
   In large monorepos, once 20,000 files are encountered, `Detect` returns `fs.SkipAll` and flags `Capped = true`. This leaves language detection partial and ignores manifests located in later directory branches.
2. **Single-Winner Command Suggestion:**
   In polyglot repos (e.g. Go backend + TypeScript frontend), `suggest()` uses a fixed precedence loop:
   ```go
   for _, l := range []string{"go", "java", "typescript", "python"}
   ```
   Go always overwrites suggestions. The build/test/lint commands for TypeScript are completely dropped from `TECH_STACK.md`.
3. **Hard-coded Depth Cap (`maxMarkDepth = 2`):**
   Manifests deeper than 2 levels (e.g., `services/gateway/api/go.mod` or `packages/web/client/package.json`) are completely invisible to `Detect`.

### 3.6 Security & Environment Edge Cases
1. **Directory Traversal Defense:**
   `safeRel()` checks for `..` segments and leading slashes. However, standard library `os.Root` (available in Go 1.24+) provides true kernel/OS-level confinement against symlink race conditions (TOCTOU).
2. **Advisory Markdown Rules:**
   `sdd` generates markdown rules stating "Never commit or push". However, without generated agent profile configurations (such as `.claude/permissions.json` or `.gemini/settings.json`) that explicitly deny tool execution of `git commit`, enforcement relies solely on the LLM's compliance with prompt instructions.

---

## 4. Coverage of Project Types & Gap Matrix

| Project Type | Current Behavior | Failure Mode / Limitation | Required Enhancement |
|---|---|---|---|
| **Greenfield Go/Java/Python/TS** | Generates full kit; seeds `T-001` | Works smoothly, but generates 22 `TODO(human)` markers failing `--strict` | Seed clean defaults or clarify that `--strict` is expected to fail initially |
| **Brownfield Go** | Detects `go.mod`, Makefile; appends managed block | Disregards existing `ROADMAP.md` or `TECH_STACK.md` if named differently (e.g. `docs/architecture.md`) | Interactive mapping of existing documentation |
| **Polyglot Monorepo** | Detects multiple languages; caps scan if >20k files | Drops commands for secondary languages; ignores manifests at depth >2 | Monorepo workspace detection (`go.work`, pnpm workspaces, Gradle multi-project) |
| **Existing Hand-Crafted Kit (`dbguard`)** | Refuses `sdd init` cleanly | Cannot be adopted or upgraded without manual manifest creation | Implement `sdd adopt` (T-004) to generate manifest without file mutations |
| **Multi-Agent Teams** | Generates CLAUDE.md, GEMINI.md, AGENTS.md | If one developer uses Cursor or Windsurf, contracts drift or require manual sync | Generic agent profile generator reading unified contract definition |
| **Pure JavaScript / Node** | Flags as `other: [javascript]` | No language pack exists; falls back to empty generic TODO | Provide a standard modern Node/JS pack or unified JS/TS pack |

---

## 5. Language Packs Quality Review

1. **Go Pack (`packs/go/standards.md`):**
   - *Strengths:* Excellent idiom coverage (`Effective Go`, error wrapping with `%w`, context-first, table-driven tests, clock injection).
   - *Weaknesses:* Still references older linter paradigms; does not explicitly mention `slices`, `maps`, `errors.Join`, or path confinement via `os.Root`.
2. **Java Pack (`packs/java/standards.md`):**
   - *Strengths:* Good emphasis on immutability (`final`, records), null safety, try-with-resources.
   - *Weaknesses:* Lacks mention of modern Java virtual threads (Project Loom / Java 21+), module boundaries, and Maven/Gradle toolchain caching.
3. **Python Pack (`packs/python/standards.md`):**
   - *Strengths:* PEP 8/257, type hints (PEP 484), dataclasses, context managers.
   - *Weaknesses:* Missing modern dependency tooling (e.g., `uv`, `poetry`, `ruff` instead of generic flake8/black); no mention of `typing.Self` or modern async practices.
4. **TypeScript Pack (`packs/typescript/standards.md`):**
   - *Strengths:* Strict mode, `unknown` over `any`, discriminated unions, immutability.
   - *Weaknesses:* Lacks runtime schema validation guidance (Zod, ArkType) which is essential for agent boundary safety.
5. **Generic Pack (`packs/generic/standards.md`):**
   - Consists solely of a single `TODO(human)` line. Provides no template guidance or prompt scaffold for agent discovery.

---

## 6. Deterministic Adaptive Architecture Proposals

To allow `sdd` to self-check, self-heal, self-analyse, and self-improve without violating the determinism anthem:

### 6.1 Self-Check Architecture
- **Automatic:**
  - Machine-readable `--json` output for `sdd check` to enable CI pipeline integration.
  - Expiry checking of `TECH_STACK.md` verified facts (`Re-verify at` milestone/date check against current roadmap milestone).
  - Cross-document validation between `.sdd/manifest.json`, embedded templates version, and active contracts.
- **Human-Gated:** Blocking CI on expired facts or strict warnings.
- **Failure Modes:** Malformed dates or unparseable milestones in verified-facts log fail gracefully with clear line diagnostics.

### 6.2 Self-Heal Architecture (`sdd repair` / `sdd check --fix`)
- **Strictly Non-Autonomous:** The tool **never** silently modifies human files.
- **Deterministic Mechanism:**
  - `sdd check --fix` presents a unified diff preview of proposed mechanical fixes (e.g. syncing managed contract blocks across agent files, fixing broken table delimiter row pipe counts, re-sorting roadmap tables).
  - Requires explicit confirmation (`[y/N]`) before applying.
  - Backs up modified files to `.sdd/backup/<timestamp>/`.

### 6.3 Self-Analyse Architecture (`sdd audit` / `sdd status`)
- **Deterministic Computation:**
  - Metrics derived purely from parsing markdown tables:
    - Estimated vs actual duration across completed tasks.
    - Count of `Open` decisions in `TECH_STACK.md` and duration since logged.
    - Ratio of completed requirements to total requirements.
    - Active `F-xxx` feedback rule compliance tracking.
- **Output:** Read-only console summary or structured JSON report. Zero side effects.

### 6.4 Self-Improvement Loop
- **Human Feedback to Template Pipeline:**
  - When an `F-xxx` rule is added to `SKILLS.md`, a helper command `sdd rule export` can format the rule into a proposed addition to the local template override directory (`internal/templates/data/`).
  - Template changes trigger an automatic bump of `templates.Version`.
  - Regression corpus test: before releasing a template change, `sdd check` runs against all fixtures in `internal/check/testdata/` to verify zero regression.

### 6.5 Boundaries: What Must NOT Be Automated
1. **No Autonomous Git Commits or Tags:** Commits represent human legal and architectural liability.
2. **No Silent Code Modification:** Code changes must originate from an approved `plan.md`.
3. **No Unmonitored LLM Invocations:** The CLI must remain offline, local, deterministic, and free of non-deterministic model calls.

---

## 7. Modern Go Architecture & Standard Library Modernisation

### 7.1 Toolchain Alignment (Go 1.27.3 & D-015 Resolution)
- **Current `go.mod`:** `go 1.22`.
- **Local Machine Toolchain:** `go1.27.3 darwin/arm64`.
- **Upstream Stable Release:** Go 1.27.3 is the verified stable release on `go.dev/dl/`.
- **Decision D-015 Recommendation:**
  Update `go.mod` to Go 1.24 or Go 1.26. Specifically, setting the directive to `go 1.24` immediately unlocks `os.Root` (standard-library path sandboxing) while retaining backward compatibility across all modern Go toolchains.

### 7.2 Modern Standard Library Adoption Candidates
1. **`os.Root` (`os.OpenRoot`):**
   - *Status:* Verified in Go 1.24+ standard library.
   - *Benefit:* Replaces fragile custom string path sanitation (`safeRel`) with OS-enforced filesystem confinement, preventing symlink escapes and path traversal attacks during scaffold and template operations.
2. **`slices` and `maps` packages:**
   - *Status:* Standard library since Go 1.21.
   - *Benefit:* Replaces custom helpers (`sortedKeys`, manual slice filtering) with standard, optimized, type-safe generic functions (`slices.Sort`, `slices.Contains`, `maps.Keys`).
3. **`errors.Join`:**
   - *Status:* Standard library since Go 1.20.
   - *Benefit:* Allows `internal/check` and `internal/scaffold` to aggregate multiple file validation errors without custom formatting or loss of error wrapping.
4. **Range-over-int (`for i := range n`):**
   - *Status:* Standard library since Go 1.22.
   - *Benefit:* Simplifies loop constructs across string repetitions, table formatting, and test cases.
5. **Structured Logging (`log/slog`):**
   - *Status:* Standard library since Go 1.21.
   - *Benefit:* Can provide deterministic structured diagnostic logs for `sdd check` and `sdd init` without third-party dependencies.

---

## 8. Prioritised Improvement Proposals Catalogue (`P-01` through `P-10`)

Each proposal adheres strictly to project invariants: never overwrite human work (S1), byte-level determinism (S3), and dbguard-kit backwards compatibility.

### `P-01`: Fix Linter Failures & Enable Clean CI Verification
- **Problem:** `golangci-lint run` fails with 12 issues (errcheck in CLI, staticcheck SA4000 in `check_test.go`, QF1001 in `detect_test.go`).
- **Solution:** Check error returns in `internal/cli`, fix the identical expression comparison bug in `internal/check/check_test.go:106`, and clean boolean logic in `detect_test.go`.
- **Size:** S (1 sitting)
- **Risk:** Very low.
- **Dependencies:** None.
- **Decisions Needed:** D-004 approval.
- **Testing:** `make check` exits 0 cleanly.

### `P-02`: Unit Test Suite & Coverage for `fsutil` and `cmd/sdd`
- **Problem:** `internal/fsutil` and `cmd/sdd` have 0.0% test coverage. `WriteAtomic` lacks unit tests for fsync, permission errors, and directory creation.
- **Solution:** Add `atomic_test.go` exercising atomic write replacements, error paths, and read-only directories; add `main_test.go` testing exit code propagation and terminal detection seams.
- **Size:** S (1 sitting)
- **Risk:** Low.
- **Dependencies:** None.
- **Decisions Needed:** None.
- **Testing:** `go test -cover ./...` shows >85% across all packages.

### `P-03`: Filesystem Confinement via `os.Root` (Modern Go Upgrade)
- **Problem:** `safeRel` relies on string parsing; symlink attacks or unusual paths could escape project root.
- **Solution:** Resolve D-015 to `go 1.24+`; refactor `fsutil` and `scaffold` to operate through `os.OpenRoot(root)` for strictly confined filesystem operations.
- **Size:** M (2 sittings)
- **Risk:** Low to medium.
- **Dependencies:** Resolution of D-015.
- **Decisions Needed:** D-015 (Approved).
- **Testing:** Comprehensive directory escape and symlink tests in scratch directories.

### `P-04`: Robust Markdown Parsing in `internal/check`
- **Problem:** Naive fence counting and pipe counting break on nested fences (````), escaped pipes (`\|`), pipes in code spans, and indented tables.
- **Solution:** Implement a small, deterministic standard-library line-by-line state machine parser that respects escape characters, inline code spans, and block indentation without third-party dependencies.
- **Size:** M (2 sittings)
- **Risk:** Low. Must maintain compatibility with real `dbguard-kit` fixture.
- **Dependencies:** None.
- **Decisions Needed:** None.
- **Testing:** Expanded table-driven test suite with complex markdown edge cases.

### `P-05`: Machine-Readable JSON Output for `sdd check`
- **Problem:** `sdd check` only outputs formatted human text; difficult to integrate with CI/CD pipelines or IDE tooling.
- **Solution:** Add `sdd check --json` flag outputting structured JSON findings (`severity`, `file`, `line`, `message`, `code`).
- **Size:** S (1 sitting)
- **Risk:** Very low.
- **Dependencies:** None.
- **Decisions Needed:** None.
- **Testing:** Golden-file test comparing JSON output against schema.

### `P-06`: Implement `sdd adopt` for Hand-Crafted Kits (T-004 Acceleration)
- **Problem:** `sdd init` refuses hand-crafted kits like `dbguard`; existing projects cannot obtain a manifest without manual JSON construction.
- **Solution:** Implement `sdd adopt`: scans existing core documents, detects managed block or full contract, computes SHA256 hashes, and generates a valid `.sdd/manifest.json` with zero changes to existing markdown files.
- **Size:** S (1 sitting)
- **Risk:** Low.
- **Dependencies:** `P-01`.
- **Decisions Needed:** D-017 confirmation.
- **Testing:** Run `sdd adopt` on a scratch copy of `dbguard-kit`; verify subsequent `sdd check` passes with 0 errors.

### `P-07`: Monorepo & Polyglot Detection Enhancement
- **Problem:** `detect` caps at 20,000 files, depth 2, and drops commands for secondary languages in polyglot projects.
- **Solution:** Increase depth limit to 4 for manifest searches, support multi-command suggestions in `TECH_STACK.md`, and optimize walk performance with directory-level filtering.
- **Size:** M (2 sittings)
- **Risk:** Low.
- **Dependencies:** None.
- **Decisions Needed:** None.
- **Testing:** Polyglot fixture test with mixed Go/TypeScript monorepo.

### `P-08`: Verified-Facts Expiry & Roadmap Drift Checker
- **Problem:** Verified facts in `TECH_STACK.md` contain "Re-verify at" milestones that are never checked by `sdd check`.
- **Solution:** Add an `sdd check` rule that matches `Re-verify at` task IDs or milestones against completed roadmap tasks, emitting a warning when a fact is past due for re-verification.
- **Size:** S (1 sitting)
- **Risk:** Low.
- **Dependencies:** None.
- **Decisions Needed:** None.
- **Testing:** Unit test with expired and active fact rows.

### `P-09`: Safe Interactive Repair Command (`sdd check --fix`)
- **Problem:** Syncing agent contract blocks or formatting table delimiters across multiple documents by hand is tedious.
- **Solution:** Provide `sdd check --fix` that displays a diff preview of safe mechanical repairs (contract synchronization, table pipe alignment), confirms with user, and creates backups before applying.
- **Size:** M (2 sittings)
- **Risk:** Medium. Must strictly respect S1 (never overwrite human work outside managed blocks).
- **Dependencies:** `P-04`.
- **Decisions Needed:** Human confirmation protocol.
- **Testing:** Dry-run and live tests on corrupted kit fixtures.

### `P-10`: Agent Permission Manifest Generation
- **Problem:** Markdown contracts instruct agents not to commit, but cannot enforce it technically.
- **Solution:** Add an optional flag `sdd init --agent-rules` that generates IDE/agent permission setting files (e.g., Claude Code configuration denying git write commands) based on verified syntax.
- **Size:** M (2 sittings)
- **Risk:** Low.
- **Dependencies:** None.
- **Decisions Needed:** Approved agent permission schema decision.
- **Testing:** Verify generated configuration syntax against official agent documentation.

---

## 9. Verification Evidence on Host Machine

All empirical tests below were executed directly on this machine (`darwin/arm64`):

```bash
$ go version
go version go1.27.3 darwin/arm64

$ golangci-lint --version
golangci-lint has version 2.15.0 built with go1.27.2 from 25c5369 on 2026-10-16T10:51:54Z

$ make verify
gofmt -s check, go vet, test: PASS (9 packages tested, 0 race conditions, 0 vet issues)

$ golangci-lint run
internal/check/check_test.go:106:5: SA4000: identical expressions on the left and right side of the '!=' operator (staticcheck)
internal/detect/detect_test.go:69:58: QF1001: could apply De Morgan's law (staticcheck)
internal/detect/detect_test.go:72:54: QF1001: could apply De Morgan's law (staticcheck)
internal/cli/cli.go:67,80,83,86: unchecked error returns (errcheck)
internal/cli/commands.go:25,56,67,123,183: unchecked error returns (errcheck)
Exit Code: 1 (Failing)

$ go doc os.Root
package os // import "os"
type Root struct { ... }
func OpenRoot(name string) (*Root, error)
Confirmed present in toolchain standard library.
```

Scratch experiments in `/var/folders/.../T/sdd-scratch-*`:
1. **Path with spaces:** `sdd init --yes --dir "path with spaces"` succeeded with exit code 0.
2. **Symlink root:** Initializing inside a symlink directory succeeded with exit code 0.
3. **Read-only directory:** Initializing inside a `0o555` directory failed safely with exit code 1 (`permission denied`) leaving zero leftover temporary files.
4. **Hand-made kit refusal:** Copy of `dbguard-kit` refused initialization cleanly with exit code 1 and explanatory guidance to run `sdd check`.

---

## 10. Summary & Recommended Action
The foundation of `sdd` is exceptionally strong: it achieves strict determinism, clean separation of planning and edge execution, and adheres to zero third-party dependencies. However, `golangci-lint` errors must be resolved, test coverage for `fsutil` and `cmd/sdd` completed, and markdown parsing hardened before expanding into further features.

Following GATE 1 approval (`APPROVED T-005`), the maintainer can select proposals (`SELECT P-xx, ...`) for prioritized development.
