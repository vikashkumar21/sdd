---
status: active
task_id: T-047
last_reviewed: 2026-11-19
---
# Plan — T-047 Remote CI failure ingestion and diagnostic auto-repair loop

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-19)
Roadmap ref: `ROADMAP.md` T-047

## 1. Approach summary
Implement a dedicated CI failure log processing engine in `internal/cilog` that performs high-speed ANSI escape code stripping, multi-vector credential and secret sanitization (AWS keys, GitHub tokens, Bearer auth, private keys, password assignments), failure diagnostic extraction across polyglot runners (Go, Node, Python, Java, SDD governance), and bugfix reproduction document seeding. Wire `--ci-log` into `sdd ingest` to stage sanitized logs into `.sdd/context/`. Add `--from-ci` to `sdd bugfix new` and `sdd task new --type bugfix` to automatically seed `reproduction.md`. Implement `sdd prompt repair <T-xxx>` to generate a turnkey auto-repair handover prompt for coding agents.

## 2. Decisions involved

| ID | Title | Options | Recommendation | Status |
|---|---|---|---|---|
| D-040 | Universal CI workflows, automated PR packaging, and CI log ingestion | Option A: Ingest CI failure logs via sdd ingest --ci-log with ANSI stripping, secret sanitization, and automated bugfix reproduction seeding into tasks/bugfixes/ / Option B: Manual text copying into staged context without automated redaction or diagnostic parsing | Option A: Ensures sensitive credentials leaked in CI outputs are scrubbed before reaching agent context, automatically extracts failing test traces, and closes the loop between CI failures and bugfix resolution. | Approved (2026-11-19) |

## 3. Design

### 3.1 CI Log Parser & Sanitization Engine (`internal/cilog`)
```go
package cilog

type Diagnostics struct {
    Platform       string   `json:"platform"`        // "github", "gitlab", "bitbucket", "azure", "gcp", "aws", "generic"
    FailedStep     string   `json:"failed_step"`     // e.g. "go test -v ./...", "make verify", "npm test"
    PrimaryError   string   `json:"primary_error"`   // headline error (e.g. "FAIL: TestFoo", "compile error")
    FailedTests    []string `json:"failed_tests"`    // list of failed test names
    FailureSnippet string   `json:"failure_snippet"` // bounded context window around failure
    RawLength      int      `json:"raw_length"`
    SanitizedLength int     `json:"sanitized_length"`
    RedactionsCount int     `json:"redactions_count"`
}

// StripANSI removes all ANSI escape sequences, terminal control characters, and color codes.
func StripANSI(input string) string

// SanitizeSecrets redacts sensitive credentials, tokens, AWS keys, GitHub tokens, and private keys.
func SanitizeSecrets(input string) (string, int)

// Parse strips ANSI, sanitizes credentials, extracts diagnostics, and returns structured result.
func Parse(data []byte) (*Diagnostics, string, error)

// SeedBugfixReproduction formats diagnostics into standard reproduction.md markdown sections.
func SeedBugfixReproduction(diag *Diagnostics, templateText string) string
```

### 3.2 CLI Command Extensions
1. **`sdd ingest --ci-log` (`internal/cli/ingest.go`):**
   - `--ci-log`: boolean flag activating CI log pipeline.
   - `--seed-bugfix`: boolean flag to automatically trigger bugfix task creation (requires `--task <T-xxx>` existing in `ROADMAP.md`; slug defaults to `--name` or sanitized failed test name).
   - Cleans and sanitizes raw input, creates `.sdd/context/<timestamp>-<name>.log`, and records metadata in `manifest.json`.
2. **`sdd bugfix new <T-xxx> <slug> --from-ci <file|staged-id>` & `sdd task new --type bugfix` (`internal/cli/task.go`, `internal/feature`):**
   - Accepts path to a local CI log or staged context ID.
   - Parses log and initializes `tasks/bugfixes/<T-xxx>-<slug>/reproduction.md` with problem summary, reproduction steps, and minimal failing test case template.
3. **`sdd prompt repair <T-xxx> [--json] [--max-loops <N>]` (`internal/cli/prompt_repair.go`):**
   - Reads `reproduction.md`, `rca.md`, `plan.md`, and staged CI context for the bugfix task.
   - Accepts `--max-loops <N>` (default: `3`, validated bounds: `1` to `5`).
   - Retrieves or initializes loop iteration counter from `.sdd/state/<T-xxx>/` (events/state).
   - Outputs an agent briefing emphasizing **Mandatory Step 0 (minimal reproducing test case)**, sanitized error trace, reproduction command, and current loop count (e.g. `Attempt: 1 of 3`).
   - If iteration counter reaches or exceeds `--max-loops`, emits explicit circuit breaker halt directives instructing the agent to cease code changes, mark task `Blocked`, and escalate to human.

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement `internal/cilog` engine: ANSI sequence stripper (`StripANSI`), secret/token sanitization (`SanitizeSecrets`), failure diagnostic extractor (`ExtractDiagnostics`), and `reproduction.md` injector (`SeedBugfixReproduction`). Author comprehensive unit tests covering diverse CI log formats (Go, Node, Python, Java, SDD checks). | `internal/cilog/cilog.go`, `internal/cilog/cilog_test.go` | `go test -v ./internal/cilog` |
| 2 | Integrate `--ci-log` and `--seed-bugfix` flags into `sdd ingest` in `internal/cli/ingest.go`. Author CLI integration tests verifying ANSI stripping, secret redaction, and `.sdd/context/manifest.json` staging. | `internal/cli/ingest.go`, `internal/cli/ingest_test.go` | `go test -v ./internal/cli -run TestIngest_CILog` |
| 3 | Integrate `--from-ci` flag into `sdd bugfix new` and `sdd task new --type bugfix` in `internal/cli/task.go` and `internal/feature`. Author tests verifying automatic population of `reproduction.md` from raw or staged CI logs. | `internal/cli/task.go`, `internal/cli/task_test.go`, `internal/feature/feature.go` | `go test -v ./internal/cli -run TestBugfixNew_FromCI` |
| 4 | Implement `sdd prompt repair <T-xxx> [--json] [--max-loops <N>]` in `internal/cli/prompt_repair.go`, wiring into `runPrompt` dispatch in `internal/cli/commands.go`. Implement loop counter tracking via task state (`.sdd/state/<T-xxx>/`) and circuit breaker escalation. Author tests verifying prompt generation, loop threshold behavior, and JSON payload structure. | `internal/cli/prompt_repair.go`, `internal/cli/prompt.go`, `internal/cli/commands.go`, `internal/cli/prompt_repair_test.go` | `go test -v ./internal/cli -run TestPromptRepair` |
| 5 | Update living documents (`README.md`, `ROADMAP.md`, `TECH_STACK.md`) and CLI help text with documentation and examples for `sdd ingest --ci-log`, `sdd bugfix new --from-ci`, and `sdd prompt repair`. | `README.md`, `ROADMAP.md`, `TECH_STACK.md`, `internal/cli/cli.go` | `make build && ./bin/sdd ingest --help && ./bin/sdd prompt --help` |
| 6 | Execute full verification sweep (`make verify`, `make lint`, `./bin/sdd check --strict`, race detection, and determinism check `-count=2`). Record evidence in `validation.md`. | (all) | `make verify && make lint && ./bin/sdd check --strict` |

## 5. Files to create or modify

| File | Action | Purpose |
|---|---|---|
| `internal/cilog/cilog.go` | create | CI log parser, ANSI stripper, secret sanitizer, and failure diagnostic engine |
| `internal/cilog/cilog_test.go` | create | Unit tests for ANSI stripping, secret sanitizing, diagnostic engine, and bugfix seeding |
| `internal/cli/ingest.go` | modify | Wire `--ci-log` and `--seed-bugfix` flags and handling into `sdd ingest` |
| `internal/cli/ingest_test.go` | modify | Tests for `sdd ingest --ci-log` staging, sanitizing, and manifest recording |
| `internal/cli/task.go` | modify | Wire `--from-ci` flag into `sdd bugfix new` and `sdd task new --type bugfix` |
| `internal/cli/task_test.go` | modify | Tests for `sdd bugfix new --from-ci` reproduction seeding |
| `internal/feature/feature.go` | modify | Extend feature options with FromCIData/FromCIOrigin and seed reproduction.md |
| `internal/feature/feature_test.go` | modify | Regression tests for task scaffolding with CI log reproduction seeding |
| `internal/cli/prompt_repair.go` | create | CLI handler for `sdd prompt repair <T-xxx>` |
| `internal/cli/prompt_repair_test.go` | create | Tests for `sdd prompt repair` markdown and JSON generation |
| `internal/cli/prompt.go` | modify | Add `repair` subcommand dispatch to `runPrompt` |
| `internal/cli/commands.go` | modify | Wire prompt repair routing |
| `internal/cli/cli.go` | modify | Update help usage text for `sdd ingest` and `sdd prompt` |
| `README.md` | modify | Document CI log ingestion, bugfix seeding, and repair prompt workflow |
| `ROADMAP.md` | modify | Update T-047 status and change log |
| `TECH_STACK.md` | modify | Record verified facts entry in §7 for T-047 |

## 6. Dependencies
Standard library only (`D-007`). Zero external third-party dependencies.

## 7. Living docs to update
- `ROADMAP.md`: update status to `Awaiting review` in Phase D, `Done` upon `COMMITTED T-047`.
- `TECH_STACK.md`: record verified facts entry in §7.
- `README.md`: document `sdd ingest --ci-log`, `sdd bugfix new --from-ci`, and `sdd prompt repair`.

## 8. Risks, mitigations, rollback
- **Risk:** CI logs may be huge (tens of megabytes of verbose output) causing memory bloat or out-of-memory.
  - **Mitigation:** Enforce `context.MaxInputBytes` (10MB limit); bound the captured failure snippet to the failure window (e.g. up to 100 lines around error); stream parsing line-by-line using `bufio.Scanner` with a bounded buffer.
- **Risk:** Sensitive secrets in CI logs (e.g. temporary tokens, webhook URLs, database passwords) might be staged into context and exposed to coding agents.
  - **Mitigation:** Multi-pass regex sanitization redacting AWS keys, GitHub tokens, Bearer auth, private keys, and key-value credential assignments into `[REDACTED]`.
- **Rollback:** `git checkout` reverts modifications; delete `internal/cilog/`, `internal/cli/prompt_repair.go`, and `internal/cli/prompt_repair_test.go`.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- **F-001:** Worker agents must resolve all findings (Blockers, Warnings, and Minors) reported during review before presenting to human at GATE 2; any retained finding must be explicitly justified in `review.md`.

## 10. Approval record
Plan Review Subagent: Approved (0 Blockers) 2026-11-19 (conv: `a8723645-0dcf-4eb2-8037-7cf14732d3aa`)
Approved by: Vikash (human)   Date: 2026-11-19   Phrase received: `APPROVED T-047`
