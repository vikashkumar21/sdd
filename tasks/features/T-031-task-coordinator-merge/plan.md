# Plan — T-031 Task Coordinator pre-merge qualification, simulation, and sdd task merge

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-05)

## 1. Approach summary
Implement a Task Coordinator pre-merge qualification engine and `sdd task merge` in `internal/coordinator/`. The engine checks `review.md` for 0 blockers, computes the reverse package dependency blast radius using pure Go AST import parsing, executes a determinism flake guard (double-run test suite), simulates 3-way git merges without touching working tree state, and orchestrates GATE 2 merge approval (`APPROVED MERGE T-xxx`).

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-029 | Two-tier git governance and Task Coordinator orchestration | Option A | Option A | Approved (2026-11-05) |

## 3. Design

### Blast-Radius Package Dependency Analyzer (`internal/coordinator/blast.go`)
- Pure Go standard library AST parser (`go/parser`, `go/token`):
  - Scans Go packages in the module directory.
  - Builds forward dependency graph: `imports[pkgPath] = []string{imported1, imported2}`.
  - Inverts graph into reverse dependency index: `dependents[importedPkg] = []string{importer1, importer2}`.
  - Given a slice of modified file paths:
    - Maps modified files to their immediate packages.
    - BFS/DFS traversal finds all transitive dependent packages (the blast radius).
    - Sorts all package paths canonically for deterministic output.

### Pre-Merge Qualification Engine (`internal/coordinator/qualify.go`)
- Data structures:
  ```go
  type QualificationReport struct {
      TaskID         string         `json:"task_id"`
      Branch         string         `json:"branch"`
      TargetBranch   string         `json:"target_branch"`
      ReviewStatus   string         `json:"review_status"`
      ReviewBlockers int            `json:"review_blockers"`
      ModifiedFiles  []string       `json:"modified_files"`
      ModifiedPkgs   []string       `json:"modified_pkgs"`
      BlastRadius    []string       `json:"blast_radius"`
      TestRun1Pass   bool           `json:"test_run1_pass"`
      TestRun2Pass   bool           `json:"test_run2_pass"`
      FlakeDetected  bool           `json:"flake_detected"`
      MergeClean     bool           `json:"merge_clean"`
      MergeConflicts []string       `json:"merge_conflicts,omitempty"`
      ReadyToMerge   bool           `json:"ready_to_merge"`
      Errors         []string       `json:"errors,omitempty"`
  }
  ```
- Qualification Steps:
  1. **Review Verification:** Inspects `review.md` for 0 🔴 Blockers and status `Approved`.
  2. **Plan Deconfliction:** Invokes `deconflict.Analyze` to verify no file overlap with other concurrent active tasks.
  3. **Blast Radius Calculation:** Computes modified packages and reverse dependency blast radius.
  4. **Regression Sweep & Flake Guard:**
     - Executes `go test` on modified packages + blast radius.
     - Runs the test suite twice in succession.
     - If Run 1 passes and Run 2 fails, flags `FlakeDetected: true`.
  5. **Merge Simulation:**
     - Invokes `git merge-tree` against `targetBranch` without modifying working tree.
     - Detects merge conflicts or divergence.
  6. **GATE 2 Presentation:**
     - Generates formatted Task Coordinator summary expecting `APPROVED MERGE T-xxx`.

### CLI Subcommand (`internal/cli/task.go`)
- `sdd task merge <T-xxx>`
  - `--target <branch>`: target merge branch (default: `main`).
  - `--dry-run`: runs full qualification and merge simulation without applying merge.
  - `--rebase`: rebases task branch onto target branch before qualification.
  - `--skip-flake-check`: skips second test run.
  - `--json`: outputs machine-readable JSON qualification report.
  - `--yes`: skips interactive confirmation for human committer.

## 4. Implementation steps
Each step is small enough to review in one sitting and names its proving check.

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Create `internal/coordinator` with AST blast-radius package analyzer and unit tests. | `internal/coordinator/blast.go`, `internal/coordinator/blast_test.go` | `go test -v ./internal/coordinator -run TestBlastRadius` |
| 2 | Implement merge simulation and rebase executor using Git CLI wrappers. | `internal/coordinator/git.go`, `internal/coordinator/git_test.go` | `go test -v ./internal/coordinator -run TestMergeSimulation` |
| 3 | Implement pre-merge qualification matrix and flake guard runner. | `internal/coordinator/qualify.go`, `internal/coordinator/qualify_test.go` | `go test -v ./internal/coordinator -run TestQualify` |
| 4 | Add `sdd task merge` command in `internal/cli/task.go` and CLI tests. | `internal/cli/task.go`, `internal/cli/cli.go`, `internal/cli/cli_test.go` | `go test -v ./internal/cli -run TestTaskMerge` |
| 5 | Surface merge qualification readiness in `internal/status`. | `internal/status/status.go`, `internal/status/status_test.go` | `go test -v ./internal/status -run TestStatusMergeReadiness` |
| 6 | Update living docs (`README.md`, `ROADMAP.md`), run verification suite (`make verify`, `make check`, `sdd check --strict`), and subagent audit. | `README.md`, `ROADMAP.md` | `make check`, `sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/coordinator/blast.go` | create | Pure Go package dependency & blast radius calculator |
| `internal/coordinator/blast_test.go` | create | Unit tests for package dependency tracing |
| `internal/coordinator/git.go` | create | Merge simulation and rebase operations |
| `internal/coordinator/git_test.go` | create | Unit tests for merge simulation and conflict detection |
| `internal/coordinator/qualify.go` | create | Qualification engine, flake guard, and report formatting |
| `internal/coordinator/qualify_test.go` | create | Unit tests for qualification rules and flake detection |
| `internal/cli/task.go` | modify | Add `sdd task merge` handler |
| `internal/cli/cli.go` | modify | Update usage text |
| `internal/cli/cli_test.go` | modify | Test `sdd task merge` CLI flags and behavior |
| `internal/status/status.go` | modify | Surface merge readiness in status summary |
| `internal/status/status_test.go` | modify | Test status summary merge readiness display |
| `README.md` | modify | Document `sdd task merge` command |
| `ROADMAP.md` | modify | Update T-031 status |

## 6. Dependencies
Standard library only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `ROADMAP.md` | Update T-031 status and progress |
| `README.md` | Document `sdd task merge` command and flags |

## 8. Risks, mitigations, rollback
- **Risk:** Variations in git version affecting `git merge-tree` syntax.
  - **Mitigation:** Wrap merge simulation with fallback detection and handle older 3-argument `git merge-tree <base> <branch1> <branch2>` vs modern 2-argument flags.
- **Rollback:** Revert git working tree changes.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
- None active in §8 register.

## 10. Approval record
Approved by: Vikash (human)   Date: 2026-11-05   Phrase received: `APPROVED T-031`