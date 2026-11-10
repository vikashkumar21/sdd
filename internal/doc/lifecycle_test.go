package doc_test

import (
	"strings"
	"testing"

	"github.com/vikashkumar21/sdd/internal/doc"
)

func TestParseFrontmatter(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		content        string
		wantMeta       doc.LifecycleMetadata
		wantHasFM      bool
		wantIsLife     bool
		wantBodyPrefix string
		wantErrSubstr  string
	}{
		{
			name: "full lifecycle frontmatter",
			content: `---
status: active
task_id: T-037
last_reviewed: 2026-11-10
superseded_by: docs/architecture.md
---
# Title
Body content here.
`,
			wantMeta: doc.LifecycleMetadata{
				Status:       "active",
				TaskID:       "T-037",
				LastReviewed: "2026-11-10",
				SupersededBy: "docs/architecture.md",
			},
			wantHasFM:      true,
			wantIsLife:     true,
			wantBodyPrefix: "# Title",
		},
		{
			name: "frontmatter with comments and quotes",
			content: `---
# comment line
status: "draft" # inline comment
task_id: 'T-001'
---
Document body
`,
			wantMeta: doc.LifecycleMetadata{
				Status: "draft",
				TaskID: "T-001",
			},
			wantHasFM:      true,
			wantIsLife:     true,
			wantBodyPrefix: "Document body",
		},
		{
			name: "non-lifecycle frontmatter (e.g. skill)",
			content: `---
name: code-review
description: Run code review
---
# Skill Overview
`,
			wantMeta:       doc.LifecycleMetadata{},
			wantHasFM:      true,
			wantIsLife:     false,
			wantBodyPrefix: "# Skill Overview",
		},
		{
			name: "no frontmatter",
			content: `# Just Markdown
No frontmatter here.
`,
			wantMeta:       doc.LifecycleMetadata{},
			wantHasFM:      false,
			wantIsLife:     false,
			wantBodyPrefix: "# Just Markdown",
		},
		{
			name: "unclosed frontmatter",
			content: `---
status: active
# No closing delimiter
`,
			wantMeta:      doc.LifecycleMetadata{},
			wantHasFM:     true,
			wantIsLife:    false,
			wantErrSubstr: "unclosed '---' frontmatter delimiter",
		},
		{
			name:           "empty content",
			content:        "",
			wantMeta:       doc.LifecycleMetadata{},
			wantHasFM:      false,
			wantIsLife:     false,
			wantBodyPrefix: "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			meta, hasFM, isLife, body, err := doc.ParseFrontmatter([]byte(tc.content))
			if tc.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("expected error containing %q, got: %v", tc.wantErrSubstr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if hasFM != tc.wantHasFM {
				t.Errorf("hasFrontmatter mismatch: got %v, want %v", hasFM, tc.wantHasFM)
			}
			if isLife != tc.wantIsLife {
				t.Errorf("isLifecycle mismatch: got %v, want %v", isLife, tc.wantIsLife)
			}
			if meta != tc.wantMeta {
				t.Errorf("meta mismatch: got %+v, want %+v", meta, tc.wantMeta)
			}
			if tc.wantBodyPrefix != "" && !strings.HasPrefix(strings.TrimSpace(string(body)), tc.wantBodyPrefix) {
				t.Errorf("body prefix mismatch: got %q, want prefix %q", string(body), tc.wantBodyPrefix)
			}
		})
	}
}

func TestValidateLifecycle(t *testing.T) {
	t.Parallel()

	existingFiles := map[string]bool{
		"docs/v2.md":             true,
		"tasks/features/next.md": true,
	}
	mockFileExists := func(path string) bool {
		return existingFiles[path]
	}

	cases := []struct {
		name      string
		meta      doc.LifecycleMetadata
		docPath   string
		wantRules []string
	}{
		{
			name: "valid active status with date",
			meta: doc.LifecycleMetadata{
				Status:       "active",
				LastReviewed: "2026-11-10",
			},
			docPath:   "MISSION.md",
			wantRules: nil,
		},
		{
			name: "valid case insensitive draft",
			meta: doc.LifecycleMetadata{
				Status: "Draft",
			},
			docPath:   "tasks/features/T-037/plan.md",
			wantRules: nil,
		},
		{
			name: "missing status",
			meta: doc.LifecycleMetadata{
				TaskID: "T-037",
			},
			docPath:   "doc.md",
			wantRules: []string{"lifecycle-status"},
		},
		{
			name: "invalid status enum",
			meta: doc.LifecycleMetadata{
				Status: "in_progress",
			},
			docPath:   "doc.md",
			wantRules: []string{"lifecycle-status"},
		},
		{
			name: "invalid date format",
			meta: doc.LifecycleMetadata{
				Status:       "active",
				LastReviewed: "10-11-2026",
			},
			docPath:   "doc.md",
			wantRules: []string{"lifecycle-date"},
		},
		{
			name: "superseded with empty target",
			meta: doc.LifecycleMetadata{
				Status: "superseded",
			},
			docPath:   "doc.md",
			wantRules: []string{"lifecycle-superseded"},
		},
		{
			name: "superseded with non-existent target",
			meta: doc.LifecycleMetadata{
				Status:       "superseded",
				SupersededBy: "nonexistent.md",
			},
			docPath:   "doc.md",
			wantRules: []string{"lifecycle-superseded"},
		},
		{
			name: "superseded with target relative to root",
			meta: doc.LifecycleMetadata{
				Status:       "superseded",
				SupersededBy: "docs/v2.md",
			},
			docPath:   "archive/old.md",
			wantRules: nil,
		},
		{
			name: "superseded with target relative to doc dir",
			meta: doc.LifecycleMetadata{
				Status:       "superseded",
				SupersededBy: "next.md",
			},
			docPath:   "tasks/features/prev.md",
			wantRules: nil,
		},
		{
			name: "multiple validation errors",
			meta: doc.LifecycleMetadata{
				Status:       "bogus",
				LastReviewed: "not-a-date",
			},
			docPath:   "doc.md",
			wantRules: []string{"lifecycle-status", "lifecycle-date"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			errs := doc.ValidateLifecycle(tc.meta, tc.docPath, mockFileExists)
			if len(errs) != len(tc.wantRules) {
				t.Fatalf("expected %d errors, got %d: %v", len(tc.wantRules), len(errs), errs)
			}
			for i, wantRule := range tc.wantRules {
				if errs[i].Rule != wantRule {
					t.Errorf("error[%d] rule mismatch: got %q, want %q (msg: %s)", i, errs[i].Rule, wantRule, errs[i].Message)
				}
			}
		})
	}
}
