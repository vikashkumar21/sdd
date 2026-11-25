---
status: completed
---
# Requirements — T-063 Living document progressive disclosure and Rule 1 micro-index

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-25)
Roadmap ref: `ROADMAP.md` T-063

## 1. Goal
Enable progressive disclosure of SDD living documents by introducing a lightweight, high-density project passport and micro-index generator (`sdd passport` and `sdd status --compact`), evolving contract Rule 1 to orient agents with the passport first and load full living documents on-demand, reducing session startup token overhead by up to ~94%.

## 2. Context and references
- Mission / principles touched: §1 Determinism, §2 Fail safe, §4 Standard library only, §6 Living documents.
- Decisions this relies on: `D-050` (Token consumption mitigation, context hygiene, and provider-aligned caching architecture — Approved 2026-11-25), `D-007` (Go standard library purity).
- Depends on tasks: `T-062` (Provider-aligned prompt prefix caching and dynamic suffix pinning — Done).

## 3. Scope
**In scope**
- Implement `internal/passport` package with deterministic clock injection (`Generate(root string, now time.Time)`), multi-task/0-task roadmap support, safe document extraction, CRLF normalization, and compact formatting ($\le 500$ tokens / $\le 2,000$ bytes for rendered Markdown).
- Implement `sdd passport` CLI command supporting default Markdown micro-index, `--json` structured output, `--dir <root>`, and integrate `--compact` flag into `sdd status`.
- Evolve Section 1 (Rule 1) of agent contracts (`AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, and `internal/templates/data/base/contract.md.tmpl`) to establish the Progressive Disclosure Protocol citing canonical section numbers (`TECH_STACK.md` Decision register §6, verified facts §7; `SKILLS.md` §1 Language standards, §8 Feedback rules), while strictly preserving the mandatory startup declaration and conflict-halt instruction.
- Update base prompt templates (`prompt.md.tmpl` maintaining $\ge 4,096$-byte static prefix floor, `prompt_compact.md.tmpl` maintaining $\le 300$-word ceiling).
- Implement mechanical validation check (`passport-consistency`) in `internal/check/passport.go` verifying budget compliance, task alignment with `ROADMAP.md`, and document link validity, with graceful skip semantics when `ROADMAP.md` is absent.
- Update `skills/task-coordinator/SKILL.md`, `internal/skill/export.go`, and canonical skill templates in `internal/templates/data/skills/`.
- Bump `templates.Version` to `2.11.0`, verify temporary kit regeneration (`sdd init --yes` + `sdd check` with 0 errors), and verify repository strict check (`./bin/sdd check --strict`) with 0 errors per Hard Rule 16.
- 100% Go standard library purity (`D-007`), race-clean (`-race`), and deterministic (`-count=2`).

**Out of scope**
- Automated mutation of living documents by `sdd passport` (`sdd passport` is strictly a read-only generator); Phase D manual living doc updates remain in scope.
- Modifying token telemetry persistence logic in `internal/state/` (owned by `T-064`).
- Third-party document summarization or vector embeddings.

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL implement an `internal/passport` package with `Generate(root string, now time.Time) (*Passport, error)` that synthesizes a deterministic project passport micro-index whose rendered Markdown strictly adheres to $\le 500$ estimated tokens ($\le 2,000$ bytes), capturing project mission, stack (polyglot-aware), active milestone, active tasks (or next task if none active), active feedback rules (parsed from `### Register` table), and living document pointers. | MUST | Unit tests (`TestGenerate`, `TestBudgetCeiling`, `TestDeterministicOutput`) |
| R-002 | The system SHALL provide a CLI command `sdd passport` (supporting Markdown output, `--json`, `--dir`, exiting `0` on success, `1` on error, and `2` on usage error) and integrate a `--compact` flag into `sdd status` (where `--compact --json` emits structured JSON passport). | MUST | CLI tests (`TestPassportCLI`, `TestStatusCompactCLI`) |
| R-003 | The system SHALL evolve Section 1 (Rule 1) in `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, and `contract.md.tmpl` to establish the Progressive Disclosure Protocol (reading passport first, then active task docs, fetching living docs on-demand with canonical citations `TECH_STACK.md` Decision register §6, verified facts §7; `SKILLS.md` §1, §8), while strictly preserving the startup declaration and conflict-halt instruction. | MUST | Check test (`TestCheckContractsMatch`) verifying contract parity |
| R-004 | The system SHALL update base prompt templates (`prompt.md.tmpl` preserving `## 1. Step 0 - read the project` and $\ge 4,096$-byte invariant prefix floor; `prompt_compact.md.tmpl` preserving $\le 300$-word ceiling and $\ge 60\%$ reduction) to incorporate passport progressive disclosure orientation. | MUST | Template & prompt tests (`TestPromptTemplateOrder`, `TestPromptCompact`, `TestPromptStaticPrefix`) |
| R-005 | The system SHALL implement a mechanical validation check (`passport-consistency`) in `internal/check/passport.go` that verifies passport generation, budget compliance ($\le 500$ tokens / $\le 2,000$ bytes), `ROADMAP.md` active task synchronization, and valid file links, skipping gracefully when `ROADMAP.md` is absent. | MUST | `TestCheckPassport` & `make build && ./bin/sdd check --strict` |
| R-006 | The system SHALL update `skills/task-coordinator/SKILL.md`, `internal/skill/export.go`, and canonical skill templates, bump `templates.Version` to `2.11.0`, verify temporary kit regeneration (`sdd init --yes` + `sdd check` with 0 errors), and verify repository strict check (`./bin/sdd check --strict` with 0 errors) per Hard Rule 16. | MUST | Version test (`TestVersion`), export test (`TestCanonicalTaskCoordinatorSkillValid`), and temp kit audit |
| R-007 | The system SHALL maintain 100% Go standard library purity (`D-007`), execute race-clean (`-race`), and pass determinism verification (`-count=2`). | MUST | `make verify`, `go test -race ./...`, `go test -count=2 ./...` |

## 5. Non-functional requirements
- **Determinism (Hard Rule 8):** Injected clock (`time.Time`) ensures byte-identical passport output across repeated runs on identical repository state; zero clock calls (`time.Now()`) inside `Generate`. If `now.IsZero()`, normalizes deterministically to `time.Unix(0, 0).UTC()`.
- **Performance:** `sdd passport` execution completes in $< 50$ms (verified via `BenchmarkGenerate`).
- **Fail-safe behaviour:** If an optional living document is missing or malformed, the generator omits the section gracefully without crashing.
- **Budget Ceiling:** The rendered Markdown micro-index must strictly stay under 500 estimated tokens (2,000 UTF-8 bytes).

## 6. Constraints and assumptions
- All living document file references in the passport must be relative to the project root.
- The startup declaration requirement in contract Rule 1 (stating active task or "none", files/sections read, and ROADMAP status, with immediate halt on conflict) is strictly preserved.

## 7. Open questions
(None — all architectural requirements settled in D-050).

## 8. Acceptance criteria
- Given an SDD repository with living documents, when running `sdd passport`, then a formatted Markdown micro-index under 500 estimated tokens and 2,000 bytes is displayed.
- Given an invocation of `sdd passport --json`, then a valid JSON document matching the `Passport` schema is returned.
- Given an invocation of `sdd status --compact`, then the compact project passport is rendered.
- Given an agent reading the passport at session startup, then the agent obtains mission, stack, milestone, active task, and active feedback rules in under 500 tokens.
- Given the contract matching check `TestCheckContractsMatch`, then all agent files and `contract.md.tmpl` match with 0 errors.
- Given `sdd check --strict` on the repository, then the check passes with 0 errors and 0 warnings.
