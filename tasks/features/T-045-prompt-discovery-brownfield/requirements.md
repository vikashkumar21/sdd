---
status: active
task_id: T-045
last_reviewed: 2026-11-15
---
# Requirements — T-045 Turnkey sdd prompt discovery and brownfield synthesis protocol

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-045

## 1. Goal
Provide a turnkey CLI command `sdd prompt discovery` that generates a structured codebase exploration and brownfield forensic extraction handover prompt enriched with the canonical `codebase-analyst` role skill, a 4-pillar progressive extraction checklist (AST/topology, UI/design systems, data/state models, technical debt/pitfalls), and a brownfield living document synthesis protocol (`TECH_STACK.md`, `ROADMAP.md`, `SKILLS.md`), supporting both human-readable text and machine-readable `--json` output.

## 2. Context and references
- Mission / principles touched:
  - Principle 1 (Deterministic): Reproducible forensic exploration checklists ensure consistent codebase mapping across diverse agents and projects.
  - Principle 2 (Fail safe): Read-only exploration boundaries prevent accidental modifications to application code during initial discovery.
  - Principle 4 (Small and shippable): Turnkey command integrates smoothly into existing `sdd prompt` suite without external dependencies.
- Decisions this relies on:
  - D-002: Spec-driven workflow with human gates.
  - D-007: Standard library only.
  - D-024: Linux Foundation / Anthropic `agentskills.io` standard adoption.
  - D-030: Unified `tasks/` directory hierarchy (`tasks/features/`, `tasks/bugfixes/`, `tasks/hotfixes/`, `tasks/spikes/`).
  - D-036: Role skill taxonomy, phase mapping (`.sdd/roles.json`), and two-tier storage.
  - D-039: Role-based agent skill `codebase-analyst` and phase-discovery mapping.
- Depends on tasks: T-043, T-044 (Done).

## 3. Scope
**In scope**
- CLI dispatch for `sdd prompt discovery [T-xxx] [--dir <project>] [--json]`:
  - Wire `discovery` subcommand into `internal/cli/commands.go:runPrompt`.
  - Implement `runPromptDiscovery` and `buildDiscoveryPrompt` in `internal/cli/prompt_discovery.go`.
- Role skill enrichment:
  - Incorporate canonical `codebase-analyst` role standard resolved via two-tier storage (`role.ResolveSkill`).
  - Respect project-level override `skills/codebase-analyst/SKILL.md` when present.
- 4-Pillar progressive extraction checklist:
  - Pillar 1: AST & Module Map (Directory topology, entry points, package dependencies, frameworks, build systems).
  - Pillar 2: Design System & UI Patterns (Font families, CSS frameworks, color themes, component hierarchy, responsive layout).
  - Pillar 3: Data & State Modeling (Database schemas, ORM models, state containers, caching layers, event queues).
  - Pillar 4: Anti-Patterns & Technical Debt (Fragile dependencies, unhandled error patterns, undocumented quirks, security risks).
- Brownfield living document synthesis protocol:
  - Architecture, frameworks, decisions -> `TECH_STACK.md` (Decision register `D-xxx` and verified-facts log).
  - Discovered conventions, coding patterns, pitfalls -> `SKILLS.md` (§1 Language standards, §7 Pitfalls log).
  - Technical debt, backlog items, unmapped components -> `ROADMAP.md` (Section 7 Parking lot, or planned tasks).
  - External service dependencies, API contracts -> `TECH_STACK.md` §8 (Dependency register).
- Context and project metadata digest:
  - Include detected project languages (`detect.Detect`), manifests, and staged context digest (`context.ReadManifest`).
- Structured machine-readable JSON output (`--json`):
  - Model `DiscoveryPromptContext` containing task metadata, role skill, detected languages, living docs targets, extraction checklist, and raw prompt text.
- Documentation and CLI ergonomics:
  - Update `sdd prompt` usage documentation in `internal/cli/cli.go` and `internal/cli/commands.go`.
  - Update `README.md` command documentation.
- Automated tests:
  - Comprehensive unit tests in `internal/cli/prompt_discovery_test.go`.

**Out of scope**
- Autonomous code modifications or project alterations (discovery is strictly read-only forensic mapping).
- Calling remote AI models or external network APIs.
- Modifying `detect.Detect` heuristics.

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL provide `sdd prompt discovery [T-xxx] [--dir <dir>] [--json]` in `sdd prompt` command dispatch. | MUST | test / inspection |
| R-002 | The system SHALL resolve and inject the canonical `codebase-analyst` role skill into the discovery prompt via two-tier storage (`role.ResolveSkill`). | MUST | test |
| R-003 | The system SHALL embed a 4-pillar progressive extraction checklist (AST & Topology, UI & Design Systems, Data & State Modeling, Anti-Patterns & Technical Debt) in the prompt output. | MUST | test |
| R-004 | The system SHALL embed a brownfield living document synthesis protocol directing discoveries into `TECH_STACK.md`, `SKILLS.md`, and `ROADMAP.md`. | MUST | test |
| R-005 | The system SHALL support `--json` output rendering a structured `DiscoveryPromptContext` matching the JSON schema. | MUST | test |
| R-006 | The system SHALL support resolving an optional focus task `[T-xxx]` or fallback to root project discovery when omitted, validating task ID format and directory existence. | MUST | test |
| R-007 | The system SHALL update `sdd prompt` usage documentation in `internal/cli/cli.go`, `internal/cli/commands.go`, and `README.md` to include `sdd prompt discovery`. | MUST | inspection |

## 5. Non-functional requirements
- **Standard Library Purity:** Pure Go standard library per Decision D-007.
- **Determinism:** Output is byte-identical for identical project files, flags, and task state.
- **Fail Safe:** Invalid task ID format (`T-abc`) or non-existent task directory returns exit code 1 with clear diagnostic stderr.
- **Performance:** Prompt assembly runs in under 100ms for standard repositories.

## 6. Constraints and assumptions
- Conforms to Linux Foundation / Anthropic `agentskills.io` standard.
- The `codebase-analyst` role skill is already canonical in embedded templates and project skills (T-043/T-044).
- The discovery prompt is read-only and explicitly instructs the agent that it SHALL NOT modify application code.

## 7. Open questions
(None. Section is empty for GATE 1.)

## 8. Acceptance criteria
- **AC-1 (Default Discovery Prompt):** Running `sdd prompt discovery` in a project prints the turnkey discovery prompt including the `codebase-analyst` role skill, detected languages, 4-pillar checklist, and living doc synthesis instructions.
- **AC-2 (Task-Focused Discovery Prompt):** Running `sdd prompt discovery T-001` links the prompt to `T-001`, resolving `tasks/features/T-001-*/requirements.md` and `plan.md`.
- **AC-3 (Machine-Readable JSON Bundle):** Running `sdd prompt discovery --json` outputs valid JSON parseable into `DiscoveryPromptContext` with non-empty `roleSkill`, `livingDocs`, `checklist`, and `prompt`.
- **AC-4 (Two-Tier Override Support):** Providing a project-level override in `skills/codebase-analyst/SKILL.md` causes `sdd prompt discovery` to use the override content.
- **AC-5 (Invalid Input Handling):** Running `sdd prompt discovery INVALID` prints an error to stderr and exits with code 1.
- **AC-6 (Verification Suite):** All unit tests in `internal/cli/prompt_discovery_test.go` pass; `make check` and `./bin/sdd check --strict` pass with 0 errors and 0 warnings.
