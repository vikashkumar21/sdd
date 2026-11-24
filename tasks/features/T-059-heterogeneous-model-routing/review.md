---
status: completed
---
# Code Review — T-059 Heterogeneous role model routing, tier specifications, and provider-agnostic dispatch

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Reviewer: Review Triad (spec-auditor, appsec-specialist, pattern-auditor)
Commit / Diff Ref: HEAD (T-059 candidate)

## 1. Requirements Traceability Ledger

| Req ID | Requirement Statement | Code Symbol & File | Positive Test Evidence | Negative / Boundary Test Evidence | Status |
|---|---|---|---|---|---|
| R-001 | The system SHALL extend `.sdd/roles.json` and `internal/role/role.go` with `model_tier`, `reasoning_effort`, and `recommended_family` metadata for all canonical roles. | `RoleInfo`, `DefaultRoleConfig` in `internal/role/role.go` | `internal/role/role_test.go:TestModelRouting_CanonicalDefaults`, `TestResolveRoleRouting` | `internal/role/role_test.go:TestValidateModelTier`, `TestValidateReasoningEffort` | Verified |
| R-002 | The system SHALL enforce mechanical validation in `sdd check` verifying valid `model_tier` (`frontier`, `advanced`, `standard`, `fast`) and `reasoning_effort` (`high`, `medium`, `low`, `none`) values. | `checkRoles` in `internal/check/roles.go` | `internal/check/roles_test.go:TestRolesCheck_ModelRouting` (case 1) | `internal/check/roles_test.go:TestRolesCheck_ModelRouting` (cases 2, 3, 4, 5) | Verified |
| R-003 | The system SHALL enrich all `sdd prompt` commands (`review`, `plan-review`, `implement`, `discovery`, `repair`) with `## Model Routing Guidance` in Markdown output and structured `model_routing` in JSON output. | `sdd prompt` commands in `internal/cli/` | `internal/cli/prompt_test.go:TestPromptReview_ModelRouting`, `TestPromptPlanReview_ModelRouting`, `TestPromptImplement_ModelRouting`; `internal/cli/prompt_discovery_test.go:TestPromptDiscovery_ModelRouting`; `internal/cli/prompt_repair_test.go:TestPromptRepair_ModelRouting` | `TestPromptReview_ModelRouting` (triad vs single lens) | Verified |
| R-004 | The system SHALL non-destructively backfill missing model routing metadata in existing `.sdd/roles.json` files during `sdd upgrade` without clobbering custom role definitions. | `upgradeRoles` in `internal/upgrade/upgrade.go` | `internal/upgrade/upgrade_test.go:TestUpgrade_RolesModelRouting` | `TestUpgrade_RolesModelRouting` (non-canonical role preserved) | Verified |
| R-005 | The system SHALL update starter templates, bump `templates.Version` to `2.7.0`, and verify fresh kit regeneration and contract parity per Hard Rule 16. | `templates.Version` in `internal/templates/templates.go`, contracts, manifests | `internal/templates/templates_test.go:TestVersion`, `internal/check/check_test.go:TestFreshKitHasNoErrors` | `internal/check/check_test.go:TestDetectsProblems/agent_contract_drift` | Verified |
| R-006 | The system SHALL document Heterogeneous Model Routing in `skills/task-coordinator/SKILL.md`, embedded templates, and living documents. | `skills/task-coordinator/SKILL.md` | `git grep -q Heterogeneous skills/task-coordinator/SKILL.md` | Living docs inspection | Verified |
| R-007 | The system SHALL implement all model routing, schema validation, prompt formatting, and upgrade logic using only the Go standard library with zero external dependencies (D-007). | Stdlib only across all packages | `go test ./...` passes without network | `go.mod` contains 0 `require` blocks | Verified |

## 2. Multi-Lens Ensemble Review

### 🔎 Spec Auditor Lens
- [x] Diff implements all requirements (R-001 ... R-007) from requirements.md
- [x] No scope creep: diff modifies ONLY files listed in plan.md Section 5
- [x] Negative and edge cases are explicitly tested and verified in validation.md
- [x] Zero unapproved third-party dependencies added to package manifests

### 🛡️ AppSec Specialist Lens
- [x] Input validation: bound sizes on all external inputs; fail-safe on ambiguity
- [x] Confinement: filesystem operations confined (os.Root / clean paths; no traversal)
- [x] Secrets hygiene: zero credentials, tokens, or PII exposed; synthetic fixtures only
- [x] Execution safety: commands use separate arguments; no string-concatenated shell execution

### 📐 Pattern & Quality Auditor Lens
- [x] Complies with approved decisions in TECH_STACK.md (D-049, D-007)
- [x] Complies with active feedback rules in SKILLS.md §8 (F-001, F-002, F-003)
- [x] Deterministic by construction (injected clocks, fixed random seeds, canonical map order)
- [x] Language standards: Effective Go conventions, context.Context first, clean linting (make lint)

## 3. Risk & Findings Register

| ID | Category | Severity | Finding | Suggested Action | Status |
|---|---|---|---|---|---|
| SEC-001 | Fail-Safe Defaults | 🔴 Blocker | `ResolveRoleRouting` did not validate `configured.ModelTier`/`ReasoningEffort`, returning invalid strings instead of falling back. | Validated in `ResolveRoleRouting` and fall back to canonical defaults or Frontier/Medium. | Resolved |
| PAT-01 | Idiomatic Go | 🔴 Blocker | `make fmt-check` failed on `internal/cli/prompt_discovery.go` due to extra indentation tabs. | Formatted with `gofmt -s -w internal/cli/prompt_discovery.go`; `make fmt-check` now passes cleanly. | Resolved |
| SEC-002 | Input Validation | 🟡 Warning | `checkRoles` did not enforce single-line format or maximum length limit (256 chars) on `recommended_family`. | Added single-line and length bounds check in `internal/check/roles.go`. | Resolved |
| PAT-02 | Living Docs | 🟡 Warning | `TECH_STACK.md:210` recorded milestone as `M15` instead of `M14`. | Updated `TECH_STACK.md` milestone to `M14`. | Resolved |
| F-059-SPEC-02 | Validation Evidence | 🟡 Warning | `validation.md` had pending placeholders. | Populated real execution outputs and marked all requirements passed in `validation.md`. | Resolved |
| SEC-003 | Determinism | 🟢 Minor | `upgrade.go` iterated over `rolesDefault.Roles` non-deterministically. | Changed iteration to `role.CanonicalRoleNames()` in `internal/upgrade/upgrade.go`. | Resolved |
| PAT-03 | Code Quality | 🟢 Minor | Unconfigured role fallback in `ResolveRoleRouting` lacked an explicit explanatory comment. | Added clarifying code comment on Frontier/Medium default routing for custom roles. | Resolved |

## 4. Reusable Knowledge & Feedback Synthesis
- **Fail-Safe Enums in Resolvers:** Resolvers that read user manifests must validate values against domain enums before accepting them, falling back to safe canonical defaults on unrecognized values.

## 5. Review Sign-off
Outcome: Approved
Notes: 0 Blockers; all 7 findings resolved in unified fix batch per Rule F-001. All 3 lenses signed off.
