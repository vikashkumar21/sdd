# Validation — T-017 Document and PDF ingestion protocol

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Passed
Roadmap ref: `ROADMAP.md` T-017

## 1. Traceability matrix

| Req | Method | Command / test | Expected | Actual | Result |
|---|---|---|---|---|---|
| R-001 | test | `go test -v ./internal/doc -run TestValidatePDF` | Validates %PDF- header and size limits | PASS: %PDF-1.4, 1.7, 2.0, size limit validated | PASS |
| R-002 | test | `go test -v ./internal/cli -run TestCLIIngestPDF` | Stages valid PDF to .sdd/context/ with format pdf | PASS: Validated and staged to .sdd/context/ with format pdf | PASS |
| R-003 | test | `go test -v ./internal/feature -run TestNewFromDoc` | Stages doc under .sdd/context/<task>/ and injects ref into requirements.md | PASS: Staged to .sdd/context/T-001/ and reference injected into requirements.md | PASS |
| R-004 | test | `go test -v ./internal/cli -run TestCLIIngestPDF` | --pdf flag validates and stages PDF | PASS: Explicit --pdf flag validated and staged | PASS |
| R-005 | test | `go test -v ./internal/cli -run TestCLIFeatureNewFromDoc` | sdd prompt includes multimodal inspection guidance for PDF | PASS: Handover prompt includes 'pdf (multimodal inspection)' | PASS |
| R-006 | test | `go test -v ./internal/doc -run TestValidatePDFMalformed` | Rejects non-existent, invalid signature, or oversized files | PASS: Missing signature, empty data, oversized PDF safely rejected | PASS |

## 2. Commands run
Record exact commands and the outcome of each.

| Command | Result |
|---|---|
| `make verify` | PASS (gofmt clean, go vet clean, all tests -count=1 pass) |
| `make check` | PASS (golangci-lint 0 issues, full test suite pass) |
| `go run ./cmd/sdd check --strict` | PASS (0 errors, 0 warnings) |

## 3. Manual checks
- Verified that `sdd feature new --from-doc <file>` stages the document under `.sdd/context/<task>/` and links it in `requirements.md`.
- Verified that `sdd prompt` labels PDF artifacts as `pdf (multimodal inspection)`.

## 4. Negative and failure cases
- `TestCLIIngestPDFInvalid`: Ingesting a non-PDF with `--pdf` exits 1 with `invalid pdf document`.
- `TestNewFromDocInvalidPDF`: Passing corrupted non-PDF file to `--from-doc` fails safe with exit code 1.
- `TestValidatePDFMalformed`: Oversized PDFs exceeding `MaxInputBytes` are rejected with descriptive error.

## 5. Determinism check
Ran `go test ./internal/doc ./internal/feature ./internal/cli -count=1` twice; outputs identical: yes.
- Pass 1:
  `ok  github.com/vikashkumar21/sdd/internal/doc     0.343s`
  `ok  github.com/vikashkumar21/sdd/internal/feature 1.534s`
  `ok  github.com/vikashkumar21/sdd/internal/cli     2.664s`
- Pass 2:
  `ok  github.com/vikashkumar21/sdd/internal/doc     0.065s`
  `ok  github.com/vikashkumar21/sdd/internal/feature 1.156s`
  `ok  github.com/vikashkumar21/sdd/internal/cli     2.587s`

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
Suggested commit message: `feat(doc): add document and PDF ingestion protocol for multimodal context staging (T-017)`
Reviewed by: Vikash   Date: 2026-11-02   Reply received: Pending GATE 2
