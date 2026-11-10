# Validation — T-037 Document lifecycle YAML frontmatter metadata and .ragignore generation

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Pass

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test / inspection | `go test -v ./internal/doc -run 'Test(ParseFrontmatter\|ValidateLifecycle)'` | Schema supports status, task_id, superseded_by, last_reviewed | Pass (all fields parsed and validated) | Pass |
| R-002 | test | `go test -v ./internal/doc -run 'Test(ParseFrontmatter\|ValidateLifecycle)'` | Pure Go parser and validator without third-party dependencies | Pass (100% standard library) | Pass |
| R-003 | test | `go test -v ./internal/scaffold -run TestGreenfieldCreatesTheWholeKit` | `sdd init` includes `.ragignore` in generated plan | Pass (`.ragignore` verified on disk) | Pass |
| R-004 | test | `go test -v ./internal/upgrade -run TestUpgradeInstallsMissingRAGIgnore` | `sdd upgrade` adds missing `.ragignore` automatically | Pass (`.ragignore` created with canonical contents) | Pass |
| R-005 | test | `go test -v ./internal/check -run TestRAGIgnoreAndLifecycle` | `sdd check` warns if `.ragignore` is missing from root | Pass (`ragignore-missing` emitted when absent) | Pass |
| R-006 | test | `go test -v ./internal/check -run TestRAGIgnoreAndLifecycle` | `sdd check` validates frontmatter status, dates, and superseded targets | Pass (`lifecycle-status`, `lifecycle-date`, `lifecycle-superseded` emitted) | Pass |
| R-007 | test | `go test -v ./internal/templates -run TestVersion` and template inspections | Base templates include frontmatter and `.ragignore.tmpl`, version is 1.7.0 | Pass (templates.Version is 1.7.0) | Pass |
| R-008 | test | `go test -v ./internal/check -run TestDetectsProblems/agent_contract_drift` | Contracts synchronized in 100% parity to templates=1.7.0 | Pass (0 drift across AGENTS/CLAUDE/GEMINI) | Pass |
| R-009 | test | `make check` and `./bin/sdd check --strict` | 0 errors and 0 warnings | Pass (0 errors, 0 warnings) | Pass |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | Pass (gofmt, go vet, and all package tests pass) |
| `make check` | Pass (verify + golangci-lint clean with 0 issues) |
| `./bin/sdd check --strict` | Pass (0 errors, 0 warnings, 1 info note) |
| `tmpdir=$(mktemp -d) && ./bin/sdd init --dir "$tmpdir" --name testkit --lang go --yes && ./bin/sdd check --dir "$tmpdir"` | Pass (0 errors, 1 expected TODO warning) |

## 3. Manual checks
- Root `.ragignore` exists and contains canonical ignore patterns (`.sdd/`, `bin/`, `dist/`, etc.): Verified.
- `sdd init` generates `.ragignore`: Verified in fresh temporary directory.
- `sdd check` reports a warning when `.ragignore` is deleted: Verified in `TestRAGIgnoreAndLifecycle`.
- `sdd check` reports a warning when Markdown frontmatter has an invalid status or nonexistent superseded target: Verified in `TestRAGIgnoreAndLifecycle`.

## 4. Negative and failure cases
- Frontmatter with invalid `status: unknown_status` -> `sdd check` warns on invalid status (`lifecycle-status`): Verified.
- Frontmatter with `status: superseded` but empty or nonexistent `superseded_by` -> `sdd check` warns on invalid superseded reference (`lifecycle-superseded`): Verified.
- Frontmatter with malformed date `last_reviewed: not-a-date` -> `sdd check` warns on date format (`lifecycle-date`): Verified.

## 5. Determinism check
- Running `./bin/sdd check --json` twice produces byte-identical output: Verified (empty diff between runs).

## 6. Deviations, known limitations, and feedback-rule compliance
- Deviations: none.
- Feedback-rule compliance (SKILLS.md, "Human feedback rules"): Rule F-001 complied with (all review findings resolved before GATE 2 close).

## 7. Definition of done
- [x] Every requirement has evidence
- [x] Build, lint, and tests pass
- [x] Negative and failure cases tested
- [x] Determinism confirmed
- [x] No files changed outside the plan; no unapproved dependencies
- [x] Independent subagent audit completed in review.md with 0 Blockers; all findings resolved per F-001
- [x] Living docs updated
- [x] Compliant with every active feedback rule
- [x] Suggested commit message provided

## 8. Sign-off
Suggested commit message: `feat(rag): add document lifecycle frontmatter and .ragignore generation (T-037)`
Reviewed by: ______   Date: ______   Reply received: pending
