# Requirements — T-033 Ensemble code review standards and AppSec Specialist rubric

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-05)
Roadmap ref: `ROADMAP.md` T-033

## 1. Goal
Codify the multi-lens ensemble code review standard (The Review Triad: Spec Auditor, AppSec Specialist, Pattern Auditor) into `skills/code-review/SKILL.md` and embedded templates, add specialized lens support to `sdd prompt review <T-xxx> [--lens <spec|security|pattern|all>]` (including `--json`), update the canonical `review.md` template across all typologies with the ensemble triad checklist, and integrate mechanical verification in `sdd check`.

## 2. Context and references
- Mission / principles touched:
  - Principle 1 (Deterministic): Deterministic multi-lens auditing with objective rubric and criteria.
  - Principle 2 (Fail safe): AppSec Specialist lens enforces least privilege, bounded inputs, and fail-safe error handling.
  - Principle 4 (Small and shippable): Isolated subagent lenses allow concurrent specialization without blocking.
- Decisions this relies on:
  - D-002: Spec-driven workflow with human gates.
  - D-007: Go, standard library only, single static binary.
  - D-024: Open Agent Skills standard (`agentskills.io`).
  - D-026: Code review specification, rubric, and task artifact (`review.md`).
  - D-029: Two-tier git governance and Task Coordinator orchestration (ensemble review standards).
- Depends on tasks: T-032 (Done).

## 3. Scope
**In scope**
- Canonical Code Review Skill Enhancement (`skills/code-review/SKILL.md` and `internal/skill/export.go`):
  - Codify the **Review Triad (Ensemble Lenses)**:
    1. **Spec Auditor:** Requirement traceability (R-xxx to validation evidence), plan boundary compliance, negative/failure test cases verification, zero scope creep.
    2. **AppSec Specialist:** Input sanitation & bounded sizes, path confinement (`os.Root` / path traversal checks), zero credential/token/PII leakage, permission boundaries, fail-safe error paths.
    3. **Pattern & Quality Auditor:** Architecture decision compliance (`D-xxx`), language standards (*Effective Go*, stdlib purity `D-007`), determinism (clocks, random seeds, map ordering), active feedback rules (`SKILLS.md` §8 `F-xxx`).
  - Preserve 3-tier severity rubric (🔴 Blocker, 🟡 Warning, 🟢 Minor) and 3-loop circuit breaker.
- CLI Handover Lens Support (`internal/cli/prompt.go`):
  - Add `--lens <spec|security|pattern|all>` (alias `--lens appsec` for `security`) to `sdd prompt review <T-xxx>`.
  - Generate specialized subagent handover prompts tailored to each reviewer persona when a specific lens is requested.
  - Support `--lens` in `--json` output with structured lens metadata.
- Lifecycle Template Updates (`review.md`):
  - Update `features/_templates/review.md`, `bugfixes/_templates/review.md`, and their embedded templates in `internal/templates/data/` to include the Ensemble Review Triad checklist.
- Mechanical Verification (`internal/check/review.go`):
  - Ensure `sdd check` verifies that `review.md` conforms to the review standard.
- Documentation & Verification:
  - Update `SKILLS.md` §10 to document the ensemble review triad.
  - Update `README.md` to document `--lens` flag.
  - Full verification suite (`make verify`, `make check`, `sdd check --strict`).

**Out of scope**
- Remote CI pull request bots or webhooks (Milestone 9).
- Automated AI code generation or external LLM API calls.

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL update `skills/code-review/SKILL.md` and `CanonicalCodeReviewSkill` in `internal/skill/export.go` to define the 3 specialized ensemble review lenses (Spec Auditor, AppSec Specialist, Pattern Auditor). | MUST | test / inspection |
| R-002 | The system SHALL provide `sdd prompt review <T-xxx> --lens <spec|security|pattern|all>` generating tailored prompts for each reviewer persona. | MUST | test |
| R-003 | `sdd prompt review <T-xxx> --lens security` SHALL accept `appsec` as an alias for `security`. | MUST | test |
| R-004 | `sdd prompt review <T-xxx> --lens <lens> --json` SHALL include the requested lens and lens-specific checklists in the machine-readable output. | MUST | test |
| R-005 | Invalid `--lens` values SHALL be rejected with exit code 2 and a clear error message listing supported lenses. | MUST | test |
| R-006 | The system SHALL update `review.md` templates in `features/_templates/`, `bugfixes/_templates/`, and embedded assets in `internal/templates/data/` to include the Review Triad checklist. | MUST | test / inspection |
| R-007 | `sdd skill export --skill code-review` SHALL export the updated ensemble code-review skill. | MUST | test |
| R-008 | `sdd check` SHALL verify that `review.md` in tasks marked `Awaiting review` or `Done` adheres to the review rubric and has non-placeholder reviewer sign-off. | MUST | test |
| R-009 | `SKILLS.md` §10 SHALL be updated with the ensemble review triad standards and instructions. | MUST | inspection |
| R-010 | The entire test suite, linter (`make check`), and mechanical consistency check (`sdd check --strict`) SHALL pass with 0 errors and 0 warnings. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Lens prompt generation must produce byte-identical outputs across repeated runs with identical inputs.
- **Fail-safe:** If an invalid lens flag is passed, command refuses execution with code 2.
- **Standard library purity:** Standard library only; no external dependencies (D-007).

## 6. Constraints and assumptions
- Skills follow the `agentskills.io` standard (`skills/<name>/SKILL.md`).
- Supported lenses: `spec`, `security` (alias `appsec`), `pattern`, and `all` (default).

## 7. Open questions
(None - all architectural choices settled in D-026 and D-029)

## 8. Acceptance criteria
- Running `sdd prompt review T-033 --lens security` outputs a prompt focused on threat modeling, input validation, and secret hygiene.
- Running `sdd prompt review T-033 --lens spec` outputs a prompt focused on requirements traceability, plan boundaries, and test evidence.
- Running `sdd prompt review T-033 --lens pattern` outputs a prompt focused on architectural decisions, determinism, and language idioms.
- Running `sdd prompt review T-033 --lens all` outputs an ensemble prompt covering all three lenses.
- `make check` and `sdd check --strict` pass with 0 errors and 0 warnings.
