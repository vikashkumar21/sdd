package cli

import (
	"bytes"
	"encoding/json"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vikashkumar21/sdd/internal/hook"
)

func initTestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := osexec.Command("git", "init", "-b", "main", dir)
	if _, err := cmd.CombinedOutput(); err != nil {
		cmd2 := osexec.Command("git", "init", dir)
		if out2, err2 := cmd2.CombinedOutput(); err2 != nil {
			t.Fatalf("git init failed: %v: %s", err2, string(out2))
		}
	}
	return dir
}

func TestHook_CLIHelpAndUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	env := IO{Out: &stdout, Err: &stderr}

	// 1. No args
	code := runHook([]string{}, env)
	if code != 2 {
		t.Errorf("expected exit code 2 for no args, got %d", code)
	}

	// 2. Help flag
	stdout.Reset()
	stderr.Reset()
	code = runHook([]string{"--help"}, env)
	if code != 0 {
		t.Errorf("expected exit code 0 for --help, got %d", code)
	}
	if !strings.Contains(stdout.String(), "usage: sdd hook") {
		t.Errorf("expected usage output, got: %s", stdout.String())
	}

	// 3. Unknown command
	stdout.Reset()
	stderr.Reset()
	code = runHook([]string{"invalid"}, env)
	if code != 2 {
		t.Errorf("expected exit code 2 for unknown command, got %d", code)
	}
}

func TestHook_CLICommands(t *testing.T) {
	root := initTestGitRepo(t)

	var stdout, stderr bytes.Buffer
	env := IO{Out: &stdout, Err: &stderr}

	// 1. Status before install
	stdout.Reset()
	stderr.Reset()
	code := runHook([]string{"status", "--dir", root, "--json"}, env)
	if code != 0 {
		t.Fatalf("hook status --json failed with code %d: %s", code, stderr.String())
	}
	var st1 hook.StatusReport
	if err := json.Unmarshal(stdout.Bytes(), &st1); err != nil {
		t.Fatalf("unmarshal status json failed: %v", err)
	}
	if st1.Installed {
		t.Errorf("expected installed=false before hook install")
	}

	// 2. Install
	stdout.Reset()
	stderr.Reset()
	code = runHook([]string{"install", "--dir", root}, env)
	if code != 0 {
		t.Fatalf("hook install failed with code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Installed sdd git hooks") {
		t.Errorf("expected install confirmation message, got: %s", stdout.String())
	}

	// 3. Status after install
	stdout.Reset()
	stderr.Reset()
	code = runHook([]string{"status", "--dir", root}, env)
	if code != 0 {
		t.Fatalf("hook status failed with code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Installed:        Yes (sdd managed)") {
		t.Errorf("expected status installed yes, got: %s", stdout.String())
	}

	// 4. Check-gate1 on main branch (should pass)
	stdout.Reset()
	stderr.Reset()
	code = runHook([]string{"check-gate1", "--dir", root, "--branch", "main"}, env)
	if code != 0 {
		t.Fatalf("check-gate1 failed on main with code %d: %s", code, stderr.String())
	}

	// 5. Check-gate1 on task branch with unapproved plan and app code (should fail)
	taskDir := filepath.Join(root, "tasks", "features", "T-042-test")
	_ = os.MkdirAll(taskDir, 0755)
	_ = os.WriteFile(filepath.Join(taskDir, "plan.md"), []byte("# Plan\nStatus: Draft\n"), 0644)
	appDir := filepath.Join(root, "internal", "sample")
	_ = os.MkdirAll(appDir, 0755)
	_ = os.WriteFile(filepath.Join(appDir, "sample.go"), []byte("package sample\n"), 0644)

	_ = osexec.Command("git", "-C", root, "config", "user.name", "Tester").Run()
	_ = osexec.Command("git", "-C", root, "config", "user.email", "tester@test.com").Run()
	_ = osexec.Command("git", "-C", root, "add", "internal/sample/sample.go").Run()

	stdout.Reset()
	stderr.Reset()
	code = runHook([]string{"check-gate1", "--dir", root, "--branch", "task/T-042-test"}, env)
	if code != 1 {
		t.Fatalf("expected check-gate1 to exit 1 for unapproved plan, got %d", code)
	}
	if !strings.Contains(stderr.String(), "GATE 1 VIOLATION") {
		t.Errorf("expected GATE 1 VIOLATION in stderr, got: %s", stderr.String())
	}

	// 6. Uninstall
	stdout.Reset()
	stderr.Reset()
	code = runHook([]string{"uninstall", "--dir", root, "--yes"}, env)
	if code != 0 {
		t.Fatalf("hook uninstall failed with code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Removed sdd git hooks") {
		t.Errorf("expected uninstall output, got: %s", stdout.String())
	}
}

func TestHook_InstallErrors(t *testing.T) {
	dir := t.TempDir() // not a git repo
	var stdout, stderr bytes.Buffer
	env := IO{Out: &stdout, Err: &stderr}

	code := runHook([]string{"install", "--dir", dir}, env)
	if code != 1 {
		t.Errorf("expected exit code 1 installing in non-git repo, got %d", code)
	}
}
