---
status: active
task_id: T-049
last_reviewed: 2026-11-18
---
# Plan — T-049 Prompt cache boundary optimization, compact handover, and brownfield README proposal

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-18)
Roadmap ref: `ROADMAP.md` T-049

## 1. Approach summary
Restructure the handover prompt template (`base/prompt.md.tmpl`) to place static invariants (Anthem, Coordinator Role Mandate, Steps, Hard Rules, Skills Catalog) at the top of the prompt, ensuring stable KV prefix caching across turns. Add `--compact` flag to `sdd prompt` to generate an ultra-dense, micro-context handover for smaller context budgets. Extend `internal/state` with token metrics (`input_tokens`, `cache_read_tokens`, `output_tokens`, `total_tokens`) and CLI subcommand `sdd task state record-tokens`. Update `internal/scaffold` so brownfield `sdd init` preserves existing `README.md` while generating `README.md.sdd-new` side-by-side.

## 2. Decisions involved

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-042 | Prompt cache prefix pinning, compact handover, task token tracking, and brownfield README proposal | Option A: Static-first prompt prefix ordering, sdd prompt --compact flag, token telemetry in .sdd/state/<TASK_ID>/, and README.md.sdd-new proposal generation in sdd init / Option B: Leave prompts dynamic-first, no compact mode, no token state, and skip README completely | Option A: Maximizes prompt cache hit rates across all LLM providers, provides a compact handover mode, tracks operational token costs per task, and offers brownfield developers an SDD README starter proposal without overwriting human work. | **Approved** 2026-11-18 |

## 3. Design

### 3.1 Static-First Prompt Template Reordering
Structure `internal/templates/data/base/prompt.md.tmpl`:
1. **Header & Anthem (Static Prefix):**
   - Role declaration: Task Coordinator.
   - Operating Anthem: ALWAYS BE DETERMINISTIC.
   - Role orchestration: .sdd/roles.json dispatches.
2. **Workflow & State Machine (Static Prefix):**
   - Step 0: Read project order.
   - Step 1: Verify repository state commands.
   - Step 2: Act on exactly one branch (Phases A–D, gates).
3. **Hard Rules & Reporting Standards (Static Prefix):**
   - Hard rules summary (1–8).
   - Gate reporting format (1–6).
4. **Available Skills Catalog (Static Prefix):**
   - Progressive disclosure skills table.
5. **Dynamic Suffix (Variable Tokens):**
   - Roadmap snapshot & generation date.
   - Focus task context & staged files.
   - Human inputs (gate replies, decisions).

*Template Versioning Note (Rule 16 Rationale):* `prompt.md.tmpl` and `prompt_compact.md.tmpl` are internal templates executed strictly by the CLI `sdd prompt` command to output text to stdout; they are not part of the project kit contract files generated into downstream repositories by `sdd init` or tracked by `.sdd/manifest.json`. Consequently, `templates.Version` remains 2.0.0.

### 3.2 Compact Prompt Mode (`sdd prompt --compact`)
Add `--compact` to `internal/cli/commands.go` (`runPrompt` flagset) and `internal/cli/prompt.go`:
- Renders `base/prompt_compact.md.tmpl`:
  - Concise role mandate & Anthem.
  - Active task status, current worktree/branch, and phase (read from `.sdd/state/<TASK_ID>/` or `ROADMAP.md`).
  - Next immediate action (e.g. "Phase B: implement plan.md", "Phase C.2: spawn Review Triad").
  - Essential gate token expectations (`APPROVED`, `COMMITTED`).
  - Total token size < 400 tokens (~75% reduction compared to standard prompt).

### 3.3 Task Token State Extension (`internal/state`)
Extend `internal/state/state.go` with idiomatic package-level functions:
```go
type TokenTelemetry struct {
	InputTokens     int64 `json:"input_tokens"`
	CacheReadTokens int64 `json:"cache_read_tokens"`
	OutputTokens    int64 `json:"output_tokens"`
	TotalTokens     int64 `json:"total_tokens"`
}

// RecordTokens validates non-negative deltas, accumulates into existing state.json
// under key "tokens", atomically writes the snapshot, and appends a "token_recorded" event.
func RecordTokens(root, taskID string, input, cache, output int64, now time.Time) (SaveResult, error)
```

**Token Accumulation Semantics:**
- Validates `input >= 0`, `cache >= 0`, `output >= 0`.
- Reads existing state via `Show(root, taskID)`. If `state.json` does not exist, an empty baseline is initialized.
- Accumulates token metrics incrementally:
  - `new.InputTokens = existing.InputTokens + input`
  - `new.CacheReadTokens = existing.CacheReadTokens + cache`
  - `new.OutputTokens = existing.OutputTokens + output`
  - `new.TotalTokens = new.InputTokens + new.CacheReadTokens + new.OutputTokens`
- Atomically saves updated state snapshot via `SaveState`.
- Appends an immutable event to `events.jsonl` via `AppendEvent`:
  ```json
  {"type": "token_recorded", "task_id": "T-xxx", "delta": {"input": 1000, "cache": 5000, "output": 200}, "totals": {"input": 1000, "cache": 5000, "output": 200, "total": 6200}, "recorded_at": "..."}
  ```

CLI Subcommand:
`sdd task state record-tokens <T-xxx> [--input <N>] [--cache <N>] [--output <N>] [--json '<json>']`

### 3.4 Brownfield README Proposal (`internal/scaffold`)
In `internal/scaffold/plan.go`:
When inspecting `README.md` with `protect: true`:
- If `!exists`: `Op{Path: "README.md", Kind: KindCreate, Content: content}`
- If `exists`:
  - `Op{Path: "README.md", Kind: KindSkip, Reason: "exists; never modified"}`
  - AND add: `Op{Path: "README.md.sdd-new", Kind: KindCreate, Content: content, Reason: "existing README preserved; starter template written to README.md.sdd-new"}`
- Manifest records `README.md.sdd-new` as proposal file.

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Restructure `base/prompt.md.tmpl` to position static invariants first and dynamic date/inputs last. Update prompt unit tests in `internal/cli/prompt_context_test.go` and `internal/cli/cli_test.go`. | `internal/templates/data/base/prompt.md.tmpl`, `internal/cli/prompt_context_test.go`, `internal/cli/cli_test.go` | `go test -v ./internal/cli -run "TestPromptStaticPrefix|TestPromptCoordinatorBriefing"` |
| 2 | Author `base/prompt_compact.md.tmpl` and implement `--compact` flag in `internal/cli/commands.go` and `internal/cli/prompt.go`. Author tests asserting compact token brevity and contents. | `internal/templates/data/base/prompt_compact.md.tmpl`, `internal/cli/commands.go`, `internal/cli/prompt.go`, `internal/cli/prompt_compact_test.go` | `go test -v ./internal/cli -run TestPromptCompact` |
| 3 | Extend `internal/state/state.go` with `TokenTelemetry`, `RecordTokens()` method, and append `token_recorded` event to `events.jsonl`. Author unit tests in `internal/state/state_test.go`. | `internal/state/state.go`, `internal/state/state_test.go` | `go test -v ./internal/state` |
| 4 | Implement `sdd task state record-tokens` subcommand in `internal/cli/task_state.go`, update `sdd task state show` to format token metrics, update `sdd status --json`, and update `internal/status/status.go`. Author CLI tests. | `internal/cli/task_state.go`, `internal/cli/task_state_test.go`, `internal/status/status.go`, `internal/status/status_test.go` | `go test -v ./internal/cli -run TestTaskState && go test -v ./internal/status` |
| 5 | Update `internal/scaffold/plan.go` to generate `README.md.sdd-new` proposal during brownfield `sdd init` when `README.md` exists. Update scaffold and cli unit tests. | `internal/scaffold/plan.go`, `internal/scaffold/scaffold_test.go`, `internal/cli/cli_test.go` | `go test -v ./internal/scaffold -run TestBrownfield && go test -v ./internal/cli -run TestBrownfieldKeepsExistingFilesAndMergesAgentFile` |
| 6 | Update `README.md` and CLI help text in `cli.go`, `commands.go`, `task.go`, and `task_state.go` documenting `--compact`, `record-tokens`, and the brownfield `README.md.sdd-new` proposal. | `README.md`, `internal/cli/cli.go`, `internal/cli/commands.go`, `internal/cli/task.go`, `internal/cli/task_state.go` | `make build && ./bin/sdd prompt --help && ./bin/sdd task state --help` |
| 7 | Run full verification suite: `make verify`, `make lint`, `./bin/sdd check --strict`, and determinism check (`-count=2`). Record evidence in `validation.md`. | (all) | `make verify && make lint && ./bin/sdd check --strict` |

## 5. Files to touch

| File | Action | Purpose |
|---|---|---|
| `internal/templates/data/base/prompt.md.tmpl` | modify | Restructure prompt template to pin static invariants as prefix |
| `internal/templates/data/base/prompt_compact.md.tmpl` | create | Canonical high-density compact handover prompt template |
| `internal/cli/commands.go` | modify | Add `--compact` flag to `sdd prompt` FlagSet |
| `internal/cli/prompt.go` | modify | Wire compact prompt generation logic |
| `internal/cli/prompt_compact_test.go` | create | Unit tests verifying `--compact` prompt structure and brevity |
| `internal/cli/prompt_context_test.go` | modify | Update assertions matching static-first prompt layout |
| `internal/cli/cli.go` | modify | Update usage text for `sdd prompt` and `sdd task state` |
| `internal/cli/cli_test.go` | modify | Update assertions for prompt output and brownfield README proposal |
| `internal/cli/task.go` | modify | Update `sdd task state` subcommand summary |
| `internal/cli/task_state.go` | modify | Add record-tokens CLI subcommand and display tokens in show |
| `internal/cli/task_state_test.go` | modify | CLI tests for `sdd task state record-tokens` |
| `internal/state/state.go` | modify | Add TokenTelemetry struct and RecordTokens method |
| `internal/state/state_test.go` | modify | Unit tests for TokenTelemetry persistence and event logging |
| `internal/status/status.go` | modify | Include token summary in `sdd status --json` |
| `internal/status/status_test.go` | modify | Unit tests verifying token fields in `sdd status --json` |
| `internal/scaffold/plan.go` | modify | Generate README.md.sdd-new proposal when README.md exists |
| `internal/scaffold/scaffold_test.go` | modify | Unit test for brownfield README.md.sdd-new proposal creation |
| `README.md` | modify | Document compact prompt, token tracking, and README proposal |
| `ROADMAP.md` | modify | Update T-049 status and change log |
| `TECH_STACK.md` | modify | Record D-042 as Approved and record verified facts |

## 6. Dependencies
Standard library only (`D-007`). Zero external third-party dependencies.

## 7. Living docs to update
- `ROADMAP.md`: update status to `Awaiting review` in Phase D, `Done` upon `COMMITTED T-049`; record in change log.
- `TECH_STACK.md`: record `D-042` as Approved, add verified facts entry in §7.
- `README.md`: document `sdd prompt --compact`, `sdd task state record-tokens`, and `README.md.sdd-new` proposal.

## 8. Risks, mitigations, rollback
- **Risk:** Reordering prompt template sections could break existing regex/string assertions in downstream tests.
  - **Mitigation:** Run `go test ./...` across all packages; adjust test assertions to verify semantic sections rather than fragile absolute line offsets.
- **Risk:** Generating `README.md.sdd-new` could clutter brownfield repositories if unexpected.
  - **Mitigation:** Clear CLI output explaining that `README.md` was preserved and `README.md.sdd-new` was created as an optional reference proposal.
- **Rollback:** `git checkout` reverts template and scaffold changes; delete created compact template and test files.

## 9. Applicable human-feedback rules
- **F-001:** Worker agents must resolve all findings (Blockers, Warnings, and Minors) reported during review before presenting to human at GATE 2; any retained finding must be explicitly justified in `review.md`.

## 10. Approval record
Approved by: Vikash   Date: 2026-11-18   Phrase received: `APPROVED T-049`
