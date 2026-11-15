---
status: active
task_id: T-042
last_reviewed: 2026-11-15
---
# Requirements — T-042 Cross-project and service dependency register in TECH_STACK.md §8

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-042

## 1. Goal
Extend the `TECH_STACK.md` template and this repository's Section 8 with a second table for dependencies on other projects, repositories, and services (name, repo/URL, direction, contract/version, owner, decision ID, status), implement mechanical consistency validation in `sdd check`, and bump template version to `1.8.0` per Contract Rule 16.

## 2. Context and references
- Mission / principles touched:
  - Principle 1 (Deterministic): Clear, explicit integration contracts eliminate agent ambiguity and prevents breaking downstream consumers.
  - Principle 2 (Fail safe): Dangling decision references or invalid directional descriptors fail with clear diagnostics.
  - Principle 3 (Human in the loop): All cross-project linkages link back to a human-approved architectural decision (`D-xxx`).
  - Contract Rule 16: A change to any template or contract bumps `templates.Version` and is verified by regenerating a kit in a temporary directory and running `sdd check` on it.
- Decisions this relies on:
  - D-002: Spec-driven workflow with human gates.
  - D-007: Go standard library only, single static binary.
  - D-013: Template overrides and manifest.
  - D-035: Task state persistence, event journaling, and IPC architecture (`sdd task state`).
  - D-038: Cross-project and service dependency register in TECH_STACK.md §8 and sdd check validation.
- Depends on tasks: T-039 (Done).

## 3. Scope
**In scope**
- Template extension in `internal/templates/data/base/TECH_STACK.md.tmpl`:
  - Split Section 8 into `### 8.1 Package and library dependencies` and `### 8.2 Cross-project and service dependencies`.
  - Canonical 7-column schema: `Name`, `Repo / URL`, `Direction`, `Contract / Version`, `Owner`, `Decision`, `Status`.
- Repository adoption in `TECH_STACK.md`:
  - Adopt Section 8.1 and 8.2 structure in this repo's `TECH_STACK.md`.
  - Register `sdd-agent` as an outbound consumer (`../sdd-agent`, Outbound, Task State JSON IPC v1, Vikash, D-035, Approved).
- Mechanical validation in `sdd check`:
  - Verify that any `D-xxx` referenced in the `Decision` column of §8.2 exists in the Decision register (§6).
  - Verify that `Direction` matches valid descriptors (`Inbound`, `Outbound`, `Bidirectional`, `None`, or placeholder).
  - Verify that `Status` matches valid statuses (`Approved`, `Proposed`, `Deprecated`, `Active`, or placeholder).
  - Verify that table formatting and column counts conform to standard markdown.
- Template version bump:
  - Bump `templates.Version` to `1.8.0` in `internal/templates/templates.go`.
  - Update `internal/templates/templates_test.go` and contract headers across `CLAUDE.md`, `GEMINI.md`, `AGENTS.md`.
  - Validate template regeneration in `internal/templates/templates_test.go` and `internal/scaffold/scaffold_test.go`.

**Out of scope**
- Network access, remote git cloning, or URL pinging (violates D-007 and offline determinism principle; all validation is strictly local and mechanical).
- Automated dependency lockfile syncing across repositories.

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL update `internal/templates/data/base/TECH_STACK.md.tmpl` to partition Section 8 into `### 8.1 Package and library dependencies` and `### 8.2 Cross-project and service dependencies`. | MUST | test / inspection |
| R-002 | The cross-project dependencies table SHALL define 7 canonical columns: `Name`, `Repo / URL`, `Direction`, `Contract / Version`, `Owner`, `Decision`, `Status`. | MUST | test / inspection |
| R-003 | The system SHALL update this repository's `TECH_STACK.md` §8 to adopt §8.1 and §8.2, registering `sdd-agent` under Decision `D-035`. | MUST | inspection / check |
| R-004 | The system SHALL validate in `sdd check` that every decision identifier (`D-xxx`) referenced in the `Decision` column of §8.2 exists in §6 Decision register. | MUST | test |
| R-005 | The system SHALL validate in `sdd check` that the `Direction` column in §8.2 contains an approved directional descriptor (`Inbound`, `Outbound`, `Bidirectional`, `Inbound / Outbound`, `Producer`, `Consumer`, `None`, or placeholder). | MUST | test |
| R-006 | The system SHALL validate in `sdd check` that the `Status` column in §8.2 contains an approved status (`Approved`, `Proposed`, `Deprecated`, `Active`, or placeholder). | MUST | test |
| R-007 | The system SHALL bump `templates.Version` to `1.8.0` in `internal/templates/templates.go` and update contract version markers in `CLAUDE.md`, `GEMINI.md`, and `AGENTS.md` per Contract Rule 16. | MUST | test |
| R-008 | The system SHALL verify that `sdd init` and `sdd upgrade` generate and preserve both dependency tables with zero regressions. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Validation logic is 100% mechanical, deterministic, and offline. Zero external network probing.
- **Fail-safe behaviour:** Dangling decision IDs or malformed table columns report actionable line-precise findings in `sdd check`.
- **Standard library purity:** Zero external dependencies (pure Go stdlib per D-007).

## 6. Constraints and assumptions
- Table parsing uses the project's robust markdown scanner (`internal/markdown`).
- Legacy kits without Section 8.2 pass `sdd check` cleanly for backward compatibility.

## 7. Open questions
(None. Section is empty for GATE 1.)

## 8. Acceptance criteria
- **AC-1 (Template Structure):** Given a newly initialized project via `sdd init`, `TECH_STACK.md` contains both §8.1 and §8.2 tables.
- **AC-2 (Decision Traceability):** Given a §8.2 row referencing an unregistered decision identifier (e.g. `D-9999`) where the decision is not in §6, `sdd check` emits a `techstack-dep-decision` error.
- **AC-3 (Direction & Status Validation):** Given a §8.2 row with invalid direction (e.g. `Sideways`), `sdd check` emits a `techstack-dep-direction` warning. Given an invalid status, it emits a `techstack-dep-status` warning.
- **AC-4 (This Repo Adoption):** Given this repository's `TECH_STACK.md`, `sdd-agent` is registered under §8.2 and passes `sdd check --strict`.
- **AC-5 (Template Version Bump):** Given `templates.Version == "1.8.0"`, regenerating a kit in a temporary directory passes `sdd check --strict`.
