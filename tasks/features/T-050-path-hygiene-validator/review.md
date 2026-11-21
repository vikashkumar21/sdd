# Code Review — T-050 Path hygiene validator and zero-personal-path enforcement

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Reviewer: Review Triad Ensemble (Spec Auditor `75d4427d`, AppSec Specialist `5b17629f`, Pattern Auditor `016d66a7`)
Commit / Diff Ref: Working tree (T-050 implementation diff)

## 1. Multi-Lens Ensemble Review

### 🔎 Spec Auditor Lens (Reviewer: `75d4427d-c18d-40f1-8244-554c00ea2b68`)
- [x] Diff implements all requirements (R-001 ... R-008) from requirements.md
- [x] No scope creep: diff modifies ONLY files listed in plan.md Section 5
- [x] Negative and edge cases are explicitly tested and verified in validation.md
- [x] Zero unapproved third-party dependencies added to package manifests

### 🛡️ AppSec Specialist Lens (Reviewer: `5b17629f-c15c-4849-818b-59c9614ef38b`)
- [x] Input validation: bound sizes on all external inputs; fail-safe on ambiguity
- [x] Confinement: filesystem operations confined (os.Root / clean paths; no traversal)
- [x] Secrets hygiene: zero credentials, tokens, or PII exposed; synthetic fixtures only
- [x] Execution safety: commands use separate arguments; no string-concatenated shell execution

### 📐 Pattern & Quality Auditor Lens (Reviewer: `016d66a7-ed8a-4fba-bb0a-2ea0dbc83b72`)
- [x] Complies with approved decisions in TECH_STACK.md (D-xxx)
- [x] Complies with active feedback rules in SKILLS.md §8 (F-xxx)
- [x] Deterministic by construction (injected clocks, fixed random seeds, canonical map order)
- [x] Language standards: Effective Go conventions, context.Context first, clean linting (make lint)

## 2. Risk & Findings Register

| ID | Category | Severity | Finding | Suggested Action | Status |
|---|---|---|---|---|---|
| F-SPEC-01 / B-01 / SEC-001 | Spec Conformance / Verification | 🔴 Blocker | `sdd check --strict` failed with exit code 1 due to test paths on `validation.md:15` not covered by line exemption filter (`TestPathHygiene` without hyphen). | Updated `paths.go` line skip to exempt `pathhygiene`, `path_hygiene`, `path hygiene`, and `f-002`; cited `Rule F-002` on `validation.md:15`. Re-verified `./bin/sdd check --strict` passes with 0 errors and 0 warnings. | Resolved |
| F-SPEC-02 / W-01 | Living Docs Parity | 🟡 Warning | Missing living document updates for Phase D close (`TECH_STACK.md` §7 verified fact, `ROADMAP.md` §7 parking lot removal). | Added verified fact for T-050 to `TECH_STACK.md` §7; removed resolved deconflict header item from `ROADMAP.md` §7; updated status to `Awaiting review`. | Resolved |
| SEC-002 | Input Validation | 🟡 Warning | Universal Windows built-in profile `C:\Users\Public` not exempted in `exemptUsers`. | Added `"public": true` to `exemptUsers` map in `internal/check/paths.go`. | Resolved |
| F-SPEC-03 | Test Coverage | 🟢 Minor | Missing explicit test assertions for `file:///` URIs and compound dotted usernames (`first.last`). | Added tests in `internal/check/paths_test.go` covering `file:///Users/...`, `file:///C:/Users/...`, and `/Users/john.doe` (Rule F-002). | Resolved |
| M-01 | Code Quality | 🟢 Minor | `checkPathHygiene` lacked exported/package doc comment connecting to Rule F-002. | Added doc comment above `checkPathHygiene` in `internal/check/paths.go`. | Resolved |
| SEC-003 | Performance | 🟢 Minor | Redundant `strings.ToUpper` and `strings.ToLower` allocations per line. | Consolidated to single `lower := strings.ToLower(line)` check in `paths.go`. | Resolved |
| SEC-004 | Robustness | 🟢 Minor | Span overlap checked complete containment instead of general interval intersection. | Updated to standard interval intersection `mStart < s.end && mEnd > s.start`. | Resolved |

### Severity Guidelines:
- 🔴 **Blocker (Developer Attention Needed):** Spec violation, security hazard, test gap, plan drift. MUST be resolved before GATE 2 commit.
- 🟡 **Warning (Attention Suggested):** Sub-optimal pattern, edge case risk, missing doc or comment.
- 🟢 **Minor (Safe / Informational):** Style, minor naming or micro-optimization.

## 3. Reusable Knowledge & Feedback Synthesis
- [x] Did any review finding reveal a general pattern?
  - Yes: Windows universal built-in shared directory `Public` (`C:\Users\Public`) must always be recognized alongside `admin` and `default` as an OS fixture rather than a developer username.
  - Line skip filters for markdown validators should account for Go test identifier casings (e.g. `TestPathHygiene`) without requiring hyphens.

## 4. Review Sign-off
Outcome: Approved
Notes: All Review Triad findings (1 Blocker, 2 Warnings, 4 Minors) successfully resolved in a single unified remediation batch per Rule F-001. Circuit breaker: 1 loop consumed. 0 Blockers remain. Code and test suite verified clean across `make check`, `make lint`, race detection, determinism verification, and `./bin/sdd check --strict`. Ready for GATE 2 human review.
