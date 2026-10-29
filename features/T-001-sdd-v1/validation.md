# Validation — T-001 sdd v1: init, check, feature new, prompt, version

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Complete except R-019 (NOT VERIFIED, assigned to T-002)

## 0. Environment

| Item | Value |
|---|---|
| OS | Linux amd64 (build sandbox) |
| `go version` | go1.22.2 linux/amd64 |
| golangci-lint | not installed: NOT VERIFIED |
| macOS | not available: NOT VERIFIED |
| Run by | the agent, in the build sandbox |

## 1. Traceability matrix

| Req | Method | Evidence | Result |
|---|---|---|---|
| R-001 | test | TestGreenfieldCreatesTheWholeKit, TestInitYesGreenfieldThenCheckPasses: pass | PASS |
| R-002 | test and mutation | scaffold and CLI tests pass; overwriting existing docs, un-protecting README.md, and disabling the parity check were each injected into a scratch copy and each made tests fail | PASS |
| R-003 | test | TestMergeIntoExistingAgentFileIsIdempotent, TestBrownfieldKeepsExistingFilesAndMergesAgentFile: pass; real run on a Java project kept existing CLAUDE.md, AGENTS.md, README.md | PASS |
| R-004 | test | TestForceReplacesWithBackupButNeverReadme, TestForceReinitBacksUpAndKeepsReadme: pass | PASS |
| R-005 | test and demo | TestInitRefusals, TestInitRefusesHandMadeKitWithoutManifest, TestNormalizeRejectsBadInput (8 cases): pass; `sdd init` on a copy of the real dbguard kit exited 1 and left the copy byte-identical | PASS |
| R-006 | test | TestInitDryRunWritesNothing: pass | PASS |
| R-007 | test and demo | TestInteractiveInit, TestInteractiveAbortWritesNothing, TestInteractiveEndOfInputDefaultsButNeverConfirms: pass; a real run under script(1) captured every scripted answer into the manifest | PASS |
| R-008 | test | detect tests (languages, mode, ignored directories, make targets, existing files, CI, determinism): pass | PASS |
| R-009 | test | four determinism tests pass; the whole suite run twice gave identical output | PASS |
| R-010 | test | TestManifestRecordsWhatHappened pass; TestFreshKitHasNoErrors asserts no drift (this caught and fixed a real bug during development) | PASS |
| R-011 | test | TestDetectsProblems (14 cases) and four further check tests: pass | PASS |
| R-012 | test and demo | TestRealWorldDbguardKitPasses pass; `sdd check` on the dbguard kit printed 0 error(s), 0 warning(s), 1 note(s) | PASS |
| R-013 | test | feature package tests (6) and TestFeatureNewAndPromptWorkflow: pass; this repository's own T-001 folder was created with `sdd feature new` | PASS |
| R-014 | test | TestFeatureNewAndPromptWorkflow: pass | PASS |
| R-015 | test | TestTemplateOverrideWinsPerFile, TestBrokenOverrideTemplateIsAnError, TestRenderErrors, TestOverrideErrors: pass | PASS |
| R-016 | test | TestEveryLanguageHasACompletePack, TestRenderedFilesHaveNoTemplateLeftovers: pass | PASS |
| R-017 | inspection | go.mod has no require lines and there is no go.sum; linux binary is "statically linked" | PASS |
| R-018 | demo | `sdd check --strict` on this repository: see §2 | PASS |
| R-019 | not run on the target | no macOS machine, no Go 1.27, no golangci-lint in the sandbox | NOT VERIFIED (T-002) |

## 2. Commands run

| Command | Result |
|---|---|
| `gofmt -s -l cmd internal` | no output (clean) |
| `go vet ./...` | ok |
| `go test ./... -count=1` (run 1 and run 2) | all 8 test packages ok; output identical between the runs |
| `go test ./... -count=1 -v`, passing results counted | 94 PASS, 0 FAIL |
| `go test -race ./... -count=1` | all 8 test packages ok |
| `go test ./... -cover` | block 94.7%, check 91.0%, cli 86.0%, detect 86.0%, feature 77.8%, roadmap 93.2%, scaffold 79.9%, templates 86.3%; cmd/sdd and fsutil have no tests of their own (main wiring was exercised by the real pseudo-terminal run; fsutil by every apply test) |
| `make dist` | four binaries built (about 3.1 to 3.3 MB each) plus SHA256SUMS; darwin_arm64 and darwin_amd64 are Mach-O executables, linux_amd64 is a statically linked ELF; only the linux_amd64 binary was executed (`sdd 0.1.0 (templates 1.0.0)`) |
| `sdd check --strict` on this repository | 0 errors, 0 warnings (one INFO note that generated files were edited, which is expected) |

## 3. Manual checks
- Rendered a brownfield Java and Go project and read every generated file: structure and content correct; found and fixed a wording flaw in the TECH_STACK template.
- Ran the real binary on Java/Maven, Python, and TypeScript fixtures and on the real dbguard kit.

## 4. Negative and failure cases
Covered by the tests in §1: unsafe roots, no terminal without `--yes`, already initialised, invalid answers, unknown language, EOF and "no" at the confirmation, broken override template, missing templates for `feature new`, unknown or duplicate tasks, 14 kit-corruption cases for `check`.

## 5. Determinism check
The full test suite was run twice with identical output; init output was compared byte for byte between two runs in scaffold and CLI tests.

## 6. Deviations, known limitations, and feedback-rule compliance
- Deviation from the sketch in the approved plan: `internal/block`, `internal/roadmap`, `internal/fsutil` were added as separate packages, and `sdd prompt` accepts `--templates`. Both are small and recorded in TECH_STACK §4.
- Bugs found and fixed during the work (by tests, by `sdd check` on its own documents, and by a dry run on the real dbguard kit): a contract example pointing at a task that does not exist in a fresh roadmap; a manifest hash that covered the block but was labelled as a whole file; a roadmap test miscounting lines; `init` not refusing a hand-made kit without a manifest, which would have appended a second contract to its agent files (now refused, with a test).
- Limitations: R-019 is NOT VERIFIED; the golangci-lint configuration has not been run; the language packs are a baseline for the human to review.

Feedback-rule compliance (SKILLS.md, "Human feedback rules"): none active.

## 7. Definition of done
- [x] Every requirement has evidence, or is explicitly NOT VERIFIED (R-019)
- [x] Build, vet, format check, and tests pass (commands above)
- [x] Negative and failure cases tested
- [x] Determinism confirmed
- [x] No unapproved dependencies; no files outside the plan
- [x] Living docs updated
- [x] Compliant with every active feedback rule (none active)
- [x] Suggested commit message provided

## 8. Sign-off
Suggested commit message: `feat: add sdd v1 (init, check, feature new, prompt) with its own kit`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-001`
