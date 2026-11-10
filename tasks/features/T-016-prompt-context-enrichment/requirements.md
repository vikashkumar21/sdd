# Requirements — T-016 Handover prompt enrichment for staged context

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-016

## 1. Goal
Enrich `sdd prompt` handover prompts with an inventory and extraction guidance for staged context artifacts in `.sdd/context/` so AI coding agents immediately discover transcripts, documents, raw notes, or Jira tasks, and systematically map them into the living documents (`MISSION.md`, `TECH_STACK.md`, `ROADMAP.md`, `SKILLS.md`, or task requirements) under human Gate 1 approval.

## 2. Context and references
- Mission / principles touched: MISSION §5 (Deterministic, Fail safe, Human in the loop), MISSION §7 (S1).
- Decisions this relies on:
  - D-007 (Standard library only)
  - D-015 (Approved: Go 1.27 toolchain)
  - D-018 (Approved: Ingestion and semantic synthesis architecture; agent extracts under Gate 1)
  - D-020 (Approved: Text & file context staged for agent reading)
- Depends on tasks: T-014 (Done).

## 3. Scope
**In scope**
- `internal/cli`:
  - Enhance `runPrompt` to inspect `.sdd/context/manifest.json` (via `context.ReadManifest`).
  - Pass structured context items into `promptData` for template rendering.
  - Distinguish general/project-level context from task-specific context (matching `--task` / `FocusTask`).
- `internal/templates/data/base/prompt.md.tmpl`:
  - Add a dedicated section `## Staged external context (.sdd/context/)` when staged artifacts exist.
  - Provide an itemized summary table/list of staged items: ID, source, origin, path, format, size, and associated task.
  - Render explicit step-by-step extraction instructions guiding the agent on how to categorize and synthesize the context:
    - Vision / Scope &rarr; `MISSION.md`
    - Decisions / Architecture &rarr; `TECH_STACK.md` (as `D-xxx` decision proposals or verified facts)
    - Tasks & Milestones &rarr; `ROADMAP.md` (`T-xxx` rows)
    - Standards / Preferences &rarr; `SKILLS.md`
    - Task-specific context &rarr; `features/T-xxx-<slug>/requirements.md`
  - Reiterate the core rule: **The agent proposes changes in draft docs under Gate 1; the human approves before any code or living doc changes.**
- Unit and CLI tests:
  - Test `sdd prompt` output with no context (preserves current clean prompt behavior).
  - Test `sdd prompt` output with general staged context (transcripts, meeting notes).
  - Test `sdd prompt` output with task-specific staged context.
  - Template override tests ensuring custom templates can access context data.

**Out of scope**
- Calling external LLMs or performing automated doc updates from within the Go CLI binary.
- Modifying markdown files directly during prompt generation (prompt remains strictly read-only stdout).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | `sdd prompt` SHALL read `.sdd/context/manifest.json` if present and expose staged items to the prompt template. | MUST | unit test: `commands_test.go` |
| R-002 | When staged context exists, `sdd prompt` output SHALL include an itemized list of staged artifacts (ID, source/origin, path, format, size). | MUST | CLI test: `TestPromptWithStagedContext` |
| R-003 | When no staged context exists, `sdd prompt` output SHALL omit the staged context section, maintaining clean backward-compatible output. | MUST | CLI test: `TestPromptWithoutContext` |
| R-004 | When a task-specific focus task is selected (`--task T-xxx`), `sdd prompt` SHALL highlight staged context specifically assigned to that task. | MUST | CLI test: `TestPromptWithTaskContext` |
| R-005 | The prompt template SHALL instruct the agent on semantic mapping rules (e.g. Scope -> MISSION.md, Architecture -> TECH_STACK.md, Tasks -> ROADMAP.md). | MUST | template inspection |
| R-006 | The prompt template SHALL explicitly instruct the agent that all synthesized requirements require human Gate 1 approval before proceeding. | MUST | template inspection |
| R-007 | `sdd prompt` SHALL remain strictly read-only and make no modifications to repository files or git state. | MUST | test: repository clean check |
| R-008 | `make check`, `go test -race ./...`, and `GOOS=windows go vet ./...` SHALL pass with zero issues. | MUST | command check |
| R-009 | All new tests SHALL be deterministic and repeatable across successive runs. | MUST | test: `go test ./internal/cli -count=2` |

## 5. Non-functional requirements
- **Determinism:** Staged context items are rendered in deterministic order (sorted by ID or manifest order).
- **Fail-safe:** If `.sdd/context/manifest.json` is corrupted or unreadable, `sdd prompt` prints a warning and falls back safely without failing.
- **Privacy & Security:** Prompt only summarizes paths, metadata, and labels; it does not dump multi-megabyte raw file dumps into chat prompts, keeping prompts within token budgets.

## 6. Constraints and assumptions
- Uses standard library only (D-007, Go 1.27).
- Relies on context manifest schema created in T-014.

## 7. Open questions
None.

## 8. Acceptance criteria
- Given a repository with staged context in `.sdd/context/`, running `sdd prompt gemini` prints an itemized context table with explicit guidance on where each type of information must be proposed.
- Given a repository without `.sdd/context/`, running `sdd prompt gemini` produces clean output without empty context sections.
- Given `make check`, all tests pass with 0 errors.
