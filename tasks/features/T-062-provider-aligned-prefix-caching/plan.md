---
status: draft
---
# Plan — T-062 Provider-aligned prompt prefix caching and dynamic suffix pinning

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-25)

## 1. Approach summary
Reorganize all SDD prompt builders (`plan-review`, `implement`, `review`, `discovery`, `repair`, and base agent prompts) into two strictly partitioned segments:
1. **Invariant Static Prefix ($\ge 1,024$ tokens / $\ge 4,096$ bytes, 4,200 bytes safety target):** SDD Anthem & Principles, Canonical Role Skill instructions, standard evaluation rubrics, model routing guidance, approved architectural decisions, and active human feedback rules. This prefix is byte-stable across tasks, sessions, and remediation loops, guaranteeing maximum KV cache hit rates on Anthropic, OpenAI, and Gemini APIs. Note: `prompt_compact.md.tmpl` is exempt from the 1,024-token floor due to its intentional $< 500$-token budget (T-049), but strictly enforces static-first ordering.
2. **Dynamic Task Suffix:** Task ID, typology, directory paths, loop counters, planned/changed file lists, staged bundle locations (`.sdd/review/<TASK_ID>/...`), defect checklists (`findings_to_verify.md`), and step-by-step task execution instructions.

We introduce a dedicated prompt caching utility (`internal/cli/prompt_cache.go`), restructure prompt templates and Go prompt generators, adapt existing test assertions in `internal/cli/cli_test.go` and `internal/cli/prompt_repair_test.go`, add comprehensive unit tests verifying byte invariance and token length thresholds, and bump `templates.Version` to `2.10.0` per Hard Rule 16.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-050 | Token consumption mitigation, context hygiene, and provider-aligned caching architecture | A) Reorganize prompts to enforce $\ge 1,024$-token static invariant prefixes with dynamic metadata pinned to suffixes.<br>B) Dynamic prompts without structural prefix/suffix separation. | Option A: Invariant prefixes maximize provider caching (up to 90% cost and latency savings) with byte-level determinism. | **Approved** 2026-11-25 |

## 3. Design
Signatures and components only; no implementation code before GATE 1.

### 3.1 Prompt Caching Architecture (`internal/cli/prompt_cache.go`)
```go
// MinStaticPrefixTokens defines the minimum token count required to activate
// provider prefix caching across Anthropic, OpenAI, and Gemini (1,024 tokens).
const MinStaticPrefixTokens = 1024

// MinStaticPrefixBytes defines the deterministic byte floor (4,096 UTF-8 bytes,
// with a 4,200 byte safety threshold) guaranteeing >= 1,024 tokens across standard LLM tokenizers.
const MinStaticPrefixBytes = 4096

// PromptParts represents a partitioned prompt separated at the cache boundary.
type PromptParts struct {
    StaticPrefix string
    DynamicSuffix string
}

// FullPrompt concatenates the static prefix and dynamic suffix.
func (p PromptParts) FullPrompt() string

// EstimateTokens calculates an approximate token count from UTF-8 string content
// using deterministic character/byte heuristics.
func EstimateTokens(s string) int

// ValidateCacheBoundary verifies that a generated prompt possesses an invariant
// prefix of at least MinStaticPrefixTokens (or MinStaticPrefixBytes).
func ValidateCacheBoundary(parts PromptParts) error
```

### 3.2 Standard Invariant Prefix Structure
```markdown
# [Role Mandate Title: e.g. Independent Pre-GATE 1 Plan Review]
## SDD Anthem & Core Principles
## Canonical Role Skill Instructions: <Role Name>
## Evaluation Rubric & Severity Standards (3-tier severity)
## Model Routing Guidance & Subagent Invocation Template
## Approved Architectural Decisions (TECH_STACK.md §6)
## Active Feedback Rules (SKILLS.md §8)
<!-- sdd:cache-boundary:end-prefix -->
```

### 3.3 Dynamic Suffix Structure
```markdown
## Dynamic Task Context & Metadata
- Task ID, Typology, Task Directory, Review Lens, Remediation Loop
- Planned Files Boundary / Changed Files List
## Staged Stateless Review Bundle (.sdd/review/<TASK_ID>/...)
- Bundle Dir, Spec, Ledger, Findings to Verify, Filtered Diff Telemetry
## Step-by-Step Task Instructions
- Specific file read order, audit checks, and recording directives
```

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Author prompt cache boundary utility with token/byte estimations and partition types | `internal/cli/prompt_cache.go`, `internal/cli/prompt_cache_test.go` | `go test -v ./internal/cli -run TestPromptCache` |
| 2 | Reorganize `buildPlanReviewPrompt`, `buildImplementPrompt`, and `buildReviewPromptWithLens` to enforce invariant static prefixes ($\ge 1,024$ tokens) and dynamic suffixes; adapt existing assertions in `cli_test.go` | `internal/cli/prompt.go`, `internal/cli/cli_test.go` | `go test -v ./internal/cli -run "TestPromptPlanReview|TestPromptImplement|TestPromptReview"` |
| 3 | Reorganize `buildDiscoveryPrompt` and `buildRepairPrompt` to adhere to the invariant prefix architecture; adapt existing assertions in `prompt_repair_test.go` | `internal/cli/prompt_discovery.go`, `internal/cli/prompt_repair.go`, `internal/cli/prompt_repair_test.go` | `go test -v ./internal/cli -run "TestPromptDiscovery|TestPromptRepair"` |
| 4 | Restructure base agent templates (`prompt.md.tmpl` and `prompt_compact.md.tmpl`) to place core contract sections ahead of dynamic state | `internal/templates/data/base/prompt.md.tmpl`, `internal/templates/data/base/prompt_compact.md.tmpl` | `go test -v ./internal/templates -run TestPromptTemplateOrder` |
| 5 | Author multi-task and multi-loop byte-invariance tests proving cross-task cache hit stability | `internal/cli/prompt_test.go` | `go test -v ./internal/cli -run TestPromptPrefixCaching_ByteInvariance` |
| 6 | Bump `templates.Version` to `2.10.0` across codebase, contract headers, manifests, and tests; verify temporary kit regeneration | `internal/templates/templates.go`, `.sdd/manifest.json`, `CLAUDE.md`, `GEMINI.md`, `AGENTS.md`, `internal/templates/templates_test.go` | `go test -v ./internal/templates -run TestVersion && ./bin/sdd check --strict` |
| 7 | Full test suite execution, race detector, determinism, and validation evidence recording | All planned files | `make test && make lint` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/cli/prompt_cache.go` | create | Prompt cache boundary structures, token estimators, and partition validators |
| `internal/cli/prompt_cache_test.go` | create | Unit tests for token estimation, cache boundary validation, and partition joins |
| `internal/cli/prompt.go` | modify | Restructure `buildPlanReviewPrompt`, `buildImplementPrompt`, `buildReviewPromptWithLens` |
| `internal/cli/cli_test.go` | modify | Adapt existing prompt header string assertions to invariant title format |
| `internal/cli/prompt_discovery.go` | modify | Restructure `buildDiscoveryPrompt` for prefix caching |
| `internal/cli/prompt_repair.go` | modify | Restructure `buildRepairPrompt` for prefix caching |
| `internal/cli/prompt_repair_test.go` | modify | Adapt existing prompt repair header assertion to invariant title format |
| `internal/cli/prompt_test.go` | modify | Multi-task and multi-loop byte invariance and token threshold tests |
| `internal/templates/data/base/prompt.md.tmpl` | modify | Invariant contract prefix ordering ahead of dynamic project state |
| `internal/templates/data/base/prompt_compact.md.tmpl` | modify | Invariant compact contract prefix ordering ahead of dynamic state |
| `internal/templates/templates.go` | modify | Bump `templates.Version` to `2.10.0` per Hard Rule 16 |
| `internal/templates/templates_test.go` | modify | Update template version test assertion to `2.10.0` |
| `CLAUDE.md` | modify | Bump contract header to `templates=2.10.0` |
| `GEMINI.md` | modify | Bump contract header to `templates=2.10.0` |
| `AGENTS.md` | modify | Bump contract header to `templates=2.10.0` |
| `.sdd/manifest.json` | modify | Bump `template_version` to `2.10.0` |
| `ROADMAP.md` | modify | Update task status, milestone progress, and change log |
| `TECH_STACK.md` | modify | Update verified facts log, architecture notes, and Claude Haiku 2,048-token floor nuance |
| `SKILLS.md` | modify | Record prompt prefix caching patterns, invariant prefix conventions, and Haiku 2,048-token floor |
| `README.md` | modify | Document prompt prefix caching token optimizations |
| `tasks/features/T-062-provider-aligned-prefix-caching/requirements.md` | modify | Complete specification requirements |
| `tasks/features/T-062-provider-aligned-prefix-caching/plan.md` | modify | Implementation plan and approval record |
| `tasks/features/T-062-provider-aligned-prefix-caching/validation.md` | modify | Validation commands and real output evidence |
| `tasks/features/T-062-provider-aligned-prefix-caching/review.md` | modify | Review Triad audit sign-off and findings ledger |

## 6. Dependencies
None. 100% Go standard library only per Decision `D-007`.

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Mark T-062 complete upon Gate 2; update Milestone 15 progress (3/5 complete, 60%) |
| `TECH_STACK.md` | Add verified facts for prompt prefix caching, byte invariance, and token thresholds |
| `SKILLS.md` | Document prompt prefix caching patterns and invariant prefix conventions |
| `README.md` | Document prompt caching optimization and KV cache alignment |

## 8. Risks, mitigations, rollback
- **Risk:** Invariant prefix falls below 1,024 tokens for bare-bones projects with few feedback rules or small role skills.
  - **Mitigation:** Invariant prefix includes complete canonical role skill instructions, standard review rubrics, 3-tier severity definitions, and model routing guidance, which together consistently exceed 4,000 bytes (~1,200 tokens). A programmatic test ensures `EstimateTokens >= 1024`.
- **Risk:** Dynamic metadata inadvertently leaks into the invariant prefix (e.g. task ID in header).
  - **Mitigation:** Explicit byte-equality assertions between `prompt(task A)` and `prompt(task B)` static prefixes catch any leakage at test time.
- **Rollback:** `git checkout main` or discard task worktree branch cleanly via `./bin/sdd task clean T-062`.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- `F-001`: Resolve all review findings (🔴 Blockers, 🟡 Warnings, 🟢 Minors) before Gate 2 close; zero unresolved findings in `review.md`.
- `F-002`: Path hygiene — zero personal home directory paths (`/Users/...`, `/home/...`) in tracked files or tests.
- `F-003`: Default task token telemetry — record token metrics via `sdd record-tokens T-062` and document in `validation.md`.

## 10. Approval record
Approved by: Vikash (human)
Date: 2026-11-25
Phrase received: `APPROVED T-062`
