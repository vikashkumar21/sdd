---
status: active
task_id: T-049
last_reviewed: 2026-11-18
---
# Requirements — T-049 Prompt cache boundary optimization, compact handover, and brownfield README proposal

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-18)
Roadmap ref: `ROADMAP.md` T-049

## 1. Goal
Maximize LLM Key-Value (KV) prompt caching efficiency and context economy by restructuring handover prompts with static-first prefix pinning, introducing a high-density micro-context `sdd prompt --compact` mode, tracking token consumption telemetry within task state store (`.sdd/state/<TASK_ID>/`), and generating `README.md.sdd-new` proposal templates during brownfield `sdd init` without modifying existing `README.md` files.

## 2. Context and references
- Mission / principles touched: Principles 1 (Deterministic), 3 (Human-in-the-loop for the irreversible), 4 (Small and shippable), 5 (Never overwrite human work).
- Decisions this relies on: `D-007` (Standard library only), `D-010` (Brownfield never overwrite), `D-017` (Managed blocks and `.sdd-new` proposals), `D-024` (Agent skills standard), `D-029` (Two-tier git governance), `D-035` (Task state IPC), `D-041` (Multi-agent coordinator contract), `D-042` (Prompt cache prefix pinning, compact handover, task token tracking, and brownfield README proposal).
- Depends on tasks: `T-039` (Task state persistence), `T-048` (Multi-agent coordinator contract).

## 3. Scope
**In scope**
- **Prompt Cache Boundary Optimization (Static-First Prefix Pinning):**
  - Restructure `base/prompt.md.tmpl` and prompt generators (`internal/cli/prompt.go`) to position all immutable, static tokens at the top of the prompt:
    1. Operating anthem and Task Coordinator role mandate.
    2. Workflow step state-machine (Steps 0–2: Phases A–D, gates).
    3. Hard rules summary.
    4. Gate reporting format.
    5. Available skills progressive disclosure catalog.
  - Position all volatile, dynamic tokens strictly at the bottom suffix:
    1. Roadmap snapshot and task status.
    2. Dynamic generation date and timestamps.
    3. Human inputs / gate replies / decisions.
    4. Staged external context files (`.sdd/context/`).
  - This ensures that turns and day-to-day session prompts preserve maximum common byte prefixes, achieving optimal KV cache hits across Gemini, Claude, and OpenAI.
- **Compact Handover Mode (`sdd prompt --compact`):**
  - Introduce `--compact` flag to `sdd prompt <agent>`.
  - Emits a high-density, micro-context prompt (~400 tokens vs ~2,000 tokens) focused purely on active task state, direct living doc pointers, and strict gate requirements—ideal for tight context budgets or rapid session resume.
- **Token Telemetry in Task State (`.sdd/state/<TASK_ID>/`):**
  - Extend `internal/state` data models to record token telemetry:
    - `tokens: { input_tokens, cache_read_tokens, output_tokens, total_tokens }`.
  - Add CLI command `sdd task state record-tokens <T-xxx> [--input <N>] [--cache <N>] [--output <N>] [--json '<json>']`.
  - Display token consumption in `sdd task state show <T-xxx>` and `sdd status --json`.
- **Brownfield `README.md.sdd-new` Proposal Generation (`sdd init`):**
  - In brownfield projects where `README.md` already exists, `sdd init` preserves the existing `README.md` untouched, but writes `README.md.sdd-new` containing SDD's starter template.
  - Allows human maintainers to easily review and copy SDD badges, workflow descriptions, and CLI instructions into their project readme without destructive clobbering.
  - Recorded in `.sdd/manifest.json` as a proposal.

**Out of scope**
- Modifying prompt caching behavior of external LLM APIs (prompt caching is server-side prefix matching).
- Destructive replacement of existing `README.md` (violates Principle 5 and Criterion S1).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL restructure `internal/templates/data/base/prompt.md.tmpl` so that static invariant sections (Anthem, Coordinator Role Orchestration, Hard Rules, State Machine Steps, Gate Reporting Format, Skills Catalog) precede dynamic suffix sections (Roadmap snapshot, dynamic date, human inputs, task context). | MUST | test / inspection |
| R-002 | The system SHALL implement `sdd prompt <agent> --compact` to generate a high-density micro-context handover prompt omitting verbose boilerplate while preserving essential task state, Anthem, and gate invariants. | MUST | test |
| R-003 | The system SHALL extend `internal/state` (`state.json` and `events.jsonl`) to store and update token consumption metrics (`input_tokens`, `cache_read_tokens`, `output_tokens`, `total_tokens`). | MUST | test |
| R-004 | The system SHALL implement `sdd task state record-tokens <T-xxx>` supporting discrete flags (`--input`, `--cache`, `--output`) and `--json` to update task token telemetry and append lifecycle events. | MUST | test |
| R-005 | The system SHALL display token telemetry in `sdd task state show <T-xxx>` and `sdd status --json`. | MUST | test |
| R-006 | The system SHALL update `sdd init` in brownfield mode so that when an existing `README.md` is detected, it is preserved untouched while a `README.md.sdd-new` proposal file is written alongside it. | MUST | test |
| R-007 | The system SHALL update `.sdd/manifest.json` and CLI status output during `sdd init` to report `README.md.sdd-new` proposal creation. | MUST | test |
| R-008 | The system SHALL update documentation in `README.md` with `sdd prompt --compact`, `sdd task state record-tokens`, and the brownfield `README.md.sdd-new` proposal behavior. | MUST | inspection |

## 5. Non-functional requirements
- **Determinism:** Given identical inputs and mock clocks, prompt generation, token tracking, and proposal generation must produce byte-identical outputs.
- **Fail-safe:** Never overwrite or mutate existing `README.md` in brownfield projects under any circumstance.
- **Standard Library Purity (`D-007`):** Pure Go standard library; zero third-party dependencies.
- **Efficiency:** `--compact` prompt must reduce handover token footprint by at least 60% compared to standard prompts.

## 6. Constraints and assumptions
- Token values are non-negative 64-bit integers (`int64`).
- Proposal filename adheres to the established SDD proposal convention (`<file>.sdd-new`).

## 7. Open questions
(None. Settled in Decision D-042.)

## 8. Acceptance criteria
- **AC-1 (Static-First Prefix Pinning):** Inspecting rendered `sdd prompt <agent>` confirms that static sections appear before dynamic date, task context, and human inputs.
- **AC-2 (Compact Prompt):** Running `sdd prompt <agent> --compact --task <T-xxx>` outputs a valid, concise prompt with under 500 tokens containing active task metadata and gate instructions.
- **AC-3 (Task Token State):** Running `sdd task state record-tokens T-049 --input 1000 --cache 5000 --output 200` saves the tokens into `state.json` and records an event in `events.jsonl`.
- **AC-4 (Brownfield README Proposal):** Running `sdd init --yes` in a brownfield project with an existing `README.md` preserves `README.md` untouched and creates `README.md.sdd-new`.
- **AC-5 (Strict Kit Consistency):** All unit tests pass, `make verify`, `make lint`, and `./bin/sdd check --strict` pass with 0 errors and 0 warnings.
