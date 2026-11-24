---
status: active
---
# Plan — T-058 Ephemeral tabula-rasa reviewer lifecycle and zero-negotiation loop isolation

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Implementing

## 1. Approach summary

Eliminate the LLM "Repair-Loop Justification Bleed" and cognitive anchoring bias in the Review Triad by establishing the **Ephemeral Tabula-Rasa Reviewer Lifecycle** and **Zero-Negotiation Loop Isolation**:
1. **Multi-Loop Bundle Staging & Finding Isolation (`internal/audit/`):**
   - Provide `FindingItem` struct representing findings parsed from `review.md`.
   - Implement `ExtractFindingsToVerify(reviewContent string) ([]FindingItem, error)` parsing findings requiring verification.
   - Implement `FormatFindingsToVerify(findings []FindingItem) string` generating clean, objective markdown tables without developer commentary.
   - Extend `StageReviewBundle` supporting loop index `1..3` via `StageReviewBundleWithLoop`.
   - Add `Loop int json:"loop,omitempty"` to `ReviewBundle` (and `ReviewPromptContext`).
   - When loop > 1, stage `.sdd/review/<TASK_ID>/code/findings_to_verify.md` and include it in `manifest.json`.
2. **CLI Loop Flag & Handover Prompts (`internal/cli/prompt.go`):**
   - Extend `sdd prompt review <T-xxx>` with `--loop <1..3>` (default 1).
   - Enforce the 3-loop circuit breaker limit (reject `--loop < 1` or `--loop > 3`).
   - For remediation loops (loop 2 and 3), output tabula-rasa prompts instructing the fresh ephemeral reviewer to evaluate the candidate diff and verify previous findings objectively without conversational bias.
3. **Mechanical Consistency Validation (`internal/check/review.go`):**
   - Implement rule `remediation-hygiene` in `sdd check` scoped to active/non-`done` tasks (`normTaskStatus != "done"`):
     - Checks that `review.md` retains structured findings tables without conversational chat logs, negotiation banter, or colloquial excuses.
4. **Agent Contracts & Protocol Skills Alignment:**
   - Update repository skills (`skills/task-coordinator/SKILL.md`, `skills/code-review/SKILL.md`), embedded canonical role skills (`internal/templates/data/skills/coordinator/SKILL.md`, `internal/templates/data/skills/task-coordinator/SKILL.md`), and export constants in `internal/skill/export.go`.
   - Mandate that every review loop spawns fresh subagents (`invoke_subagent`) and strictly forbids reusing existing subagent sessions (`send_message`).
   - Update agent contracts (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`) and embedded source contract template `internal/templates/data/base/contract.md.tmpl`.
   - Bump `templates.Version` to `2.6.0` in `internal/templates/templates.go`, manifests, and tests per Hard Rule 16.
5. **Living Docs & Full Verification:**
   - Update `README.md`, `TECH_STACK.md`, `SKILLS.md`, and `ROADMAP.md`.
   - Comprehensive unit and integration test coverage; zero external dependencies (`D-007`).

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-049 | Cognitive bias immunity: unanchored traceability ledger, ephemeral reviewer spawning, and heterogeneous model routing | Option A: Full architectural triad hardening in M14 (T-057 traceability ledger, T-058 ephemeral loop isolation, T-059 model routing) / Option B: Prompt advisory only / Option C: Model diversity only | Option A: Structurally eliminates diff-anchoring, justification bleed, and model homogeneity for rigorous multi-agent verification. | Approved (2026-11-24) |

## 3. Design

### 1. Multi-Loop Findings Extraction Structures (`internal/audit/bundle.go`)
```go
type FindingItem struct {
    ID          string `json:"id"`
    Category    string `json:"category,omitempty"`
    Severity    string `json:"severity"`
    Finding     string `json:"finding"`
    Action      string `json:"action,omitempty"`
    Status      string `json:"status"`
}

// ExtractFindingsToVerify parses review.md and extracts all findings needing verification.
func ExtractFindingsToVerify(reviewContent string) ([]FindingItem, error)

// FormatFindingsToVerify formats findings into a clean, objective checklist markdown table.
func FormatFindingsToVerify(findings []FindingItem) string

// StageReviewBundleWithLoop stages an isolated review bundle for a specific remediation loop.
func StageReviewBundleWithLoop(root, taskID, lens string, loop int, now ...time.Time) (*ReviewBundle, error)

// ReviewBundle extended with Loop field
type ReviewBundle struct {
    TaskID    string    `json:"task_id"`
    Lens      string    `json:"lens"`
    Loop      int       `json:"loop,omitempty"`
    TaskDir   string    `json:"task_dir"`
    BundleDir string    `json:"bundle_dir"`
    CreatedAt time.Time `json:"created_at"`
    Files     []string  `json:"files"`
}
```

### 2. Finding Checklist Schema (`findings_to_verify.md`)
```markdown
# Previous Findings to Verify (Remediation Loop 2)

> **Zero-Negotiation Isolation:** The candidate code below must be evaluated objectively against requirements. Verify whether previous defect conditions are resolved without conversational bias.

| Finding ID | Lens / Category | Severity | Defect Description | Suggested Action | Status |
|---|---|---|---|---|---|
| F-001 | Spec | 🔴 Blocker | Missing negative test for invalid tokens | Add test case | Pending Verification |
```

### 3. CLI Prompt Dispatch (`internal/cli/prompt.go`)
- Command: `sdd prompt review <T-xxx> [--lens <spec|security|pattern|all>] [--loop <1..3>] [--json]`
- Loop validation: loop $\ge 1$ and loop $\le 3$. Exit 2 on out-of-range values.
- `ReviewPromptContext` extended with `Loop int json:"loop,omitempty"`.

### 4. Mechanical Reviewer Hygiene Rule (`internal/check/review.go`)
- Rule: `remediation-hygiene`
- Scoped strictly to active/non-`done` tasks (`normTaskStatus != "done"`): `Specifying`, `Awaiting plan approval`, `Implementing`, `Awaiting review`. Completed tasks are ignored to prevent retroactive failures.
- Verifies that `review.md` findings register does not contain conversational chat artifacts (e.g. `User:`, `Agent:`, `I fixed this because`, `Sorry for the mistake`).

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement `FindingItem`, `ExtractFindingsToVerify`, `FormatFindingsToVerify`, and `StageReviewBundleWithLoop` in `internal/audit/bundle.go`; write unit tests in `internal/audit/bundle_test.go`. | `internal/audit/bundle.go`, `internal/audit/bundle_test.go` | `go test -v ./internal/audit -run "TestExtractFindingsToVerify|TestStageReviewBundle_Loop"` |
| 2 | Update `sdd prompt review` with `--loop <1..3>` flag and tabula-rasa remediation prompt formatting in `internal/cli/prompt.go`; write unit tests in `internal/cli/prompt_test.go`. | `internal/cli/prompt.go`, `internal/cli/prompt_test.go` | `go test -v ./internal/cli -run TestPromptReview_Loop` |
| 3 | Implement mechanical `remediation-hygiene` validation in `internal/check/review.go` (scoped to non-done tasks) and author unit test suite in `internal/check/review_test.go`. | `internal/check/review.go`, `internal/check/review_test.go` | `go test -v ./internal/check -run TestCheckRemediationHygiene` |
| 4 | Update coordinator and review skill documentation across repository skills (`skills/task-coordinator/SKILL.md`, `skills/code-review/SKILL.md`), embedded canonical role skill templates (`internal/templates/data/skills/coordinator/SKILL.md`, `internal/templates/data/skills/task-coordinator/SKILL.md`), and export constants in `internal/skill/export.go` detailing the Ephemeral Tabula-Rasa Reviewer Lifecycle. | Skills, templates, export | `go test -v ./internal/skill -run TestExport` |
| 5 | Update agent contracts (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`) and embedded source contract template `internal/templates/data/base/contract.md.tmpl`. Bump `templates.Version` to `2.6.0` in `internal/templates/templates.go`, `.sdd/manifest.json`, and test fixtures per Hard Rule 16. | Contracts, templates, manifest | `go test -v ./internal/templates -run "TestVersion|TestContractTemplate"` |
| 6 | Update living documents (`README.md`, `TECH_STACK.md`, `SKILLS.md`, `ROADMAP.md`). | Living docs | `sdd check --strict` |
| 7 | Run full verification suite (`make verify`, determinism `-count=2`, `-race`, outside audit runner `sdd task audit T-058`), Plan Review subagent audit, GATE 1 approval, implementation, and review triad sign-off. | All | Full suite clean |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/audit/bundle.go` | modify | Implement `FindingItem`, `ExtractFindingsToVerify`, `FormatFindingsToVerify`, `StageReviewBundleWithLoop`, and stage `findings_to_verify.md` |
| `internal/audit/bundle_test.go` | modify | Unit tests for finding extraction, checklist formatting, and multi-loop bundle staging |
| `internal/cli/prompt.go` | modify | Update `sdd prompt review` with `--loop` flag and remediation prompt generation |
| `internal/cli/prompt_test.go` | modify | Unit tests for `--loop` flag parsing, validation, and prompt assertions |
| `internal/check/review.go` | modify | Implement mechanical `remediation-hygiene` rule in `sdd check` scoped to non-done tasks |
| `internal/check/review_test.go` | modify | Unit tests for `remediation-hygiene` rule (detecting conversational chatter in reviews) |
| `skills/task-coordinator/SKILL.md` | modify | Document Ephemeral Reviewer Lifecycle and Zero-Negotiation Protocol |
| `skills/code-review/SKILL.md` | modify | Document Ephemeral Reviewer Lifecycle and Zero-Negotiation Protocol |
| `internal/templates/data/skills/coordinator/SKILL.md` | modify | Update embedded canonical coordinator role skill with Ephemeral Reviewer Lifecycle |
| `internal/templates/data/skills/task-coordinator/SKILL.md` | modify | Update embedded task-coordinator skill with Ephemeral Reviewer Lifecycle |
| `internal/skill/export.go` | modify | Update canonical exported coordinator and review skill string constants |
| `internal/skill/export_test.go` | modify | Assert exported canonical skill contents |
| `internal/templates/data/base/contract.md.tmpl` | modify | Update embedded agent contract template with Ephemeral Reviewer Lifecycle and version 2.6.0 |
| `internal/templates/templates.go` | modify | Bump `templates.Version` to `2.6.0` |
| `internal/templates/templates_test.go` | modify | Assert `templates.Version == "2.6.0"` and contract template parity |
| `CLAUDE.md` | modify | Update contract version to `2.6.0` per Hard Rule 16 and codify ephemeral reviewer lifecycle |
| `GEMINI.md` | modify | Update contract version to `2.6.0` per Hard Rule 16 and codify ephemeral reviewer lifecycle |
| `AGENTS.md` | modify | Update contract version to `2.6.0` per Hard Rule 16 and codify ephemeral reviewer lifecycle |
| `.sdd/manifest.json` | modify | Update templates version to `2.6.0` |
| `README.md` | modify | Document ephemeral reviewer lifecycle and zero-negotiation remediation loops |
| `TECH_STACK.md` | modify | Verified facts log entry for T-058 |
| `SKILLS.md` | modify | Section 16: Ephemeral Tabula-Rasa Reviewer Lifecycle & Zero-Negotiation Standard |
| `ROADMAP.md` | modify | Update T-058 status tracking |
| `.sdd/state/T-058/**` | create | Task state and token telemetry |
| `tasks/features/T-058-ephemeral-reviewer-lifecycle/requirements.md` | modify | Task requirements |
| `tasks/features/T-058-ephemeral-reviewer-lifecycle/plan.md` | modify | Task plan |
| `tasks/features/T-058-ephemeral-reviewer-lifecycle/validation.md` | modify | Task validation |
| `tasks/features/T-058-ephemeral-reviewer-lifecycle/review.md` | modify | Task review |

## 6. Dependencies

None (100% standard library per `D-007`).

## 7. Living docs to update

- `ROADMAP.md`: update T-058 status (`Specifying` -> `Awaiting plan approval` -> `Implementing` -> `Awaiting review` -> `Done`).
- `TECH_STACK.md`: record verified facts entry for T-058.
- `SKILLS.md`: add Section 16 "Ephemeral Tabula-Rasa Reviewer Lifecycle & Zero-Negotiation Standard".
- `README.md`: document ephemeral reviewer lifecycle and zero-negotiation remediation loops.

## 8. Risks, mitigations, rollback

- **Risk:** Existing historical completed tasks in repository fail `sdd check` under new `remediation-hygiene` rule.
  - **Mitigation:** The checker targets active/non-`done` tasks (`normTaskStatus != "done"`), preventing retroactive failure of historical M0–M13 tasks.
- **Risk:** Remediation loops without findings (Loop 1) fail if `findings_to_verify.md` is expected.
  - **Mitigation:** `findings_to_verify.md` is only generated when `loop > 1` or when previous findings exist in `review.md`.

### Rollback
- Revert template version bump and restore `bundle.go`, `prompt.go`, and `review.go` changes.

## 9. Applicable human-feedback rules

- **F-001:** Resolve all review findings (Blockers, Warnings, Minors) across Review Triad before Gate 2 close.
- **F-002:** Zero personal absolute filesystem paths in generated code, templates, or documentation.
- **F-003:** Track and record session token consumption via `sdd record-tokens T-058`, document metrics in `validation.md`, and report at Gate 2.

## 10. Approval record
Approved by: Vikash (human)   Date: ______   Phrase received: `APPROVED T-058`
