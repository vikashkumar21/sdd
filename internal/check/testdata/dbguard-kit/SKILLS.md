# SKILLS — dbguard

> **ALWAYS BE DETERMINISTIC. Human approval resolves every decision.**

Status: living document · Last updated: 2026-10-22

**What this file is.** The standards, techniques, and checklists that the agent *and* the human apply while building dbguard. It is a working playbook, not a wish list: add a rule when you learn one, and record pitfalls as you hit them (§11). Items marked **(verify)** are reported but not yet confirmed by us. **Human feedback that applies generally is recorded in §12 and is binding on all later work.**

## 1. Go idioms and standards

References: *Effective Go*, *Go Code Review Comments*, the Google Go Style Guide.

- Formatting and imports: `gofmt` and `goimports` clean; `go vet` clean; linter per D-017.
- **Naming:** short, clear, no stutter (`audit.Log`, not `audit.AuditLog`). Package names are lowercase, singular, no underscores.
- **Interfaces:** small, defined in the *consumer* package. Accept interfaces, return concrete types. No interface until there are two implementations or a test seam is needed.
- **Structure:** everything private goes in `internal/`. No package-level mutable state; no `init()` side effects. Constructors take dependencies explicitly.
- **Zero values** should be useful where possible. Avoid boolean-flag parameters; prefer small option structs or distinct functions.
- **Comments:** every exported identifier has a doc comment starting with its name. Comment *why*, not *what*.

## 2. Errors, context, concurrency

- Wrap with `fmt.Errorf("doing X: %w", err)`. Error strings are lowercase with no trailing punctuation.
- Use sentinel errors or typed errors (`errors.Is` / `errors.As`) for anything a caller must branch on, especially decision-relevant failures (parse error, policy refusal, expired approval).
- No `panic` outside `main` and tests. Never ignore an error silently; if intentionally ignored, say why.
- `context.Context` is the first parameter of anything that does I/O or can block. Never store a context in a struct.
- Every goroutine has an owner, a stop condition, and error propagation. Prefer synchronous code; add concurrency only with a measured need.
- Inject a `Clock` and an ID generator. Never call `time.Now()` or a random source inside decision logic.

## 3. Testing standards

- **Table-driven** tests with named cases; failures print inputs and a diff (`go-cmp`, if approved in D-019).
- **Golden corpus:** SQL statements and their expected tier in `testdata/`, reviewed by the human whenever it changes. A change to expected output is a decision, not a test fix.
- **Fuzz** `classify` (Go native fuzzing): no panics, and *no writing statement is ever classified Safe*.
- **Fail-closed tests are mandatory:** unparsable, unknown statement type, empty input, huge input, odd encodings, multi-statement.
- **Determinism:** run tests twice and compare; no map-iteration-order dependence, no time dependence.
- **Integration tests:** behind `//go:build integration`; only against the throwaway docker-compose Postgres on localhost. The test harness verifies the target host before running any SQL.
- Fixtures are synthetic; no real data or credentials.
- Record the exact commands and results in `validation.md`; "should pass" is not evidence.

## 4. PostgreSQL and SQL-safety knowledge

Facts the classifier, executor, and dry-run depend on:

- **`EXPLAIN ANALYZE` executes the statement.** Plain `EXPLAIN` does not.
- **Constant-true `WHERE`** (`WHERE true`, `1=1`) deletes or updates everything. "Has a WHERE" is not enough; the predicate must be *effective*.
- **Data-modifying CTEs** can hide writes inside a top-level `SELECT`.
- **`SELECT` can call volatile functions that write.** A `READ ONLY` transaction makes Postgres reject writes (including `nextval`) and most DDL. It is a strong backstop, but **not absolute**: it is a high-level read-only notion, temp tables remain writable, and it does not stop side effects outside Postgres. Least-privilege roles remain the real baseline.
- **Transactional DDL:** most DDL (including `DROP TABLE`, `TRUNCATE`) is transactional in Postgres, but takes heavy locks (often `ACCESS EXCLUSIVE`). Exceptions that cannot run in a transaction block: `CREATE INDEX CONCURRENTLY`, `REINDEX CONCURRENTLY`, `VACUUM`, `CREATE/DROP DATABASE`, `ALTER SYSTEM`.
- **Sequences are not rolled back**; triggers and volatile functions *do* run during a dry-run; `NOTIFY` is delivered only on commit, so rollback discards it.
- **Command tags** report rows affected (for example `DELETE 42`); `SET LOCAL` is scoped to the transaction.
- **Timeouts:** `statement_timeout`, `lock_timeout`, `idle_in_transaction_session_timeout`.
- **`COPY … PROGRAM`** and `DO` blocks run arbitrary code and are never Safe.
- **`PREPARE` / `EXECUTE`** hide the executed text from static classification.
- **Multi-statement strings** (`SELECT 1; DROP TABLE x`) are a classic smuggling route; see D-021.
- **`pg_query` fingerprints ignore literal values:** never use them for approval binding.

## 5. MCP server skills (official Go SDK)

- Create an `mcp.Server`, add tools, run it over an `mcp.Transport` (stdio for a local server). Tool inputs are typed Go structs with `json` and `jsonschema` tags; tool failures are reported as results with `IsError` set (per the SDK README).
- **In stdio mode, stdout belongs to the protocol.** Any log or `fmt.Println` on stdout corrupts the stream. Log to stderr or a file (we do this in every mode).
- **Tool names** use a consistent prefix and verbs (`dbguard_query`, `dbguard_execute`; D-025). Descriptions are *prompts to the agent*: say precisely what the tool does, when a refusal happens, and what to do next.
- **Refusals must be actionable:** state the tier, the reason, the approval id, and the exact next step. Do not leak DSNs, internal paths, or driver errors verbatim.
- **Annotations** (`readOnlyHint`, `destructiveHint`, `idempotentHint`, `openWorldHint`) are hints for clients. **Never rely on them for enforcement.**
- **Do not build on** the MCP `roots`, `sampling`, or `logging` features: deprecated as of spec 2026-07-28 (SEP-2577), per the SDK README.
- **Client constraints (verify in T-002):** practitioners report client tool-call timeouts (about 60 s in some clients) and idle disconnects. Hence: no blocking approval waits; return `approval_required` immediately.
- **Streamable HTTP (production transport, D-005).** Per the MCP transport guidance (spec 2025-06-18; re-check against 2026-07-28 in T-020): servers MUST validate the `Origin` header, SHOULD bind to `127.0.0.1` rather than `0.0.0.0` when local, and SHOULD authenticate every connection. Also validate `Host`: CVE-2026-42559 (Rust SDK) shows DNS rebinding against loopback servers is a live attack. **Do not assume the Go SDK handler does any of this** (not verified); wrap it in your own middleware and test the rejections. Fail closed: refuse to start on an unsafe configuration.
- Test with MCP Inspector (`npx @modelcontextprotocol/inspector`) *and* with a real agent.

## 6. SQLite usage (audit log and approvals)

- Enable WAL mode and set `busy_timeout`; the MCP server and the CLI are separate processes sharing one file on a local filesystem (not a network share).
- One writer at a time: keep write transactions short.
- Enforce append-only with triggers that reject `UPDATE` and `DELETE` on the audit table.
- Schema changes go through numbered migrations; never edit applied migrations.

## 7. Security and privacy hygiene

- The agent never receives a DSN. Config stores the *name* of an environment variable, never the secret itself. Check config file permissions (0600) when practical.
- Application logs contain no credentials and no row data. The audit log's statement text follows D-029 and is documented as sensitive.
- Recommend least-privilege roles, one per environment; document how to create them (T-016).
- Treat all agent-supplied text as hostile input: bounded sizes, strict parsing, no string-built SQL for dbguard's own queries (parameterise everything).
- Tests and tooling never touch a non-local database.

## 8. Spec-driven authoring skills

**Requirements (`requirements.md`)**
- One requirement per row, ID `R-xxx`, phrased "The system SHALL …". Atomic, testable, no vague words ("fast", "robust") unless measurable.
- Each has a verification method (test / inspection / demo). Include explicit non-goals.
- Acceptance criteria in Given / When / Then form where behaviour is involved.
- `Open questions` must be empty before GATE 1.

**Plan (`plan.md`)**
- Steps small enough to review in one sitting; each names the files and the test that proves it.
- List new or affected decisions with options and a recommendation; list new dependencies with justification.
- List every living doc that will change (§6 of `CLAUDE.md`), plus risks and rollback.

**Validation (`validation.md`)**
- A traceability matrix: R-xxx → method → exact command or test → expected → actual → status.
- Evidence means a real command and a real result. Anything not verified is labelled **NOT VERIFIED**, with the reason.

## 9. Review checklists

**Human at GATE 1** — Do the requirements say what I want, and only that? Are all open decisions resolved? Is every file in the plan necessary? Is any dependency avoidable? Are the applicable §12 rules listed?

**Human at GATE 2** — Does the diff touch only planned files? Do the tests include fail-closed and negative cases? Any secrets, DSNs, or real data? Was the determinism check run? Are the living docs updated? Is the suggested commit message accurate? Is compliance with every active §12 rule recorded?

## 10. Human skill map and learning log (human fills in)

Use this to decide where a spike is needed (see ROADMAP §5). Familiarity: Strong / OK / New.

| Area | Needed in | Familiarity | Planned learning action |
|---|---|---|---|
| Go stdlib and idioms | all | | |
| `pgx` / Postgres wire behaviour | T-009, T-010 | | |
| Postgres transaction, lock, and command-tag semantics | T-009, T-010 | | |
| SQL ASTs and `libpg_query` parse trees | T-003, T-005, T-006 | | |
| MCP protocol and the Go SDK | T-002, T-013 | | |
| SQLite (WAL, driver specifics) | T-008 | | |
| CLI design in Go | T-014 | | |
| Go fuzzing | T-006 | | |
| docker-compose test databases | T-015 | | |
| Release and packaging tooling | T-017 | | |

## 11. Pitfalls log (append-only; newest at the bottom)

| Date | Pitfall | Source / task |
|---|---|---|
| 2026-10-22 | `pg_query_go` needs cgo, which conflicts with "single static binary" and with a pure-Go-only toolchain | Verified research; D-010 |
| 2026-10-22 | `EXPLAIN ANALYZE <stmt>` executes the statement | PostgreSQL behaviour; T-006 |
| 2026-10-22 | `WHERE 1=1` / `WHERE true` defeats a naive "has WHERE" check | Design analysis; T-006 |
| 2026-10-22 | Logging to stdout breaks a stdio MCP server | MCP builder guidance; T-002 |
| 2026-10-22 | Fingerprints that ignore literals must not be used to bind approvals | Design analysis; T-011 |
| 2026-10-22 | MCP clients reportedly enforce about a 60 s tool timeout and drop idle connections **(verify)** | Practitioner reports; T-002 |
| 2026-10-22 | A loopback-only HTTP MCP server can still be attacked from a victim's browser via DNS rebinding unless `Host` and `Origin` are validated and requests are authenticated | CVE-2026-42559; MCP transport guidance; T-020 |
| 2026-10-22 | `gochecknoglobals` flags the ldflags-injected `Version` variable; allow it with a narrow `//nolint` and a reason | Design analysis; T-001 |
| 2026-10-22 | golangci-lint must be built with a Go version at least as new as the module's `go` directive, or it refuses to run **(verify; from memory)** | T-001 |

## 12. Human feedback rules (binding; grows over time)

Human feedback is how this playbook improves. The rules registered below are **binding on every later task**.

### Process

1. **Classify** every piece of human feedback on a plan, implementation, or code:
   - **General**: would apply to future code or tasks (style, structure, testing, naming, security, workflow). Record it here as an `F-xxx` rule.
   - **Task-specific**: only about this task. Apply it and note it in that task's `validation.md` §6. Not recorded here.
   - **Unclear**: ask the human (decision protocol): "general or task-specific?" Never guess.
2. **Record** a general rule in the same change set: ID, date, source (task and gate), the rule in one or two imperative sentences, where it applies, and *how it is verified* (a lint rule, a test, a checklist item, or manual review). The human reviews the wording at GATE 2.
3. **Apply immediately:** fix the current task's work and sweep existing code for violations; list what changed.
4. **Qualify every later task:** each `plan.md` lists the active F-rules that apply and how it complies; each `validation.md` records compliance per rule (pass / fail / n.a.). A task cannot reach `Awaiting review` with a failing rule.
5. **Conflicts:** if a new rule conflicts with an existing rule, decision, or requirement, STOP and report. The human decides; the older rule is marked `Superseded` (never deleted).
6. **Promote to tooling** where possible: enforce a rule through the linter, a test, or a Makefile check instead of relying on memory.

### Register

| ID | Date | Rule | Applies to | Source | Verified by | Status |
|---|---|---|---|---|---|---|
| | | (none yet) | | | | |
