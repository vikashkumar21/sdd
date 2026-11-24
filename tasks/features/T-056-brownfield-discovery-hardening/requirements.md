---
status: draft
---
# Requirements — T-056 Forensic brownfield discovery hardening, coverage ledger, and multi-language review templates

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-056
Decision ref: `TECH_STACK.md` D-048 (Approved 2026-11-23)

## 1. Goal

Eliminate single-agent blind spots and Go-specific template bias in brownfield discovery by introducing a mandatory Coverage Ledger, unanchored challenger inventorying, enriched review bundles (`validation.md` in plan review; staged context in discovery), language-pack-aware code review templates, and scratchpad citation hygiene.

## 2. Context and references

- Mission / principles: Principle 1 (Deterministic by construction), Principle 2 (Fail safe), Principle 5 (Never overwrite human work).
- Decisions: D-007 (Standard library only), D-024 (agentskills.io standard), D-033 (Document lifecycle frontmatter), D-036 (Two-tier role skills), D-045 (Adversarial discovery ensemble), D-048 (Forensic brownfield discovery hardening).
- Depends on: T-053 (Adversarial discovery ensemble), T-055 (Default task token telemetry).

## 3. Scope

### In scope
1. **Coverage Ledger Standard & Check Integration (`R-001`):**
   - Standardize `## Coverage Ledger` format for forensic discovery reporting:
     `| Manifest / Config File | Total Lines | Lines Read | Lines Skipped | Reason for Skip | Fully Verified (Yes/No) |`.
   - Integrate mechanical check rule `coverage-ledger` in `sdd check` verifying that brownfield discovery reports contain a valid Coverage Ledger and that unverified/skipped files have explicit recorded justifications.
2. **Review Bundle Enrichment (`R-002`):**
   - In `internal/audit/bundle.go`:
     - Update `StagePlanBundle` to include `validation.md` (the draft Traceability Matrix) in `spec.md`, allowing `plan-reviewer` to audit test traceability prior to Gate 1.
     - Update `StageDiscoveryBundle` to stage `.sdd/context/` artifacts into `.sdd/review/<TASK_ID>/discovery/context/` so the challenger can cross-verify external reference packs (e.g. migration guides, Jira specs).
3. **Unanchored Challenger Inventory-First Protocol (`R-003`):**
   - Update `internal/templates/data/skills/discovery-challenger/SKILL.md`:
     - Mandate Step 1: Unanchored Independent Scan — build an independent inventory of all build files, Dockerfiles, and manifests before inspecting `findings.md`.
     - Step 2: Diff independent inventory against analyst's Coverage Ledger to catch omissions.
     - Step 3: Cross-check codebase against external reference documents in `.sdd/review/<TASK_ID>/discovery/context/`.
     - Generalize blind-spot vectors to be language-agnostic (JVM, Python, TypeScript, Go).
4. **Platform, Runtime & Architecture Discovery Migration Profile (`R-004`):**
   - Update `internal/templates/data/skills/codebase-analyst/SKILL.md` and `internal/cli/prompt_discovery.go`:
     - Restructure progressive extraction checklist to cover Runtime Defaults (compiler/runtime version, GC ergonomics, character encoding/UTF-8 defaults, timezone/locale defaults, TLS/keystore protocols), Container & Image Inventory, and Dependency Age/EOL.
     - Conditionally focus UI/Design System extraction only when frontend assets are detected.
5. **Language-Pack-Aware `review.md` Scaffolding (`R-005`):**
   - Replace hardcoded "Effective Go" in review templates for review-bearing task typologies (`features` and `bugfixes`).
   - Add language review rubrics to language packs in `internal/templates/data/packs/` (`go`, `java`, `python`, `typescript`, `generic`).
   - Dynamically render language-specific review standards during `sdd init` / `sdd upgrade` (and task scaffolding) based on detected or specified languages.
6. **Scratchpad Citation Hygiene Guard (`R-006`):**
   - In `internal/check/paths.go` / `internal/check/check.go`:
     - Flag citations in `TECH_STACK.md`, `ROADMAP.md`, `SKILLS.md`, or task documents that reference temporary scratch paths (`scratch/`, `.sdd/tmp/`, `/tmp/`, `.system_generated/`).
7. **Template Version Bump & Contract Parity (`R-007`):**
   - Bump `templates.Version` from `2.3.0` to `2.4.0` per Hard Rule 16.
   - Update contract markers `templates=2.4.0` across `AGENTS.md`, `GEMINI.md`, `CLAUDE.md`, and `.sdd/manifest.json`.
   - Verify `sdd upgrade` migrations and fresh kit generation pass strict check.
8. **Standard Library Purity (`R-008`):**
   - 100% standard library implementation; zero external dependencies per `D-007`.

### Out of scope
- Automated AST parsers for every programming language (agents execute discovery using existing toolchains under Gate 1 / Gate 2).
- Automatic package manager network calls (e.g. `npm audit`, `mvn dependency:tree` via network; respects offline/sandboxed execution).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL standardize the `## Coverage Ledger` format for discovery tasks and validate in `sdd check` that discovery tasks record complete line ranges or explicit skip justifications for all inspected manifests. | MUST | test / inspection |
| R-002 | The system SHALL enrich review bundles in `internal/audit/bundle.go`: `StagePlanBundle` SHALL include `validation.md` in `spec.md`, and `StageDiscoveryBundle` SHALL stage `.sdd/context/` artifacts into `.sdd/review/<TASK_ID>/discovery/context/`. | MUST | test |
| R-003 | The system SHALL update `discovery-challenger/SKILL.md` to require an unanchored independent inventory scan prior to inspecting analyst findings, diff against the Coverage Ledger, cross-check staged reference context, and evaluate language-agnostic vectors. | MUST | test / inspection |
| R-004 | The system SHALL update `codebase-analyst/SKILL.md` and `internal/cli/prompt_discovery.go` to include Platform, Runtime & Architecture migration checklist items (runtime defaults, container inventory, dependency EOL, and coverage ledger). | MUST | test / inspection |
| R-005 | The system SHALL scaffold `review.md` using language-pack-aware review rubrics across review-bearing typologies (`features` and `bugfixes`) instead of hardcoding Go conventions, selecting the rubric matching the project's primary detected language. | MUST | test |
| R-006 | The system SHALL validate citation hygiene in `sdd check` under rule `citation-hygiene`, flagging file citations that reference ephemeral scratch paths (`scratch/`, `/tmp/`, `.sdd/tmp/`, `.system_generated/`). | MUST | test |
| R-007 | The system SHALL bump `templates.Version` to `2.4.0`, maintain contract parity across `AGENTS.md`, `GEMINI.md`, and `CLAUDE.md`, and verify clean `sdd upgrade` migrations. | MUST | test |
| R-008 | The system SHALL maintain 100% standard library purity (`D-007`) and zero external dependencies. | MUST | test |

## 5. Non-functional requirements

- **Determinism:** All template renderings, review bundle staging, and check verifications are 100% deterministic with canonical sorting and injected clocks.
- **Fail-safe:** Missing or malformed coverage ledgers produce clear warnings/errors; ephemeral scratch citations are flagged before Gate 2 commit.
- **Performance budget:** `sdd check` executes in < 50ms for repository markdown files.
- **Security & Hygiene:** Confinement enforced via `os.OpenRoot` and `filepath.IsLocal`. Zero personal filesystem paths (`Rule F-002`).

## 6. Constraints and assumptions

- Brownfield discovery can run on diverse language environments (Go, Java, Python, TypeScript, generic polyglot).
- Review bundles remain strictly stateless, isolated in `.sdd/review/<TASK_ID>/`, and gitignored by default.

## 7. Open questions

(none)

## 8. Acceptance criteria

- **AC-1 (`validation.md` in Plan Bundle):** Calling `StagePlanBundle` produces a `spec.md` containing `requirements.md`, `plan.md`, and `validation.md`.
- **AC-2 (`context/` in Discovery Bundle):** Calling `StageDiscoveryBundle` on a project with `.sdd/context/` entries copies them to `.sdd/review/<TASK_ID>/discovery/context/` and lists them in `bundle.Files`.
- **AC-3 (Coverage Ledger Validation):** `sdd check` detects missing Coverage Ledger in completed discovery tasks or warns on unjustified skipped lines.
- **AC-4 (Language-Aware `review.md`):** Running `sdd feature new` on a Java project generates `review.md` containing Java conventions instead of "Effective Go".
- **AC-5 (Scratchpad Citation Hygiene):** `sdd check` flags any markdown citation pointing to `scratch/` or `.system_generated/` under rule `citation-hygiene`.
- **AC-6 (Template Version & Upgrade):** `templates.Version` is `2.4.0`; `sdd upgrade` cleanly migrates older kits and passes `sdd check --strict`.
