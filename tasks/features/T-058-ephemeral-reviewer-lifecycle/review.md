---
status: completed
---
# Code Review — T-058 Ephemeral tabula-rasa reviewer lifecycle and zero-negotiation loop isolation

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Reviewer: Review Triad (spec-auditor, appsec-specialist, pattern-auditor)
Commit / Diff Ref: HEAD (T-058 candidate)

## 1. Requirements Traceability Ledger

| Req ID | Requirement Statement | Code Symbol & File | Positive Test Evidence | Negative / Boundary Test Evidence | Status |
|---|---|---|---|---|---|
| R-001 | The system SHALL codify the Ephemeral Reviewer Lifecycle standard in coordinator and review skills and contract templates, strictly forbidding reviewer subagent conversation reuse across remediation loops. | `skills/task-coordinator/SKILL.md`, `skills/code-review/SKILL.md`, `internal/templates/data/base/contract.md.tmpl` | `TestContractTemplateCoordinatorAndWorkflow` in `templates_test.go` | `TestManagedBlockParityIgnoresTextOutsideTheBlock` in `check_test.go` | Verified |
| R-002 | The system SHALL enforce Zero-Negotiation Loop Isolation, requiring remediation loops to present candidate diffs and structured finding checklists with zero developer prose, excuses, or negotiation transcripts. | `StageReviewBundleWithLoop`, `FormatFindingsToVerify` in `internal/audit/bundle.go` | `TestStageReviewBundle_Loop` in `bundle_test.go` | `TestStageReviewBundle_SymlinkEscape` in `bundle_test.go` | Verified |
| R-003 | The system SHALL implement `ExtractFindingsToVerify` and stage an objective `findings_to_verify.md` in `.sdd/review/<TASK_ID>/code/` for remediation loops (loop > 1). | `ExtractFindingsToVerify`, `FormatFindingsToVerify`, `StageReviewBundleWithLoop` in `internal/audit/bundle.go` | `TestExtractFindingsToVerify`, `TestStageReviewBundle_Loop` in `bundle_test.go` | `TestExtractFindingsToVerify` (malformed, placeholder, and header rows skipped) | Verified |
| R-004 | The system SHALL support `--loop <1..3>` in `sdd prompt review`, generating tabula-rasa remediation prompts that instruct fresh ephemeral reviewers without conversational anchoring. | `runPromptReview`, `buildReviewPromptWithLens` in `internal/cli/prompt.go` | `TestPromptReview_Loop` (loop 2 text and JSON) in `prompt_test.go` | `TestPromptReview_Loop` (invalid loop 0, 4 exits code 2) in `prompt_test.go` | Verified |
| R-005 | The system SHALL implement the mechanical `remediation-hygiene` rule in `sdd check`, reporting errors for conversational transcripts or negotiation chatter in `review.md`. | `checkReviewRemediationHygiene` in `internal/check/review.go` | `TestCheckRemediationHygiene` (flags conversational chatter) in `review_test.go` | `TestCheckRemediationHygiene` (ignores done tasks, clean review passes) in `review_test.go` | Verified |
| R-006 | The system SHALL update agent contract templates (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`), bump `templates.Version` to `2.6.0`, and verify fresh kit parity per Hard Rule 16. | `templates.Version` in `internal/templates/templates.go`, contracts, manifests | `TestVersion`, `TestFreshKitHasNoErrors` in `templates_test.go` | `TestDetectsProblems/agent_contract_drift` in `check_test.go` | Verified |
| R-007 | The system SHALL implement all bundle staging, finding extraction, prompt formatting, and verification logic using only the Go standard library with zero third-party dependencies (D-007). | Stdlib only in `internal/audit`, `internal/check`, `internal/cli`, `internal/templates` | `make test` (all packages pass) | `go vet ./...`, `go.mod` (zero external dependencies) | Verified |

## 2. Multi-Lens Ensemble Review

### 🔎 Spec Auditor Lens
- [x] Diff implements all requirements (R-001 ... R-007) from requirements.md
- [x] No scope creep: diff modifies ONLY files listed in plan.md Section 5
- [x] Negative and edge cases are explicitly tested and verified in validation.md
- [x] Zero unapproved third-party dependencies added to package manifests

**Spec Auditor Evaluation & Findings:**
- **Requirements Traceability (R-001 through R-007):**
  - **R-001 (Ephemeral Reviewer Lifecycle Standard):** Fully codified in `skills/task-coordinator/SKILL.md` (lines 46–50), `skills/code-review/SKILL.md` (lines 56–72), `internal/templates/data/base/contract.md.tmpl`, `CLAUDE.md`, `GEMINI.md`, and `AGENTS.md`. Explicitly mandates single-loop lifecycles and strictly forbids `send_message` conversation reuse across remediation loops.
  - **R-002 (Zero-Negotiation Loop Isolation):** Enforced via `StageReviewBundleWithLoop` in `internal/audit/bundle.go`, which stages candidate diffs alongside clean, objective finding checklists (`findings_to_verify.md`) with zero developer prose or justification transcripts.
  - **R-003 (ExtractFindingsToVerify & findings_to_verify.md Staging):** Implemented in `internal/audit/bundle.go` via `ExtractFindingsToVerify` and `FormatFindingsToVerify`. Verified by unit tests in `bundle_test.go` (`TestExtractFindingsToVerify` and `TestStageReviewBundle_Loop`).
  - **R-004 (CLI Loop Flag & Handover Prompts):** Added `--loop <1..3>` flag to `sdd prompt review` in `internal/cli/prompt.go`. Generates tabula-rasa remediation handover prompts for loop 2 and 3. Bounded strictly by circuit breaker ($1 \le \text{loop} \le 3$), failing with exit code 2 on invalid values.
  - **R-005 (Mechanical remediation-hygiene Rule):** Implemented in `internal/check/review.go` as `checkReviewRemediationHygiene`. Flags conversational chatter/developer excuses in active tasks; ignores completed `done` tasks to prevent retroactive failures. Verified in `review_test.go` (`TestCheckRemediationHygiene`).
  - **R-006 (Agent Contracts & Version Bump):** Embedded contract template `internal/templates/data/base/contract.md.tmpl` and active contracts (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`) updated to `templates=2.6.0`. `templates.Version` bumped to `2.6.0` in `internal/templates/templates.go` and `.sdd/manifest.json`. Tested via `TestVersion` and `TestFreshKitHasNoErrors`.
  - **R-007 (Standard Library Purity):** 100% Go standard library implementation (`D-007`); zero third-party dependencies added to `go.mod`.
- **Plan Boundary Confinement:**
  - Git status check confirms that 100% of modified and created files correspond strictly to the approved file list in Section 5 of `plan.md`. Zero plan drift.
- **Negative & Boundary Test Coverage:**
  - Automated tests cover invalid loop bounds (`--loop 0`, `--loop 4`), malformed and placeholder findings rows in review.md, clean vs chatter review parsing, and done task filtering.
- **Findings Count:** 0 Blockers, 0 Warnings, 0 Minors.
- **Outcome:** Approved (Spec Auditor Lens)

### 🛡️ AppSec Specialist Lens
- [x] Input validation: bound sizes on all external inputs; fail-safe on ambiguity
- [x] Confinement: filesystem operations confined (os.Root / clean paths; no traversal)
- [x] Secrets hygiene: zero credentials, tokens, or PII exposed; synthetic fixtures only
- [x] Execution safety: commands use separate arguments; no string-concatenated shell execution

**AppSec Specialist Evaluation & Findings:**
- **Threat Surface & Input Validation:**
  - `sdd prompt review <T-xxx> --loop <1..3>` strictly bounds the `--loop` flag to integers 1 through 3, rejecting anything else (`< 1` or `> 3`) with code 2 and usage errors.
  - `StageReviewBundleWithLoop(root, taskID, lens, loop)` enforces the circuit breaker bounds ($1 \le \text{loop} \le 3$) at the Go API boundary, failing fast with an error.
  - `ExtractFindingsToVerify` parses lines using `bufio.NewScanner` with bounded line checks and skips malformed or placeholder rows.
  - `FormatFindingsToVerify` escapes pipe characters (`\|`) and flattens newlines to prevent markdown table injection or syntax corruption.
- **Zero-Negotiation Isolation as Prompt Injection Defense:**
  - Removing developer rationales, excuses, and conversational negotiation transcripts from remediation review bundles eliminates "justification bleed" and prevents prompt injection/social engineering of reviewer LLMs.
- **Filesystem Confinement & Traversal Prevention:**
  - Task IDs are validated with `^T-\d{3,}$` regex, completely disallowing path traversal characters (`..`, `/`, `\`).
  - Staging uses `os.OpenRoot(root)` via `fsutil.WriteAtomicRoot`, confining all bundle writes within the root repository.
  - Symlinks in task folders are detected and rejected (`fi.Mode()&os.ModeSymlink != 0`).
  - Staged files use restrictive mode `0o644`.
  - `.sdd/review` is ensured in `.gitignore` to prevent committing frozen bundles.
- **Secrets & Path Hygiene:**
  - Zero credentials, secrets, or real PII.
  - Rule F-002: Zero developer-specific local absolute paths (/Users/..., /home/...).
- **Command Injection Safety:**
  - Prompts generate structured Markdown instructions; no external shell execution with string concatenation.
- **Standard Library Purity:**
  - Zero external dependencies in `go.mod` (Decision `D-007`).
- **Findings Count:** 0 Blockers, 0 Warnings, 0 Minors.
- **Outcome:** Approved (AppSec Specialist Lens)

### 📐 Pattern & Quality Auditor Lens
- [x] Complies with approved decisions in TECH_STACK.md (D-049, D-007)
- [x] Complies with active feedback rules in SKILLS.md §8 (F-001, F-002, F-003)
- [x] Deterministic by construction (injected clocks, fixed random seeds, canonical map order)
- [x] Language standards: Effective Go conventions, context.Context first, clean linting (make lint)
  - Evaluated D-049: Ephemeral Reviewer Lifecycle and Zero-Negotiation Loop Isolation are fully codified across contract templates (CLAUDE.md, GEMINI.md, AGENTS.md), coordinator skills, and embedded template data.
  - Evaluated D-007: Standard library purity preserved; zero third-party dependencies added to go.mod.
  - Determinism verified: time parameter properly injected via `env.now()`, sorting and table generation deterministic, `go test -count=2` passes identically.
  - Active feedback rules: F-001 (0 Blockers), F-002 (zero personal absolute paths verified), F-003 (token telemetry tracked).
  - Code hygiene: `go vet` clean, `make verify` clean, race detector (`-race`) clean with zero data races.

## 3. Risk & Findings Register

| ID | Category | Severity | Finding | Suggested Action | Status |
|---|---|---|---|---|---|
| (none) | | | | | |

### Severity Guidelines:
- 🔴 **Blocker (Developer Attention Needed):** Spec violation, security hazard, test gap, plan drift. MUST be resolved before GATE 2 commit.
- 🟡 **Warning (Attention Suggested):** Sub-optimal pattern, edge case risk, missing doc or comment.
- 🟢 **Minor (Safe / Informational):** Style, minor naming or micro-optimization.

## 4. Reusable Knowledge & Feedback Synthesis
- [x] Did any review finding reveal a general pattern? Ephemeral tabula-rasa spawning combined with zero-negotiation checklists structurally prevents justification bleed across iterative review loops.

## 5. Review Sign-off
Outcome: Approved
Notes: Full Review Triad sign-off complete. Spec Auditor, AppSec Specialist, and Pattern & Quality Auditor lenses all certify 0 Blockers, 0 Warnings, 0 Minors. Rule F-001 satisfied. Ready for GATE 2 human review.
