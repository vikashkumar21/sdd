package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPermissionsCommand(t *testing.T) {
	root := t.TempDir()

	res := exec(t, "", false, "", "permissions", "--dir", root)
	if res.code != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", res.code, res.err)
	}
	if !strings.Contains(res.out, "Generated/updated 3 agent permission manifest(s)") {
		t.Fatalf("unexpected stdout:\n%s", res.out)
	}

	// Verify all 3 files exist
	claudePath := filepath.Join(root, ".claude", "settings.json")
	if _, err := os.Stat(claudePath); err != nil {
		t.Errorf("missing .claude/settings.json: %v", err)
	}
	cursorPath := filepath.Join(root, ".cursor", "rules", "sdd-git-protection.mdc")
	if _, err := os.Stat(cursorPath); err != nil {
		t.Errorf("missing cursor rule: %v", err)
	}
	geminiPath := filepath.Join(root, ".gemini", "rules", "git-protection.md")
	if _, err := os.Stat(geminiPath); err != nil {
		t.Errorf("missing gemini rule: %v", err)
	}
}

func TestPermissionsDryRun(t *testing.T) {
	root := t.TempDir()

	res := exec(t, "", false, "", "permissions", "--dir", root, "--dry-run")
	if res.code != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", res.code, res.err)
	}
	if !strings.Contains(res.out, "Dry run: 3 manifest(s) planned. No files written.") {
		t.Fatalf("unexpected stdout:\n%s", res.out)
	}

	// Verify no files were created
	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Errorf("expected 0 files after dry-run, found %d", len(entries))
	}
}

func TestPermissionsAgentsFilter(t *testing.T) {
	root := t.TempDir()

	res := exec(t, "", false, "", "permissions", "--dir", root, "--agents", "claude")
	if res.code != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", res.code, res.err)
	}
	if !strings.Contains(res.out, "Generated/updated 1 agent permission manifest(s)") {
		t.Fatalf("unexpected stdout:\n%s", res.out)
	}

	claudePath := filepath.Join(root, ".claude", "settings.json")
	if _, err := os.Stat(claudePath); err != nil {
		t.Errorf("expected .claude/settings.json to exist")
	}

	cursorPath := filepath.Join(root, ".cursor")
	if _, err := os.Stat(cursorPath); err == nil {
		t.Errorf("expected .cursor to NOT exist")
	}
}

func TestInitWithPermissions(t *testing.T) {
	root := t.TempDir()

	res := exec(t, "", false, "", "init", "--yes", "--dir", root, "--permissions", "--name", "permproj")
	if res.code != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", res.code, res.err)
	}

	claudePath := filepath.Join(root, ".claude", "settings.json")
	if _, err := os.Stat(claudePath); err != nil {
		t.Errorf("missing .claude/settings.json after init --permissions: %v", err)
	}
	cursorPath := filepath.Join(root, ".cursor", "rules", "sdd-git-protection.mdc")
	if _, err := os.Stat(cursorPath); err != nil {
		t.Errorf("missing cursor rule after init --permissions: %v", err)
	}
	geminiPath := filepath.Join(root, ".gemini", "rules", "git-protection.md")
	if _, err := os.Stat(geminiPath); err != nil {
		t.Errorf("missing gemini rule after init --permissions: %v", err)
	}
}
