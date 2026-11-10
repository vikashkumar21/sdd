# Requirements — T-026 Official code-review skill and task review template (review.md)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-05)
Roadmap ref: `ROADMAP.md` T-026

## 1. Goal
Provide a standardized AI code review specification and artifact in `sdd`, introducing a dedicated `review.md` template for task folders, a canonical `agentskills.io` `code-review` skill for AI agents, and mechanical consistency checks in `sdd check`.

## 2. Context and references
- Mission / principles touched: Principle 1 (Deterministic), Principle 2 (Fail safe), Principle 3 (Human in the loop for the irreversible), Principle 5 (Never overwrite human work).
- Decisions this relies on: D-007 (Stdlib only), D-024 (agentskills.io open standard), D-026 (Approved - Code review specification, rubric, and task artifact).
- Depends on tasks: T-025 (Universal installer scripts).

## 3. Scope
**In scope**
- Task Review Artifact:
  - New embedded template `internal/templates/data/features/review.md`.
  - Scaffolding in `features/_templates/review.md` on `init` and `upgrade`.
  - Automatic creation of `features/T-xxx-<slug>/review.md` by `sdd feature new`.
- Review Rubric & Structure:
  - Header: Status (`Pending`, `Changes requested`, `Approved`), Reviewer, Diff Ref.
  - Section 1: Scope & Spec Alignment checklist (traceability to `R-xxx`, strictly within `plan.md` file list, no unapproved dependencies).
  - Section 2: Architectural & Standards Conformance checklist (`TECH_STACK.md` D-xxx decisions, `SKILLS.md` F-xxx feedback rules, determinism).
  - Section 3: 3-Tier Risk & Findings Register (ID, Category, Severity, Finding, Suggested Action):
    - 🔴 **Blocker (Developer Attention Needed):** Spec violation, security hazard, test gap, plan drift. Blocks GATE 2.
    - 🟡 **Warning (Attention Suggested):** Sub-optimal pattern, edge case risk, missing doc.
    - 🟢 **Minor (Safe / Informational):** Style, minor naming or performance micro-optimization.
  - Section 4: Reusable Knowledge & Feedback Synthesis (turning review findings into `SKILLS.md` `F-xxx` rules or new skills).
- Canonical Agent Skill:
  - Reference `code-review` skill definition in `internal/skill` conforming to `agentskills.io` standard.
  - Exportable via `sdd skill export --skill code-review` (or `--skill all` / default options).
- Consistency Checking:
  - `sdd check` verifies that tasks marked `Awaiting review` or `Done` have zero unresolved 🔴 Blocker findings in `review.md`.
  - `sdd check` verifies review status is valid (`Pending`, `Changes requested`, `Approved`).
- Templates Version Bump:
  - Bump `templates.Version` per Rule 14.

**Out of scope**
- LLM API calls or automated model invocation (that belongs in external agent orchestrators; SDD remains pure, deterministic, stdlib-only).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL provide `features/_templates/review.md` embedded template and include it in new and upgraded kits. | MUST | test / demo |
| R-002 | `review.md` SHALL structure code review into Scope Alignment, Architecture Conformance, 3-Tier Risk Register (🔴 Blocker, 🟡 Warning, 🟢 Minor), and Feedback Synthesis. | MUST | inspection / test |
| R-003 | `sdd feature new` SHALL generate `features/T-xxx-<slug>/review.md` alongside requirements, plan, and validation when the template exists. | MUST | test |
| R-004 | The system SHALL provide a canonical `code-review` skill definition conforming to `agentskills.io` standard. | MUST | test |
| R-005 | `sdd skill export` SHALL default to exporting all canonical skills (`sdd-protocol` and `code-review`), while supporting `--skill <name>` for individual skills. | MUST | test |
| R-006 | `sdd upgrade` SHALL create `features/_templates/review.md` when upgrading older projects to bring review functionality to existing kits without overwriting human edits. | MUST | test |
| R-007 | `sdd check` SHALL verify that when a task has a `review.md` and status is `Awaiting review` or `Done`, there are no unresolved 🔴 Blocker findings. | MUST | test |
| R-008 | `sdd check` SHALL verify that `review.md` status is one of `Pending`, `Changes requested`, or `Approved`. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** `review.md` rendering and validation must produce byte-identical, deterministic results given identical inputs.
- **Backwards Compatibility:** Projects initialized before T-026 lacking `features/_templates/review.md` SHALL continue to function with `sdd feature new` without error.
- **Standard Library Only:** Zero external Go dependencies (`D-007`).
- **Template Versioning:** Rule 14 compliance (bump `templates.Version`, test in temp directory).

## 6. Constraints and assumptions
- Review template resides in `features/_templates/review.md`.
- Canonical skill resides in `internal/skill` and exports to `skills/code-review/SKILL.md`.

## 7. Open questions
(None. Settled in approved Decision D-026.)

## 8. Acceptance criteria
- Running `sdd feature new T-xxx slug` generates all four documents: `requirements.md`, `plan.md`, `validation.md`, and `review.md`.
- Running `sdd skill export --skill code-review` exports a valid `skills/code-review/SKILL.md` passing `sdd check`.
- Running `sdd check` flags an error/warning if a task is marked `Awaiting review` or `Done` while `review.md` contains an unresolved 🔴 Blocker.
