# MISSION — sdd

> **ALWAYS BE DETERMINISTIC. Human approval resolves every decision.**

Status: living document · Last updated: 2026-10-26 · Maintainer: Vikash (human)

## 1. One-liner

Generate and check a spec-driven development kit (agent contracts, living documents, task templates) for any project, so any AI coding agent can take over under human-approved gates.

## 2. The problem

Giving an AI coding agent a project without a fixed contract produces inconsistent behaviour: it starts coding before the plan is approved, commits on its own, forgets earlier decisions, and every new session starts from zero. Writing the contract, the living documents, and the task templates by hand for each project (as was done for the first project, dbguard) is slow, and the copies drift apart.

## 3. Who it is for

- First: Vikash, starting new backend projects and taking over existing ones in Go, Java, Python, and TypeScript.
- Then: any developer who works with AI coding agents and wants approvals, decisions, and feedback to be explicit and recorded.

## 4. Scope

**In scope (v1)**

- `sdd init` for greenfield and brownfield projects: detect, ask, show a plan, write without overwriting, record a manifest.
- `sdd check`: mechanical consistency checks of a kit.
- `sdd feature new`: create a task folder from the project's own templates.
- `sdd prompt`: print a handover prompt for an agent.
- Embedded templates with an override directory; language packs for Go, Java, Python, TypeScript, and a generic fallback.
- Agent contracts for Claude (`CLAUDE.md`), Gemini (`GEMINI.md`), and any agent that reads `AGENTS.md`.

**Out of scope for now** (see the ROADMAP Parking lot)

- Understanding a codebase (the agent does that, under the gates, in the Discovery task).
- Calling any AI model, or any network access.
- Upgrading existing kits (T-003) and adopting kits made before sdd (T-004).
- Windows support; generating per-agent permission settings that deny git writes.

## 5. Principles

1. **Deterministic.** The same inputs give byte-identical output. Humans resolve ambiguity.
2. **Fail safe.** Unknown or ambiguous means refuse or stop and ask, never guess.
3. **Human in the loop for the irreversible.** Writing files is previewed and confirmed; nothing is committed by the tool.
4. **Small and shippable.** Thin vertical slices; each task independently reviewable and committable.
5. **Never overwrite human work.** Existing files are skipped or merged through a marked block; replacements always leave a backup.

## 6. Known limits

- Detection reads file names and a few manifests only; every detected value is a suggestion and is marked unverified.
- The language packs are a baseline for a human to refine; they are not a substitute for a team's own standards.
- The rules are markdown. An agent can ignore them; sdd does not enforce them (for example it does not block `git commit`).
- v0.1.0 is built and tested on Go 1.22 and Linux. macOS, Go 1.27, and golangci-lint v2 are not yet verified (T-002).

## 7. Success criteria

| # | Criterion | Measure |
|---|---|---|
| S1 | `init` never destroys existing content | Tests prove existing files are skipped or merged, README.md is never modified, and `--force` always leaves a backup |
| S2 | Generated kits are internally consistent | `sdd check` reports 0 errors on fresh kits for every language and mode, and on the real dbguard kit |
| S3 | Deterministic | Two identical runs produce byte-identical files |
| S4 | Works on the maintainer's machine | `make check` passes and the macOS binary runs (T-002) |
| S5 | Useful on real projects | Used on dbguard and at least two team repositories without hand-fixing anything except the TODO placeholders |

## 8. Hypotheses still to validate (NOT facts)

- **H1** — A generated kit is enough for another agent to follow the workflow without extra instruction. Test: hand a fresh project to Gemini with `sdd prompt gemini`.
- **H2** — Existing spec-driven tools (GitHub Spec Kit, OpenSpec) do not provide this exact contract (exact approval phrases, human-only commits, decision and feedback registers). Reported to be widely used; not compared in detail. If one does, adopt it instead of maintaining this.
- **H3** — The baseline Java, Python, and TypeScript packs are good enough to start with. Test: use them on the team's repositories and record corrections as feedback rules.

## 9. Business intent

A personal productivity tool first, with no commercial plan. If it proves useful beyond the maintainer, sharing it is a later, separate decision.
