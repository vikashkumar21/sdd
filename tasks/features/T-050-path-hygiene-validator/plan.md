---
status: draft
---
# Plan — T-050 Path hygiene validator and zero-personal-path enforcement

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-21)

Approved by: Vikash (human)   Date: 2026-11-21   Phrase received: `APPROVED T-050`

## 1. Approach summary
Implement a dedicated `checkPathHygiene(rel, text string)` function in `internal/check/paths.go` integrated into the markdown inspection loop in `internal/check/check.go`. Detect macOS (`/Users/<user>`), Linux (`/home/<user>`), and Windows (`[A-Za-z]:\Users\<user>`) absolute home directory patterns using Go standard library `regexp`. Explicitly exempt generic placeholders (`username`, `example`, `user`, `runner`, `admin`, `default`, ellipsis `...`) and lines discussing Rule `F-002`/`path-hygiene`. Clean the single remaining historical occurrence in `tasks/features/T-005-independent-review/review.md`. Add unit tests in `internal/check/paths_test.go` and verify full suite.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| (none) | Path hygiene rule and scope | Settled under binding Human Feedback Rule F-002 | Automate Rule F-002 as `path-hygiene` check in `sdd check` | Approved (F-002) |

## 3. Design
- Package: `internal/check`
- File: `internal/check/paths.go`
- Signatures:
  ```go
  var (
      macUserHomeRe = regexp.MustCompile(`(?:^|[^\w./-])/Users/([a-zA-Z0-9._-]+)(?:/|[^\w./-]|$)`)
      linuxUserHomeRe = regexp.MustCompile(`(?:^|[^\w./-])/home/([a-zA-Z0-9._-]+)(?:/|[^\w./-]|$)`)
      winUserHomeRe = regexp.MustCompile(`(?i)(?:^|[^\w./-])[a-z]:\\Users\\([a-zA-Z0-9._-]+)(?:\\|[^\w./-]|$)`)
  )

  func isExemptUser(user string) bool
  func (c *checker) checkPathHygiene(rel, text string)
  ```
- Workflow:
  1. For each markdown file loaded in `c.markdownFiles()`, split text into lines.
  2. For each line, check if the line contains `F-002` or `path-hygiene` rule discussions; if so, skip.
  3. Scan line for all matches against `macUserHomeRe`, `linuxUserHomeRe`, and `winUserHomeRe` using `FindAllStringSubmatchIndex` to inspect every path on the line.
  4. For each match, if the path ends with `...` or `isExemptUser(strings.ToLower(u))` is true (`username`, `example`, `user`, `runner`, `admin`, `default`), skip.
  5. Otherwise, emit `c.add(Warn, rel, lineNum, "path-hygiene", "local developer home path detected (%s); use repository-relative paths or generic clone URLs per Rule F-002", matchedPath)`.

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Update `internal/deconflict/deconflict.go` header matching to check column 1 specifically, preventing false exclusion of files with "path" in name or reason | `internal/deconflict/deconflict.go`, `internal/deconflict/deconflict_test.go` | `go test -v ./internal/deconflict` |
| 2 | Create `internal/check/paths.go` with regex patterns, exempt user filter, and `checkPathHygiene` | `internal/check/paths.go` | `go test -v ./internal/check -run TestPathHygiene` |
| 3 | Integrate `checkPathHygiene` into `internal/check/check.go` markdown inspection loop | `internal/check/check.go` | `go test -v ./internal/check` |
| 4 | Create unit tests covering positive cases, negative cases, Windows/Linux/macOS paths, and exemptions | `internal/check/paths_test.go` | `go test -v ./internal/check -run TestPathHygiene` |
| 5 | Sanitize historical machine path in `tasks/features/T-005-independent-review/review.md` line 7 per Rule F-002 | `tasks/features/T-005-independent-review/review.md` | `git grep "/Users/vikash" tasks/` |
| 6 | Verify full test suite, race detection, determinism, linting, and strict check | Entire repository | `make verify && make lint && ./bin/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/deconflict/deconflict.go` | modify | Check column 1 specifically for table headers to avoid false skipping of paths containing 'path' |
| `internal/deconflict/deconflict_test.go` | modify | Unit test verifying extraction of plan files with 'path' in filename or description |
| `internal/check/paths.go` | create | Path hygiene checking logic and exemption filters |
| `internal/check/paths_test.go` | create | Unit tests for path hygiene validator across platforms |
| `internal/check/check.go` | modify | Call `c.checkPathHygiene` in `c.run()` markdown loop |
| `tasks/features/T-005-independent-review/review.md` | modify | Sanitize historical local machine path in review notes |
| `tasks/features/T-050-path-hygiene-validator/requirements.md` | modify | Task specification |
| `tasks/features/T-050-path-hygiene-validator/plan.md` | modify | Task implementation plan |
| `tasks/features/T-050-path-hygiene-validator/validation.md` | modify | Verification matrix and evidence |
| `tasks/features/T-050-path-hygiene-validator/review.md` | modify | Review Triad audit record |
| `ROADMAP.md` | modify | Status updates (Specifying -> Implementing -> Awaiting review -> Done) |
| `TECH_STACK.md` | modify | Verified facts log entry for T-050 |

## 6. Dependencies
None. Go standard library only (`regexp`, `strings`).

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Update T-050 status across gates and remove resolved deconflict header matching entry from Section 7 Parking lot |
| `TECH_STACK.md` | Record verified fact for path hygiene check |

## 8. Risks, mitigations, rollback
- **Risk:** False positives on documentation mentioning path syntax or standard CI runner home directories (`/home/runner`).
  - *Mitigation:* Explicit whitelist for `runner`, `username`, `example`, `...`, and rule descriptions; full unit test coverage.
- **Rollback:** Delete `internal/check/paths.go` and `internal/check/paths_test.go`, revert `check.go` invocation.
