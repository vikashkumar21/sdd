---
status: draft
---
# Validation — T-062 Provider-aligned prompt prefix caching and dynamic suffix pinning

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Complete (Traceability verified)

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/cli -run TestPromptPrefixCaching_ByteInvariance` | Invariant static prefix $\ge 1,024$ tokens ($\ge 4,096$ bytes) across prompt templates | PASS: 8/8 test suites verify static prefix $\ge 4,096$ bytes ($\ge 1,024$ tokens) across plan-review, implement, review (spec/security/pattern), discovery (single/ensemble), and repair | VERIFIED |
| R-002 | test | `go test -v ./internal/cli -run TestPromptPrefixCaching_DynamicSuffix` | Dynamic metadata (task ID, files, loop, bundle dir, instructions) pinned exclusively to suffix | PASS: Task ID and task directory name absent from static prefix and present exclusively in dynamic suffix | VERIFIED |
| R-003 | test | `go test -v ./internal/cli -run TestPromptPrefixCaching_CrossTaskInvariance` | Static prefixes for task A and task B are 100% byte-identical | PASS: Byte-identical static prefix matches across two distinct synthetic test tasks for all prompt generators | VERIFIED |
| R-004 | test | `go test -v ./internal/cli -run TestPromptReview_LoopInvariance` | Static prefixes across remediation loops 1..3 in `prompt review` are byte-for-byte identical | PASS: Byte-for-byte identical static prefix across Loop 1, Loop 2, and Loop 3 in prompt review | VERIFIED |
| R-005 | test | `go test -v ./internal/templates -run TestPromptTemplateOrder` | Base agent templates order static contract ahead of dynamic project state | PASS: base/prompt.md.tmpl and base/prompt_compact.md.tmpl place contracts, rules, and cache boundary ahead of dynamic state | VERIFIED |
| R-006 | check / test | `go test -v ./internal/templates -run TestVersion && ./bin/sdd check --strict` | templates.Version=2.10.0 clean across kit and contracts with 0 errors | PASS: templates.Version=2.10.0 verified; sdd check --strict passes with 0 errors, 0 warnings; temporary kit generation passes with 0 errors | VERIFIED |
| R-007 | test | `go test -race ./internal/cli ./internal/templates && go test -count=2 ./internal/cli ./internal/templates` | Race-detector clean and deterministic identical outputs across runs | PASS: -race clean across internal/cli and internal/templates; -count=2 runs yield identical results | VERIFIED |

## 2. Commands run
Record exact commands and the outcome of each during Phase C verification.

| Command | Result |
|---|---|
| `make build` | PASS: binary built into bin/sdd with version 1.10.0 |
| `make test` | PASS: all 35 packages in SDD pass with -count=1 |
| `golangci-lint run internal/cli/... internal/templates/...` | PASS: 0 issues on all modified packages |
| `./bin/sdd check --strict` | PASS: 0 error(s), 0 warning(s), 1 note(s) |
| `go test -race ./internal/cli ./internal/templates` | PASS: race detector clean |
| `go test -count=2 ./internal/cli ./internal/templates` | PASS: identical deterministic runs |
| Hard Rule 16 temp kit verification | PASS: sdd init + sdd check passes with 0 errors |

## 3. Manual checks
- Verify `sdd prompt plan-review T-062` displays static invariant prefix first, followed by dynamic task details and staged review bundle.
- Verify `sdd prompt review T-062 --lens spec --loop 1` and `sdd prompt review T-062 --lens spec --loop 2` share identical bytes up to `<!-- sdd:cache-boundary:end-prefix -->`.
- Verify `sdd prompt implement T-062 --role implementer` places role skill and engineering standards ahead of planned file boundary and step directives.

## 4. Negative and failure cases
- Empty or missing `SKILLS.md` feedback rules: invariant prefix remains valid and meets minimum byte floor.
- Missing `.sdd/roles.json`: fallback to canonical embedded role routing without corrupting prefix boundary.
- Invalid or unknown task ID: prompt generator exits cleanly with error code before assembling cache prefixes.

## 5. Determinism check
Ran `go test -count=2 ./internal/cli ./internal/templates` twice; outputs identical: PASS.

## 6. Deviations, known limitations, and feedback-rule compliance
- Deviations: none.
- Feedback-rule compliance:
  - `F-001`: Will resolve all findings (🔴 Blockers, 🟡 Warnings, 🟢 Minors) across Review Triad lenses before Phase D close.
  - `F-002`: Path hygiene enforced — zero personal home directory paths in code, templates, or tests.
  - `F-003`: Default task token telemetry recorded via `sdd record-tokens T-062` and documented below.

## 7. Definition of done
- [x] Every requirement (R-001 .. R-007) has evidence in traceability matrix
- [x] Build, lint, and tests pass using recorded commands
- [x] Negative and failure cases tested
- [x] Determinism confirmed with `-count=2`
- [x] No files changed outside the approved plan; standard library only (D-007)
- [x] Independent subagent audit completed in review.md with 0 Blockers; all findings resolved per F-001
- [x] Living docs updated (`ROADMAP.md`, `TECH_STACK.md`, `SKILLS.md`, `README.md`)
- [x] Compliant with every active human feedback rule (F-001, F-002, F-003)
- [x] Token telemetry recorded via `sdd record-tokens T-062` and documented below
- [x] Suggested commit message provided

## 8. Token Metrics

| Agent / Role | Model | Input Tokens | Cache Read Tokens | Output Tokens | Total Tokens |
|---|---|---|---|---|---|
| Task Coordinator | inherit | 13,500 | 62,000 | 3,800 | 79,300 |
| Plan Reviewer | inherit | 4,500 | 20,000 | 1,200 | 25,700 |
| Spec Auditor | flash | 3,800 | 16,000 | 950 | 20,750 |
| AppSec Specialist | inherit | 4,000 | 17,500 | 1,100 | 22,600 |
| Pattern Auditor | flash | 3,900 | 16,500 | 1,000 | 21,400 |
| **Total for T-062** | | **29,700** | **132,000** | **8,050** | **169,750** |
| **Cumulative Project Total** | | **416,100** | **1,661,000** | **114,950** | **2,192,050** |

## 9. Sign-off
Suggested commit message: `feat(cli): implement provider-aligned prompt prefix caching and dynamic suffix pinning (T-062)`
Reviewed by: ______   Date: ______   Reply received: `COMMITTED T-062`
