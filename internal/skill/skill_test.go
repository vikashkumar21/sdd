package skill_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vikashkumar21/sdd/internal/skill"
)

func TestParseSkill(t *testing.T) {
	t.Parallel()

	content := `---
name: code-review
description: "Review pull requests for security vulnerabilities and Go idioms."
license: MIT
compatibility: "Go 1.22+, golangci-lint v2"
allowed-tools: read_file run_command
metadata:
  author: security-team
  version: 1.0.0
---

# Code Review Guide
Follow these guidelines when inspecting pull requests.
`

	sk, err := skill.Parse([]byte(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sk.Name != "code-review" {
		t.Errorf("expected name 'code-review', got %q", sk.Name)
	}
	if sk.Description != "Review pull requests for security vulnerabilities and Go idioms." {
		t.Errorf("expected description, got %q", sk.Description)
	}
	if sk.License != "MIT" {
		t.Errorf("expected license 'MIT', got %q", sk.License)
	}
	if sk.Compatibility != "Go 1.22+, golangci-lint v2" {
		t.Errorf("expected compatibility, got %q", sk.Compatibility)
	}
	if sk.AllowedTools != "read_file run_command" {
		t.Errorf("expected allowed tools, got %q", sk.AllowedTools)
	}
	if sk.Metadata["author"] != "security-team" || sk.Metadata["version"] != "1.0.0" {
		t.Errorf("unexpected metadata: %+v", sk.Metadata)
	}
	if !strings.Contains(sk.Body, "# Code Review Guide") {
		t.Errorf("unexpected body: %q", sk.Body)
	}
}

func TestParseSkillBlockScalars(t *testing.T) {
	t.Parallel()

	// Literal block scalar '|'
	contentPipe := `---
name: db-migrate
description: |
  Perform PostgreSQL schema migrations.
  Supports goose and rollback tests.
---

Instructions here.
`
	skPipe, err := skill.Parse([]byte(contentPipe))
	if err != nil {
		t.Fatalf("unexpected error on pipe block: %v", err)
	}
	if !strings.Contains(skPipe.Description, "Perform PostgreSQL schema migrations.\nSupports goose and rollback tests.") {
		t.Errorf("expected multiline description with newline, got %q", skPipe.Description)
	}

	// Folded block scalar '>'
	contentFold := `---
name: db-migrate
description: >
  Perform PostgreSQL schema migrations.
  Supports goose and rollback tests.
---

Instructions here.
`
	skFold, err := skill.Parse([]byte(contentFold))
	if err != nil {
		t.Fatalf("unexpected error on fold block: %v", err)
	}
	if !strings.Contains(skFold.Description, "Perform PostgreSQL schema migrations. Supports goose and rollback tests.") {
		t.Errorf("expected folded description with spaces, got %q", skFold.Description)
	}
}

func TestParseSkillMissingDelimiters(t *testing.T) {
	t.Parallel()

	// Missing opening delimiter
	_, err1 := skill.Parse([]byte("name: test\n---\nbody"))
	if err1 == nil {
		t.Error("expected error for missing opening delimiter, got nil")
	}

	// Missing closing delimiter
	_, err2 := skill.Parse([]byte("---\nname: test\nbody without closing"))
	if err2 == nil {
		t.Error("expected error for missing closing delimiter, got nil")
	}
}

func TestValidateSkillName(t *testing.T) {
	t.Parallel()

	validNames := []string{
		"code-review",
		"db-migrate",
		"skill123",
		"a",
		"a-b-c-d",
	}
	for _, n := range validNames {
		errs := skill.Validate(skill.Skill{Name: n, Description: "Valid description"}, n)
		if len(errs) != 0 {
			t.Errorf("expected name %q to be valid, got errors: %v", n, errs)
		}
	}

	invalidNames := []struct {
		name string
		desc string
	}{
		{"CodeReview", "uppercase characters"},
		{"code_review", "underscore"},
		{"code--review", "consecutive hyphens"},
		{"-code", "leading hyphen"},
		{"code-", "trailing hyphen"},
		{"", "empty"},
		{strings.Repeat("a", 65), "exceeds 64 chars"},
	}
	for _, tc := range invalidNames {
		errs := skill.Validate(skill.Skill{Name: tc.name, Description: "Valid description"}, "")
		if len(errs) == 0 {
			t.Errorf("expected name %q (%s) to fail validation, got no errors", tc.name, tc.desc)
		}
	}
}

func TestValidateSkillDirectoryMatch(t *testing.T) {
	t.Parallel()

	// Matching directory
	sk := skill.Skill{Name: "code-review", Description: "Valid description"}
	errs := skill.Validate(sk, "code-review")
	if len(errs) != 0 {
		t.Fatalf("expected no errors for matching directory, got: %v", errs)
	}

	// Mismatching directory
	errs2 := skill.Validate(sk, "other-directory")
	if len(errs2) == 0 {
		t.Fatal("expected error for mismatching directory, got none")
	}
	if !strings.Contains(errs2[0].Message, "does not match parent directory") {
		t.Errorf("unexpected error message: %v", errs2[0])
	}
}

func TestValidateSkillDescription(t *testing.T) {
	t.Parallel()

	// Empty description
	skEmpty := skill.Skill{Name: "test", Description: "   "}
	errsEmpty := skill.Validate(skEmpty, "test")
	if len(errsEmpty) == 0 {
		t.Fatal("expected error for empty description, got none")
	}

	// Oversized description (> 1024)
	skLarge := skill.Skill{Name: "test", Description: strings.Repeat("x", 1025)}
	errsLarge := skill.Validate(skLarge, "test")
	if len(errsLarge) == 0 {
		t.Fatal("expected error for description > 1024 chars, got none")
	}
}

func TestValidateSkillCompatibility(t *testing.T) {
	t.Parallel()

	skOversizedCompat := skill.Skill{
		Name:          "test",
		Description:   "Valid description",
		Compatibility: strings.Repeat("c", 501),
	}
	errs := skill.Validate(skOversizedCompat, "test")
	if len(errs) == 0 {
		t.Fatal("expected error for compatibility > 500 chars, got none")
	}
}

func TestDiscover(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// 1. Create valid skill in skills/code-review/SKILL.md
	crDir := filepath.Join(root, "skills", "code-review")
	if err := os.MkdirAll(crDir, 0o755); err != nil {
		t.Fatal(err)
	}
	crContent := `---
name: code-review
description: Review code changes.
---
Body text.
`
	if err := os.WriteFile(filepath.Join(crDir, "SKILL.md"), []byte(crContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// 2. Create valid skill in .sdd/skills/db-migrate/SKILL.md
	dbDir := filepath.Join(root, ".sdd", "skills", "db-migrate")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dbContent := `---
name: db-migrate
description: Database migration guidelines.
---
Body text.
`
	if err := os.WriteFile(filepath.Join(dbDir, "SKILL.md"), []byte(dbContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// 3. Create non-skill dir without SKILL.md
	emptyDir := filepath.Join(root, "skills", "not-a-skill")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}

	discovered, err := skill.Discover(root)
	if err != nil {
		t.Fatalf("unexpected discover error: %v", err)
	}

	if len(discovered) != 2 {
		t.Fatalf("expected 2 discovered skills, got %d", len(discovered))
	}

	names := []string{discovered[0].Skill.Name, discovered[1].Skill.Name}
	if !strings.Contains(strings.Join(names, ","), "code-review") || !strings.Contains(strings.Join(names, ","), "db-migrate") {
		t.Errorf("unexpected discovered skills: %+v", discovered)
	}
}
