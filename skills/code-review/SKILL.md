---
name: code-review
description: Deterministic code review rubric, progressive context ingestion, 3-tier risk grading (Blocker, Warning, Minor), and 3-subagent parallel review triad standards (Spec Auditor, AppSec Specialist, Pattern Auditor) for AI coding agents.
---

# SDD Code Review Standard

> **ANTHEM: ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**

## Overview & Triggers
Activate this skill whenever:
- Performing code review on a feature branch, staged change set, or pull request.
- Verifying code against approved specifications before Phase D / GATE 2 commit.
- Inspecting code diffs for spec alignment, architectural drift, security hazards, or unhandled errors.

## The 3-Subagent Parallel Review Triad
Every code review in Phase C.2 deploys three independent subagents in parallel, each auditing through exactly one specialized lens:

1. **🔎 Spec Auditor Lens (Scope & Correctness):**
   - **Traceability:** Every requirement (`R-xxx`) in `requirements.md` maps to concrete implementation and test evidence in `validation.md`.
   - **Boundary Enforcement:** Touch ONLY files listed in `plan.md` Section 5. Zero unapproved file modifications or scope creep.
   - **Negative Coverage:** Negative, boundary, and failure cases are explicitly tested and verified.
   - **Standard Library Purity:** Zero unapproved third-party dependencies (`D-007`).

2. **🛡️ AppSec Specialist Lens (Security & Safety):**
   - **Threat Surface & Input Validation:** Bound sizes on all external inputs; parse strictly; treat untrusted input as hostile.
   - **Confinement & Traversal:** Verify filesystem confinement via `os.Root` or clean paths; reject path traversal and escape sequences.
   - **Secrets & Privacy Hygiene:** Zero credentials, tokens, connection strings, or personal data in logs, tests, docs, or code. Synthetic fixtures only.
   - **Command Safety:** Never construct shell commands via string concatenation; maintain separate arguments.
   - **Fail Safe:** Ambiguous or unknown state fails to the most restrictive outcome.

3. **📐 Pattern & Quality Auditor Lens (Architecture & Standards):**
   - **Decision Conformance:** Respects all approved decisions in `TECH_STACK.md` (`D-xxx`).
   - **Determinism:** Inject clocks, random sources, and ID generators; avoid map iteration order dependencies.
   - **Code Quality:** Idiomatic Go (*Effective Go*, *Go Code Review Comments*); `context.Context` first for blocking/IO; `%w` error wrapping; clean linting (`make lint`).
   - **Concurrency Safety:** Verify goroutine lifecycles, race detector compliance, and zero data races.
   - **Continuous Learning:** Active feedback rules in `SKILLS.md` §8 (`F-xxx`) complied with; recurring patterns captured.

## Reviewer Model Tier Calibration & Token Conservation
To balance deep adversarial inspection with economic token discipline, Review Triad subagents are routed to calibrated model tiers (`D-049`, `D-050`):
- **🔎 Spec Auditor (Standard Tier — `Model: "flash"` / `haiku` / `gpt-4o`):** Standard model tier with low reasoning effort. Performs deterministic requirements traceability scanning, omission checks, and boundary enforcement against structured tables.
- **🛡️ AppSec Specialist (Frontier Tier — `Model: "inherit"` / `pro` / `opus` / `o1`):** Frontier model tier with high reasoning effort. Performs deep adversarial threat analysis, taint tracking, filesystem confinement audits, and security edge case discovery.
- **📐 Pattern & Quality Auditor (Standard Tier — `Model: "flash"` / `haiku` / `gpt-4o`):** Standard model tier with low reasoning effort. Performs architectural decision (`D-xxx`) validation, active feedback rule (`F-xxx`) compliance, and Go idiom/style checks.

## Unanchored Requirements Traceability Ledger & Omission-First Review Protocol

To eliminate the LLM "Diff Framing" anchoring bias (where reviewers critique existing code but overlook omissions, edge cases, and missing tests), SDD mandates the **Omission-First Review Protocol** (`D-049`):

### Step 1: Unanchored Traceability Scan (Before reading diff.patch)
1. Open `.sdd/review/<TASK_ID>/<lens>/ledger.md` (or `.sdd/review/<TASK_ID>/code/ledger.md`).
2. For every requirement (`R-xxx`), search the codebase independently to locate:
   - **Code Symbol & File:** The concrete functions, types, and files implementing the requirement.
   - **Positive Test Evidence:** Automated test name proving the happy path.
   - **Negative / Boundary Test Evidence:** Automated test name proving failure, validation, boundary, or gating paths.
3. If ANY requirement lacks code symbols, positive tests, or negative/boundary tests:
   - It is an immediate 🔴 **Blocker** (`Omission`).
4. Transfer the completed ledger into `## 1. Requirements Traceability Ledger` in `tasks/<typology>/T-xxx-<slug>/review.md`.

### Step 2: Incremental Diff Quality Audit (After ledger completion)
Only after every `R-xxx` is accounted for in Step 1, proceed to inspect `diff.patch` against `spec.md` using the respective lens rubrics (Spec, AppSec, Pattern).

## Ephemeral Tabula-Rasa Reviewer Lifecycle & Zero-Negotiation Standard

To eliminate "Repair-Loop Justification Bleed" where developers sway reviewers with conversational rationales or excuses, SDD mandates strict reviewer session ephemerality and zero-negotiation loop isolation (`D-049`):

### 1. Ephemeral Subagent Spawning
- Reviewer subagent conversations MUST NOT be reused across remediation loops. Reusing conversations via `send_message` or carrying conversational transcripts into subsequent loops is strictly forbidden.
- For every review iteration (Loop 1 initial review, Loop 2 & Loop 3 remediation reviews), the Task Coordinator dispatches fresh, independent reviewer subagents (`invoke_subagent`).

### 2. Zero-Negotiation Loop Isolation
- When re-staging review bundles for Loop 2 and 3 via `sdd prompt review <TASK_ID> --loop <N>`, the bundle contains ONLY:
  - The latest candidate diff (`diff.patch`).
  - The unanchored requirements ledger (`ledger.md`) and specifications (`spec.md`).
  - An objective defect verification checklist (`findings_to_verify.md`) generated from previous finding IDs and severities.
- Developer prose, chat transcripts, excuses, or negotiation dialogues are strictly scrubbed from the bundle and prompt.
- The ephemeral reviewer audits the candidate code objectively to determine whether defects are resolved, immune to cognitive anchoring.

## Batch Remediation & Conflict Resolution
- **Single Remediator:** Review subagents are strictly read-only auditors; only the primary worker agent modifies code.
- **Batched Remediation:** When multiple lenses report changes, the primary agent consolidates all findings into a single fix pass, consuming 1 loop in the 3-loop circuit breaker.
- **Conflict Resolution:** Security, safety, and determinism take precedence over convenience. If recommendations fundamentally conflict, the agent halts and issues `DECISION NEEDED: D-xxx` to the human.

## 3-Tier Risk & Finding Rubric

| Severity | Definition | Action Required |
|---|---|---|
| 🔴 **Blocker** | Spec violation, security hazard, test gap, plan deviation, unapproved dependency. | MUST be resolved before GATE 2 commit. |
| 🟡 **Warning** | Sub-optimal pattern, edge case risk, missing doc or comment. | MUST be resolved or explicitly justified before GATE 2 (Rule F-001). |
| 🟢 **Minor** | Style, naming refinement, or non-critical micro-optimization. | MUST be resolved or explicitly justified before GATE 2 (Rule F-001). |

## Rule F-001 Finding Resolution Mandate
- Worker agents MUST resolve all findings (🔴 Blockers, 🟡 Warnings, and 🟢 Minors) reported during review before presenting to the human at GATE 2.
- Any finding not remediated must have an explicit written justification with human approval recorded in `review.md`.
- `review.md` can only achieve `Approved` sign-off when 0 Blockers, 0 unresolved Warnings, and 0 unresolved Minors remain.

## Lens-Scoped Review Bundles & Diff Filtering

To eliminate directory collisions and save 30–70% of redundant context tokens during Review Triad execution (`D-050`):
- **Isolated Bundle Directories:** Review bundles are isolated per lens under `.sdd/review/<TASK_ID>/<lens>/` (`spec/`, `security/`, `pattern/`, and `code/` for all).
- **Domain Diff Filtering:** `diff.patch` contents are deterministically filtered by lens:
  - `security`: Retains executable code, scripts, configs, and dependency files; filters non-executable task markdown and doc changelogs.
  - `pattern`: Retains source code, test files, architecture decisions, and coding standards; filters validation logs and roadmap changelogs.
  - `spec`: Retains source code, test files, and task specifications; preserves file headers for boundary checking while eliding bloated doc diff bodies.
- **Finding Isolation:** Remediation checklists (`findings_to_verify.md`) filter findings by lens domain (`SPEC-`, `SEC-`, `PAT-` / `PQ-`).

## 3-Loop Circuit Breaker
- Maximum 3 review-fix iterations (1 initial audit + up to 2 fix attempts) between worker agent and reviewer triad.
- If Blockers persist after attempt 3, halt immediately, mark the task `Blocked`, and escalate to the human.

## Review Output Format
Record all findings in `tasks/<typology>/T-xxx-<slug>/review.md` following the standard template. If any 🔴 Blocker remains, or if any 🟡 Warning or 🟢 Minor remains unresolved without explicit justification, the task CANNOT proceed to GATE 2.
