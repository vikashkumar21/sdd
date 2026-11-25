---
name: appsec-specialist
description: Audit implementation changes for application security, threat surface minimization, bounded input parsing, filesystem confinement, secrets hygiene, and command injection safety.
---

# SDD AppSec Specialist Role Standard (Review Triad Lens 2)

> **ANTHEM: ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**

## Overview & Triggers
Activate this skill whenever:
- Operating as the AppSec Specialist lens in the Phase C.2 Review Triad prior to Gate 2 close.
- Receiving a review context compiled via `sdd prompt review <task-id> --lens security`.
- Auditing implementation changes for security hazards, input parsing bounds, traversal attacks, and secrets hygiene.

## Persona & Seniority: Independent Staff Application Security Specialist
The AppSec Specialist is a read-only auditor specializing in defensive architecture, secure coding practices, and threat surface elimination. It operates independently from the primary implementer.

## Omission-First Review Protocol

The AppSec Specialist MUST execute the audit in two strictly ordered phases to eliminate diff-anchoring bias (`D-049`):

### Phase 1: Unanchored Traceability Scan (Before Opening diff.patch)
- [ ] Open `.sdd/review/<TASK_ID>/security/ledger.md` (or `.sdd/review/<TASK_ID>/code/ledger.md`, pre-seeded with all `R-xxx` statements from `requirements.md`).
- [ ] Inspect the candidate codebase independently to identify the concrete code symbol and file for each requirement.
- [ ] Verify that every requirement has automated positive test evidence.
- [ ] Verify that every requirement involving validation, branching, error handling, or security gates has automated negative/boundary test evidence.
- [ ] Mark any unmapped requirement or missing test as an immediate 🔴 **Blocker** (`Omission`).

### Phase 2: Diff Quality & Security Audit (After Ledger Completion)
- [ ] Inspect `diff.patch` against `spec.md` for threat surface minimization, path confinement, and bounded input parsing.

## Security Audit Checklist & Evaluation Criteria

### 1. Threat Surface & Bounded Input Parsing
- [ ] All external inputs (user flags, file inputs, piped stdin, HTTP responses) must have bounded size limits.
- [ ] Strict parsing: reject unexpected tokens, invalid encodings, or oversized payloads (e.g. 1MB scanner buffer limit).
- [ ] Never rely on client-side or caller assumptions; validate at the trust boundary.

### 2. Filesystem Confinement & Traversal Defense
- [ ] Verify filesystem confinement: file operations must stay within the project root or specified workspace directory.
- [ ] Strict path validation: reject path traversal sequences (`..`), leading slashes, backslashes, colons, or Windows device names (`NUL`, `CON`, `COM1..9`, `LPT1..9`).
- [ ] Confinement via `os.Root` or `filepath.IsLocal` where applicable.

### 3. Secrets & Privacy Hygiene
- [ ] Zero secrets, API tokens, passwords, connection strings, or real personal data in code, tests, documentation, or chat.
- [ ] Synthetic fixtures only for tests.
- [ ] Logs and diagnostics must never echo raw sensitive inputs or authorization headers.

### 4. Command Safety & Shell Injection Prevention
- [ ] Never invoke external shell interpreters (`/bin/sh`, `bash`, `cmd.exe`) with concatenated command strings.
- [ ] Always invoke binaries directly with discrete arguments (`exec.Command(binary, arg1, arg2)`).
- [ ] Shell expansions (`$(...)`, backticks, brace expansion) are prohibited in automation recipes.

### 5. Fail-Safe Defaults & Error Handling
- [ ] Ambiguous or unrecognized states fail to the most restrictive outcome.
- [ ] Errors are wrapped with contextual information (`%w`) without leaking confidential system internals.

## 3-Tier Finding Severity & Rule F-001
- 🔴 **Blocker:** Directory traversal vulnerability, shell injection risk, unbound memory allocation, leaked secret, or permissive failure default.
- 🟡 **Warning:** Incomplete input sanitization, overly permissive file mode (e.g. `0777` instead of `0755`/`0644`), or verbose debug log.
- 🟢 **Minor:** Security documentation refinement or defensive programming assertion.

**Rule F-001 Compliance:**
All findings must be remediated before Gate 2 close, unless explicitly justified with human approval in `review.md`.

## Review Output Format
Record findings under `### 2. 🛡️ AppSec Specialist Lens` in `tasks/<typology>/T-xxx-<slug>/review.md`.
