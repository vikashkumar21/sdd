# Validation — T-004 sdd adopt: write a manifest for kits created before sdd (P-06, for example dbguard)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Passed
Roadmap ref: `ROADMAP.md` T-004

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/adopt -run TestAdoptMissingCoreDocs` | Refuses adoption when core docs are missing | Missing core docs refused; hints for VISION/ARCHITECTURE provided | PASS |
| R-002 | test | `go test -v ./internal/adopt -run TestAdoptAlreadyAdopted` | Refuses adoption when manifest exists unless --force | Refused without --force; succeeds and creates backup with --force | PASS |
| R-003 | test | `go test -v ./internal/adopt -run TestAdoptDbguardKitAndCheck` | Detects present agent files and task templates | Discovered CLAUDE.md, GEMINI.md, core docs, and task templates | PASS |
| R-004 | test | `go test -v ./internal/adopt -run TestAdoptDbguardKitAndCheck` | Hashes computed and state marked existing | SHA256 hashes generated for all files; state set to "existing" | PASS |
| R-005 | test | `go test -v ./internal/adopt -run TestAdoptZeroMutation` | Zero files modified other than writing manifest | Before and after SHA256 hashes match identically for all project files | PASS |
| R-006 | test | `go test -v ./internal/cli -run TestCLIAdoptDryRun` | --dry-run prints manifest JSON without writing files | JSON printed to stdout; no .sdd/manifest.json created | PASS |
| R-007 | test | `go test -v ./internal/adopt -run TestAdoptDbguardKitAndCheck` | Adopted dbguard-kit passes sdd check with 0 errors | check.Run completed with 0 errors on adopted kit | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | PASS (fmt-check, vet, all tests pass) |
| `make check` | PASS (0 issues from golangci-lint, all tests pass) |
| `go run ./cmd/sdd check --strict` | PASS (0 errors, 0 warnings, 1 info note) |

## 3. Manual checks
- Verified `sdd adopt --dry-run` on a test kit prints valid manifest JSON.
- Verified byte-for-byte SHA256 integrity of all documentation files before and after `sdd adopt` in `TestAdoptZeroMutation`.

## 4. Negative and failure cases
- Missing core docs (`MISSION.md`, `TECH_STACK.md`, `ROADMAP.md`, `SKILLS.md`) refused with exit code 1.
- Targeting filesystem root refused with exit code 1.
- Existing `.sdd/manifest.json` refused without `--force`.

## 5. Determinism check
Ran `go test ./internal/adopt ./internal/cli` twice; outputs identical: yes. Evidence: 2 runs pass with identical test counts.

## 6. Deviations, known limitations, and feedback-rule compliance
None.

Feedback-rule compliance (SKILLS.md, "Human feedback rules"): none active.

## 7. Definition of done
- [x] Every requirement has evidence
- [x] Build, lint, and tests pass (commands above)
- [x] Negative and failure cases tested
- [x] Determinism confirmed
- [x] No files changed outside the plan; no unapproved dependencies
- [x] Living docs updated
- [x] Compliant with every active feedback rule
- [x] Suggested commit message provided

## 8. Sign-off
Suggested commit message: `feat(adopt): add sdd adopt command for pre-existing hand-crafted kits (T-004)`
Reviewed by: Vikash   Date: 2026-11-02   Reply received: `COMMITTED T-004`
