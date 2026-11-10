package check

import (
	"bufio"
	"regexp"
	"strings"

	"github.com/vikashkumar21/sdd/internal/roadmap"
)

var (
	reviewStatusRe   = regexp.MustCompile(`(?i)^Status:\s*([^\r\n]+)`)
	reviewReviewerRe = regexp.MustCompile(`(?i)^Reviewer:\s*([^\r\n]+)`)
)

func (c *checker) checkReviewDocument(rel string, taskStatus string) {
	reviewPath := rel + "/review.md"
	content, ok := c.load(reviewPath)
	if !ok {
		return
	}

	normTaskStatus := roadmap.Kind(taskStatus)

	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0
	var reviewStatus string
	var reviewer string
	var reviewerLine int

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// 1. Check review status header
		if reviewStatus == "" && reviewStatusRe.MatchString(trimmed) {
			m := reviewStatusRe.FindStringSubmatch(trimmed)
			if len(m) > 1 {
				raw := strings.TrimSpace(m[1])
				// Clean any template pipes: "Pending | Changes requested | Approved"
				if strings.Contains(raw, "|") {
					reviewStatus = "pending"
				} else {
					lower := strings.ToLower(raw)
					switch {
					case strings.Contains(lower, "approved"):
						reviewStatus = "approved"
					case strings.Contains(lower, "changes requested") || strings.Contains(lower, "changes"):
						reviewStatus = "changes requested"
					case strings.Contains(lower, "pending"):
						reviewStatus = "pending"
					default:
						c.add(Warn, reviewPath, lineNum, "review", "review.md status is %q (expected Pending, Changes requested, or Approved)", raw)
						reviewStatus = raw
					}
				}
			}
		}

		// Check reviewer attribution
		if reviewer == "" && reviewReviewerRe.MatchString(trimmed) {
			m := reviewReviewerRe.FindStringSubmatch(trimmed)
			if len(m) > 1 {
				reviewer = strings.TrimSpace(m[1])
				reviewerLine = lineNum
			}
		}

		// 2. Check for unresolved findings (Blockers, Warnings, Minors per Rule F-001)
		if strings.HasPrefix(trimmed, "|") && !strings.Contains(trimmed, "---") {
			parts := splitTableRow(trimmed)
			// Format: | ID | Category | Severity | Finding | Suggested Action | Status |
			if len(parts) >= 7 {
				id := strings.TrimSpace(parts[1])
				severity := strings.TrimSpace(parts[3])
				finding := strings.TrimSpace(parts[4])
				statusCol := strings.ToLower(strings.TrimSpace(parts[6]))

				// Skip placeholder, header, empty, or non-finding rows
				isPlaceholderRow := strings.EqualFold(id, "(none)") ||
					(id == "" && severity == "" && finding == "") ||
					strings.EqualFold(severity, "severity") || strings.Contains(severity, "---")

				if !isPlaceholderRow && (severity != "" || finding != "") {
					isBlocker := strings.Contains(severity, "🔴") || strings.Contains(strings.ToLower(severity), "blocker")
					isWarning := strings.Contains(severity, "🟡") || strings.Contains(strings.ToLower(severity), "warning")
					isMinor := strings.Contains(severity, "🟢") || strings.Contains(strings.ToLower(severity), "minor")

					// A finding is resolved if marked resolved, justified, waived, accepted, closed, etc.
					isResolved := statusCol == "resolved" ||
						statusCol == "justified" ||
						statusCol == "waived" ||
						statusCol == "accepted" ||
						statusCol == "acceptable" ||
						statusCol == "closed" ||
						strings.HasPrefix(statusCol, "resolved") ||
						strings.HasPrefix(statusCol, "closed") ||
						strings.HasPrefix(statusCol, "accept") ||
						strings.HasPrefix(statusCol, "justif") ||
						strings.HasPrefix(statusCol, "waive")

					// Template placeholders or open/pending/empty statuses are unresolved
					isPlaceholderFinding := strings.Contains(finding, "<critical defect") ||
						(strings.HasPrefix(finding, "<") && strings.HasSuffix(finding, ">"))
					if isPlaceholderFinding || strings.Contains(statusCol, "<") || strings.Contains(statusCol, "open") || strings.Contains(statusCol, "pending") || statusCol == "" {
						isResolved = false
					}

					if !isResolved && (normTaskStatus == "awaiting review" || normTaskStatus == "done") {
						if isBlocker {
							c.add(Error, reviewPath, lineNum, "review", "task is %s but review.md contains unresolved 🔴 Blocker finding: %s", roadmap.Normalize(taskStatus), finding)
						} else if isWarning || isMinor {
							c.add(Warn, reviewPath, lineNum, "review-findings", "task is %s but review.md contains unresolved %s finding: %s (Rule F-001 mandates resolving all findings or explicit justification)", roadmap.Normalize(taskStatus), severity, finding)
						} else {
							c.add(Warn, reviewPath, lineNum, "review-findings", "task is %s but review.md contains unresolved finding: %s (Rule F-001 mandates resolving all findings or explicit justification)", roadmap.Normalize(taskStatus), finding)
						}
					}
				}
			}
		}
	}

	// 3. If task is Awaiting review or Done in ROADMAP.md and review is Approved, verify independent reviewer
	if (normTaskStatus == "awaiting review" || normTaskStatus == "done") && reviewStatus == "approved" {
		revLower := strings.ToLower(reviewer)
		isPlaceholder := reviewer == "" ||
			strings.Contains(revLower, "<agent") ||
			strings.Contains(revLower, "[agent") ||
			strings.Contains(revLower, "______") ||
			strings.Contains(revLower, "pending") ||
			revLower == "self" ||
			revLower == "author"
		if normTaskStatus == "awaiting review" {
			isPlaceholder = isPlaceholder || strings.Contains(revLower, "self-audit") || strings.Contains(revLower, "self-approved")
		}
		if isPlaceholder {
			c.add(Warn, reviewPath, reviewerLine, "review", "task is %s with Approved review.md but reviewer attribution is missing, placeholder, or self-approved (%q); independent subagent reviewer required", roadmap.Normalize(taskStatus), reviewer)
		}
	}

	// 4. If task is Done in ROADMAP.md, review status should be Approved
	if normTaskStatus == "done" {
		if reviewStatus != "" && reviewStatus != "approved" {
			c.add(Warn, reviewPath, 0, "review", "task is Done in ROADMAP.md but review.md status is %q", reviewStatus)
		}
	}

	// 5. Verify multi-lens review triad structure in Approved reviews
	if (normTaskStatus == "awaiting review" || normTaskStatus == "done") && reviewStatus == "approved" {
		if strings.Contains(content, "Multi-Lens Ensemble Review") || strings.Contains(content, "Review Triad") {
			hasSpec := strings.Contains(content, "Spec Auditor")
			hasSec := strings.Contains(content, "AppSec Specialist") || strings.Contains(content, "Security")
			hasPat := strings.Contains(content, "Pattern & Quality Auditor") || strings.Contains(content, "Pattern Auditor")

			if !hasSpec {
				c.add(Warn, reviewPath, 0, "review-triad", "review.md is missing Spec Auditor lens section")
			}
			if !hasSec {
				c.add(Warn, reviewPath, 0, "review-triad", "review.md is missing AppSec Specialist lens section")
			}
			if !hasPat {
				c.add(Warn, reviewPath, 0, "review-triad", "review.md is missing Pattern & Quality Auditor lens section")
			}
		}
	}
}

// splitTableRow splits a table row into cells, properly ignoring escaped pipes
// and pipes occurring inside inline code spans (`...`).
func splitTableRow(line string) []string {
	var cells []string
	var current strings.Builder
	inCode := false
	codeDelimLen := 0
	escaped := false

	runes := []rune(line)
	n := len(runes)

	for i := 0; i < n; i++ {
		r := runes[i]
		if escaped {
			escaped = false
			current.WriteRune(r)
			continue
		}
		if r == '\\' {
			escaped = true
			current.WriteRune(r)
			continue
		}
		if r == '`' {
			backticks := 1
			for i+1 < n && runes[i+1] == '`' {
				backticks++
				i++
			}
			for b := 0; b < backticks; b++ {
				current.WriteRune('`')
			}
			if !inCode {
				inCode = true
				codeDelimLen = backticks
			} else if backticks == codeDelimLen {
				inCode = false
				codeDelimLen = 0
			}
			continue
		}
		if !inCode && r == '|' {
			cells = append(cells, current.String())
			current.Reset()
			continue
		}
		current.WriteRune(r)
	}
	cells = append(cells, current.String())
	return cells
}
