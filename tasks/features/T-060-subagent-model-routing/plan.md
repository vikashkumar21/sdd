---
status: draft
---
# Plan — T-060 Subagent model routing, tier enforcement, and calibrated reasoning dispatch

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Draft | Awaiting approval (GATE 1)

## 1. Approach summary

Operationalize the heterogeneous model routing specifications introduced in `T-059` by embedding mandatory `Model` parameter dispatch rules into Task Coordinator governance skills, agent contracts, and `sdd prompt` handover generators. Implement mechanical verification in `internal/check` to guarantee that coordinator skills and contracts enforce tier-based subagent dispatch, preventing token exhaustion and cognitive anchoring.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-050 | Token consumption mitigation, context hygiene, and provider-aligned caching architecture | Option A: Subagent model routing & tier enforcement in skills and contracts, lens-scoped review bundles, provider-aligned prompt prefix caching, living document progressive disclosure, and native transcript telemetry ingestion. Option B: Ad-hoc manual prompts without contract or mechanical enforcement. | Option A (determinstic, enforceable, measurable token savings) | Proposed (at Gate 1) |

## 3. Design

### Tier to Runtime Dispatch Mappings:
- **`frontier` (High Reasoning):**
  - Tool Invocation: `invoke_subagent(Role="...", Model="inherit")` (or `"pro"`)
  - Target Roles: `coordinator`, `architect`, `codebase-analyst`, `discovery-challenger`, `appsec-specialist`
- **`advanced` (Medium-High Reasoning):**
  - Tool Invocation: `invoke_subagent(Role="...", Model="inherit")` (or `"pro"`)
  - Target Roles: `plan-reviewer`, `implementer`
- **`standard` (Medium-Low Reasoning / High Verification Speed):**
  - Tool Invocation: `invoke_subagent(Role="...", Model="flash")`
  - Target Roles: `spec-auditor`, `pattern-auditor`, `qa`, `devops`
- **`fast` (Low/No Reasoning / Rapid Scans):**
  - Tool Invocation: `invoke_subagent(Role="...", Model="flash_lite")` (or `"flash"`)
  - Target Roles: mechanical lint / formatting sweeps

### Handover Prompt Generator (`internal/cli/prompt.go`):
- For each role prompt, append an executable dispatch snippet:
  ```json
  // Antigravity / Gemini Dispatch
  invoke_subagent(Role="Independent Spec Auditor", Model="flash", Prompt="...")
  ```

### Mechanical Checker Rule (`internal/check/coordinator.go`):
- Rule: `coordinator-tier-dispatch`
- Scans `skills/task-coordinator/SKILL.md` and agent contracts to verify:
  1. Mentions of `spec-auditor` and `pattern-auditor` mandate `flash` / standard tier dispatch.
  2. Forbids uncalibrated frontier inheritance for routine verification roles.
  3. Gracefully skips without errors if `skills/task-coordinator/SKILL.md` is absent, maintaining full backward compatibility with legacy (`dbguard-kit`) and greenfield kits.

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Update coordinator skill (`skills/task-coordinator/SKILL.md`) and code review skill (`skills/code-review/SKILL.md`) with explicit `invoke_subagent` dispatch snippets and tier mappings. | `skills/task-coordinator/SKILL.md`, `skills/code-review/SKILL.md` | `git grep -n "Model" skills/task-coordinator/SKILL.md` |
| 2 | Synchronize embedded canonical role skill templates (`internal/templates/data/skills/coordinator/SKILL.md`, `internal/templates/data/skills/task-coordinator/SKILL.md`). | `internal/templates/data/skills/coordinator/SKILL.md`, `internal/templates/data/skills/task-coordinator/SKILL.md` | `go test -v ./internal/skill -run TestExport` |
| 3 | Update agent contracts (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`) and source contract template (`internal/templates/data/base/contract.md.tmpl`) with Tier-Calibrated Subagent Dispatch Rule in Section 2 and Section 4. | `CLAUDE.md`, `GEMINI.md`, `AGENTS.md`, `internal/templates/data/base/contract.md.tmpl` | `go test -v ./internal/check -run TestCheckContractsMatch` |
| 4 | Enhance `sdd prompt review`, `sdd prompt plan-review`, and `sdd prompt implement` in `internal/cli/prompt.go` to render concrete `invoke_subagent` execution blocks with `Model` parameter; add unit tests in `internal/cli/prompt_test.go`. | `internal/cli/prompt.go`, `internal/cli/prompt_test.go` | `go test -v ./internal/cli -run TestPromptReview_ModelRouting` |
| 5 | Implement mechanical `coordinator-tier-dispatch` check in `internal/check/coordinator.go`, register in `internal/check/check.go`, and author unit test suite in `internal/check/coordinator_test.go`. | `internal/check/coordinator.go`, `internal/check/check.go`, `internal/check/coordinator_test.go` | `go test -v ./internal/check -run TestCheckCoordinatorTierDispatch` |
| 6 | Bump `templates.Version` to `2.8.0` in `internal/templates/templates.go`, `.sdd/manifest.json`, and update test fixtures in `internal/templates/templates_test.go` and `internal/upgrade/upgrade_test.go`. | `internal/templates/templates.go`, `.sdd/manifest.json`, `internal/templates/templates_test.go`, `internal/upgrade/upgrade_test.go` | `go test -v ./internal/templates ./internal/upgrade` |
| 7 | Update living docs (`ROADMAP.md`, `TECH_STACK.md`, `SKILLS.md`, `README.md`). | `ROADMAP.md`, `TECH_STACK.md`, `SKILLS.md`, `README.md` | `./bin/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `skills/task-coordinator/SKILL.md` | modify | Mandate model tier parameter in `invoke_subagent` dispatch |
| `skills/code-review/SKILL.md` | modify | Document reviewer model tier calibration |
| `internal/templates/data/skills/coordinator/SKILL.md` | modify | Sync coordinator template with tier dispatch |
| `internal/templates/data/skills/task-coordinator/SKILL.md` | modify | Sync task coordinator template with tier dispatch |
| `internal/templates/data/base/contract.md.tmpl` | modify | Codify Tier-Calibrated Subagent Dispatch Rule in contract template |
| `CLAUDE.md` | modify | Contract rule synchronization |
| `GEMINI.md` | modify | Contract rule synchronization |
| `AGENTS.md` | modify | Contract rule synchronization |
| `internal/cli/prompt.go` | modify | Output concrete `invoke_subagent` snippet with `Model` parameter |
| `internal/cli/prompt_test.go` | modify | Unit tests for dispatch snippet rendering |
| `internal/check/coordinator.go` | create | Mechanical rule `coordinator-tier-dispatch` |
| `internal/check/coordinator_test.go` | create | Unit tests for coordinator tier dispatch checker |
| `internal/check/check.go` | modify | Wire and register `coordinator-tier-dispatch` rule in `c.run()` |
| `internal/templates/templates.go` | modify | Bump `templates.Version` to `2.8.0` |
| `.sdd/manifest.json` | modify | Update template version to `2.8.0` |
| `internal/templates/templates_test.go` | modify | Update version test assertions |
| `internal/upgrade/upgrade_test.go` | modify | Update version test assertions |
| `ROADMAP.md` | modify | Update task status |
| `TECH_STACK.md` | modify | Record decision D-050 approval and verified fact |
| `SKILLS.md` | modify | Document Tier-Calibrated Subagent Dispatch standard |
| `README.md` | modify | Document subagent model routing and cost controls |
| `tasks/features/T-060-subagent-model-routing/requirements.md` | modify | Task requirements |
| `tasks/features/T-060-subagent-model-routing/plan.md` | modify | Task plan |
| `tasks/features/T-060-subagent-model-routing/validation.md` | modify | Task validation |
| `tasks/features/T-060-subagent-model-routing/review.md` | modify | Task review |

## 6. Dependencies

None (100% standard library per `D-007`).

## 7. Living docs to update

- `ROADMAP.md`: update T-060 status (`Specifying` -> `Awaiting plan approval` -> `Implementing` -> `Awaiting review` -> `Done`).
- `TECH_STACK.md`: record approved decision `D-050` and verified fact for T-060.
- `SKILLS.md`: add Section 18 "Tier-Calibrated Subagent Dispatch & Token Optimization Standard".
- `README.md`: document subagent tier routing and token optimization.

## 8. Risks, mitigations, rollback

- **Risk:** Older subagent harnesses fail if `Model` parameter is passed.
  - **Mitigation:** The prompt outputs provider-agnostic dispatch comments explaining fallback to `inherit` when unsupported.
- **Risk:** Quality regression on Spec Auditor or Pattern Auditor using `flash`.
  - **Mitigation:** Verification requirements are structured and tabular (checklist and regex validation), which `flash` models handle with high fidelity while avoiding verbose stream bloat.
- **Risk:** Legacy kits (e.g. `dbguard-kit`) or greenfield kits without `skills/task-coordinator/SKILL.md` fail `sdd check`.
  - **Mitigation:** `coordinator-tier-dispatch` check inspects file existence first and skips cleanly if absent.

### Rollback
- Revert template version bump and restore original skills, contracts, and prompt files.

## 9. Applicable human-feedback rules

- **F-001:** Resolve all review findings across Review Triad before Gate 2 close.
- **F-002:** Zero personal absolute filesystem paths in generated code, templates, or documentation.
- **F-003:** Track and record session token consumption via `sdd record-tokens T-060`, document metrics in `validation.md`, and report at Gate 2.

## 10. Approval record
Approved by: Vikash (human)   Date: ______   Phrase received: `APPROVED T-060`
