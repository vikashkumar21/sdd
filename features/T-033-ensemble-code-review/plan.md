# Plan — T-033 Ensemble code review standards and AppSec Specialist rubric

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-05)

## 1. Approach summary
Codify the multi-lens ensemble code review standard (The Review Triad: Spec Auditor, AppSec Specialist, Pattern Auditor) into `skills/code-review/SKILL.md` and embedded templates. Enhance `sdd prompt review <T-xxx>` with `--lens <spec|security|pattern|all>` (alias `appsec` -> `security`) and `--json` support. Update the canonical `review.md` template across typologies with the ensemble triad checklist. Document the standard in `SKILLS.md` and `README.md`, and verify through the automated test suite and independent subagent audit.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-026 | Code review specification, rubric, and task artifact (`review.md`) | Option A | Option A | Approved (2026-11-05) |
| D-029 | Two-tier git governance and Task Coordinator orchestration | Option A | Option A | Approved (2026-11-05) |

## 3. Design

### Canonical Code Review Skill Enhancement (`skills/code-review/SKILL.md`)
- Define the **Ensemble Review Triad**:
  1. **🔎 Spec Auditor Lens:**
     - Requirement traceability matrix check (every `R-xxx` has a corresponding test and verification).
     - Plan boundary enforcement: verify `git diff` against `plan.md` Section 5. Zero unapproved file modifications.
     - Negative and failure case coverage check.
  2. **🛡️ AppSec Specialist Lens:**
     - Threat surface and untrusted input audit (strict parsing, size limits).
     - Path traversal & filesystem confinement (`os.Root` bounds).
     - Secrets, tokens, and PII hygiene (ensure fixtures are synthetic).
     - Command execution safety (no unsanitized shell concatenation, argument separation).
     - Fail-safe error handling (restrictive outcome on ambiguity).
  3. **📐 Pattern & Quality Auditor Lens:**
     - Architectural decision alignment (every `D-xxx` strictly respected).
     - Standard library purity (`D-007`, no unapproved third-party dependencies).
     - Determinism by construction (injected clocks, fixed random seeds, canonical map order).
     - Idiomatic language standards (*Effective Go*, *Go Code Review Comments*).
     - Continuous learning compliance (`SKILLS.md` §8 `F-xxx` feedback rules).
- Retain the 3-tier risk rubric (🔴 Blocker, 🟡 Warning, 🟢 Minor) and the 3-loop circuit breaker.
- Mirror changes into `CanonicalCodeReviewSkill` constant in `internal/skill/export.go`.

### CLI Lens Support (`internal/cli/prompt.go`)
- Add `--lens` flag to `sdd prompt review <T-xxx>`:
  - Allowed values: `spec`, `security` (or `appsec`), `pattern`, `all` (default: `all`).
  - Validation: reject invalid lens values with code 2 and help text listing valid options.
- Dynamic Prompt Customization:
  - When `--lens spec`: outputs prompt focused on requirements traceability, negative test cases, and plan boundaries.
  - When `--lens security` or `appsec`: outputs prompt focused on threat vectors, input bounds, path confinement, and secret leaks.
  - When `--lens pattern`: outputs prompt focused on decision register compliance, determinism, language style, and feedback rules.
  - When `--lens all`: outputs comprehensive ensemble prompt covering all three lenses.
- Structured JSON (`--json`):
  - Add `lens` field to `ReviewPromptContext`.
  - Add `lensFocus` array listing specific verification questions for the requested lens.

### Template Updates (`review.md`)
- Update `features/_templates/review.md`, `bugfixes/_templates/review.md`, and embedded templates in `internal/templates/data/` to include the Review Triad checklist:
  ```markdown
  ## 1. Multi-Lens Ensemble Review
  - [ ] 🔎 **Spec Auditor:** Requirements (R-xxx) verified in validation.md; plan boundary strictly respected; zero scope creep.
  - [ ] 🛡️ **AppSec Specialist:** Input bounds, path confinement, zero secret leaks, fail-safe on error.
  - [ ] 📐 **Pattern Auditor:** Architecture decision conformance (D-xxx), determinism, language idioms, SKILLS.md §8 rules.
  ```

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Update `skills/code-review/SKILL.md` and `CanonicalCodeReviewSkill` in `internal/skill/export.go` with the Review Triad standard. | `skills/code-review/SKILL.md`, `internal/skill/export.go`, `internal/skill/export_test.go` | `go test -v ./internal/skill` |
| 2 | Add `--lens` flag, validation, and specialized prompt builders to `internal/cli/prompt.go`. | `internal/cli/prompt.go`, `internal/cli/cli_test.go` | `go test -v ./internal/cli -run TestPromptReview` |
| 3 | Update `review.md` templates across `features/`, `bugfixes/`, and `internal/templates/data/`. | `features/_templates/review.md`, `bugfixes/_templates/review.md`, `internal/templates/data/` | `go test -v ./internal/check` |
| 4 | Update `SKILLS.md` §10 and `README.md` to document the ensemble review triad and `--lens` flag. | `SKILLS.md`, `README.md`, `TECH_STACK.md` | `sdd check --strict` |
| 5 | Run verification suite (`make verify`, `make check`), invoke independent subagent review for Phase C.2 audit, and update `ROADMAP.md`. | `ROADMAP.md`, `features/T-033-ensemble-code-review/` | `make check`, `review.md` sign-off |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `skills/code-review/SKILL.md` | modify | Codify the Ensemble Review Triad standard |
| `internal/skill/export.go` | modify | Update CanonicalCodeReviewSkill with triad standard |
| `internal/skill/export_test.go` | modify | Test canonical skill validation with ensemble standard |
| `internal/cli/prompt.go` | modify | Add `--lens` flag and specialized prompt builders |
| `internal/cli/cli_test.go` | modify | Test `--lens` options, aliases, and JSON output |
| `features/_templates/review.md` | modify | Include Ensemble Review Triad checklist |
| `bugfixes/_templates/review.md` | modify | Include Ensemble Review Triad checklist |
| `internal/templates/data/features/review.md` | modify | Include Ensemble Review Triad checklist |
| `internal/templates/data/bugfixes/review.md` | modify | Include Ensemble Review Triad checklist |
| `SKILLS.md` | modify | Update §10 with ensemble code review standards |
| `README.md` | modify | Document `--lens` flag and ensemble review capabilities |
| `ROADMAP.md` | modify | Update T-033 status |
| `TECH_STACK.md` | modify | Record verified fact for T-033 |
| `features/T-033-ensemble-code-review/requirements.md` | modify | Requirements specification |
| `features/T-033-ensemble-code-review/plan.md` | modify | Implementation plan |
| `features/T-033-ensemble-code-review/validation.md` | modify | Traceability matrix and test results |
| `features/T-033-ensemble-code-review/review.md` | modify | Independent subagent review audit |

## 6. Living docs to update

| File | Update when |
|---|---|
| `ROADMAP.md` | T-033 status changes |
| `SKILLS.md` | §10 updated with ensemble review triad |
| `README.md` | Document `sdd prompt review --lens` |
| `TECH_STACK.md` | Verified facts log |

## 7. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Overly complex prompts that confuse subagent reviewers | Keep prompts concise, imperative, and focused strictly on the selected lens persona with standard rubric |
| Backward compatibility with existing review.md artifacts | Triad checklist is additive; existing review artifacts remain valid in `sdd check` |
| Linter issues with formatted strings | Use `fmt.Fprintf` directly with string builders |

## 8. Rollback plan
Revert changes with `git checkout -- .` and remove `features/T-033-ensemble-code-review/`.

## 9. Approval record
Approved by: Vikash (human)   Date: 2026-11-05   Phrase received: `APPROVED T-033`
