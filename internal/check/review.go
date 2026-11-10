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

		// 2. Check for unresolved 🔴 Blocker findings
		if strings.Contains(line, "🔴") || strings.Contains(strings.ToLower(line), "blocker") {
			// Check if this is a table row in the findings register
			if strings.HasPrefix(trimmed, "|") && !strings.Contains(trimmed, "Severity") && !strings.Contains(trimmed, "---") {
				parts := strings.Split(trimmed, "|")
				// Format: | ID | Category | Severity | Finding | Suggested Action | Status |
				if len(parts) >= 6 {
					severity := strings.TrimSpace(parts[3])
					finding := strings.TrimSpace(parts[4])
					statusCol := ""
					if len(parts) >= 7 {
						statusCol = strings.ToLower(strings.TrimSpace(parts[6]))
					}

					isBlocker := strings.Contains(severity, "🔴") || strings.Contains(strings.ToLower(severity), "blocker")
					isUnresolved := statusCol == "open" || statusCol == "pending" || statusCol == "" || strings.Contains(finding, "<critical defect")

					if isBlocker && isUnresolved {
						if normTaskStatus == "awaiting review" || normTaskStatus == "done" {
							c.add(Error, reviewPath, lineNum, "review", "task is %s but review.md contains unresolved 🔴 Blocker finding: %s", roadmap.Normalize(taskStatus), finding)
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
