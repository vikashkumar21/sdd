---
status: draft
---
# Plan — T-056 Forensic brownfield discovery hardening, coverage ledger, and multi-language review templates

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Implementing

## 1. Approach summary

Harden brownfield discovery and multi-language support across six concrete pillars:
1. **Review Bundle Enrichment:** In `internal/audit/bundle.go`, update `StagePlanBundle` to freeze `validation.md` into `spec.md`, and update `StageDiscoveryBundle` to copy `.sdd/context/` items into `.sdd/review/<TASK_ID>/discovery/context/`.
2. **Unanchored Challenger Protocol:** Update `internal/templates/data/skills/discovery-challenger/SKILL.md` to instruct the challenger to build an independent repository inventory first, diff against the analyst's Coverage Ledger, and verify against staged reference documents using generalized language-agnostic vectors.
3. **Forensic Discovery Checklist & Coverage Ledger:** Update `internal/templates/data/skills/codebase-analyst/SKILL.md` and `internal/cli/prompt_discovery.go` with platform/runtime defaults, container/CI inventory, dependency EOL, and the mandatory `## Coverage Ledger` format.
4. **Language-Pack-Aware Code Review Scaffolding:** Add language review standards to `internal/templates/data/packs/` (`go`, `java`, `python`, `typescript`, `generic`), parameterize `review.md` in review-bearing typologies (`features` and `bugfixes`), and wire language detection in `internal/feature` and `internal/scaffold/plan.go`.
5. **Mechanical Checks in `sdd check`:** Author `citation-hygiene` (rejecting references to `scratch/`, `/tmp/`, `.system_generated/`) and `coverage-ledger` (validating coverage tables in discovery tasks).
6. **Template Version Bump:** Bump `templates.Version` to `2.4.0` in `internal/templates/templates.go`, update contract markers across `AGENTS.md`, `GEMINI.md`, `CLAUDE.md`, and verify `internal/upgrade` tests.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-048 | Forensic brownfield discovery hardening, coverage ledger, and multi-language review templates | Option A: Coverage ledger, unanchored challenger protocol, bundle enrichment, language-pack review templates / Option B: Leave as-is | Option A: Essential for reliable brownfield discovery across Go, Java, Python, and TypeScript codebases. | Approved |

## 3. Design

### 1. Bundle Staging Interfaces (`internal/audit/bundle.go`)
- `StagePlanBundle(root, taskID string, now ...time.Time) (*PlanBundle, error)`
  - Reads `requirements.md`, `plan.md` (or `patch.md`), and `validation.md` (if present) into `spec.md`.
- `StageDiscoveryBundle(root, taskID, findingsContent, rubricContent string, now ...time.Time) (*DiscoveryBundle, error)`
  - Checks `.sdd/context/`; if entries exist, copies each staged artifact into `.sdd/review/<TASK_ID>/discovery/context/<origin-basename>` and registers them in `DiscoveryBundle.Files`.

### 2. Language Review Standards Taxonomy (`internal/templates/data/packs/` & `internal/templates/templates.go`)
Extend `templates.Pack` struct in `internal/templates/templates.go` with `ReviewStandards string`, populated by reading `packs/<id>/review_standards.md`:
```go
type Pack struct {
	ID              string
	Title           string
	Standards       string
	Contract        string
	SkillMap        string
	ReviewStandards string // markdown standard for review.md
}
```
Add `review_standards.md` to each pack:
- `go/review_standards.md`: "Effective Go conventions, context.Context first, clean linting (make lint / golangci-lint)"
- `java/review_standards.md`: "Idiomatic Java conventions, thread safety, clean build (gradle check / mvn verify)"
- `python/review_standards.md`: "PEP 8 / PEP 257 standards, type hints, clean linting (ruff / pytest)"
- `typescript/review_standards.md`: "Strict TypeScript idioms, async/await error propagation, clean linting (npm test / eslint)"
- `generic/review_standards.md`: "Idiomatic language conventions, clean build and test suite"

### 3. Check Rules (`internal/check/check.go` & `internal/check/paths.go`)
- `citation-hygiene`: RE2 regex detecting ephemeral paths (`(?i)(?:^|[\s`"'(])((?:/tmp/|scratch/|\.sdd/tmp/|\.system_generated/)[^\s`"')]+)`) in tracked markdown documents.
- `coverage-ledger`: Inspects completed discovery task folders or discovery living docs to ensure tables containing `Coverage Ledger` and `Lines Read` exist with non-empty justifications for any skipped lines.

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Update `StagePlanBundle` to include `validation.md` in `spec.md`, and `StageDiscoveryBundle` to copy `.sdd/context/` files into `.sdd/review/<TASK_ID>/discovery/context/`. | `internal/audit/bundle.go`, `internal/audit/bundle_test.go` | `go test -v ./internal/audit -run TestStageBundle` |
| 2 | Add language-specific review standards to language packs, extend `templates.Pack`, and update `review.md` in `internal/templates/data/tasks/features/` and `bugfixes/`. | `internal/templates/templates.go`, `internal/templates/data/packs/**/review_standards.md`, `internal/templates/data/tasks/features/review.md`, `internal/templates/data/tasks/bugfixes/review.md` | `go test -v ./internal/templates` |
| 3 | Wire language-pack review standards into `internal/scaffold/plan.go` and `internal/feature/feature.go` task scaffolding so `sdd init` / `sdd upgrade` seeds project-language-appropriate review rubrics into `tasks/<typology>/_templates/review.md`, and `sdd feature new` / `sdd bugfix new` preserves them. | `internal/scaffold/plan.go`, `internal/feature/feature.go`, `internal/feature/feature_test.go`, `internal/scaffold/scaffold_test.go` | `go test -v ./internal/feature ./internal/scaffold -run "TestFeatureNew_LanguageReview|TestScaffold"` |
| 4 | Author `citation-hygiene` and `coverage-ledger` checks in `internal/check/` with regression tests. | `internal/check/check.go`, `internal/check/paths.go`, `internal/check/paths_test.go`, `internal/check/check_test.go` | `go test -v ./internal/check -run "TestCitationHygiene|TestCoverageLedger"` |
| 5 | Update `codebase-analyst` and `discovery-challenger` role skills and `prompt_discovery.go` with Coverage Ledger, runtime defaults, and unanchored challenger protocol. | `internal/templates/data/skills/codebase-analyst/SKILL.md`, `internal/templates/data/skills/discovery-challenger/SKILL.md`, `internal/cli/prompt_discovery.go`, `internal/cli/prompt_discovery_test.go` | `go test -v ./internal/cli -run TestPromptDiscovery` |
| 6 | Bump `templates.Version` to `2.4.0` in `internal/templates/templates.go`, `.sdd/manifest.json`, `internal/templates/data/base/contract.md.tmpl`, `AGENTS.md`, `GEMINI.md`, `CLAUDE.md`, and update upgrade tests. | `internal/templates/templates.go`, `internal/templates/templates_test.go`, `.sdd/manifest.json`, `AGENTS.md`, `GEMINI.md`, `CLAUDE.md`, `internal/upgrade/upgrade_test.go` | `go test -v ./internal/upgrade ./internal/check -run "TestVersion|TestFreshKitHasNoErrors"` |
| 7 | Update living documents (`ROADMAP.md`, `TECH_STACK.md`, `SKILLS.md`, `README.md`) and run full verification suite (`make verify`). | Living docs | `make verify && ./bin/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/audit/bundle.go` | modify | Include validation.md in plan bundle; stage context in discovery bundle |
| `internal/audit/bundle_test.go` | modify | Tests for plan bundle validation.md and discovery context staging |
| `internal/audit/audit_test.go` | modify | Assert validation.md content inclusion in TestStagePlanBundle |
| `internal/templates/data/packs/go/review_standards.md` | create | Go code review standards snippet |
| `internal/templates/data/packs/java/review_standards.md` | create | Java code review standards snippet |
| `internal/templates/data/packs/python/review_standards.md` | create | Python code review standards snippet |
| `internal/templates/data/packs/typescript/review_standards.md` | create | TypeScript code review standards snippet |
| `internal/templates/data/packs/generic/review_standards.md` | create | Generic fallback code review standards snippet |
| `internal/templates/data/tasks/features/review.md` | modify | Language-pack-aware review rubric template |
| `internal/templates/data/tasks/bugfixes/review.md` | modify | Language-pack-aware review rubric template |
| `internal/templates/data/skills/codebase-analyst/SKILL.md` | modify | Coverage ledger and platform/runtime migration pillars |
| `internal/templates/data/skills/discovery-challenger/SKILL.md` | modify | Unanchored inventory-first protocol and generalized vectors |
| `internal/cli/prompt_discovery.go` | modify | Prompt discovery checklist updates for coverage ledger and runtime defaults |
| `internal/cli/prompt_discovery_test.go` | modify | Test prompt discovery enriched guidance |
| `internal/feature/feature.go` | modify | Pass language review standards to review.md template |
| `internal/feature/feature_test.go` | modify | Test language-aware review.md generation |
| `internal/scaffold/answers.go` | modify | Add ReviewStandards field to scaffold.Data |
| `internal/scaffold/plan.go` | modify | Render language review standards during kit init/upgrade |
| `internal/upgrade/upgrade.go` | modify | Inject language review standards during kit upgrade |
| `internal/check/paths.go` | modify | Implement citation-hygiene scanner |
| `internal/check/paths_test.go` | modify | Unit tests for citation-hygiene scanner |
| `internal/check/check.go` | modify | Wire citation-hygiene and coverage-ledger rules |
| `internal/check/check_test.go` | modify | Unit tests for citation-hygiene and coverage-ledger rules |
| `internal/templates/templates.go` | modify | Bump templates.Version to 2.4.0 |
| `internal/templates/templates_test.go` | modify | Assert templates.Version is 2.4.0 |
| `.sdd/manifest.json` | modify | Template version 2.4.0 |
| `AGENTS.md` | modify | Contract marker templates=2.4.0 |
| `GEMINI.md` | modify | Contract marker templates=2.4.0 |
| `CLAUDE.md` | modify | Contract marker templates=2.4.0 |
| `README.md` | modify | Document coverage ledger and language-aware review |
| `TECH_STACK.md` | modify | Record D-048 approval and verified facts |
| `SKILLS.md` | modify | Document Coverage Ledger and Unanchored Challenger protocols |
| `ROADMAP.md` | modify | Milestone 13 and T-056 status updates |
| `.sdd/state/T-056/**` | create | Task state store and token telemetry events |
| `tasks/features/T-056-brownfield-discovery-hardening/requirements.md` | modify | Task requirements document |
| `tasks/features/T-056-brownfield-discovery-hardening/plan.md` | modify | Task plan document |
| `tasks/features/T-056-brownfield-discovery-hardening/validation.md` | modify | Task validation document |
| `tasks/features/T-056-brownfield-discovery-hardening/review.md` | modify | Task review document |

## 6. Dependencies

None (100% standard library per `D-007`).

## 7. Living docs to update

- `ROADMAP.md`: update T-056 status (`Specifying` -> `Implementing` -> `Awaiting review` -> `Done`), update Milestone 13.
- `TECH_STACK.md`: record verified facts log entry and layout updates.
- `SKILLS.md`: add Section 13 on Coverage Ledger and Unanchored Adversarial Challenger standards.
- `README.md`: document Coverage Ledger and multi-language review templates.
- `AGENTS.md`, `GEMINI.md`, `CLAUDE.md`: contract parity and marker `templates=2.4.0`.

## 8. Risks, mitigations, rollback

- **Risk:** Existing projects with scratchpad references in old tasks failing `citation-hygiene`.
  - **Mitigation:** Only check tracked non-suppressed living documents and active task folders; provide clear line numbers and suppressions support via `.sdd/suppressions.json`.
- **Risk:** Projects without `.sdd/context/` breaking `StageDiscoveryBundle`.
  - **Mitigation:** Graceful conditional check: only copy context files if `.sdd/context/` exists and contains files.

### Rollback
- If implementation fails, revert task branch or uncommitted changes via `git checkout -- .` and `git clean -fd`.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")

- **F-001:** Resolve all review findings (Blockers, Warnings, Minors) across Review Triad before Gate 2 close.
- **F-002:** Public documentation, examples, and generated templates must never contain developer-specific local absolute paths; all paths must be repository-relative.
- **F-003:** Track and record session token consumption via `sdd record-tokens T-056`, document metrics in `validation.md`, and report at Gate 2.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-24   Phrase received: `APPROVED T-056`
