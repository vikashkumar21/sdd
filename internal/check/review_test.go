package check

import (
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
