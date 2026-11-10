# Validation — T-037 Document lifecycle YAML frontmatter metadata and .ragignore generation

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Not run

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test / inspection | `go test -v ./internal/doc -run TestLifecycle` | Schema supports status, task_id, superseded_by, last_reviewed | pending | pending |
| R-002 | test | `go test -v ./internal/doc -run TestLifecycle` | Pure Go parser and validator without third-party dependencies | pending | pending |
| R-003 | test | `go test -v ./internal/scaffold -run TestBuildPlan` | `sdd init` includes `.ragignore` in generated plan | pending | pending |
| R-004 | test | `go test -v ./internal/upgrade -run TestUpgrade` | `sdd upgrade` adds missing `.ragignore` automatically | pending | pending |
| R-005 | test | `go test -v ./internal/check -run TestRAGIgnore` | `sdd check` warns if `.ragignore` is missing from root | pending | pending |
| R-006 | test | `go test -v ./internal/check -run TestLifecycleValidation` | `sdd check` validates frontmatter status, dates, and superseded targets | pending | pending |
| R-007 | test | `go test -v ./internal/templates -run TestVersion` and template inspections | Base templates include frontmatter and `.ragignore.tmpl`, version is 1.7.0 | pending | pending |
| R-008 | test | `go test -v ./internal/check -run TestDetectsProblems/agent_contract_drift` | Contracts synchronized in 100% parity to templates=1.7.0 | pending | pending |
| R-009 | test | `make check` and `./bin/sdd check --strict` | 0 errors and 0 warnings | pending | pending |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | pending |
| `make check` | pending |
| `./bin/sdd check --strict` | pending |
| `tmpdir=$(mktemp -d) && ./bin/sdd init --dir "$tmpdir" --name testkit --lang go --yes && ./bin/sdd check --dir "$tmpdir"` | pending |

## 3. Manual checks
- Verify root `.ragignore` exists and contains canonical ignore patterns.
- Verify `sdd init` generates `.ragignore`.
- Verify `sdd check` reports a warning when `.ragignore` is deleted.
- Verify `sdd check` reports a warning when Markdown frontmatter has an invalid status or nonexistent superseded target.

## 4. Negative and failure cases
- Frontmatter with invalid `status: unknown_status` -> `sdd check` warns on invalid status (`lifecycle-status`).
- Frontmatter with `status: superseded` but empty or nonexistent `superseded_by` -> `sdd check` warns on invalid superseded reference (`lifecycle-superseded`).
- Frontmatter with malformed date `last_reviewed: not-a-date` -> `sdd check` warns on date format (`lifecycle-date`).

## 5. Determinism check
- Running `./bin/sdd check --json` twice produces byte-identical output: pending.

## 6. Deviations, known limitations, and feedback-rule compliance
- Deviations: none.
- Feedback-rule compliance (SKILLS.md, "Human feedback rules"): Rule F-001 complied with (all findings will be resolved before GATE 2 close).

## 7. Definition of done
- [ ] Every requirement has evidence
- [ ] Build, lint, and tests pass
- [ ] Negative and failure cases tested
- [ ] Determinism confirmed
- [ ] No files changed outside the plan; no unapproved dependencies
- [ ] Independent subagent audit completed in review.md with 0 Blockers; all findings resolved per F-001
- [ ] Living docs updated
- [ ] Compliant with every active feedback rule
- [ ] Suggested commit message provided

## 8. Sign-off
Suggested commit message: `feat(rag): add document lifecycle frontmatter and .ragignore generation (T-037)`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-037`
