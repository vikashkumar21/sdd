package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/context"
)

func TestPromptCompact(t *testing.T) {
	root := t.TempDir()
	rInit := exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "compactflow", "--lang", "go")
	if rInit.code != 0 {
		t.Fatalf("init failed: %+v", rInit)
	}

	// 1. Standard prompt baseline
	pStandard := exec(t, "", false, "", "prompt", "gemini", "--dir", root)
	if pStandard.code != 0 {
		t.Fatalf("standard prompt failed: %+v", pStandard)
	}
	standardWords := len(strings.Fields(pStandard.out))

	// 2. Compact prompt execution
	pCompact := exec(t, "", false, "", "prompt", "gemini", "--compact", "--dir", root)
	if pCompact.code != 0 {
		t.Fatalf("compact prompt failed: %+v", pCompact)
	}
	compactWords := len(strings.Fields(pCompact.out))

	// Assert essential compact invariants
	for _, want := range []string{
		"acting as Task Coordinator",
		"ANTHEM: ALWAYS BE DETERMINISTIC",
		"Operating Contract: GEMINI.md",
		".sdd/roles.json",
		"GATE 1 (Plan Approval)",
		"`APPROVED T-xxx`",
		"GATE 2 (Close & Commit)",
		"`COMMITTED T-xxx`",
		"Human Inputs",
		"Gate reply: NONE YET",
	} {
		if !strings.Contains(pCompact.out, want) {
			t.Errorf("compact prompt missing expected invariant %q:\n%s", want, pCompact.out)
		}
	}

	// Assert brevity constraints (AC-2: < 500 tokens; ~250 words is comfortably < 400 tokens)
	if compactWords > 300 {
		t.Errorf("compact prompt word count %d exceeds maximum threshold of 300 words:\n%s", compactWords, pCompact.out)
	}

	// Assert at least 60% reduction compared to standard prompt
	reduction := float64(standardWords-compactWords) / float64(standardWords)
	if reduction < 0.60 {
		t.Errorf("expected at least 60%% token reduction, got %.2f%% (standard: %d words, compact: %d words)",
			reduction*100, standardWords, compactWords)
	}
}

func TestPromptCompactWithTaskAndContext(t *testing.T) {
	root := t.TempDir()
	exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "compactctx")

	item := context.Item{
		Source: "file",
		Origin: "spec.pdf",
		Task:   "T-001",
		Data:   []byte("Sample requirements text"),
	}
	if _, err := context.Stage(root, item, time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	p := exec(t, "", false, "", "prompt", "claude", "--compact", "--task", "T-001", "--dir", root)
	if p.code != 0 {
		t.Fatalf("compact prompt failed: %+v", p)
	}

	for _, want := range []string{
		"Focus task: T-001",
		"Focus task context:",
		"spec.pdf",
		"Operating Contract: CLAUDE.md",
	} {
		if !strings.Contains(p.out, want) {
			t.Errorf("compact prompt missing expected context element %q:\n%s", want, p.out)
		}
	}
}
