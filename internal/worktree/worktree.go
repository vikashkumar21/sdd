// Package worktree manages isolated Git worktrees for SDD tasks.
package worktree

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vikashkumar21/sdd/internal/fsutil"
	"github.com/vikashkumar21/sdd/internal/roadmap"
)

const (
	// RegistryRelPath is the location of the worktree registry file.
	RegistryRelPath = ".sdd/worktrees.json"
	// WorktreesBaseDir is the directory where worktrees are mounted.
	WorktreesBaseDir = ".sdd/worktrees"
)

var branchRegex = regexp.MustCompile(`^(feature|bugfix|hotfix|spike)/T-\d{3}-[a-z0-9-]+$`)

// Entry describes one active task worktree.
type Entry struct {
	TaskID    string `json:"task_id"`
	Typology  string `json:"typology"`
	Slug      string `json:"slug"`
	Branch    string `json:"branch"`
	Path      string `json:"path"` // slash-separated relative path
	BaseRef   string `json:"base_ref"`
	CreatedAt string `json:"created_at"`
}

// Registry stores the active worktrees schema.
type Registry struct {
	Schema    int     `json:"schema"`
	Worktrees []Entry `json:"worktrees"`
}

// EnsureGitInstalled verifies that git is present on the system PATH.
func EnsureGitInstalled() error {
	if _, err := exec.LookPath("git"); err != nil {
		return errors.New("git command not found on PATH: please install Git to use worktree isolation commands")
	}
	return nil
}

// EnsureGitRepo checks whether root is inside an active git repository.
func EnsureGitRepo(root string) error {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "true" {
		return errors.New("not a git repository: sdd worktree commands require an active git repository (run 'git init' first)")
	}
	return nil
}

// ReadRegistry reads .sdd/worktrees.json if present, returning an empty registry if absent.
func ReadRegistry(root string) (*Registry, error) {
	regPath := filepath.Join(root, filepath.FromSlash(RegistryRelPath))
	data, err := os.ReadFile(regPath)
	if errors.Is(err, os.ErrNotExist) {
		return &Registry{Schema: 1, Worktrees: []Entry{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read worktree registry: %w", err)
	}

	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse worktree registry: %w", err)
	}
	if reg.Worktrees == nil {
		reg.Worktrees = []Entry{}
	}
	return &reg, nil
}

// WriteRegistry writes the worktree registry atomically to .sdd/worktrees.json.
func WriteRegistry(root string, reg *Registry) error {
	if reg == nil {
		return errors.New("cannot write nil registry")
	}
	reg.Schema = 1
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode worktree registry: %w", err)
	}
	data = append(data, '\n')

	regPath := filepath.Join(root, filepath.FromSlash(RegistryRelPath))
	dir := filepath.Dir(regPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create registry directory: %w", err)
	}

	return fsutil.WriteAtomic(regPath, data, 0o644)
}

// TaskLocation describes a discovered task folder and its typology.
type TaskLocation struct {
	TaskID   string
	Typology string
	Slug     string
	Folder   string
}

var taskDirRe = regexp.MustCompile(`^(T-\d{3})-(.+)$`)

// FindTask locates the task folder and determines its typology across the 4 work directories.
func FindTask(root, taskID string) (*TaskLocation, error) {
	normID := strings.ToUpper(strings.TrimSpace(taskID))
	typologies := []struct {
		dir  string
		name string
	}{
		{"features", "feature"},
		{"bugfixes", "bugfix"},
		{"hotfixes", "hotfix"},
		{"spikes", "spike"},
	}

	for _, t := range typologies {
		entries, err := os.ReadDir(filepath.Join(root, t.dir))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), "_") {
				continue
			}
			m := taskDirRe.FindStringSubmatch(e.Name())
			if m != nil && strings.ToUpper(m[1]) == normID {
				return &TaskLocation{
					TaskID:   normID,
					Typology: t.name,
					Slug:     m[2],
					Folder:   filepath.Join(t.dir, e.Name()),
				}, nil
			}
		}
	}

	// If not found in folders, check if it's listed in ROADMAP.md
	roadmapPath := filepath.Join(root, "ROADMAP.md")
	if data, err := os.ReadFile(roadmapPath); err == nil {
		tasks := roadmap.Parse(string(data))
		for _, task := range tasks {
			if strings.ToUpper(task.ID) == normID {
				return nil, fmt.Errorf("task %s has no task folder yet; run 'sdd task new %s <slug>' first", normID, normID)
			}
		}
	}

	return nil, fmt.Errorf("task %s not found in ROADMAP.md or task folders", normID)
}

// StartOptions configures worktree creation.
type StartOptions struct {
	Base    string // Base ref or branch (default HEAD)
	FromTag string // Tag to branch from (primarily for hotfixes)
	Force   bool   // Overwrite existing worktree/branch if already present
	Now     time.Time
}

// EnsureGitignoreWorktrees ensures .sdd/worktrees/ is ignored in .gitignore.
func EnsureGitignoreWorktrees(root string) error {
	gitignorePath := filepath.Join(root, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if errors.Is(err, os.ErrNotExist) {
		// Create .gitignore with .sdd/worktrees/
		content := "# SDD worktrees\n.sdd/worktrees/\n"
		return os.WriteFile(gitignorePath, []byte(content), 0o644)
	}
	if err != nil {
		return fmt.Errorf("read .gitignore: %w", err)
	}

	str := string(data)
	lines := strings.Split(str, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == ".sdd/worktrees" || trimmed == ".sdd/worktrees/" || trimmed == "/.sdd/worktrees/" {
			return nil // already ignored
		}
	}

	var buf bytes.Buffer
	buf.WriteString(str)
	if !strings.HasSuffix(str, "\n") && len(str) > 0 {
		buf.WriteString("\n")
	}
	buf.WriteString("\n# SDD worktrees\n.sdd/worktrees/\n")

	return fsutil.WriteAtomic(gitignorePath, buf.Bytes(), 0o644)
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		return "", fmt.Errorf("git %s: %w (%s)", strings.Join(args, " "), err, msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func latestReleaseTag(root string) string {
	out, err := runGit(root, "describe", "--tags", "--abbrev=0")
	if err == nil && out != "" {
		return out
	}
	return "HEAD"
}

func branchExists(root, branch string) bool {
	_, err := runGit(root, "rev-parse", "--verify", "refs/heads/"+branch)
	return err == nil
}

// Start provisions a git worktree for a task.
func Start(root, taskID string, opts StartOptions) (*Entry, error) {
	if err := EnsureGitInstalled(); err != nil {
		return nil, err
	}
	if err := EnsureGitRepo(root); err != nil {
		return nil, err
	}

	loc, err := FindTask(root, taskID)
	if err != nil {
		return nil, err
	}

	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	branch := fmt.Sprintf("%s/%s-%s", loc.Typology, loc.TaskID, loc.Slug)
	relPath := fmt.Sprintf("%s/%s-%s", WorktreesBaseDir, loc.TaskID, loc.Slug)
	absPath := filepath.Join(root, filepath.FromSlash(relPath))

	// Determine base ref
	baseRef := opts.Base
	if baseRef == "" {
		if loc.Typology == "hotfix" {
			if opts.FromTag != "" {
				baseRef = opts.FromTag
			} else {
				baseRef = latestReleaseTag(root)
			}
		} else {
			baseRef = "HEAD"
		}
	}

	// Check existing registry entry
	reg, err := ReadRegistry(root)
	if err != nil {
		return nil, err
	}

	var existingIdx = -1
	for i, e := range reg.Worktrees {
		if e.TaskID == loc.TaskID {
			existingIdx = i
			break
		}
	}

	if _, statErr := os.Stat(absPath); statErr == nil {
		if !opts.Force {
			return nil, fmt.Errorf("worktree for %s already exists at %s (use --force to re-create)", loc.TaskID, relPath)
		}
		// Force remove existing worktree directory
		_, _ = runGit(root, "worktree", "remove", "--force", absPath)
		_ = os.RemoveAll(absPath)
	}

	// Ensure .gitignore ignores .sdd/worktrees/
	if err := EnsureGitignoreWorktrees(root); err != nil {
		return nil, err
	}

	// Create worktree with git
	if branchExists(root, branch) {
		if opts.Force {
			_, _ = runGit(root, "branch", "-D", branch)
			if _, err := runGit(root, "worktree", "add", "-b", branch, absPath, baseRef); err != nil {
				return nil, fmt.Errorf("create worktree: %w", err)
			}
		} else {
			// Connect existing branch to worktree
			if _, err := runGit(root, "worktree", "add", absPath, branch); err != nil {
				return nil, fmt.Errorf("connect existing branch to worktree: %w", err)
			}
		}
	} else {
		if _, err := runGit(root, "worktree", "add", "-b", branch, absPath, baseRef); err != nil {
			return nil, fmt.Errorf("create worktree: %w", err)
		}
	}

	entry := Entry{
		TaskID:    loc.TaskID,
		Typology:  loc.Typology,
		Slug:      loc.Slug,
		Branch:    branch,
		Path:      filepath.ToSlash(relPath),
		BaseRef:   baseRef,
		CreatedAt: now.UTC().Format(time.RFC3339),
	}

	if existingIdx >= 0 {
		reg.Worktrees[existingIdx] = entry
	} else {
		reg.Worktrees = append(reg.Worktrees, entry)
	}

	if err := WriteRegistry(root, reg); err != nil {
		return nil, err
	}

	return &entry, nil
}

// CleanOptions configures worktree removal.
type CleanOptions struct {
	Force        bool // Remove even if uncommitted changes exist (creates backup)
	DeleteBranch bool // Delete the local task git branch
	Now          time.Time
}

// Clean removes a task worktree, creating an emergency backup if dirty.
func Clean(root, taskID string, opts CleanOptions) error {
	if err := EnsureGitInstalled(); err != nil {
		return err
	}
	if err := EnsureGitRepo(root); err != nil {
		return err
	}

	normID := strings.ToUpper(strings.TrimSpace(taskID))
	reg, err := ReadRegistry(root)
	if err != nil {
		return err
	}

	var entry *Entry
	var entryIdx = -1
	for i, e := range reg.Worktrees {
		if strings.ToUpper(e.TaskID) == normID {
			entry = &reg.Worktrees[i]
			entryIdx = i
			break
		}
	}

	var absPath string
	var branch string
	if entry != nil {
		absPath = filepath.Join(root, filepath.FromSlash(entry.Path))
		branch = entry.Branch
	} else {
		// Look for directory in .sdd/worktrees matching task ID
		worktreesDir := filepath.Join(root, WorktreesBaseDir)
		entries, err := os.ReadDir(worktreesDir)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() && strings.HasPrefix(strings.ToUpper(e.Name()), normID+"-") {
					absPath = filepath.Join(worktreesDir, e.Name())
					break
				}
			}
		}
		if absPath == "" {
			return fmt.Errorf("no active worktree found for %s", normID)
		}
	}

	// Check if directory exists on disk
	if _, statErr := os.Stat(absPath); statErr == nil {
		// Check for dirty uncommitted changes
		statusOut, _ := runGit(absPath, "status", "--porcelain")
		isDirty := strings.TrimSpace(statusOut) != ""

		if isDirty {
			if !opts.Force {
				return fmt.Errorf("worktree for %s has uncommitted changes; commit them, or run 'sdd task clean %s --force' to backup and delete", normID, normID)
			}

			// Perform emergency backup of dirty worktree
			now := opts.Now
			if now.IsZero() {
				now = time.Now()
			}
			backupTimestamp := now.UTC().Format("20060102-150405")
			backupDir := filepath.Join(root, ".sdd", "backup", "worktrees", fmt.Sprintf("%s-%s", normID, backupTimestamp))
			if err := copyDir(absPath, backupDir); err != nil {
				return fmt.Errorf("backup dirty worktree: %w", err)
			}
		}

		// Remove worktree via git
		if _, err := runGit(root, "worktree", "remove", "--force", absPath); err != nil {
			// Fallback: prune and remove directory manually
			_, _ = runGit(root, "worktree", "prune")
			_ = os.RemoveAll(absPath)
		}
	}

	// Optionally delete local task branch
	if opts.DeleteBranch && branch != "" {
		_, _ = runGit(root, "branch", "-D", branch)
	}

	// Prune from registry
	if entryIdx >= 0 {
		reg.Worktrees = append(reg.Worktrees[:entryIdx], reg.Worktrees[entryIdx+1:]...)
		if err := WriteRegistry(root, reg); err != nil {
			return err
		}
	}

	return nil
}

// List returns all registered task worktrees.
func List(root string) ([]Entry, error) {
	reg, err := ReadRegistry(root)
	if err != nil {
		return nil, err
	}
	return reg.Worktrees, nil
}

// IsValidBranch reports whether branch conforms to SDD branch naming standards.
func IsValidBranch(branch string) bool {
	return branchRegex.MatchString(branch)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == ".git" || strings.HasPrefix(rel, ".git/") || strings.HasPrefix(rel, ".git\\") {
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, in)
	return err
}
