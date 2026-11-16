---
status: active
task_id: T-048
last_reviewed: 2026-11-16
---
# Validation — T-048 Multi-agent coordinator contract and handover prompt template alignment

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Complete
Roadmap ref: `ROADMAP.md` T-048

## 1. Traceability matrix (R-xxx -> evidence)

| ID | Method | Command | Expected | Actual | Status |
|---|---|---|---|---|---|
| R-001 | test / inspection | `go test -v ./internal/templates -run TestContractTemplateCoordinatorAndWorkflow` | Contract template Section 2 details Coordinator and Subagent actions across all phases and references tasks/<typology>/ | PASS: Section 2 table contains Coordinator & Subagent Actions across all phases and tasks/<typology>/ paths | Verified |
| R-002 | test / inspection | `go test -v ./internal/templates -run TestContractTemplateCoordinatorAndWorkflow` | Contract template Section 4 Rule 13/14 codifies Task Coordinator role dispatch per `.sdd/roles.json` | PASS: Rules 1 and 13 codify Task Coordinator and canonical role dispatches | Verified |
| R-003 | test / inspection | `go test -v ./internal/cli -run TestPromptCoordinatorBriefing` | Handover prompt briefs incoming agent as Task Coordinator dispatching `.sdd/roles.json` roles | PASS: prompt briefs agent as Task Coordinator with role orchestration and tasks/<typology>/ paths | Verified |
| R-004 | test | `go test -v ./internal/templates -run TestVersion` | `templates.Version == "2.0.0"` | PASS: templates.Version is 2.0.0 | Verified |
| R-005 | test | `go test -v ./internal/check -run "TestDetectsProblems/agent_contract_drift" && ./bin/sdd check --strict` | `AGENTS.md`, `CLAUDE.md`, `GEMINI.md` synchronized in 100% lockstep (`templates=2.0.0`) | PASS: contract parity passes; 0 errors, 0 warnings | Verified |
| R-006 | test | `go test -v ./internal/upgrade -run TestUpgradeContractToV2` | `sdd upgrade` detects `templates=1.9.0` contracts and updates them to `2.0.0` | PASS: CLAUDE.md upgraded in place to templates=2.0.0 preserving custom sections | Verified |
| R-007 | test | `tmpdir=$(mktemp -d) && ./bin/sdd init --dir "$tmpdir" --name testkit --lang go --yes >/dev/null && ./bin/sdd check --dir "$tmpdir" && rm -rf "$tmpdir"` | Fresh kit generated with `templates=2.0.0` contracts passes `sdd check` (0 errors) | PASS: kit generated with templates=2.0.0 contracts passes sdd check with 0 errors | Verified |
| R-008 | test / inspection | `make build && ./bin/sdd version` | `sdd 1.7.0 (templates 2.0.0)` | PASS: sdd 1.7.0 (templates 2.0.0) | Verified |

## 2. Verification log

```bash
$ go test -v ./internal/templates -run TestContractTemplateCoordinatorAndWorkflow
=== RUN   TestContractTemplateCoordinatorAndWorkflow
--- PASS: TestContractTemplateCoordinatorAndWorkflow (0.00s)
PASS
ok  	github.com/vikashkumar21/sdd/internal/templates	0.092s

$ go test -v ./internal/cli -run TestPromptCoordinatorBriefing
=== RUN   TestPromptCoordinatorBriefing
--- PASS: TestPromptCoordinatorBriefing (0.12s)
PASS
ok  	github.com/vikashkumar21/sdd/internal/cli	3.263s

$ go test -v ./internal/templates -run TestVersion
=== RUN   TestVersion
--- PASS: TestVersion (0.00s)
PASS
ok  	github.com/vikashkumar21/sdd/internal/templates	0.088s

$ go test -v ./internal/check -run "TestDetectsProblems/agent_contract_drift" && ./bin/sdd check --strict
=== RUN   TestDetectsProblems
=== RUN   TestDetectsProblems/agent_contract_drift
--- PASS: TestDetectsProblems (0.13s)
    --- PASS: TestDetectsProblems/agent_contract_drift (0.00s)
PASS
ok  	github.com/vikashkumar21/sdd/internal/check	0.443s
INFO  .sdd/manifest.json  7 generated file(s) have changed since sdd generated them (expected: you edit them)

0 error(s), 0 warning(s), 1 note(s)

$ go test -v ./internal/upgrade -run TestUpgradeContractToV2
=== RUN   TestUpgradeContractToV2
--- PASS: TestUpgradeContractToV2 (0.15s)
PASS
ok  	github.com/vikashkumar21/sdd/internal/upgrade	0.484s

$ tmpdir=$(mktemp -d) && ./bin/sdd init --dir "$tmpdir" --name testkit --lang go --yes >/dev/null && ./bin/sdd check --dir "$tmpdir" && rm -rf "$tmpdir"
WARN  (project)  22 unresolved TODO(human) placeholder(s) (MISSION.md: 11, TECH_STACK.md: 8, ROADMAP.md: 3)
0 error(s), 1 warning(s), 0 note(s)

$ make build && ./bin/sdd version
go build -trimpath -ldflags "-s -w -X github.com/vikashkumar21/sdd/internal/version.Version=1.7.0" -o bin/sdd ./cmd/sdd
sdd 1.7.0 (templates 2.0.0)

$ make verify
go vet ./...
go test ./... -count=1
[All 28 packages PASS]

$ make lint
golangci-lint run
0 issues.

$ go test -count=2 ./internal/templates ./internal/cli ./internal/upgrade ./internal/check
ok  	github.com/vikashkumar21/sdd/internal/templates	0.072s
ok  	github.com/vikashkumar21/sdd/internal/cli	22.847s
ok  	github.com/vikashkumar21/sdd/internal/upgrade	10.568s
ok  	github.com/vikashkumar21/sdd/internal/check	22.526s
```

## 3. Non-functional and Negative test cases

| Test Case | Method | Expected | Actual | Status |
|---|---|---|---|---|
| Legacy contract upgrade | `go test -v ./internal/upgrade -run TestUpgradeContractToV2` | Preserves edits outside managed block; upgrades inner contract to 2.0.0 | PASS: edits outside block preserved, inner contract upgraded | Verified |
| Contract parity drift | `go test -v ./internal/check -run "TestDetectsProblems/agent_contract_drift"` | Error flagged if any agent file contract deviates by a single byte | PASS: drift detected if contracts diverge | Verified |
| Strict check | `./bin/sdd check --strict` | 0 errors, 0 warnings on repository | PASS: 0 errors, 0 warnings | Verified |

## 4. Plan boundary compliance
- Only files listed in `plan.md` Section 6 modified.
- Zero unapproved file modifications.

## 5. Dependency hygiene
- Standard library only (`D-007`). Zero new dependencies.

## 6. Feedback rule compliance
- **F-001:** All review findings across all 3 lenses resolved before GATE 2; compliance recorded in `review.md`.

## 7. Determinism check
- Running tests twice (`-count=2`) produces identical pass/fail results.

## 8. Sign-off
Suggested commit message: `feat(contract): align embedded contract and handover prompt with multi-agent coordinator workflow (v1.7.0 / templates 2.0.0)`
Reviewed by: Vikash (human)   Date: 2026-11-16   Reply received: `COMMITTED T-048`
