// Package permissions generates and manages IDE and AI agent permission configuration files
// (e.g. Claude Code, Cursor, Gemini/Antigravity) that technically enforce the SDD contract
// rule prohibiting AI agents from executing git state mutation commands.
package permissions

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vikashkumar21/sdd/internal/fsutil"
)

// Supported targets
const (
	TargetClaude = "claude"
	TargetCursor = "cursor"
	TargetGemini = "gemini"
	TargetAll    = "all"
)

// SupportedTargets lists canonical target identifiers.
func SupportedTargets() []string {
	return []string{TargetClaude, TargetCursor, TargetGemini}
}

// Action describes what will happen to a target file.
type Action string

const (
	ActionCreate Action = "create"
	ActionMerge  Action = "merge"
	ActionSkip   Action = "skip"
)

// FilePlan holds the planned creation or modification of a permission file.
type FilePlan struct {
	Path        string // relative slash-separated path
	Target      string // "claude", "cursor", "gemini"
	Action      Action
	Description string
	Content     string
	BackupPath  string // set if a pre-existing file was backed up
}

// Plan is the set of planned file modifications.
type Plan struct {
	Files []FilePlan
}

// ClaudeDeniedPatterns is the canonical list of bash command patterns denied in Claude Code.
var ClaudeDeniedPatterns = []string{
	"git add *",
	"git checkout -- *",
	"git cherry-pick *",
	"git commit *",
	"git merge *",
	"git push *",
	"git rebase *",
	"git reset *",
	"git stash *",
	"git tag *",
}

const cursorRuleContent = `---
description: Prevent AI coding agents from modifying git state per SDD contract
alwaysApply: true
---

# SDD Git State Protection Rule

> ALWAYS BE DETERMINISTIC. The human reviews diffs and commits.

AI coding agents working in this repository MUST NOT execute any git command that modifies git state:
- ` + "`git add`" + `
- ` + "`git commit`" + `
- ` + "`git push`" + `
- ` + "`git tag`" + `
- ` + "`git reset`" + `
- ` + "`git rebase`" + `
- ` + "`git stash`" + `
- ` + "`git checkout --`" + `
- ` + "`git merge`" + `
- ` + "`git cherry-pick`" + `

Read-only git commands (` + "`git status`" + `, ` + "`git diff`" + `, ` + "`git log`" + `, ` + "`git show`" + `) are permitted.
Suggest commit messages and diff summaries; the human reviews and commits.
`

const geminiRuleContent = `# SDD Git State Protection Rule

> ALWAYS BE DETERMINISTIC. The human reviews diffs and commits.

AI coding agents working in this repository MUST NOT execute any git command that modifies git state:
- ` + "`git add`" + `
- ` + "`git commit`" + `
- ` + "`git push`" + `
- ` + "`git tag`" + `
- ` + "`git reset`" + `
- ` + "`git rebase`" + `
- ` + "`git stash`" + `
- ` + "`git checkout --`" + `
- ` + "`git merge`" + `
- ` + "`git cherry-pick`" + `

Read-only git commands (` + "`git status`" + `, ` + "`git diff`" + `, ` + "`git log`" + `, ` + "`git show`" + `) are permitted.
Suggest commit messages and diff summaries; the human reviews and commits.
`

// NormalizeTargets expands "all" or defaults and removes duplicates.
func NormalizeTargets(targets []string) ([]string, error) {
	if len(targets) == 0 {
		return SupportedTargets(), nil
	}
	seen := map[string]bool{}
	var result []string
	for _, raw := range targets {
		t := strings.ToLower(strings.TrimSpace(raw))
		if t == "" {
			continue
		}
		if t == TargetAll {
			return SupportedTargets(), nil
		}
		switch t {
		case TargetClaude, TargetCursor, TargetGemini:
			if !seen[t] {
				seen[t] = true
				result = append(result, t)
			}
		case "agents":
			if !seen[TargetCursor] {
				seen[TargetCursor] = true
				result = append(result, TargetCursor)
			}
		default:
			return nil, fmt.Errorf("permissions: unsupported agent target %q (valid: claude, cursor, gemini, agents, all)", t)
		}
	}
	sort.Strings(result)
	return result, nil
}

// BuildPlan computes the planned permission files to create or merge.
func BuildPlan(root string, targets []string, force bool) (Plan, error) {
	normTargets, err := NormalizeTargets(targets)
	if err != nil {
		return Plan{}, err
	}

	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return Plan{}, fmt.Errorf("permissions: open root %s: %w", root, err)
	}
	defer func() { _ = rootHandle.Close() }()

	var plan Plan

	for _, target := range normTargets {
		switch target {
		case TargetClaude:
			fp, err := planClaude(rootHandle)
			if err != nil {
				return Plan{}, err
			}
			plan.Files = append(plan.Files, fp)
		case TargetCursor:
			fp, err := planMarkdownFile(rootHandle, ".cursor/rules/sdd-git-protection.mdc", TargetCursor, "Cursor git protection rule", cursorRuleContent, force)
			if err != nil {
				return Plan{}, err
			}
			plan.Files = append(plan.Files, fp)
		case TargetGemini:
			fp, err := planMarkdownFile(rootHandle, ".gemini/rules/git-protection.md", TargetGemini, "Gemini git protection rule", geminiRuleContent, force)
			if err != nil {
				return Plan{}, err
			}
			plan.Files = append(plan.Files, fp)
		}
	}

	return plan, nil
}

func planClaude(root *os.Root) (FilePlan, error) {
	relPath := ".claude/settings.json"
	f, err := root.Open(filepath.FromSlash(relPath))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || os.IsNotExist(err) {
			// Generate fresh settings.json
			content, err := generateFreshClaudeSettings()
			if err != nil {
				return FilePlan{}, err
			}
			return FilePlan{
				Path:        relPath,
				Target:      TargetClaude,
				Action:      ActionCreate,
				Description: "Claude Code settings with git write denial",
				Content:     content,
			}, nil
		}
		return FilePlan{}, fmt.Errorf("permissions: open %s: %w", relPath, err)
	}
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(f)
	if err != nil {
		return FilePlan{}, fmt.Errorf("permissions: read %s: %w", relPath, err)
	}

	merged, err := MergeClaudeSettings(data)
	if err != nil {
		return FilePlan{}, fmt.Errorf("permissions: merge %s: %w", relPath, err)
	}

	return FilePlan{
		Path:        relPath,
		Target:      TargetClaude,
		Action:      ActionMerge,
		Description: "merge git write denial rules into existing Claude Code settings",
		Content:     merged,
	}, nil
}

func generateFreshClaudeSettings() (string, error) {
	settings := map[string]any{
		"permissions": map[string]any{
			"deny": ClaudeDeniedPatterns,
		},
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

// MergeClaudeSettings merges ClaudeDeniedPatterns into existing JSON data.
func MergeClaudeSettings(data []byte) (string, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return generateFreshClaudeSettings()
	}

	var rootObj map[string]any
	if err := json.Unmarshal(data, &rootObj); err != nil {
		return "", fmt.Errorf("invalid json: %w", err)
	}
	if rootObj == nil {
		rootObj = make(map[string]any)
	}

	// Retrieve or create permissions object
	var perms map[string]any
	if p, ok := rootObj["permissions"].(map[string]any); ok {
		perms = p
	} else {
		perms = make(map[string]any)
		rootObj["permissions"] = perms
	}

	// Retrieve or create deny slice
	existingDeny := map[string]bool{}
	var denyList []string
	if dList, ok := perms["deny"].([]any); ok {
		for _, item := range dList {
			if s, ok := item.(string); ok {
				if !existingDeny[s] {
					existingDeny[s] = true
					denyList = append(denyList, s)
				}
			}
		}
	}

	for _, pattern := range ClaudeDeniedPatterns {
		if !existingDeny[pattern] {
			existingDeny[pattern] = true
			denyList = append(denyList, pattern)
		}
	}
	sort.Strings(denyList)

	denyAny := make([]any, len(denyList))
	for i, s := range denyList {
		denyAny[i] = s
	}
	perms["deny"] = denyAny

	marshalled, err := json.MarshalIndent(rootObj, "", "  ")
	if err != nil {
		return "", err
	}
	return string(marshalled) + "\n", nil
}

func planMarkdownFile(root *os.Root, relPath, target, desc, content string, force bool) (FilePlan, error) {
	f, err := root.Open(filepath.FromSlash(relPath))
	if err == nil {
		_ = f.Close()
		if !force {
			return FilePlan{
				Path:        relPath,
				Target:      target,
				Action:      ActionSkip,
				Description: fmt.Sprintf("%s (already exists, skipped without --force)", desc),
				Content:     content,
			}, nil
		}
	} else if !errors.Is(err, fs.ErrNotExist) && !os.IsNotExist(err) {
		return FilePlan{}, fmt.Errorf("permissions: check %s: %w", relPath, err)
	}

	return FilePlan{
		Path:        relPath,
		Target:      target,
		Action:      ActionCreate,
		Description: desc,
		Content:     content,
	}, nil
}

// Apply writes the planned permission files beneath root.
func Apply(root string, plan Plan, force bool, now time.Time) error {
	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return fmt.Errorf("permissions: open root %s: %w", root, err)
	}
	defer func() { _ = rootHandle.Close() }()

	backupRel := filepath.ToSlash(filepath.Join(".sdd", "backup", now.UTC().Format("20060102-150405")))

	for _, f := range plan.Files {
		if f.Action == ActionSkip {
			continue
		}

		// If overwriting an existing file, create backup first
		existingFile, err := rootHandle.Open(filepath.FromSlash(f.Path))
		if err == nil {
			orig, rerr := io.ReadAll(existingFile)
			_ = existingFile.Close()
			if rerr == nil {
				backupPath := filepath.Join(backupRel, filepath.FromSlash(f.Path))
				_ = fsutil.WriteAtomicRoot(rootHandle, backupPath, orig, 0o644)
			}
		}

		if err := fsutil.WriteAtomicRoot(rootHandle, filepath.FromSlash(f.Path), []byte(f.Content), 0o644); err != nil {
			return fmt.Errorf("permissions: write %s: %w", f.Path, err)
		}
	}

	return nil
}
