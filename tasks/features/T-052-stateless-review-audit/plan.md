---
status: draft
---
# Plan — T-052 Stateless review context isolation and outside validation audit runner

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-21)

Approved by: Vikash (human)   Date: 2026-11-21   Phrase received: `APPROVED T-052`

## 1. Approach summary
Implement native stateless review context isolation by default across all four SDD reviewer roles (Plan Reviewer, Spec Auditor, AppSec Specialist, Pattern Auditor) and build an outside verification audit runner (`sdd task audit`).
1. Build `internal/audit`:
   - `StagePlanBundle(root, taskID string) (*PlanBundle, error)`: Stages frozen task specification and plan-reviewer rubric into `.sdd/review/<TASK_ID>/plan/`.
   - `StageReviewBundle(root, taskID, lens string) (*ReviewBundle, error)`: Stages frozen specification (`spec.md`), git diff patch (`diff.patch`), lens rubric (`rubric.md`), and metadata into `.sdd/review/<TASK_ID>/code/`.
   - `FormatPlanReviewPrompt(bundle *PlanBundle, root string) (string, error)` and `FormatReviewPrompt(bundle *ReviewBundle, root string) (string, error)`: Formats tabula-rasa reviewer prompts stripped of conversational history.
   - `EnsureGitignoreReview(root string) error`: Automatically ensures `.sdd/review/` is ignored by `.gitignore`.
   - `RunAudit(root, taskID string, opts AuditOpts) (*AuditReport, error)`: Parses executable validation commands from backtick spans in `validation.md`, validates against safety whitelist, splits compound `&&` chains, executes commands, checks exit codes/durations, and produces structured audit reports.
2. Update `sdd prompt plan-review` and `sdd prompt review` in `internal/cli/prompt.go` to use stateless reviewer bundles by default.
3. Add `sdd task audit <TASK_ID>` (and alias `sdd review audit`) in `internal/cli/task.go`, `internal/cli/cli.go`, and `internal/cli/audit.go`.
4. Comprehensive unit tests covering bundle generation, tabula-rasa prompts, command parsing, execution whitelist safety, and CLI commands.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-044 | Stateless reviewer context isolation and mechanical audit runner | A) Native tabula-rasa context staging for all 4 reviewers (.sdd/review/<TASK_ID>/) by default in sdd prompt plan-review and review, and sdd task audit executing validation commands mechanically for outside verification.<br>B) Ad-hoc prompt instructions without frozen file bundles or CLI execution runner. | Option A: Native Go standard library implementation providing deterministic context isolation by default and mechanical execution verification. | **Approved** 2026-11-21 |

## 3. Design
- Package: `internal/audit`
- Files:
  - `internal/audit/bundle.go`: Structs, bundle staging for plan review and code review triad, diff extraction, rubric resolution, `.gitignore` maintenance, and tabula-rasa prompt formatting.
  - `internal/audit/runner.go`: `validation.md` parser, command safety whitelisting, command execution, and audit reporting.
  - `internal/audit/audit_test.go`: Unit tests.
  - `internal/cli/audit.go`: CLI command dispatch for `sdd task audit` and `sdd review audit`.
  - `internal/cli/audit_test.go`: CLI tests.
- Core signatures:
  ```go
  type PlanBundle struct {
      TaskID    string    `json:"task_id"`
      TaskDir   string    `json:"task_dir"`
      BundleDir string    `json:"bundle_dir"`
      CreatedAt time.Time `json:"created_at"`
      Files     []string  `json:"files"`
  }

  type ReviewBundle struct {
      TaskID    string    `json:"task_id"`
      Lens      string    `json:"lens"`
      TaskDir   string    `json:"task_dir"`
      BundleDir string    `json:"bundle_dir"`
      CreatedAt time.Time `json:"created_at"`
      Files     []string  `json:"files"`
  }

  func StagePlanBundle(root, taskID string) (*PlanBundle, error)
  func StageReviewBundle(root, taskID, lens string) (*ReviewBundle, error)
  func FormatPlanReviewPrompt(bundle *PlanBundle, root string) (string, error)
  func FormatReviewPrompt(bundle *ReviewBundle, root string) (string, error)

  type AuditOpts struct {
      Verbose bool          `json:"verbose"`
      Timeout time.Duration `json:"timeout"`
  }

  type CommandAudit struct {
      Command  string        `json:"command"`
      Passed   bool          `json:"passed"`
      ExitCode int           `json:"exit_code"`
      Duration time.Duration `json:"duration"`
      Output   string        `json:"output,omitempty"`
      Error    string        `json:"error,omitempty"`
  }

  type AuditReport struct {
      TaskID    string         `json:"task_id"`
      Passed    bool           `json:"passed"`
      Commands  []CommandAudit `json:"commands"`
      TotalTime time.Duration  `json:"total_time"`
  }

  func RunAudit(root, taskID string, opts AuditOpts) (*AuditReport, error)
  ```
- Subsystem Details:
  1. **Plan Review Staging (`StagePlanBundle`):**
     - Finds the task folder in `tasks/<typology>/<TASK_ID>-*`.
     - Creates `.sdd/review/<TASK_ID>/plan/`.
     - Copies `requirements.md` and `plan.md` into `.sdd/review/<TASK_ID>/plan/spec.md`.
     - Loads canonical `plan-reviewer` role skill instructions into `rubric.md`.
     - Writes `manifest.json` recording bundle metadata.
     - Appends `.sdd/review/` to `.gitignore` if not already present.
  2. **Code Review Staging (`StageReviewBundle`):**
     - Finds the task folder in `tasks/<typology>/<TASK_ID>-*`.
     - Creates `.sdd/review/<TASK_ID>/code/`.
     - Combines `requirements.md` and `plan.md` into `spec.md` with explicit separator headers.
     - Extracts working tree / branch diff via `git diff HEAD` (or against merge base) into `diff.patch`, augmented with untracked new files (`git status --porcelain`) formatted as new file diffs so uncommitted new files are visible to auditors.
     - Loads canonical role skill instructions for the selected lens (`spec`, `security`, `pattern`, or composite `all`) into `rubric.md`.
     - Writes `manifest.json` recording bundle metadata.
     - Preserves standard prompt headers and JSON context field mappings for backward compatibility.
  3. **Audit Extraction & Execution (`RunAudit`):**
     - Parses `validation.md` from the task directory.
     - Scans Section 1 (Traceability matrix) and Section 2 (Commands run) for text inside backticks (e.g. `` `go test -v ...` ``).
     - Splits compound commands joined by `&&` into discrete sequential commands with standard short-circuit failure semantics.
     - Checks each command against safety whitelist:
       - Permitted prefixes: `go test`, `make`, `sdd check`, `./bin/sdd check`, `bin/sdd check`, `sdd status`, `./bin/sdd status`, `bin/sdd status`, `git grep`, `git diff`, `git status`.
       - Rejects destructive or unwhitelisted commands (`rm`, `curl`, `wget`, `bash`, `sh`, redirects `>`).
     - Executes whitelisted commands directly via `exec.Command` with timeout (default 2 minutes if `opts.Timeout == 0`), capturing combined stdout/stderr and exit codes.
     - Evaluates determinism commands (`-count=2`) and asserts exit code 0.
     - Compiles `AuditReport`.

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Create `internal/audit/bundle.go` with reviewer bundle staging (plan and code review), diff extraction, `.gitignore` maintenance, and tabula-rasa prompt formatting | `internal/audit/bundle.go` | `go test -v ./internal/audit -run TestStageBundle` |
| 2 | Create `internal/audit/runner.go` with validation command extraction (backtick parsing & `&&` splitting), safety whitelisting, and mechanical execution | `internal/audit/runner.go` | `go test -v ./internal/audit -run TestRunAudit` |
| 3 | Add unit tests for bundle staging, tabula-rasa prompt generation, command extraction, and whitelist safety | `internal/audit/audit_test.go` | `go test -v ./internal/audit` |
| 4 | Update `sdd prompt plan-review` and `sdd prompt review` in `internal/cli/prompt.go` to use stateless reviewer bundles by default | `internal/cli/prompt.go`, `internal/cli/prompt_test.go` | `go test -v ./internal/cli -run TestPromptReview` |
| 5 | Implement `sdd task audit` in `internal/cli/task.go` and `internal/cli/audit.go` (and `sdd review audit` alias in `internal/cli/cli.go`) with `--json` support | `internal/cli/audit.go`, `internal/cli/task.go`, `internal/cli/cli.go`, `internal/cli/audit_test.go`, `.gitignore` | `go test -v ./internal/cli -run TestTaskAudit` |
| 6 | Verify full test suite, race detection, determinism, linting, and strict check | Entire repository | `make verify && make lint && ./bin/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/audit/bundle.go` | create | Reviewer bundle staging, diff extraction, and tabula-rasa prompt generator |
| `internal/audit/runner.go` | create | Outside validation command parser, safety whitelist, and execution runner |
| `internal/audit/audit_test.go` | create | Unit tests for audit package |
| `internal/cli/audit.go` | create | CLI subcommand handler for `sdd task audit` and `sdd review audit` |
| `internal/cli/audit_test.go` | create | CLI integration tests for audit command |
| `internal/cli/task.go` | modify | Dispatch `audit` subcommand in `runTask` |
| `internal/cli/prompt.go` | modify | Generate stateless reviewer bundles by default for plan-review and review |
| `internal/cli/prompt_test.go` | create | Test stateless reviewer prompts and bundle staging |
| `internal/cli/cli.go` | modify | Route `review audit` CLI command |
| `.gitignore` | modify | Add `.sdd/review/` transient bundle directory to gitignore |
| `tasks/features/T-052-stateless-review-audit/requirements.md` | modify | Task specification |
| `tasks/features/T-052-stateless-review-audit/plan.md` | modify | Implementation plan |
| `tasks/features/T-052-stateless-review-audit/validation.md` | modify | Verification matrix and evidence |
| `tasks/features/T-052-stateless-review-audit/review.md` | modify | Review Triad audit record |
| `ROADMAP.md` | modify | Status updates (Specifying -> Implementing -> Awaiting review -> Done) |
| `TECH_STACK.md` | modify | Approve Decision D-044 and log verified fact |
| `README.md` | modify | Document stateless reviewer isolation and `sdd task audit` commands |

## 6. Dependencies
None. Standard Go library only (`os`, `os/exec`, `path/filepath`, `strings`, `regexp`, `encoding/json`, `time`). Preserves `D-007`.

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Update T-052 status across gates |
| `TECH_STACK.md` | Approve Decision D-044 in register and record verified fact |
| `README.md` | Document stateless reviewer isolation and `sdd task audit` CLI usage |

## 8. Risks, mitigations, rollback
- **Risk:** Malicious or destructive command execution during outside validation audit.
  - *Mitigation:* Strict safety whitelist permitting only known verification commands (`go test`, `make`, `sdd check`, `./bin/sdd`, `bin/sdd`, `sdd status`, `git grep`, `git diff`, `git status`). All unapproved commands are refused with clear error.
- **Rollback:** Delete `internal/audit/`, `internal/cli/audit.go`, and revert changes to `cli.go`, `task.go`, `prompt.go`, and `.gitignore`.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- **Rule F-001 (Resolve all review findings):** All findings across Spec, Security, and Pattern lenses will be resolved before Phase D close.
- **Rule F-002 (Zero personal local paths):** All generated bundles, prompts, and tests will use repository-relative paths and generic placeholders.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-21   Phrase received: `APPROVED T-052`
