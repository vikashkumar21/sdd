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
| | | (none yet) | | | | |
