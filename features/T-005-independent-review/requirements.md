# Requirements — T-005 Independent review, adaptive-design proposals, and improvement plan (docs only until approved)

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-10-29)
Roadmap ref: `ROADMAP.md` T-005

## 1. Goal
Deliver an adversarial, senior Go architectural and code review of `sdd`, verify toolchain and runtime compatibility on macOS (Go 1.27.3, darwin/arm64, golangci-lint v2), provide a deterministic adaptive-design blueprint (self-checking, self-healing, self-analysing, self-improving), and establish a prioritised catalogue of proposals (P-01..Pn) for subsequent gated implementation.

## 2. Context and references
- Mission / principles touched: MISSION §5 (Deterministic, Fail safe, Human in the loop, Small and shippable, Never overwrite human work), MISSION §7 (S1–S5), MISSION §8 (Hypotheses H1, H2, H3).
- Decisions this relies on: D-001 through D-014 (Approved), D-015 (Proposed, addressed in review), D-016 (Proposed), D-017 (Proposed).
- Depends on tasks: T-001 (Awaiting review / committed in `d91c9c2`).

## 3. Scope
**In scope**
- Section 7 Review brief:
  1. Genesis, problem statement, D-001..D-017 sequence, and verified facts vs reported claims.
  2. Comparative analysis against manual kits, GitHub Spec Kit (`github/spec-kit`), and OpenSpec (`openspec.dev`), resolving H2.
  3. Adversarial code review covering known weaknesses and deep edge cases (`cmd/sdd`, `internal/fsutil`, test coverage, regex limitations, template versioning, name-based detection, manifest compatibility, symlinks, permissions).
  4. Project-type coverage matrix (Go, Java, Python, TypeScript, polyglot monorepos, hand-made kits like dbguard, multi-agent teams).
  5. Language pack quality review against modern idioms.
  6. Prioritised improvement proposals catalogue (`P-01`..`Pn`) preserving invariants (S1, S3, dbguard compatibility).
- Section 8 Adaptive design brief:
  - Deterministic architectures for self-check, self-heal, self-analyse, self-improve, and adapting to AI change.
  - Explicit boundaries of what must NOT be automated.
- Section 9 Modern Go brief:
  - Toolchain verification (`go1.27.3 darwin/arm64`, `golangci-lint v2.15.0`), resolution proposal for D-015, stdlib module layout, modern stdlib candidates (`os.Root`, `slices`, `maps`, `log/slog`, `errors.Join`, `range-over-int`).
- Section 10 Verification brief:
  - Real execution evidence on this machine in temporary directories.
- Zero production code edits: documentation only within `features/T-005-independent-review/`.

**Out of scope**
- Writing production Go code outside `features/T-005-independent-review/`.
- Adding third-party dependencies without an approved decision.
- Executing git write operations (`git add`, `commit`, `push`, etc.).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The review SHALL document the genesis, decision log (D-001..D-017), and separate verified facts from reported claims with dates and sources. | MUST | inspection: `review.md` §1 |
| R-002 | The review SHALL independently evaluate GitHub Spec Kit and OpenSpec via web research, compare them against `sdd`, and make an evidence-based recommendation on H2. | MUST | inspection: `review.md` §2 |
| R-003 | The review SHALL audit all Go packages adversarially (`cmd/sdd`, `internal/*`), document real failure modes, and quantify test coverage and linter findings on this machine. | MUST | inspection & test: `golangci-lint`, test runs, `review.md` §3 |
| R-004 | The review SHALL systematically evaluate coverage across greenfield and brownfield projects in Go, Java, Python, TypeScript, monorepos, hand-made kits, and multi-agent setups. | MUST | inspection: `review.md` §4 |
| R-005 | The review SHALL audit the bundled language packs against community idioms and standards. | MUST | inspection: `review.md` §5 |
| R-006 | The review SHALL catalogue prioritised proposals (`P-01`..`Pn`) with size (S/M/L), dependencies, risks, test plans, and decision requirements, preserving S1, S3, and dbguard compatibility. | MUST | inspection: `review.md` §6 |
| R-007 | The review SHALL specify deterministic adaptive designs for self-check, self-heal, self-analyse, self-improve, and AI change adaptation, enumerating non-automatable boundaries. | MUST | inspection: `review.md` §7 |
| R-008 | The review SHALL evaluate modern Go alignment (Go 1.27.3, stdlib layout, modern stdlib features including `os.Root`, `slices`, `maps`, `errors.Join`) and propose D-015 resolution. | MUST | inspection: `review.md` §8 |
| R-009 | The review SHALL record empirical verification on this machine in temporary scratch directories with exact commands and outputs. | MUST | inspection & execution: `validation.md`, `review.md` §9 |

## 5. Non-functional requirements
- **Determinism:** Proposals and review conclusions must rely strictly on repeatable evidence and reproducible tests.
- **Fail safe:** Any ambiguity in requirements or tool behaviour must default to the most restrictive outcome or explicit human gates.
- **Safety:** Scratch experiments must run only within isolated temporary directories (`/tmp` or `os.TempDir()`), never touching project roots or real repos.
- **Documentation integrity:** No modifications to existing repository files outside `features/T-005-independent-review/`.

## 6. Constraints and assumptions
- Reviewer operates strictly as an independent senior Go engineer.
- Toolchain on this machine is verified: `go version go1.27.3 darwin/arm64`, `golangci-lint 2.15.0`.
- All third-party library additions are prohibited without a human-approved `DECISION NEEDED`.

## 7. Open questions
(None. All open items are framed as proposed decisions or proposals in `plan.md` and `review.md`.)

## 8. Acceptance criteria
- Given the codebase, when audited against `golangci-lint v2.15.0`, all existing lint failures and compiler/vet issues are documented with file and line evidence.
- Given the current `go.mod` (1.22) and machine toolchain (1.27.3), options and trade-offs for D-015 are documented with an explicit recommendation.
- Given the adaptive architecture requirements, designs for self-check, self-heal, self-analyse, and self-improve are detailed with strict human gates.
- Given the entire review, proposals `P-01` through `Pn` are catalogued with sizes, dependencies, and risks for the human to select from.
