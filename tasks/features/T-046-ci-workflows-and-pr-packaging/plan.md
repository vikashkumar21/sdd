---
status: active
task_id: T-046
last_reviewed: 2026-11-19
---
# Plan — T-046 Universal CI workflows and automated PR packaging (sdd task pr)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-19)
Roadmap ref: `ROADMAP.md` T-046

## 1. Approach summary
Implement universal CI workflow templates and generation engine (`internal/ci`) covering GitHub Actions, GitLab CI, Bitbucket Pipelines, Azure DevOps, Google Cloud Build (GCP), and AWS CodeBuild, paired with `sdd ci init`. Workflows incorporate language-aware toolchain setup, `sdd check --strict`, linting, explicit compile/build steps, test execution, determinism flake guards, and placeholder publish stages for application binaries and Docker images. Implement an automated task PR packaging engine (`internal/taskpr`) and CLI command `sdd task pr <T-xxx>` to synthesize task quartet documents (`requirements.md`, `plan.md`, `validation.md`, `review.md`) into standardized PR titles, structured Markdown bodies, and machine-readable JSON payloads (`--json`).

## 2. Decisions involved

| ID | Title | Options | Recommendation | Status |
|---|---|---|---|---|
| D-040 | Universal CI workflows, automated PR packaging, and CI log ingestion | Option A: Adaptable CI workflow templates (GitHub Actions, GitLab CI, Bitbucket, Azure DevOps, Google Cloud Build, AWS CodeBuild) with compile/build step and binary/docker publish placeholders, sdd task pr packaging command, and structured PR JSON bundle / Option B: Simple markdown printer without multi-platform CI templates or machine-readable JSON | Option A: Provides complete multi-platform CI coverage across top cloud and git providers, automates PR packaging across all 4 typologies, and outputs JSON for seamless integration with human developers and Agent SDKs while maintaining D-007 standard library purity. | Open (resolves at GATE 1) |

## 3. Design
### 3.1 CI Workflow Generation Engine (`internal/ci`)
```go
package ci

type Platform string

const (
    PlatformGitHub    Platform = "github"
    PlatformGitLab    Platform = "gitlab"
    PlatformBitbucket Platform = "bitbucket"
    PlatformAzure     Platform = "azure"
    PlatformGCP       Platform = "gcp"
    PlatformAWS       Platform = "aws"
    PlatformAll       Platform = "all"
)

// PlatformPathMap defines canonical workflow paths per platform:
//   PlatformGitHub    -> ".github/workflows/sdd-check.yml"
//   PlatformGitLab    -> ".gitlab-ci.yml"
//   PlatformBitbucket -> "bitbucket-pipelines.yml"
//   PlatformAzure     -> "azure-pipelines.yml"
//   PlatformGCP       -> "cloudbuild.yaml"
//   PlatformAWS       -> "buildspec.yml"
```

type Config struct {
    Platform Platform
    DryRun   bool
    Force    bool
    Project  string
    Command  string
}

type ResultFile struct {
    Path     string
    Content  string
    Created  bool
    Skipped  bool
    Modified bool
}

func GenerateWorkflows(root string, cfg Config) ([]ResultFile, error)
func ApplyWorkflows(root string, files []ResultFile, force bool) error
```

### 3.2 PR Packaging Engine (`internal/taskpr`)
```go
package taskpr

type PROptions struct {
    TaskID          string
    ProjectDir      string
    AllowUnreviewed bool
    BaseBranch      string
    HeadBranch      string
}

type PRBundle struct {
    TaskID         string   `json:"task_id"`
    Typology       string   `json:"typology"`
    Title          string   `json:"title"`
    Summary        string   `json:"summary"`
    Body           string   `json:"body"`
    BaseBranch     string   `json:"base_branch"`
    HeadBranch     string   `json:"head_branch"`
    ChangedFiles   []string `json:"changed_files"`
    ReviewStatus   string   `json:"review_status"`
    QualifiesForPR bool     `json:"qualifies_for_pr"`
    Warnings       []string `json:"warnings,omitempty"`
}

func PackageTaskPR(root string, opts PROptions) (*PRBundle, error)
func FormatPRMarkdown(bundle *PRBundle) string
```

**Typology Document Fallbacks & Compatibility:**
- **Features:** reads `requirements.md`, `plan.md`, `validation.md`, `review.md`.
- **Bugfixes:** falls back to `reproduction.md` and `rca.md` if `requirements.md` is absent; reads `plan.md`, `validation.md`, `review.md`.
- **Hotfixes:** falls back to `incident.md` for problem context and `patch.md` for changes; reads `validation.md`; if `review.md` is absent, review qualification is treated as N/A or derived from `patch.md` Section 4 expedited approval record.
- **Spikes:** falls back to `hypotheses.md` and `findings.md`; treats `review.md` and `validation.md` as N/A (spikes are disposable prototypes without merge review).

**CLI Exit Code Semantics:**
- If a task has unresolved 🔴 Blockers or unapproved review status:
  - In standard markdown mode: outputs a qualification failure message to stderr and exits with code 1.
  - In `--json` mode: emits the full structured `PRBundle` JSON to stdout with `"qualifies_for_pr": false` and warning list, and exits with code 1 so callers (e.g. Agent SDKs, scripts) can parse the qualification failure payload.
  - Passing `--allow-unreviewed` overrides the qualification check, sets `"qualifies_for_pr": true`, and exits with code 0.

### 3.3 CLI Command Surface
- `sdd ci init [--platform <github|gitlab|bitbucket|azure|gcp|aws|all>] [--dry-run] [--force] [--dir <project>]`
- `sdd task pr <T-xxx> [--json] [--dry-run] [--output <file>] [--allow-unreviewed] [--base <branch>] [--dir <project>]`

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Create CI workflow templates in `internal/templates/data/ci/` for GitHub Actions (`github.yml.tmpl`), GitLab CI (`gitlab.yml.tmpl`), Bitbucket Pipelines (`bitbucket.yml.tmpl`), Azure DevOps (`azure.yml.tmpl`), Google Cloud Build (`gcp.yml.tmpl`), and AWS CodeBuild (`aws.yml.tmpl`) with compile/build steps and placeholder publish stages for app binaries and Docker images. Author `internal/ci` generation and apply engine with unit tests. | `internal/templates/data/ci/*.tmpl`, `internal/ci/ci.go`, `internal/ci/ci_test.go` | `go test -v ./internal/ci` |
| 2 | Implement `sdd ci init` CLI command and dispatch in `internal/cli/ci.go`, wiring platform flags, dry-run, force protection, and project detection. Author CLI tests. | `internal/cli/ci.go`, `internal/cli/cli.go`, `internal/cli/ci_test.go` | `go test -v ./internal/cli -run TestCIInit` |
| 3 | Implement `internal/taskpr` packaging engine: parse task quartet documents across all 4 typologies (`features`, `bugfixes`, `hotfixes`, `spikes`), extract requirements summary, plan changes, validation evidence, Review Triad certification, qualify PR readiness, and render Markdown and JSON bundles. Author tests. | `internal/taskpr/taskpr.go`, `internal/taskpr/taskpr_test.go` | `go test -v ./internal/taskpr` |
| 4 | Implement `sdd task pr <T-xxx>` subcommand in `internal/cli/task.go`, supporting `--json`, `--dry-run`, `--output`, `--allow-unreviewed`, and `--base`. Wire into `runTask` dispatch and author CLI tests. | `internal/cli/task.go`, `internal/cli/task_test.go` | `go test -v ./internal/cli -run TestTaskPR` |
| 5 | Update `README.md` and `sdd task` / `sdd` help outputs with documentation and examples for `sdd ci init` and `sdd task pr`. | `README.md`, `internal/cli/cli.go`, `internal/cli/task.go` | `make build && ./bin/sdd task` |
| 6 | Execute full verification: `make verify`, `make lint`, `./bin/sdd check --strict`, and determinism check (`-count=2`). Record evidence in `validation.md`. | (all) | `make verify && make lint && ./bin/sdd check --strict` |

## 5. Files to touch

| File | Action | Purpose |
|---|---|---|
| `internal/templates/data/ci/github.yml.tmpl` | create | Canonical GitHub CI workflow template |
| `internal/templates/data/ci/gitlab.yml.tmpl` | create | Canonical GitLab CI workflow template |
| `internal/templates/data/ci/bitbucket.yml.tmpl` | create | Canonical Bitbucket Pipelines workflow template |
| `internal/templates/data/ci/azure.yml.tmpl` | create | Canonical Azure DevOps workflow template |
| `internal/templates/data/ci/gcp.yml.tmpl` | create | Canonical Google Cloud Build workflow template |
| `internal/templates/data/ci/aws.yml.tmpl` | create | Canonical AWS CodeBuild workflow template |
| `internal/ci/ci.go` | create | Multi-platform CI workflow generation and apply engine |
| `internal/ci/ci_test.go` | create | Unit and golden tests for CI workflow generation |
| `internal/taskpr/taskpr.go` | create | Task PR packaging engine extracting quartet metadata |
| `internal/taskpr/taskpr_test.go` | create | Unit tests for PR packaging, formatting, and qualification |
| `internal/cli/ci.go` | create | CLI handler for `sdd ci init` |
| `internal/cli/ci_test.go` | create | CLI tests for `sdd ci init` |
| `internal/cli/cli.go` | modify | Register `sdd ci` top-level command dispatch and help text |
| `internal/cli/task.go` | modify | Register `sdd task pr` subcommand and help text |
| `internal/cli/task_test.go` | modify | CLI tests for `sdd task pr` |
| `README.md` | modify | Document `sdd ci init` and `sdd task pr` |
| `ROADMAP.md` | modify | Update T-046 status and change log |
| `TECH_STACK.md` | modify | Update D-040 status to Approved and add verified facts entry |

## 6. Dependencies
Standard library only (`D-007`). Zero external third-party dependencies.

## 7. Living docs to update
- `ROADMAP.md`: update status to `Awaiting review` in Phase D, `Done` upon `COMMITTED T-046`; record in change log.
- `TECH_STACK.md`: update `D-040` to Approved, record verified facts entry in §7.
- `README.md`: document `sdd ci init` and `sdd task pr` usage and examples.

## 8. Risks, mitigations, rollback
- **Risk:** Projects may have existing custom CI workflow files (e.g. `.github/workflows/ci.yml`).
  - **Mitigation:** Default to creating `.github/workflows/sdd-check.yml`; never overwrite any existing file without explicit `--force`.
- **Risk:** Tasks in early phases or incomplete reviews could be packaged prematurely into PRs.
  - **Mitigation:** Inspect `review.md` and `validation.md`; fail with clear error and exit code 1 if 🔴 Blockers exist or review is not Approved, unless `--allow-unreviewed` is explicitly passed.
- **Rollback:** `git checkout` reverts modifications; delete created `internal/ci/` and `internal/taskpr/` files.

## 9. Applicable human-feedback rules
- **F-001:** Worker agents must resolve all findings (Blockers, Warnings, and Minors) reported during review before presenting to human at GATE 2; any retained finding must be explicitly justified in `review.md`.

## 10. Approval record
Plan Review Subagent: Approved (0 Blockers) 2026-11-19 (conv: `95121abd-1e40-488b-92bb-a59a13b2f562`)
Approved by: Vikash (human)   Date: 2026-11-19   Phrase received: `APPROVED T-046`
