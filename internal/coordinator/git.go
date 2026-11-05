package coordinator

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// MergeSimulation contains the results of a 3-way merge simulation.
type MergeSimulation struct {
	CanMerge    bool     `json:"can_merge"`
	MergeBase   string   `json:"merge_base"`
	BehindCount int      `json:"behind_count"`
	AheadCount  int      `json:"ahead_count"`
	Conflicts   []string `json:"conflicts,omitempty"`
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		return "", fmt.Errorf("git %s: %s (%w)", strings.Join(args, " "), msg, err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// GetChangedFiles returns the list of files modified on taskBranch relative to targetBranch.
func GetChangedFiles(root string, taskBranch, targetBranch string) ([]string, error) {
	// Find merge base
	base, err := runGit(root, "merge-base", targetBranch, taskBranch)
	if err != nil {
		return nil, fmt.Errorf("finding merge-base between %s and %s: %w", targetBranch, taskBranch, err)
	}

	out, err := runGit(root, "diff", "--name-only", base, taskBranch)
	if err != nil {
		return nil, fmt.Errorf("getting changed files: %w", err)
	}

	var files []string
	seen := make(map[string]bool)
	for _, line := range strings.Split(out, "\n") {
		f := strings.TrimSpace(line)
		if f != "" && !seen[f] {
			seen[f] = true
			files = append(files, filepath.ToSlash(f))
		}
	}
	sort.Strings(files)
	return files, nil
}

// SimulateMerge evaluates whether taskBranch can be merged cleanly into targetBranch without altering working tree state.
func SimulateMerge(root string, taskBranch, targetBranch string) (*MergeSimulation, error) {
	// 1. Get merge base
	base, err := runGit(root, "merge-base", targetBranch, taskBranch)
	if err != nil {
		return nil, fmt.Errorf("finding merge-base: %w", err)
	}

	// 2. Get commit divergence (behind / ahead)
	countsStr, err := runGit(root, "rev-list", "--left-right", "--count", targetBranch+"..."+taskBranch)
	var behind, ahead int
	if err == nil {
		fields := strings.Fields(countsStr)
		if len(fields) == 2 {
			behind, _ = strconv.Atoi(fields[0])
			ahead, _ = strconv.Atoi(fields[1])
		}
	}

	// 3. Try modern git merge-tree --write-tree
	cmd := exec.Command("git", "merge-tree", "--write-tree", targetBranch, taskBranch)
	cmd.Dir = root
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	runErr := cmd.Run()

	combined := outBuf.String() + "\n" + errBuf.String()

	if runErr == nil {
		// Clean merge
		return &MergeSimulation{
			CanMerge:    true,
			MergeBase:   base,
			BehindCount: behind,
			AheadCount:  ahead,
			Conflicts:   nil,
		}, nil
	}

	// Check if --write-tree failed because of merge conflict vs unsupported flag
	if strings.Contains(combined, "CONFLICT") || strings.Contains(combined, "conflict") {
		conflicts := extractConflicts(combined)
		return &MergeSimulation{
			CanMerge:    false,
			MergeBase:   base,
			BehindCount: behind,
			AheadCount:  ahead,
			Conflicts:   conflicts,
		}, nil
	}

	// Fallback to 3-argument git merge-tree
	cmdLegacy := exec.Command("git", "merge-tree", base, targetBranch, taskBranch)
	cmdLegacy.Dir = root
	var legacyOut bytes.Buffer
	cmdLegacy.Stdout = &legacyOut
	if legacyErr := cmdLegacy.Run(); legacyErr == nil {
		legText := legacyOut.String()
		if strings.Contains(legText, "changed in both") {
			conflicts := extractLegacyConflicts(legText)
			return &MergeSimulation{
				CanMerge:    false,
				MergeBase:   base,
				BehindCount: behind,
				AheadCount:  ahead,
				Conflicts:   conflicts,
			}, nil
		}
		return &MergeSimulation{
			CanMerge:    true,
			MergeBase:   base,
			BehindCount: behind,
			AheadCount:  ahead,
			Conflicts:   nil,
		}, nil
	}

	return nil, fmt.Errorf("merge simulation failed: %s (%w)", strings.TrimSpace(combined), runErr)
}

func extractConflicts(output string) []string {
	var conflicts []string
	seen := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "CONFLICT (content): Merge conflict in ") {
			f := strings.TrimPrefix(trimmed, "CONFLICT (content): Merge conflict in ")
			f = filepath.ToSlash(strings.TrimSpace(f))
			if f != "" && !seen[f] {
				seen[f] = true
				conflicts = append(conflicts, f)
			}
		} else if strings.Contains(trimmed, "CONFLICT") && strings.Contains(trimmed, " in ") {
			parts := strings.Split(trimmed, " in ")
			if len(parts) >= 2 {
				f := filepath.ToSlash(strings.TrimSpace(parts[len(parts)-1]))
				if f != "" && !seen[f] {
					seen[f] = true
					conflicts = append(conflicts, f)
				}
			}
		}
	}
	sort.Strings(conflicts)
	return conflicts
}

func extractLegacyConflicts(output string) []string {
	var conflicts []string
	seen := make(map[string]bool)
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.Contains(line, "changed in both") && i > 0 {
			prev := strings.TrimSpace(lines[i-1])
			if prev != "" && !seen[prev] {
				seen[prev] = true
				conflicts = append(conflicts, filepath.ToSlash(prev))
			}
		}
	}
	sort.Strings(conflicts)
	return conflicts
}

// RebaseTaskBranch rebases taskBranch on top of targetBranch.
func RebaseTaskBranch(root string, taskBranch, targetBranch string) error {
	// First checkout the task branch to rebase
	origBranch, _ := runGit(root, "branch", "--show-current")

	_, err := runGit(root, "checkout", taskBranch)
	if err != nil {
		return fmt.Errorf("checking out %s: %w", taskBranch, err)
	}

	_, rebaseErr := runGit(root, "rebase", targetBranch)
	if rebaseErr != nil {
		// Clean abort
		_, _ = runGit(root, "rebase", "--abort")
		if origBranch != "" && origBranch != taskBranch {
			_, _ = runGit(root, "checkout", origBranch)
		}
		return fmt.Errorf("rebase failed onto %s (aborted cleanly): %w", targetBranch, rebaseErr)
	}

	if origBranch != "" && origBranch != taskBranch {
		_, _ = runGit(root, "checkout", origBranch)
	}
	return nil
}

// ExecuteMerge performs the actual git merge of taskBranch into targetBranch.
func ExecuteMerge(root string, taskBranch, targetBranch, commitMsg string) error {
	origBranch, _ := runGit(root, "branch", "--show-current")

	// Checkout target
	if _, err := runGit(root, "checkout", targetBranch); err != nil {
		return fmt.Errorf("checkout %s: %w", targetBranch, err)
	}

	// Merge with --no-ff
	if _, err := runGit(root, "merge", "--no-ff", "-m", commitMsg, taskBranch); err != nil {
		_, _ = runGit(root, "merge", "--abort")
		if origBranch != "" {
			_, _ = runGit(root, "checkout", origBranch)
		}
		return fmt.Errorf("merge %s into %s failed (aborted cleanly): %w", taskBranch, targetBranch, err)
	}

	return nil
}
