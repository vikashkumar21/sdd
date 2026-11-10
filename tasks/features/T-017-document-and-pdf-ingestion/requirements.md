# Requirements — T-017 Document and PDF ingestion protocol

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Approved (2026-11-02)
Roadmap ref: `ROADMAP.md` T-017

## 1. Goal
Ingest, validate, and stage PDF and document specifications into `.sdd/context/` for multimodal AI coding agents, and integrate document references into `sdd feature new` and `sdd prompt`.

## 2. Context and references
- Mission / principles: Principle 1 (Deterministic), Principle 4 (Small and shippable), Principle 5 (Never overwrite human work).
- Decisions this relies on: D-007 (Stdlib only), D-018 (Context staging architecture), D-020 (Approved - Document/PDF ingestion protocol Option A), D-023 (Approved - Version bump to v1.0.0 after T-017).
- Depends on tasks: T-014 (Context staging pipeline and sdd ingest).

## 3. Scope
**In scope**
- PDF header and signature validation (`%PDF-1.x` through `%PDF-2.x`) and size boundary enforcement in `internal/doc`.
- CLI commands & flags:
  - `sdd ingest --file <doc.pdf>` and `sdd ingest --pdf <file>`: validate PDF structure and stage into `.sdd/context/` with format `pdf` and SHA256 checksum.
  - `sdd feature new <T-xxx> <slug> --from-doc <file>`: validate and stage reference document into `.sdd/context/<task>/` and insert reference entry into `requirements.md` under `## 2. Context and references`.
- Handover prompt (`sdd prompt`):
  - Include multimodal inspection directive when staged context includes PDF or document artifacts.

**Out of scope**
- Heavy third-party Go C-bindings or poppler/mupdf binaries (per D-007 and D-020).
- Cloud OCR or network-based text extraction (per D-018 and D-020).
- Editing or modifying PDF files.

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL validate PDF files by verifying the `%PDF-` signature and bounding file size to `MaxInputBytes`. | MUST | test |
| R-002 | The system SHALL stage valid PDF documents into `.sdd/context/` with format `pdf` and record entries in `.sdd/context/manifest.json`. | MUST | test |
| R-003 | The system SHALL support `--from-doc <file>` in `sdd feature new`, staging the document under `.sdd/context/<task>/` and adding a reference entry to `requirements.md` under `## 2. Context and references`. | MUST | test |
| R-004 | The system SHALL support `--pdf` in `sdd ingest` to explicitly validate and stage PDF files. | MUST | test |
| R-005 | The system SHALL include multimodal document inspection guidance in `sdd prompt` when staged context includes PDF artifacts. | MUST | test |
| R-006 | The system SHALL fail safe with an explanatory error when a file specified with `--pdf` or `--from-doc` is non-existent, exceeds size limits, or lacks a valid PDF signature. | MUST | test |

## 5. Non-functional requirements
- **Determinism:** Staging the same PDF document SHALL produce identical file contents, SHA256 hashes, and deterministic manifest records.
- **Fail-safe behaviour:** Corrupted, truncated, or invalid PDF files SHALL report clear errors without panicking.
- **Security and privacy:** Offline operation only; zero external network requests or credential storage (`D-018`, `D-020`).
- **Dependencies:** Standard library only (`D-007`).

## 6. Constraints and assumptions
- AI agents reading the repository (Claude, Gemini, Antigravity) are multimodal and can inspect `.pdf` files directly when referenced in `.sdd/context/`.
- PDF files do not exceed `context.MaxInputBytes` (default 10 MB).

## 7. Open questions
(None. Resolved via Decision D-020.)

## 8. Acceptance criteria
- Given a valid PDF file, when `sdd ingest --file spec.pdf` or `sdd ingest --pdf spec.pdf` runs, then the file is staged under `.sdd/context/` with format `pdf` and sha256 checksum in `manifest.json`.
- Given a corrupted non-PDF file passed to `sdd ingest --pdf invalid.pdf`, then the command exits 1 with a descriptive error.
- Given a PDF file passed to `sdd feature new T-002 arch-impl --from-doc spec.pdf`, then the file is staged under `.sdd/context/T-002/` and `features/T-002-arch-impl/requirements.md` includes `- Reference document: ...` in `## 2. Context and references`.
- Given staged PDF context, when `sdd prompt gemini` is run, then the prompt notes that staged PDF documents should be inspected multimodally.
