---
status: completed
---
# Code Review — T-060 Subagent model routing, tier enforcement, and calibrated reasoning dispatch

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Reviewer: Independent Review Triad (Spec Auditor, AppSec Specialist, Pattern Auditor)
Commit / Diff Ref: HEAD (feature/T-060-subagent-model-routing)

## 1. Requirements Traceability Ledger

| Req ID | Requirement Statement | Code Symbol & File | Positive Test Evidence | Negative / Boundary Test Evidence | Status |
|---|---|---|---|---|---|
| R-001 | Define runtime model mappings for capability tiers: frontier (inherit/pro/opus), advanced (inherit/pro/sonnet), standard (flash/haiku/gpt-4o), fast (flash_lite/haiku/gpt-4o-mini) | `skills/task-coordinator/SKILL.md:270-285`, `SKILLS.md:340-365`, `internal/cli/prompt.go:218-230` | `git grep -n frontier skills/task-coordinator/SKILL.md` (Section 5 tier tables) | `TestResolveSubagentModelParam` in `internal/cli/prompt_test.go:466-498` (unknown tier fallback to inherit) | Verified |
| R-002 | Mandate explicit Model parameter during subagent invocation: Model="flash" for spec-auditor, pattern-auditor, qa, devops and Model="inherit"/pro for appsec-specialist and plan-reviewer | `skills/task-coordinator/SKILL.md:290-325`, `skills/code-review/SKILL.md:160-175`, `internal/templates/data/skills/coordinator/SKILL.md` | `git grep -n Model: skills/task-coordinator/SKILL.md` (invocation blocks) | `TestCheckCoordinatorTierDispatch_SkillMissingFlash` in `internal/check/coordinator_test.go:107-131` | Verified |
| R-003 | Codify Tier-Calibrated Subagent Dispatch Rule in agent contracts (CLAUDE.md, GEMINI.md, AGENTS.md) and contract.md.tmpl | `internal/templates/data/base/contract.md.tmpl:41-44,142-145`, `CLAUDE.md:41-44,142-145`, `GEMINI.md:41-44,142-145`, `AGENTS.md:41-44,142-145` | `./bin/sdd check --strict` (0 errors, 0 warnings; contract parity verified) | `TestCheckCoordinatorTierDispatch_ContractMissingTierDispatch` and `_ContractMissingRole` in `internal/check/coordinator_test.go:133-181` | Verified |
| R-004 | Output concrete executable invocation snippets with Model parameter in sdd prompt review, plan-review, and implement | `internal/cli/prompt.go:218-265,305-325,485-520` (`resolveSubagentModelParam`, `roleTitleFor`, `formatModelRoutingSection`) | `TestPromptReview_ModelRouting`, `TestPromptPlanReview_ModelRouting`, `TestPromptImplement_ModelRouting` in `internal/cli/prompt_test.go` | Table-driven `TestResolveSubagentModelParam` and `TestRoleTitleFor` covering empty, whitespace, unknown tiers/roles in `internal/cli/prompt_test.go:466-538` | Verified |
| R-005 | Mechanical rule `coordinator-tier-dispatch` in internal/check verifying coordinator skills and contracts declare model tier dispatch | `internal/check/coordinator.go:11-54` (`checkCoordinatorTierDispatch`), registered in `internal/check/check.go:270` | `TestCheckCoordinatorTierDispatch_Valid` in `internal/check/coordinator_test.go:9-52` | `TestCheckCoordinatorTierDispatch_SkillMissingRole`, `_SkillMissingFlash`, `_ContractMissingRole`, `_ContractMissingTierDispatch`, and `_MissingSkill_PassesCleanly` in `internal/check/coordinator_test.go:54-181` | Verified |
| R-006 | Bump templates.Version to 2.8.0 in templates.go and manifest.json; verify kit generation and sdd upgrade --dry-run | `internal/templates/templates.go:18`, `.sdd/manifest.json:4`, `contract headers (templates=2.8.0)` | `TestVersion` confirms 2.8.0; `TestFreshKitHasNoErrors`, `TestUpgradeAlreadyUpToDate`, and `TestUpgradeDryRunMigration` in `internal/templates/templates_test.go` and `internal/upgrade/upgrade_test.go` pass | Upgrade migration dry-run and stale version detection tests pass | Verified |
| R-007 | 100% standard library purity (D-007), pass make verify, race-clean, and determinism (-count=2) | All 35 Go packages | `make verify` passes clean; `./bin/sdd check --strict` passes (0 errors, 0 warnings); `go test -race -count=2` clean | Package-scoped `golangci-lint` passes with 0 issues | Verified |

## 2. Multi-Lens Ensemble Review

### 🔎 Spec Auditor Lens
- [x] Diff implements all requirements (R-001 ... R-007) from requirements.md
- [x] No scope creep: diff modifies ONLY files listed in plan.md Section 5
- [x] Negative and edge cases are explicitly tested and verified in validation.md
- [x] Zero unapproved third-party dependencies added to package manifests

### 🛡️ AppSec Specialist Lens
- [x] Input validation: model tier and parameter names validated against safe canonical enums with safe default fallbacks
- [x] Confinement: filesystem operations confined; zero arbitrary shell or script execution
- [x] Secrets hygiene: zero credentials, tokens, or PII exposed; synthetic fixtures only
- [x] Command safety: prompt outputs use structured invocation blocks; no shell string concatenations

### 📐 Pattern & Quality Auditor Lens
- [x] Complies with approved decisions in TECH_STACK.md (D-049, D-050)
- [x] Complies with active feedback rules in SKILLS.md §8 (F-001, F-002, F-003)
- [x] Deterministic by construction: pure role resolution, no environment or clock dependencies
- [x] Language standards: Effective Go conventions, context first, clean linting (package-scoped 0 issues)

## 3. Risk & Findings Register

| ID | Category | Severity | Finding | Suggested Action | Status |
|---|---|---|---|---|---|
| SPEC-001 | Functional Check | 🔴 Blocker | `checkCoordinatorTierDispatch` did not check `appsec-specialist` or `pattern-auditor` in contract, and used overly loose string matching in skill. | `internal/check/coordinator.go` updated to validate all 3 triad roles (`spec-auditor`, `appsec-specialist`, `pattern-auditor`) and mandate explicit `Model="flash"` and `Model="inherit"`. Added 6 negative unit tests in `coordinator_test.go`. | Resolved |
| SPEC-002 | Test Gaps | 🔴 Blocker | Missing negative and boundary tests for `resolveSubagentModelParam` (fast tier, empty, unknown tier) and `roleTitleFor` (empty/whitespace role fallback). | Added table-driven test suites `TestResolveSubagentModelParam` and `TestRoleTitleFor` in `internal/cli/prompt_test.go:466-538`. | Resolved |
| SPEC-003 | Doc Evidence | 🟡 Warning | `validation.md` had `Status: Not run` and lacked R-006 temp kit upgrade command evidence. | `validation.md` updated with `Status: Passed` and R-006 test evidence for `TestFreshKitHasNoErrors`, `TestUpgradeAlreadyUpToDate`, and `TestUpgradeDryRunMigration`. | Resolved |
| SPEC-004 | Invariant Scope | 🟡 Warning | `internal/skill/export.go` contains legacy protocol export strings without explicit `Model` parameter. | Rule F-001 Justification: `internal/skill/export.go` is outside `plan.md` Section 5. Editing it would violate plan boundary confinement (Hard Rule 11). Canonical embedded templates in `internal/templates/data/skills/` are fully updated and verified by `TestExport`. Retaining `internal/skill/export.go` as-is has zero functional impact and is justified. | Justified |
| SPEC-005 | Model Routing Tests | 🟢 Minor | `prompt_test.go` skipped explicit CLI test cases for `--lens security`, `--lens pattern`, and `--role devops`. | `internal/cli/prompt_test.go` updated with explicit CLI test executions for `--lens security`, `--lens pattern`, and `--role devops`. | Resolved |
| PQ-001 | Checker Robustness | 🟡 Warning | Weak heuristic in `checkCoordinatorTierDispatch` for `skills/task-coordinator/SKILL.md`. | Unified with SPEC-001: checker mandates explicit `Model="flash"` / `Model="inherit"` parameter enforcement. | Resolved |
| PQ-002 | Static Analysis | 🟡 Warning | `validation.md` left `make lint` pending while repo-wide lint fails on pre-existing `internal/hook/hook.go`. | `validation.md` records package-scoped `golangci-lint` (0 issues) and documents the 4 pre-existing `hook.go` issues. | Resolved |
| PQ-003 | Living Doc Parity | 🟡 Warning | `TECH_STACK.md` D-050 used unbolded `Approved 2026-11-25`. | Formatted `D-050` as `**Approved** 2026-11-25` in `TECH_STACK.md:157`. | Resolved |
| PQ-004 | Code Deduplication | 🟢 Minor | `buildImplementPrompt` had duplicated local role switch. | Replaced local switch with `roleTitle := roleTitleFor(targetRole)`. | Resolved |
| PQ-005 | Doc Syntax | 🟢 Minor | Code fences for `invoke_subagent(...)` snippets used ```` ```json ```` instead of ```` ```text ````. | Updated code fence tags from `json` to `text` across `internal/cli/prompt.go`, `SKILLS.md`, and coordinator skills. | Resolved |
| PQ-006 | Doc Consistency | 🟢 Minor | `standard` tier `recommended_family` string varied across `SKILLS.md` §17 and §18. | Aligned consistent recommended families across sections in `SKILLS.md`. | Resolved |
| PQ-007 | Boundary Verification | 🟢 Minor | `resolveSubagentModelParam` unknown tier fallback untested. | Unified with SPEC-002: table-driven test cases added in `prompt_test.go`. | Resolved |

### Severity Guidelines:
- 🔴 **Blocker (Developer Attention Needed):** Spec violation, security hazard, test gap, plan drift. MUST be resolved before GATE 2 commit.
- 🟡 **Warning (Attention Suggested):** Sub-optimal pattern, edge case risk, missing doc or comment.
- 🟢 **Minor (Safe / Informational):** Style, minor naming or micro-optimization.

## 4. Reusable Knowledge & Feedback Synthesis
- [x] Did any review finding reveal a general pattern?
  - Yes: Ephemeral tabula-rasa reviewers should test fallback and boundary behaviors for enum parsing functions (`resolveSubagentModelParam`, `roleTitleFor`) and ensure checker rules test all members of an ensemble triad, not only the first member.
  - Dogfooding `Model: "flash"` on `spec-auditor` and `pattern-auditor` drastically reduced subagent latency and token consumption while maintaining rigorous tabular verification standards.

## 5. Review Sign-off
Outcome: Approved
Notes: Certified by Independent Review Triad (Loop 2) with 0 Blockers and all findings resolved or justified per Rule F-001.
- 🔎 **Spec Auditor (`fa926889-3ace-4088-a3bc-0eeda649b7e4`)**: Approved (0 Blockers, 0 Warnings, 0 Minors)
- 🛡️ **AppSec Specialist (`c73edd01-4f48-4ba4-98d9-2fc252dc18a6`)**: Approved (0 Blockers, 0 Warnings, 0 Minors)
- 📐 **Pattern & Quality Auditor (`fc6b6bc4-588e-4293-8c71-2bff469e4a60`)**: Approved (0 Blockers, 0 Warnings, 0 Minors)
