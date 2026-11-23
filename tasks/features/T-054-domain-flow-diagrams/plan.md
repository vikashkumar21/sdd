---
status: draft
---
# Plan — T-054 Deep domain flow documentation and visual Mermaid diagrams

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-22) | Done

## 1. Approach summary

Provide native scaffolding, standard templates, and mechanical validation for visual architecture and domain flow documentation:
1. Implement package `internal/docflow` containing generator logic, file planning, atomic writes, and manifest recording for `docs/architecture/` and `docs/business_flows/`.
2. Author embedded templates in `internal/templates/data/docs/` (`architecture/overview.md.tmpl`, `architecture/data_model.md.tmpl`, `business_flows/overview.md.tmpl`, `business_flows/flow.md.tmpl`) with clean Mermaid syntax (`flowchart`, `erDiagram`, `sequenceDiagram`, `stateDiagram-v2`) and lifecycle frontmatter.
3. Wire CLI commands `sdd doc init` and `sdd doc flow <name>` into `internal/cli/cli.go` and `internal/cli/doc.go`.
4. Implement a deterministic Mermaid code block validator `CheckMermaid(text string) []MermaidFinding` in `internal/markdown/mermaid.go` and integrate as rule `mermaid-syntax` in `internal/check/check.go`.
5. Enrich `sdd prompt discovery` in `internal/cli/discovery.go` to guide codebase analysis towards visual domain flow documentation.
6. Bump `templates.Version` from `2.2.0` to `2.3.0` in `internal/templates/templates.go`, update contract block markers across `AGENTS.md`, `GEMINI.md`, `CLAUDE.md`, and `contract.md.tmpl`, and verify `sdd upgrade` migrations.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-046 | Domain flow scaffolding and Mermaid diagram standards | Option A: `docs/architecture/` and `docs/business_flows/` generation with automated Mermaid topology, ER, and sequence diagrams / Option B: Plain markdown files without diagrams | Option A: Native Mermaid diagrams render seamlessly in GitHub/GitLab/IDEs, provide immediate visual comprehension, and eliminate external binary rendering dependencies. | Approved |

## 3. Design

### CLI Interface
```bash
sdd doc init [--dir <path>] [--force] [--dry-run] [--json]
sdd doc flow <name> [--title <title>] [--dir <path>] [--force] [--dry-run] [--json]
```

### Core Data Structures (`internal/docflow/docflow.go`)
```go
package docflow

type InitOptions struct {
    Dir      string
    Force    bool
    DryRun   bool
    JSON     bool
    Now      time.Time
}

type FlowOptions struct {
    Dir      string
    Name     string
    Title    string
    Force    bool
    DryRun   bool
    JSON     bool
    Now      time.Time
}

type PlanEntry struct {
    RelPath string
    Action  string // create, skip, replace
    Reason  string
}

type Result struct {
    Created []string `json:"created,omitempty"`
    Skipped []string `json:"skipped,omitempty"`
    Backups []string `json:"backups,omitempty"`
}
```

### Mermaid Validator (`internal/markdown/mermaid.go`)
```go
package markdown

type MermaidFinding struct {
    Line    int
    Rule    string
    Message string
}

func CheckMermaid(text string) []MermaidFinding
```
Validates:
1. First non-comment line inside ` ```mermaid ` must declare a valid diagram type: `flowchart`, `graph`, `sequenceDiagram`, `erDiagram`, `classDiagram`, `stateDiagram-v2`, `stateDiagram`, `gantt`, `pie`, `gitGraph`, `c4Context`, `c4Container`, `c4Component`, `journey`, `quadrantChart`, `xychart-beta`, `block-beta`.
2. Balanced brackets `[]`, `()`, and `{}` across diagram body lines.

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Create Mermaid parser & validator in `internal/markdown/mermaid.go` with unit tests covering valid/invalid diagram headers, balanced/unbalanced brackets, complex shapes (`[("...")]`, `(["..."])`), quoted labels with punctuation, and comments (`%% ...`). | `internal/markdown/mermaid.go`, `internal/markdown/mermaid_test.go` | `go test -v ./internal/markdown -run TestMermaid` |
| 2 | Integrate `checkMermaid` into `internal/check/check.go` under rule `mermaid-syntax`. Update `scanDirs` in `internal/check/check.go` to include `"docs"` so all documents under `docs/` are inspected for `mermaid-syntax`, code fences, tables, and lifecycle metadata. Author regression tests in `internal/check/check_test.go`. | `internal/check/check.go`, `internal/check/check_test.go` | `go test -v ./internal/check -run TestCheckMermaid` |
| 3 | Author embedded document templates in `internal/templates/data/docs/architecture/` and `internal/templates/data/docs/business_flows/`. | `internal/templates/data/docs/architecture/**`, `internal/templates/data/docs/business_flows/**` | `go test -v ./internal/templates` |
| 4 | Implement `internal/docflow` package for scaffolding `docs/architecture/` and `docs/business_flows/` with atomic writes, backup generation, and manifest updates. Author unit tests in `internal/docflow/docflow_test.go`. | `internal/docflow/docflow.go`, `internal/docflow/docflow_test.go` | `go test -v ./internal/docflow` |
| 5 | Wire `sdd doc` CLI commands (`init` and `flow`) in `internal/cli/doc.go` and `internal/cli/cli.go`. Author CLI dispatch tests in `internal/cli/doc_test.go`. | `internal/cli/doc.go`, `internal/cli/doc_test.go`, `internal/cli/cli.go`, `internal/cli/cli_test.go` | `go test -v ./internal/cli -run TestDoc` |
| 6 | Enrich `sdd prompt discovery` in `internal/cli/discovery.go` with domain flow prompts and checklist items. Update tests in `internal/cli/discovery_test.go`. | `internal/cli/discovery.go`, `internal/cli/discovery_test.go` | `go test -v ./internal/cli -run TestDiscovery` |
| 7 | Bump `templates.Version` to `2.3.0` in `internal/templates/templates.go`, update test fixtures, contract markers (`templates=2.3.0`) in `internal/templates/data/base/contract.md.tmpl`, `AGENTS.md`, `GEMINI.md`, `CLAUDE.md`, and `.sdd/manifest.json`. Verify `internal/upgrade` tests. | `internal/templates/templates.go`, `internal/templates/templates_test.go`, `.sdd/manifest.json`, `internal/templates/data/base/contract.md.tmpl`, `AGENTS.md`, `GEMINI.md`, `CLAUDE.md`, `internal/upgrade/upgrade_test.go` | `go test -v ./internal/templates ./internal/upgrade ./internal/check -run "TestVersion|TestFreshKitHasNoErrors"` |
| 8 | Scaffold repository documentation for `sdd` itself (`docs/architecture/overview.md`, `docs/architecture/data_model.md`, `docs/business_flows/overview.md`, `docs/business_flows/sample_flow.md`) via `sdd doc init`, and run `sdd doc flow task_lifecycle` to generate `docs/business_flows/task_lifecycle.md`. Verify strict check. | `docs/architecture/**`, `docs/business_flows/**` | `./bin/sdd check --strict` |
| 9 | Update living docs (`ROADMAP.md`, `TECH_STACK.md`, `SKILLS.md`, `README.md`) and run full verification suite (`make verify`). | Living docs | `make verify && ./bin/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/markdown/mermaid.go` | create | Deterministic Mermaid code block validator |
| `internal/markdown/mermaid_test.go` | create | Unit tests for Mermaid syntax and bracket balancing |
| `internal/check/check.go` | modify | Integrate mermaid-syntax check rule and add docs to scanDirs |
| `internal/check/check_test.go` | modify | Test mermaid-syntax check rule |
| `internal/templates/data/docs/architecture/overview.md.tmpl` | create | Embedded architecture overview template |
| `internal/templates/data/docs/architecture/data_model.md.tmpl` | create | Embedded data model ER diagram template |
| `internal/templates/data/docs/business_flows/overview.md.tmpl` | create | Embedded business flows catalog template |
| `internal/templates/data/docs/business_flows/flow.md.tmpl` | create | Embedded domain flow sequence & state template |
| `internal/templates/templates.go` | modify | Bump templates.Version to 2.3.0 |
| `internal/templates/templates_test.go` | modify | Update templates.Version assertion |
| `internal/templates/data/base/contract.md.tmpl` | modify | Update marker to templates=2.3.0 |
| `internal/docflow/docflow.go` | create | Core engine for doc scaffolding, plan generation, and safe writes |
| `internal/docflow/docflow_test.go` | create | Unit tests for docflow generator and flags |
| `internal/cli/doc.go` | create | CLI handlers for sdd doc init and sdd doc flow |
| `internal/cli/doc_test.go` | create | Tests for sdd doc CLI subcommands |
| `internal/cli/cli.go` | modify | Wire doc command into CLI router and usage |
| `internal/cli/cli_test.go` | modify | Test doc command dispatch |
| `internal/cli/prompt_discovery.go` | modify | Enrich discovery prompt with visual domain flow targets |
| `internal/cli/prompt_discovery_test.go` | modify | Test discovery prompt enrichment |
| `.sdd/state/T-054/**` | create | Deterministic task state store and token telemetry events |
| `internal/upgrade/upgrade_test.go` | modify | Verify upgrade to template version 2.3.0 |
| `.sdd/manifest.json` | modify | Update template version to 2.3.0 |
| `AGENTS.md` | modify | Contract marker templates=2.3.0 |
| `GEMINI.md` | modify | Contract marker templates=2.3.0 |
| `CLAUDE.md` | modify | Contract marker templates=2.3.0 |
| `docs/architecture/overview.md` | create | SDD system architecture overview |
| `docs/architecture/data_model.md` | create | SDD data model ER diagram |
| `docs/business_flows/overview.md` | create | SDD business flows catalog |
| `docs/business_flows/sample_flow.md` | create | Default starter domain flow created by sdd doc init |
| `docs/business_flows/task_lifecycle.md` | create | SDD task lifecycle flow created by sdd doc flow task_lifecycle |
| `README.md` | modify | Document sdd doc subcommands and Mermaid standards |
| `TECH_STACK.md` | modify | Approve D-046 with Option A and update verified facts |
| `SKILLS.md` | modify | Add Mermaid Diagram and Domain Flow Standards section |
| `ROADMAP.md` | modify | Update T-054 status and milestones |
| `tasks/features/T-054-domain-flow-diagrams/requirements.md` | modify | Task requirements document |
| `tasks/features/T-054-domain-flow-diagrams/plan.md` | modify | Task plan document |
| `tasks/features/T-054-domain-flow-diagrams/validation.md` | modify | Task validation document |
| `tasks/features/T-054-domain-flow-diagrams/review.md` | modify | Task review document |

## 6. Dependencies

None (100% standard library per `D-007`).

## 7. Living docs to update

- `ROADMAP.md`: update T-054 status (`Specifying` -> `Implementing` -> `Awaiting review` -> `Done`), update Milestone 12 progress.
- `TECH_STACK.md`: approve `D-046` (Option A), record architecture additions in §3/§4, update verified facts log.
- `SKILLS.md`: add Section 12 on Mermaid Diagram Standards and Domain Flow Architecture.
- `README.md`: document `sdd doc init`, `sdd doc flow`, and visual diagram conventions.
- `AGENTS.md`, `GEMINI.md`, `CLAUDE.md`: contract parity and marker `templates=2.3.0`.

## 8. Risks, mitigations, rollback

- **Risk:** Existing repositories with custom `docs/` directories could be accidentally overwritten by `sdd doc init`.
  - **Mitigation:** Fail-safe non-destructive defaults: `docflow` checks if target files exist; without `--force`, existing files are skipped untouched. Overwrites require `--force` and automatically back up existing files to `.sdd/backup/docs/`.
- **Risk:** Malformed Mermaid diagrams causing markdown rendering errors in GitHub or IDEs.
  - **Mitigation:** `sdd check` includes `mermaid-syntax` mechanical validation verifying diagram declarations and bracket balancing across all repository markdown files.

### Rollback
- If implementation fails or human rejects, remove uncommitted changes via `git checkout -- .` and `git clean -fd`. Task branch isolation ensures `main` branch remains untouched.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")

- **F-001:** Resolve all review findings (Blockers, Warnings, Minors) across Review Triad before Gate 2 close.
- **F-002:** Public documentation, examples, and generated templates must never contain developer-specific local absolute paths; all paths must be repository-relative.
- **F-003:** Track and record session token consumption via `sdd record-tokens T-054`, document metrics in `validation.md`, and report at Gate 2.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-22   Phrase received: `APPROVED T-054`
