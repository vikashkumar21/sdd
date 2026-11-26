---
status: completed
---
# Validation — T-064 Native IDE transcript discovery and ingestion in sdd record-tokens

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Complete (Phase C.1 & C.2 passed; dogfood telemetry recorded)
Roadmap ref: `ROADMAP.md` T-064

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/transcript -run TestSummarize` | window edges inclusive; only MODEL/PLANNER_RESPONSE counted; CHECKPOINT and others ignored; missing usage skipped and reported; `LastStepSeen` correct | PASS (3 subtests: open_ended_from_step_1, inclusive_window_1..3, past_end_returns_zero_steps) | PASS |
| R-002 | test | `go test -v ./internal/transcript -run TestSummarizeErrors` | malformed, negative, non-integer, partial usage, missing/negative `step_index`, overflow, long line, oversize input all error with line number only | PASS (all 9 negative subtests passed) | PASS |
| R-003 | test | `go test -v ./internal/transcript -run TestSource` | traversal, non-UUID, uppercase UUID, symlink, directory, missing file, empty home all refused; no scanning | PASS (TestSource_ValidateConversationID, TestSource_DefaultRoot, TestSource_OpenConversation, TestSource_OpenFileAndKey) | PASS |
| R-004 | test | `go test -v ./internal/state -run TestRecordIngest` | cursor/role persistence, replay/overlap/role-mismatch/corruption guards verified | PASS (TestRecordIngest, TestRecordIngest_Guards) | PASS |
| R-005 | test | `go test -v ./internal/cli -run TestTranscriptIngest` | first run requires `--from-step`; replay = nothing new; tokens and cursor change together; one `token_ingested` event; no absolute path in state, events, output | PASS (TestTranscriptIngest_WindowAndIdempotency, TestTranscriptIngest_DryRunAndExplicitFile, TestTranscriptIngest_FlagsAndExitCodes) | PASS |
| R-006 | test | `go test -v ./internal/cli -run RecordTokens` | dry-run leaves state and events byte-identical; exclusivity and bad `--role` exit 2; existing manual tests unchanged | PASS (TestRecordTokensDispatch, TestTaskStateRecordTokens, TestTopLevelRecordTokens) | PASS |
| R-007 | test | `go test -v ./internal/transcript -run TestNoContentLeak` | sentinel text never appears in output or errors (plus CLI leak check in `TestTranscriptIngest`) | PASS (TestNoContentLeak + TestTranscriptIngest_DryRunAndExplicitFile) | PASS |
| R-008 | check | `make verify` | all 36 packages build, vet, and pass tests with stdlib only | PASS (go vet clean; all 36 packages ok) | PASS |
| R-009 | inspection | `git grep -n transcript README.md SKILLS.md TECH_STACK.md` | docs describe flags, Antigravity-only scope, coordinator workflow; D-051..D-053 present | PASS (matches in README.md, SKILLS.md, TECH_STACK.md) | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make build` | PASS (built `bin/sdd`) |
| `make test` | PASS (included in `make verify`, all 36 packages ok) |
| `make verify` | PASS (`go vet ./...` clean; all 36 packages `ok`) |
| `go test -race ./internal/transcript ./internal/state ./internal/cli` | PASS (`0.16s` / `1.40s` / `13.97s`, 0 races) |
| `go test -count=2 ./internal/transcript ./internal/state ./internal/cli` | PASS (all 3 packages identical across 2 runs) |
| `go test -bench=BenchmarkSummarize -benchmem ./internal/transcript` | PASS (`606904 ns/op`, `321829 B/op`, `6003 allocs/op` per 1,000 steps) |
| `./bin/sdd check --strict` | PASS (`0 error(s), 0 warning(s), 1 note(s)`) |
| `git diff --stat main...HEAD -- internal/templates CLAUDE.md GEMINI.md AGENTS.md skills` | PASS (empty — 0 files changed) |

## 3. Manual checks
- Scoped linter: `golangci-lint run ./internal/transcript/... ./internal/state/... ./internal/cli/...` → `0 issues.`
- Real-transcript cross-check: on step window `23516..23540` of the live ~30 MB Antigravity transcript (`23,680` steps) using `--transcript <file> --dry-run --json`, `sdd record-tokens` reported `steps=9, input_tokens=699136, cache_read_tokens=1398662, output_tokens=9473, total_tokens=2107271` in `0.063s` (`< 2s`), matching an independent Python summation byte-for-byte. (In the sandbox, `--conversation` is blocked from opening `~/.gemini/antigravity/brain`, verified separately via `TestSource_OpenConversation` and `TestTranscriptIngest_WindowAndIdempotency` in temp homes.)
- Replay idempotency: verified in `TestTranscriptIngest_WindowAndIdempotency` (`nothing_new: true`, `saved: false`, totals unchanged).

## 4. Negative and failure cases
- No `--from-step` on first use → exit 1, nothing written (`TestTranscriptIngest_WindowAndIdempotency`).
- `--conversation ../evil`, uppercase/non-UUID, or unknown ID → exit 1, nothing opened or written (`TestSource_ValidateConversationID`, `TestTranscriptIngest_FlagsAndExitCodes`).
- `--transcript` pointing at a directory or a symlink → exit 1 (`TestSource_OpenFileAndKey`).
- Line > 1 MiB, input > MaxBytes, malformed JSON, partial usage, negative token (even out of window), missing/negative `step_index`, int64 overflow → error citing line number only, nothing written (`TestSummarizeErrors`, `TestNoContentLeak`).
- `--from-step` ≤ cursor, `--from-step` > `--to-step`, role mismatch, malformed `ingest` field in `state.json` → refused, nothing written (`TestRecordIngest`, `TestRecordIngest_Guards`).
- `--transcript` or `--conversation` combined with `--input`/`--cache`/`--output`/`--stdin`, `--transcript-root` without `--conversation`, negative `--from-step`/`--to-step`, invalid `--role` → exit 2 (`TestTranscriptIngest_FlagsAndExitCodes`).

## 5. Determinism check
Ran `go test -count=2 ./internal/transcript ./internal/state ./internal/cli`; all tests passed identically across both runs. Source key SHA-256 and JSON outputs are deterministic.

## 6. Deviations, known limitations, and feedback-rule compliance
Known limitations (by design, see requirements §3): Antigravity format only; no auto-fill of this document; no subagent auto-discovery; no concurrent-run lock. Deviations: none.

Feedback-rule compliance (SKILLS.md, "Human feedback rules"):
- `F-001`: pass (all 5 Loop 1 findings resolved and verified with 0 remaining findings in Loop 2 Review Triad).
- `F-002`: pass (verified zero personal paths in code, tests, docs, state, events, and JSON output via `TestTranscriptIngest_WindowAndIdempotency` and `./bin/sdd check --strict`).
- `F-003`: pass (measured coordinator telemetry recorded via dogfood `sdd record-tokens T-064 --transcript ... --from-step 23542 --role coordinator`).

## 7. Definition of done
- [x] Every requirement has evidence
- [x] Build, lint, and tests pass (commands above)
- [x] Negative and failure cases tested
- [x] Determinism confirmed
- [x] No files changed outside the plan; no unapproved dependencies
- [x] Living docs updated
- [x] Compliant with every active feedback rule
- [x] Token telemetry recorded via sdd task state record-tokens (or sdd record-tokens) and documented below (Rule F-003)
- [x] Suggested commit message provided

## 8. Token Metrics

| Agent / Role | Model | Input Tokens | Cache Read Tokens | Output Tokens | Total Tokens |
|---|---|---|---|---|---|
| Task Coordinator (measured, steps 23542..23733) | Gemini 4 Hydrogen / Claude Sonnet 5.5 | 1,670,575 | 11,259,428 | 94,988 | 13,024,991 |
| Subagents (2 Plan Reviewers + 6 Triad Reviewers) | pro / inherit / flash | unmeasured in sandbox | unmeasured in sandbox | unmeasured in sandbox | unmeasured in sandbox |
| **Total for T-064 (recorded)** | | **1,670,575** | **11,259,428** | **94,988** | **13,024,991** |
| **Cumulative Project Total** | | **2,130,975** | **13,119,428** | **221,938** | **15,472,341** |

*Note: Coordinator tokens were measured directly via `sdd record-tokens T-064 --transcript ... --from-step 23542 --role coordinator`. Subagent transcripts reside in separate conversation folders blocked by the sandbox (`operation not permitted`); they are marked unmeasured rather than estimated (Rule 10) and can be added after merge with `sdd record-tokens T-064 --conversation <id> --from-step 0 --role <role>`.*

## 9. Sign-off
Suggested commit message: `feat(state): ingest real token counters from IDE transcripts in sdd record-tokens (T-064)`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-064`
