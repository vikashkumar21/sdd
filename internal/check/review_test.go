package check

import (
	"strings"
	"testing"
)

func TestCheckReviewDocument_FrontmatterHandling(t *testing.T) {
	contentWithFrontmatter := `---
status: draft
task_id: T-055
---
# Code Review — T-055 Test

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Pending
Reviewer: Independent Review Triad
Commit / Diff Ref: HEAD

## 1. Multi-Lens Ensemble Review
`

	c := &checker{
		texts: map[string]string{
			"tasks/features/T-055-test/review.md": contentWithFrontmatter,
		},
	}

	c.checkReviewDocument("tasks/features/T-055-test", "specifying")

	if len(c.findings) != 0 {
		t.Fatalf("expected 0 findings when frontmatter is present, got %d: %+v", len(c.findings), c.findings)
	}
}

func TestCheckReviewDocument_InvalidStatus(t *testing.T) {
	content := `# Code Review — T-055 Test

Status: InvalidStatus
Reviewer: Independent Review Triad

## 1. Multi-Lens Ensemble Review
`

	c := &checker{
		texts: map[string]string{
			"tasks/features/T-055-test/review.md": content,
		},
	}

	c.checkReviewDocument("tasks/features/T-055-test", "specifying")

	if len(c.findings) == 0 {
		t.Fatalf("expected finding for invalid review status, got none")
	}
	found := false
	for _, f := range c.findings {
		if f.Rule == "review" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected finding with rule 'review', got: %+v", c.findings)
	}
}

func TestCheckTraceabilityLedger_ValidSchema(t *testing.T) {
	reqContent := `# Requirements — T-057
## 4. Functional requirements
| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | System SHALL extract ledger | MUST | test |
| R-002 | System SHALL validate ledger | MUST | test |
`
	revContent := `# Code Review — T-057 Test
Status: Approved
Reviewer: Independent Review Triad

## 1. Requirements Traceability Ledger
| Req ID | Requirement Statement | Code Symbol & File | Positive Test Evidence | Negative / Boundary Test Evidence | Status |
|---|---|---|---|---|---|
| R-001 | System SHALL extract ledger | ExtractRequirements in bundle.go | TestExtractRequirements | TestExtractRequirements_Empty | Verified |
| R-002 | System SHALL validate ledger | checkReviewTraceabilityLedger in review.go | TestCheckTraceabilityLedger | TestCheckTraceabilityLedger_Omission | Verified |

## 2. Multi-Lens Ensemble Review
### 🔎 Spec Auditor Lens
### 🛡️ AppSec Specialist Lens
### 📐 Pattern & Quality Auditor Lens
`

	c := &checker{
		texts: map[string]string{
			"tasks/features/T-057-test/requirements.md": reqContent,
			"tasks/features/T-057-test/review.md":       revContent,
		},
	}

	c.checkReviewDocument("tasks/features/T-057-test", "awaiting review")

	var ledgerFindings []Finding
	for _, f := range c.findings {
		if f.Rule == "traceability-ledger" {
			ledgerFindings = append(ledgerFindings, f)
		}
	}

	if len(ledgerFindings) != 0 {
		t.Fatalf("expected 0 traceability-ledger findings for valid schema, got %d: %+v", len(ledgerFindings), ledgerFindings)
	}
}

func TestCheckTraceabilityLedger_Omission(t *testing.T) {
	reqContent := `# Requirements — T-057
## 4. Functional requirements
| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | System SHALL extract ledger | MUST | test |
| R-002 | System SHALL validate ledger | MUST | test |
`
	// Missing R-002 row
	revContent := `# Code Review — T-057 Test
Status: Approved
Reviewer: Independent Review Triad

## 1. Requirements Traceability Ledger
| Req ID | Requirement Statement | Code Symbol & File | Positive Test Evidence | Negative / Boundary Test Evidence | Status |
|---|---|---|---|---|---|
| R-001 | System SHALL extract ledger | ExtractRequirements in bundle.go | TestExtractRequirements | TestExtractRequirements_Empty | Verified |

## 2. Multi-Lens Ensemble Review
`

	c := &checker{
		texts: map[string]string{
			"tasks/features/T-057-test/requirements.md": reqContent,
			"tasks/features/T-057-test/review.md":       revContent,
		},
	}

	c.checkReviewDocument("tasks/features/T-057-test", "awaiting review")

	foundOmission := false
	for _, f := range c.findings {
		if f.Rule == "traceability-ledger" && f.Severity == Error && strings.Contains(f.Message, "R-002") {
			foundOmission = true
			break
		}
	}

	if !foundOmission {
		t.Fatalf("expected Error finding for omitted R-002, got: %+v", c.findings)
	}
}

func TestCheckTraceabilityLedger_PlaceholderEvidence(t *testing.T) {
	reqContent := `# Requirements — T-057
## 4. Functional requirements
| ID | Requirement | Priority | Verification |
|---|---|---|---|
| R-001 | System SHALL extract ledger | MUST | test |
`
	// Code Symbol has TODO placeholder
	revContent := `# Code Review — T-057 Test
Status: Approved
Reviewer: Independent Review Triad

## 1. Requirements Traceability Ledger
| Req ID | Requirement Statement | Code Symbol & File | Positive Test Evidence | Negative / Boundary Test Evidence | Status |
|---|---|---|---|---|---|
| R-001 | System SHALL extract ledger | <TODO: add symbol> | TestExtractRequirements | TestExtractRequirements_Empty | Verified |
`

	c := &checker{
		texts: map[string]string{
			"tasks/features/T-057-test/requirements.md": reqContent,
			"tasks/features/T-057-test/review.md":       revContent,
		},
	}

	c.checkReviewDocument("tasks/features/T-057-test", "awaiting review")

	foundPlaceholder := false
	for _, f := range c.findings {
		if f.Rule == "traceability-ledger" && f.Severity == Error && strings.Contains(f.Message, "placeholder") {
			foundPlaceholder = true
			break
		}
	}

	if !foundPlaceholder {
		t.Fatalf("expected Error finding for placeholder Code Symbol, got: %+v", c.findings)
	}
}

func TestCheckTraceabilityLedger_NegativeEvidenceMissing(t *testing.T) {
	reqContent := `# Requirements — T-057
## 4. Functional requirements
| ID | Requirement | Priority | Verification |
|---|---|---|---|
| R-001 | System SHALL extract ledger | MUST | test |
`
	// Negative test evidence is n/a placeholder
	revContent := `# Code Review — T-057 Test
Status: Approved
Reviewer: Independent Review Triad

## 1. Requirements Traceability Ledger
| Req ID | Requirement Statement | Code Symbol & File | Positive Test Evidence | Negative / Boundary Test Evidence | Status |
|---|---|---|---|---|---|
| R-001 | System SHALL extract ledger | ExtractRequirements in bundle.go | TestExtractRequirements | n/a | Verified |
`

	c := &checker{
		texts: map[string]string{
			"tasks/features/T-057-test/requirements.md": reqContent,
			"tasks/features/T-057-test/review.md":       revContent,
		},
	}

	c.checkReviewDocument("tasks/features/T-057-test", "awaiting review")

	foundNeg := false
	for _, f := range c.findings {
		if f.Rule == "traceability-ledger" && f.Severity == Error && strings.Contains(f.Message, "Negative / Boundary Test Evidence") {
			foundNeg = true
			break
		}
	}

	if !foundNeg {
		t.Fatalf("expected Error finding for missing negative evidence, got: %+v", c.findings)
	}
}

func TestCheckTraceabilityLedger_OmissionStatus(t *testing.T) {
	reqContent := `# Requirements — T-057
## 4. Functional requirements
| ID | Requirement | Priority | Verification |
|---|---|---|---|
| R-001 | System SHALL extract ledger | MUST | test |
`
	// Row status marked Omission
	revContent := `# Code Review — T-057 Test
Status: Approved
Reviewer: Independent Review Triad

## 1. Requirements Traceability Ledger
| Req ID | Requirement Statement | Code Symbol & File | Positive Test Evidence | Negative / Boundary Test Evidence | Status |
|---|---|---|---|---|---|
| R-001 | System SHALL extract ledger | ExtractRequirements in bundle.go | TestExtractRequirements | TestExtractRequirements_Empty | Omission |
`

	c := &checker{
		texts: map[string]string{
			"tasks/features/T-057-test/requirements.md": reqContent,
			"tasks/features/T-057-test/review.md":       revContent,
		},
	}

	c.checkReviewDocument("tasks/features/T-057-test", "awaiting review")

	foundOmissionStatus := false
	for _, f := range c.findings {
		if f.Rule == "traceability-ledger" && f.Severity == Error && strings.Contains(f.Message, "marked \"Omission\"") {
			foundOmissionStatus = true
			break
		}
	}

	if !foundOmissionStatus {
		t.Fatalf("expected Error finding for row marked Omission, got: %+v", c.findings)
	}
}

func TestCheckTraceabilityLedger_MissingLedgerInAwaitingReview(t *testing.T) {
	// A task in awaiting review with Approved review that has no ledger section
	revContent := `# Code Review — T-057 Test
Status: Approved
Reviewer: Independent Review Triad

## 1. Multi-Lens Ensemble Review
`

	c := &checker{
		texts: map[string]string{
			"tasks/features/T-057-test/review.md": revContent,
		},
	}

	c.checkReviewDocument("tasks/features/T-057-test", "awaiting review")

	foundMissingLedger := false
	for _, f := range c.findings {
		if f.Rule == "traceability-ledger" && f.Severity == Error && strings.Contains(f.Message, "missing ## Requirements Traceability Ledger") {
			foundMissingLedger = true
			break
		}
	}

	if !foundMissingLedger {
		t.Fatalf("expected Error finding for missing ledger in awaiting review, got: %+v", c.findings)
	}
}
