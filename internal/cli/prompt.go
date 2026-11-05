package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/vikashkumar21/sdd/internal/deconflict"
)

var taskIDPattern = regexp.MustCompile(`^[A-Za-z]+-\d+$`)

// ReviewPromptContext represents the structured context bundle for subagent reviews.
type ReviewPromptContext struct {
	TaskID        string   `json:"taskId"`
	TaskDir       string   `json:"taskDir"`
	Typology      string   `json:"typology"`
	Requirements  string   `json:"requirements,omitempty"`
	Plan          string   `json:"plan,omitempty"`
	Validation    string   `json:"validation,omitempty"`
	Review        string   `json:"review,omitempty"`
	ChangedFiles  []string `json:"changedFiles"`
	FeedbackRules string   `json:"feedbackRules"`
	Prompt        string   `json:"prompt"`
}

func runPromptReview(args []string, env IO) int {
	fs := newFlagSet("prompt review", env)
	dir := fs.String("dir", ".", "project root")
	jsonFlag := fs.Bool("json", false, "output machine-readable review context bundle as JSON")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}

	if len(pos) != 1 {
		fprintln(env.Err, "usage: sdd prompt review <T-xxx> [--dir <project>] [--json]")
		return 2
	}

	rawTaskID := strings.ToUpper(strings.TrimSpace(pos[0]))
	if !taskIDPattern.MatchString(rawTaskID) {
		fprintf(env.Err, "sdd prompt review: invalid task ID format %q (expected format like T-001)\n", pos[0])
		return 1
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd prompt review: %v\n", err)
		return 1
	}

	taskDirRel, typology, err := findTaskDirectory(root, rawTaskID)
	if err != nil {
		fprintf(env.Err, "sdd prompt review: %v\n", err)
		return 1
	}

	taskDirAbs := filepath.Join(root, filepath.FromSlash(taskDirRel))

	requirements := readTaskDoc(taskDirAbs, "requirements.md")
	plan := readTaskDoc(taskDirAbs, "plan.md")
	if plan == "" {
		plan = readTaskDoc(taskDirAbs, "patch.md") // hotfix typology
	}
	validation := readTaskDoc(taskDirAbs, "validation.md")
	review := readTaskDoc(taskDirAbs, "review.md")

	changedFiles := resolveChangedFiles(root, plan)
	feedbackRules := extractActiveFeedbackRules(root)

	promptText := buildReviewPrompt(rawTaskID, taskDirRel, changedFiles, feedbackRules)

	ctx := ReviewPromptContext{
		TaskID:        rawTaskID,
		TaskDir:       taskDirRel,
		Typology:      typology,
		Requirements:  requirements,
		Plan:          plan,
		Validation:    validation,
		Review:        review,
		ChangedFiles:  changedFiles,
		FeedbackRules: feedbackRules,
		Prompt:        promptText,
	}

	if *jsonFlag {
		enc := json.NewEncoder(env.Out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(ctx); err != nil {
			fprintf(env.Err, "sdd prompt review: json encode: %v\n", err)
			return 1
		}
		return 0
	}

	fprint(env.Out, promptText)
	return 0
}

func findTaskDirectory(root, taskID string) (relDir, typology string, err error) {
	typologies := []string{"features", "bugfixes", "hotfixes", "spikes"}
	prefix := taskID + "-"

	for _, topo := range typologies {
		base := filepath.Join(root, topo)
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), "_") {
				continue
			}
			if e.Name() == taskID || strings.HasPrefix(e.Name(), prefix) {
				return filepath.ToSlash(filepath.Join(topo, e.Name())), topo, nil
			}
		}
	}

	return "", "", fmt.Errorf("task %s folder not found across features/, bugfixes/, hotfixes/, or spikes/", taskID)
}

func readTaskDoc(taskDir, filename string) string {
	b, err := os.ReadFile(filepath.Join(taskDir, filename))
	if err != nil {
		return ""
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

func resolveChangedFiles(root, planContent string) []string {
	fileMap := make(map[string]bool)

	// 1. Attempt git diff against main or HEAD
	if isGitRepo(root) {
		cmd := osexec.Command("git", "diff", "--name-only", "main...HEAD")
		cmd.Dir = root
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil && out.Len() > 0 {
			scanner := bufio.NewScanner(&out)
			for scanner.Scan() {
				f := strings.TrimSpace(scanner.Text())
				if f != "" {
					fileMap[filepath.ToSlash(f)] = true
				}
			}
		}

		// Also check unstaged/staged working tree changes
		statusCmd := osexec.Command("git", "status", "--porcelain")
		statusCmd.Dir = root
		var statusOut bytes.Buffer
		statusCmd.Stdout = &statusOut
		if err := statusCmd.Run(); err == nil && statusOut.Len() > 0 {
			scanner := bufio.NewScanner(&statusOut)
			for scanner.Scan() {
				line := scanner.Text()
				if len(line) >= 4 {
					f := strings.TrimSpace(line[3:])
					// Handle rename "old -> new"
					if idx := strings.Index(f, "->"); idx != -1 {
						f = strings.TrimSpace(f[idx+2:])
					}
					if f != "" {
						fileMap[filepath.ToSlash(f)] = true
					}
				}
			}
		}
	}

	// 2. If no git files found or as fallback, extract from plan
	if len(fileMap) == 0 && planContent != "" {
		for _, f := range deconflict.ExtractPlanFiles(planContent) {
			fileMap[f] = true
		}
	}

	var files []string
	for f := range fileMap {
		files = append(files, f)
	}
	sort.Strings(files)
	return files
}

func isGitRepo(root string) bool {
	gitDir := filepath.Join(root, ".git")
	fi, err := os.Stat(gitDir)
	return err == nil && (fi.IsDir() || !fi.IsDir()) // .git can be a file in worktrees
}

func extractActiveFeedbackRules(root string) string {
	content, err := os.ReadFile(filepath.Join(root, "SKILLS.md"))
	if err != nil {
		return "(none active)"
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	inSection8 := false
	inRegister := false
	var ruleLines []string

	section8Header := regexp.MustCompile(`^##\s+8\.\s+Human feedback rules`)
	nextSectionHeader := regexp.MustCompile(`^##\s+\d+\.`)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if section8Header.MatchString(trimmed) {
			inSection8 = true
			continue
		}

		if inSection8 && nextSectionHeader.MatchString(trimmed) {
			break
		}

		if inSection8 {
			if strings.HasPrefix(trimmed, "### Register") {
				inRegister = true
				continue
			}
			if inRegister && strings.HasPrefix(trimmed, "|") {
				if strings.Contains(trimmed, "---") || strings.Contains(strings.ToLower(trimmed), "applies to") {
					continue
				}
				if strings.Contains(trimmed, "(none yet)") {
					continue
				}
				parts := strings.Split(trimmed, "|")
				if len(parts) >= 4 {
					id := strings.TrimSpace(parts[1])
					rule := strings.TrimSpace(parts[3])
					if id != "" && rule != "" {
						ruleLines = append(ruleLines, fmt.Sprintf("- **%s:** %s", id, rule))
					}
				}
			}
		}
	}

	if len(ruleLines) == 0 {
		return "(none active)"
	}
	return strings.Join(ruleLines, "\n")
}

func buildReviewPrompt(taskID, taskDir string, files []string, feedbackRules string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Handover: Independent Subagent Code & Spec Review for %s\n\n", taskID)
	fmt.Fprintf(&sb, "You are acting as an **Independent Subagent Reviewer** for task `%s`.\n", taskID)
	sb.WriteString("Your mandate is to impartially audit the implementation before GATE 2 human review.\n")
	sb.WriteString("**You are strictly independent; the author agent cannot review or approve its own work.**\n\n")

	sb.WriteString("## SDD Anthem & Principles\n")
	sb.WriteString("> **ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**\n\n")

	sb.WriteString("## Review Rubric (3-Tier Severity)\n")
	sb.WriteString("- 🔴 **Blocker (Developer Attention Needed):** Spec violation (`requirements.md` not met), plan drift (files touched outside `plan.md`), security hazard (secrets/unvalidated inputs), test gaps, unapproved dependency. MUST BE RESOLVED before commit.\n")
	sb.WriteString("- 🟡 **Warning (Attention Suggested):** Sub-optimal pattern, missing doc or comment, edge case risk.\n")
	sb.WriteString("- 🟢 **Minor (Safe / Informational):** Style, minor naming or non-critical improvement.\n\n")

	sb.WriteString("## Task Details\n")
	fmt.Fprintf(&sb, "- **Task ID:** %s\n", taskID)
	fmt.Fprintf(&sb, "- **Task Directory:** %s\n", taskDir)
	sb.WriteString("- **Changed Files:**\n")
	if len(files) == 0 {
		sb.WriteString("  - (none resolved from git diff or plan)\n")
	} else {
		for _, f := range files {
			fmt.Fprintf(&sb, "  - `%s`\n", f)
		}
	}
	sb.WriteString("\n")

	sb.WriteString("## Active Feedback Rules (SKILLS.md §8)\n")
	sb.WriteString(feedbackRules)
	sb.WriteString("\n\n")

	sb.WriteString("## Review Instructions\n")
	fmt.Fprintf(&sb, "1. Inspect the git diff and changed files against `%s/requirements.md` and `%s/plan.md`.\n", taskDir, taskDir)
	fmt.Fprintf(&sb, "2. Verify that every requirement has concrete test evidence in `%s/validation.md`.\n", taskDir)
	sb.WriteString("3. Verify that zero files were modified outside the approved plan.\n")
	fmt.Fprintf(&sb, "4. Record all findings and sign-off in `%s/review.md`.\n", taskDir)
	sb.WriteString("   - If findings include any 🔴 Blocker, set status to `Changes requested` with specific remediation steps.\n")
	sb.WriteString("   - If 0 Blockers remain, set status to `Approved` with your reviewer attribution.\n")

	return sb.String()
}
