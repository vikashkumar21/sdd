# CLAUDE.md — sdd agent contract (Claude)

<!-- sdd:begin contract templates=1.0.0 -->
> **ANTHEM: ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**

This is the operating contract for AI agents working on `sdd`. The rules are identical in every agent file (CLAUDE.md, GEMINI.md, AGENTS.md); only the title line differs. If you change a rule in one file, mirror it in the others in the same change set (§9). `sdd check` verifies that they match.

---

## 1. Read first — every session, in this order

1. `MISSION.md` — why this exists, scope, principles, non-goals
2. `TECH_STACK.md` — stack, architecture, the Decision register (D-xxx), the verified-facts log
3. `ROADMAP.md` — milestones, tasks (T-xxx), statuses, parking lot
4. `SKILLS.md` — standards, playbooks, and the Human feedback rules (F-xxx)
5. `README.md` — current public-facing state
6. If a task is active: `features/T-xxx-<slug>/requirements.md`, `plan.md`, `validation.md`

Start every session by stating: the active task (or "none"), the files you read, and its current status in `ROADMAP.md`. If two files contradict each other, STOP and report the conflict. Do not pick a winner.

## 2. The workflow (strict — no phase may be skipped or reordered)

| Phase | What the agent does | Output | Gate |
|---|---|---|---|
| A. Specify | Create `features/T-xxx-<kebab-slug>/` from `features/_templates/` (`sdd feature new T-xxx <slug>` does this mechanically). Draft `requirements.md`, `plan.md`, `validation.md`. **No code. No changes outside this folder.** | Three drafted docs | **GATE 1** |
| B. Implement | Execute `plan.md` step by step. Touch only files listed in the plan. | Code and tests | none |
| C. Verify | Run every check in `validation.md`; record the real commands and real results. | Filled `validation.md` | none |
| D. Close | Update the living docs listed in the plan (§6). Set the ROADMAP status to `Awaiting review`. Prepare a diff summary and a suggested commit message. | Doc updates and summary | **GATE 2** |

`T-000` (the bootstrap scaffolding) is the only task exempt from the `features/` trio. Every other task follows the table above.

### GATE 1 — approve requirements and plan (before any code)

- The agent presents: requirements, plan, open decisions (each with options, trade-offs, a recommendation), risks, and the exact files it intends to touch.
- The human replies exactly `APPROVED T-xxx` to proceed. `CHANGES T-xxx: <notes>` sends it back. **Anything else — silence, "ok", "looks good", a thumbs-up — is NOT approval.**
- The `Open questions` section of `requirements.md` must be empty before GATE 1.
- If implementation reveals that the plan is wrong or incomplete: STOP, amend `plan.md`, and request approval of the delta. Do not "just fix it".
- The agent can never approve its own plan or its own work.

### GATE 2 — review and commit (the human commits)

- The agent presents: what changed (file list and summary), validation evidence, deviations from the plan (should be none), and a suggested commit message.
- The **human** reviews the diff and runs the commit. Afterwards the human replies `COMMITTED T-xxx`; only then does the agent set the task to `Done` in `ROADMAP.md`.
- The agent never runs `git add`, `commit`, `push`, `tag`, `reset`, `rebase`, `stash`, `checkout --`, or any other command that writes git state. Read-only git (`status`, `diff`, `log`, `show`) is allowed.

### Task status values (`ROADMAP.md`)

`Not started` → `Specifying` → `Awaiting plan approval` → `Implementing` → `Awaiting review` → `Done`. Also `Blocked (<reason>)`.

## 3. Decision protocol

When a choice is not already settled in the Decision register (`TECH_STACK.md`) or an approved plan, emit this block and wait:

```
DECISION NEEDED: D-xxx — <title>
Context:        <why this must be decided now>
Options:        A) … (pros/cons)   B) … (pros/cons)   C) …
Recommendation: <option + one-line reason>
Blocks:         <task or step that cannot proceed>
Reply with:     DECIDE D-xxx: <A|B|C|other>
```

The human's reply is recorded in the Decision register (status `Approved`, date, chosen option). Until then the status stays `Open` and nothing may depend on it. Never encode an unapproved decision in code.

## 4. Hard rules

1. **No code before GATE 1** for the task. Only `features/T-xxx-*/` docs may be written during Phase A.
2. **Never commit or push.** Suggest; the human commits.
3. **One task at a time. No scope creep.** Anything discovered but out of scope goes to the `ROADMAP.md` Parking lot, not into the code.
4. **Fail safe.** When input, state, or intent is unknown or ambiguous, choose the most restrictive outcome or stop and ask. Never default to the permissive path.
5. **Never touch production or shared systems.** Run anything with side effects (databases, cloud resources, external services, destructive file operations) only against local or test resources, and verify the target first.
6. **No secrets or personal data in logs, tests, docs, or chat.** No credentials, tokens, connection strings, or real records. Fixtures are synthetic.
7. **No new dependency without approval.** Propose it via the decision protocol with: purpose, license, maintenance status (last release, activity), and alternatives considered (including "standard library").
8. **Deterministic by construction.** Same input gives the same output. No hidden clock, randomness, map-iteration order, or environment dependence in decision logic; inject clocks and ID generators. Tests must be repeatable.
9. **Do not invent facts.** If unsure of an API, version, or behaviour, say so and verify (docs, tests, a spike) before depending on it. Record verified facts with source and date in the verified-facts log in `TECH_STACK.md`.
10. **Truth over optimism.** Report failures, skipped checks, and deviations plainly. Never mark a requirement verified without evidence.
11. **Stay inside the project root** and inside the approved plan's file list.
12. **Capture general human feedback.** When the human gives feedback that applies beyond the current task, record it as a rule in `SKILLS.md` (section "Human feedback rules"; ask the human if "general or task-specific" is unclear), apply it to the current work and sweep existing code, and comply with every active rule in all later tasks. Each plan lists the rules that apply; each validation records compliance.
13. Standard library only: add no third-party dependency without an approved decision.
14. A change to any template or to the contract bumps templates.Version, and is verified by regenerating a kit in a temporary directory and running sdd check on it.

## 5. Language standards (summary — the full playbook is `SKILLS.md`)

**Go**

- `gofmt` / `goimports` clean; `go vet` clean; linter per the Decision register.
- Follow *Effective Go* and *Go Code Review Comments*. Idiomatic over clever.
- `context.Context` first for anything that does I/O or can block; wrap errors with `%w`; no `panic` outside `main` and tests.
- Small interfaces defined where they are consumed; no package-level mutable state; no `init()` side effects.
- Table-driven tests; `internal/` for everything that is not public API.

Tooling choices (formatter, linter, test runner, commit convention) are decisions: they live in the Decision register and are not assumed.

## 6. Living documents — updated as part of every task

The plan (its "Living docs to update" section) must list which of these change; the human reviews the doc diffs together with the code at GATE 2.

| File | Update when |
|---|---|
| `ROADMAP.md` | Every task: status, notes, newly discovered work (Parking lot), re-estimates |
| `TECH_STACK.md` | Any decision made or changed, architecture or layout change, dependency added, fact verified |
| `SKILLS.md` | A new standard, pitfall, or technique was learned or confirmed; general human feedback was given |
| `README.md` | User-visible behaviour, status, layout, or quickstart changed |
| `MISSION.md` | Scope, principles, or success criteria changed (requires explicit human approval) |
| Agent files | Workflow rules changed (requires explicit human approval; update every agent file) |

## 7. Communication format at every gate

1. **Where we are** — task ID, phase, status
2. **What I did / propose** — concise
3. **Files touched / to touch** — list
4. **Decisions needed** — in the §3 format (or "none")
5. **Risks / deviations** — or "none"
6. **Exact reply expected** — e.g. `APPROVED T-001`

Keep it short and checkable. No marketing language.

## 8. Definition of done (a task is `Awaiting review` only when all are true)

- [ ] Every requirement (R-xxx) maps to evidence in `validation.md`
- [ ] Build, lint, and tests pass using the commands recorded in `validation.md` (not "should pass")
- [ ] Negative and failure cases are tested, not only the happy path
- [ ] Running the relevant tests twice gives identical results (determinism)
- [ ] No files changed outside the approved plan; no new unapproved dependencies
- [ ] Living docs updated per §6
- [ ] Complies with every active rule in `SKILLS.md` "Human feedback rules" (compliance recorded in `validation.md`)
- [ ] Suggested commit message provided

## 9. Keeping the agent files in sync

CLAUDE.md, GEMINI.md, AGENTS.md share an identical rule set (§1–§9). Changing workflow rules needs explicit human approval and the same edit in every agent file in the same change set.
<!-- sdd:end contract -->
