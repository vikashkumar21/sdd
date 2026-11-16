---
status: active
task_id: T-046
last_reviewed: 2026-11-16
---
# Validation — T-046 Universal CI workflows and automated PR packaging (sdd task pr)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Not run | Complete
Roadmap ref: `ROADMAP.md` T-046

## 1. Traceability matrix (R-xxx -> evidence)

| ID | Method | Command | Expected | Actual | Status |
|---|---|---|---|---|---|
| R-001 | test / inspection | `go test -v ./internal/ci -run TestCIWorkflowTemplates` | Canonical CI workflow templates for GitHub, GitLab, Bitbucket, and Azure DevOps render valid YAML and execute `sdd check --strict` and test suites | pending | pending |
| R-002 | test | `go test -v ./internal/cli -run TestCIInit` | `sdd ci init` creates workflow files safely, supports `--dry-run`, `--platform`, and protects existing files without `--force` | pending | pending |
| R-003 | test | `go test -v ./internal/taskpr -run TestPackageTaskPR` | `PackageTaskPR` extracts requirements summary, plan changes, validation evidence, and review sign-off across all typologies | pending | pending |
| R-004 | test | `go test -v ./internal/taskpr -run TestFormatPRMarkdown` | Formatted PR Markdown contains rich overview, traceability matrix, and Review Triad audit certification | pending | pending |
| R-005 | test | `go test -v ./internal/cli -run TestTaskPR_JSON` | `sdd task pr <T-xxx> --json` outputs structured JSON bundle with all fields for Agent SDKs and CLIs | pending | pending |
| R-006 | test | `go test -v ./internal/cli -run TestTaskPR_DryRunAndOutput` | `sdd task pr` supports `--dry-run` and `--output <file>` without touching git state | pending | pending |
| R-007 | test | `go test -v ./internal/taskpr -run TestQualificationGate` | `sdd task pr` flags error on unresolved review blockers unless `--allow-unreviewed` is passed | pending | pending |
| R-008 | test / inspection | `make build && ./bin/sdd task pr --help && ./bin/sdd ci --help` | CLI help output and README.md document `sdd task pr` and `sdd ci init` | pending | pending |

## 2. Verification log

```bash
# To be filled during Phase C.1 verification
```

## 3. Non-functional and Negative test cases

| Test Case | Method | Expected | Actual | Status |
|---|---|---|---|---|
| Overwrite protection | `go test -v ./internal/ci` | Existing CI files preserved unless `--force` is specified | pending | pending |
| Invalid platform refusal | `go test -v ./internal/cli` | Unknown CI platform exits with error and allowed platforms list | pending | pending |
| Missing task directory | `go test -v ./internal/taskpr` | Non-existent task ID returns descriptive error | pending | pending |
| Unreviewed task refusal | `go test -v ./internal/taskpr` | Task with unresolved blockers rejected unless `--allow-unreviewed` | pending | pending |
| Strict check | `./bin/sdd check --strict` | 0 errors, 0 warnings on repository | pending | pending |

## 4. Plan boundary compliance
- Only files listed in `plan.md` Section 5 modified.
- Zero unapproved file modifications.

## 5. Dependency hygiene
- Standard library only (`D-007`). Zero new dependencies.

## 6. Feedback rule compliance
- **F-001:** All review findings across all 3 lenses resolved before GATE 2; compliance recorded in `review.md`.

## 7. Determinism check
- Running tests twice (`-count=2`) produces identical pass/fail results.

## 8. Token Metrics

| Agent / Role | Model | Input Tokens | Cache Read Tokens | Output Tokens | Total Tokens |
|---|---|---|---|---|---|
| Task Coordinator | | | | | |
| Plan Reviewer (`d28944bf`) | Inherit | 26,450 | 0 | 1,820 | 28,270 |
| Spec Auditor | | | | | |
| AppSec Specialist | | | | | |
| Pattern Auditor | | | | | |
| **Total for T-046** | | | | | |
| **Cumulative Session Total** | | | | | |

## 9. Sign-off
Suggested commit message: `feat(ci): add universal CI workflow templates and sdd task pr packaging command`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-046`
