package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupGitRepo initialises a temporary git repository with an initial commit.
func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v (%s)", strings.Join(args, " "), err, string(out))
		}
	}

	run("init")
	run("config", "user.name", "Test User")
	run("config", "user.email", "test@example.com")

	// Create initial files
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test Repo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-m", "Initial commit")

	return dir
}

func TestWorktreeStartAndList(t *testing.T) {
	root := setupGitRepo(t)

	// Create a feature task folder
	taskDir := filepath.Join(root, "features", "T-001-worktree-demo")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}

	entry, err := Start(root, "T-001", StartOptions{Now: time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if entry.TaskID != "T-001" {
		t.Errorf("expected TaskID T-001, got %s", entry.TaskID)
	}
	if entry.Typology != "feature" {
		t.Errorf("expected Typology feature, got %s", entry.Typology)
	}
	if entry.Branch != "feature/T-001-worktree-demo" {
		t.Errorf("expected Branch feature/T-001-worktree-demo, got %s", entry.Branch)
	}
	if !IsValidBranch(entry.Branch) {
		t.Errorf("expected branch to pass IsValidBranch: %s", entry.Branch)
	}

	// Verify worktree exists on disk
	absWorktree := filepath.Join(root, filepath.FromSlash(entry.Path))
	if _, err := os.Stat(absWorktree); err != nil {
		t.Fatalf("worktree directory does not exist on disk: %v", err)
	}

	// Verify list returns entry
	list, err := List(root)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 worktree in list, got %d", len(list))
	}
	if list[0].TaskID != "T-001" {
		t.Errorf("expected list task ID T-001, got %s", list[0].TaskID)
	}
}

func TestBranchNamingTypologies(t *testing.T) {
	root := setupGitRepo(t)

	cases := []struct {
		dir            string
		taskFolder     string
		taskID         string
		expectedTypo   string
		expectedBranch string
	}{
		{"features", "T-010-user-auth", "T-010", "feature", "feature/T-010-user-auth"},
		{"bugfixes", "T-011-null-pointer", "T-011", "bugfix", "bugfix/T-011-null-pointer"},
		{"hotfixes", "T-012-data-corruption", "T-012", "hotfix", "hotfix/T-012-data-corruption"},
		{"spikes", "T-013-wasm-eval", "T-013", "spike", "spike/T-013-wasm-eval"},
	}

	for _, c := range cases {
		taskPath := filepath.Join(root, c.dir, c.taskFolder)
		if err := os.MkdirAll(taskPath, 0o755); err != nil {
			t.Fatal(err)
		}

		entry, err := Start(root, c.taskID, StartOptions{})
		if err != nil {
			t.Fatalf("Start %s failed: %v", c.taskID, err)
		}

		if entry.Typology != c.expectedTypo {
			t.Errorf("expected Typology %s, got %s", c.expectedTypo, entry.Typology)
		}
		if entry.Branch != c.expectedBranch {
			t.Errorf("expected Branch %s, got %s", c.expectedBranch, entry.Branch)
		}
		if !IsValidBranch(entry.Branch) {
			t.Errorf("expected valid branch format for %s", entry.Branch)
		}
	}
}

func TestHotfixBaseRef(t *testing.T) {
	root := setupGitRepo(t)

	// Create a tag v1.0.0
	cmd := exec.Command("git", "tag", "v1.0.0")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git tag failed: %v (%s)", err, string(out))
	}

	// Commit something else to advance HEAD past tag
	if err := os.WriteFile(filepath.Join(root, "other.txt"), []byte("more"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmdAdd := exec.Command("git", "add", "other.txt")
	cmdAdd.Dir = root
	_ = cmdAdd.Run()
	cmdCommit := exec.Command("git", "commit", "-m", "advance HEAD")
	cmdCommit.Dir = root
	_ = cmdCommit.Run()

	hotfixDir := filepath.Join(root, "hotfixes", "T-020-security-fix")
	if err := os.MkdirAll(hotfixDir, 0o755); err != nil {
		t.Fatal(err)
	}

	entry, err := Start(root, "T-020", StartOptions{})
	if err != nil {
		t.Fatalf("Start hotfix failed: %v", err)
	}

	if entry.BaseRef != "v1.0.0" {
		t.Errorf("expected hotfix to default to latest release tag v1.0.0, got %s", entry.BaseRef)
	}
}

func TestGitignoreWorktrees(t *testing.T) {
	root := setupGitRepo(t)

	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("node_modules/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	taskDir := filepath.Join(root, "features", "T-001-demo")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := Start(root, "T-001", StartOptions{}); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, ".sdd/worktrees/") {
		t.Errorf("expected .gitignore to contain .sdd/worktrees/, got:\n%s", content)
	}
	if !strings.Contains(content, "node_modules/") {
		t.Errorf("expected existing .gitignore rules preserved, got:\n%s", content)
	}
}

func TestWorktreeCleanDirtyBackup(t *testing.T) {
	root := setupGitRepo(t)

	taskDir := filepath.Join(root, "features", "T-001-backup-demo")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}

	fixedNow := time.Date(2026, 11, 5, 14, 30, 0, 0, time.UTC)
	entry, err := Start(root, "T-001", StartOptions{Now: fixedNow})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	absWorktree := filepath.Join(root, filepath.FromSlash(entry.Path))

	// Create an uncommitted file in the worktree
	dirtyFile := filepath.Join(absWorktree, "wip.txt")
	if err := os.WriteFile(dirtyFile, []byte("uncommitted agent work"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1. Clean without --force should error
	errWithoutForce := Clean(root, "T-001", CleanOptions{Force: false})
	if errWithoutForce == nil || !strings.Contains(errWithoutForce.Error(), "uncommitted changes") {
		t.Fatalf("expected error on dirty worktree without --force, got: %v", errWithoutForce)
	}

	// 2. Clean with --force should succeed and create a backup
	errWithForce := Clean(root, "T-001", CleanOptions{
		Force:        true,
		DeleteBranch: true,
		Now:          fixedNow,
	})
	if errWithForce != nil {
		t.Fatalf("Clean with force failed: %v", errWithForce)
	}

	// Verify worktree directory is removed
	if _, err := os.Stat(absWorktree); !os.IsNotExist(err) {
		t.Errorf("expected worktree to be removed, but stat was: %v", err)
	}

	// Verify backup was created
	backupPath := filepath.Join(root, ".sdd", "backup", "worktrees", "T-001-20261105-143000", "wip.txt")
	backupBytes, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("expected backup file %s, read error: %v", backupPath, err)
	}
	if string(backupBytes) != "uncommitted agent work" {
		t.Errorf("unexpected backup content: %s", string(backupBytes))
	}

	// Verify registry is empty
	list, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 worktrees in registry after clean, got %d", len(list))
	}
}

func TestFailSafeNonGit(t *testing.T) {
	emptyDir := t.TempDir()

	err := EnsureGitRepo(emptyDir)
	if err == nil || !strings.Contains(err.Error(), "not a git repository") {
		t.Fatalf("expected not a git repository error, got: %v", err)
	}

	_, startErr := Start(emptyDir, "T-001", StartOptions{})
	if startErr == nil || !strings.Contains(startErr.Error(), "not a git repository") {
		t.Fatalf("expected Start to fail in non-git dir, got: %v", startErr)
	}
}
