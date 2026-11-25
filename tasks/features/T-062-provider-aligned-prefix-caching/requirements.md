---
status: draft
---
# Requirements — T-062 Provider-aligned prompt prefix caching and dynamic suffix pinning

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-25)
Roadmap ref: `ROADMAP.md` T-062

## 1. Goal
Reorganize all SDD prompt generators and templates to enforce an invariant, byte-stable static prefix of at least 1,024 tokens with all dynamic task metadata pinned strictly to suffixes, maximizing prompt KV cache hit rates ($\ge 50-90\%$) across Anthropic, OpenAI, and Gemini LLM providers.

## 2. Context and references
- Mission / principles touched: MISSION.md §5 Principle 1 (Deterministic), Principle 4 (Small and shippable), Business intent (token efficiency and latency reduction).
- Decisions this relies on:
  - `D-007`: Go standard library only (distribution as single static binary, no third-party deps).
  - `D-044`: Stateless review context isolation and tabula-rasa bundles.
  - `D-049`: Ephemeral tabula-rasa reviewer lifecycle and heterogeneous role model routing.
  - `D-050`: Token consumption mitigation, context hygiene, and provider-aligned caching architecture (Milestone 15).
- Depends on tasks:
  - `T-060` (Subagent model routing, tier enforcement, and calibrated reasoning dispatch — Done).
  - `T-061` (Lens-scoped review bundles and diff filtering — Done).

## 3. Scope
**In scope**
- Reorganizing prompt builders in `internal/cli/`:
  - `buildPlanReviewPrompt` (`sdd prompt plan-review <T-xxx>`)
  - `buildImplementPrompt` (`sdd prompt implement <T-xxx> [--role <role>]`)
  - `buildReviewPromptWithLens` (`sdd prompt review <T-xxx> [--lens <lens>] [--loop <loop>]`)
  - `buildDiscoveryPrompt` (`sdd prompt discovery [T-xxx] [--ensemble]`)
  - `buildRepairPrompt` (`sdd prompt repair <T-xxx> [--max-loops <N>]`)
- Reorganizing base handover templates in `internal/templates/data/base/`:
  - `prompt.md.tmpl`
  - `prompt_compact.md.tmpl`
- Defining a standardized invariant prefix structure:
  - Global Anthem & Operating Principles
  - Canonical Role Mandate & Skill Instructions
  - Review / Implementation Rubrics & Severity Definitions (🔴 Blocker, 🟡 Warning, 🟢 Minor)
  - Model Routing Guidance & Dispatch Snippets
  - Approved Decisions (`TECH_STACK.md`) & Active Feedback Rules (`SKILLS.md §8`)
  - General Review & Traceability Standards
- Pinning all dynamic fields to prompt suffixes:
  - Task ID, Typology, Directory, Worktree path, Lens name, Loop number
  - Planned Files Boundary, Changed Files, Diff Patch path
  - Staged Bundle paths (`.sdd/review/<TASK_ID>/...`)
  - Dynamic Step-by-Step Task Instructions & Checklists
- Programmatic token/byte verification utility and automated tests ensuring $\ge 4,096$ invariant bytes ($\ge 1,024$ tokens) and byte-identical prefix stability across different tasks.
- Bumping `templates.Version` to `2.10.0` across all contract headers, templates, manifests, and tests.

**Out of scope**
- Modifying provider network APIs or sending HTTP requests to LLM endpoints (SDD is network-free and static CLI per Principle 4 and Mission §4).
- Altering the review triad diff filtering engine (delivered in T-061).
- Living document progressive micro-indexing (assigned to T-063).
- Native IDE transcript auto-discovery (assigned to T-064).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL guarantee an invariant static prefix of at least 1,024 tokens ($\ge 4,096$ bytes, with 4,200 bytes safety target) across standard `sdd prompt` templates (`plan-review`, `implement`, `review`, `discovery`, `repair`, and base `prompt.md.tmpl`). Note: `prompt_compact.md.tmpl` is exempt from the 1,024-token floor due to its $< 500$-token design budget (T-049), but strictly adheres to static-first ordering per R-005. | MUST | Unit test in `internal/cli/prompt_test.go` |
| R-002 | The system SHALL pin all dynamic metadata (Task ID, typology, directory, loop iteration, changed/planned files, staged bundle paths, and task-specific instructions) strictly to the prompt suffix following the invariant prefix. | MUST | Test & golden comparison in `internal/cli/prompt_test.go` |
| R-003 | The system SHALL ensure deterministic byte invariance of the static prefix across tasks: running `sdd prompt` for task A and task B with the same role/lens SHALL generate 100% byte-identical static prefixes. | MUST | Determinism test in `internal/cli/prompt_test.go` |
| R-004 | The system SHALL maintain byte-identical static prefixes across remediation loops (Loop 1, Loop 2, Loop 3) in `sdd prompt review`, pinning the loop counter, defect checklist, and diff references exclusively to the suffix. | MUST | Unit test in `internal/cli/prompt_test.go` |
| R-005 | The base agent handover templates (`base/prompt.md.tmpl` and `base/prompt_compact.md.tmpl`) SHALL place static operating contracts, anthems, reading orders, hard rules, and skills catalog ahead of dynamic project state, open tasks, focus tasks, and staged context. | MUST | Template verification in `internal/templates/templates_test.go` |
| R-006 | The system SHALL bump `templates.Version` to `2.10.0` across codebase, contract headers (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`), starter templates, manifests, and test assertions per Hard Rule 16. | MUST | Verification via temporary kit generation and `sdd check` |
| R-007 | The implementation SHALL use 100% Go standard library (`D-007`), execute race-free (`-race`), and yield byte-identical outputs across repeated runs (`-count=2`). | MUST | Test execution in `validation.md` |

## 5. Non-functional requirements
- **Determinism:** Zero randomness, fixed sort orders for decisions and feedback rules, invariant string assembly.
- **Fail-safe behaviour:** Graceful fallbacks when skills or config files are missing without breaking prefix boundaries.
- **Performance budget:** Sub-millisecond prompt generation ($< 5\text{ms}$); zero network I/O.
- **Security and privacy:** No exposure of host paths in invariant prefixes; relative or synthetic paths in templates.

## 6. Constraints and assumptions
- Token counts are estimated deterministically using character/byte length heuristics ($\ge 4,096$ UTF-8 bytes $\approx \ge 1,024$ tokens across Claude/OpenAI/Gemini tokenizers).
- LLM provider caching requires exact prefix byte-matching: even a single differing character in the prefix causes a cache miss.

## 7. Open questions
(None — all architectural requirements settled in D-050).

## 8. Acceptance criteria
- **Given** two distinct tasks (e.g. `T-001` and `T-002`), **when** `sdd prompt plan-review` or `sdd prompt review --lens <lens>` is invoked for each, **then** the first $\ge 4,096$ bytes ($\ge 1,024$ tokens) of the prompt outputs are byte-for-byte identical.
- **Given** remediation loop 1 and loop 2 for the same task, **when** `sdd prompt review --loop 1` and `sdd prompt review --loop 2` are generated, **then** their static prefixes are byte-for-byte identical.
- **Given** a generated kit from the updated templates, **when** `sdd check` is run in a clean temporary directory, **then** it passes with 0 errors and 0 warnings.
