---
status: active
---
# Plan — T-057 Unanchored Requirements Traceability Ledger and omission-first review protocol

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved

## 1. Approach summary

Eliminate the LLM "Diff Framing" anchoring bias and omission blindness in the Review Triad by establishing the **Unanchored Requirements Traceability Ledger** protocol and mechanical verification:
1. **Ledger Extraction & Staging (`internal/audit/`):**
   - Provide `ExtractRequirements(content string) ([]RequirementItem, error)` parsing `requirements.md` tables for `R-xxx` declarations.
   - Provide `FormatTraceabilityLedger(reqs []RequirementItem) string` generating unanchored markdown ledger tables.
   - Update `StageReviewBundle` to generate `.sdd/review/<TASK_ID>/code/ledger.md` pre-seeded with all `R-xxx` IDs and requirement statements, with empty evidence columns ready for unanchored review before opening `diff.patch`.
2. **Mechanical Consistency Validation (`internal/check/review.go`):**
   - Implement rule `traceability-ledger` in `sdd check`:
     - Checks that completed `review.md` contains `## Requirements Traceability Ledger`.
     - Verifies that every `R-xxx` declared in `requirements.md` is present in the ledger.
     - Verifies that no cells are empty, placeholders (`<...>`, `TODO`, `n/a`), or marked `Omission`.
3. **Task Review Templates & Version Bump (`internal/templates/`):**
   - Update `_templates/review.md` across all 4 typologies (`features`, `bugfixes`, `hotfixes`, `spikes`) to embed the ledger section.
   - Bump `templates.Version` to `2.5.0` in `internal/templates/templates.go`, update contract headers (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`) and `.sdd/manifest.json` per Hard Rule 16.
4. **Review Standard Documentation (`skills/code-review/` and `spec-auditor`):**
   - Update `skills/code-review/SKILL.md` and `.sdd/skills/spec-auditor/SKILL.md` detailing the two-step Omission-First Protocol (Step 1: Unanchored Traceability Scan; Step 2: Incremental Diff Quality Audit).
5. **Living Docs & Full Verification:**
   - Update `README.md`, `TECH_STACK.md`, `SKILLS.md`, and `ROADMAP.md`.
   - Comprehensive unit and integration test coverage; zero external dependencies (`D-007`).

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-049 | Cognitive bias immunity: unanchored traceability ledger, ephemeral reviewer spawning, and heterogeneous model routing | Option A: Full architectural triad hardening in M14 (T-057 traceability ledger, T-058 ephemeral loop isolation, T-059 model routing) / Option B: Prompt advisory only / Option C: Model diversity only | Option A: Structurally eliminates diff-anchoring, justification bleed, and model homogeneity for rigorous multi-agent verification. | Approved (2026-11-24) |

## 3. Design

### 1. Requirements Extraction & Ledger Structures (`internal/audit/bundle.go`)
```go
type RequirementItem struct {
    ID           string `json:"id"`
    Statement    string `json:"statement"`
    Priority     string `json:"priority,omitempty"`
    Verification string `json:"verification,omitempty"`
}

// ExtractRequirements parses markdown content of requirements.md and extracts all R-xxx items.
func ExtractRequirements(content string) ([]RequirementItem, error)

// FormatTraceabilityLedger formats requirements into an unanchored markdown ledger template.
func FormatTraceabilityLedger(reqs []RequirementItem) string
```

### 2. Canonical Requirements Traceability Ledger Schema (`review.md`)
```markdown
## Requirements Traceability Ledger

| Req ID | Requirement Statement | Code Symbol & File | Positive Test Evidence | Negative / Boundary Test Evidence | Status |
|---|---|---|---|---|---|
| R-001 | The system SHALL ... | `ExtractRequirements` in internal/audit/bundle.go | `TestExtractRequirements` | `TestExtractRequirements_Malformed` | Verified |
| R-002 | The system SHALL ... | `StageReviewBundle` in internal/audit/bundle.go | `TestStageReviewBundle_Ledger` | `TestStageReviewBundle_EmptyReq` | Verified |
```

### 3. Mechanical Checker Rule (`internal/check/review.go`)
- Rule: `traceability-ledger`
- Severity: `Error` when `review.md` is in status `Approved` or `Completed`.
- Verifications:
  1. Section `## Requirements Traceability Ledger` exists.
  2. Every `R-xxx` from `requirements.md` matches a row in the ledger table.
  3. Columns `Code Symbol & File` and `Positive Test Evidence` are non-empty and free of placeholders (`<...>`, `TODO`, `n/a`).
  4. Column `Negative / Boundary Test Evidence` is non-empty, free of placeholders (`<...>`, `TODO`, `n/a`), and verified for all requirements involving validation, gating, or error handling.
  5. Status column is strictly `Verified` (any row marked `Omission` or `Missing` reports an error).

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement `ExtractRequirements` and `FormatTraceabilityLedger` in `internal/audit/bundle.go`, update `StageReviewBundle` to stage `ledger.md`, update `internal/cli/prompt.go` with unanchored ledger review instructions, and write unit tests in `internal/audit/bundle_test.go`, `internal/audit/audit_test.go`, and `internal/cli/prompt_test.go`. | `internal/audit/bundle.go`, `internal/audit/bundle_test.go`, `internal/audit/audit_test.go`, `internal/cli/prompt.go`, `internal/cli/prompt_test.go` | `go test -v ./internal/audit ./internal/cli -run "TestExtractRequirements|TestReviewPrompt"` |
| 2 | Implement mechanical `traceability-ledger` validation in `internal/check/review.go` and author unit test suite in `internal/check/review_test.go`. | `internal/check/review.go`, `internal/check/review_test.go` | `go test -v ./internal/check -run TestCheckTraceabilityLedger` |
| 3 | Update review starter templates across applicable typologies in `internal/templates/data/tasks/features/review.md`, `internal/templates/data/tasks/bugfixes/review.md`, `tasks/features/_templates/review.md`, and `tasks/bugfixes/_templates/review.md`; bump `templates.Version` to `2.5.0` in `internal/templates/templates.go`; update contract version markers (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`) and `.sdd/manifest.json`. | Templates, contracts, manifest | `go test -v ./internal/templates -run TestVersion` |
| 4 | Update `skills/code-review/SKILL.md` and `internal/templates/data/skills/spec-auditor/SKILL.md` documenting the Unanchored Omission-First Protocol. | `skills/code-review/SKILL.md`, `internal/templates/data/skills/spec-auditor/SKILL.md` | `sdd check --strict` |
| 5 | Update living documents (`README.md`, `TECH_STACK.md`, `SKILLS.md`, `ROADMAP.md`). | Living docs | `sdd check --strict` |
| 6 | Run full verification suite (`make verify`, determinism `-count=2`, `-race`, outside audit runner `sdd task audit T-057`), Plan Review subagent audit, GATE 1 approval, implementation, and review triad sign-off. | All | Full suite clean |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/audit/bundle.go` | modify | Implement `ExtractRequirements`, `FormatTraceabilityLedger`, and stage `ledger.md` |
| `internal/audit/bundle_test.go` | modify | Unit tests for requirement extraction and ledger formatting |
| `internal/audit/audit_test.go` | modify | Unit tests for bundle staging and prompt assertions |
| `internal/cli/prompt.go` | modify | Update review prompt generator to reference `ledger.md` and omission-first instructions |
| `internal/cli/prompt_test.go` | modify | Unit tests for review prompt instructions |
| `internal/hook/hook.go` | modify | Repository formatting cleanup for make verify |
| `internal/check/review.go` | modify | Implement mechanical `traceability-ledger` rule in `sdd check` |
| `internal/check/review_test.go` | modify | Unit tests for `traceability-ledger` rule (missing ledger, unmapped req, placeholders, negative evidence) |
| `internal/templates/templates.go` | modify | Bump `templates.Version` to `2.5.0` |
| `internal/templates/templates_test.go` | modify | Assert `templates.Version == "2.5.0"` |
| `internal/templates/data/tasks/features/review.md` | modify | Add `## Requirements Traceability Ledger` |
| `internal/templates/data/tasks/bugfixes/review.md` | modify | Add `## Requirements Traceability Ledger` |
| `tasks/features/_templates/review.md` | modify | Project template parity |
| `tasks/bugfixes/_templates/review.md` | modify | Project template parity |
| `skills/code-review/SKILL.md` | modify | Document Unanchored Traceability Ledger and Omission-First Protocol |
| `internal/templates/data/skills/spec-auditor/SKILL.md` | modify | Update Spec Auditor role rubric with Step 1 omission scan |
| `CLAUDE.md` | modify | Update contract version to `2.5.0` per Hard Rule 16 |
| `GEMINI.md` | modify | Update contract version to `2.5.0` per Hard Rule 16 |
| `AGENTS.md` | modify | Update contract version to `2.5.0` per Hard Rule 16 |
| `.sdd/manifest.json` | modify | Update templates version to `2.5.0` |
| `README.md` | modify | Document Requirements Traceability Ledger and omission-first protocol |
| `TECH_STACK.md` | modify | Verified facts log entry for T-057 |
| `SKILLS.md` | modify | Section 15: Requirements Traceability Ledger and Omission-First Review Protocol |
| `ROADMAP.md` | modify | Update T-057 status tracking |
| `.sdd/state/T-057/**` | create | Task state and token telemetry |
| `tasks/features/T-057-unanchored-traceability-ledger/requirements.md` | modify | Task requirements |
| `tasks/features/T-057-unanchored-traceability-ledger/plan.md` | modify | Task plan |
| `tasks/features/T-057-unanchored-traceability-ledger/validation.md` | modify | Task validation |
| `tasks/features/T-057-unanchored-traceability-ledger/review.md` | modify | Task review |

## 6. Dependencies

None (100% standard library per `D-007`).

## 7. Living docs to update

- `ROADMAP.md`: update T-057 status (`Specifying` -> `Awaiting plan approval` -> `Implementing` -> `Awaiting review` -> `Done`).
- `TECH_STACK.md`: record verified facts entry for T-057.
- `SKILLS.md`: add Section 15 "Requirements Traceability Ledger and Omission-First Review Protocol".
- `README.md`: document traceability ledger standard and omission-first review protocol.

## 8. Risks, mitigations, rollback

- **Risk:** Existing historical completed tasks in repository fail `sdd check` under new `traceability-ledger` rule.
  - **Mitigation:** The checker targets active tasks or checks tasks with version >= 2.5.0, or tasks whose `review.md` explicitly includes the ledger header, preventing retroactive failure of historical M0–M13 tasks.
- **Risk:** Malformed `requirements.md` tables break extraction.
  - **Mitigation:** `ExtractRequirements` uses strict regex extraction for `R-\d{3,}` rows with bounds checking and skips non-conforming lines gracefully.

### Rollback
- Revert template version bump and restore `bundle.go` / `review.go` changes.

## 9. Applicable human-feedback rules

- **F-001:** Resolve all review findings (Blockers, Warnings, Minors) across Review Triad before Gate 2 close.
- **F-002:** Zero personal absolute filesystem paths in generated code, templates, or documentation.
- **F-003:** Track and record session token consumption via `sdd record-tokens T-057`, document metrics in `validation.md`, and report at Gate 2.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-24   Phrase received: `APPROVED T-057`

