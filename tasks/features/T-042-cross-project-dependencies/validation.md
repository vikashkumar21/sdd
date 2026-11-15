# Validation — T-042 Cross-project and service dependency register in TECH_STACK.md §8

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Complete

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/templates -run TestTechStackTemplateDependencySections` | pass | PASS (TestTechStackTemplateDependencySections) | PASS |
| R-002 | test / inspection | `go test -v ./internal/templates -run TestTechStackTemplateDependencySections` | pass | PASS (TestTechStackTemplateDependencySections: 7 canonical columns verified) | PASS |
| R-003 | inspection / check | `./bin/sdd check --strict` | pass | PASS (0 error(s), 0 warning(s), 1 note(s)) | PASS |
| R-004 | test | `go test -v ./internal/check -run TestCrossProjectDependencies_DanglingDecision` | pass | PASS (TestCrossProjectDependencies_DanglingDecision) | PASS |
| R-005 | test | `go test -v ./internal/check -run TestCrossProjectDependencies_DirectionEnum` | pass | PASS (TestCrossProjectDependencies_DirectionEnum) | PASS |
| R-006 | test | `go test -v ./internal/check -run TestCrossProjectDependencies_StatusEnum` | pass | PASS (TestCrossProjectDependencies_StatusEnum) | PASS |
| R-007 | test | `go test -v ./internal/templates -run TestVersion` | pass | PASS (TestVersion: Version = "1.8.0") | PASS |
| R-008 | test | `go test -v ./internal/scaffold -run TestScaffoldTECH_STACK` && `go test -v ./internal/upgrade -run TestUpgradeTECH_STACK` | pass | PASS (TestScaffoldTECH_STACK & TestUpgradeTECH_STACK) | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make build` | pass: `go build -trimpath -ldflags ... -o bin/sdd ./cmd/sdd` |
| `make test` | pass: `go test ./... -count=1` all packages ok |
| `make lint` | pass: `golangci-lint run` 0 issues |
| `make verify` | pass: `go vet ./...` & `go test ./...` ok |
| `tmpdir=$(mktemp -d) && ./bin/sdd init --dir "$tmpdir" --name testkit --lang go --yes && ./bin/sdd check --dir "$tmpdir"` | pass: fresh kit generated with 1.8.0 templates and checked with 0 errors |
| `./bin/sdd check --strict` | pass: 0 error(s), 0 warning(s), 1 note(s) |

## 3. Manual checks
- `sdd init` in a fresh directory renders both §8.1 and §8.2 tables in `TECH_STACK.md` (verified in `TestScaffoldTECH_STACK` and Contract Rule 16 temporary-directory check).
- `sdd upgrade` in an existing project preserves §8.1 and §8.2 tables and safely upgrades untouched files to 1.8.0 (verified in `TestUpgradeTECH_STACK`).
- `TECH_STACK.md` in this repository adopts §8.1 and §8.2 with `sdd-agent` registered under `D-035` (verified in `checkTechStackDependencies`).
- `./bin/sdd check --strict` passes with 0 errors and 0 warnings.

## 4. Negative and failure cases
- A §8.2 row containing a dangling `D-xxx` decision ID not present in §6 reports `techstack-dep-decision` error (verified in `TestCrossProjectDependencies_DanglingDecision`).
- A §8.2 row containing an invalid decision format (e.g. `D-9999`) reports `techstack-dep-decision` error (verified in `TestCrossProjectDependencies_InvalidDecisionFormat`).
- A §8.2 row containing an unrecognized direction descriptor reports `techstack-dep-direction` warning (verified in `TestCrossProjectDependencies_DirectionEnum`).
- A §8.2 row containing an unrecognized status value reports `techstack-dep-status` warning (verified in `TestCrossProjectDependencies_StatusEnum`).
- A §8.2 row with fewer than 7 columns reports `techstack-dep-columns` warning (verified in `TestCrossProjectDependencies_MalformedColumns`).
- A legacy `TECH_STACK.md` lacking §8.2 passes cleanly with 0 findings (verified in `TestCrossProjectDependencies_BackwardCompatibility`).

## 5. Determinism check
Ran `go test ./internal/check ./internal/templates ./internal/scaffold ./internal/upgrade -count=2`; all runs passed identically with 0 flakes.

## 6. Deviations, known limitations, and feedback-rule compliance
None.

Feedback-rule compliance (SKILLS.md, "Human feedback rules"):
- Rule F-001 (Resolve all review findings: Blockers, Warnings, and Minors): Phase C.2 Review Triad audit initiated.

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
Suggested commit message: `feat(templates): add cross-project dependency register to TECH_STACK.md and sdd check validation`
Reviewed by: Vikash (human)   Date: 2026-11-15   Reply received: `COMMITTED T-042`
