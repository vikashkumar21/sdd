---
status: active
---
# Requirements — T-058 Ephemeral tabula-rasa reviewer lifecycle and zero-negotiation loop isolation

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Implementing
Roadmap ref: `ROADMAP.md` T-058
Decision ref: `TECH_STACK.md` D-049 (Approved 2026-11-24)

## 1. Goal

Eliminate the LLM "Repair-Loop Justification Bleed" and cognitive anchoring bias in the Review Triad by codifying the **Ephemeral Tabula-Rasa Reviewer Lifecycle** (forbidding reviewer conversation reuse across remediation loops), enforcing **Zero-Negotiation Loop Isolation** (staging candidate diffs with pure finding checklists and zero developer prose), enhancing `internal/audit` and `sdd prompt review` with loop-aware bundle generation, and enforcing mechanical review hygiene in `sdd check`.

## 2. Context and references

- Mission / principles: Principle 1 (Deterministic by construction), Principle 2 (Fail safe), Principle 3 (Human in the loop for the irreversible).
- Decisions: D-003 / D-007 (Go standard library only), D-026 / D-031 / D-032 / D-033 (Code review standard, Review Triad, Rule F-001), D-044 (Stateless review context isolation), D-049 (Cognitive bias immunity: unanchored auditing, ephemeral reviewer spawning, heterogeneous model routing).
- Depends on: T-057 (Unanchored Requirements Traceability Ledger and omission-first review protocol).
- Active Feedback Rules: F-001 (Resolve all findings before Gate 2), F-002 (Zero personal absolute paths), F-003 (Token telemetry).

## 3. Scope

### In scope
1. **Ephemeral Tabula-Rasa Reviewer Lifecycle Protocol (`R-001`):**
   - Codify the Ephemeral Reviewer Lifecycle in `skills/task-coordinator/SKILL.md`, `skills/code-review/SKILL.md`, and agent contract templates (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`).
   - Forbid reusing reviewer subagent conversations (`send_message` to previous reviewers) across remediation loops.
   - Mandate that each loop (Loop 1 initial audit, Loop 2 & Loop 3 remediation audits) must dispatch brand new, independent subagent instances (`invoke_subagent`) without prior conversational memory.
2. **Zero-Negotiation Loop Isolation (`R-002`):**
   - Eliminate developer justification bleed by mandating that remediation prompts and review bundles contain zero developer narrative, excuses, apologies, or negotiation transcripts.
   - Reviewers in remediation loops evaluate purely against the refreshed candidate diff (`diff.patch`), original requirements (`spec.md`), unanchored traceability ledger (`ledger.md`), and an isolated checklist of previous findings to verify (`findings_to_verify.md`).
3. **Automated Multi-Loop Review Bundle Staging (`R-003`):**
   - In `internal/audit/bundle.go`, implement `ExtractFindingsToVerify(reviewContent string) ([]FindingItem, error)` parsing previous findings from `review.md`.
   - Update `StageReviewBundle` (or provide `StageReviewBundleWithLoop`) supporting loop index (1..3).
   - When loop > 1 or prior findings exist, generate `.sdd/review/<TASK_ID>/code/findings_to_verify.md` and record it in `manifest.json`.
4. **Loop-Aware Review Prompt Generator (`R-004`):**
   - Extend `sdd prompt review <T-xxx>` in `internal/cli/prompt.go` with `--loop <1..3>` flag (default 1).
   - When `--loop` is 2 or 3, output a specialized tabula-rasa remediation handover prompt instructing the new ephemeral reviewer to audit candidate diffs objectively against requirements and verify whether previous finding conditions are resolved without conversational bias.
5. **Mechanical Verification in `sdd check` (`R-005`):**
   - Implement rule `remediation-hygiene` (or `reviewer-isolation`) in `internal/check/review.go`:
     - Flags error if `review.md` contains conversational negotiation transcripts, developer excuses, or unescaped chat logs.
     - Enforces that findings retain structured table formatting without conversational drift.
6. **Agent Contract Alignment & Version Bump (`R-006`):**
   - Update contract templates (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`) and coordinator skill documentation to mandate ephemeral subagent spawning and zero-negotiation bundles.
   - Bump `templates.Version` to `2.6.0` in `internal/templates/templates.go`, contracts, manifests, and test fixtures per Hard Rule 16.
7. **Standard Library Purity (`R-007`):**
   - 100% Go standard library implementation (`D-007`); zero external dependencies.

### Out of scope
- Dynamic model routing and provider dispatch (addressed in T-059).
- Automated git worktree re-provisioning per loop (handled by coordinator in current worktree).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL codify the Ephemeral Reviewer Lifecycle standard in coordinator and review skills and contract templates, strictly forbidding reviewer subagent conversation reuse across remediation loops. | MUST | inspection / check |
| R-002 | The system SHALL enforce Zero-Negotiation Loop Isolation, requiring remediation loops to present candidate diffs and structured finding checklists with zero developer prose, excuses, or negotiation transcripts. | MUST | test / inspection |
| R-003 | The system SHALL implement `ExtractFindingsToVerify` and stage an objective `findings_to_verify.md` in `.sdd/review/<TASK_ID>/code/` for remediation loops (loop > 1). | MUST | test |
| R-004 | The system SHALL support `--loop <1..3>` in `sdd prompt review`, generating tabula-rasa remediation prompts that instruct fresh ephemeral reviewers without conversational anchoring. | MUST | test |
| R-005 | The system SHALL implement the mechanical `remediation-hygiene` rule in `sdd check`, reporting errors for conversational transcripts or negotiation chatter in `review.md`. | MUST | test |
| R-006 | The system SHALL update agent contract templates (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`), bump `templates.Version` to `2.6.0`, and verify fresh kit parity per Hard Rule 16. | MUST | test |
| R-007 | The system SHALL implement all bundle staging, finding extraction, prompt formatting, and verification logic using only the Go standard library with zero third-party dependencies (`D-007`). | MUST | test |

## 5. Non-functional requirements

- **Determinism:** `ExtractFindingsToVerify` extracts findings in deterministic source order. Multi-loop bundle staging produces identical outputs for identical inputs.
- **Fail-safe:** If an invalid `--loop` value (e.g. `< 1` or `> 3`) is passed, `sdd prompt review` exits with code 2 and a clear usage error.
- **Performance budget:** Multi-loop bundle staging and finding extraction add < 15ms.

## 6. Constraints and assumptions

- The 3-loop circuit breaker remains the hard ceiling for remediation iterations.
- Existing historical tasks retain their completed review records; `remediation-hygiene` applies to active/new tasks.

## 7. Open questions

(none — Decision D-049 Approved on 2026-11-24)

## 8. Acceptance criteria

- **AC-1 (Ephemeral Dispatch Standard):** `skills/task-coordinator/SKILL.md` and contract templates explicitly specify that each remediation loop spawns a fresh subagent with zero conversational history.
- **AC-2 (Zero-Negotiation Bundle):** Running `sdd prompt review T-058 --loop 2` stages `.sdd/review/T-058/code/findings_to_verify.md` containing only structured finding IDs, severities, and verification targets without developer commentary.
- **AC-3 (Loop Validation):** Passing `--loop 4` to `sdd prompt review` fails with an invalid loop error (enforcing the 3-loop circuit breaker limit).
- **AC-4 (Review Hygiene Audit):** A `review.md` containing colloquial negotiation chatter or conversational transcripts fails `sdd check --strict` under rule `remediation-hygiene`.
- **AC-5 (Template Version Bump):** `templates.Version` is bumped to `2.6.0`, contracts match `templates=2.6.0`, and regenerating a test kit in a temporary directory passes `sdd check --strict`.
- **AC-6 (Standard Library Purity):** Zero external dependencies added to `go.mod`.
