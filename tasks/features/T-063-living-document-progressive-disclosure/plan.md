---
status: completed
---
# Plan — T-063 Living document progressive disclosure and Rule 1 micro-index

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-25)
Roadmap ref: `ROADMAP.md` T-063

## 1. Approach summary
Introduce an `internal/passport` package with deterministic clock-injected generation (`Generate(root string, now time.Time) (*Passport, error)`) that creates a compact, high-density project passport ($\le 500$ tokens / $\le 2,000$ bytes for rendered Markdown). The passport synthesizes project mission, stack (polyglot-aware), active milestone, active tasks (or next task if none active), active feedback rules (parsed from `### Register` table), and canonical living document pointers. Expose this via `sdd passport` and `sdd status --compact`. Evolve Section 1 (Rule 1) in agent contracts (`AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `contract.md.tmpl`) to authorize the Progressive Disclosure Protocol citing canonical section numbers (`TECH_STACK.md` Decision register §6, verified facts §7; `SKILLS.md` §1 Language standards, §8 Feedback rules) and aligning active task docs with `review.md`, where agents orient using `sdd passport` at startup and load full living docs on-demand, while strictly preserving the startup declaration and conflict-halt requirement. Implement mechanical validation in `sdd check` (`passport-consistency`), update canonical skills, bump `templates.Version` to `2.11.0`, and verify kit generation per Hard Rule 16.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-050 | Token consumption mitigation, context hygiene, and provider-aligned caching architecture | A) Tier routing, lens bundles, prefix caching, progressive disclosure, native transcripts B) Ad-hoc token prompts | Option A | Approved (2026-11-25) |

## 3. Design

### Data Structures (`internal/passport/passport.go`)
```go
package passport

import (
	"time"
	"github.com/vikashkumar21/sdd/internal/status"
)

type Passport struct {
	ProjectName     string                `json:"project_name"`
	Mission         string                `json:"mission"`
	Stack           string                `json:"stack"`
	ActiveMilestone *status.MilestoneInfo `json:"active_milestone,omitempty"`
	ActiveTasks     []status.TaskInfo     `json:"active_tasks,omitempty"`
	NextTask        *status.TaskInfo      `json:"next_task,omitempty"`
	ActiveRules     []RuleSummary         `json:"active_rules,omitempty"`
	LivingDocs      []DocPointer          `json:"living_docs"`
	GeneratedAt     time.Time             `json:"generated_at"`
	EstimatedTokens int                   `json:"estimated_tokens"`
	ByteSize        int                   `json:"byte_size"`
}

type RuleSummary struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
}

type DocPointer struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description"`
	WhenToRead  string `json:"when_to_read"`
}

const (
	MaxPassportTokens = 500
	MaxPassportBytes  = 2000
)

// Generate builds a deterministic Passport from the repository at root.
// now is injected per Hard Rule 8 for deterministic timestamping; if now.IsZero(),
// it normalizes to time.Unix(0, 0).UTC(). It NEVER calls time.Now().
func Generate(root string, now time.Time) (*Passport, error)
func RenderMarkdown(p *Passport) string
func RenderJSON(p *Passport) ([]byte, error)
func EstimateTokens(s string) int
func Validate(root string, p *Passport, rendered string) []string
```

### Extraction, Parsing Semantics & Non-Mutating Purity (`internal/passport/passport.go`)
1. **Clock Injection & Determinism (Hard Rule 8):** `Generate` takes `now time.Time`. If `now.IsZero()`, it normalizes to `time.Unix(0, 0).UTC()`. Zero calls to `time.Now()` inside `Generate`. CLI callers (`passport.go`, `status.go`) pass `env.now().UTC()`, while tests and `check/passport.go` pass a fixed timestamp.
2. **Active Tasks Handling:** Reads `status.Scan(root)`. If `ROADMAP.md` is missing, `status.Scan` returns an error, which `Generate` returns as a required-file error (exit `1` in CLI). If `len(ps.ActiveTasks) > 0`, all active tasks are recorded in `p.ActiveTasks`. If 0 tasks are active, `p.NextTask` is populated from `ps.NextTask`, and Markdown renders `- Active Task: none (next: <NextTask>)`.
3. **Mission Extraction:** Scans `MISSION.md` (bounded to first 16KB). Skips YAML frontmatter, headings, `>` blockquotes (such as anthems), and `Status:` lines, extracting the first substantive sentence under `## 1. One-liner` (or `# Mission`). Truncates cleanly on UTF-8 rune boundaries to $\le 200$ characters. Fallback: repository root directory name.
4. **Stack Extraction (Polyglot):** Scans `TECH_STACK.md` (bounded to 32KB). Parses the `## 1. Stack at a glance` markdown table row matching `Language(s)` or `Language` and extracts the Choice cell. If absent or empty, falls back to `"Unspecified"`.
5. **Feedback Rules Extraction:** Scans `SKILLS.md` (bounded to 64KB) under heading `(?i)^##\s+.*Human feedback rules`. Parses the `### Register` markdown table rows (`| ID | Date | Rule | Applies to | Source | Verified by | Status |`), skipping header, separator, rows with empty `ID`, and rows where any cell contains `(none yet)`. Trims emphasis (`*`, `` ` ``) and filters out any rule where status case-insensitively begins with `"superseded"` (`strings.HasPrefix(strings.ToLower(status), "superseded")`). Truncates each rule summary on UTF-8 rune boundaries to $\le 100$ characters.
6. **Canonical Living Document Pointers:** Fixed deterministic list of candidate pointers:
   - `MISSION.md`: Core principles, non-goals, and boundary invariants. (Inspect when evaluating architectural scope).
   - `TECH_STACK.md`: Full stack layout, Decision register (§6), verified-facts log (§7). (Inspect when adding dependencies or verifying APIs).
   - `ROADMAP.md`: Milestone progress, task dependency graph, and parking lot. (Inspect when scheduling next work).
   - `SKILLS.md`: Language standards (§1), playbooks, and full feedback registry (§8). (Inspect when writing code, tests, or reviewing).
   - `README.md`: Public CLI usage, quickstart, and feature highlights.
   Only pointers for files that actually exist on disk are included in `p.LivingDocs`.
7. **Deterministic Budget Enforcement, Non-Mutating Render & CRLF Invariance:**
   - All input content is normalized from CRLF (`\r\n`) to LF (`\n`).
   - `RenderMarkdown(p)` treats `p` as read-only, rendering copies/views without mutating `p.ActiveTasks` or `p.ActiveRules`.
   - If rendered Markdown exceeds `MaxPassportBytes` (2,000 bytes) or `MaxPassportTokens` (500 tokens), `RenderMarkdown` applies deterministic compaction priority:
     a) Active tasks: show first 3 tasks + `... +N more`.
     b) Rules: shorten rule summaries to 60 characters on rune boundaries, then cap rule count to top 5 + `... +N more`.
     c) Hard ceiling: if rendered Markdown still exceeds `len(rendered) > MaxPassportBytes` or `EstimateTokens(rendered) > MaxPassportTokens` (recalling that 1,854 ASCII runes is ~501 tokens), truncates on rune boundaries to satisfy both bounds simultaneously.
   - `Generate` computes `rendered := RenderMarkdown(p)`, and populates `p.ByteSize = len(rendered)` and `p.EstimatedTokens = EstimateTokens(rendered)`.
   - `EstimateTokens` implements the standard `(runes * 10) / 37` heuristic with the exact `tokens == 0 && runes > 0 => 1` guard matching `internal/cli/prompt_cache.go`.
   - `passport.Validate(root, p, rendered)` validates the supplied rendered string against `MaxPassportBytes` / `MaxPassportTokens` and verifies file existence of all declared `DocPointer` entries using `os.OpenRoot` and `filepath.IsLocal` path confinement.

### CLI Command Specifications (`internal/cli/passport.go`, `internal/cli/status.go`, `internal/cli/cli.go`)
- **`sdd passport [--json] [--dir <dir>]`:**
  - `args`: no positional arguments allowed. Any positional argument yields exit code `2`.
  - Default: prints `RenderMarkdown(p)` to `env.Out`. Exit code `0`.
  - `--json`: prints formatted JSON to `env.Out`. Exit code `0`.
  - On error (e.g. missing `ROADMAP.md`): prints error to `env.Err` and exits `1`.
- **`sdd status [--json] [--compact] [--dir <dir>]`:**
  - `--compact`: Renders the compact passport micro-index.
  - `--compact --json`: Renders structured JSON passport.
  - Exit code `0` on success, `1` on error, `2` on usage error.

### Mechanical Checker (`internal/check/passport.go`)
- Rule: `passport-consistency` registered in `internal/check/check.go`.
- Audit checks:
  1. If `ROADMAP.md` is absent in `root` (checked via `c.load("ROADMAP.md")`), `passport-consistency` returns early without emitting an error (preventing spurious failures in partial/empty dir tests).
  2. Runs `passport.Generate(root, time.Unix(0, 0).UTC())`. If generation fails, emits `Error` (`passport-generation`).
  3. Validates budget: calls `passport.Validate(root, p, rendered)`. If rendered string exceeds bounds, emits `Warn` (`passport-budget`).
  4. Validates active task alignment: checks all active/blocked task IDs from `ROADMAP.md` against `p.ActiveTasks` (full slice before display truncation). Emits `Error` (`passport-task-mismatch`) if out of sync.
  5. Validates document pointers: verifies each file declared in `p.LivingDocs` exists on disk using `os.OpenRoot` and `filepath.IsLocal` confinement. Emits `Error` (`passport-doc-missing`) if missing.

### Contract Rule 1 Evolution (`contract.md.tmpl`, `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`)
Section 1 is evolved to the Progressive Disclosure Protocol, strictly preserving the startup declaration and conflict-halt instructions, and aligning active task docs with `review.md`:
```markdown
## 1. Read first — every session, in this order

1. Project Passport / Micro-Index (`sdd passport` or `sdd status --compact`) — orient on project mission, stack, active milestone, active tasks, and active feedback rules (<= 500 tokens).
2. If a task is active: `tasks/<typology>/T-xxx-<slug>/requirements.md`, `plan.md`, `validation.md`, `review.md`.
3. Progressive Disclosure (on-demand): Inspect specific sections of living documents only when relevant to the active decision or step:
   - `TECH_STACK.md` (Decision register §6, verified facts §7) when evaluating architectures or dependencies.
   - `SKILLS.md` (§1 Language standards, §8 Feedback rules) when writing code, tests, or reviewing.
   - `ROADMAP.md` when scheduling next tasks or updating milestone progress.
   - `README.md` when checking public CLI usage and quickstart.
   - `MISSION.md` when checking project boundaries and non-goals.

Start every session by stating: the active task (or "none"), the files/sections you read, and its current status in `ROADMAP.md`. If two files contradict each other, STOP and report the conflict. Do not pick a winner.
```

### Prompt Templates Alignment (`prompt.md.tmpl`, `prompt_compact.md.tmpl`)
- `base/prompt.md.tmpl`: Preserves `## 1. Step 0 - read the project` heading, updates description to reference progressive disclosure via `sdd passport`, and guarantees the invariant static prefix remains $\ge 4,096$ bytes before `<!-- sdd:cache-boundary:end-prefix -->`.
- `base/prompt_compact.md.tmpl`: Preserves $\le 300$-word ceiling and $\ge 60\%$ reduction vs standard prompt, referencing `sdd passport`. Progressive disclosure assertions go into `internal/templates/templates_test.go` and `internal/cli/cli_test.go`.

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement `internal/passport` with deterministic clock-injected extraction, table-based feedback rule parsing, polyglot stack extraction, markdown formatting, JSON rendering, token estimation, and compaction. Author comprehensive unit tests covering happy paths, 0-active tasks, multi-active tasks, missing optional files, budget bounds, determinism, and benchmarks ($< 50$ms). | `internal/passport/passport.go`, `internal/passport/passport_test.go` | `go test -v ./internal/passport` |
| 2a | Implement `sdd passport` CLI command and help text. Delegate `EstimateTokens` in `internal/cli/prompt_cache.go` to `passport.EstimateTokens`. Author CLI unit tests. | `internal/cli/passport.go`, `internal/cli/passport_test.go`, `internal/cli/cli.go`, `internal/cli/cli_test.go`, `internal/cli/prompt_cache.go` | `go test -v ./internal/cli -run TestPassport` |
| 2b | Integrate `--compact` flag into `sdd status`. Author CLI tests for `sdd status --compact` and `sdd status --compact --json`. | `internal/cli/status.go`, `internal/cli/cli_test.go` | `go test -v ./internal/cli -run TestStatusCompact` |
| 3 | Implement `passport-consistency` mechanical audit in `internal/check/passport.go` and register in `internal/check/check.go`. Author unit tests in `internal/check/passport_test.go` and `internal/check/check_test.go` (`TestCheckPassport`). | `internal/check/passport.go`, `internal/check/passport_test.go`, `internal/check/check.go`, `internal/check/check_test.go` | `go test -v ./internal/check -run TestCheckPassport` |
| 4a | Update agent contract template (`internal/templates/data/base/contract.md.tmpl`) and synchronize root contracts (`AGENTS.md`, `CLAUDE.md`, `GEMINI.md`) with Progressive Disclosure Section 1 citing Decision register §6. Implement proving test `TestCheckContractsMatch` in `internal/check/check_test.go` (reading `.sdd/manifest.json`, rendering `base/contract.md.tmpl`, extracting the managed block, and asserting byte equality against root agent files). | `internal/templates/data/base/contract.md.tmpl`, `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `internal/check/check_test.go` | `go test -v ./internal/check -run TestCheckContractsMatch` |
| 4b | Update base prompt templates (`prompt.md.tmpl`, `prompt_compact.md.tmpl`) to reference progressive disclosure. Verify prefix byte invariance ($\ge 4,200$ bytes safety target, $> 500$ bytes headroom), static prefix test, and compact token budget in `internal/templates/templates_test.go`, `internal/cli/prompt_compact_test.go`, and `internal/cli/prompt_context_test.go`. | `internal/templates/data/base/prompt.md.tmpl`, `internal/templates/data/base/prompt_compact.md.tmpl`, `internal/templates/templates_test.go`, `internal/cli/prompt_compact_test.go`, `internal/cli/prompt_context_test.go` | `go test -v ./internal/templates -run TestPromptTemplateOrder && go test -v ./internal/cli -run TestPromptCompact && go test -v ./internal/cli -run TestPromptStaticPrefix` |
| 4c | Update `skills/task-coordinator/SKILL.md`, `internal/skill/export.go`, and canonical skill templates (`coordinator`, `task-coordinator`) to document the progressive disclosure pattern. Verify export sync via `TestCanonicalTaskCoordinatorSkillValid` in `internal/skill/export_test.go`. | `skills/task-coordinator/SKILL.md`, `internal/skill/export.go`, `internal/skill/export_test.go`, `internal/templates/data/skills/coordinator/SKILL.md`, `internal/templates/data/skills/task-coordinator/SKILL.md` | `go test -v ./internal/skill -run TestCanonicalTaskCoordinatorSkillValid` |
| 5 | Bump `templates.Version` to `2.11.0` across `templates.go`, `.sdd/manifest.json`, contract headers (`templates=2.11.0`), and template tests. Verify temporary kit generation (`sdd init --yes` + `sdd check` passes with 0 errors) and repository strict check (`./bin/sdd check --strict` passes with 0 errors). | `internal/templates/templates.go`, `.sdd/manifest.json`, `internal/templates/templates_test.go` | `go test -v ./internal/templates -run TestVersion && make build && ./bin/sdd check --strict` |
| 6 | Update living documents (`ROADMAP.md` Phase B: `Implementing`, Phase D: `Awaiting review`, `TECH_STACK.md` layout §4 and facts §7, `SKILLS.md` §20, `README.md`). Run full verification suite (`make verify`, race detector `go test -race ./...`, and determinism `go test -count=2 ./...`). | `ROADMAP.md`, `TECH_STACK.md`, `SKILLS.md`, `README.md` | `make verify && go test -race ./... && go test -count=2 ./...` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/passport/passport.go` | create | Core project passport extraction, formatting, and token estimation logic |
| `internal/passport/passport_test.go` | create | Unit tests for passport generation, budget ceilings, determinism, benchmarks, and missing files |
| `internal/cli/passport.go` | create | `sdd passport` CLI command implementation |
| `internal/cli/passport_test.go` | create | CLI unit tests for `sdd passport` flags and exit codes |
| `internal/cli/status.go` | modify | Integrate `--compact` flag into `sdd status` |
| `internal/cli/cli.go` | modify | Register `passport` command in top-level dispatcher and help text |
| `internal/cli/cli_test.go` | modify | Test help catalog, passport dispatch, and status compact |
| `internal/cli/prompt_cache.go` | modify | Delegate token estimation to `passport.EstimateTokens` |
| `internal/cli/prompt_compact_test.go` | modify | Update or assert compact prompt token budget with progressive disclosure |
| `internal/cli/prompt_context_test.go` | modify | Assert static prefix heading and byte invariance |
| `internal/check/passport.go` | create | `passport-consistency` mechanical audit rule |
| `internal/check/passport_test.go` | create | Unit tests for passport consistency check |
| `internal/check/check.go` | modify | Register `passport-consistency` in standard checker suite |
| `internal/check/check_test.go` | modify | Add `TestCheckPassport` and `TestCheckContractsMatch` |
| `internal/templates/data/base/contract.md.tmpl` | modify | Evolve Section 1 to Progressive Disclosure Protocol citing Decision register §6 |
| `internal/templates/data/base/prompt.md.tmpl` | modify | Update base prompt template with passport orientation |
| `internal/templates/data/base/prompt_compact.md.tmpl` | modify | Update compact base prompt template with passport orientation |
| `internal/templates/data/skills/coordinator/SKILL.md` | modify | Document progressive disclosure in coordinator skill template |
| `internal/templates/data/skills/task-coordinator/SKILL.md` | modify | Document progressive disclosure in task-coordinator skill template |
| `skills/task-coordinator/SKILL.md` | modify | Update root coordinator skill |
| `internal/skill/export.go` | modify | Update hardcoded coordinator skill in skill export |
| `internal/skill/export_test.go` | modify | Verify exported skill sync via `TestCanonicalTaskCoordinatorSkillValid` |
| `internal/templates/templates.go` | modify | Bump `templates.Version` to `2.11.0` |
| `internal/templates/templates_test.go` | modify | Assert version bump and kit templates |
| `.sdd/manifest.json` | modify | Update manifest template version to `2.11.0` |
| `AGENTS.md` | modify | Bump header to `templates=2.11.0` and sync Section 1 |
| `CLAUDE.md` | modify | Bump header to `templates=2.11.0` and sync Section 1 |
| `GEMINI.md` | modify | Bump header to `templates=2.11.0` and sync Section 1 |
| `ROADMAP.md` | modify | Update task status (Phase B: `Implementing`, Phase D: `Awaiting review`) and changelog |
| `TECH_STACK.md` | modify | Add `internal/passport/` to layout in §4 and record verified facts in §7 |
| `SKILLS.md` | modify | Document progressive disclosure standard in §20 |
| `README.md` | modify | Document `sdd passport` CLI feature and usage |
| `tasks/features/T-063-living-document-progressive-disclosure/requirements.md` | modify | Task requirements |
| `tasks/features/T-063-living-document-progressive-disclosure/plan.md` | modify | Implementation plan |
| `tasks/features/T-063-living-document-progressive-disclosure/validation.md` | modify | Validation ledger |
| `tasks/features/T-063-living-document-progressive-disclosure/review.md` | modify | Code review audit |

## 6. Dependencies
Standard library only (`D-007`). Zero new third-party dependencies.

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Set `T-063` status to `Implementing` upon Gate 1, `Awaiting review` at Gate 2, update changelog |
| `TECH_STACK.md` | Add `internal/passport/` to directory layout §4, record verified facts in §7 |
| `SKILLS.md` | Add §20 Living Document Progressive Disclosure Standard |
| `README.md` | Add `sdd passport` command reference and feature overview |

## 8. Risks, mitigations, rollback
- **Risk:** Living documents may be missing or unformatted in partial/greenfield repos.
  - **Mitigation:** Safe fallback defaults in `internal/passport` with graceful degradation; `passport-consistency` skips cleanly when `ROADMAP.md` is absent.
- **Risk:** Contract mismatch across agent files.
  - **Mitigation:** Verified mechanically via `TestCheckContractsMatch` in `internal/check/check_test.go`.
- **Risk:** Static prefix cache boundary invalidated by prompt template edits.
  - **Mitigation:** Verified via `TestPromptTemplateOrder` asserting static prefix stays $\ge 4,096$ bytes before `<!-- sdd:cache-boundary:end-prefix -->`.
- **Risk:** CRLF line endings on Windows breaking finding counts or budget calculations.
  - **Mitigation:** Strict CRLF normalization (`\r\n` -> `\n`) across all living doc reads before byte/token measurement, verified by `TestCheckCRLFInvariance`.
- **Rollback:** `git checkout main` / discard branch `feature/T-063-living-document-progressive-disclosure`.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- `F-001`: Resolve all findings (Blockers, Warnings, Minors) across Review Triad lenses before Gate 2 close.
- `F-002`: Path hygiene — zero personal absolute home directory paths in code, templates, or tests.
- `F-003`: Record default task token telemetry via `sdd record-tokens T-063` before Gate 2.

## 10. Approval record
Approved by: Vikash Kumar   Date: 2026-11-25   Phrase received: `APPROVED T-063`
