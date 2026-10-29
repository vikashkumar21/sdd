# MISSION — dbguard

> **ALWAYS BE DETERMINISTIC. Human approval resolves every decision.**

Status: living document · Last updated: 2026-10-22 · Maintainer: Vikash (human)

## 1. One-liner

**dbguard is a safety layer that sits between AI coding agents and your database.** It classifies every SQL statement an agent wants to run, blocks or holds the dangerous ones for human approval, and keeps an audit trail of everything.

## 2. The problem

AI coding agents (Claude Code, Gemini/Antigravity, Cursor, Replit Agent, …) increasingly have database access: through MCP servers, scripts they write, or CLIs they drive. They act with the credentials of a human developer but without a human's hesitation before something irreversible.

Three failure modes keep recurring:

1. **No policy gate.** A `SELECT` and a `DROP TABLE` are treated identically.
2. **No environment awareness.** Nothing distinguishes "production" from "a disposable dev database". An agent can be pointed at the wrong one, for example by a command flag or config that looks harmless.
3. **No audit trail.** Nothing links what the agent was asked, what it generated, what was approved, and what actually ran.

Publicly reported incidents motivate this. **Sources are to be added and verified in T-018; until then treat these as unverified summaries:**

- an AI agent deleting a production database during a code freeze (Replit, 2025);
- an agent deleting a whole production table from a vague "clean up" instruction;
- a migration-tool flag silently pointing an agent at a live schema;
- an agent reportedly using a stray API token to delete a production database and its backups (PocketOS, 2026). This one appears to be an infrastructure API call rather than SQL, so it is *outside* dbguard's boundary (see §6).

## 3. Who it is for (initial)

- Developers and small teams who give AI coding agents database access and want a seatbelt.
- Solo / indie builders using agents heavily against real data.
- Self-hosted first: credentials never leave the user's machine or infrastructure.

## 4. MVP scope

**In scope**

- PostgreSQL only.
- An MCP server (official Go SDK) that agents use *instead of* a raw database MCP server. dbguard holds the real credentials; the agent never sees them. Transports: **stdio** for local development and testing, and **Streamable HTTP** for the production MVP (authenticated, loopback by default; see D-031 and D-032).
- AST-based risk classification into **Safe / Caution / Destructive**, plus **Blocked** for statements that must never run through dbguard. Anything unknown or unparsable fails closed.
- A per-connection environment tag (`production` / `development`), with stricter handling for production.
- Transactional **dry-run** (execute → measure → roll back) for DML, with row-count thresholds.
- Human approval through the CLI (`dbguard pending / approve / deny`), bound to the exact statement.
- Append-only **audit log** (SQLite).
- A single self-hosted binary, a README quickstart, and an incident-based test suite.

**Out of scope for the MVP** (see the ROADMAP Parking lot)

- Other databases (MySQL, MongoDB, …).
- Prompt-injection / intent detection.
- Web dashboard, Slack/Teams approvals, SSO/RBAC, hosted or multi-tenant tier.
- Advanced HTTP features (built-in TLS, rate limiting, SSO; the MVP's own authentication is decided in D-031), policy DSL, tamper-evident (hash-chained) logs.
- Guarding shell/CLI database tools (`psql`, migration CLIs) and infrastructure-API tokens.

## 5. Principles

1. **Deterministic.** Same input and same config give the same decision, every time. No heuristics that vary between runs. Humans resolve ambiguity.
2. **Fail closed.** Unknown, unparsable, or ambiguous means the most restrictive outcome.
3. **Human in the loop for the irreversible.** Destructive actions need explicit, specific approval.
4. **Defence in depth.** Parser classification, engine-level read-only transactions, dry-run, the environment guard, and least-privilege roles. No single layer is trusted alone.
5. **Self-hosted, minimal trust.** Credentials stay local; the agent never sees a DSN; application logs never contain credentials or row data. The HTTP endpoint is never unauthenticated and binds to loopback by default.
6. **Boring technology.** Go, Postgres, SQLite, stdlib first. Every dependency must earn its place.
7. **Small and shippable.** Thin vertical slices; each task independently reviewable and committable.

## 6. Known limits (stated plainly; they will also appear in the README)

- dbguard protects only the path that goes **through** it. If the agent also holds a raw DSN, can run `psql`, or has another database tool or MCP server, it can bypass dbguard. Deployment guidance must say: remove the other routes and credentials.
- It does not cover infrastructure-API abuse (cloud/PaaS tokens that delete volumes or databases). That is a different control surface.
- Parser classification is necessary but not sufficient. It is complemented by read-only transactions, dry-run, and least-privilege roles (which we recommend but cannot enforce).
- Dry-run cannot perfectly simulate every side effect (see `TECH_STACK.md` §6).
- It is not a defence against a malicious human who has database credentials.
- In HTTP mode dbguard is a network service that holds database credentials. It must sit behind authentication, TLS (typically a reverse proxy), and restricted network access. dbguard refuses unsafe defaults but cannot make a misconfigured deployment safe.

## 7. MVP success criteria (Proposed: thresholds need human approval)

| # | Criterion | Measure |
|---|---|---|
| S1 | Destructive statements are never auto-executed | 100% of the destructive corpus is blocked or held; **0 false negatives** |
| S2 | SQL-level incident replays are stopped | Every SQL-level scenario in the incident suite is caught (T-015) |
| S3 | The safe path is fast | Overhead on Safe-tier queries p95 under 20 ms, excluding DB time (D-028) |
| S4 | Deterministic | The corpus run twice gives byte-identical decisions |
| S5 | Easy to adopt | A stranger installs and sees a first block within 10 minutes of starting the README |
| S6 | Real-world signal | At least 5 external people shown it; at least 2 keep it enabled for 2+ weeks (T-019) |
| S7 | The HTTP endpoint is not open | 100% of unauthenticated, wrong-Host, and wrong-Origin requests are rejected in tests (T-020, T-015) |

S1 is absolute. False positives are tolerated, tracked, and tuned later.

## 8. Hypotheses still to validate (NOT facts)

- **H1 — Willingness to pay.** Developers may prefer free DIY (a read-only DB role and careful config) over a tool. Test in T-019.
- **H2 — Platform absorption.** Agent vendors or database platforms may build this natively; some already ship partial mitigations such as prod/dev database separation. Re-evaluate at every milestone.
- **H3 — Adoption surface.** Developers will route database access through an MCP server instead of a raw connection. First signal in T-002.

## 9. Business intent (north star, Proposed)

Build a small, trusted, niche tool that stays relevant to its users. Near term: ship the MVP, get real users, learn whether H1–H3 hold. Longer term, decided with evidence and not now: an open-source core with possible paid team features (Slack approvals, hosted audit, policy management). The license decision (D-026) is deferred to T-017.
