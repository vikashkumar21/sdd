---
status: active
---
# Requirements — T-059 Heterogeneous role model routing, tier specifications, and provider-agnostic dispatch

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved 2026-11-24
Roadmap ref: `ROADMAP.md` T-059
Decision ref: `TECH_STACK.md` D-049 (Approved 2026-11-24)

## 1. Goal

Eliminate homogeneous frontier model over-consumption and optimize multi-agent token budgets by extending `.sdd/roles.json` with heterogeneous model routing metadata (`model_tier`, `reasoning_effort`, `recommended_family`), enriching `sdd prompt` handover generators (both human-readable Markdown and machine-readable JSON) with model routing directives, validating tier specifications in `sdd check`, supporting non-destructive migration in `sdd upgrade`, and bumping templates to version `2.7.0`.

## 2. Context and references

- Mission / principles: Principle 1 (Deterministic by construction), Principle 2 (Fail safe), Principle 4 (Small and shippable).
- Decisions: D-003 / D-007 (Go standard library only), D-036 (Role skill taxonomy and .sdd/roles.json), D-041 (Multi-agent coordinator contract), D-042 (Prompt cache optimization and token tracking), D-049 (Cognitive bias immunity: unanchored auditing, ephemeral reviewer spawning, heterogeneous model routing).
- Depends on: T-058 (Ephemeral tabula-rasa reviewer lifecycle and zero-negotiation loop isolation).
- Active Feedback Rules: F-001 (Resolve all findings before Gate 2), F-002 (Zero personal absolute paths), F-003 (Token telemetry).

## 3. Scope

### In scope
1. **Model Routing Schema in `.sdd/roles.json` (`R-001`):**
   - Extend `RoleInfo` with `model_tier`, `reasoning_effort`, and `recommended_family`.
   - Seed all 11 canonical SDD roles (`coordinator`, `architect`, `codebase-analyst`, `discovery-challenger`, `plan-reviewer`, `implementer`, `qa`, `devops`, `spec-auditor`, `appsec-specialist`, `pattern-auditor`) with calibrated model routing metadata.
2. **Mechanical Model Routing Validation in `sdd check` (`R-002`):**
   - Validate `model_tier` enum values (`frontier`, `advanced`, `standard`, `fast`).
   - Validate `reasoning_effort` enum values (`high`, `medium`, `low`, `none`).
   - Reject empty or whitespace-only `recommended_family` if specified.
3. **Handover Prompt Model Routing Integration (`R-003`):**
   - Enrich `sdd prompt review`, `sdd prompt plan-review`, `sdd prompt implement`, `sdd prompt discovery`, and `sdd prompt repair` output with a dedicated `## Model Routing Guidance` section.
   - Include `model_routing` fields in prompt JSON contexts for programmatic orchestration by Agent SDKs.
4. **Non-Destructive Upgrade & Migration (`R-004`):**
   - Extend `sdd upgrade` to non-destructively backfill missing model routing metadata on canonical roles while preserving custom user modifications.
5. **Starter Template Alignment & Version Bump (`R-005`):**
   - Update starter templates and bump `templates.Version` to `2.7.0` in `internal/templates/templates.go`, contracts, and manifests per Hard Rule 16.
6. **Coordinator Skill Documentation (`R-006`):**
   - Document Heterogeneous Model Routing principles in `skills/task-coordinator/SKILL.md` and embedded role templates.
7. **Standard Library Purity (`R-007`):**
   - 100% Go standard library implementation (`D-007`); zero third-party dependencies.

### Out of scope
- Direct API provider network calls or live model invocation (SDD generates prompts and manifests; agents and humans execute).
- Vendor-locked provider configuration files.

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL extend `.sdd/roles.json` and `internal/role/role.go` with `model_tier`, `reasoning_effort`, and `recommended_family` metadata for all canonical roles. | MUST | test |
| R-002 | The system SHALL enforce mechanical validation in `sdd check` verifying valid `model_tier` (`frontier`, `advanced`, `standard`, `fast`) and `reasoning_effort` (`high`, `medium`, `low`, `none`) values. | MUST | test |
| R-003 | The system SHALL enrich all `sdd prompt` commands (`review`, `plan-review`, `implement`, `discovery`, `repair`) with `## Model Routing Guidance` in Markdown output and structured `model_routing` in JSON output. | MUST | test |
| R-004 | The system SHALL non-destructively backfill missing model routing metadata in existing `.sdd/roles.json` files during `sdd upgrade` without clobbering custom role definitions. | MUST | test |
| R-005 | The system SHALL update starter templates, bump `templates.Version` to `2.7.0`, and verify fresh kit regeneration and contract parity per Hard Rule 16. | MUST | test |
| R-006 | The system SHALL document Heterogeneous Model Routing in `skills/task-coordinator/SKILL.md`, embedded templates, and living documents. | MUST | inspection |
| R-007 | The system SHALL implement all model routing, schema validation, prompt formatting, and upgrade logic using only the Go standard library with zero external dependencies (`D-007`). | MUST | test |

## 5. Non-functional requirements

- **Determinism:** Model routing metadata serialization in JSON and prompt markdown must be byte-identical across runs. Map iterations must be sorted.
- **Fail-safe:** If an unknown tier or reasoning effort is encountered in `.sdd/roles.json`, `sdd check` flags an error and `sdd prompt` falls back safely to default routing.
- **Backward compatibility:** Older `.sdd/roles.json` files without `model_tier` or `reasoning_effort` remain valid and parse cleanly without errors.

## 6. Constraints and assumptions

- Model tiers represent abstract capability classes (`frontier`, `advanced`, `standard`, `fast`) to remain provider-agnostic.
- `recommended_family` provides representative examples across major model families (Claude, Gemini, OpenAI) without hard vendor lock-in.

## 7. Open questions

(none — Decision D-049 Approved on 2026-11-24)

## 8. Acceptance criteria

- **AC-1 (Schema & Canonical Roles):** `.sdd/roles.json` contains `model_tier`, `reasoning_effort`, and `recommended_family` for all 11 canonical roles.
- **AC-2 (Check Validation):** A role with invalid `model_tier: "super-fast"` fails `sdd check` with a `roles-tier` error; a role with invalid `reasoning_effort: "extreme"` fails with a `roles-reasoning` error.
- **AC-3 (Prompt Integration):** `sdd prompt review T-059` outputs `## Model Routing Guidance` detailing the recommended tier for the requested lens; `sdd prompt review T-059 --json` includes a populated `model_routing` object.
- **AC-4 (Non-Destructive Upgrade):** Upgrading a legacy `roles.json` without routing fields populates canonical tiers while preserving custom roles and existing overrides.
- **AC-5 (Template Version Bump):** `templates.Version` is bumped to `2.7.0`, contracts match `templates=2.7.0`, and regenerating a test kit passes `sdd check --strict`.
- **AC-6 (Standard Library Purity):** Zero external dependencies in `go.mod`.
