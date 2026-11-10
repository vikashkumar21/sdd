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
	Lens          string   `json:"lens"`
	LensFocus     []string `json:"lensFocus,omitempty"`
	Requirements  string   `json:"requirements,omitempty"`
	Plan          string   `json:"plan,omitempty"`
	Validation    string   `json:"validation,omitempty"`
	Review        string   `json:"review,omitempty"`
	ChangedFiles  []string `json:"changedFiles"`
	FeedbackRules string   `json:"feedbackRules"`
	Prompt        string   `json:"prompt"`
}

// PlanReviewPromptContext represents the structured context bundle for pre-GATE 1 plan reviews.
type PlanReviewPromptContext struct {
	TaskID        string   `json:"taskId"`
	TaskDir       string   `json:"taskDir"`
	Typology      string   `json:"typology"`
	Requirements  string   `json:"requirements,omitempty"`
	Plan          string   `json:"plan,omitempty"`
	Validation    string   `json:"validation,omitempty"`
	PlannedFiles  []string `json:"plannedFiles"`
	FeedbackRules string   `json:"feedbackRules"`
	Decisions     []string `json:"decisions"`
	Rubric        []string `json:"rubric"`
	Prompt        string   `json:"prompt"`
}

func runPromptPlanReview(args []string, env IO) int {
	fs := newFlagSet("prompt plan-review", env)
	dir := fs.String("dir", ".", "project root")
	jsonFlag := fs.Bool("json", false, "output machine-readable plan review context bundle as JSON")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}

	if len(pos) != 1 {
		fprintln(env.Err, "usage: sdd prompt plan-review <T-xxx> [--dir <project>] [--json]")
		return 2
	}

	rawTaskID := strings.ToUpper(strings.TrimSpace(pos[0]))
	if !taskIDPattern.MatchString(rawTaskID) {
		fprintf(env.Err, "sdd prompt plan-review: invalid task ID format %q (expected format like T-001)\n", pos[0])
		return 1
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fprintf(env.Err, "sdd prompt plan-review: %v\n", err)
		return 1
	}

	taskDirRel, typology, err := findTaskDirectory(root, rawTaskID)
	if err != nil {
		fprintf(env.Err, "sdd prompt plan-review: %v\n", err)
		return 1
	}

	taskDirAbs := filepath.Join(root, filepath.FromSlash(taskDirRel))

	requirements := readTaskDoc(taskDirAbs, "requirements.md")
	plan := readTaskDoc(taskDirAbs, "plan.md")
	if plan == "" {
		plan = readTaskDoc(taskDirAbs, "patch.md")
	}
	validation := readTaskDoc(taskDirAbs, "validation.md")

	plannedFiles := deconflict.ExtractPlanFiles(plan)
	feedbackRules := extractActiveFeedbackRules(root)
	decisions := extractApprovedDecisions(root)

	rubric := getPlanReviewRubric()
	promptText := buildPlanReviewPrompt(rawTaskID, taskDirRel, typology, plannedFiles, feedbackRules, decisions, rubric)

	ctx := PlanReviewPromptContext{
		TaskID:        rawTaskID,
		TaskDir:       taskDirRel,
		Typology:      typology,
		Requirements:  requirements,
		Plan:          plan,
		Validation:    validation,
		PlannedFiles:  plannedFiles,
		FeedbackRules: feedbackRules,
		Decisions:     decisions,
		Rubric:        rubric,
		Prompt:        promptText,
	}

	if *jsonFlag {
		enc := json.NewEncoder(env.Out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(ctx); err != nil {
			fprintf(env.Err, "sdd prompt plan-review: json encode: %v\n", err)
			return 1
		}
		return 0
	}

	fprint(env.Out, promptText)
	return 0
}

func getPlanReviewRubric() []string {
	return []string{
		"Spec Completeness: atomic R-xxx requirements with clear verification methods; explicit non-goals; empty 'Open questions' section.",
		"Plan Soundness: steps small enough to review in one sitting; each step names exact files and proving check.",
		"Boundary Precision: Section 5 files list is exhaustive; zero unapproved file modifications or dependencies (D-007).",
		"Risk & Rollback: failure modes, concurrency constraints, and clean rollback paths identified.",
		"Testability: validation matrix maps every requirement to concrete test commands and expected outcomes.",
	}
}

func extractApprovedDecisions(root string) []string {
	content, err := os.ReadFile(filepath.Join(root, "TECH_STACK.md"))
	if err != nil {
		return nil
	}
	var decisions []string
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "| D-") && strings.Contains(line, "Approved") {
			parts := strings.Split(line, "|")
			if len(parts) >= 3 {
				id := strings.TrimSpace(parts[1])
				title := strings.TrimSpace(parts[2])
				decisions = append(decisions, fmt.Sprintf("%s: %s", id, title))
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil
	}
	return decisions
}

func buildPlanReviewPrompt(taskID, taskDir, typology string, plannedFiles []string, feedbackRules string, decisions, rubric []string) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# Handover: Independent Pre-GATE 1 Plan Review for %s\n\n", taskID)
	fmt.Fprintf(&sb, "You are acting as an **Independent Plan Reviewer** for task `%s` (%s).\n", taskID, typology)
	sb.WriteString("Your mandate is to impartially audit the Phase A specification and implementation plan **before** GATE 1 presentation to the human.\n")
	sb.WriteString("**You are strictly independent; the author agent cannot review or approve its own plan.**\n\n")

	sb.WriteString("## SDD Anthem & Principles\n")
	sb.WriteString("> **ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**\n\n")

	sb.WriteString("## Plan Review Rubric\n")
	for _, item := range rubric {
		fmt.Fprintf(&sb, "- [ ] %s\n", item)
	}
	sb.WriteString("\n")

	sb.WriteString("## Task Details\n")
	fmt.Fprintf(&sb, "- **Task ID:** %s\n", taskID)
	fmt.Fprintf(&sb, "- **Task Directory:** %s\n", taskDir)
	fmt.Fprintf(&sb, "- **Typology:** %s\n", typology)
	sb.WriteString("- **Planned Files to Touch:**\n")
	if len(plannedFiles) == 0 {
		sb.WriteString("  - (none resolved from plan)\n")
	} else {
		for _, f := range plannedFiles {
			fmt.Fprintf(&sb, "  - `%s`\n", f)
		}
	}
	sb.WriteString("\n")

	if len(decisions) > 0 {
		sb.WriteString("## Approved Architectural Decisions (TECH_STACK.md)\n")
		for _, d := range decisions {
			fmt.Fprintf(&sb, "- %s\n", d)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Active Feedback Rules (SKILLS.md §8)\n")
	sb.WriteString(feedbackRules)
	sb.WriteString("\n\n")

	sb.WriteString("## Plan Review Instructions\n")
	fmt.Fprintf(&sb, "1. Read `%s/requirements.md`, `%s/plan.md`, and `%s/validation.md`.\n", taskDir, taskDir, taskDir)
	sb.WriteString("2. Audit against the **Plan Review Rubric** above.\n")
	sb.WriteString("3. Report any 🔴 Blocker (missing step, boundary hole, untestable assertion, unapproved dependency) or 🟡 Warning.\n")
	sb.WriteString("4. Verify that the `Open questions` section in `requirements.md` is empty.\n")
	sb.WriteString("5. Provide an explicit sign-off recommendation: `[Approved / Changes Requested]` with actionable remediation notes.\n")

	return sb.String()
}

func runPromptReview(args []string, env IO) int {
	fs := newFlagSet("prompt review", env)
	dir := fs.String("dir", ".", "project root")
	jsonFlag := fs.Bool("json", false, "output machine-readable review context bundle as JSON")
	lensFlag := fs.String("lens", "all", "review lens: spec, security (or appsec), pattern, or all")

	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return parseExit(err)
	}

	if len(pos) != 1 {
		fprintln(env.Err, "usage: sdd prompt review <T-xxx> [--lens <spec|security|pattern|all>] [--dir <project>] [--json]")
		return 2
	}

	normLens := strings.ToLower(strings.TrimSpace(*lensFlag))
	if normLens == "appsec" {
		normLens = "security"
	}
	switch normLens {
	case "spec", "security", "pattern", "all":
		// valid
	default:
		fprintf(env.Err, "sdd prompt review: invalid --lens %q (expected: spec, security, pattern, or all)\n", *lensFlag)
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

	lensFocus := getLensFocusItems(normLens)
	promptText := buildReviewPromptWithLens(rawTaskID, taskDirRel, normLens, changedFiles, feedbackRules, lensFocus)

	ctx := ReviewPromptContext{
		TaskID:        rawTaskID,
		TaskDir:       taskDirRel,
		Typology:      typology,
		Lens:          normLens,
		LensFocus:     lensFocus,
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

func getLensFocusItems(lens string) []string {
	specItems := []string{
		"Traceability: every requirement (R-xxx) maps to verified code and concrete test evidence in validation.md.",
		"Plan boundaries: touch ONLY files listed in plan.md Section 5 (zero scope creep).",
		"Negative and failure cases are explicitly tested and verified.",
		"Standard library purity: zero unapproved third-party dependencies (D-007).",
	}
	secItems := []string{
		"Input validation & bound sizes on all external inputs; fail safe on ambiguity.",
		"Filesystem confinement: reject directory traversal, enforce path bounds via os.Root or clean canonical paths.",
		"Secrets & privacy hygiene: zero credentials, tokens, connection strings, or personal data; synthetic test fixtures only.",
		"Command execution safety: never concatenate shell strings; use separate arguments.",
	}
	patItems := []string{
		"Architectural decision conformance: strictly respect approved decisions in TECH_STACK.md (D-xxx).",
		"Determinism: inject clocks, random sources, and ID generators; no map iteration order dependency.",
		"Code idioms: follow Effective Go, Go Code Review Comments, context.Context first, %w error wrapping.",
		"Continuous learning: verify compliance with active feedback rules in SKILLS.md §8 (F-xxx).",
		"Concurrency safety: verify goroutine lifecycles, race detector compliance, and no data races.",
	}

	switch lens {
	case "spec":
		return specItems
	case "security":
		return secItems
	case "pattern":
		return patItems
	default:
		var all []string
		all = append(all, specItems...)
		all = append(all, secItems...)
		all = append(all, patItems...)
		return all
	}
}

func findTaskDirectory(root, taskID string) (relDir, typology string, err error) {
	typologies := []struct {
		dir  string
		name string
	}{
		{filepath.Join("tasks", "features"), "features"},
		{filepath.Join("tasks", "bugfixes"), "bugfixes"},
		{filepath.Join("tasks", "hotfixes"), "hotfixes"},
		{filepath.Join("tasks", "spikes"), "spikes"},
		{"features", "features"},
		{"bugfixes", "bugfixes"},
		{"hotfixes", "hotfixes"},
		{"spikes", "spikes"},
	}
	prefix := taskID + "-"

	for _, topo := range typologies {
		base := filepath.Join(root, topo.dir)
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), "_") {
				continue
			}
			if e.Name() == taskID || strings.HasPrefix(e.Name(), prefix) {
				return filepath.ToSlash(filepath.Join(topo.dir, e.Name())), topo.name, nil
			}
		}
	}

	return "", "", fmt.Errorf("task %s folder not found across tasks/ (or legacy features/, bugfixes/, hotfixes/, spikes/)", taskID)
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

func buildReviewPromptWithLens(taskID, taskDir, lens string, files []string, feedbackRules string, focusItems []string) string {
	var sb strings.Builder

	lensTitle := "Ensemble Review Triad"
	roleDesc := "across all three pillars (Spec Auditor, AppSec Specialist, Pattern Auditor)"
	switch lens {
	case "spec":
		lensTitle = "Spec Auditor Lens"
		roleDesc = "with a dedicated focus on requirement traceability, negative test cases, and plan boundaries"
	case "security":
		lensTitle = "AppSec Specialist Lens"
		roleDesc = "with a dedicated focus on threat modeling, input validation, path confinement, and secret hygiene"
	case "pattern":
		lensTitle = "Pattern & Quality Auditor Lens"
		roleDesc = "with a dedicated focus on architectural decision conformance, determinism, language idioms, and feedback rules"
	}

	fmt.Fprintf(&sb, "# Handover: Independent Subagent Code & Spec Review for %s (%s)\n\n", taskID, lensTitle)
	fmt.Fprintf(&sb, "You are acting as an **Independent Subagent Reviewer (%s)** for task `%s`.\n", lensTitle, taskID)
	fmt.Fprintf(&sb, "Your mandate is to impartially audit the implementation %s before GATE 2 human review.\n", roleDesc)
	sb.WriteString("**You are strictly independent; the author agent cannot review or approve its own work.**\n\n")

	sb.WriteString("## SDD Anthem & Principles\n")
	sb.WriteString("> **ALWAYS BE DETERMINISTIC. When a decision is needed, STOP and ask the human. Never guess. Never silently choose.**\n\n")

	sb.WriteString("## Review Rubric (3-Tier Severity)\n")
	sb.WriteString("- 🔴 **Blocker (Developer Attention Needed):** Spec violation (`requirements.md` not met), plan drift (files touched outside `plan.md`), security hazard (secrets/unvalidated inputs), test gaps, unapproved dependency. MUST BE RESOLVED before commit.\n")
	sb.WriteString("- 🟡 **Warning (Attention Suggested):** Sub-optimal pattern, edge case risk, missing doc or comment.\n")
	sb.WriteString("- 🟢 **Minor (Safe / Informational):** Style, minor naming or non-critical improvement.\n\n")

	sb.WriteString("## Lens Focus & Verification Questions\n")
	for _, item := range focusItems {
		fmt.Fprintf(&sb, "- [ ] %s\n", item)
	}
	sb.WriteString("\n")

	sb.WriteString("## Task Details\n")
	fmt.Fprintf(&sb, "- **Task ID:** %s\n", taskID)
	fmt.Fprintf(&sb, "- **Task Directory:** %s\n", taskDir)
	fmt.Fprintf(&sb, "- **Review Lens:** %s\n", lens)
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

	if lens != "all" {
		sb.WriteString("## Parallel Review Triad Protocol\n")
		fmt.Fprintf(&sb, "You are one of three parallel subagents in the **3-Subagent Parallel Review Triad** (Spec Auditor, AppSec Specialist, Pattern Auditor).\n")
		fmt.Fprintf(&sb, "Your evaluation focus is EXCLUSIVELY the %s. Record your findings under the `%s` section in `%s/review.md`.\n", lensTitle, lensTitle, taskDir)
		sb.WriteString("If multiple subagents report findings, the primary agent will remediate all findings together in a single batched fix commit (consuming 1 loop in the 3-loop circuit breaker).\n\n")
	}

	sb.WriteString("## Review Instructions\n")
	fmt.Fprintf(&sb, "1. Inspect the git diff and changed files against `%s/requirements.md` and `%s/plan.md`.\n", taskDir, taskDir)
	fmt.Fprintf(&sb, "2. Verify each point listed under **Lens Focus & Verification Questions**.\n")
	fmt.Fprintf(&sb, "3. Record all findings and sign-off in `%s/review.md`.\n", taskDir)
	sb.WriteString("   - If findings include any 🔴 Blocker, set status to `Changes requested` with specific remediation steps.\n")
	sb.WriteString("   - If 0 Blockers remain, set status to `Approved` with your reviewer attribution (including your lens name).\n")

	return sb.String()
}
