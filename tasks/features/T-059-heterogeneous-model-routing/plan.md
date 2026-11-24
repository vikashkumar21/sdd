---
status: active
---
# Plan — T-059 Heterogeneous role model routing, tier specifications, and provider-agnostic dispatch

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved 2026-11-24

## 1. Approach summary

Eliminate homogeneous frontier model over-consumption and optimize multi-agent token budgets by implementing heterogeneous model routing:
1. **Model Routing Data Structures & Defaults (`internal/role/role.go`):**
   - Extend `RoleInfo` with `ModelTier string json:"model_tier,omitempty"`, `ReasoningEffort string json:"reasoning_effort,omitempty"`, and `RecommendedFamily string json:"recommended_family,omitempty"`.
   - Define canonical tier constants (`TierFrontier`, `TierAdvanced`, `TierStandard`, `TierFast`) and reasoning effort constants (`ReasoningHigh`, `ReasoningMedium`, `ReasoningLow`, `ReasoningNone`).
   - Implement `ModelRouting` struct and validation helpers: `ValidateModelTier`, `ValidateReasoningEffort`.
   - Update `DefaultRoleConfig()` to seed all 11 canonical roles with calibrated model tiers and reasoning effort.
2. **Mechanical Consistency Validation (`internal/check/roles.go`):**
   - In `checkRoles()`, validate `model_tier` against allowed tiers (`roles-tier`).
   - Validate `reasoning_effort` against allowed efforts (`roles-reasoning`).
   - Validate `recommended_family` is non-empty when present (`roles-family`).
   - Add comprehensive unit test cases in `internal/check/roles_test.go`.
3. **CLI Handover Prompt Integration (`internal/cli/`):**
   - Update `ReviewPromptContext`, `PlanReviewPromptContext`, `ImplementPromptContext`, `DiscoveryPromptContext`, and `RepairPromptContext` to include `ModelRouting` (or routing details).
   - In `sdd prompt review`, `sdd prompt plan-review`, `sdd prompt implement`, `sdd prompt discovery`, and `sdd prompt repair`, format a `## Model Routing Guidance` markdown section.
   - For multi-lens review (`all`), include per-lens subagent model routing recommendations.
   - Add unit tests in `internal/cli/prompt_test.go` and `internal/cli/prompt_discovery_test.go`.
4. **Non-Destructive Manifest Migration (`internal/upgrade/upgrade.go`):**
   - Extend `sdd upgrade` roles processing: when upgrading `.sdd/roles.json`, backfill missing `model_tier`, `reasoning_effort`, and `recommended_family` on canonical roles without clobbering user overrides or custom roles.
   - Add unit test in `internal/upgrade/upgrade_test.go`.
5. **Starter Templates & Version Bump (`internal/templates/`):**
   - Update repository `.sdd/roles.json` and template `.sdd/roles.json`.
   - Bump `templates.Version` to `2.7.0` in `internal/templates/templates.go`, contracts, and `.sdd/manifest.json` per Hard Rule 16.
   - Assert kit parity and fresh kit generation clean.
6. **Task Coordinator Skill & Living Documents:**
   - Update `skills/task-coordinator/SKILL.md` with Section on Heterogeneous Model Routing.
   - Update `internal/templates/data/skills/coordinator/SKILL.md`, `internal/templates/data/skills/task-coordinator/SKILL.md`, and `internal/skill/export.go`.
   - Update `README.md`, `TECH_STACK.md`, `SKILLS.md`, and `ROADMAP.md`.
7. **Verification & Review Triad:**
   - Run `make verify`, `-count=2`, `-race`, outside audit runner `sdd task audit T-059`.
   - Dispatch independent Plan Reviewer (Phase A) and Review Triad (Phase C.2).

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-049 | Cognitive bias immunity: unanchored traceability ledger, ephemeral reviewer spawning, and heterogeneous model routing | Option A: Full architectural triad hardening in M14 (T-057 ledger, T-058 ephemeral loops, T-059 model routing) / Option B: Advisory prompt comments only / Option C: Provider-specific configs | Option A: Structurally eliminates diff-anchoring, justification bleed, and model homogeneity for cost-effective, rigorous verification. | Approved (2026-11-24) |

## 3. Design

### 1. Model Routing Schema (`internal/role/role.go`)
```go
type ModelTier string

const (
    TierFrontier ModelTier = "frontier"
    TierAdvanced ModelTier = "advanced"
    TierStandard ModelTier = "standard"
    TierFast     ModelTier = "fast"
)

type ReasoningEffort string

const (
    ReasoningHigh   ReasoningEffort = "high"
    ReasoningMedium ReasoningEffort = "medium"
    ReasoningLow    ReasoningEffort = "low"
    ReasoningNone   ReasoningEffort = "none"
)

type ModelRouting struct {
    Tier              string `json:"model_tier,omitempty"`
    ReasoningEffort   string `json:"reasoning_effort,omitempty"`
    RecommendedFamily string `json:"recommended_family,omitempty"`
}

type RoleInfo struct {
    Skill             string `json:"skill"`
    Description       string `json:"description"`
    ModelTier         string `json:"model_tier,omitempty"`
    ReasoningEffort   string `json:"reasoning_effort,omitempty"`
    RecommendedFamily string `json:"recommended_family,omitempty"`
}
```

### 2. Canonical Role Routing Allocation
| Role | Phase | Model Tier | Reasoning Effort | Recommended Family |
|---|---|---|---|---|
| `coordinator` | Governance / Phase D | `frontier` | `medium` | `claude-sonnet / gemini-pro / gpt-4o` |
| `architect` | Phase A (Plan) | `frontier` | `high` | `claude-opus / gemini-pro / o1` |
| `codebase-analyst` | Discovery | `frontier` | `high` | `claude-opus / gemini-pro / o1` |
| `discovery-challenger` | Discovery (Adversarial) | `frontier` | `high` | `claude-opus / gemini-pro / o1` |
| `plan-reviewer` | Pre-Gate 1 Audit | `advanced` | `high` | `claude-sonnet / gemini-pro / o3-mini` |
| `implementer` | Phase B (Code) | `advanced` | `medium` | `claude-sonnet / gemini-flash / gpt-4o` |
| `qa` | Operations / Verify | `standard` | `low` | `claude-haiku / gemini-flash / gpt-4o-mini` |
| `devops` | Operations / CI | `standard` | `low` | `claude-haiku / gemini-flash / gpt-4o-mini` |
| `spec-auditor` | Phase C.2 Triad | `standard` | `medium` | `claude-sonnet / gemini-flash / gpt-4o` |
| `appsec-specialist` | Phase C.2 Triad | `frontier` | `high` | `claude-opus / gemini-pro / o1` |
| `pattern-auditor` | Phase C.2 Triad | `standard` | `medium` | `claude-sonnet / gemini-flash / gpt-4o` |

### 3. Prompt Output Section
```markdown
## Model Routing Guidance
- **Role:** spec-auditor
- **Model Tier:** standard
- **Reasoning Effort:** medium
- **Recommended Family:** claude-sonnet / gemini-flash / gpt-4o
```

In JSON context structures (`ReviewPromptContext`, `PlanReviewPromptContext`, `ImplementPromptContext`, `DiscoveryPromptContext`, `RepairPromptContext`), the field is exposed as:
`ModelRouting *role.ModelRouting json:"model_routing,omitempty"` (supporting both `model_routing` and camelCase `modelRouting` via JSON parsing).

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Extend `RoleInfo` with model routing fields, define tier/reasoning enums, implement validation and resolution functions, and seed `DefaultRoleConfig()` in `internal/role/role.go`; write unit tests in `internal/role/role_test.go`. | `internal/role/role.go`, `internal/role/role_test.go` | `go test -v ./internal/role -run "TestModelRouting|TestValidateModelTier"` |
| 2 | Implement mechanical `roles-tier`, `roles-reasoning`, and `roles-family` validation rules in `internal/check/roles.go`; write unit tests in `internal/check/roles_test.go`. | `internal/check/roles.go`, `internal/check/roles_test.go` | `go test -v ./internal/check -run TestRolesCheck_ModelRouting` |
| 3 | Enrich `sdd prompt` handover generators (`review`, `plan-review`, `implement`, `discovery`, `repair`) in `internal/cli/` with model routing markdown section and JSON context fields; write unit tests in `internal/cli/`. | `internal/cli/prompt.go`, `internal/cli/prompt_discovery.go`, `internal/cli/prompt_repair.go`, `internal/cli/prompt_test.go`, `internal/cli/prompt_discovery_test.go`, `internal/cli/prompt_repair_test.go` | `go test -v ./internal/cli -run "TestPrompt.*ModelRouting"` |
| 4 | Extend `sdd upgrade` in `internal/upgrade/upgrade.go` to backfill missing model routing fields into existing `.sdd/roles.json` manifests without overwriting custom roles; write unit tests in `internal/upgrade/upgrade_test.go`. | `internal/upgrade/upgrade.go`, `internal/upgrade/upgrade_test.go` | `go test -v ./internal/upgrade -run TestUpgrade_RolesModelRouting` |
| 5 | Update repository `.sdd/roles.json`, bump `templates.Version` to `2.7.0` in `internal/templates/templates.go`, contracts (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`), and manifests per Hard Rule 16. | `CLAUDE.md`, `GEMINI.md`, `AGENTS.md`, `.sdd/manifest.json`, `.sdd/roles.json`, `internal/templates/templates.go`, `internal/templates/templates_test.go`, `internal/templates/data/base/contract.md.tmpl` | `go test -v ./internal/templates -run "TestVersion|TestContractTemplate"` |
| 6 | Document Heterogeneous Model Routing in `skills/task-coordinator/SKILL.md`, embedded coordinator templates, and living docs (`README.md`, `TECH_STACK.md`, `SKILLS.md`, `ROADMAP.md`). | `skills/task-coordinator/SKILL.md`, `internal/templates/data/skills/coordinator/SKILL.md`, `internal/templates/data/skills/task-coordinator/SKILL.md`, `internal/skill/export.go`, `README.md`, `TECH_STACK.md`, `SKILLS.md`, `ROADMAP.md` | `sdd check --strict` |
| 7 | Full verification suite (`make verify`, determinism `-count=2`, `-race`, outside audit runner `sdd task audit T-059`), Plan Review subagent audit, GATE 1 approval, implementation, and review triad sign-off. | All Section 5 files | Full test suite clean |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/role/role.go` | modify | Add `ModelTier`, `ReasoningEffort`, `RecommendedFamily` to `RoleInfo`, define tier enums, validation functions, and default values |
| `internal/role/role_test.go` | modify | Unit tests for model routing enums, validation, and defaults |
| `internal/check/roles.go` | modify | Mechanical validation rules: `roles-tier`, `roles-reasoning`, `roles-family` |
| `internal/check/roles_test.go` | modify | Unit tests for model routing validation in `roles.json` |
| `internal/cli/prompt.go` | modify | Enrich `sdd prompt review`, `plan-review`, and `implement` with model routing text and JSON |
| `internal/cli/prompt_discovery.go` | modify | Enrich `sdd prompt discovery` with model routing text and JSON |
| `internal/cli/prompt_repair.go` | modify | Enrich `sdd prompt repair` with model routing text and JSON |
| `internal/cli/prompt_test.go` | modify | Unit tests for prompt model routing output and JSON contexts |
| `internal/cli/prompt_discovery_test.go` | modify | Unit tests for discovery prompt model routing output and JSON |
| `internal/cli/prompt_repair_test.go` | modify | Unit tests for repair prompt model routing output and JSON |
| `internal/upgrade/upgrade.go` | modify | Non-destructive backfill of model routing metadata during `sdd upgrade` |
| `internal/upgrade/upgrade_test.go` | modify | Unit tests asserting non-destructive upgrade of legacy roles.json |
| `.sdd/roles.json` | modify | Update repository roles.json with model routing metadata |
| `skills/task-coordinator/SKILL.md` | modify | Document Heterogeneous Model Routing standards |
| `internal/templates/data/skills/coordinator/SKILL.md` | modify | Update embedded coordinator role skill with model routing |
| `internal/templates/data/skills/task-coordinator/SKILL.md` | modify | Update embedded task-coordinator skill with model routing |
| `internal/skill/export.go` | modify | Update canonical exported coordinator skill string constants |
| `internal/templates/data/base/contract.md.tmpl` | modify | Update contract version to 2.7.0 and reference model routing |
| `internal/templates/templates.go` | modify | Bump `templates.Version` to `2.7.0` |
| `internal/templates/templates_test.go` | modify | Assert `templates.Version == "2.7.0"` |
| `CLAUDE.md` | modify | Update contract version to `2.7.0` per Hard Rule 16 |
| `GEMINI.md` | modify | Update contract version to `2.7.0` per Hard Rule 16 |
| `AGENTS.md` | modify | Update contract version to `2.7.0` per Hard Rule 16 |
| `.sdd/manifest.json` | modify | Update templates version to `2.7.0` |
| `README.md` | modify | Document heterogeneous role model routing and tier allocation |
| `TECH_STACK.md` | modify | Verified facts log entry for T-059 |
| `SKILLS.md` | modify | Section 17: Heterogeneous Role Model Routing & Tier Specification Standard |
| `ROADMAP.md` | modify | Update T-059 status tracking |
| `.sdd/state/T-059/**` | create | Task state and token telemetry |
| `tasks/features/T-059-heterogeneous-model-routing/requirements.md` | modify | Task requirements |
| `tasks/features/T-059-heterogeneous-model-routing/plan.md` | modify | Task plan |
| `tasks/features/T-059-heterogeneous-model-routing/validation.md` | modify | Task validation |
| `tasks/features/T-059-heterogeneous-model-routing/review.md` | modify | Task review |

## 6. Dependencies

None (100% standard library per `D-007`).

## 7. Living docs to update

- `ROADMAP.md`: update T-059 status (`Specifying` -> `Awaiting plan approval` -> `Implementing` -> `Awaiting review` -> `Done`).
- `TECH_STACK.md`: record verified facts entry for T-059 and close Milestone 14.
- `SKILLS.md`: add Section 17 "Heterogeneous Role Model Routing & Tier Specification Standard".
- `README.md`: document heterogeneous model routing, tiers, and prompt JSON context.

## 8. Risks, mitigations, rollback

- **Risk:** Existing custom `.sdd/roles.json` files without routing fields fail `sdd check`.
  - **Mitigation:** Model routing fields (`model_tier`, `reasoning_effort`, `recommended_family`) are optional in `sdd check`. Only non-empty values are validated against the enum constraints.
- **Risk:** Legacy projects fail on `sdd upgrade` if roles already exist.
  - **Mitigation:** Upgrader merges missing routing fields into existing canonical roles without touching custom roles or overriding user customizations.

### Rollback
- Revert template version bump and restore `role.go`, `check/roles.go`, and `cli/prompt.go` changes.

## 9. Applicable human-feedback rules

- **F-001:** Resolve all review findings (Blockers, Warnings, Minors) across Review Triad before Gate 2 close.
- **F-002:** Zero personal absolute filesystem paths in generated code, templates, or documentation.
- **F-003:** Track and record session token consumption via `sdd record-tokens T-059`, document metrics in `validation.md`, and report at Gate 2.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-24   Phrase received: `APPROVED T-059`
