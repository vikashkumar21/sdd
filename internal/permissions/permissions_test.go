package permissions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClaudeSettingsGeneration(t *testing.T) {
	root := t.TempDir()

	plan, err := BuildPlan(root, []string{"claude"}, false)
	if err != nil {
		t.Fatalf("BuildPlan failed: %v", err)
	}
	if len(plan.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(plan.Files))
	}
	fp := plan.Files[0]
	if fp.Path != ".claude/settings.json" || fp.Action != ActionCreate {
		t.Errorf("unexpected file plan: %+v", fp)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(fp.Content), &parsed); err != nil {
		t.Fatalf("failed to parse generated JSON: %v", err)
	}
	perms, ok := parsed["permissions"].(map[string]any)
	if !ok {
		t.Fatalf("missing permissions object in JSON: %+v", parsed)
	}
	deny, ok := perms["deny"].([]any)
	if !ok || len(deny) != len(ClaudeDeniedPatterns) {
		t.Fatalf("unexpected deny list: %+v", perms)
	}

	// Apply and verify file on disk
	if err := Apply(root, plan, false, time.Now()); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("file not written: %v", err)
	}
	if !strings.Contains(string(data), "git commit *") {
		t.Errorf("missing git commit denial in file:\n%s", string(data))
	}
}

func TestCursorAndGeminiRulesGeneration(t *testing.T) {
	root := t.TempDir()

	plan, err := BuildPlan(root, []string{"cursor", "gemini"}, false)
	if err != nil {
		t.Fatalf("BuildPlan failed: %v", err)
	}
	if len(plan.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(plan.Files))
	}

	if err := Apply(root, plan, false, time.Now()); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	cursorPath := filepath.Join(root, ".cursor", "rules", "sdd-git-protection.mdc")
	cursorData, err := os.ReadFile(cursorPath)
	if err != nil {
		t.Fatalf("cursor rule missing: %v", err)
	}
	if !strings.Contains(string(cursorData), "alwaysApply: true") || !strings.Contains(string(cursorData), "git commit") {
		t.Errorf("unexpected cursor content:\n%s", string(cursorData))
	}

	geminiPath := filepath.Join(root, ".gemini", "rules", "git-protection.md")
	geminiData, err := os.ReadFile(geminiPath)
	if err != nil {
		t.Fatalf("gemini rule missing: %v", err)
	}
	if !strings.Contains(string(geminiData), "git commit") {
		t.Errorf("unexpected gemini content:\n%s", string(geminiData))
	}
}

func TestSettingsJSONMerge(t *testing.T) {
	root := t.TempDir()

	// Pre-create a .claude/settings.json with custom user settings
	initialJSON := `{
  "customKey": "customValue",
  "permissions": {
    "allow": ["commandA *"],
    "deny": ["otherDenied *", "git add *"]
  }
}`
	claudeDir := filepath.Join(root, ".claude")
	_ = os.MkdirAll(claudeDir, 0o755)
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(initialJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(root, []string{"claude"}, false)
	if err != nil {
		t.Fatalf("BuildPlan failed: %v", err)
	}
	if len(plan.Files) != 1 || plan.Files[0].Action != ActionMerge {
		t.Fatalf("expected ActionMerge, got %+v", plan.Files)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(plan.Files[0].Content), &parsed); err != nil {
		t.Fatalf("invalid merged json: %v", err)
	}

	// Verify customKey was preserved
	if parsed["customKey"] != "customValue" {
		t.Errorf("expected customKey to be preserved, got %v", parsed["customKey"])
	}

	// Verify allow list was preserved
	perms := parsed["permissions"].(map[string]any)
	allow := perms["allow"].([]any)
	if len(allow) != 1 || allow[0] != "commandA *" {
		t.Errorf("expected allow list preserved, got %+v", allow)
	}

	// Verify deny list includes "otherDenied *" and all SDD git deny commands without duplicate "git add *"
	deny := perms["deny"].([]any)
	hasOther := false
	addCount := 0
	for _, d := range deny {
		s := d.(string)
		if s == "otherDenied *" {
			hasOther = true
		}
		if s == "git add *" {
			addCount++
		}
	}
	if !hasOther {
		t.Errorf("missing original custom deny pattern otherDenied *")
	}
	if addCount != 1 {
		t.Errorf("expected exactly 1 instance of 'git add *', got %d", addCount)
	}
}

func TestSkipExistingMarkdownWithoutForce(t *testing.T) {
	root := t.TempDir()

	geminiPath := filepath.Join(root, ".gemini", "rules", "git-protection.md")
	_ = os.MkdirAll(filepath.Dir(geminiPath), 0o755)
	_ = os.WriteFile(geminiPath, []byte("USER CUSTOM RULES"), 0o644)

	// Without force: should skip
	plan, err := BuildPlan(root, []string{"gemini"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Files) != 1 || plan.Files[0].Action != ActionSkip {
		t.Fatalf("expected ActionSkip, got %+v", plan.Files)
	}

	// With force: should create/overwrite and backup
	planForce, err := BuildPlan(root, []string{"gemini"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(planForce.Files) != 1 || planForce.Files[0].Action != ActionCreate {
		t.Fatalf("expected ActionCreate with force, got %+v", planForce.Files)
	}

	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	if err := Apply(root, planForce, true, now); err != nil {
		t.Fatal(err)
	}

	// Check that backup was written
	backupDir := filepath.Join(root, ".sdd", "backup", "20261102-120000", ".gemini", "rules", "git-protection.md")
	backupData, err := os.ReadFile(backupDir)
	if err != nil {
		t.Fatalf("backup missing: %v", err)
	}
	if string(backupData) != "USER CUSTOM RULES" {
		t.Errorf("unexpected backup content: %s", string(backupData))
	}
}

func TestNormalizeTargetsAndErrors(t *testing.T) {
	// Defaults to all
	all, err := NormalizeTargets(nil)
	if err != nil || len(all) != 3 {
		t.Errorf("expected 3 default targets, got %v (%v)", all, err)
	}

	// "all" keyword
	allKeyword, err := NormalizeTargets([]string{"all"})
	if err != nil || len(allKeyword) != 3 {
		t.Errorf("expected 3 targets for 'all', got %v (%v)", allKeyword, err)
	}

	// Invalid target
	_, err = NormalizeTargets([]string{"invalid-agent"})
	if err == nil || !strings.Contains(err.Error(), "unsupported agent target") {
		t.Errorf("expected error for invalid target, got %v", err)
	}
}
