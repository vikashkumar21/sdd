package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportProtocolSkillBasic(t *testing.T) {
	root := t.TempDir()

	res, err := ExportProtocolSkill(ExportOptions{
		ProjectRoot: root,
		Clock:       fixedClock,
	})
	if err != nil {
		t.Fatalf("ExportProtocolSkill failed: %v", err)
	}

	expectedFile := filepath.ToSlash(filepath.Join(root, "skills", "sdd-protocol", "SKILL.md"))
	if res.Path != expectedFile {
		t.Errorf("Path = %s, want %s", res.Path, expectedFile)
	}

	content, err := os.ReadFile(filepath.Join(root, "skills", "sdd-protocol", "SKILL.md"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	s := string(content)
	if !strings.Contains(s, "name: sdd-protocol") {
		t.Errorf("missing name in exported skill: %s", s)
	}
	if !strings.Contains(s, "ALWAYS BE DETERMINISTIC") {
		t.Errorf("missing anthem in exported skill: %s", s)
	}
	if !strings.Contains(s, "## The 4-Phase Workflow") {
		t.Errorf("missing workflow in exported skill: %s", s)
	}
}

func TestExportProtocolSkillValid(t *testing.T) {
	parsed, err := Parse([]byte(CanonicalSDDProtocolSkill))
	if err != nil {
		t.Fatalf("canonical skill parse error: %v", err)
	}

	errs := Validate(parsed, filepath.Join("skills", "sdd-protocol"))
	if len(errs) > 0 {
		t.Fatalf("canonical skill validation errors: %+v", errs)
	}

	if parsed.Name != "sdd-protocol" {
		t.Errorf("Name = %s, want sdd-protocol", parsed.Name)
	}
}

func TestExportProtocolSkillTarget(t *testing.T) {
	targetDir := t.TempDir()

	res, err := ExportProtocolSkill(ExportOptions{
		TargetDir: targetDir,
		Clock:     fixedClock,
	})
	if err != nil {
		t.Fatalf("ExportProtocolSkill with target failed: %v", err)
	}

	expected := filepath.ToSlash(filepath.Join(targetDir, "sdd-protocol", "SKILL.md"))
	if res.Path != expected {
		t.Errorf("Path = %s, want %s", res.Path, expected)
	}

	if _, err := os.Stat(filepath.FromSlash(res.Path)); err != nil {
		t.Errorf("exported skill does not exist: %v", err)
	}
}

func TestExportProtocolSkillGlobal(t *testing.T) {
	fakeHome := t.TempDir()
	// Create fake antigravity directory
	geminiSkills := filepath.Join(fakeHome, ".gemini", "antigravity")
	if err := os.MkdirAll(geminiSkills, 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := ExportProtocolSkill(ExportOptions{
		Global: true,
		Home:   func() string { return fakeHome },
		Clock:  fixedClock,
	})
	if err != nil {
		t.Fatalf("ExportProtocolSkill global failed: %v", err)
	}

	expected := filepath.ToSlash(filepath.Join(fakeHome, ".gemini", "antigravity", "skills", "sdd-protocol", "SKILL.md"))
	if res.Path != expected {
		t.Errorf("Path = %s, want %s", res.Path, expected)
	}

	if _, err := os.Stat(filepath.FromSlash(res.Path)); err != nil {
		t.Errorf("global skill file missing: %v", err)
	}
}

func TestExportProtocolSkillCollisionRefusal(t *testing.T) {
	root := t.TempDir()

	_, err := ExportProtocolSkill(ExportOptions{
		ProjectRoot: root,
		Clock:       fixedClock,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Re-export without force
	_, err = ExportProtocolSkill(ExportOptions{
		ProjectRoot: root,
		Clock:       fixedClock,
	})
	if err == nil {
		t.Fatal("expected collision error without force, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExportProtocolSkillForceBackup(t *testing.T) {
	root := t.TempDir()

	_, err := ExportProtocolSkill(ExportOptions{
		ProjectRoot: root,
		Clock:       fixedClock,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Modify skill content to simulate previous version
	skillFile := filepath.Join(root, "skills", "sdd-protocol", "SKILL.md")
	if err := os.WriteFile(skillFile, []byte("Old custom skill content"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := ExportProtocolSkill(ExportOptions{
		ProjectRoot: root,
		Force:       true,
		Clock:       fixedClock,
	})
	if err != nil {
		t.Fatalf("ExportProtocolSkill with force failed: %v", err)
	}

	if res.BackupDir != ".sdd/backup/20261103-120000" {
		t.Errorf("BackupDir = %s, want .sdd/backup/20261103-120000", res.BackupDir)
	}

	backupFile := filepath.Join(root, ".sdd", "backup", "20261103-120000", "skills", "sdd-protocol", "SKILL.md")
	backupBytes, err := os.ReadFile(backupFile)
	if err != nil {
		t.Fatalf("backup file not found: %v", err)
	}
	if string(backupBytes) != "Old custom skill content" {
		t.Errorf("backup content mismatch: %s", string(backupBytes))
	}
}

func TestCanonicalCodeReviewSkillValid(t *testing.T) {
	parsed, err := Parse([]byte(CanonicalCodeReviewSkill))
	if err != nil {
		t.Fatalf("canonical code-review skill parse error: %v", err)
	}

	errs := Validate(parsed, filepath.Join("skills", "code-review"))
	if len(errs) > 0 {
		t.Fatalf("canonical code-review skill validation errors: %+v", errs)
	}

	if parsed.Name != "code-review" {
		t.Errorf("Name = %s, want code-review", parsed.Name)
	}
	if !strings.Contains(CanonicalCodeReviewSkill, "3-Tier Risk & Finding Rubric") {
		t.Errorf("missing rubric in canonical code-review skill")
	}
	if !strings.Contains(CanonicalCodeReviewSkill, "The Ensemble Review Triad") {
		t.Errorf("missing ensemble review triad in canonical code-review skill")
	}
}

func TestCanonicalTaskCoordinatorSkillValid(t *testing.T) {
	parsed, err := Parse([]byte(CanonicalTaskCoordinatorSkill))
	if err != nil {
		t.Fatalf("canonical task-coordinator skill parse error: %v", err)
	}

	errs := Validate(parsed, filepath.Join("skills", "task-coordinator"))
	if len(errs) > 0 {
		t.Fatalf("canonical task-coordinator skill validation errors: %+v", errs)
	}

	if parsed.Name != "task-coordinator" {
		t.Errorf("Name = %s, want task-coordinator", parsed.Name)
	}
	if !strings.Contains(CanonicalTaskCoordinatorSkill, "Two-Tier Agent Roles") {
		t.Errorf("missing two-tier roles in canonical task-coordinator skill")
	}
}

func TestExportSkillsAll(t *testing.T) {
	root := t.TempDir()

	results, err := ExportSkills("all", ExportOptions{
		ProjectRoot: root,
		Clock:       fixedClock,
	})
	if err != nil {
		t.Fatalf("ExportSkills failed: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 exported skills, got %d", len(results))
	}

	protoPath := filepath.Join(root, "skills", "sdd-protocol", "SKILL.md")
	if _, err := os.Stat(protoPath); err != nil {
		t.Errorf("sdd-protocol not exported: %v", err)
	}

	revPath := filepath.Join(root, "skills", "code-review", "SKILL.md")
	if _, err := os.Stat(revPath); err != nil {
		t.Errorf("code-review not exported: %v", err)
	}

	coordPath := filepath.Join(root, "skills", "task-coordinator", "SKILL.md")
	if _, err := os.Stat(coordPath); err != nil {
		t.Errorf("task-coordinator not exported: %v", err)
	}
}
