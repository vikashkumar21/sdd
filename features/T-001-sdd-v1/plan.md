# Plan — T-001 sdd v1: init, check, feature new, prompt, version

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Implemented

## 1. Approach summary
A standard-library-only Go module of small packages: `detect` (read-only scan), `templates` (embedded files, override directory, packs), `scaffold` (answers, a pure plan, an applying step, the manifest), `check` (rules), `feature` (task folders), `roadmap` and `block` (parsing and markers), `fsutil` (atomic writes), and `cli` (commands). Planning never writes; one function applies the plan.

## 2. Decisions (new or affected)

| ID | Decision | Options | Recommendation | Status |
|---|---|---|---|---|
| D-006 to D-014 | Human decisions X1 to X9 (build own tool, Go standard library, name `sdd`, v1 scope, brownfield policy, four language packs, three agent files, overrides and manifest, repository and module) | see TECH_STACK §6 | as decided | **Approved** |
| D-015 | `go` directive 1.22 versus 1.27 | A) 1.22, verifiable in the sandbox; B) 1.27 | A until verified on the Mac | Proposed, decided at T-002 |
| D-016 | Template syntax `{{ }}`, strict | A) text/template; B) a custom delimiter | A | Proposed |
| D-017 | Manifest schema 1 and the managed-block marker format | fixed text | as implemented | Proposed |

## 3. Design
See `TECH_STACK.md` §3 and §4 (architecture diagram, data flow, directory layout).

## 4. Implementation steps (as carried out)

| # | Step | Files | Proving check |
|---|---|---|---|
| 1 | Foundations: module, version, managed-block and roadmap packages | go.mod, internal/version, internal/block, internal/roadmap | go vet; package tests |
| 2 | Detection | internal/detect | table-driven tests on fixtures |
| 3 | Template engine and five language packs | internal/templates | pack and render tests |
| 4 | Agent contract and the five document templates, handover prompt, feature templates | internal/templates/data | rendered-output tests; manual read-through of rendered files |
| 5 | Scaffold: answers, plan, apply, manifest | internal/scaffold, internal/fsutil | never-overwrite, merge, force, determinism tests; mutation checks |
| 6 | Checker | internal/check | 14 problem cases plus the real dbguard kit as a regression fixture |
| 7 | feature new | internal/feature | folder, token, and status tests |
| 8 | CLI and main | internal/cli, cmd/sdd | end-to-end tests with scripted input; real pseudo-terminal run |
| 9 | Build tooling, README, dogfood | Makefile, .golangci.yml, .gitignore, README.md, the repository's own kit | make verify; sdd check --strict |

## 5. Files to create or modify
Everything under `cmd/` and `internal/`, plus Makefile, .golangci.yml, .gitignore, README.md, and the kit files (CLAUDE.md, GEMINI.md, AGENTS.md, MISSION.md, TECH_STACK.md, ROADMAP.md, SKILLS.md, features/).

## 6. Dependencies
None. Standard library only (D-007).

## 7. Living docs to update
MISSION, TECH_STACK (decisions and verified facts), ROADMAP (status), SKILLS (pitfalls), README: all done in this task.

## 8. Risks, mitigations, rollback
- **Unverified on macOS, Go 1.27, golangci-lint:** assigned to T-002; recorded as NOT VERIFIED.
- **The marker format becomes a compatibility surface** once other projects contain it (D-017): decide before wide use.
- **Template changes must bump `templates.Version`:** convention recorded in the agent contract.
- **Rollback:** delete the repository; the tool writes nothing outside the projects it is run in.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active yet (the register is empty).

## 10. Approval record
Approved by: Vikash (in conversation)   Date: 2026-10-26   Phrase received: decisions X1 to X9 ("others default to your recommendations"). This is not the exact `APPROVED T-001` phrase; the exact-phrase gates were not yet applied to this repository when the plan was approved, and this is recorded here for honesty.
