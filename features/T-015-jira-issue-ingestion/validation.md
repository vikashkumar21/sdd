# Validation — T-015 Jira issue ingestion (JSON, Markdown, and text)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Passed
Roadmap ref: `ROADMAP.md` T-015

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/jira -run TestParseJiraJSON` | Extracts key, summary, description, and acceptance criteria from JSON | PASS: REST JSON, simplified JSON, and ADF parsed correctly | PASS |
| R-002 | test | `go test -v ./internal/jira -run TestParseJiraMarkdownAndText` | Extracts fields from Markdown and plain text | PASS: Markdown headers, bullet points, and plain text parsed | PASS |
| R-003 | test | `go test -v ./internal/feature -run TestNewFromJira` | Populates requirements.md Goal, Context, and Acceptance criteria | PASS: Goal, Context, requirements table with escaped pipes, and criteria populated | PASS |
| R-004 | test | `go test -v ./internal/feature -run TestNewFromJiraStagedContext` | Staged raw Jira issue into .sdd/context/<task>/ with format jira | PASS: Staged to .sdd/context/T-001/ with format jira and recorded in manifest.json | PASS |
| R-005 | test | `go test -v ./internal/cli -run TestCLIIngestJira` | --jira in sdd ingest validates and stages Jira issue with format jira | PASS: Validated and staged format jira in manifest.json | PASS |
| R-006 | test | `go test -v ./internal/jira -run TestParseJiraMalformed` | Fails safe with descriptive error on malformed or empty Jira input | PASS: Empty, whitespace, unparseable JSON/text all reject safely | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | PASS (gofmt clean, go vet clean, all packages test -count=1 pass) |
| `make check` | PASS (golangci-lint 0 issues, full test suite pass) |
| `go run ./cmd/sdd check --strict` | PASS (0 errors, 0 warnings) |

## 3. Manual checks
- Verified that `sdd feature new --from-jira` populates `requirements.md` with Goal, Jira issue reference in Context, escaped table pipes, and criteria bullets.
- Verified that `sdd ingest --jira` properly identifies and stages Jira issue artifacts with format `jira`.

## 4. Negative and failure cases
- `TestCLIIngestJiraInvalid`: Ingesting unstructured text without Jira key or summary fails safe with exit code 1 and error message `invalid jira issue`.
- `TestCLIFeatureNewFromJira`: Passing non-existent file to `--from-jira` exits with code 1.
- `TestParseJiraMalformed`: Empty input, whitespace-only, empty JSON, and unparseable input return descriptive errors.

## 5. Determinism check
Ran `go test ./internal/jira ./internal/feature ./internal/cli -count=1` twice; outputs identical: yes.
- Pass 1:
  `ok  github.com/vikashkumar21/sdd/internal/jira    0.070s`
  `ok  github.com/vikashkumar21/sdd/internal/feature 0.883s`
  `ok  github.com/vikashkumar21/sdd/internal/cli     2.250s`
- Pass 2:
  `ok  github.com/vikashkumar21/sdd/internal/jira    0.067s`
  `ok  github.com/vikashkumar21/sdd/internal/feature 0.910s`
  `ok  github.com/vikashkumar21/sdd/internal/cli     2.231s`

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
Suggested commit message: `feat(jira): add Jira issue ingestion for task templates and context staging (T-015)`
Reviewed by: Vikash   Date: 2026-11-02   Reply received: `COMMITTED T-015`
