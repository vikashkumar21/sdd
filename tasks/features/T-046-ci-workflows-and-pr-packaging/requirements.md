---
status: active
task_id: T-046
last_reviewed: 2026-11-19
---
# Requirements — T-046 Universal CI workflows and automated PR packaging (sdd task pr)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-19)
Roadmap ref: `ROADMAP.md` T-046

## 1. Goal
Provide adaptable, universal CI workflow templates across major CI platforms (GitHub Actions, GitLab CI, Bitbucket Pipelines, Azure DevOps, Google Cloud Build, AWS CodeBuild) to enforce `sdd check --strict`, compile/build verification, and test suites remotely, with placeholder stages for publishing application binaries and Docker images, and introduce `sdd task pr <T-xxx>` to deterministically package task artifacts into production-ready Pull Request titles, descriptions, and machine-readable JSON bundles for human and agent workflows.

## 2. Context and references
- Mission / principles touched: Principles 1 (Deterministic), 3 (Human-in-the-loop for the irreversible), 4 (Small and shippable), 5 (Never overwrite human work).
- Decisions this relies on: `D-007` (Standard library only), `D-026` (`review.md` 4th doc), `D-027` (Typology lifecycle), `D-029` (Two-tier git governance), `D-031` (Review Triad), `D-032` (Rule F-001), `D-035` (Task state IPC), `D-040` (Universal CI workflows, automated PR packaging, and CI log ingestion).
- Depends on tasks: `T-042` (Cross-project dependencies), `T-044` (Canonical protocol skills sync), `T-045` (Turnkey prompt discovery), `T-048` (Multi-agent coordinator contract), `T-049` (Prompt cache optimization and token tracking).

## 3. Scope
**In scope**
- **Universal CI Workflow Templates (`internal/ci` & `internal/templates/data/ci/`):**
  - Canonical CI workflow templates for:
    1. GitHub Actions: `.github/workflows/sdd-check.yml`
    2. GitLab CI: `.gitlab-ci.yml`
    3. Bitbucket Pipelines: `bitbucket-pipelines.yml`
    4. Azure DevOps Pipelines: `azure-pipelines.yml`
    5. Google Cloud Build (GCP): `cloudbuild.yaml`
    6. AWS CodeBuild (AWS): `buildspec.yml`
  - CI workflows enforce:
    - Toolchain setup (Go, Node, Python, Java per project languages in manifest).
    - `sdd check --strict` execution using downloaded static `sdd` binary.
    - Code linting and formatting check (`make lint`, `eslint`, `ruff`, etc.).
    - Explicit compile and build step (`go build ./...`, `npm run build`, `./mvnw compile`, `python -m compileall .`).
    - Verification and test commands (`make verify`, `npm test`, `pytest`, `mvn test`).
    - Determinism flake guard (`-count=2` repeat check).
    - Optional/placeholder release publish stages:
      - App binary publishing step with URL placeholders (e.g. S3 / GCS / GitHub release bucket).
      - Docker container image build and publish step with registry URL placeholders (e.g. `ghcr.io`, AWS ECR, GCP Artifact Registry).
  - CLI subcommand `sdd ci init [--platform <github|gitlab|bitbucket|azure|gcp|aws|all>] [--dir <project>] [--force]`:
    - Generates target platform CI workflow file safely without overwriting existing files unless `--force`.
    - Supports `--dry-run` to preview file contents and destination.
- **Automated PR Packaging Command (`sdd task pr <T-xxx>`):**
  - Reads and synthesizes task artifacts: `requirements.md`, `plan.md`, `validation.md`, `review.md`.
  - Generates standardized PR Title: `<type>(<scope>): <summary>` (e.g. `feat(T-046): Universal CI workflows and automated PR packaging`).
  - Generates rich, structured Markdown PR Description:
    - Summary & Purpose (extracted from `requirements.md` Section 1 & 2).
    - Changes & Scope (extracted from `plan.md` Section 5 & 6).
    - Verification Evidence & Traceability (extracted from `validation.md` Section 1).
    - Review Triad Sign-off & Audit Certification (extracted from `review.md` Section 1 & 4, verifying 0 Blockers and Rule F-001 resolution).
    - Decisions and feedback rules touched.
  - Flags:
    - `--dry-run`: output formatted PR Markdown to terminal stdout without touching git or disk.
    - `--json`: output structured JSON payload (`PRBundle`: `task_id`, `typology`, `title`, `body`, `base_branch`, `head_branch`, `changed_files`, `review_status`, `qualifies_for_pr`) for programmatic Agent SDKs, GitHub CLI (`gh pr create`), GitLab CLI (`glab mr create`), or CI scripts.
    - `--output <file>`: write generated PR description directly to a markdown file.
    - `--dir <project>`: specify project root.
  - PR Qualification Gate:
    - Validates that task has 0 unresolved review blockers in `review.md` before packaging (or emits a clear warning in `--dry-run` / `--json` if not yet qualified).
- **Unit and Integration Tests:**
  - Comprehensive unit and golden tests in `internal/ci` and `internal/cli`.
  - Determinism verification (`-count=2`).
  - Strict compliance with standard library purity (`D-007`).

**Out of scope**
- Directly executing network API calls to GitHub, GitLab, or Bitbucket APIs (violates `D-007` and sandbox safety; `sdd task pr` packages the deterministic payload for the human or CLI tool like `gh`).
- Ingesting remote CI logs (this is the explicit scope of `T-047`).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL provide canonical CI workflow templates for GitHub Actions, GitLab CI, Bitbucket Pipelines, Azure DevOps, Google Cloud Build, and AWS CodeBuild in `internal/templates/data/ci/` executing `sdd check --strict`, language-specific compile/build steps, test suites, determinism verification, and placeholder publish stages for app binaries and Docker images. | MUST | test / inspection |
| R-002 | The system SHALL implement `sdd ci init [--platform <github|gitlab|bitbucket|azure|gcp|aws|all>]` to generate platform-specific CI configurations into the project repository, supporting `--dry-run` and refusing to overwrite existing files unless `--force` is specified. | MUST | test |
| R-003 | The system SHALL implement `sdd task pr <T-xxx>` to parse task documents across all 4 typologies (`tasks/features/`, `tasks/bugfixes/`, `tasks/hotfixes/`, `tasks/spikes/`) with typology-specific document fallbacks (`requirements.md` / `reproduction.md` / `incident.md` / `hypotheses.md`; `plan.md` / `patch.md`) and synthesize a standardized PR title and Markdown description. | MUST | test |
| R-004 | The system SHALL extract verification traceability from `validation.md` and Review Triad sign-off status from `review.md`, including 0 Blockers verification and Rule F-001 resolution evidence in the packaged PR description (treating missing `review.md` on spikes as N/A). | MUST | test |
| R-005 | The system SHALL support `sdd task pr <T-xxx> --json`, outputting a machine-readable JSON structure containing `task_id`, `typology`, `title`, `body`, `base_branch`, `head_branch`, `changed_files`, `review_status`, and `qualifies_for_pr`. | MUST | test |
| R-006 | The system SHALL support `sdd task pr <T-xxx> --output <file>` and `--dry-run`, writing to a target file or terminal stdout without modifying repository git state. | MUST | test |
| R-007 | The system SHALL validate that `sdd task pr` checks task qualification and warns or fails with exit code 1 if `review.md` contains unresolved 🔴 Blockers or unapproved status unless `--allow-unreviewed` is specified (in `--json` mode emitting the JSON payload with `qualifies_for_pr: false` and exiting code 1). | MUST | test |
| R-008 | The system SHALL update `sdd task` help output and `README.md` to document `sdd task pr` and `sdd ci init`. | MUST | inspection |

## 5. Non-functional requirements
- **Determinism:** PR bundle generation and CI templates must produce byte-identical outputs given identical task documents and inputs.
- **Fail-safe:** Never overwrite existing CI workflow files without explicit `--force`. Never execute git commit or push commands.
- **Standard Library Purity (`D-007`):** Pure Go standard library; zero third-party dependencies.
- **Confinement:** Path operations confined to repository root via `os.Root` or canonical filepath cleaning.

## 6. Constraints and assumptions
- Pull Request creation on remote platforms (GitHub, GitLab, Bitbucket, Azure) is handled either by the human or external CLIs (`gh pr create --title ... --body-file ...`) using the output generated by `sdd task pr`.
- Task document parsing handles all standard Markdown headers, tables, and frontmatter.

## 7. Open questions
(None. Settled in Decision D-040.)

## 8. Acceptance criteria
- **AC-1 (CI Templates Generation):** Given a project initialized with `sdd`, running `sdd ci init --platform github` creates `.github/workflows/sdd-check.yml` with `sdd check --strict`, explicit compile/build, `make verify`, determinism checks, and placeholder publish stages.
- **AC-2 (Multi-Platform CI Support):** Running `sdd ci init --platform all --dry-run` outputs plans for GitHub Actions, GitLab CI, Bitbucket Pipelines, Azure DevOps, Google Cloud Build, and AWS CodeBuild without writing to disk.
- **AC-3 (PR Packaging Title & Body):** Running `sdd task pr T-046` on a completed task outputs a formatted PR title (e.g. `feat(T-046): Universal CI workflows...`) and Markdown body with sections for Overview, Plan Traceability, Validation Evidence, and Review Triad Sign-off.
- **AC-4 (Machine-Readable JSON):** Running `sdd task pr T-046 --json` outputs valid JSON containing all PR bundle fields.
- **AC-5 (Review Gate Verification):** Running `sdd task pr <T-xxx>` on a task with unresolved blockers fails with exit code 1 unless `--allow-unreviewed` is passed.
- **AC-6 (Clean Integration):** `./bin/sdd check --strict`, `make verify`, and `make lint` pass with 0 errors and 0 warnings.
