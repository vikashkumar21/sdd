---
status: completed
---
# Requirements — T-064 Native IDE transcript discovery and ingestion in sdd record-tokens

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-26)
Roadmap ref: `ROADMAP.md` T-064

## 1. Goal
Let `sdd record-tokens` read the real per-step token counters from an IDE conversation transcript, scoped to one task and safe to re-run, so nobody has to type or estimate them.

## 2. Context and references
- Mission / principles touched: §1 Determinism, §2 Fail safe, §4 Standard library only, §6 Living documents (F-003 token telemetry).
- Decisions this relies on: `D-050` (Approved 2026-11-25); `D-051` (transcript source and trust boundary: option B), `D-052` (task windowing and idempotent cursor: option A), `D-053` (counted records and subagents: option A) — each decided by the human on 2026-11-26 and recorded in `TECH_STACK.md` §6; `D-007` (stdlib only).
- Depends on tasks: `T-060` (Done), `T-039`/`T-049`/`T-055` (existing `internal/state` token accumulation and `sdd record-tokens`; Done).
- Verified facts (2026-11-26, one Antigravity transcript on the maintainer's Mac): the compact `transcript.jsonl` has one JSON object per line. Records with `source:"MODEL"` and `type:"PLANNER_RESPONSE"` carry integer `input_tokens`, `cache_read_tokens`, `output_tokens`; `SYSTEM`/`CHECKPOINT` records also carry them (meaning not verified, so not counted); the `model` field is present on only a few recent records; one transcript spans many tasks; each subagent has its own transcript. Largest line seen: about 8.6 KB (compact) and 51 KB (full). Other IDEs' formats are NOT verified.

## 3. Scope
**In scope**
- New package `internal/transcript`: strict, bounded JSONL summariser over a step window, and strict source resolution (explicit file or validated conversation ID under a configurable root).
- `sdd record-tokens <T-xxx>` (and `sdd task state record-tokens`) gain `--conversation <id>`, `--transcript <file>`, `--transcript-root <dir>`, `--from-step`, `--to-step`, `--role`, `--dry-run`.
- A per-task, per-source ingested-step cursor stored in `.sdd/state/<T-xxx>/state.json`, updated atomically together with the token totals, so re-running never double-counts.
- A `token_ingested` event in `events.jsonl`.
- Documentation of the Antigravity-only support and the coordinator workflow (`README.md`, `SKILLS.md`, `TECH_STACK.md`, `ROADMAP.md`).

**Out of scope**
- Claude Code, Cursor, Gemini CLI or any other transcript format (unverified).
- Directory scanning, globbing, or "newest transcript" discovery.
- Automatic discovery of subagent conversation IDs from a parent transcript.
- Counting `SYSTEM`/`CHECKPOINT` records, or per-model attribution.
- Writing `validation.md` §8 automatically (parked: follow-up task).
- Coordinator skill, prompt, or contract text changes (so `templates.Version` stays `2.11.0`; parked: follow-up task).
- Any network access; any change to the manual `--input/--cache/--output/--stdin` behaviour.

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL provide `transcript.Summarize(r io.Reader, opts Options) (Summary, error)` that sums `input_tokens`, `cache_read_tokens` and `output_tokens` over records with `source=="MODEL"` and `type=="PLANNER_RESPONSE"` whose `step_index` lies in `[FromStep, ToStep]` (`ToStep` open-ended when `NoLimit`), skips and counts in-window `MODEL`/`PLANNER_RESPONSE` records where all three usage fields are absent/null, and reports steps counted and the highest `step_index` seen in the file. Records of any other source or type SHALL NOT be counted. | MUST | Unit tests (table-driven, synthetic JSONL) |
| R-002 | The system SHALL fail `Summarize` with an error (returning no partial result) on: malformed JSON, a negative/non-integer/partial token field, a missing/negative/non-integer `step_index`, int64 overflow while summing, a line longer than `MaxLineBytes` (default 1 MiB), or input longer than `MaxBytes` (default 256 MiB). Errors SHALL cite the line number only, never transcript content. | MUST | Unit tests for each failure |
| R-003 | The system SHALL accept exactly one source per ingestion: `--transcript <file>` (an existing regular file; symlinks and directories rejected) or `--conversation <id>` (strict lowercase UUID) resolved as `<root>/<id>/.system_generated/logs/transcript.jsonl` through `os.OpenRoot` confinement, where `<root>` is `--transcript-root` or `<home>/.gemini/antigravity/brain` (home from `IO.Home`; empty home errors). The system SHALL NOT list, glob, or scan directories. | MUST | Unit + CLI tests (traversal ids, symlink, directory, missing file, empty home, both/neither flag) |
| R-004 | The system SHALL require `--from-step` on the first ingestion of a source for a task, default `--from-step` to `cursor+1` on later ingestions of the same source, and reject an explicit `--from-step` ≤ stored cursor, `--from-step` > `--to-step`, negative step flags, or a `--role` different from the one stored for that source without writing. If the window yields 0 counted steps, the system SHALL report "nothing new", write nothing, and exit 0. | MUST | CLI + state tests, including re-run idempotency |
| R-005 | The system SHALL, on a non-empty ingestion, add the summed delta to `tokens` and set the source cursor (`last_step` = `min(requestedTo, LastStepSeen)` or `LastStepSeen` when open-ended) in ONE atomic `state.json` write, then append one `token_ingested` event. State, events and command output SHALL NOT contain absolute filesystem paths: the source key is the conversation ID, or `file:` + the first 16 hex characters of the SHA-256 of the cleaned absolute path. | MUST | State + CLI tests (assert no path substring; cursor and tokens change together) |
| R-006 | The system SHALL support `--dry-run` (computing and printing text or `--json` without writing), validate `--role` against `^[a-z][a-z0-9-]{0,39}$` (default `coordinator`), reject invalid flag combinations with exit 2 (mutually exclusive with `--input`/`--cache`/`--output`/`--stdin`; `--transcript-root` requires `--conversation`; `--from-step`/`--to-step`/`--role`/`--dry-run` require a transcript source), use exit codes 0 (success/nothing new), 1 (runtime/validation error), 2 (usage error), and leave manual `--input`/`--cache`/`--output`/`--stdin` behaviour unchanged. | MUST | CLI tests incl. regression of existing manual tests |
| R-007 | The system SHALL decode only the six fields it needs (`step_index`, `source`, `type`, `input_tokens`, `cache_read_tokens`, `output_tokens`) from each line and SHALL NOT retain, print, or log any other transcript content (prompts, thinking, tool arguments). | MUST | Unit + CLI test with a sentinel string in `content` asserting it never appears in output, state, events, or errors |
| R-008 | The system SHALL use only the Go standard library (`D-007`), pass `go test -race` and `-count=2`, `make verify`, package-scoped `golangci-lint` with 0 issues, and `./bin/sdd check --strict` with 0 errors and 0 warnings. No template or contract file SHALL change, so `templates.Version` stays `2.11.0`. | MUST | `make verify`; race; count=2; lint; `git diff --stat` shows no `internal/templates/` change |
| R-009 | The system SHALL document in `README.md`, `SKILLS.md`, and `TECH_STACK.md` the new flags, the Antigravity-only scope, and the coordinator workflow (`--conversation <own id> --from-step N`; one run per subagent with `--role`). | MUST | Inspection + `sdd check --strict` |

## 5. Non-functional requirements
- **Determinism:** same transcript and window give byte-identical output and state; no clock reads except the injected `IO.Now`; no map-order dependence in output (sorted keys / fixed field order).
- **Fail-safe:** any ambiguity (no `--from-step` on first use, overlapping window, role mismatch, unreadable or oversized input, partial usage) refuses and writes nothing.
- **Performance:** the 30 MB / 23k-line real compact transcript summarises in under 2 s (measured, not assumed), backed by a synthetic `BenchmarkSummarize`.
- **Security and privacy:** bounded reads (R-002); confined open (R-003); no personal paths persisted (R-005, F-002); no content retained (R-007).
- **Logging:** none beyond the command's own output.

## 6. Constraints and assumptions
- Assumption (verified once, 2026-11-26): the Antigravity default root is `~/.gemini/antigravity/brain` and the compact file is `<id>/.system_generated/logs/transcript.jsonl`. On other machines or versions the user passes `--transcript-root` or `--transcript`.
- `input_tokens` is summed as reported; this task does not interpret whether it includes cached tokens.
- Concurrent ingestion runs for the same task are unsupported (no file lock); the cursor check is repeated inside the atomic write to narrow the window.

## 7. Open questions
None. (`D-051`, `D-052`, `D-053` decided by the human on 2026-11-26 and recorded in `TECH_STACK.md` §6.)

## 8. Acceptance criteria
- Given a transcript with MODEL steps 10–20 and a first run `--from-step 10`, when it is run twice, then the second run reports "nothing new" and `tokens` is unchanged.
- Given a first run with no `--from-step`, then it exits 1 and writes nothing.
- Given `--conversation ../x` or a non-UUID, then it exits 1 and opens nothing.
- Given a negative token value on any line, then it exits 1 and writes nothing.
- Given `--dry-run`, then state and events are byte-identical before and after.
- Given the real transcript, when the same window is summed by an independent script, then both totals match exactly.
