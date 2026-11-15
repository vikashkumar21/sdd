---
status: completed
task_id: T-045
last_reviewed: 2026-11-15
---
# Plan — T-045 Turnkey sdd prompt discovery and brownfield synthesis protocol

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved

## 1. Approach summary
Implement `sdd prompt discovery [T-xxx] [--dir <dir>] [--json]` in `internal/cli/prompt_discovery.go` and wire it into `internal/cli/commands.go:runPrompt` and `internal/cli/cli.go`. The command loads project detection metadata via `detect.Detect` (detected languages, framework heuristics, manifests), staged context via `context.ReadManifest`, resolves the canonical `codebase-analyst` role skill via two-tier storage (`role.ResolveSkill`), optionally links to a focus task (`[T-xxx]`), and renders the turnkey discovery prompt. The prompt embeds the 4-pillar progressive extraction checklist and living document synthesis instructions (`TECH_STACK.md`, `SKILLS.md`, `ROADMAP.md`). Supports human-readable markdown text and machine-readable `--json` output (`DiscoveryPromptContext`).

## 2. Decisions (new or affected)
None. Conforms to approved decisions D-002, D-007, D-024, D-030, D-036, and D-039.

## 3. Design

### Context Data Structure
```go
// DiscoveryPromptContext represents the structured context bundle emitted with --json.
type DiscoveryPromptContext struct {
	TaskID            string   `json:"taskId,omitempty"`
	TaskDir           string   `json:"taskDir,omitempty"`
	Typology          string   `json:"typology,omitempty"`
	ProjectRoot       string   `json:"projectRoot"`
	RoleSkill         string   `json:"roleSkill"`
	DetectedLanguages []string `json:"detectedLanguages"`
	LivingDocs        []string `json:"livingDocs"`
	Checklist         []string `json:"checklist"`
	Requirements      string   `json:"requirements,omitempty"`
	Plan              string   `json:"plan,omitempty"`
	Prompt            string   `json:"prompt"`
}
```

### CLI Command Dispatch Flow
1. Parse flags `--dir`, `--json`, and optional positional task ID `[T-xxx]` or flag `--task`.
2. Normalize project root to absolute path via `filepath.Abs`.
3. If task ID is provided:
   - Validate format (`T-xxx`).
   - Locate task directory via `findTaskDirectory`.
   - Read `requirements.md` and `plan.md` if available.
4. If task ID is omitted:
   - Search `ROADMAP.md` for the first open task mentioning discovery/explore, or run project-wide discovery.
5. Scan project metadata using `detect.Detect(root)` for detected languages and manifests, and inspect staged context via `context.ReadManifest(root)`.
6. Resolve `codebase-analyst` role skill using `role.ResolveSkill(root, "codebase-analyst", nil)`.
7. Assemble prompt text with 4-pillar checklist and brownfield living doc synthesis instructions.
8. Output either JSON (`DiscoveryPromptContext`) or formatted markdown text to `env.Out`.

### 4-Pillar Checklist & Living Document Synthesis Protocol
- **Pillar 1: AST & Module Map:** Directory topology, entry points, package manifests, build scripts -> `TECH_STACK.md` §1 & §4.
- **Pillar 2: Design System & UI Patterns:** Fonts, CSS frameworks, color themes, component hierarchy -> `SKILLS.md` §1.
- **Pillar 3: Data & State Modeling:** Schemas, ORM models, state stores, caching, APIs -> `TECH_STACK.md` §3 & §8.
- **Pillar 4: Anti-Patterns & Technical Debt:** Fragile dependencies, unhandled error patterns, security hazards -> `SKILLS.md` §7 (Pitfalls) & `ROADMAP.md` §7 (Parking lot).

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement `internal/cli/prompt_discovery.go` with `runPromptDiscovery`, `DiscoveryPromptContext`, `buildDiscoveryPrompt`, and extraction helpers, and scaffold `internal/cli/prompt_discovery_test.go` with initial dispatch and rendering test | `internal/cli/prompt_discovery.go`, `internal/cli/prompt_discovery_test.go` | `go test -v ./internal/cli -run TestPromptDiscovery` |
| 2 | Wire `discovery` subcommand in `internal/cli/commands.go:runPrompt` and update prompt usage messages in `internal/cli/cli.go` and `internal/cli/commands.go` | `internal/cli/commands.go`, `internal/cli/cli.go` | `go test -v ./internal/cli -run "TestPromptDiscovery\|TestPromptDispatch"` |
| 3 | Expand unit test suite in `internal/cli/prompt_discovery_test.go` covering text prompt, `--json` context, task-focus, two-tier overrides, and error cases | `internal/cli/prompt_discovery_test.go` | `go test -v ./internal/cli -run TestPromptDiscovery` |
| 4 | Update `README.md`, `ROADMAP.md` (status, Section 8 active task, Section 9 change log), `TECH_STACK.md`, and verify full test suite | `README.md`, `ROADMAP.md`, `TECH_STACK.md` | `make verify && make check && ./bin/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `README.md` | modify | Document `sdd prompt discovery` command |
| `ROADMAP.md` | modify | Update T-045 lifecycle status, Section 8 active task, and Section 9 change log |
| `TECH_STACK.md` | modify | Record verified fact for sdd prompt discovery |
| `internal/cli/cli.go` | modify | Update sdd prompt usage message in CLI help string |
| `internal/cli/commands.go` | modify | Dispatch discovery subcommand in runPrompt |
| `internal/cli/prompt_discovery.go` | create | Implement runPromptDiscovery and DiscoveryPromptContext |
| `internal/cli/prompt_discovery_test.go` | create | Unit tests for discovery prompt generation and JSON bundling |
| `tasks/features/T-045-prompt-discovery-brownfield/plan.md` | modify | Implementation plan |
| `tasks/features/T-045-prompt-discovery-brownfield/requirements.md` | modify | Requirements specification |
| `tasks/features/T-045-prompt-discovery-brownfield/review.md` | modify | Review Triad audit sign-off |
| `tasks/features/T-045-prompt-discovery-brownfield/validation.md` | modify | Traceability matrix and verification evidence |

## 6. Dependencies
None. Pure Go standard library per Decision D-007.

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Update T-045 status (`Specifying` -> `Awaiting plan approval` -> `Implementing` -> `Awaiting review` -> `Done`), refresh Section 8 active task to T-045, and add T-044 to Section 9 change log |
| `TECH_STACK.md` | Record verified fact for turnkey sdd prompt discovery in Section 7 |
| `README.md` | Document `sdd prompt discovery` usage and options |

## 8. Risks, mitigations, rollback
- **Risk:** Stale or missing `codebase-analyst` role skill if templates are broken.  
  **Mitigation:** `role.ResolveSkill` automatically uses embedded fallback templates, and unit tests verify fallback resolution.
- **Rollback:** `git checkout` to restore `internal/cli/commands.go`, `internal/cli/cli.go`, and remove created files.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- **Rule F-001 (Resolve all review findings):** Phase C.2 Review Triad audit will be conducted and all reported findings (Blockers, Warnings, Minors) will be resolved before Phase D close.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-15   Phrase received: `APPROVED T-045`
