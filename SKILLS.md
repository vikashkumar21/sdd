# SKILLS — sdd

> **ALWAYS BE DETERMINISTIC. Human approval resolves every decision.**

Status: living document · Last updated: 2026-10-26

**What this file is.** The standards, techniques, and checklists that the agent *and* the human apply while building sdd. It is a working playbook, not a wish list: add a rule when you learn one, and record pitfalls as you hit them. Items marked **(verify)** are reported but not yet confirmed. **Human feedback that applies generally is recorded in §8 (Human feedback rules) and is binding on all later work.**

## 1. Language standards

The packs below are a baseline written by `sdd`. Review and refine them; changes are ordinary documentation changes and follow the gates.

### Go

- **Formatting and imports:** `gofmt` and `goimports` clean; `go vet` clean; linter per the Decision register.
- **Style references:** *Effective Go*, *Go Code Review Comments*, the Google Go Style Guide. Idiomatic over clever.
- **Naming:** short, clear, no stutter (`audit.Log`, not `audit.AuditLog`); package names are lowercase, singular, no underscores.
- **Interfaces:** small, defined in the consumer package; accept interfaces, return concrete types; no interface until there are two implementations or a test seam is needed.
- **Structure:** everything not meant as public API goes in `internal/`; no package-level mutable state; no `init()` side effects; constructors take their dependencies explicitly.
- **Errors:** wrap with `fmt.Errorf("doing X: %w", err)`; lowercase messages without trailing punctuation; sentinel or typed errors (`errors.Is` / `errors.As`) for anything a caller must branch on; no `panic` outside `main` and tests; never ignore an error silently.
- **Context and concurrency:** `context.Context` is the first parameter of anything that does I/O or can block; never store a context in a struct; every goroutine has an owner, a stop condition, and error propagation; prefer synchronous code unless a measured need exists.
- **Determinism:** inject a clock and an ID generator; never call `time.Now()` or a random source inside decision logic; never rely on map iteration order.
- **Tests:** table-driven with named cases; golden files for large expected outputs; `t.Parallel()` only when provably safe; integration tests behind a build tag.
- **Comments:** every exported identifier has a doc comment starting with its name; comment *why*, not *what*.

## 2. Testing standards

- **Table-driven or parametrised** tests with named cases; failures print inputs and a readable diff.
- **Golden files** for large expected outputs, reviewed by the human whenever they change. A change to expected output is a decision, not a test fix.
- **Failure and negative cases are mandatory:** invalid, empty, oversized, unknown, and malformed input; not only the happy path.
- **Determinism:** run tests twice and compare; no dependence on time, randomness, map or set ordering, or the environment.
- **Integration tests** run only against local or throwaway resources, and the harness verifies the target before running anything with side effects.
- Fixtures are synthetic; no real data or credentials.
- Record the exact commands and results in `validation.md`; "should pass" is not evidence.

## 3. Security and privacy hygiene

- Secrets are never committed, logged, or pasted into chat. Configuration stores the *name* of an environment variable or secret, never the value.
- Logs contain no credentials and no personal data.
- Treat all external input as hostile: bound sizes, parse strictly, never build queries or commands by string concatenation.
- Least privilege everywhere: separate credentials per environment, and tests and tools never touch production.
- New dependencies go through the decision protocol (purpose, licence, maintenance, alternatives).

## 4. Spec-driven authoring skills

**Requirements (`requirements.md`)**
- One requirement per row, ID `R-xxx`, phrased "The system SHALL …". Atomic, testable, no vague words ("fast", "robust") unless measurable.
- Each has a verification method (test / inspection / demo). Include explicit non-goals.
- Acceptance criteria in Given / When / Then form where behaviour is involved.
- `Open questions` must be empty before GATE 1.

**Plan (`plan.md`)**
- Steps small enough to review in one sitting; each names the files and the check that proves it.
- List new or affected decisions with options and a recommendation; list new dependencies with justification.
- List every living doc that will change, plus risks and rollback.

**Validation (`validation.md`)**
- A traceability matrix: R-xxx → method → exact command or test → expected → actual → status.
- Evidence means a real command and a real result. Anything not verified is labelled **NOT VERIFIED**, with the reason.

## 5. Review checklists

**Human at GATE 1** — Do the requirements say what I want, and only that? Are all open decisions resolved? Is every file in the plan necessary? Is any dependency avoidable? Are the applicable feedback rules listed?

**Human at GATE 2** — Does the diff touch only planned files? Do the tests include failure and negative cases? Any secrets or real data? Was the determinism check run? Are the living docs updated? Is the suggested commit message accurate? Is compliance with every active feedback rule recorded?

## 6. Human skill map and learning log (human fills in)

Use this to decide where a spike is needed (see the ROADMAP's unknown-technology protocol). Familiarity: Strong / OK / New.

| Area | Needed in | Familiarity | Planned learning action |
|---|---|---|---|
| Go standard library and idioms | all | | |
| Go testing (table-driven, golden files, fuzzing) | tests | | |

## 7. Pitfalls log (append-only; newest at the bottom)

| Date | Pitfall | Source / task |
|---|---|---|
| 2026-10-26 | `/bin/sh` on Debian and Ubuntu is dash: no brace expansion (`{a,b}`) and no process substitution (`<(...)`) in scripts or Makefile recipes; use POSIX sh or call bash explicitly | Build session; T-001 |
| 2026-10-26 | A manifest hash must cover exactly what it is compared against: hashing the managed block but labelling the entry as a whole file made every fresh kit look edited | Found by a test; T-001 |
| 2026-10-26 | Example IDs in documents (for example `APPROVED T-004`) must exist in every generated roadmap, or `sdd check` reports a dangling reference; use IDs that always exist | Found by `sdd check`; T-001 |
| 2026-10-26 | A hand-written README with an example task ID that is not in the roadmap fails `sdd check`; point examples at real tasks | Found by `sdd check`; T-001 |
| 2026-10-26 | Running init on a hand-made kit with no manifest would append a duplicate contract to agent files that have no managed block; found by a dry run on the real dbguard kit; init now refuses | Found by a dry run; T-001 |
| 2026-10-26 | Table cells must not contain a literal pipe character (even inside backticks); validate user-supplied values (names, commands) before rendering them into tables | Design; T-001 |
| 2026-11-02 | In T-008, markdown parser now supports escaped pipes and pipes inside inline code spans without breaking table column validation | Robust parser; T-008 |
| 2026-11-21 | Single-agent brownfield discovery suffers from confirmation bias and misses unreferenced entrypoints or ambient dependencies; dispatching an adversarial discovery-challenger under tabula-rasa isolation uncovers hidden blind spots | Brownfield discovery; T-053 |

## 8. Human feedback rules (binding; grows over time)

Human feedback is how this playbook improves. The rules registered below are **binding on every later task**.

### Process

1. **Classify** every piece of human feedback on a plan, implementation, or code:
   - **General**: would apply to future code or tasks (style, structure, testing, naming, security, workflow). Record it here as an `F-xxx` rule.
   - **Task-specific**: only about this task. Apply it and note it in that task's `validation.md`. Not recorded here.
   - **Unclear**: ask the human (decision protocol): "general or task-specific?" Never guess.
2. **Record** a general rule in the same change set: ID, date, source (task and gate), the rule in one or two imperative sentences, where it applies, and *how it is verified* (a lint rule, a test, a checklist item, or manual review). The human reviews the wording at GATE 2.
3. **Apply immediately:** fix the current task's work and sweep existing code for violations; list what changed.
4. **Qualify every later task:** each `plan.md` lists the active F-rules that apply and how it complies; each `validation.md` records compliance per rule (pass / fail / n.a.). A task cannot reach `Awaiting review` with a failing rule.
5. **Conflicts:** if a new rule conflicts with an existing rule, decision, or requirement, STOP and report. The human decides; the older rule is marked `Superseded` (never deleted).
6. **Promote to tooling** where possible: enforce a rule through the linter, a test, or a build check instead of relying on memory.

### Register

| ID | Date | Rule | Applies to | Source | Verified by | Status |
|---|---|---|---|---|---|---|
| F-001 | 2026-11-10 | Worker agents must resolve all findings (Blockers, Warnings, and Minors) reported during review before presenting to human at GATE 2; any retained finding not remediated must be explicitly justified in review.md | All tasks (code review, plan review, GATE 2) | Human feedback at T-033 / T-035 (D-032) | sdd check review validation, subagent review triad, GATE 2 checklist | Active |
| F-002 | 2026-11-20 | Public documentation, examples, and generated templates must never contain developer-specific local absolute filesystem paths (/Users/..., /home/...); all commands and paths must use repository-relative paths or generic clone URLs | Public documentation, examples, templates, scripts, living docs | Human feedback / External agent review | Path hygiene check, inspection, sdd check | Active |
| F-003 | 2026-11-21 | Task Coordinators and worker agents must track session token consumption, record telemetry via `sdd task state record-tokens <T-xxx>`, document metrics in `validation.md`, and report token usage at GATE 2 | All tasks (task completion, validation.md, GATE 2) | Human feedback after T-053 | sdd task state show, validation.md token table, GATE 2 checklist | Active |

## 9. Agent Skill Authoring and Refinement Standard (agentskills.io)

Skills follow the Linux Foundation / Anthropic `agentskills.io` open standard adopted in Decision D-024.

### 1. Directory Structure and Frontmatter
- Location: `skills/<name>/SKILL.md` (project-facing) or `.sdd/skills/<name>/SKILL.md` (internal/infrastructure).
- Directory match: `<name>` must match parent folder name and satisfy regex `^[a-z0-9]+(?:-[a-z0-9]+)*$` (1–64 characters).
- Frontmatter (YAML):
  - `name`: Matches folder name.
  - `description`: 1–1024 characters; explains *what* the skill does and *when/why* an agent should trigger it.
  - `compatibility` (optional): Max 500 characters.

### 2. Body Structure (Progressive Disclosure)
1. `## Overview & Triggers` — Specific trigger conditions and intent.
2. `## Step-by-Step Instructions` — Idiomatic, deterministic, phase-by-phase workflow.
3. `## Verification` — Concrete checks or tests to confirm correctness.
4. `## Troubleshooting` — Edge cases, pitfalls, and fixes.

### 3. Agent Refinement of Raw Ingested Notes
When skills are created from raw text or files via `sdd skill new --from-file` or `--from-text`, `sdd` encloses raw notes in:
```markdown
<!-- sdd:skill-refine:begin -->
... raw notes ...
<!-- sdd:skill-refine:end -->
```
Any agent taking over development reads these notes and synthesizes them into the standard sections above under Gate 1 / Gate 2 review.

### 4. Programmatic Agent SDK Integration
Skills exported via `sdd skill export` or authored in `skills/` conform to the Linux Foundation / Anthropic open standard and can be mounted directly into programmatic agent applications:
- **SDK Directory Loaders:** Point native SDK skill loaders (e.g., Antigravity Python SDK, Anthropic Agent SDK) at the skill directory:
  ```python
  from antigravity.skills import SkillLoader
  sdd_skill = SkillLoader.from_directory("./skills/sdd-protocol")
  agent = Agent(skills=[sdd_skill], tools=[...])
  ```
- **System Prompt Seeding:** In frameworks without native skill directory loaders (LangGraph, raw Gemini/OpenAI SDKs), ingest the skill markdown directly into the system prompt:
  ```python
  with open("skills/sdd-protocol/SKILL.md") as f:
      agent.system_prompt += "\n" + f.read()
  ```
- **Gating Enforcement:** SDK agents operating under `sdd-protocol` must enforce GATE 1 programmatically—refusing to invoke file-modifying or code-writing tools until the human sends the exact `APPROVED T-xxx` token.

## 10. Code Review and Verification Standard

Code review in SDD is an independent quality and risk audit conducted before Phase D / GATE 2 commit. It ensures that changes satisfy specifications without regressions, architectural drift, or security vulnerabilities.

### 1. The Ensemble Review Triad
Review audits are conducted through three specialized lenses:
1. **🔎 Spec Auditor Lens (Correctness & Scope):**
   - **Traceability:** Every requirement (`R-xxx`) in `requirements.md` maps to concrete implementation and test evidence in `validation.md`.
   - **Boundary Enforcement:** Touch ONLY files listed in `plan.md` Section 5. Zero unapproved file modifications or scope creep.
   - **Negative Coverage:** Negative, boundary, and failure cases are explicitly tested and verified.
2. **🛡️ AppSec Specialist Lens (Security & Safety):**
   - **Threat Surface & Input Validation:** Bound sizes on all external inputs; parse strictly; treat untrusted input as hostile.
   - **Confinement & Traversal:** Verify filesystem confinement via `os.Root` or clean paths; reject path traversal and escape sequences.
   - **Secrets & Privacy Hygiene:** Zero credentials, tokens, connection strings, or personal data in logs, tests, docs, or code. Synthetic fixtures only.
   - **Command Safety:** Never construct shell commands via string concatenation; maintain separate arguments.
   - **Fail Safe:** Ambiguous or unknown state fails to the most restrictive outcome.
3. **📐 Pattern & Quality Auditor Lens (Architecture & Standards):**
   - **Decision Conformance:** Respects all approved decisions in `TECH_STACK.md` (`D-xxx`).
   - **Standard Library Purity:** Zero unapproved third-party dependencies (`D-007`).
   - **Determinism:** Inject clocks, random sources, and ID generators; avoid map iteration order dependencies.
   - **Code Quality:** Idiomatic Go (*Effective Go*, *Go Code Review Comments*); `context.Context` first for blocking/IO; `%w` error wrapping; clean linting (`make lint`).
   - **Continuous Learning:** Active feedback rules in `SKILLS.md` §8 (`F-xxx`) complied with; recurring patterns captured.

### 2. 3-Tier Risk Rubric
- 🔴 **Blocker (Developer Attention Needed):** Spec violation, security hazard, test gap, plan drift, unapproved dependency. MUST be resolved before GATE 2 commit.
- 🟡 **Warning (Attention Suggested):** Sub-optimal pattern, edge case risk, missing doc or comment.
- 🟢 **Minor (Safe / Informational):** Style, naming refinement, or non-critical micro-optimization.

### 3. Review Protocol & Circuit Breaker
- **Independent Auditor Required:** The primary author agent is strictly forbidden from self-approving its own `review.md`. An independent subagent reviewer must certify `review.md` with 0 Blockers before Phase D close.
- **3-Loop Circuit Breaker:** At most 3 review-fix iterations (1 initial audit + up to 2 fix attempts) between author and reviewer. If Blockers persist after 3 iterations, the agent MUST halt, mark the task `Blocked`, and escalate to the human.
- **Review Artifact (`review.md`):** Review findings are recorded in `tasks/features/T-xxx-<slug>/review.md` (or typology equivalent under `tasks/`). If any unresolved 🔴 Blocker remains, `sdd check` flags an error and the task CANNOT proceed to GATE 2 commit.

## 11. Adversarial Brownfield Discovery Ensemble Standard

When exploring an existing or unfamiliar codebase, single-agent analysis suffers from cognitive anchoring and confirmation bias—overlooking ambient dependencies, hidden background routines, or schema mismatches. SDD enforces an adversarial multi-agent discovery ensemble protocol (`T-053`, `D-045`):

### 1. Two-Tier Role Dispatch & Review Isolation
- **Primary Analyst (`codebase-analyst`):** Conducts 4-pillar forensic extraction (AST/topology, UI/styling, data/state models, and technical debt), synthesizing initial drafts of `TECH_STACK.md`, `ROADMAP.md` Parking lot, and `SKILLS.md`.
- **Adversarial Challenger (`discovery-challenger`):** Dispatched tabula-rasa into a fresh context backed by an isolated review bundle in `.sdd/review/<TASK_ID>/discovery/` containing `findings.md`, `rubric.md`, and `manifest.json`. The challenger operates without conversational anchoring to the primary analyst.

### 2. 7 Blind-Spot Audit Vectors
The discovery challenger systematically interrogates the codebase against 7 comprehensive vectors:
1. **Vector 1: Hidden Entry Points & Triggers:** CLI flags/subcommands, background daemon loops, cron/schedulers, queue workers, webhooks, and event consumers unreferenced in primary entrypoints.
2. **Vector 2: Concurrency, Lifecycle & Shutdown Invariants:** Unbounded goroutines/threads, missing cancellation propagation, SIGINT/SIGTERM drain hooks, package-level mutable state, race conditions in init.
3. **Vector 3: Data Boundary Invariants & Schema Drift:** Unhandled database errors, uncommitted/unrolled-back transactions, raw queries bypassing ORM/query builders, schema migration mismatches, in-memory cache desync.
4. **Vector 4: Dynamic Dispatch & Runtime Feature Flags:** Reflection-based routing, plugin systems, runtime feature flags, environment-conditional code paths invisible to static call graphs.
5. **Vector 5: Orphan & Dead Code:** Dangling configurations, abandoned routes/endpoints, unexported/unreferenced symbols, unused dependencies.
6. **Vector 6: Implicit External & Ambient Dependencies:** Undocumented env vars, ambient binaries/system tools assumed to exist on the host, undeclared cloud SDK defaults, unrecorded third-party APIs.
7. **Vector 7: Architectural & Documentation Contradictions:** Direct discrepancies between existing README/architecture docs and actual runtime code behavior, stale comments, unverified assumptions.

### 3. Deterministic Reconciliation Protocol (Gate 1 Prerequisite)
Before presenting specifications to the human at Gate 1:
1. Cross-reference primary analyst AST map with challenger-discovered entry points and async event consumers.
2. Reconcile database models and data flows against migration history and raw queries.
3. Ensure all implicit external dependencies, tools, and env vars are registered in `TECH_STACK.md` §8 and configuration schemas.
4. Sweep codebase for dead code, abandoned endpoints, and dangling config keys, logging candidates into `ROADMAP.md` §7 Parking Lot.
5. Verify that existing documentation discrepancies are resolved in living documents prior to Gate 1 approval.

## 12. Visual Mermaid Diagram and Domain Flow Documentation Standard

Visual architecture and domain flows make software topologies, data schemas, and state transitions immediately comprehensible to human maintainers and AI agents alike. SDD codifies native Mermaid documentation standards (`T-054`, `D-046`):

### 1. Document Structure & Scaffolding
- **Architecture Overview (`docs/architecture/overview.md`):** System topology, component boundaries, entry points, and storage layers documented via Mermaid `flowchart TD` / `flowchart LR`.
- **Data Models (`docs/architecture/data_model.md`):** Entity relationship diagrams via Mermaid `erDiagram`, detailing primary/foreign keys, attributes, and cardinality (`||--o{`, `||--||`, etc.).
- **Domain Flow Catalog (`docs/business_flows/overview.md`):** Table mapping core business capabilities to dedicated flow documents.
- **Specific Domain Flows (`docs/business_flows/<name>.md`):** Detailed transaction flows containing sequence diagrams (`sequenceDiagram`) with `autonumber`, activation bars, alt/opt error branches, and entity state lifecycles (`stateDiagram-v2`).
- **Scaffolding Ergonomics:** Use `sdd doc init` to scaffold standard architecture and domain flow directories, and `sdd doc flow <name> [--title <title>]` to add new domain flows.

### 2. Mermaid Formatting Rules
- **Header Declarations:** The first non-comment line inside a ` ```mermaid ` fence must declare a recognized diagram type (`flowchart`, `graph`, `sequenceDiagram`, `erDiagram`, `stateDiagram-v2`, `classDiagram`, etc.).
- **Special Characters in Labels:** Always quote node labels containing parentheses, brackets, or special characters (e.g. `id["Label (extra info)"]`, `db[("User Store [PostgreSQL]")]`).
- **Comments & Cleanliness:** Use `%%` for inline comments. Keep diagrams focused on single bounded contexts or primary flows; decompose oversized flows into child diagrams.
- **Mechanical Validation:** `sdd check` enforces `mermaid-syntax` verification across all repository markdown files, validating diagram headers, non-empty blocks, and delimiter balancing.

## 13. Forensic Brownfield Coverage Ledger and Unanchored Challenger Standard

Brownfield discovery on non-trivial polyglot repositories easily overlooks critical libraries, legacy runtime quirks, and unreferenced native binaries if an analyst only skims top-level manifests or if a challenger is anchored on the analyst's output (`T-056`, `D-048`).

### 1. Mandatory Coverage Ledger Table
Every brownfield discovery report must include a `## Coverage Ledger` table recording every manifest, build file, configuration, and image specification inspected:

| Manifest / Config File | Total Lines | Lines Read | Lines Skipped | Reason for Skip | Fully Verified (Yes/No) |
|---|---|---|---|---|---|
| pom.xml | 240 | 1-240 | 0 | None | Yes |
| libraries.gradle | 650 | 1-120 | 121-650 | Internal test dependencies only | Yes |
| Dockerfile | 45 | 1-45 | 0 | None | Yes |

- **Complete Line Accounting:** For every file, `Lines Read` + `Lines Skipped` must account for `Total Lines`.
- **Explicit Justifications:** Any non-zero `Lines Skipped` requires a non-empty, actionable explanation in `Reason for Skip`. Blanket skipping without justification fails `sdd check` under rule `coverage-ledger`.
- **Full Verification Check:** `Fully Verified (Yes/No)` explicitly signals whether all declared dependencies, plugins, and configurations were cross-checked.

### 2. Unanchored Challenger Inventory-First Protocol
The `discovery-challenger` auditor must never begin by reading the analyst's findings:
1. **Step 1: Unanchored Independent Inventory:** Before opening `findings.md`, scan the project root to independently inventory all build manifests (`pom.xml`, `build.gradle*`, `package.json`, `Cargo.toml`, `go.mod`, `pyproject.toml`), container definitions (`Dockerfile*`, compose files), CI/CD pipelines, and vendor/native binaries.
2. **Step 2: Inventory Diff against Coverage Ledger:** Compare the independent inventory against the primary analyst's `## Coverage Ledger`. Flag any missing manifests, partial line reads, or unjustified skips as findings.
3. **Step 3: Staged Reference Context Cross-Check:** Check `.sdd/review/<TASK_ID>/discovery/context/` for staged external reference packs (e.g. migration guides, target platform specs, stakeholder requests) and verify that the codebase reality aligns with or addresses all referenced items.

### 3. Scratchpad Citation Hygiene Guard
All tracked markdown documentation (`MISSION.md`, `TECH_STACK.md`, `ROADMAP.md`, `SKILLS.md`, `README.md`, and task folders) must cite persistent repository paths or commands. Citing ephemeral scratchpad paths (`scratch/`, `/tmp/`, `.sdd/tmp/`, `.system_generated/`) violates Rule `citation-hygiene` in `sdd check`.

## 14. Hard Git Commit Gate and Repository Hook Architecture

SDD enforces the "Human Only Commits" rule (Hard Rule 2) mechanically through repository Git hooks provisioned via `sdd hook install` (`T-051`, `D-043`):

### 1. Branch-Aware Commit and Merge Policies
- **Protected Branches (`main`, `master`):**
  - Direct autonomous agent commits and unverified direct merges are strictly blocked.
  - Interactive commits and merges by human maintainers require confirmation via `/dev/tty` (`[sdd hook] Direct commit/merge to main requires human approval. Proceed? (y/N)`).
  - Automated CI or programmatic scripts must supply explicit authorization token `SDD_HUMAN_APPROVED=1`.
  - Non-interactive sessions without the authorization token exit with code 1.
- **Task Branches (`task/*`, `feature/*`, `bugfix/*`, `hotfix/*`, worktrees):**
  - Autonomous commits by worker agents are permitted per Hard Rule 13 to support iterative development and test saves.

### 2. Gate 1 Discipline Enforcement (`sdd hook check-gate1`)
- On task, feature, bugfix, and hotfix branches, the pre-commit hook runs `sdd hook check-gate1 --branch "$BRANCH"`.
- If staged files include application source code outside `tasks/` (e.g. `*.go`, `*.py`, `*.java`, `*.ts`, `cmd/`, `internal/`), the checker verifies that the corresponding task `plan.md` has been approved (`Status: Approved` or `Status: Implementing`).
- Staging application code prior to Gate 1 approval aborts the commit with a Gate 1 violation.

### 3. Mechanical Kit Consistency Audit
- During `git commit`, the pre-commit hook runs `sdd check --strict`.
- If consistency checks (contract parity, link resolution, table syntax, coverage ledger) fail, the commit is aborted.
- Humans can bypass mechanical checks in emergencies via `git commit --no-verify` or `SDD_SKIP_CHECK=1`.

### 4. Git Worktree Compatibility & Chaining Lifecycle
- **Dynamic Path Resolution:** Hooks query `git rev-parse --git-path hooks` to resolve the active hooks directory across isolated worktrees (`.sdd/worktrees/`) where `.git` is a gitdir reference file.
- **Non-Destructive Chaining:** Pre-existing foreign hooks are moved to `*.sdd-orig` and chained via `exec "$ORIG_HOOK" "$@"`. Re-installing SDD hooks is strictly idempotent and never overwrites existing `.sdd-orig` backups.
- **Clean Uninstallation:** `sdd hook uninstall` removes SDD hooks and restores original hooks. Refuses to delete unrecognized foreign hooks without `--force`.
- **Coordinator Merge Integration:** `sdd task merge` qualifies merges and injects `SDD_HUMAN_APPROVED=1` into `git merge`, enabling turnkey merge qualification into `main`.
