# Requirements — T-001 sdd v1: init, check, feature new, prompt, version

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-10-26, in conversation; see plan.md §10)
Roadmap ref: `ROADMAP.md` T-001

## 1. Goal
Deliver a standard-library-only Go command, `sdd`, that generates, checks, and extends the spec-driven development kit for any greenfield or brownfield project, and prove it on its own repository.

## 2. Context and references
- Mission principles touched: MISSION §5 (deterministic, fail safe, never overwrite human work).
- Decisions relied on: D-006 to D-014 (all Approved, human decisions X1 to X9).
- Depends on: T-000.

## 3. Scope
**In scope**
- Commands `init`, `check`, `feature new`, `prompt`, `version`.
- Embedded templates with an override directory; packs for Go, Java, Python, TypeScript, and a generic fallback.
- The manifest `.sdd/manifest.json`.
- Generating this repository's own kit with `sdd init` and keeping it passing `sdd check --strict`.

**Out of scope**
- `sdd upgrade` (T-003) and `sdd adopt` (T-004).
- Windows support; per-agent permission settings; any network access or AI model call.
- Verification on macOS, Go 1.27, and golangci-lint (T-002).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | `sdd init` SHALL create CLAUDE.md, GEMINI.md, AGENTS.md, MISSION.md, TECH_STACK.md, ROADMAP.md, SKILLS.md, README.md, the three feature templates, and `.sdd/manifest.json` in a greenfield project. | MUST | test: TestGreenfieldCreatesTheWholeKit, TestInitYesGreenfieldThenCheckPasses |
| R-002 | `sdd init` SHALL never overwrite or delete an existing file unless `--force` is given, and SHALL never modify an existing README.md even with `--force`. | MUST | test: TestBrownfieldNeverTouchesExistingFiles, TestExistingDocsAreSkippedWithoutForce, TestForceReplacesWithBackupButNeverReadme, mutation checks |
| R-003 | For an existing agent file without a managed block, `sdd init` SHALL append one managed block, keep all existing text, and be idempotent on a second run. | MUST | test: TestMergeIntoExistingAgentFileIsIdempotent, TestBrownfieldKeepsExistingFilesAndMergesAgentFile |
| R-004 | With `--force`, `sdd init` SHALL back up every file it replaces under `.sdd/backup/<timestamp>/` and SHALL refresh only the managed block of an agent file, keeping text outside it. | MUST | test: TestForceReplacesWithBackupButNeverReadme, TestForceReinitBacksUpAndKeepsReadme |
| R-005 | `sdd init` SHALL refuse to run in the filesystem root or the home directory, when already initialised (without `--force`), without a terminal and without `--yes`, when a hand-made kit (all four core documents) exists without a manifest (unless `--force`), and with invalid answers (unknown language or agent, newline, pipe, or backtick in table fields, bad mode). | MUST | test: TestInitRefusals, TestInitRefusesHandMadeKitWithoutManifest, TestNormalizeRejectsBadInput |
| R-006 | `sdd init --dry-run` SHALL print the plan and write nothing. | MUST | test: TestInitDryRunWritesNothing |
| R-007 | In a terminal, `sdd init` SHALL prompt with detected defaults, re-ask on invalid choices (three attempts), show the plan, and write only after an explicit yes; end of input or "no" SHALL write nothing. | MUST | test: TestInteractiveInit, TestInteractiveAbortWritesNothing, TestInteractiveEndOfInputDefaultsButNeverConfirms; demo: real pseudo-terminal run |
| R-008 | Detection SHALL be read-only and report languages, suggested mode, suggested commands, existing documents, CI, and manifests, ignoring vendored and build directories, and SHALL mark nothing as verified. | MUST | test: detect package tests |
| R-009 | All generated output SHALL be deterministic: two identical runs give byte-identical files. | MUST | test: TestOutputIsDeterministic, TestInitIsDeterministic, TestCheckIsDeterministic, TestDetectIsDeterministic |
| R-010 | The manifest SHALL record the tool and template versions, the answers, and for every file its kind, state, and a hash of the generated content; a freshly generated kit SHALL show no drift. | MUST | test: TestManifestRecordsWhatHappened, TestFreshKitHasNoErrors |
| R-011 | `sdd check` SHALL report: missing required files, differing agent contracts, unresolved D- and T- references, broken tables, unbalanced fences, a missing feedback-rules section, invalid statuses, duplicate task IDs, feature-folder problems, blank approval records and leftover not-yet-run markers past their gates, and unresolved TODO placeholders; it SHALL exit 1 on errors (and on warnings with `--strict`). | MUST | test: TestDetectsProblems (14 cases), TestFreshKitHasNoErrors, TestManagedBlockParityIgnoresTextOutsideTheBlock, TestEditedGeneratedFilesAreReportedAsInfoOnly |
| R-012 | `sdd check` SHALL accept the real, hand-made dbguard kit with no errors. | MUST | test: TestRealWorldDbguardKitPasses; demo: CLI run on the kit |
| R-013 | `sdd feature new <T-xxx> <slug>` SHALL create the task folder from the project's own templates only when the task exists in ROADMAP.md, no folder exists for it, and the ID and slug are valid, and SHALL move Not started to Specifying. | MUST | test: feature package tests, TestFeatureNewAndPromptWorkflow |
| R-014 | `sdd prompt <agent>` SHALL print a handover prompt naming the project, the agent file, the open tasks, and the focus task, and SHALL fail clearly when the kit or task is missing. | MUST | test: TestFeatureNewAndPromptWorkflow |
| R-015 | Template files in an override directory SHALL replace the embedded file of the same path, other files SHALL fall back to the embedded ones, and a broken template SHALL be an error, never a silent default. | MUST | test: TestTemplateOverrideWinsPerFile, TestBrokenOverrideTemplateIsAnError, TestRenderErrors, TestOverrideErrors |
| R-016 | Language packs for Go, Java, Python, TypeScript, and the generic fallback SHALL each provide standards, a contract summary, and a skill map, and rendered files SHALL contain no template leftovers. | MUST | test: TestEveryLanguageHasACompletePack, TestRenderedFilesHaveNoTemplateLeftovers |
| R-017 | The module SHALL use only the Go standard library and build without cgo. | MUST | inspection: go.mod, file type of the built binary |
| R-018 | This repository's own kit SHALL be generated by `sdd init` and pass `sdd check --strict`. | MUST | demo: command output in validation.md |
| R-019 | The tool SHALL build and pass its checks on the maintainer's macOS machine with Go 1.27 and golangci-lint v2. | SHOULD | NOT VERIFIED here; assigned to T-002 |

## 5. Non-functional requirements
- **Determinism:** time is injected; lists are in canonical order.
- **Fail safe:** every invalid input is an error with a clear message; exit codes are 0 success, 1 failure or findings, 2 usage error.
- **Safety of writes:** every write is atomic (temporary file, then rename).
- **Offline:** no network access at build or run time.

## 6. Constraints and assumptions
- Built and tested with Go 1.22.2 on Linux in the build sandbox; `go.mod` says `go 1.22` so that could be verified there (D-015).
- golangci-lint is not available in the sandbox, so the lint configuration is unverified.

## 7. Open questions
None. D-015 to D-017 are Proposed and are decided at the named tasks; none blocks this task.

## 8. Acceptance criteria
- Given an empty directory, when `sdd init --yes` runs, then the whole kit exists and `sdd check` reports 0 errors.
- Given an existing project, when `sdd init --yes` runs, then no existing file is modified except that the agent files gain a managed block.
- Given this repository, when `sdd check --strict` runs, then it exits 0.
