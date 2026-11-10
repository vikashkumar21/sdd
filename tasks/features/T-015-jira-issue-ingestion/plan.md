# Plan — T-015 Jira issue ingestion (JSON, Markdown, and text)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-015

## 1. Approach summary
1. Create `internal/jira/jira.go` with multi-format parsing (JSON, Markdown, and plain text) and template injection:
   - Parse JSON from Jira REST API exports and simplified JSON schemas.
   - Parse Markdown and structured plain text with regex-based key, summary, description, and acceptance criteria extraction.
   - Provide `InjectRequirements(templateText string, issue Issue) string` to seed `requirements.md`.
2. Extend `internal/feature/feature.go` and `internal/cli/feature.go`:
   - Add `--from-jira <file>` flag to `sdd feature new`.
   - Inject parsed Jira details into `requirements.md` and stage the raw Jira input into `.sdd/context/<task>/` as a `jira` format artifact.
3. Extend `internal/cli/ingest.go`:
   - Add `--jira` flag to `sdd ingest` to validate Jira issues and stage them with format `jira`.
4. Add comprehensive unit and integration tests across `internal/jira`, `internal/feature`, and `internal/cli`.
5. Update living documents (`README.md`, `ROADMAP.md`).

## 2. Decisions (new or affected)
None. Settled in approved Decision D-019 (Option A: offline file/text-based Jira ingestion).

## 3. Design
- **Jira Issue Structure (`internal/jira/jira.go`)**:
  ```go
  type Issue struct {
      Key                string   `json:"key"`
      Summary            string   `json:"summary"`
      Description        string   `json:"description"`
      AcceptanceCriteria []string `json:"acceptance_criteria"`
      Type               string   `json:"type,omitempty"`
      Priority           string   `json:"priority,omitempty"`
      RawFormat          string   `json:"raw_format"`
  }
  ```
- **Parsing Strategy**:
  - `Parse(data []byte) (Issue, error)`:
    - Attempt JSON unmarshal into standard Jira REST format (`fields.summary`, `fields.description`, `fields.issuetype.name`, `fields.priority.name`, and acceptance criteria fields) or simplified JSON (`summary`, `description`, `acceptance_criteria`).
    - If not valid JSON, evaluate as Markdown/plain text:
      - Match issue key `\b([A-Z][A-Z0-9]+-\d+)\b`.
      - Extract summary from header line (`# KEY: Summary` or `Summary: ...`).
      - Extract description and acceptance criteria sections using section headers.
- **Template Injection**:
  - Replaces Goal placeholder with `issue.Summary`.
  - Appends Jira reference (`- Jira issue: PROJ-123`) to Context and references.
  - Formats each acceptance criterion as an `R-xxx` row in the Functional requirements table.
  - Appends each acceptance criterion under `## 8. Acceptance criteria`.

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement Jira issue parser and template injector | `internal/jira/jira.go`, `internal/jira/jira_test.go` | `go test ./internal/jira` |
| 2 | Extend feature package and CLI with --from-jira flag | `internal/feature/feature.go`, `internal/feature/feature_test.go`, `internal/cli/commands.go` | `go test ./internal/feature -run TestNewFromJira` |
| 3 | Extend ingest CLI with --jira flag | `internal/cli/ingest.go`, `internal/cli/ingest_test.go` | `go test ./internal/cli -run TestCLIIngestJira` |
| 4 | Update living docs (README.md, ROADMAP.md) | `README.md`, `ROADMAP.md` | `go run ./cmd/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/jira/jira.go` | create | Multi-format Jira issue parser and template injector |
| `internal/jira/jira_test.go` | create | Unit tests for JSON, Markdown, and text Jira parsing |
| `internal/feature/feature.go` | modify | Support Jira issue injection and staging during task creation |
| `internal/feature/feature_test.go` | modify | Tests for feature creation from Jira issue |
| `internal/cli/commands.go` | modify | Add --from-jira flag to sdd feature new (where runFeature resides) |
| `internal/cli/ingest.go` | modify | Add --jira flag to sdd ingest |
| `internal/cli/ingest_test.go` | modify | Tests for sdd ingest --jira |
| `README.md` | modify | Document Jira ingestion and --from-jira usage |
| `ROADMAP.md` | modify | Track T-015 status |

## 6. Dependencies
None. Standard library only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `README.md` | Document `sdd feature new --from-jira` and `sdd ingest --jira` |
| `ROADMAP.md` | Update T-015 status across gates |

## 8. Risks, mitigations, rollback
- **Risk:** Inconsistent formatting across various Jira team exports.
  - *Mitigation:* Resilient parser handles REST JSON, simplified JSON, Markdown, and text with fallbacks and helpful errors.
- **Rollback:** `git restore .` reverts all changes.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active.

## 10. Approval record
Approved by: Vikash   Date: 2026-11-02   Phrase received: `APPROVED T-015`
