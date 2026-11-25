---
status: draft
---
# Requirements — T-060 Subagent model routing, tier enforcement, and calibrated reasoning dispatch

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Draft | Approved (pending GATE 1)
Roadmap ref: `ROADMAP.md` T-060
Decision ref: `TECH_STACK.md` D-050 (Proposed)

## 1. Goal

Mandate and enforce subagent model routing, capability tier pinning (`frontier`, `advanced`, `standard`, `fast`), and calibrated reasoning effort dispatch across Task Coordinator skills, agent contracts, prompt generators, and mechanical checks. Prevent token exhaustion by ensuring routine subagent tasks (such as Spec Auditor, Pattern Auditor, QA, and DevOps) are systematically routed to `standard` / `fast` model tiers (`flash` / `haiku` / `gpt-4o-mini`) rather than inheriting expensive frontier reasoning models.

## 2. Context and references

- Mission / principles touched: Determinism, Fail-safe behavior, Economic Discipline & Token Efficiency.
- Decisions this relies on: `D-049` (Heterogeneous role model routing in `.sdd/roles.json`), `D-050` (Token consumption mitigation and tier enforcement architecture).
- Depends on tasks: `T-059` (Heterogeneous role model routing, tier specifications, and provider-agnostic dispatch - Done).
- Context artifacts: `.sdd/roles.json`, `skills/task-coordinator/SKILL.md`, `CLAUDE.md`, `GEMINI.md`, `AGENTS.md`.

## 3. Scope

**In scope**
- Defining canonical tier-to-runtime dispatch mappings across major agent environments (Antigravity/Gemini, Claude Code/Anthropic, OpenAI).
- Updating `skills/task-coordinator/SKILL.md`, `skills/code-review/SKILL.md`, and embedded coordinator templates to mandate explicit `Model` parameter dispatch during subagent invocation (`invoke_subagent`).
- Updating agent contracts (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`) and embedded contract template (`contract.md.tmpl`) to codify the Tier-Calibrated Subagent Dispatch Rule.
- Enhancing `sdd prompt` handover commands (`review`, `plan-review`, `implement`) to generate executable dispatch snippets with the exact `Model` parameter for standard, advanced, and frontier roles.
- Implementing a mechanical check rule (`coordinator-tier-dispatch`) in `internal/check/` verifying that coordinator skills and contracts declare model tier dispatch for all Review Triad roles.
- Bumping `templates.Version` from `2.7.0` to `2.8.0` in `internal/templates/templates.go`, `.sdd/manifest.json`, and associated test fixtures per Hard Rule 16.

**Out of scope**
- Modifying external IDE harnesses or writing platform-specific network clients.
- Modifying Review Triad diff extraction slices (deferred to `T-061`).
- Prompt prefix caching reorganization (deferred to `T-062`).
- Modifying discovery prompt templates (discovery ensemble model routing was established in `T-053`/`T-059`).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL define and document runtime model mappings for capability tiers: `frontier` (`inherit` / `pro` / `opus`), `advanced` (`inherit` / `pro` / `sonnet`), `standard` (`flash` / `haiku` / `gpt-4o`), and `fast` (`flash_lite` / `haiku` / `gpt-4o-mini`). | MUST | inspection |
| R-002 | The Task Coordinator skill (`skills/task-coordinator/SKILL.md`) and embedded templates SHALL mandate passing the explicit `Model` parameter during subagent invocation: `Model="flash"` for `spec-auditor`, `pattern-auditor`, `qa`, `devops`, and `Model="inherit"` / `pro` for `appsec-specialist` and `plan-reviewer`. | MUST | inspection |
| R-003 | The agent contracts (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`) and `contract.md.tmpl` SHALL codify the Tier-Calibrated Subagent Dispatch Rule in Section 2 (Workflow) and Section 4 (Hard Rules), forbidding uncalibrated frontier inheritance for routine auditor roles. | MUST | test / inspection |
| R-004 | The `sdd prompt review`, `sdd prompt plan-review`, and `sdd prompt implement` CLI commands SHALL output concrete executable invocation snippets containing the exact `Model` parameter matching the role's declared capability tier. | MUST | test |
| R-005 | `internal/check` SHALL implement mechanical rule `coordinator-tier-dispatch` verifying that coordinator skills and contracts specify explicit model tier assignments for all Review Triad roles. | MUST | test |
| R-006 | The system SHALL bump `templates.Version` to `2.8.0` in `internal/templates/templates.go` and `.sdd/manifest.json`, verifying that kit generation and `sdd upgrade --dry-run` pass strict check. | MUST | test |
| R-007 | The implementation SHALL maintain 100% Go standard library purity (`D-007`), pass `make verify`, run race-clean, and demonstrate determinism (`-count=2`). | MUST | test |

## 5. Non-functional requirements

- **Determinism:** Tier-to-model resolution is pure and deterministic; identical inputs produce byte-identical dispatch snippets.
- **Fail-safe behavior:** If an unrecognized tier or role is supplied, fallback defaults to canonical defaults or fail-safe frontier inheritance with explicit logging.
- **Economic Discipline:** Systematically cuts Review Triad subagent token costs by $\ge 60\%$ by offloading structured matrix checks (Spec Auditor) and style checks (Pattern Auditor) to high-speed/low-cost standard models.
- **Secrets & Privacy:** Zero personal paths (`F-002`), zero telemetry leakage.

## 6. Constraints and assumptions

- Subagent dispatch in IDEs (such as Antigravity) uses the `invoke_subagent` tool which supports `Model: "inherit" | "flash" | "flash_lite" | "pro"`.
- When `Model` is omitted, the IDE defaults to `inherit`, which binds to the active frontier model; therefore, explicit `Model` specification is mandatory for all non-frontier roles.

## 7. Open questions

(None — all architectural decisions settled in `D-049` and proposed in `D-050`).

## 8. Acceptance criteria

- Given a Task Coordinator dispatching the Review Triad, when `spec-auditor` and `pattern-auditor` are invoked, then `Model: "flash"` is passed explicitly.
- Given a human running `sdd prompt review T-xxx --lens spec`, when prompt output is generated, then an executable `invoke_subagent` block with `Model: "flash"` is presented.
- Given `./bin/sdd check --strict`, when coordinator skill or contracts omit model tier dispatch rules, then the checker flags finding `coordinator-tier-dispatch`.
