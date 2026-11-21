---
status: draft
---
# Validation — T-050 Path hygiene validator and zero-personal-path enforcement

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Complete

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/check -run TestPathHygiene_MarkdownScan` | PASS: Scans all markdown files in root and tasks/ | PASS (0.00s): Scanned GUIDE.md and tasks/.../requirements.md, 2 findings | PASS |
| R-002 | test | `go test -v ./internal/check -run TestPathHygiene_Platforms` (Rule F-002) | PASS: Flags macOS, Linux, and Windows home paths | PASS (0.00s): Detected /Users/alice, /home/bob, C:\Users\charlie, D:/Users/david | PASS |
| R-003 | test | `go test -v ./internal/check -run TestPathHygiene_Exemptions` | PASS: Permits runner, example, username, ellipsis, and rule lines | PASS (0.00s): 0 findings across exempt paths, ellipsis, and rule lines | PASS |
| R-004 | test | `go test -v ./internal/check -run TestPathHygiene_WarningFormat` | PASS: Warns with path-hygiene rule, line number, file | PASS (0.00s): Warn severity, sub/doc.md:2, rule path-hygiene, formatted message | PASS |
| R-005 | test | `go test -v ./internal/check -run TestPathHygiene_Suppression` | PASS: Waivers in suppressions.json suppress finding to Info | PASS (0.00s): finding suppressed via .sdd/suppressions.json | PASS |
| R-006 | inspection | `git grep "/Users/vikash" tasks/` (Rule F-002) | 0 occurrences across tasks/ | Exit 1 (0 matches across tasks/) | PASS |
| R-007 | test | `./bin/sdd check --strict` | 0 errors, 0 warnings | 0 error(s), 0 warning(s), 1 note(s) | PASS |
| R-008 | test | `go test -v ./internal/deconflict -run TestExtractPlanFiles_PathNames` | PASS: Correctly extracts files containing 'path' | PASS (0.00s): Extracted internal/check/paths.go, paths_test.go, deconflict.go | PASS |

## 2. Commands run

| Command | Result |
|---|---|
| `make build` | PASS (go build -trimpath -o bin/sdd ./cmd/sdd) |
| `make test` | PASS (ok across all packages in 15.7s) |
| `make vet` | PASS (go vet ./... clean) |
| `make lint` | PASS (golangci-lint run: 0 issues) |
| `make check` | PASS (vet, lint 0 issues, all tests pass) |
| `go test -race ./internal/check ./internal/deconflict` | PASS (check 18.076s, deconflict 1.344s, 0 data races) |
| `go test -count=2 ./internal/check ./internal/deconflict` | PASS (check 13.299s, deconflict 0.070s) |
| `./bin/sdd check --strict` | PASS (0 error(s), 0 warning(s), 1 note(s)) |

## 3. Manual checks
- Verify `git grep -rn "/Users/vikash" .` returns 0 occurrences across all tracked documentation (Rule F-002).
  - Actual: Matches only in historical regression fixture `internal/check/testdata/dbguard-kit/` (read-only hand-made kit fixture). Zero occurrences in tracked project docs, templates, or tasks.

## 4. Negative and failure cases
- Positive detection of `/Users/john/project` -> generates WARN path-hygiene on exact line (verified in `TestPathHygiene_WarningFormat`).
- Positive detection of `/home/dev/repo` -> generates WARN path-hygiene on exact line (verified in `TestPathHygiene_MarkdownScan`).
- Positive detection of `C:\Users\tester\repo` -> generates WARN path-hygiene on exact line (verified in `TestPathHygiene_Platforms`).
- Detection of `/home/runner` -> no warning (verified in `TestPathHygiene_Exemptions`).
- Detection of `/Users/example/demo` -> no warning (verified in `TestPathHygiene_Exemptions`).
- Path ending with `/Users/...` -> no warning (verified in `TestPathHygiene_Exemptions`).

## 5. Determinism check
Ran `go test -count=2 ./internal/check ./internal/deconflict` twice; outputs identical: yes.
Evidence:
- Check run 1: ok github.com/vikashkumar21/sdd/internal/check 13.299s
- Check run 2: ok github.com/vikashkumar21/sdd/internal/deconflict 0.070s
- Zero non-deterministic output or state dependencies.

## 6. Deviations, known limitations, and feedback-rule compliance
- Compliance with Rule F-001 (Resolve all review findings): Verified; all Review Triad findings (Blocker F-SPEC-01/B-01/SEC-001, Warnings F-SPEC-02/W-01 and SEC-002, Minors F-SPEC-03, M-01, SEC-003, SEC-004) resolved in unified remediation batch.
- Compliance with Rule F-002 (Zero personal local paths): Verified by path hygiene validator and repository scan.

## 7. Definition of done
- [x] Every requirement has evidence
- [x] Build, lint, and tests pass (commands above)
- [x] Negative and failure cases tested
- [x] Determinism confirmed
- [x] No files changed outside the plan; no unapproved dependencies
- [x] Living docs updated
- [x] Compliant with every active feedback rule (F-001, F-002)
- [x] Suggested commit message provided

## 8. Sign-off
Suggested commit message: `feat(check): add path hygiene validator and enforce zero personal paths (T-050)`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-050`
