# ROADMAP — dbguard

> **ALWAYS BE DETERMINISTIC. Human approval resolves every decision.**

Status: living document · Last updated: 2026-10-22

## 1. How this file works

- **Task IDs** are `T-xxx`. Each task (except `T-000`) gets a folder `features/T-xxx-<slug>/` with `requirements.md`, `plan.md`, `validation.md`, created from `features/_templates/` when the task starts.
- **Statuses:** `Not started` → `Specifying` → `Awaiting plan approval` → `Implementing` → `Awaiting review` → `Done`. Also `Blocked (<reason>)`.
- **Gates:** the human replies `APPROVED T-xxx` (GATE 1, before any code) and `COMMITTED T-xxx` (GATE 2, after reviewing and committing). See `CLAUDE.md` §2.
- **One task at a time.** Work discovered along the way goes to the Parking lot (§7), not into the current task.
- **Sizes** are rough and get recalibrated after M1 (§2).
- **Decisions** listed per task (D-xxx) must be Approved in `TECH_STACK.md` §11 at that task's GATE 1.

## 2. Estimates and capacity

| Size | Meaning | Rough effort |
|---|---|---|
| S | one focused sitting | about 3 h |
| M | two to three sittings | about 5 h |
| L | four or more sittings | about 9 h |

**Capacity (confirmed by the human, 2026-10-22): 12 hours per week.**

**Total for T-001 to T-020 is about 112 h, so the MVP is roughly 9–10 weeks at 12 h/week.** For reference:

| Hours per week | Approximate duration |
|---|---|
| 12 (planned) | 9–10 weeks |
| 17–20 | 6–7 weeks |
| 25 | 4.5–5 weeks |

Adding the production HTTP transport (T-020, plus the wider T-002 spike) cost about 13 h, roughly one week at 12 h/week. **Re-baseline after T-003 using actual hours.** If you are behind, cut in the order given in §6.

## 3. Milestones

| Milestone | Goal | Tasks | Est. hours | Cumulative hours | Approx. end (12 h/week) | Exit criteria |
|---|---|---|---|---|---|---|
| M0 | Project initialisation | T-000, T-000.1 | n/a | n/a | done on commit | Scaffolding reviewed and committed |
| M1 | Foundations and de-risking | T-001 – T-003 | 17 | 17 | week 2 | Skeleton builds; MCP wiring proven over stdio and HTTP from a real agent; parser and cgo decision made |
| M2 | Decision core (pure logic) | T-004 – T-007 | 24 | 41 | week 4 | Config, parser model, classifier, and policy complete with a golden corpus |
| M3 | Persistence and execution | T-008 – T-011 | 24 | 65 | week 6 | Audit log, Postgres layer, dry-run, and approvals work against a throwaway Postgres |
| M4 | Wiring and interfaces | T-012 – T-014, T-020 | 24 | 89 | week 8 | An agent can call dbguard over stdio and over authenticated HTTP, and a human can approve through the CLI, end to end |
| M5 | Proof, hardening, release | T-015 – T-019 | 23 | 112 | week 10 | Incident suite green; security pass done; installable; first five users shown |

## 4. Tasks

### M0 — Project initialisation

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-000 | Scaffolding: agent contracts, living docs, feature templates, `.gitignore` | S | none | Files reviewed by the human; `COMMITTED T-000` | D-001, D-002 | Done (`COMMITTED T-000`) |
| T-000.1 | Scaffolding amendments from review round 1 | S | T-000 | Transport scope (stdio plus Streamable HTTP), feedback rule (SKILLS §12), capacity, Go 1.27, and new decisions D-030 to D-032 recorded; patch reviewed and committed; `COMMITTED T-000.1` | D-005, D-012, D-030, D-031, D-032 | **Awaiting review** |

### M1 — Foundations and de-risking

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-001 | Project skeleton | S | T-000.1 | `go.mod`, directory layout, `cmd/dbguard` with `version` only, Makefile (`build test vet fmt fmt-check lint check clean`), lint config, conventions; `make check` passes on an otherwise empty skeleton | D-012, D-030, D-016, D-017, D-018, D-019 | Blocked (D-030: GitHub handle needed before GATE 1) |
| T-002 | Spike: MCP hello-world over stdio and Streamable HTTP | L | T-001 | A one-tool MCP server (official Go SDK, pinned) served over stdio and over Streamable HTTP on loopback, both connected from Claude Code and Gemini/Antigravity. Recorded: stdout stays clean in stdio mode; whether the SDK's HTTP handler validates `Host`/`Origin` or authenticates; whether each agent can send a custom `Authorization` header; client timeouts. First signal on H3 | D-005 (pin); informs D-031, D-032 | Not started |
| T-003 | Spike: SQL parser and CGo posture | M | T-001 | The three parser candidates are run against a 50+ statement conformance corpus (multi-statement, CTE writes, `EXPLAIN ANALYZE`, `DO`, parse errors); parse latency and `CGO_ENABLED=0` / cross-compile impact measured; D-010 and D-011 decided | D-010, D-011 | Not started |

### M2 — Decision core (pure logic, no I/O)

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-004 | Config model and loader | M | T-001 | Connection registry (id, environment tag, DSN read from a named environment variable, never a literal), thresholds, paths. Strict decoding. Missing environment tag, unknown field, or literal DSN refuses to start | D-014 | Not started |
| T-005 | Parser wrapper and statement model | M | T-003 | SQL string becomes internal `Statement`s; parse errors are typed; multi-statement handled per D-021 | D-021 | Not started |
| T-006 | Risk classifier v0, golden corpus, fuzz | L | T-005 | Tiers per `TECH_STACK.md` §5 including constant-true `WHERE`, `EXPLAIN ANALYZE`, data-modifying CTEs; unknown fails closed; golden corpus in `testdata/`; fuzz target proves no panics and that no writing statement is ever Safe | none new | Not started |
| T-007 | Policy engine | M | T-006, T-004 | Pure function (tier, environment, dry-run rows, config) → decision with reasons; every cell of the matrix covered by table-driven tests | D-023 | Not started |

### M3 — Persistence and execution

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-008 | Audit log (SQLite) | M | T-001, T-003 | Schema, migrations approach, append-only enforcement, WAL for cross-process access, injected clock; statement-text policy applied | D-011, D-029 | Not started |
| T-009 | Postgres execution layer | M | T-004 | Connection pool per connection id; Safe tier in a `READ ONLY` transaction; statement and lock timeouts; refuses non-local targets in test mode; error classification | D-013, D-020 | Not started |
| T-010 | Dry-run executor | L | T-009 | DML dry-run (transaction, execute, rows affected, rollback) with documented limits; integration tests against docker-compose Postgres | D-024 | Not started |
| T-011 | Approval store and binding | M | T-008 | State machine (pending / approved / denied / expired / consumed); binding to the exact statement hash, connection, environment, tier; TTL; one-time use; elevated confirmation for production-destructive | D-022 | Not started |

### M4 — Wiring and interfaces

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-012 | Pipeline orchestrator | M | T-007, T-010, T-011, T-008 | Parse → classify → policy → dry-run → policy → execute → audit, behind interfaces with fakes; end-to-end unit tests without a database; performance budget measured | D-028 | Not started |
| T-013 | MCP tools | M | T-012, T-002 | `dbguard_query` / `dbguard_execute` (names per D-025) wired to the pipeline over stdio; actionable refusal messages; structured output; verified with MCP Inspector and a real agent | D-025 | Not started |
| T-014 | CLI | M | T-011, T-013 | `serve` (stdio), `pending`, `show`, `approve`, `deny`, `log`, `version`; approving shows the exact statement and dry-run summary | D-015 | Not started |
| T-020 | Streamable HTTP transport and authentication | L | T-013, T-004, T-002 | Production MVP transport: `serve` gains an HTTP mode serving the same tools. Loopback bind by default; a non-loopback bind is refused unless explicitly configured and authentication is configured; every request authenticated per D-031 with the caller identity written to the audit log; `Host` and `Origin` validated; request-size limits; graceful shutdown; unauthenticated, wrong-Host, and wrong-Origin requests rejected in tests; deployment note (reverse proxy for TLS) | D-031, D-032 | Not started |

### M5 — Proof, hardening, release

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-015 | Incident-scenario E2E suite | M | T-014, T-020 | docker-compose Postgres; SQL-level scenarios modelled on publicly reported incidents; HTTP rejection scenarios; asserts S1, S2, S4, S7 | D-019 | Not started |
| T-016 | Security and privacy pass | M | T-015 | Threat-model review (including the HTTP surface); credential handling; log redaction check; least-privilege role guide; fail-closed audit of every decision path | none new | Not started |
| T-017 | Packaging and release | M | T-016 | Build matrix, install path, CI, LICENSE, `SECURITY.md` | D-026, D-027 | Not started |
| T-018 | README quickstart, demo, agent configs, verified sources | S | T-017 | A stranger reaches a first block in 10 minutes over stdio (S5); HTTP deployment note; incident sources verified and linked | none new | Not started |
| T-019 | Pilot (non-code) | M | T-018 | First five people shown; feedback captured; H1–H3 assessed; next milestone proposed | none new | Not started |

## 5. Unknown-technology protocol (spikes)

Last time a project stalled partly on technology that was new to the builder. To prevent that:

- Anything unfamiliar gets a **time-boxed spike** first (one sitting). Output is notes and a decision, not production code.
- Spike code is disposable: it lives under `features/T-xxx-<slug>/spike/` and is never imported by `internal/`.
- Spikes follow the same gates as any task (requirements, plan, validation; `APPROVED` before coding).
- Record what you learned in `SKILLS.md` (§10 and §11).

## 6. Scope-cut order if behind schedule (deterministic; the human confirms each cut)

1. In T-014, drop the `show` and `log` subcommands (keep `serve`, `pending`, `approve`, `deny`, `version`).
2. In T-006, defer the fuzz target (keep the golden corpus).
3. In T-017, drop any install script or package-manager formula (keep release binaries and CI).
4. In T-015, reduce the incident suite to at least 5 scenarios.

**Never cut:** the T-006 golden corpus, T-011 binding, the T-016 fail-closed audit, success criterion S1, and the T-020 authentication and loopback-by-default (S7).

## 7. Parking lot (post-MVP; do not start without a new approved task)

MySQL and other engines · MongoDB · Slack/Teams approvals · local web UI for approvals · hosted audit and team features · policy DSL (OPA/CEL) · tamper-evident hash-chained audit log · built-in TLS, rate limiting, and OAuth 2.1 / SSO for the HTTP transport · prompt-injection / intent detection · per-agent identity and RBAC · guarding `psql` and migration CLIs (closes the bypass gap) · infrastructure-API token guard · read-replica dry-run · migration lock analysis · literal redaction options · OpenTelemetry metrics · Windows support · multi-tenant mode.

## 8. Next action

1. Human reviews the T-000.1 patch, commits it, and replies `COMMITTED T-000.1`.
2. Human replies `DECIDE D-030: <module path>` (the GitHub handle is the missing piece).
3. The agent has drafted T-001 in `features/T-001-project-skeleton/`. Human reviews and replies `APPROVED T-001` (or `CHANGES T-001: <notes>`). No code is written before that.

## 9. Change log

| Date | Change |
|---|---|
| 2026-10-22 | Initial roadmap created (T-000 delivered for review) |
| 2026-10-22 | T-000 committed by the human; status set to Done |
| 2026-10-22 | T-000.1: transport scope amended (stdio for local and test, Streamable HTTP for the production MVP); T-020 added; T-002 widened to L; capacity set to 12 h/week; total re-estimated to about 112 h (9–10 weeks) |
| 2026-10-22 | T-001 requirements, plan, and validation drafted (blocked on D-030) |
