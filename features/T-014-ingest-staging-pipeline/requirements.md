# Requirements — T-014 Context staging pipeline and sdd ingest command

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-014

## 1. Goal
Provide an offline, deterministic context-staging pipeline and `sdd ingest` command (plus `--context` support on `sdd init`) that safely stages external transcripts, documents, raw text, and piped stdin into `.sdd/context/` so AI agents can synthesize requirements under Gate 1 approvals without network dependencies or secret leakage.

## 2. Context and references
- Mission / principles touched: MISSION §5 (Deterministic, Fail safe, Human in the loop), MISSION §7 (S1).
- Decisions this relies on:
  - D-007 (Standard library only)
  - D-015 (Approved: Go 1.27 toolchain)
  - D-018 (Approved: Ingestion & semantic synthesis architecture via agent-assisted staging in `.sdd/context/`)
  - D-020 (Approved: Raw text & files staged for agent reading)
  - D-021 (Approved: Path safety and cross-platform determinism)
- Depends on tasks: T-002 (Done), T-007 (Done).

## 3. Scope
**In scope**
- `internal/context`:
  - Package to manage staged artifacts in `.sdd/context/`.
  - Content addressing or deterministic slug naming with sequence prefixes: `.sdd/context/<timestamp>-<slug>.<ext>` or `.sdd/context/<id>.<ext>`.
  - Manifest file `.sdd/context/manifest.json` tracking staged inputs: origin (`raw-text`, `file`, `stdin`), source name/path, byte size, SHA-256 hash, staged date, format/type hint (`text`, `markdown`, `json`, `transcript`, `pdf`).
  - Read & stage operations confined strictly within the project root via `*os.Root`.
- `sdd ingest` CLI command:
  - Supports `--file <path>` (can be specified multiple times or as comma/space-separated list).
  - Supports `--text "<raw text>"` or `--text-file <path>`.
  - Supports reading from piped `stdin` when `--file` and `--text` are absent and stdin is not a terminal.
  - Supports optional `--name <label>` to customize the staged filename slug.
  - Supports optional `--task <T-xxx>` to associate context with a specific task (e.g. staging into `.sdd/context/T-001/` or metadata association).
- `sdd init` integration:
  - Flag `--context-file <path>` and `--context-text "<raw text>"` to ingest kickoff context during project initialization.
- Deterministic checks and bounds:
  - Size bounding (reject single inputs exceeding 10 MB fail-safe limit).
  - Validates relative paths, checks against Windows reserved device names, and confines file writes with `fsutil.WriteAtomicRoot`.

**Out of scope**
- Parsing proprietary binary PDF structures into text (deferred to T-017 / multimodal agents per D-020).
- Direct Jira REST API network calls (deferred to T-015 / follow-up plugins per D-019).
- Modifying project markdown files automatically without human Gate 1 approval (per D-018 and agent contract §2).

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | `internal/context` SHALL provide a `Stage` function that persists input bytes into `.sdd/context/` confined via `*os.Root`. | MUST | unit test: `context_test.go` |
| R-002 | Every staged artifact SHALL be recorded in `.sdd/context/manifest.json` with source, hash, byte size, timestamp, and format hint. | MUST | test: `TestContextManifest` |
| R-003 | `sdd ingest` SHALL accept raw text strings via `--text` and stage them as markdown/text files. | MUST | CLI test: `TestIngestText` |
| R-004 | `sdd ingest` SHALL accept files via `--file` (single or multiple) and copy them into `.sdd/context/`. | MUST | CLI test: `TestIngestFiles` |
| R-005 | `sdd ingest` SHALL accept piped standard input when non-interactive and no `--file` or `--text` flag is provided. | MUST | CLI test: `TestIngestStdin` |
| R-006 | `sdd ingest` SHALL accept `--task <T-xxx>` and associate the staged context with that task in the context manifest. | MUST | CLI test: `TestIngestTaskAssociation` |
| R-007 | `sdd init` SHALL accept `--context-file` and `--context-text` flags, staging them into `.sdd/context/` upon project initialization. | MUST | CLI test: `TestInitWithContext` |
| R-008 | The staging pipeline SHALL enforce a 10 MB maximum input limit and fail safely when exceeded. | MUST | test: `TestInputSizeLimit` |
| R-009 | All staged filenames and relative paths SHALL be checked with `fsutil.ValidateLocalRelPath` and avoid Windows reserved names. | MUST | test: `TestStagingPathValidation` |
| R-010 | `sdd check` SHALL verify that `.sdd/context/manifest.json` (if present) is valid JSON and all referenced staged files exist. | MUST | test: `TestCheckContextManifest` |
| R-011 | `make check`, `go test -race ./...`, and `GOOS=windows go vet ./...` SHALL pass with zero issues. | MUST | command check |

## 5. Non-functional requirements
- **Determinism:** Staging generates reproducible filenames and hashes. Time injection is used for test isolation.
- **Fail-safe:** Input size bounds prevent denial of service; invalid paths or corrupted files abort immediately without partial state.
- **Security & Privacy:** Offline only. Zero network requests or external credential storage. All filesystem writes are confined to the project root.

## 6. Constraints and assumptions
- Uses standard library only (D-007, Go 1.27).
- Confinement uses `os.Root` established in T-007.

## 7. Open questions
None.

## 8. Acceptance criteria
- Given a raw text string, running `sdd ingest --text "meeting notes"` creates `.sdd/context/...` and updates `.sdd/context/manifest.json`.
- Given a file, running `sdd ingest --file notes.txt` copies the content into `.sdd/context/` with SHA-256 verification.
- Given piped stdin, running `cat notes.txt | sdd ingest --name "notes"` stages the input successfully.
- Given `sdd init --name demo --context-text "Project specs..."`, the project is initialized and the context is staged.
- Running `sdd check` verifies the integrity of `.sdd/context/manifest.json`.
