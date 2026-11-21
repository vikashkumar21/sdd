---
status: draft
---
# Plan — T-053 Adversarial multi-agent brownfield discovery ensemble

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Implemented

## 1. Approach summary
Implement an adversarial multi-agent brownfield discovery ensemble protocol that pairs the primary `codebase-analyst` with a dedicated, canonical `discovery-challenger` role skill. Operating under the identical stateless reviewer isolation model introduced in T-052, the Task Coordinator dispatches both roles during brownfield exploration. The analyst drafts initial living document proposals and architectural maps; the challenger is launched in a fresh, tabula-rasa context backed by a staged review bundle in `.sdd/review/<TASK_ID>/discovery/`. The challenger systematically interrogates the codebase against 7 comprehensive blind-spot vectors (hidden entry points, concurrency/lifecycle invariants, data boundaries/schema drift, dynamic dispatch/feature flags, orphan/dead code, implicit/ambient dependencies, and architectural contradictions). Findings are merged via a deterministic reconciliation checklist before Gate 1 approval.

The CLI command `sdd prompt discovery` is enhanced with an `--ensemble` flag that formats the multi-agent briefing, stages `.sdd/review/<TASK_ID>/discovery/`, and outputs structured JSON when combined with `--json`. Role configuration, `sdd check` validation, and non-destructive `sdd upgrade` migrations are updated to recognize `discovery-challenger`.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-045 | Adversarial discovery ensemble protocol and challenger role | Option A: Canonical `discovery-challenger` role in `.sdd/roles.json`, stateless bundle in `.sdd/review/<TASK_ID>/discovery/`, `sdd prompt discovery --ensemble`, 7 blind-spot vectors, and deterministic reconciliation protocol / Option B: Single-agent prompt enhancement without challenger role | Option A: Formally separates analysis from adversarial critique and guarantees stateless reviewer isolation, preventing discovery blind spots and false assumptions prior to Gate 1. | Proposed (Approved at GATE 1) |

## 3. Design

### 3.1 Role Skill: `discovery-challenger`
- File: `internal/templates/data/skills/discovery-challenger/SKILL.md`
- Conforms to `agentskills.io` specification with YAML frontmatter:
  - `name: discovery-challenger`
  - `description: "Adversarial codebase challenger: audits discovery findings, uncovers hidden entry points, implicit fallbacks, orphan code, and unreferenced architecture"`
- Rubric:
  - Vector 1: Hidden Entry Points & Triggers (CLI flags/subcommands, background daemon loops, cron/schedulers, queue workers, webhooks, and event consumers unreferenced in primary entrypoints)
  - Vector 2: Concurrency, Lifecycle & Shutdown Invariants (unbounded goroutines/threads, missing cancellation propagation, SIGINT/SIGTERM drain hooks, package-level mutable state, race conditions in init)
  - Vector 3: Data Boundary Invariants & Schema Drift (unhandled database errors, uncommitted/unrolled-back transactions, raw queries bypassing the ORM, schema migration mismatches, in-memory cache desync)
  - Vector 4: Dynamic Dispatch & Runtime Feature Flags (reflection-based routing, plugin systems, runtime feature flags, environment-conditional code paths invisible to static call graphs)
  - Vector 5: Orphan & Dead Code (dangling configurations, abandoned routes/endpoints, unexported/unreferenced symbols, unused dependencies)
  - Vector 6: Implicit External & Ambient Dependencies (undocumented env vars, ambient binaries/system tools assumed to exist on the host, undeclared cloud SDK defaults, unrecorded third-party APIs)
  - Vector 7: Architectural & Documentation Contradictions (direct discrepancies between existing README/architecture docs and actual runtime code behavior, stale comments, unverified assumptions)

### 3.2 Role Registry & Phase Mapping
- File: `internal/role/role.go`
  - `CanonicalRoleNames()`: 11 canonical roles (sorted alphabetically).
  - `PhaseMapping`: Add `Challenger string json:"challenger,omitempty"`.
  - `DefaultRoleConfig()`:
    - Register `"discovery-challenger"` in `Roles`.
    - Set `"phase-discovery": PhaseMapping{Analyst: "codebase-analyst", Challenger: "discovery-challenger", Planner: "architect"}`.
  - `LoadRoles`: Remains a pure loader reading `.sdd/roles.json` as authored; does not silently mutate or overwrite user configuration. Missing defaults are handled gracefully in consumers (`sdd upgrade` and `prompt discovery`).

### 3.3 Skill CLI & Export Alignment
- File: `internal/cli/skill.go`: Update `--roles` flag usage description to include `discovery-challenger`.
- File: `internal/cli/skill_test.go`: Update canonical role count assertion from 10 to 11.
- File: `internal/templates/templates_test.go`: Add `"discovery-challenger"` to `TestEmbeddedRoleSkills` table.
- File: `.sdd/roles.json` and `internal/scaffold/scaffold_test.go`: Update default kit role configuration to match 11 canonical roles.

### 3.4 Stateless Discovery Review Bundle
- File: `internal/audit/bundle.go`
  - Add `DiscoveryBundle` struct:
    ```go
    type DiscoveryBundle struct {
        TaskID    string    `json:"task_id"`
        TaskDir   string    `json:"task_dir"`
        BundleDir string    `json:"bundle_dir"`
        CreatedAt time.Time `json:"created_at"`
        Files     []string  `json:"files"`
    }
    ```
  - Implement `StageDiscoveryBundle(root, taskID, findingsContent, rubricContent string, now ...time.Time) (*DiscoveryBundle, error)`
  - Creates `.sdd/review/<TASK_ID>/discovery/` containing:
    - `findings.md`: The primary analyst's drafted findings, living doc proposals, and module maps.
    - `rubric.md`: The canonical `discovery-challenger` skill & 7 blind-spot vectors.
    - `manifest.json`: bundle metadata manifest recording task_id, bundle_dir, created_at, and files list matching PlanBundle/ReviewBundle.

### 3.5 CLI Flag & Prompt Generation
- File: `internal/cli/prompt_discovery.go`
  - Add flag: `ensembleFlag := fs.Bool("ensemble", false, "output multi-agent adversarial discovery ensemble brief")`
  - Update `DiscoveryPromptContext` with ensemble metadata (using `omitempty` for backward compatibility):
    ```go
    type DiscoveryPromptContext struct {
        // ... existing fields ...
        Ensemble                bool     `json:"ensemble,omitempty"`
        BundleDir               string   `json:"bundleDir,omitempty"`
        ChallengerRoleSkill     string   `json:"challengerRoleSkill,omitempty"`
        BlindSpotVectors        []string `json:"blindSpotVectors,omitempty"`
        ReconciliationChecklist []string `json:"reconciliationChecklist,omitempty"`
    }
    ```
  - When `--ensemble` is set:
    - Resolves both `codebase-analyst` and `discovery-challenger` skills via two-tier storage.
    - If task ID or discovery task is identified, calls `audit.StageDiscoveryBundle` (passing `env.now()` for deterministic clock injection) to stage `.sdd/review/<TASK_ID>/discovery/`.
    - Generates multi-agent prompt text including:
      - Phase Overview: Coordinator dispatch of primary analyst and tabula-rasa adversarial challenger.
      - Stateless Review Bundle reference (`.sdd/review/<TASK_ID>/discovery/`).
      - Primary Analyst brief (AST, UI/CSS, state/data, technical debt).
      - Challenger audit rubric covering the 7 blind-spot vectors.
      - 5-step Reconciliation Protocol to reconcile differences into `TECH_STACK.md`, `ROADMAP.md` Parking lot, and `SKILLS.md` before Gate 1 approval.
  - When `--ensemble` is false:
    - Retains 100% backward-compatible single-agent prompt output, omitting ensemble-only JSON fields.

### 3.6 Upgrade and Consistency Checking
- File: `internal/check/roles.go`:
  - Validate that `phase-discovery.challenger` (when non-empty) resolves to a registered role. Retains backward compatibility when `phase-discovery.challenger` is omitted in legacy manifests.
- File: `internal/upgrade/upgrade.go`:
  - Non-destructively migrate existing `.sdd/roles.json`:
    - Add `Roles["discovery-challenger"]` only if absent.
    - Set `Phases["phase-discovery"].Challenger = "discovery-challenger"` only if empty.
    - Emit `ActionUpdate` only when changes are made; preserve existing custom roles and descriptions; return/handle malformed JSON cleanly.

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Create canonical `discovery-challenger` skill template and update embedded skills test | `internal/templates/data/skills/discovery-challenger/SKILL.md`, `internal/templates/templates_test.go` | `go test ./internal/templates ./internal/skill` |
| 2a | Register `discovery-challenger` role and `PhaseMapping.Challenger` in `internal/role` | `internal/role/role.go`, `internal/role/role_test.go` | `go test ./internal/role` |
| 2b | Update skill CLI export help text, canonical role tests, and scaffold test fixture | `internal/cli/skill.go`, `internal/cli/skill_test.go`, `.sdd/roles.json`, `internal/scaffold/scaffold_test.go` | `go test ./internal/cli -run TestSkillExport` and `go test ./internal/scaffold` |
| 3 | Implement `StageDiscoveryBundle` in `internal/audit` | `internal/audit/bundle.go`, `internal/audit/bundle_test.go` | `go test ./internal/audit` |
| 4 | Update `internal/check` to validate `phase-discovery.challenger` mapping | `internal/check/roles.go`, `internal/check/roles_test.go` | `go test ./internal/check` |
| 5 | Update `internal/upgrade` to migrate `.sdd/roles.json` non-destructively | `internal/upgrade/upgrade.go`, `internal/upgrade/upgrade_test.go` | `go test ./internal/upgrade` |
| 6 | Implement `--ensemble` flag, bundle staging, context structures, prompt builder, and JSON export | `internal/cli/prompt_discovery.go`, `internal/cli/cli.go`, `internal/cli/commands.go`, `internal/cli/prompt_discovery_test.go` | `go test ./internal/cli` |
| 7 | End-to-end CLI integration verification, determinism, and path hygiene checks | All touched files | `make verify` and `sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/templates/data/skills/discovery-challenger/SKILL.md` | create | Canonical agentskills.io skill template for adversarial discovery challenger |
| `internal/templates/templates_test.go` | modify | Add discovery-challenger to embedded role skills verification table |
| `internal/role/role.go` | modify | Add `discovery-challenger` canonical role, PhaseMapping.Challenger field, default config |
| `internal/role/role_test.go` | modify | Unit tests for 11 canonical roles and phase-discovery challenger mapping |
| `internal/cli/skill.go` | modify | Update `--roles` flag usage description to list discovery-challenger |
| `internal/cli/skill_test.go` | modify | Update canonical role count assertion from 10 to 11 in skill export tests |
| `.sdd/roles.json` | modify | Update repository roles config with discovery-challenger and phase-discovery mapping |
| `internal/scaffold/scaffold_test.go` | modify | Update scaffold test assertion for discovery-challenger and 11 roles |
| `internal/audit/bundle.go` | modify | Implement DiscoveryBundle and StageDiscoveryBundle in .sdd/review/<TASK_ID>/discovery/ |
| `internal/audit/bundle_test.go` | create | Unit tests for StageDiscoveryBundle |
| `internal/check/roles.go` | modify | Check rule validating phase-discovery challenger role |
| `internal/check/roles_test.go` | modify | Test coverage for challenger phase validation |
| `internal/upgrade/upgrade.go` | modify | Non-destructive migration of `.sdd/roles.json` for discovery-challenger |
| `internal/upgrade/upgrade_test.go` | modify | Test upgrade adding discovery-challenger to existing roles.json |
| `internal/cli/prompt_discovery.go` | modify | Add `--ensemble` flag, bundle staging, context structs, prompt generator, JSON export |
| `internal/cli/prompt_discovery_test.go` | modify | Tests for `--ensemble` flag, bundle staging, JSON output, backward compatibility |
| `internal/cli/cli.go` | modify | Update command help text for `prompt discovery` |
| `internal/cli/commands.go` | modify | Update usage string for `prompt discovery` |
| `tasks/features/T-053-discovery-ensemble/requirements.md` | modify | Task requirements document |
| `tasks/features/T-053-discovery-ensemble/plan.md` | modify | Task plan document |
| `tasks/features/T-053-discovery-ensemble/validation.md` | modify | Task validation document |
| `tasks/features/T-053-discovery-ensemble/review.md` | modify | Task review document |
| `ROADMAP.md` | modify | Living doc update for T-053 status |
| `TECH_STACK.md` | modify | Living doc update for D-045 and verified facts |
| `SKILLS.md` | modify | Living doc update for ensemble discovery patterns |
| `README.md` | modify | Living doc update for CLI discovery usage |

## 6. Dependencies
Standard library only (`D-007`). Zero new third-party dependencies.
Note on template version: `templates.Version` remains `2.1.0` (as in `T-043`, role skills are internal embedded governance templates rather than kit-scaffolded files governed by manifest hashes).

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Update T-053 status and change log |
| `TECH_STACK.md` | Approve `D-045` and record verified fact for discovery ensemble |
| `SKILLS.md` | Document discovery challenger 7-vector audit methodology |
| `README.md` | Document `sdd prompt discovery --ensemble` CLI ergonomics |

## 8. Risks, mitigations, rollback
- **Risk:** Prompt token inflation when concatenating both role skills in terminal output.
  - *Mitigation:* Support `--json` for machine ingestion and ensure prompt markdown structure uses concise, high-signal sections with clean separation.
- **Risk:** Breaking existing `.sdd/roles.json` in user projects initialized before T-053.
  - *Mitigation:* `LoadRoles` parses without error when `challenger` is absent; `sdd upgrade` adds the role non-destructively; `sdd check` handles legacy manifests gracefully.
- **Rollback:** Discard task branch or restore previous files via Git (`git restore`).

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- `F-001` (Resolve all review findings): Primary worker agent must resolve all Blockers, Warnings, and Minors reported during the Review Triad audit before Gate 2 close.
- `F-002` (Path hygiene): Public documentation, generated prompt templates, and test cases must use relative paths or repository-relative links; zero developer-specific absolute paths (`/Users/...`, `/home/...`).

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-21   Phrase received: `APPROVED T-053`
