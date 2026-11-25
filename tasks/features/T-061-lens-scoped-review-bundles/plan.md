# Plan — T-061 Lens-scoped review bundles and diff filtering

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved 2026-11-25 | Implementing

## 1. Approach summary
Implement lens-scoped directory isolation (`.sdd/review/<TASK_ID>/<lens>/` where `<lens>` is `spec`, `security`, `pattern`, and `code` for fallback `all`) and a deterministic diff filtering engine (`FilterGitDiffByLens`). By filtering out irrelevant documentation changelogs, task markdown files, or non-functional assets based on the auditor's specific lens, the Review Triad's context token consumption is reduced by 30–70% while completely eliminating directory collisions during parallel reviewer execution. Multi-loop remediation findings (`findings_to_verify.md`) are likewise filtered by lens domain (`FilterFindingsForLens`), and manifest telemetry (`raw_diff_bytes`, `filtered_diff_bytes`, `diff_savings_percent`) is recorded in `manifest.json`. Templates version is bumped to `2.9.0` per Hard Rule 16.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-050 | Token consumption mitigation, context hygiene, and provider-aligned caching architecture | A) Tier routing, lens bundles, prefix caching, micro-index, transcript ingestion B) Uniform frontier model routing without bundle filtering | Option A (Settled in T-060) | **Approved** 2026-11-25 |

## 3. Design
1. **Isolated Directory Scoping:**
   - Determine the bundle directory based on normalized lens, handling alias `"appsec"` -> `"security"`:
     ```go
     normLens := strings.ToLower(strings.TrimSpace(lens))
     if normLens == "appsec" {
         normLens = "security"
     }
     if normLens == "" {
         normLens = "all"
     }
     bundleDirRel := filepath.Join(".sdd", "review", normTaskID, normLens)
     if normLens == "all" {
         bundleDirRel = filepath.Join(".sdd", "review", normTaskID, "code")
     }
     ```
   - Parallel reviewers running `spec`, `security`, and `pattern` operate in isolated filesystem directories without file collisions or locks.

2. **Diff Filtering Engine (`internal/audit/filter.go`):**
   - Signature:
     ```go
     func FilterGitDiffByLens(rawDiff, lens string) (filteredDiff string, rawBytes, filteredBytes, rawFiles, filteredFiles int, savingsPercent float64)
     ```
   - Unified diff parser splits diff into hunks by `diff --git a/... b/...` header.
   - Extracts target relative path from `b/<path>` (or `a/<path>` for deletions).
   - Classifies path by lens relevance:
     - `security`: Retains source code (`*.go`, `*.py`, `*.ts`, `*.js`, `*.java`, etc.), scripts (`*.sh`, `*.bash`, `*.ps1`, `Makefile`, `Dockerfile*`), configuration files (`*.json`, `*.yaml`, `*.yml`, `*.toml`), dependency files (`go.mod`, `go.sum`, `package.json`, `pom.xml`), CI workflows (`.github/**`, etc.), and agent contracts (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`, `.sdd/roles.json`, `.sdd/manifest.json`, hooks). Filters out pure documentation (`tasks/**/*.md`, `docs/**/*.md`, `README.md`, `ROADMAP.md`, `MISSION.md`).
     - `pattern`: Retains source code, test files, architecture decisions, coding standards, living docs (`TECH_STACK.md`, `SKILLS.md`), and contracts (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`). Filters out task runtime docs (`tasks/**/validation.md`, `tasks/**/review.md`) and roadmap changelogs.
     - `spec`: Retains source code, test files, and task specifications (`tasks/**/requirements.md`, `tasks/**/plan.md`). For boundary verification, retains all touched file headers while truncating or omitting non-functional documentation diff bodies.
     - `all`: Unfiltered full git diff.
   - Fail-safe fallback: If all files are excluded, outputs `# Note: all changes in this task are outside %s lens focus; no lens-specific diffs detected.\n`.
   - Zero-division guard: If `rawBytes <= 0`, `savingsPercent` explicitly evaluates to `0.0`.

3. **Remediation Findings Filtering:**
   - Signature:
     ```go
     func FilterFindingsForLens(findings []FindingItem, lens string) []FindingItem
     ```
   - Filters finding items based on Category and ID prefix (`SPEC-`, `SEC-`, `PAT-` / `PQ-`). Unassigned or general findings are included across all lenses to guarantee zero defect drops.

4. **Telemetry & Manifest Tracking:**
   - Extend `ReviewBundle` struct:
     ```go
     type ReviewBundle struct {
         TaskID             string    `json:"task_id"`
         Lens               string    `json:"lens"`
         Loop               int       `json:"loop,omitempty"`
         TaskDir            string    `json:"task_dir"`
         BundleDir          string    `json:"bundle_dir"`
         CreatedAt          time.Time `json:"created_at"`
         Files              []string  `json:"files"`
         RawDiffBytes       int       `json:"raw_diff_bytes"`
         FilteredDiffBytes  int       `json:"filtered_diff_bytes"`
         DiffSavingsPercent float64   `json:"diff_savings_percent"`
         RawFilesCount      int       `json:"raw_files_count"`
         FilteredFilesCount int       `json:"filtered_files_count"`
     }
     ```

5. **Review Prompt Instructions Refinement:**
   - Refine `FormatReviewPrompt` in `internal/audit/bundle.go` so instructions direct ephemeral subagent reviewers to output their findings, verified ledger, and sign-off in their audit report/message, clarifying that the primary worker agent batch-remediates findings and updates `tasks/<typology>/<task>/review.md` without concurrent write contention.

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement `FilterGitDiffByLens`, `FilterFindingsForLens`, and diff hunk parser with zero-division guard | `internal/audit/filter.go`, `internal/audit/filter_test.go` | `go test -v ./internal/audit -run TestFilterGitDiffByLens` |
| 2 | Wire lens directory scoping (`.sdd/review/<TASK_ID>/<lens>/`), `"appsec"` alias normalization, prompt instruction refinement, and diff filtering into `StageReviewBundleWithLoop` | `internal/audit/bundle.go`, `internal/audit/bundle_test.go` | `go test -v ./internal/audit -run TestStageReviewBundle_LensScoped` |
| 3 | Update `sdd prompt review` and `ReviewPromptContext` JSON export to surface lens bundle directory and telemetry | `internal/cli/prompt.go`, `internal/cli/prompt_test.go` | `go test -v ./internal/cli -run TestPromptReview` |
| 4 | Synchronize role skills (`code-review`, `task-coordinator`, `spec-auditor`, `coordinator`) and contract template with lens-scoped review bundles | `skills/code-review/SKILL.md`, `skills/task-coordinator/SKILL.md`, `internal/templates/data/skills/`, `internal/templates/data/base/contract.md.tmpl` | `./bin/sdd check --strict` |
| 5 | Bump `templates.Version` to `2.9.0` across codebase, contracts, manifests, and tests | `internal/templates/templates.go`, `.sdd/manifest.json`, `CLAUDE.md`, `GEMINI.md`, `AGENTS.md`, `internal/templates/templates_test.go`, `internal/upgrade/upgrade_test.go` | `go test -v ./internal/templates ./internal/upgrade -run "TestVersion|TestUpgrade"` |
| 6 | Execute verification suites, outside audit, determinism, and race checks | `validation.md`, `review.md` | `make verify && go test -race ./... && go test -count=2 ./... && ./bin/sdd task audit T-061` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/audit/filter.go` | create | Deterministic diff filtering engine (`FilterGitDiffByLens`), finding filter (`FilterFindingsForLens`), and zero-division guard |
| `internal/audit/filter_test.go` | create | Unit tests for diff parsing, boundary preservation, zero-division, and finding filtering |
| `internal/audit/bundle.go` | modify | Lens-scoped directory isolation (`.sdd/review/<TASK_ID>/<lens>/`), `"appsec"` normalization, prompt instruction refinement, bundle telemetry |
| `internal/audit/bundle_test.go` | modify | Test assertions for lens directories, diff savings, and telemetry in `manifest.json` |
| `internal/audit/audit_test.go` | modify | Update legacy test bundle path assertions for lens isolation |
| `internal/cli/prompt.go` | modify | Expose lens-scoped bundle directory, bundle files, and filtering telemetry in `sdd prompt review` |
| `internal/cli/prompt_test.go` | modify | Test assertions for lens-scoped prompt generation and JSON context export |
| `internal/templates/templates.go` | modify | Bump `templates.Version` from `2.8.0` to `2.9.0` |
| `internal/templates/templates_test.go` | modify | Update template version assertions |
| `internal/templates/data/base/contract.md.tmpl` | modify | Update contract template if necessary for lens bundle consistency |
| `internal/templates/data/skills/spec-auditor/SKILL.md` | modify | Update embedded spec-auditor skill to reference lens-scoped ledger path (`spec/ledger.md`) |
| `internal/templates/data/skills/coordinator/SKILL.md` | modify | Update embedded coordinator skill to reference lens-scoped review bundles |
| `internal/templates/data/skills/task-coordinator/SKILL.md` | modify | Update embedded task-coordinator skill to document Review Triad parallel execution in isolated lens directories |
| `skills/code-review/SKILL.md` | modify | Canonical project skill update for lens-scoped review bundles and diff filtering |
| `skills/task-coordinator/SKILL.md` | modify | Canonical project skill update for lens-scoped Review Triad orchestration |
| `CLAUDE.md` | modify | Bump contract header to `templates=2.9.0` |
| `GEMINI.md` | modify | Bump contract header to `templates=2.9.0` |
| `AGENTS.md` | modify | Bump contract header to `templates=2.9.0` |
| `.sdd/manifest.json` | modify | Bump `template_version` to `2.9.0` |
| `internal/upgrade/upgrade_test.go` | modify | Update upgrade version test fixtures |
| `ROADMAP.md` | modify | Update T-061 status and change log |
| `TECH_STACK.md` | modify | Document verified facts and review bundle architecture |
| `SKILLS.md` | modify | Update Section 10 Code Review standard with lens-scoped review bundles |
| `README.md` | modify | Update documentation with lens-scoped review bundles and token savings |
| `tasks/features/T-061-lens-scoped-review-bundles/requirements.md` | modify | Task specification |
| `tasks/features/T-061-lens-scoped-review-bundles/plan.md` | modify | Task implementation plan |
| `tasks/features/T-061-lens-scoped-review-bundles/validation.md` | modify | Validation matrix and evidence |
| `tasks/features/T-061-lens-scoped-review-bundles/review.md` | modify | Code review artifact |

## 6. Dependencies
None. Go standard library only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Update T-061 status to `Awaiting review`, then `Done`; update change log |
| `TECH_STACK.md` | Record verified fact for lens-scoped review bundles in Section 7 |
| `SKILLS.md` | Update Section 10 Code Review standard with lens-scoped review bundles and diff filtering |
| `README.md` | Document lens-scoped review bundles in CLI usage and review sections |

## 8. Risks, mitigations, rollback
- Risk: Diff filtering inadvertently strips code hunks needed for security or spec review.
  → Mitigation: Inclusions are whitelist-based by file extension and prefix. If any unrecognized extension or ambiguous file type is encountered, it defaults to inclusion. For `--lens all`, filtering is completely bypassed.
- Risk: Reviewers running in parallel experience file locking or race conditions on the task's `review.md`.
  → Mitigation: Ephemeral reviewers write to their own lens directory review artifacts; only the primary worker agent batch-remediates and updates `review.md` during Phase D close.
- Rollback: Revert git commits; all bundle directories are under `.sdd/review/` which is in `.gitignore`.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- **Rule F-001 (Resolve all review findings):** All findings (🔴 Blockers, 🟡 Warnings, 🟢 Minors) across all 3 lenses must be resolved or explicitly justified before GATE 2 close.
- **Rule F-002 (Path hygiene):** Zero personal absolute filesystem paths (`/Users/...`, `/home/...`) in documentation, templates, or tests.
- **Rule F-003 (Token telemetry):** Track and record session token consumption via `sdd record-tokens T-061` and report metrics at GATE 2.

## 10. Approval record
Approved by: Vikash
Date: 2026-11-25
Phrase received: `APPROVED T-061`
