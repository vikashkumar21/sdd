---
status: draft
---
# Code Review — T-061 Lens-scoped review bundles and diff filtering

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Pending | Changes requested | Approved
Reviewer: <Agent Name / Human>
Commit / Diff Ref: <HEAD / diff summary>

## 1. Requirements Traceability Ledger

| Req ID | Requirement Statement | Code Symbol & File | Positive Test Evidence | Negative / Boundary Test Evidence | Status |
|---|---|---|---|---|---|
| R-001 | The system SHALL stage review bundles in isolated directories per lens: .sdd/review/<TASK_ID>/spec/ for spec, .sdd/review/<TASK_ID>/security/ for security, .sdd/review/<TASK_ID>/pattern/ for pattern, and .sdd/review/<TASK_ID>/code/ for all. | pending | pending | pending | pending |
| R-002 | The system SHALL implement a deterministic diff filtering engine (FilterGitDiffByLens) that tailors diff.patch contents to the specific audit focus of each lens. | pending | pending | pending | pending |
| R-003 | The system SHALL filter remediation findings in findings_to_verify.md by lens domain (FilterFindingsForLens) during multi-loop reviews (Loop > 1). | pending | pending | pending | pending |
| R-004 | The system SHALL record diff filtering telemetry in manifest.json including raw_diff_bytes, filtered_diff_bytes, diff_savings_percent, raw_files_count, and filtered_files_count. | pending | pending | pending | pending |
| R-005 | The system SHALL update sdd prompt review to output prompts and JSON context pointing to the lens-scoped bundle directory, bundle files, and filtering telemetry. | pending | pending | pending | pending |
| R-006 | The system SHALL update skills/code-review/SKILL.md, skills/task-coordinator/SKILL.md, and embedded templates to document lens-scoped review bundles, and bump templates.Version to 2.9.0. | pending | pending | pending | pending |
| R-007 | The system SHALL maintain 100% Go standard library purity (D-007), execute race-clean (-race), and pass determinism verification (-count=2). | pending | pending | pending | pending |

## 2. Multi-Lens Ensemble Review

### 🔎 Spec Auditor Lens
- [ ] Diff implements all requirements (R-001 ... R-007) from requirements.md
- [ ] No scope creep: diff modifies ONLY files listed in plan.md Section 5
- [ ] Negative and edge cases are explicitly tested and verified in validation.md
- [ ] Zero unapproved third-party dependencies added to package manifests

### 🛡️ AppSec Specialist Lens
- [ ] Input validation: bound sizes on all external inputs; fail-safe on ambiguity
- [ ] Confinement: filesystem operations confined (os.Root / clean paths; no traversal)
- [ ] Secrets hygiene: zero credentials, tokens, or PII exposed; synthetic fixtures only
- [ ] Execution safety: commands use separate arguments; no string-concatenated shell execution

### 📐 Pattern & Quality Auditor Lens
- [ ] Complies with approved decisions in TECH_STACK.md (D-xxx)
- [ ] Complies with active feedback rules in SKILLS.md §8 (F-xxx)
- [ ] Deterministic by construction (injected clocks, fixed random seeds, canonical map order)
- [ ] Language standards: Effective Go conventions, context.Context first, clean linting (make lint)

## 3. Risk & Findings Register

| ID | Category | Severity | Finding | Suggested Action | Status |
|---|---|---|---|---|---|
| (none) | n/a | 🟢 Minor | Initial review pending implementation | Complete Phase B implementation | Open |

### Severity Guidelines:
- 🔴 **Blocker (Developer Attention Needed):** Spec violation, security hazard, test gap, plan drift. MUST be resolved before GATE 2 commit.
- 🟡 **Warning (Attention Suggested):** Sub-optimal pattern, edge case risk, missing doc or comment.
- 🟢 **Minor (Safe / Informational):** Style, minor naming or micro-optimization.

## 4. Reusable Knowledge & Feedback Synthesis
- [ ] Did any review finding reveal a general pattern?
  - If general: record an F-xxx rule in SKILLS.md §8 (Human feedback rules) or scaffold a skill via `sdd skill new`.
  - If task-specific: resolved in this change set.

## 5. Review Sign-off
Outcome: Pending
Notes: Initial skeleton for Phase C.2 independent audit.
