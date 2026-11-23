---
status: draft
---
# Requirements — T-054 Deep domain flow documentation and visual Mermaid diagrams

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-22)
Roadmap ref: `ROADMAP.md` T-054
Decision ref: `TECH_STACK.md` D-046 (Approved 2026-11-22)

## 1. Goal

Provide automated scaffolding, templates, CLI ergonomics, and mechanical validation for deep domain flows and visual architecture documentation (`docs/architecture/` and `docs/business_flows/`) with standardized Mermaid topology, entity-relationship, and sequence diagrams.

## 2. Context and references

- Mission / principles: Principle 1 (Deterministic by construction), Principle 2 (Fail safe), Principle 5 (Never overwrite human work).
- Decisions: D-007 (Standard library only), D-024 (agentskills.io standard), D-033 (Document lifecycle frontmatter), D-046 (Domain flow scaffolding and Mermaid diagram standards).
- Depends on: T-053 (Adversarial discovery ensemble).

## 3. Scope

### In scope
- CLI commands under `sdd doc`:
  - `sdd doc init [--dir <path>] [--force] [--dry-run] [--json]`: Scaffolds standard `docs/architecture/` (`overview.md`, `data_model.md`) and `docs/business_flows/` (`overview.md`, `sample_flow.md`) with valid document lifecycle frontmatter and Mermaid diagrams.
  - `sdd doc flow <name> [--title <title>] [--dir <path>] [--force] [--dry-run] [--json]`: Scaffolds a new domain flow document in `docs/business_flows/<name>.md` with sequence diagram, state transitions, and lifecycle metadata.
- Embedded starter templates in `internal/templates/data/docs/`:
  - `architecture/overview.md.tmpl`: System topology & component boundaries (`flowchart TD`).
  - `architecture/data_model.md.tmpl`: Entity-relationship schemas & lifecycles (`erDiagram`).
  - `business_flows/overview.md.tmpl`: Domain capability catalog.
  - `business_flows/flow.md.tmpl`: Scenario flow with actors, sequence diagram (`sequenceDiagram`), and state transitions (`stateDiagram-v2`).
- Mechanical validation in `sdd check`:
  - `mermaid-syntax`: Validate Mermaid fenced code blocks (` ```mermaid ... ``` `) across markdown files (valid diagram declaration, non-empty, balanced brackets/parentheses/braces, comment handling, and quoted labels).
  - Directory traversal: Extend `markdownFiles()` directory scanner in `internal/check/check.go` to include `docs/`, ensuring all markdown files in `docs/**/*.md` are inspected for `mermaid-syntax`, code fences, tables, and lifecycle frontmatter (`lifecycle-*`).
- Discovery prompt enrichment:
  - Update `sdd prompt discovery` to direct codebase-analyst and discovery-challenger to inspect and synthesize `docs/architecture/` and `docs/business_flows/`.
- Manifest integration:
  - Register generated doc templates in `.sdd/manifest.json`.
- Template version bump:
  - Bump `templates.Version` from `2.2.0` to `2.3.0` and bump contract markers per Hard Rule 16.

### Out of scope
- Generating headless image files (PNG/SVG) via external tools like Chromium or `mermaid-cli` (violates D-007 stdlib purity and zero-network principles; Mermaid renders natively in GitHub, GitLab, and IDE Markdown previews).
- Dynamic runtime AST-to-Mermaid code generation (agents author diagrams using the standard templates under Gate 1 / Gate 2).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL provide `sdd doc init` to scaffold `docs/architecture/overview.md`, `docs/architecture/data_model.md`, `docs/business_flows/overview.md`, and `docs/business_flows/sample_flow.md` with standard Mermaid diagrams and lifecycle frontmatter. | MUST | test / demo |
| R-002 | The system SHALL provide `sdd doc flow <name> [--title <title>]` to scaffold a new domain flow document in `docs/business_flows/<name>.md` with Mermaid sequence and state diagrams. | MUST | test / demo |
| R-003 | The system SHALL enforce fail-safe file operations for `sdd doc` commands: refuse unsafe roots, skip existing files unless `--force` is provided, create backups in `.sdd/backup/` when overwriting, and support `--dry-run` and `--json`. | MUST | test |
| R-004 | The system SHALL validate Mermaid code blocks in `sdd check` via a deterministic scanner (`mermaid-syntax`) and extend `sdd check` markdown directory traversal to inspect `docs/` for `mermaid-syntax`, code fences, tables, and lifecycle frontmatter. | MUST | test |
| R-005 | The system SHALL update `sdd prompt discovery` context to instruct codebase analysts and discovery challengers to synthesize and cross-verify `docs/architecture/` and `docs/business_flows/`. | MUST | test / inspection |
| R-006 | The system SHALL bump `templates.Version` to `2.3.0`, update contract markers to `templates=2.3.0`, maintain contract parity across `AGENTS.md`, `GEMINI.md`, and `CLAUDE.md`, and verify clean `sdd upgrade` migrations and fresh kit generation. | MUST | test |
| R-007 | The system SHALL maintain 100% standard library purity (`D-007`) and zero external dependencies. | MUST | test |

## 5. Non-functional requirements

- **Determinism:** Scaffolding outputs and check scanner are 100% deterministic with injected clocks and canonical path ordering.
- **Fail-safe:** Ambiguous or unknown commands fail with exit code 1 or 2; overwrites require explicit `--force`.
- **Performance budget:** Check scanner completes in < 50ms for repository markdown files.
- **Security & Hygiene:** Path traversal guarded via `filepath.IsLocal` and `os.Root` confinement. No secrets or personal paths (Rule F-002).

## 6. Constraints and assumptions

- Markdown renderers (GitHub, GitLab, Obsidian, VS Code) render Mermaid blocks natively without third-party CLI binaries.
- All documents in `docs/` conform to the document lifecycle standard (`D-033`).

## 7. Open questions

(none)

## 8. Acceptance criteria

- **AC-1 (`sdd doc init`):** Running `sdd doc init --dry-run` shows planned creation of 4 files under `docs/`. Running without `--dry-run` writes the files with valid frontmatter and Mermaid blocks. Running again without `--force` skips them without error.
- **AC-2 (`sdd doc flow`):** Running `sdd doc flow user-login --title "User Authentication Flow"` creates `docs/business_flows/user-login.md` with sequence diagram and lifecycle frontmatter.
- **AC-3 (`mermaid-syntax` Validation):** `sdd check` detects unclosed brackets or unrecognized diagram headers in Mermaid blocks with clear line numbers and file paths. Valid Mermaid blocks pass with 0 errors.
- **AC-4 (Discovery Enrichment):** `sdd prompt discovery` includes instructions and checklist items for domain flow documentation.
- **AC-5 (Template Version & Upgrade):** `templates.Version` is `2.3.0`; `sdd upgrade` and fresh kit generation pass strict check.
