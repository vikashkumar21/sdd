// Package hook manages repository-level Git pre-commit and pre-merge-commit
// hooks to enforce SDD commit discipline and branch protection mechanically.
package hook

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// HookHeader is the signature marker placed in SDD-managed hooks.
const HookHeader = "# sdd managed hook — do not edit manually"

// ManagedHooks lists the standard hooks installed and maintained by sdd.
var ManagedHooks = []string{"pre-commit", "pre-merge-commit"}

var (
	taskBranchRe = regexp.MustCompile(`^(?:task|feature|bugfix|hotfix)/([A-Za-z0-9]+-[0-9]+)`)
)

// InstallOptions configures hook installation.
type InstallOptions struct {
	Force bool
}

// InstallReport details the outcome of hook installation.
type InstallReport struct {
	HooksDir           string   `json:"hooks_dir"`
	PreCommitPath      string   `json:"pre_commit_path"`
	PreMergeCommitPath string   `json:"pre_merge_commit_path"`
	Chained            bool     `json:"chained"`
	OriginalHooks      []string `json:"original_hooks,omitempty"`
}

// StatusReport details the current hook installation status.
type StatusReport struct {
	Installed       bool     `json:"installed"`
	HooksDir        string   `json:"hooks_dir"`
	PreCommitExists bool     `json:"pre_commit_exists"`
	PreMergeExists  bool     `json:"pre_merge_exists"`
	IsSDDHook       bool     `json:"is_sdd_hook"`
	Executable      bool     `json:"executable"`
	Chained         bool     `json:"chained"`
	OriginalHooks   []string `json:"original_hooks,omitempty"`
}

// UninstallOptions configures hook uninstallation.
type UninstallOptions struct {
	Force bool
}

// UninstallReport details the outcome of hook uninstallation.
type UninstallReport struct {
	HooksDir      string   `json:"hooks_dir"`
	RemovedHooks  []string `json:"removed_hooks"`
	RestoredHooks []string `json:"restored_hooks"`
}

// Gate1Report represents the result of a Gate 1 discipline check.
type Gate1Report struct {
	Compliant bool   `json:"compliant"`
	TaskID    string `json:"task_id,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// HookScriptTemplate is the POSIX-compliant shell script provisioned into hooks.
const HookScriptTemplate = `#!/bin/sh
# sdd managed hook — do not edit manually

# 1. Bypass check
if [ "$SDD_SKIP_HOOK" = "1" ]; then
    exit 0
fi

# 2. Resolve current branch and hook directory (worktree compatible)
BRANCH=$(git symbolic-ref --short HEAD 2>/dev/null || git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")
GIT_HOOKS_DIR=$(git rev-parse --git-path hooks 2>/dev/null || echo ".git/hooks")
HOOK_NAME=$(basename "$0")

# 3. Protected Branch Gate (main / master)
if [ "$BRANCH" = "main" ] || [ "$BRANCH" = "master" ]; then
    if [ "$SDD_HUMAN_APPROVED" != "1" ]; then
        if [ -r /dev/tty ] && [ -t 1 ] && [ -t 2 ]; then
            printf "\n[sdd hook] Direct commit/merge to %s requires human approval.\n" "$BRANCH" >/dev/tty
            printf "[sdd hook] Proceed with %s? (y/N): " "$HOOK_NAME" >/dev/tty
            read -r reply </dev/tty
            case "$reply" in
                [yY]|[yY][eE][sS]) ;;
                *) printf "[sdd hook] Action aborted by human.\n" >/dev/tty; exit 1 ;;
            esac
        else
            printf "\n[sdd hook] ERROR: Direct autonomous commit/merge to %s is blocked (Hard Rule 2).\n" "$BRANCH" >&2
            printf "[sdd hook] Only human commits or qualified merges (sdd task merge) are permitted.\n" >&2
            exit 1
        fi
    fi
fi

# 4. Gate 1 Discipline Check (Task, Feature, Bugfix, Hotfix branches)
if echo "$BRANCH" | grep -Eq '^(task/|feature/|bugfix/|hotfix/)[A-Za-z0-9]+-[0-9]+'; then
    if [ -x "./bin/sdd" ]; then
        ./bin/sdd hook check-gate1 --branch "$BRANCH" || exit 1
    elif command -v sdd >/dev/null 2>&1; then
        sdd hook check-gate1 --branch "$BRANCH" || exit 1
    fi
fi

# 5. Pre-commit Kit Consistency Audit (commits only)
if [ "$HOOK_NAME" = "pre-commit" ] && [ "$SDD_SKIP_CHECK" != "1" ]; then
    if [ -x "./bin/sdd" ]; then
        ./bin/sdd check --strict || exit 1
    elif command -v sdd >/dev/null 2>&1; then
        sdd check --strict || exit 1
    fi
fi

# 6. Chain to original hook if present (worktree compatible)
ORIG_HOOK="$GIT_HOOKS_DIR/$HOOK_NAME.sdd-orig"
if [ -f "$ORIG_HOOK" ]; then
    exec "$ORIG_HOOK" "$@"
fi
exit 0
`

// ResolveHooksDir finds the canonical hooks directory for the repository at root.
func ResolveHooksDir(root string) (string, error) {
	// First check if root exists
	fi, err := os.Stat(root)
	if err != nil || !fi.IsDir() {
		return "", fmt.Errorf("project root %q is not a valid directory", root)
	}

	gitPath := filepath.Join(root, ".git")
	gitInfo, err := os.Stat(gitPath)
	if err != nil {
		return "", fmt.Errorf("project at %q is not a git repository (missing .git)", root)
	}

	// Try querying git plumbing for the canonical hooks path
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--git-path", "hooks")
	out, err := cmd.Output()
	if err == nil {
		p := strings.TrimSpace(string(out))
		if p != "" {
			if !filepath.IsAbs(p) {
				p = filepath.Join(root, p)
			}
			return filepath.Clean(p), nil
		}
	}

	// If git rev-parse is unavailable or fails, handle directory vs file directly
	if gitInfo.IsDir() {
		return filepath.Join(root, ".git", "hooks"), nil
	}

	// .git is a file (worktree or submodule); parse gitdir
	data, err := os.ReadFile(gitPath)
	if err == nil && strings.HasPrefix(string(data), "gitdir:") {
		gitdir := strings.TrimSpace(strings.TrimPrefix(string(data), "gitdir:"))
		if !filepath.IsAbs(gitdir) {
			gitdir = filepath.Join(root, gitdir)
		}
		// In a worktree, gitdir looks like /repo/.git/worktrees/<name>
		// The common hooks directory is usually /repo/.git/hooks
		parent := filepath.Dir(gitdir)
		if filepath.Base(parent) == "worktrees" {
			return filepath.Join(filepath.Dir(parent), "hooks"), nil
		}
		return filepath.Join(gitdir, "hooks"), nil
	}

	return filepath.Join(root, ".git", "hooks"), nil
}

// Install provisions SDD pre-commit and pre-merge-commit hooks in the repository.
func Install(root string, opts InstallOptions) (*InstallReport, error) {
	hooksDir, err := ResolveHooksDir(root)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating hooks directory: %w", err)
	}

	report := &InstallReport{
		HooksDir: hooksDir,
	}

	for _, hookName := range ManagedHooks {
		hookPath := filepath.Join(hooksDir, hookName)
		origPath := filepath.Join(hooksDir, hookName+".sdd-orig")

		existingData, readErr := os.ReadFile(hookPath)
		if readErr == nil {
			// File exists
			if strings.Contains(string(existingData), HookHeader) {
				// Already an SDD hook — update in place, keep any existing orig backup untouched
				if _, statErr := os.Stat(origPath); statErr == nil {
					report.Chained = true
					report.OriginalHooks = append(report.OriginalHooks, origPath)
				}
			} else {
				// Foreign hook — backup to .sdd-orig if not already backed up
				if _, statErr := os.Stat(origPath); statErr != nil {
					if err := copyFile(hookPath, origPath); err != nil {
						return nil, fmt.Errorf("backing up foreign hook %s: %w", hookName, err)
					}
				}
				report.Chained = true
				report.OriginalHooks = append(report.OriginalHooks, origPath)
			}
		}

		// Write new hook script with executable permissions (0o755)
		if err := os.WriteFile(hookPath, []byte(HookScriptTemplate), 0o755); err != nil {
			return nil, fmt.Errorf("writing hook %s: %w", hookName, err)
		}
		// Ensure mode is 0o755 (umask might restrict it)
		_ = os.Chmod(hookPath, 0o755)

		if hookName == "pre-commit" {
			report.PreCommitPath = hookPath
		} else if hookName == "pre-merge-commit" {
			report.PreMergeCommitPath = hookPath
		}
	}

	return report, nil
}

// Status checks the status of SDD hooks in the repository.
func Status(root string) (*StatusReport, error) {
	hooksDir, err := ResolveHooksDir(root)
	if err != nil {
		return nil, err
	}

	report := &StatusReport{
		HooksDir: hooksDir,
	}

	allInstalled := true
	allSDD := true
	allExec := true

	for _, hookName := range ManagedHooks {
		hookPath := filepath.Join(hooksDir, hookName)
		origPath := filepath.Join(hooksDir, hookName+".sdd-orig")

		fi, err := os.Stat(hookPath)
		if err != nil {
			allInstalled = false
			continue
		}

		if hookName == "pre-commit" {
			report.PreCommitExists = true
		} else if hookName == "pre-merge-commit" {
			report.PreMergeExists = true
		}

		if fi.Mode()&0o111 == 0 {
			allExec = false
		}

		data, err := os.ReadFile(hookPath)
		if err != nil || !strings.Contains(string(data), HookHeader) {
			allSDD = false
		}

		if _, err := os.Stat(origPath); err == nil {
			report.Chained = true
			report.OriginalHooks = append(report.OriginalHooks, origPath)
		}
	}

	report.Installed = allInstalled
	report.IsSDDHook = allSDD
	report.Executable = allExec

	return report, nil
}

// Uninstall safely removes SDD hooks and restores any chained foreign hooks.
func Uninstall(root string, opts UninstallOptions) (*UninstallReport, error) {
	hooksDir, err := ResolveHooksDir(root)
	if err != nil {
		return nil, err
	}

	report := &UninstallReport{
		HooksDir: hooksDir,
	}

	for _, hookName := range ManagedHooks {
		hookPath := filepath.Join(hooksDir, hookName)
		origPath := filepath.Join(hooksDir, hookName+".sdd-orig")

		data, err := os.ReadFile(hookPath)
		if err == nil {
			// Check if it is an SDD hook
			isSDD := strings.Contains(string(data), HookHeader)
			if !isSDD && !opts.Force {
				return nil, fmt.Errorf("refusing to remove non-sdd hook %q without --force", hookPath)
			}

			if err := os.Remove(hookPath); err != nil {
				return nil, fmt.Errorf("removing hook %s: %w", hookName, err)
			}
			report.RemovedHooks = append(report.RemovedHooks, hookPath)
		}

		// Restore original hook if present
		if _, err := os.Stat(origPath); err == nil {
			if err := os.Rename(origPath, hookPath); err != nil {
				return nil, fmt.Errorf("restoring original hook %s: %w", hookName, err)
			}
			_ = os.Chmod(hookPath, 0o755)
			report.RestoredHooks = append(report.RestoredHooks, hookPath)
		}
	}

	return report, nil
}

// CheckGate1 evaluates Gate 1 plan approval discipline on a task branch.
func CheckGate1(root, branch string) (*Gate1Report, error) {
	if root == "" {
		root = "."
	}
	m := taskBranchRe.FindStringSubmatch(branch)
	if m == nil {
		return &Gate1Report{Compliant: true}, nil
	}
	taskID := m[1]

	// Query staged files
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", root, "diff", "--cached", "--name-only")
	out, err := cmd.Output()
	if err != nil {
		// If git diff fails, fail open/closed? Fail restrictive
		return &Gate1Report{
			Compliant: false,
			TaskID:    taskID,
			Reason:    fmt.Sprintf("could not query staged files: %v", err),
		}, nil
	}

	stagedFiles := strings.Split(strings.TrimSpace(string(out)), "\n")
	hasAppCode := false

	for _, rawPath := range stagedFiles {
		f := strings.TrimSpace(rawPath)
		if f == "" {
			continue
		}
		norm := filepath.ToSlash(f)
		if isApplicationCode(norm) {
			hasAppCode = true
			break
		}
	}

	if !hasAppCode {
		// Only task docs or markdown staged — compliant
		return &Gate1Report{Compliant: true, TaskID: taskID}, nil
	}

	// Application code is staged. Verify that plan.md has an approved record
	planContent, err := loadTaskPlan(root, taskID)
	if err != nil {
		return &Gate1Report{
			Compliant: false,
			TaskID:    taskID,
			Reason:    fmt.Sprintf("task %s has application code staged, but plan.md could not be found: %v", taskID, err),
		}, nil
	}

	if !isPlanApproved(planContent) {
		return &Gate1Report{
			Compliant: false,
			TaskID:    taskID,
			Reason:    fmt.Sprintf("GATE 1 VIOLATION: task %s has application code staged, but plan.md is not approved (Hard Rule 1).\nRun 'sdd prompt plan-review %s' to audit the plan and obtain human approval.", taskID, taskID),
		}, nil
	}

	return &Gate1Report{Compliant: true, TaskID: taskID}, nil
}

func isApplicationCode(normPath string) bool {
	// Task documents, kit definitions, and living docs are not application code
	if strings.HasPrefix(normPath, "tasks/") || strings.HasPrefix(normPath, ".sdd/") || strings.HasPrefix(normPath, ".git/") || strings.HasPrefix(normPath, ".github/") {
		return false
	}
	if strings.HasSuffix(normPath, ".md") || strings.HasSuffix(normPath, ".txt") || strings.HasSuffix(normPath, ".json") || strings.HasSuffix(normPath, ".yaml") || strings.HasSuffix(normPath, ".yml") {
		// Non-application doc/config
		return false
	}
	return true
}

func loadTaskPlan(root, taskID string) (string, error) {
	typologies := []string{"features", "bugfixes", "hotfixes", "spikes"}
	for _, typo := range typologies {
		dir := filepath.Join(root, "tasks", typo)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() && strings.HasPrefix(e.Name(), taskID+"-") {
				planPath := filepath.Join(dir, e.Name(), "plan.md")
				data, err := os.ReadFile(planPath)
				if err == nil {
					return string(data), nil
				}
				// Also check patch.md for hotfixes
				patchPath := filepath.Join(dir, e.Name(), "patch.md")
				patchData, err := os.ReadFile(patchPath)
				if err == nil {
					return string(patchData), nil
				}
			}
		}
	}
	return "", errors.New("task folder not found under tasks/")
}

func isPlanApproved(content string) bool {
	lines := strings.Split(content, "\n")
	hasApprovalRecord := false
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		lower := strings.ToLower(line)

		// Check line-anchored frontmatter or section status
		if strings.HasPrefix(lower, "status:") {
			val := strings.TrimSpace(strings.TrimPrefix(lower, "status:"))
			if strings.HasPrefix(val, "approved") || strings.HasPrefix(val, "implementing") || val == "active" || val == "completed" {
				return true
			}
		}

		if strings.HasPrefix(line, "Approved by:") && !strings.Contains(line, "<human>") && !strings.Contains(line, "Pending") && !strings.Contains(line, "______") {
			hasApprovalRecord = true
			if strings.Contains(line, "APPROVED") || strings.Contains(line, "`APPROVED") {
				return true
			}
		}
		if strings.HasPrefix(line, "Phrase received:") && (strings.Contains(line, "APPROVED") || strings.Contains(line, "`APPROVED")) {
			if hasApprovalRecord {
				return true
			}
		}
	}

	return false
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	fi, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fi.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
