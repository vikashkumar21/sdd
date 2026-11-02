# Plan — T-017 Document and PDF ingestion protocol

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved
Roadmap ref: `ROADMAP.md` T-017

## 1. Approach summary
1. Implement `internal/doc/doc.go`:
   - `ValidatePDF(data []byte) (version string, err error)` verifying `%PDF-` signature and bounding size to `context.MaxInputBytes`.
   - `IsPDF(filename string, data []byte) bool`.
   - `InjectDocReference(requirementsMarkdown, stagedPath, origin, format string) string`.
2. Implement unit tests in `internal/doc/doc_test.go`:
   - Valid PDF headers (`%PDF-1.4`, `%PDF-1.7`, `%PDF-2.0`).
   - Corrupted/non-PDF inputs, empty data, truncated signatures, size over limit.
   - Reference injection into requirements templates.
3. Extend `internal/feature/feature.go` and `internal/feature/feature_test.go`:
   - Add `FromDocData` and `FromDocPath` to `feature.Options`.
   - Stage document under `.sdd/context/<task>/` and inject reference into `requirements.md`.
   - Add tests `TestNewFromDocPDF` and `TestNewFromDocInvalidPDF`.
4. Extend `internal/cli/commands.go`:
   - Add `--from-doc <file>` flag to `sdd feature new`.
   - In `runPrompt`, enrich format display for PDF context artifacts with multimodal guidance.
5. Extend `internal/cli/ingest.go` and `internal/cli/ingest_test.go`:
   - Add `--pdf` flag to `sdd ingest`.
   - Auto-detect `.pdf` files and validate signature before staging.
   - Add CLI test `TestCLIIngestPDF` and `TestCLIIngestPDFInvalid`.
6. Update living docs (`README.md`, `ROADMAP.md`).

## 2. Decisions (new or affected)
None. Settled in approved Decision D-020 (Option A: stdlib offline multimodal staging) and D-023 (Version bump to v1.0.0 after T-017).

## 3. Design
- **Signature & Boundary Validation (`internal/doc/doc.go`)**:
  - Checks for `%PDF-` within the first 1024 bytes.
  - Enforces `context.MaxInputBytes` (10 MB).
  - Extracts version substring (`1.4`, `1.7`, `2.0`).
- **Feature Document Staging**:
  - `sdd feature new <T-xxx> <slug> --from-doc <file>`:
    - Validates PDF if extension is `.pdf` or header is `%PDF-`.
    - Stages into `.sdd/context/<task>/<timestamp>-ref-doc.pdf`.
    - Updates `features/<task>-<slug>/requirements.md` under `## 2. Context and references`:
      `- Reference document: .sdd/context/<task>/... (Format: pdf, Source: spec.pdf)`
- **CLI Ingestion**:
  - `sdd ingest --pdf --file spec.pdf` validates `%PDF-` signature and marks format as `"pdf"`.

## 4. Implementation steps

| # | Step | Files | Proving test / check |
|---|---|---|---|
| 1 | Implement PDF validator and doc reference injector | `internal/doc/doc.go`, `internal/doc/doc_test.go` | `go test ./internal/doc` |
| 2 | Extend feature package with document staging & reference injection | `internal/feature/feature.go`, `internal/feature/feature_test.go` | `go test ./internal/feature -run TestNewFromDoc` |
| 3 | Extend CLI commands for --from-doc and prompt enrichment | `internal/cli/commands.go` | `go test ./internal/cli -run TestPromptWithStagedContext` |
| 4 | Extend CLI ingest with --pdf flag | `internal/cli/ingest.go`, `internal/cli/ingest_test.go` | `go test ./internal/cli -run TestCLIIngestPDF` |
| 5 | Update living docs (README.md, ROADMAP.md) | `README.md`, `ROADMAP.md` | `go run ./cmd/sdd check --strict` |

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| `internal/doc/doc.go` | create | PDF signature validation and document reference injector |
| `internal/doc/doc_test.go` | create | Unit tests for PDF validation and doc reference injection |
| `internal/feature/feature.go` | modify | Support document staging and reference injection during feature creation |
| `internal/feature/feature_test.go` | modify | Tests for feature creation with --from-doc |
| `internal/cli/commands.go` | modify | Add --from-doc to sdd feature new, multimodal prompt guidance |
| `internal/cli/ingest.go` | modify | Add --pdf flag to sdd ingest |
| `internal/cli/ingest_test.go` | modify | Tests for sdd ingest --pdf and sdd feature new --from-doc |
| `README.md` | modify | Document --from-doc and --pdf flags |
| `ROADMAP.md` | modify | Track T-017 status |

## 6. Dependencies
None. Standard library only (`D-007`).

## 7. Living docs to update

| File | Change |
|---|---|
| `README.md` | Document `sdd feature new --from-doc` and `sdd ingest --pdf` |
| `ROADMAP.md` | Update T-017 status across gates |

## 8. Risks, mitigations, rollback
- **Risk:** Malformed, password-protected, or oversized PDFs could cause memory or parser failures.
  - *Mitigation:* Strict bounded size check (`MaxInputBytes`) and lightweight signature validation without executing arbitrary postscript code.
- **Rollback:** `git restore .` reverts all changes.

## 9. Applicable human-feedback rules (SKILLS.md, "Human feedback rules")
None active.

## 10. Approval record
Approved by: Vikash   Date: 2026-11-02   Phrase received: `APPROVED T-017`
