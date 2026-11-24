package hook

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// Run git init in dir
	cmd := exec.Command("git", "init", "-b", "main", dir)
	if _, err := cmd.CombinedOutput(); err != nil {
		// Fallback for older git without -b
		cmd2 := exec.Command("git", "init", dir)
		if out2, err2 := cmd2.CombinedOutput(); err2 != nil {
			t.Fatalf("git init failed: %v: %s", err2, string(out2))
		}
	}
	_ = os.WriteFile(filepath.Join(dir, ".gitconfig"), []byte("[user]\nname = Test\nemail = test@example.com\n"), 0644)
	return dir
}

func TestInstall_Success(t *testing.T) {
	root := initGitRepo(t)

	rep, err := Install(root, InstallOptions{})
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	if rep.PreCommitPath == "" || rep.PreMergeCommitPath == "" {
		t.Errorf("expected both hook paths populated, got: %+v", rep)
	}
	if rep.Chained {
		t.Errorf("expected chained=false on clean repo")
	}

	// Verify files exist and have executable bit
	for _, name := range ManagedHooks {
		p := filepath.Join(rep.HooksDir, name)
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatalf("hook %s not found: %v", name, err)
		}
		if fi.Mode()&0o111 == 0 {
			t.Errorf("expected executable permissions for %s, got mode: %v", name, fi.Mode())
		}
		content, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(content), HookHeader) {
			t.Errorf("hook %s missing header", name)
		}
	}
}

func TestInstall_NotGitRepo(t *testing.T) {
	dir := t.TempDir()
	_, err := Install(dir, InstallOptions{})
	if err == nil {
		t.Fatal("expected error installing in non-git directory, got nil")
	}
}

func TestInstall_ChainingAndIdempotency(t *testing.T) {
	root := initGitRepo(t)
	hooksDir, err := ResolveHooksDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create foreign pre-commit hook
	foreignContent := "#!/bin/sh\necho 'foreign pre-commit hook'\nexit 0\n"
	preCommitPath := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(preCommitPath, []byte(foreignContent), 0755); err != nil {
		t.Fatal(err)
	}

	// 1. Install should chain
	rep1, err := Install(root, InstallOptions{})
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	if !rep1.Chained {
		t.Errorf("expected rep.Chained=true")
	}

	origPath := filepath.Join(hooksDir, "pre-commit.sdd-orig")
	origBytes, err := os.ReadFile(origPath)
	if err != nil {
		t.Fatalf("orig hook missing: %v", err)
	}
	if string(origBytes) != foreignContent {
		t.Errorf("orig hook content mismatch: %q", string(origBytes))
	}

	// 2. Re-install should be idempotent and NOT overwrite pre-commit.sdd-orig
	rep2, err := Install(root, InstallOptions{})
	if err != nil {
		t.Fatalf("Re-install failed: %v", err)
	}
	if !rep2.Chained {
		t.Errorf("expected rep2.Chained=true")
	}
	origBytesAfter, _ := os.ReadFile(origPath)
	if string(origBytesAfter) != foreignContent {
		t.Errorf("orig hook was clobbered on re-install: %q", string(origBytesAfter))
	}

	// 3. Uninstall should restore the foreign hook
	unrep, err := Uninstall(root, UninstallOptions{})
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}
	if len(unrep.RestoredHooks) == 0 {
		t.Errorf("expected restored hooks, got: %+v", unrep)
	}
	restoredBytes, err := os.ReadFile(preCommitPath)
	if err != nil {
		t.Fatalf("pre-commit missing after restore: %v", err)
	}
	if string(restoredBytes) != foreignContent {
		t.Errorf("restored content mismatch: %q", string(restoredBytes))
	}
	if _, err := os.Stat(origPath); err == nil {
		t.Errorf("origPath should have been removed after restore")
	}
}

func TestStatus(t *testing.T) {
	root := initGitRepo(t)

	// Status before install
	s1, err := Status(root)
	if err != nil {
		t.Fatal(err)
	}
	if s1.Installed || s1.PreCommitExists {
		t.Errorf("expected not installed, got: %+v", s1)
	}

	// Status after install
	if _, err := Install(root, InstallOptions{}); err != nil {
		t.Fatal(err)
	}
	s2, err := Status(root)
	if err != nil {
		t.Fatal(err)
	}
	if !s2.Installed || !s2.PreCommitExists || !s2.PreMergeExists || !s2.IsSDDHook || !s2.Executable {
		t.Errorf("unexpected status after install: %+v", s2)
	}
}

func TestUninstall_RefuseForeignWithoutForce(t *testing.T) {
	root := initGitRepo(t)
	hooksDir, _ := ResolveHooksDir(root)
	_ = os.MkdirAll(hooksDir, 0755)

	foreignHook := filepath.Join(hooksDir, "pre-commit")
	_ = os.WriteFile(foreignHook, []byte("#!/bin/sh\necho foreign\n"), 0755)

	// Uninstall without force should fail
	_, err := Uninstall(root, UninstallOptions{Force: false})
	if err == nil || !strings.Contains(err.Error(), "refusing to remove non-sdd hook") {
		t.Fatalf("expected refusal error, got: %v", err)
	}

	// Uninstall with force should succeed
	_, err = Uninstall(root, UninstallOptions{Force: true})
	if err != nil {
		t.Fatalf("expected force uninstall to succeed, got: %v", err)
	}
	if _, err := os.Stat(foreignHook); err == nil {
		t.Errorf("foreign hook should have been removed with force")
	}
}

func TestCheckGate1(t *testing.T) {
	root := initGitRepo(t)

	// 1. Non-task branch is compliant
	r1, err := CheckGate1(root, "main")
	if err != nil || !r1.Compliant {
		t.Fatalf("expected compliant for main branch, got: %+v, err: %v", r1, err)
	}

	// Configure git author
	cmd := exec.Command("git", "-C", root, "config", "user.name", "Tester")
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", root, "config", "user.email", "test@test.com")
	_ = cmd.Run()

	// Stage a task document (markdown only) on task branch
	taskDocPath := filepath.Join(root, "tasks", "features", "T-099-demo")
	_ = os.MkdirAll(taskDocPath, 0755)
	_ = os.WriteFile(filepath.Join(taskDocPath, "requirements.md"), []byte("# Requirements\n"), 0644)

	cmd = exec.Command("git", "-C", root, "add", "tasks/features/T-099-demo/requirements.md")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v: %s", err, string(out))
	}

	r2, err := CheckGate1(root, "task/T-099-demo")
	if err != nil || !r2.Compliant {
		t.Fatalf("expected compliant when only task docs staged, got: %+v, err: %v", r2, err)
	}

	// Stage application code with unapproved plan
	_ = os.WriteFile(filepath.Join(taskDocPath, "plan.md"), []byte("# Plan\nStatus: Draft\n"), 0644)
	appCodePath := filepath.Join(root, "internal", "pkg")
	_ = os.MkdirAll(appCodePath, 0755)
	_ = os.WriteFile(filepath.Join(appCodePath, "app.go"), []byte("package pkg\n"), 0644)

	cmd = exec.Command("git", "-C", root, "add", "internal/pkg/app.go")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v: %s", err, string(out))
	}

	r3, err := CheckGate1(root, "feature/T-099-demo")
	if err != nil {
		t.Fatal(err)
	}
	if r3.Compliant {
		t.Fatalf("expected Gate 1 non-compliant for unapproved plan, got compliant: %+v", r3)
	}
	if !strings.Contains(r3.Reason, "GATE 1 VIOLATION") {
		t.Errorf("expected GATE 1 VIOLATION reason, got: %s", r3.Reason)
	}

	// Update plan to Approved
	_ = os.WriteFile(filepath.Join(taskDocPath, "plan.md"), []byte("---\nstatus: approved\n---\n# Plan\nStatus: Approved\n"), 0644)
	r4, err := CheckGate1(root, "feature/T-099-demo")
	if err != nil || !r4.Compliant {
		t.Fatalf("expected compliant for approved plan, got: %+v, err: %v", r4, err)
	}
}

func TestHookScriptExecution(t *testing.T) {
	root := initGitRepo(t)
	rep, err := Install(root, InstallOptions{})
	if err != nil {
		t.Fatal(err)
	}

	hookScript := rep.PreCommitPath

	// 1. On main without TTY and without SDD_HUMAN_APPROVED: must exit 1
	cmd := exec.Command("sh", hookScript)
	cmd.Dir = root
	// Non-interactive (no tty connected to stdin/stdout/stderr)
	cmd.Env = append(os.Environ(), "SDD_SKIP_CHECK=1", "SDD_HUMAN_APPROVED=")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected hook to exit with failure on main non-interactively, got success: %s", string(out))
	}
	if !strings.Contains(string(out), "Direct autonomous commit/merge to main is blocked") {
		t.Errorf("expected blocked message on main, got: %s", string(out))
	}

	// 2. On main with SDD_HUMAN_APPROVED=1: succeeds
	cmd2 := exec.Command("sh", hookScript)
	cmd2.Dir = root
	cmd2.Env = append(os.Environ(), "SDD_SKIP_CHECK=1", "SDD_HUMAN_APPROVED=1")
	out2, err2 := cmd2.CombinedOutput()
	if err2 != nil {
		t.Fatalf("expected hook to succeed with SDD_HUMAN_APPROVED=1, failed: %v: %s", err2, string(out2))
	}

	// 3. With SDD_SKIP_HOOK=1: bypasses completely
	cmd3 := exec.Command("sh", hookScript)
	cmd3.Dir = root
	cmd3.Env = append(os.Environ(), "SDD_SKIP_HOOK=1", "SDD_HUMAN_APPROVED=")
	out3, err3 := cmd3.CombinedOutput()
	if err3 != nil {
		t.Fatalf("expected hook to bypass with SDD_SKIP_HOOK=1, failed: %v: %s", err3, string(out3))
	}
}

func TestHookScriptExecution_CheckStrictFailure(t *testing.T) {
	root := initGitRepo(t)
	rep, err := Install(root, InstallOptions{})
	if err != nil {
		t.Fatal(err)
	}

	hookScript := rep.PreCommitPath

	// Create a mock sdd binary in a temp directory that fails on "check --strict"
	mockBinDir := t.TempDir()
	mockSddPath := filepath.Join(mockBinDir, "sdd")
	mockScript := `#!/bin/sh
if [ "$1" = "check" ]; then
    echo "SIMULATED_CHECK_STRICT_FAILURE: mechanical audit failed" >&2
    exit 1
fi
exit 0
`
	if err := os.WriteFile(mockSddPath, []byte(mockScript), 0o755); err != nil {
		t.Fatalf("failed to write mock sdd script: %v", err)
	}

	// 1. Execute hook with SDD_HUMAN_APPROVED=1, mock bin dir on PATH, but NO SDD_SKIP_CHECK
	cmd := exec.Command("sh", hookScript)
	cmd.Dir = root
	cmd.Env = []string{
		"PATH=" + mockBinDir + ":" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"SDD_HUMAN_APPROVED=1",
	}
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected pre-commit hook to abort when sdd check --strict fails, got success: %s", string(out))
	}
	if !strings.Contains(string(out), "SIMULATED_CHECK_STRICT_FAILURE") {
		t.Errorf("expected failure message from sdd check in hook output, got: %s", string(out))
	}

	// 2. Execute hook with SDD_SKIP_CHECK=1: should bypass check failure and succeed
	cmdBypass := exec.Command("sh", hookScript)
	cmdBypass.Dir = root
	cmdBypass.Env = []string{
		"PATH=" + mockBinDir + ":" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"SDD_HUMAN_APPROVED=1",
		"SDD_SKIP_CHECK=1",
	}
	outBypass, errBypass := cmdBypass.CombinedOutput()
	if errBypass != nil {
		t.Fatalf("expected pre-commit hook to succeed with SDD_SKIP_CHECK=1, failed: %v: %s", errBypass, string(outBypass))
	}

	// 3. Execute hook with a passing mock sdd binary: should succeed
	mockBinDirPass := t.TempDir()
	mockSddPassPath := filepath.Join(mockBinDirPass, "sdd")
	_ = os.WriteFile(mockSddPassPath, []byte("#!/bin/sh\nexit 0\n"), 0o755)

	cmdPass := exec.Command("sh", hookScript)
	cmdPass.Dir = root
	cmdPass.Env = []string{
		"PATH=" + mockBinDirPass + ":" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"SDD_HUMAN_APPROVED=1",
	}
	outPass, errPass := cmdPass.CombinedOutput()
	if errPass != nil {
		t.Fatalf("expected pre-commit hook to succeed when sdd check succeeds, failed: %v: %s", errPass, string(outPass))
	}
}
