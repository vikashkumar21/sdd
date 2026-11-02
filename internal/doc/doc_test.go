package doc_test

import (
	"strings"
	"testing"

	"github.com/vikashkumar21/sdd/internal/context"
	"github.com/vikashkumar21/sdd/internal/doc"
)

func TestValidatePDF(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		data        []byte
		wantVersion string
	}{
		{
			name:        "pdf 1.4",
			data:        []byte("%PDF-1.4\n%âãÏÓ\n1 0 obj\n<<>>\nendobj"),
			wantVersion: "1.4",
		},
		{
			name:        "pdf 1.7 with spaces",
			data:        []byte("%PDF-1.7\r\n2 0 obj\n"),
			wantVersion: "1.7",
		},
		{
			name:        "pdf 2.0 modern",
			data:        []byte("%PDF-2.0\n%%EOF"),
			wantVersion: "2.0",
		},
		{
			name:        "pdf with leading whitespace",
			data:        []byte("   \n%PDF-1.5\n"),
			wantVersion: "1.5",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ver, err := doc.ValidatePDF(tc.data)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tc.name, err)
			}
			if ver != tc.wantVersion {
				t.Errorf("version mismatch: got %q, want %q", ver, tc.wantVersion)
			}
		})
	}
}

func TestValidatePDFMalformed(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		data  []byte
		check func(error) bool
	}{
		{
			name: "empty",
			data: []byte{},
			check: func(err error) bool {
				return err != nil && strings.Contains(err.Error(), "empty")
			},
		},
		{
			name: "plain text without pdf signature",
			data: []byte("This is plain text document"),
			check: func(err error) bool {
				return err != nil && strings.Contains(err.Error(), "missing %PDF-")
			},
		},
		{
			name: "oversized pdf",
			data: append([]byte("%PDF-1.4\n"), make([]byte, context.MaxInputBytes+10)...),
			check: func(err error) bool {
				return err != nil && strings.Contains(err.Error(), "exceeds maximum allowed")
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := doc.ValidatePDF(tc.data)
			if err == nil {
				t.Fatalf("expected error for case %s, got nil", tc.name)
			}
			if tc.check != nil && !tc.check(err) {
				t.Errorf("unexpected error message for %s: %v", tc.name, err)
			}
		})
	}
}

func TestIsPDF(t *testing.T) {
	t.Parallel()

	if !doc.IsPDF("spec.pdf", []byte("anything")) {
		t.Error("expected IsPDF true for spec.pdf")
	}
	if !doc.IsPDF("spec.PDF", []byte("anything")) {
		t.Error("expected IsPDF true for uppercase spec.PDF")
	}
	if !doc.IsPDF("spec", []byte("%PDF-1.4\n")) {
		t.Error("expected IsPDF true for %PDF- bytes")
	}
	if doc.IsPDF("notes.txt", []byte("plain text")) {
		t.Error("expected IsPDF false for plain text")
	}
}

func TestInjectDocReference(t *testing.T) {
	t.Parallel()

	template := `# Requirements — T-002 arch-impl

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Draft
Roadmap ref: ROADMAP.md T-002

## 1. Goal
Implement architecture design.

## 2. Context and references
- Mission / principles: Principle 1

## 3. Scope
`

	stagedPath := ".sdd/context/T-002/20261102-architecture.pdf"
	origin := "architecture.pdf"
	format := "pdf"

	res := doc.InjectDocReference(template, stagedPath, origin, format)
	expectedRef := "- Reference document: `.sdd/context/T-002/20261102-architecture.pdf` (Format: pdf, Source: architecture.pdf)"

	if !strings.Contains(res, expectedRef) {
		t.Errorf("expected reference line %q in result:\n%s", expectedRef, res)
	}
	if !strings.Contains(res, "- Mission / principles: Principle 1") {
		t.Error("existing Context contents preserved")
	}
}
