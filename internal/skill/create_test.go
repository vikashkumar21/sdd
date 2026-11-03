package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var fixedClock = func() time.Time {
	return time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
}

func TestCreateSkillBasic(t *testing.T) {
	root := t.TempDir()

	res, err := CreateSkill(root, CreateOptions{
		Name:        "code-review",
		Description: "Review pull requests for quality and security.",
		Clock:       fixedClock,
	})
	if err != nil {
		t.Fatalf("CreateSkill failed: %v", err)
	}

	if res.Path != "skills/code-review/SKILL.md" {
		t.Errorf("Path = %s, want skills/code-review/SKILL.md", res.Path)
	}

	content, err := os.ReadFile(filepath.Join(root, "skills", "code-review", "SKILL.md"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	s := string(content)
	if !strings.Contains(s, "name: code-review") {
		t.Errorf("missing name frontmatter in content: %s", s)
	}
	if !strings.Contains(s, "description: Review pull requests for quality and security.") {
		t.Errorf("missing description frontmatter in content: %s", s)
	}
	if !strings.Contains(s, "## Overview & Triggers") {
		t.Errorf("missing Overview & Triggers heading in content: %s", s)
	}
}

func TestCreateSkillInvalidName(t *testing.T) {
	root := t.TempDir()

	invalidNames := []string{"CodeReview", "code_review", "-review", "review-", "code--review", ""}
	for _, name := range invalidNames {
		_, err := CreateSkill(root, CreateOptions{
			Name:  name,
			Clock: fixedClock,
		})
		if err == nil {
			t.Errorf("expected error for invalid name %q, got nil", name)
		}
	}
}

func TestCreateSkillIngestion(t *testing.T) {
	root := t.TempDir()

	// Ingestion from file
	notesFile := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(notesFile, []byte("Step 1: Check tests.\nStep 2: Run linter."), 0o644); err != nil {
		t.Fatal(err)
	}

	resFile, err := CreateSkill(root, CreateOptions{
		Name:     "lint-helper",
		FromFile: "notes.txt",
		Clock:    fixedClock,
	})
	if err != nil {
		t.Fatalf("CreateSkill from file failed: %v", err)
	}

	fileContent, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(resFile.Path)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(fileContent), "Step 1: Check tests.") {
		t.Errorf("missing ingested file content in %s", string(fileContent))
	}

	// Ingestion from text
	resText, err := CreateSkill(root, CreateOptions{
		Name:     "text-helper",
		FromText: "Direct text instructions here.",
		Clock:    fixedClock,
	})
	if err != nil {
		t.Fatalf("CreateSkill from text failed: %v", err)
	}

	textContent, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(resText.Path)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(textContent), "Direct text instructions here.") {
		t.Errorf("missing ingested raw text in %s", string(textContent))
	}
}

func TestCreateSkillDescription(t *testing.T) {
	root := t.TempDir()

	// Default description when omitted
	res, err := CreateSkill(root, CreateOptions{
		Name:  "my-helper",
		Clock: fixedClock,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(res.Path)))
	if !strings.Contains(string(data), "description: Agent skill for my-helper") {
		t.Errorf("expected default description, got:\n%s", string(data))
	}

	// Oversized description (>1024 chars) rejected
	longDesc := strings.Repeat("x", 1025)
	_, err = CreateSkill(root, CreateOptions{
		Name:        "oversized-desc",
		Description: longDesc,
		Clock:       fixedClock,
	})
	if err == nil {
		t.Error("expected error for description > 1024 chars, got nil")
	}
}

func TestCreateSkillInternal(t *testing.T) {
	root := t.TempDir()

	res, err := CreateSkill(root, CreateOptions{
		Name:     "internal-tool",
		Internal: true,
		Clock:    fixedClock,
	})
	if err != nil {
		t.Fatal(err)
	}

	expectedPath := filepath.ToSlash(filepath.Join(".sdd", "skills", "internal-tool", "SKILL.md"))
	if res.Path != expectedPath {
		t.Errorf("Path = %s, want %s", res.Path, expectedPath)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(res.Path))); err != nil {
		t.Errorf("internal skill file not created: %v", err)
	}
}

func TestCreateSkillCollisionRefusal(t *testing.T) {
	root := t.TempDir()

	// First creation succeeds
	_, err := CreateSkill(root, CreateOptions{
		Name:  "duplicate-test",
		Clock: fixedClock,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Second creation without force fails
	_, err = CreateSkill(root, CreateOptions{
		Name:  "duplicate-test",
		Clock: fixedClock,
	})
	if err == nil {
		t.Fatal("expected collision error without force, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCreateSkillForceBackup(t *testing.T) {
	root := t.TempDir()

	// Initial version
	_, err := CreateSkill(root, CreateOptions{
		Name:     "force-test",
		FromText: "Initial version instructions.",
		Clock:    fixedClock,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Overwrite with force
	res, err := CreateSkill(root, CreateOptions{
		Name:     "force-test",
		FromText: "Updated version instructions.",
		Force:    true,
		Clock:    fixedClock,
	})
	if err != nil {
		t.Fatalf("overwrite with force failed: %v", err)
	}

	if res.BackupDir != ".sdd/backup/20261103-120000" {
		t.Errorf("BackupDir = %s, want .sdd/backup/20261103-120000", res.BackupDir)
	}

	backupFile := filepath.Join(root, ".sdd", "backup", "20261103-120000", "skills", "force-test", "SKILL.md")
	backupBytes, err := os.ReadFile(backupFile)
	if err != nil {
		t.Fatalf("backup file not found: %v", err)
	}
	if !strings.Contains(string(backupBytes), "Initial version instructions.") {
		t.Errorf("backup did not preserve initial version: %s", string(backupBytes))
	}

	newBytes, err := os.ReadFile(filepath.Join(root, "skills", "force-test", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(newBytes), "Updated version instructions.") {
		t.Errorf("new file did not get updated: %s", string(newBytes))
	}
}

func TestCreateSkillRefinementBlock(t *testing.T) {
	root := t.TempDir()

	res, err := CreateSkill(root, CreateOptions{
		Name:     "refine-test",
		FromText: "Rough notes to be refined.",
		Clock:    fixedClock,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(res.Path)))
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "<!-- sdd:skill-refine:begin -->") || !strings.Contains(s, "<!-- sdd:skill-refine:end -->") {
		t.Errorf("missing sdd:skill-refine comment markers in:\n%s", s)
	}
	if !strings.Contains(s, "Rough notes to be refined.") {
		t.Errorf("missing raw notes in refinement block:\n%s", s)
	}
	if !strings.Contains(s, "## Overview & Triggers") || !strings.Contains(s, "## Step-by-Step Instructions") {
		t.Errorf("missing standard progressive disclosure sections in:\n%s", s)
	}
}

func TestCreateSkillMutuallyExclusiveInputs(t *testing.T) {
	root := t.TempDir()

	_, err := CreateSkill(root, CreateOptions{
		Name:     "conflict-test",
		FromFile: "something.txt",
		FromText: "something inline",
		Clock:    fixedClock,
	})
	if err == nil {
		t.Error("expected error when both from-file and from-text are specified, got nil")
	}
}
