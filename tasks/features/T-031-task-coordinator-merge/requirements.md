# Requirements — T-031 Task Coordinator pre-merge qualification, simulation, and sdd task merge

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-05)
Roadmap ref: `ROADMAP.md` T-031

## 1. Goal
Provide a deterministic Task Coordinator pre-merge qualification engine and `sdd task merge` CLI command that verifies 0 review blockers, computes blast-radius package regression sweeps, executes determinism flake guards (double-run), simulates git merges cleanly without repository corruption, and coordinates GATE 2 human merge approval (`APPROVED MERGE T-xxx`).

## 2. Context and references
- Mission / principles touched:
  - Principle 1 (Deterministic): Byte-identical reports and repeatability verification.
  - Principle 2 (Fail safe): Stop and refuse merge if any qualification check, test, or review blocker fails.
  - Principle 3 (Human in the loop for the irreversible): Merge to target branch requires explicit confirmation or Gate 2 phrase.
- Decisions this relies on:
  - D-002: Spec-driven workflow with human gates.
  - D-007: Go, standard library only, single static binary.
  - D-026: Code review specification, rubric, and task artifact (`review.md`).
  - D-028: Git worktree concurrency and workspace isolation.
  - D-029: Two-tier git governance and Task Coordinator orchestration.
- Depends on tasks: T-030 (Done).

## 3. Scope
**In scope**
- Pre-Merge Qualification Engine in `internal/merge`:
  - Review Document Audit: Verifies `review.md` has 0 unresolved 🔴 Blockers and status is `Approved`.
  - Static Deconfliction Verification: Verifies task has no active target file collisions with other in-flight tasks.
  - Working Tree Cleanliness: Verifies task branch has no uncommitted dirty state before qualification.
- Blast-Radius Calculation & Regression Sweep:
  - Identifies all modified `.go` files between task branch and target branch.
  - Computes the modified packages and their downstream importer graph (blast radius).
  - Executes package test sweep covering modified packages + all downstream dependents.
- Determinism Flake Guard:
  - Executes the regression test sweep twice in succession.
  - If Run 1 passes but Run 2 fails, marks qualification as failed with `Flake Detected: non-deterministic test failure`.
- Dry-Run Merge Simulation:
  - Simulates git merge against target branch (e.g. `main`) without altering working tree state.
  - Detects upstream branch drift and git conflicts.
- Rebase Helper:
  - Flag `--rebase` safely rebases the task branch onto the latest target branch.
- CLI Subcommand:
  - `sdd task merge <T-xxx>` with flags:
    - `--target <branch>` (default: `main`)
    - `--dry-run`: qualifies, computes blast radius, checks flake guard, and simulates merge without writing git branch ref.
    - `--rebase`: rebases task branch onto target branch before qualification.
    - `--skip-flake-check`: optional flag to skip the second test run.
    - `--json`: machine-readable JSON qualification report.
    - `--yes`: skips interactive confirmation for human committers.
  - Coordinates GATE 2 merge presentation expecting `APPROVED MERGE T-xxx`.

**Out of scope**
- Autonomous Task Coordinator system prompt / skill packaging (handled in T-032).
- Ensemble multi-lens code review specialized agents (handled in T-033).
- Remote GitHub Actions / GitLab CI pipeline triggers (handled in M9).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL verify that `review.md` exists and contains 0 unresolved 🔴 Blockers before qualifying a task for merge. | MUST | test |
| R-002 | The system SHALL compute the blast radius (downstream importing packages) of all packages modified on the task branch. | MUST | test |
| R-003 | The system SHALL run a regression test sweep across all modified packages and their downstream blast radius. | MUST | test |
| R-004 | The system SHALL run the test suite twice in succession (Flake Guard) and fail qualification if results differ. | MUST | test |
| R-005 | The system SHALL simulate the merge against the target branch without modifying git state in `--dry-run` mode. | MUST | test |
| R-006 | The system SHALL detect git merge conflicts during simulation and report the exact conflicting files without corrupting repository state. | MUST | test |
| R-007 | The system SHALL provide `sdd task merge <T-xxx>` supporting `--target`, `--dry-run`, `--rebase`, `--skip-flake-check`, and `--json`. | MUST | test |
| R-008 | The system SHALL support `--rebase` to automatically rebase the task branch onto the target branch before qualification. | MUST | test |
| R-009 | The system SHALL output machine-readable JSON via `sdd task merge --json` containing qualification status, blast radius, test results, and merge readiness. | MUST | test |
| R-010 | The system SHALL format a GATE 2 presentation for the Task Coordinator expecting `APPROVED MERGE T-xxx`. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Blast radius package lists and file paths are sorted canonically; qualification JSON reports are byte-identical across runs.
- **Fail-safe:** If any check fails (blocker in `review.md`, test failure, flaky test, merge conflict), `sdd task merge` exits with non-zero code and leaves git working tree untouched.
- **Standard library only:** No third-party dependencies (`D-007`); utilizes standard Go parser/toolchain and Git CLI wrappers.

## 6. Constraints and assumptions
- Requires an active git repository with standard task branch naming (e.g. `feature/T-xxx-<slug>`).
- Package dependency tracing analyzes standard Go module package import paths.

## 7. Open questions
(None - all architectural choices settled in D-028 and D-029)

## 8. Acceptance criteria
- Given a task with unresolved 🔴 Blockers in `review.md`, `sdd task merge T-xxx` refuses merge qualification and outputs the blocking findings.
- Given a task modifying `internal/deconflict`, `sdd task merge` computes blast radius including `internal/cli`, `internal/check`, and `internal/status`, and runs tests across all of them.
- Given a task where tests pass on Run 1 but fail on Run 2, `sdd task merge` flags `Flake Detected` and halts merge.
- Given `--dry-run`, the command simulates merge and reports clean qualification with zero changes to target branch.