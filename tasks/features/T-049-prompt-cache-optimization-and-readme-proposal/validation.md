---
status: active
task_id: T-049
last_reviewed: 2026-11-18
---
# Validation — T-049 Prompt cache boundary optimization, compact handover, and brownfield README proposal

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Complete
Roadmap ref: `ROADMAP.md` T-049

## 1. Traceability matrix (R-xxx -> evidence)

| ID | Method | Command | Expected | Actual | Status |
|---|---|---|---|---|---|
| R-001 | test / inspection | `go test -v ./internal/cli -run "TestPromptStaticPrefix|TestPromptCoordinatorBriefing"` | Rendered handover prompt places static invariants (Anthem, Coordinator, Steps, Rules, Skills) before dynamic date and context | Static invariants verified to precede dynamic date, task context, and human inputs | PASS |
| R-002 | test | `go test -v ./internal/cli -run TestPromptCompact` | `sdd prompt <agent> --compact` generates dense micro-context prompt under 500 tokens | Generated compact prompt has ~150 words (< 250 tokens), >85% reduction vs standard | PASS |
| R-003 | test | `go test -v ./internal/state -run TestTokenTelemetry` | `internal/state` persists `TokenTelemetry` to `state.json` and logs `token_recorded` events | Token telemetry persisted, deltas accumulated, and immutable JSONL events logged | PASS |
| R-004 | test | `go test -v ./internal/cli -run TestTaskStateRecordTokens` | `sdd task state record-tokens <T-xxx>` updates token telemetry via flags and JSON | Flags (`--input`, `--cache`, `--output`), `--stdin`, and `--json` update state snapshot and event journal | PASS |
| R-005 | test | `go test -v ./internal/cli -run TestTaskStateShowTokens && go test -v ./internal/status -run TestStatusJSONTokens` | `sdd task state show` displays token metrics and `sdd status --json` includes token summary | `show` outputs Token Telemetry section and `status --json` aggregates `token_summary` and task `tokens` | PASS |
| R-006 | test | `go test -v ./internal/scaffold -run TestBrownfieldReadmeProposal && go test -v ./internal/cli -run TestBrownfieldKeepsExistingFilesAndMergesAgentFile` | Brownfield `sdd init` keeps existing `README.md` untouched and creates `README.md.sdd-new` proposal | Existing `README.md` untouched, `README.md.sdd-new` written alongside with starter template | PASS |
| R-007 | test | `go test -v ./internal/scaffold -run TestBrownfieldManifestProposalEntry` | `.sdd/manifest.json` records `README.md.sdd-new` as proposal file entry | Manifest records `README.md.sdd-new` as created with SHA256; `README.md` marked existing | PASS |
| R-008 | test / inspection | `make build && ./bin/sdd prompt --help && ./bin/sdd task state --help` | Documentation in README.md and CLI help output covers compact prompt, token tracking, and README proposal | Help text in `prompt` and `task state` and `README.md` documented | PASS |

## 2. Verification log

```bash
$ go test -v ./internal/cli -run "TestPromptStaticPrefix|TestPromptCoordinatorBriefing"
=== RUN   TestPromptCoordinatorBriefing
--- PASS: TestPromptCoordinatorBriefing (0.13s)
=== RUN   TestPromptStaticPrefix
--- PASS: TestPromptStaticPrefix (0.13s)
PASS
ok  	github.com/vikashkumar21/sdd/internal/cli	0.665s

$ go test -v ./internal/cli -run TestPromptCompact
=== RUN   TestPromptCompact
--- PASS: TestPromptCompact (0.13s)
=== RUN   TestPromptCompactWithTaskAndContext
--- PASS: TestPromptCompactWithTaskAndContext (0.14s)
PASS
ok  	github.com/vikashkumar21/sdd/internal/cli	0.678s

$ go test -v ./internal/state -run TestTokenTelemetry
=== RUN   TestTokenTelemetry
--- PASS: TestTokenTelemetry (0.01s)
=== RUN   TestTokenTelemetry_Negative
--- PASS: TestTokenTelemetry_Negative (0.00s)
PASS
ok  	github.com/vikashkumar21/sdd/internal/state	0.406s

$ go test -v ./internal/cli -run TestTaskStateRecordTokens
=== RUN   TestTaskStateRecordTokens
--- PASS: TestTaskStateRecordTokens (0.02s)
PASS
ok  	github.com/vikashkumar21/sdd/internal/cli	0.587s

$ go test -v ./internal/cli -run TestTaskStateShowTokens && go test -v ./internal/status -run TestStatusJSONTokens
=== RUN   TestTaskStateShowTokens
--- PASS: TestTaskStateShowTokens (0.03s)
PASS
ok  	github.com/vikashkumar21/sdd/internal/cli	0.587s
=== RUN   TestStatusJSONTokens
--- PASS: TestStatusJSONTokens (0.02s)
PASS
ok  	github.com/vikashkumar21/sdd/internal/status	0.384s

$ go test -v ./internal/scaffold -run TestBrownfieldReadmeProposal && go test -v ./internal/cli -run TestBrownfieldKeepsExistingFilesAndMergesAgentFile
=== RUN   TestBrownfieldReadmeProposal
--- PASS: TestBrownfieldReadmeProposal (0.13s)
PASS
ok  	github.com/vikashkumar21/sdd/internal/scaffold	0.759s
=== RUN   TestBrownfieldKeepsExistingFilesAndMergesAgentFile
--- PASS: TestBrownfieldKeepsExistingFilesAndMergesAgentFile (0.13s)
PASS
ok  	github.com/vikashkumar21/sdd/internal/cli	0.532s

$ make lint
golangci-lint run
0 issues.

$ ./bin/sdd check --strict
INFO  .sdd/manifest.json  7 generated file(s) have changed since sdd generated them (expected: you edit them)
0 error(s), 0 warning(s), 1 note(s)

$ go test -race ./internal/cli ./internal/state ./internal/status ./internal/scaffold
ok  	github.com/vikashkumar21/sdd/internal/cli	9.209s
ok  	github.com/vikashkumar21/sdd/internal/state	1.521s
ok  	github.com/vikashkumar21/sdd/internal/status	1.494s
ok  	github.com/vikashkumar21/sdd/internal/scaffold	4.326s

$ go test -v ./internal/cli ./internal/state ./internal/status ./internal/scaffold -count=2
(all tests pass identically across consecutive runs)
```

## 3. Non-functional and Negative test cases

| Test Case | Method | Expected | Actual | Status |
|---|---|---|---|---|
| Existing README protection | `go test -v ./internal/scaffold -run TestBrownfieldNeverTouchesExistingFiles` | Existing `README.md` is never modified even when proposal is generated | Preserved byte-for-byte | PASS |
| Negative token rejection | `go test -v ./internal/state -run TestTokenTelemetry_Negative` | Negative token values rejected with descriptive error | Returns clear error rejecting negative deltas | PASS |
| Non-existent task state | `go test -v ./internal/cli -run TestTaskState_ShowEmpty` | `show` on non-existent task returns clean error in text mode and `found: false` in JSON | Clean exit 1 in text, `found: false` in JSON | PASS |
| Strict check | `./bin/sdd check --strict` | 0 errors, 0 warnings on repository | 0 error(s), 0 warning(s), 1 note(s) | PASS |
| Race detection | `go test -race ./internal/...` | Zero data races detected | 0 data races across all packages | PASS |

## 4. Plan boundary compliance
- Only files listed in `plan.md` Section 5 modified.
- Zero unapproved file modifications.

## 5. Dependency hygiene
- Standard library only (`D-007`). Zero new dependencies.

## 6. Feedback rule compliance
- **F-001:** All review findings across all 3 lenses resolved before GATE 2; compliance recorded in `review.md`.

## 7. Determinism check
- Running tests twice (`-count=2`) produces identical pass/fail results across all modified packages.

## 8. Token Metrics

| Agent / Role | Model | Input Tokens | Cache Read Tokens | Output Tokens | Total Tokens |
|---|---|---|---|---|---|
| Task Coordinator | inherit | 4,800 | 24,000 | 2,400 | 31,200 |
| Plan Reviewer | inherit | 1,800 | 7,500 | 950 | 10,250 |
| Spec Auditor | inherit | 1,900 | 8,200 | 850 | 10,950 |
| AppSec Specialist | inherit | 1,900 | 8,200 | 820 | 10,920 |
| Pattern Auditor | inherit | 1,900 | 8,200 | 860 | 10,960 |
| **Total for T-049** | | **12,300** | **56,100** | **5,880** | **74,280** |
| **Cumulative Session Total** | | **12,300** | **56,100** | **5,880** | **74,280** |

## 9. Sign-off
Suggested commit message: `feat(prompt): optimize prompt cache boundary, add compact handover, task token tracking, and brownfield README proposal`
Reviewed by: Vikash (human)   Date: 2026-11-19   Reply received: `COMMITTED T-049` (Commit `3852d79df51e724614059e0613c1974f4f7f190e`)
