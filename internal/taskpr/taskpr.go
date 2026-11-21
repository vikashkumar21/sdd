// Package taskpr synthesizes task quartet documents across all work typologies
// into standardized Pull Request titles, Markdown descriptions, and machine-readable
// JSON bundles, enforcing pre-PR review blocker qualification.
package taskpr

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vikashkumar21/sdd/internal/deconflict"
	"github.com/vikashkumar21/sdd/internal/state"
	"github.com/vikashkumar21/sdd/internal/worktree"
)

var (
	h1TitleRe      = regexp.MustCompile(`(?i)^#\s+(?:Requirements|Plan|Validation|Review|Reproduction|RCA|Incident|Patch|Hypotheses|Findings)?\s*[-—:]*\s*(?:T-\d{3})?\s*[-—:]*\s*(.+)$`)
	reviewStatusRe = regexp.MustCompile(`(?i)^Status:\s*([^\r\n]+)`)
)

// PROptions configures task PR packaging.
type PROptions struct {
	TaskID          string
	ProjectDir      string
	AllowUnreviewed bool
	BaseBranch      string
	HeadBranch      string
}

// PRBundle contains all synthesized pull request metadata and contents.
type PRBundle struct {
	TaskID         string                `json:"task_id"`
	Typology       string                `json:"typology"`
	Title          string                `json:"title"`
	Summary        string                `json:"summary"`
	Body           string                `json:"body"`
	BaseBranch     string                `json:"base_branch"`
	HeadBranch     string                `json:"head_branch"`
	ChangedFiles   []string              `json:"changed_files"`
	Tokens         *state.TokenTelemetry `json:"tokens,omitempty"`
	ReviewStatus   string                `json:"review_status"`
	ReviewBlockers int                   `json:"review_blockers"`
	QualifiesForPR bool                  `json:"qualifies_for_pr"`
	Warnings       []string              `json:"warnings,omitempty"`
}

// PackageTaskPR reads task documents in root and packages a structured PRBundle.
func PackageTaskPR(root string, opts PROptions) (*PRBundle, error) {
	if strings.TrimSpace(opts.TaskID) == "" {
		return nil, fmt.Errorf("task ID is required")
	}

	loc, err := worktree.FindTask(root, opts.TaskID)
	if err != nil {
		return nil, fmt.Errorf("task %s not found in tasks/ or roadmap: %w", opts.TaskID, err)
	}

	bundle := &PRBundle{
		TaskID:   loc.TaskID,
		Typology: loc.Typology,
	}

	// Base & Head branches (sanitized against control chars, newlines, and markdown injection)
	sanitizeBranch := func(s string) string {
		s = strings.ReplaceAll(s, "\r", "")
		s = strings.ReplaceAll(s, "\n", "")
		s = strings.ReplaceAll(s, "`", "")
		s = strings.ReplaceAll(s, "|", "")
		return strings.TrimSpace(s)
	}

	bundle.BaseBranch = sanitizeBranch(opts.BaseBranch)
	if bundle.BaseBranch == "" {
		bundle.BaseBranch = "main"
	}
	bundle.HeadBranch = sanitizeBranch(opts.HeadBranch)
	if bundle.HeadBranch == "" {
		bundle.HeadBranch = fmt.Sprintf("%s/%s-%s", loc.Typology, loc.TaskID, loc.Slug)
	}

	taskDir := filepath.Join(root, filepath.FromSlash(loc.Folder))

	// 1. Title and Summary extraction across typologies
	rawTitle, summary := extractTitleAndSummary(taskDir, loc.Typology)
	if rawTitle == "" {
		rawTitle = strings.ReplaceAll(loc.Slug, "-", " ")
		if len(rawTitle) > 0 {
			rawTitle = strings.ToUpper(rawTitle[:1]) + rawTitle[1:]
		}
	}
	bundle.Summary = summary

	// Format standardized PR Title
	bundle.Title = formatPRTitle(loc.Typology, loc.TaskID, rawTitle)

	// 2. Extract changed files from plan or patch
	planContent := readFileOrEmpty(taskDir, "plan.md")
	if planContent == "" && loc.Typology == "hotfix" {
		planContent = readFileOrEmpty(taskDir, "patch.md")
	}
	if planContent != "" {
		bundle.ChangedFiles = deconflict.ExtractPlanFiles(planContent)
	}

	// 3. Validation evidence extraction
	valContent := readFileOrEmpty(taskDir, "validation.md")
	valEvidence := extractValidationEvidence(valContent)

	// 4. Review Triad qualification audit
	auditReview(taskDir, loc.Typology, opts.AllowUnreviewed, bundle)

	// 5. Token telemetry extraction from task state store
	if st, exists, err := state.ReadState(root, bundle.TaskID); err == nil && exists {
		if toks, ok := state.ExtractTokens(st); ok && toks.TotalTokens > 0 {
			bundle.Tokens = &toks
		}
	}

	// 6. Render PR Markdown Body
	bundle.Body = FormatPRMarkdown(bundle, valEvidence)

	return bundle, nil
}

func formatPRTitle(typology, taskID, rawTitle string) string {
	var prefix string
	switch strings.ToLower(typology) {
	case "feature":
		prefix = "feat"
	case "bugfix":
		prefix = "fix"
	case "hotfix":
		prefix = "hotfix"
	case "spike":
		prefix = "spike"
	default:
		prefix = "chore"
	}
	return fmt.Sprintf("%s(%s): %s", prefix, taskID, rawTitle)
}

func extractTitleAndSummary(taskDir, typology string) (string, string) {
	// Candidate files in preference order
	var candidates []string
	switch typology {
	case "bugfix":
		candidates = []string{"requirements.md", "reproduction.md", "rca.md"}
	case "hotfix":
		candidates = []string{"requirements.md", "incident.md", "patch.md"}
	case "spike":
		candidates = []string{"requirements.md", "hypotheses.md", "findings.md"}
	default:
		candidates = []string{"requirements.md"}
	}

	var content string
	for _, c := range candidates {
		text := readFileOrEmpty(taskDir, c)
		if text != "" {
			content = text
			break
		}
	}
	if content == "" {
		return "", ""
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	var title string
	var summaryLines []string
	inGoalSection := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line == "---" {
			continue
		}

		if title == "" && strings.HasPrefix(line, "# ") {
			m := h1TitleRe.FindStringSubmatch(line)
			if len(m) > 1 {
				title = strings.TrimSpace(m[1])
			} else {
				title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			}
			continue
		}

		if strings.HasPrefix(line, "## 1. Goal") || strings.HasPrefix(line, "## 1. Summary") ||
			strings.HasPrefix(line, "## 1. Defect") || strings.HasPrefix(line, "## 1. Incident") ||
			strings.HasPrefix(line, "## 1. Hypotheses") {
			inGoalSection = true
			continue
		}

		if inGoalSection {
			if strings.HasPrefix(line, "## ") {
				break
			}
			if !strings.HasPrefix(line, ">") {
				summaryLines = append(summaryLines, line)
			}
		}
	}
	_ = scanner.Err()

	summary := strings.TrimSpace(strings.Join(summaryLines, " "))
	return title, summary
}

func extractValidationEvidence(content string) string {
	if content == "" {
		return "No validation document found."
	}
	scanner := bufio.NewScanner(strings.NewReader(content))
	var matrixLines []string
	inMatrix := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "## 1. Traceability matrix") {
			inMatrix = true
			continue
		}

		if inMatrix {
			if strings.HasPrefix(trimmed, "## ") {
				break
			}
			if strings.HasPrefix(trimmed, "|") {
				matrixLines = append(matrixLines, line)
			}
		}
	}
	_ = scanner.Err()

	if len(matrixLines) > 0 {
		return strings.Join(matrixLines, "\n")
	}
	return "Validation evidence recorded in validation.md."
}

func auditReview(taskDir, typology string, allowUnreviewed bool, bundle *PRBundle) {
	// Spikes are non-merging prototypes; review is N/A
	if typology == "spike" {
		bundle.ReviewStatus = "N/A (spike prototype)"
		bundle.ReviewBlockers = 0
		bundle.QualifiesForPR = true
		return
	}

	reviewPath := filepath.Join(taskDir, "review.md")
	data, err := os.ReadFile(reviewPath)
	if err != nil {
		if typology == "hotfix" {
			// Hotfixes may use expedited approvals in patch.md
			bundle.ReviewStatus = "expedited (patch.md)"
			bundle.ReviewBlockers = 0
			bundle.QualifiesForPR = true
			return
		}

		bundle.ReviewStatus = "missing"
		bundle.ReviewBlockers = 0
		bundle.QualifiesForPR = false
		bundle.Warnings = append(bundle.Warnings, "review.md not found in task directory")
		if allowUnreviewed {
			bundle.QualifiesForPR = true
		}
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var status string
	blockers := 0

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if status == "" && reviewStatusRe.MatchString(trimmed) {
			m := reviewStatusRe.FindStringSubmatch(trimmed)
			if len(m) > 1 {
				status = strings.TrimSpace(m[1])
			}
		}

		if strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") {
			parts := strings.Split(trimmed, "|")
			// Table row format: | ID | Category | Severity | Finding | Suggested Action | Status |
			if len(parts) >= 7 {
				severity := strings.TrimSpace(parts[3])
				statusCol := strings.ToLower(strings.TrimSpace(parts[6]))

				isBlocker := strings.Contains(severity, "🔴") || strings.Contains(strings.ToLower(severity), "blocker")
				if isBlocker {
					isResolved := statusCol == "resolved" ||
						statusCol == "closed" ||
						statusCol == "justified" ||
						statusCol == "waived" ||
						statusCol == "accepted" ||
						statusCol == "acceptable" ||
						strings.HasPrefix(statusCol, "resolved") ||
						strings.HasPrefix(statusCol, "closed") ||
						strings.HasPrefix(statusCol, "justif") ||
						strings.HasPrefix(statusCol, "waive") ||
						strings.HasPrefix(statusCol, "accept")

					// Template placeholders or open/pending/empty statuses are unresolved
					if strings.Contains(statusCol, "open") ||
						strings.Contains(statusCol, "pending") ||
						strings.Contains(statusCol, "<") ||
						statusCol == "" {
						isResolved = false
					}

					if !isResolved {
						blockers++
					}
				}
			}
		}
	}
	_ = scanner.Err()

	if status == "" {
		status = "unknown"
	}

	bundle.ReviewStatus = status
	bundle.ReviewBlockers = blockers

	qualifies := true
	if blockers > 0 {
		qualifies = false
		bundle.Warnings = append(bundle.Warnings, fmt.Sprintf("%d unresolved 🔴 Blocker(s) found in review.md", blockers))
	}

	if !strings.EqualFold(status, "Approved") {
		qualifies = false
		bundle.Warnings = append(bundle.Warnings, fmt.Sprintf("review status is %q (expected Approved)", status))
	}

	bundle.QualifiesForPR = qualifies
	if allowUnreviewed {
		bundle.QualifiesForPR = true
	}
}

// FormatPRMarkdown formats the PRBundle into a structured GitHub/GitLab Pull Request Markdown description.
func FormatPRMarkdown(bundle *PRBundle, valEvidence string) string {
	var sb strings.Builder

	sb.WriteString("## Summary\n")
	if bundle.Summary != "" {
		sb.WriteString(bundle.Summary)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("Automated Pull Request packaged from task specification.\n\n")
	}

	sb.WriteString("## Work Typology & Branches\n")
	fmt.Fprintf(&sb, "- **Task ID:** `%s`\n", bundle.TaskID)
	fmt.Fprintf(&sb, "- **Typology:** `%s`\n", bundle.Typology)
	fmt.Fprintf(&sb, "- **Base Branch:** `%s`\n", bundle.BaseBranch)
	fmt.Fprintf(&sb, "- **Head Branch:** `%s`\n\n", bundle.HeadBranch)

	sb.WriteString("## Planned Changes\n")
	if len(bundle.ChangedFiles) > 0 {
		fmt.Fprintf(&sb, "%d planned file(s) touched:\n", len(bundle.ChangedFiles))
		for _, f := range bundle.ChangedFiles {
			fmt.Fprintf(&sb, "- `%s`\n", f)
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("See commit diff for changed files.\n\n")
	}

	sb.WriteString("## Verification & Traceability\n")
	if valEvidence != "" {
		sb.WriteString(valEvidence)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("Verified per task validation matrix.\n\n")
	}

	if bundle.Tokens != nil && bundle.Tokens.TotalTokens > 0 {
		sb.WriteString("## Token Telemetry\n")
		fmt.Fprintf(&sb, "- **Input Tokens:** %d\n", bundle.Tokens.InputTokens)
		fmt.Fprintf(&sb, "- **Cache Read Tokens:** %d\n", bundle.Tokens.CacheReadTokens)
		fmt.Fprintf(&sb, "- **Output Tokens:** %d\n", bundle.Tokens.OutputTokens)
		fmt.Fprintf(&sb, "- **Total Task Tokens:** %d\n\n", bundle.Tokens.TotalTokens)
	}

	sb.WriteString("## Review Triad Audit Certification\n")
	fmt.Fprintf(&sb, "- **Review Status:** `%s`\n", bundle.ReviewStatus)
	fmt.Fprintf(&sb, "- **Unresolved Blockers:** `%d`\n", bundle.ReviewBlockers)
	if bundle.QualifiesForPR {
		if len(bundle.Warnings) > 0 {
			sb.WriteString("- **PR Qualification:** `QUALIFIED (Bypassed via --allow-unreviewed)`\n")
			for _, w := range bundle.Warnings {
				fmt.Fprintf(&sb, "  - ⚠️ %s\n", w)
			}
		} else {
			sb.WriteString("- **PR Qualification:** `QUALIFIED` (0 Blockers, Rule F-001 satisfied)\n")
		}
	} else {
		sb.WriteString("- **PR Qualification:** `BLOCKED`\n")
		for _, w := range bundle.Warnings {
			fmt.Fprintf(&sb, "  - ⚠️ %s\n", w)
		}
	}

	return sb.String()
}

func readFileOrEmpty(dir, name string) string {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return ""
	}
	return string(data)
}
