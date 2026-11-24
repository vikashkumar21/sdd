package coordinator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestGitRepo(t *testing.T) (string, func(...string)) {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s failed: %v (%s)", strings.Join(args, " "), err, string(out))
		}
	}

	run("init")
	run("config", "user.name", "Test User")
	run("config", "user.email", "test@example.com")
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Base\n"), 0o644)
	run("add", "README.md")
	run("commit", "-m", "init")

	return dir, run
}

func TestMergeSimulationClean(t *testing.T) {
	dir, run := setupTestGitRepo(t)

	// Create feature branch
	run("checkout", "-b", "feature/T-001-clean")
	_ = os.WriteFile(filepath.Join(dir, "feature.go"), []byte("package main\n"), 0o644)
	run("add", "feature.go")
	run("commit", "-m", "add feature")

	// Get changed files
	files, err := GetChangedFiles(dir, "feature/T-001-clean", "master")
	if err != nil {
		// Try 'main' if branch name is main
		files, err = GetChangedFiles(dir, "feature/T-001-clean", "main")
	}
	if err != nil {
		t.Fatalf("GetChangedFiles failed: %v", err)
	}
	if len(files) != 1 || files[0] != "feature.go" {
		t.Errorf("expected [feature.go], got %+v", files)
	}

	targetBranch := "master"
	if _, err := runGit(dir, "rev-parse", "--verify", "main"); err == nil {
		targetBranch = "main"
	}

	sim, err := SimulateMerge(dir, "feature/T-001-clean", targetBranch)
	if err != nil {
		t.Fatalf("SimulateMerge failed: %v", err)
	}

	if !sim.CanMerge {
		t.Errorf("expected clean merge, got conflicts: %+v", sim.Conflicts)
	}
	if sim.AheadCount != 1 {
		t.Errorf("expected ahead count 1, got %d", sim.AheadCount)
	}
}

func TestMergeSimulationConflict(t *testing.T) {
	dir, run := setupTestGitRepo(t)

	targetBranch := "master"
	if _, err := runGit(dir, "rev-parse", "--verify", "main"); err == nil {
		targetBranch = "main"
	}

	// Create feature branch and edit README.md
	run("checkout", "-b", "feature/T-002-conflict")
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Feature Branch Edit\n"), 0o644)
	run("add", "README.md")
	run("commit", "-m", "edit in feature")

	// Switch to target branch and make conflicting edit
	run("checkout", targetBranch)
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Target Branch Edit\n"), 0o644)
	run("add", "README.md")
	run("commit", "-m", "edit in target")

	sim, err := SimulateMerge(dir, "feature/T-002-conflict", targetBranch)
	if err != nil {
		t.Fatalf("SimulateMerge failed: %v", err)
	}

	if sim.CanMerge {
		t.Errorf("expected merge collision hazard, but CanMerge was true")
	}
	if len(sim.Conflicts) == 0 {
		t.Errorf("expected conflicts list to be non-empty")
	}
	foundReadme := false
	for _, c := range sim.Conflicts {
		if strings.Contains(c, "README.md") {
			foundReadme = true
		}
	}
	if !foundReadme {
		t.Errorf("expected README.md in conflicts: %+v", sim.Conflicts)
	}
}

func TestRebaseTaskBranch(t *testing.T) {
	dir, run := setupTestGitRepo(t)

	targetBranch := "master"
	if _, err := runGit(dir, "rev-parse", "--verify", "main"); err == nil {
		targetBranch = "main"
	}

	run("checkout", "-b", "feature/T-003-rebase")
	_ = os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature\n"), 0o644)
	run("add", "feature.txt")
	run("commit", "-m", "feature commit")

	// Add commit to target
	run("checkout", targetBranch)
	_ = os.WriteFile(filepath.Join(dir, "target.txt"), []byte("target\n"), 0o644)
	run("add", "target.txt")
	run("commit", "-m", "target commit")

	if err := RebaseTaskBranch(dir, "feature/T-003-rebase", targetBranch); err != nil {
		t.Fatalf("RebaseTaskBranch failed: %v", err)
	}
}

func TestExecuteMerge(t *testing.T) {
	dir, run := setupTestGitRepo(t)

	targetBranch := "master"
	if _, err := runGit(dir, "rev-parse", "--verify", "main"); err == nil {
		targetBranch = "main"
	}

	run("checkout", "-b", "feature/T-004-merge")
	_ = os.WriteFile(filepath.Join(dir, "merged.txt"), []byte("merged\n"), 0o644)
	run("add", "merged.txt")
	run("commit", "-m", "merged commit")

	if err := ExecuteMerge(dir, "feature/T-004-merge", targetBranch, "feat: merge feature T-004"); err != nil {
		t.Fatalf("ExecuteMerge failed: %v", err)
	}

	// Verify file exists on target branch
	if _, err := os.Stat(filepath.Join(dir, "merged.txt")); err != nil {
		t.Errorf("expected merged.txt to exist on %s after merge: %v", targetBranch, err)
	}
}

func TestExecuteMerge_TokenInjection(t *testing.T) {
	dir, run := setupTestGitRepo(t)

	targetBranch := "master"
	if _, err := runGit(dir, "rev-parse", "--verify", "main"); err == nil {
		targetBranch = "main"
	}

	// Install a pre-merge-commit hook that requires SDD_HUMAN_APPROVED=1
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("failed to create hooks dir: %v", err)
	}
	hookScript := `#!/bin/sh
if [ "$SDD_HUMAN_APPROVED" != "1" ]; then
    echo "MERGE_BLOCKED_NO_TOKEN" >&2
    exit 1
fi
echo "TOKEN_VERIFIED" > "` + filepath.Join(dir, "merge_hook_token.log") + `"
exit 0
`
	preMergePath := filepath.Join(hooksDir, "pre-merge-commit")
	if err := os.WriteFile(preMergePath, []byte(hookScript), 0o755); err != nil {
		t.Fatalf("failed to write pre-merge-commit hook: %v", err)
	}

	// Create feature branch
	run("checkout", "-b", "feature/T-005-token")
	_ = os.WriteFile(filepath.Join(dir, "token_feature.txt"), []byte("token feature\n"), 0o644)
	run("add", "token_feature.txt")
	run("commit", "-m", "feature with token hook")

	// Verify direct git merge on targetBranch fails without SDD_HUMAN_APPROVED
	run("checkout", targetBranch)
	directMergeCmd := exec.Command("git", "merge", "--no-ff", "-m", "unauthorized merge", "feature/T-005-token")
	directMergeCmd.Dir = dir
	directMergeCmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		// Explicitly no SDD_HUMAN_APPROVED
	}
	out, err := directMergeCmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected direct merge without token to fail via pre-merge-commit hook, got success: %s", string(out))
	}
	if !strings.Contains(string(out), "MERGE_BLOCKED_NO_TOKEN") {
		t.Logf("direct merge output: %s", string(out))
	}
	// Abort failed merge if git left it in merging state
	_, _ = runGit(dir, "merge", "--abort")

	// Now execute merge through ExecuteMerge, which injects SDD_HUMAN_APPROVED=1
	if err := ExecuteMerge(dir, "feature/T-005-token", targetBranch, "feat: authorized merge"); err != nil {
		t.Fatalf("ExecuteMerge failed despite token injection: %v", err)
	}

	// Verify token log was written
	tokenLogPath := filepath.Join(dir, "merge_hook_token.log")
	content, err := os.ReadFile(tokenLogPath)
	if err != nil {
		t.Fatalf("expected merge hook to write token log: %v", err)
	}
	if !strings.Contains(string(content), "TOKEN_VERIFIED") {
		t.Errorf("expected TOKEN_VERIFIED in token log, got: %s", string(content))
	}

	// Verify feature file exists on targetBranch
	if _, err := os.Stat(filepath.Join(dir, "token_feature.txt")); err != nil {
		t.Errorf("expected token_feature.txt to exist after qualified merge: %v", err)
	}
}
