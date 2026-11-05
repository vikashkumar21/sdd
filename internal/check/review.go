package check

import (
	"bufio"
	"regexp"
	"strings"

	"github.com/vikashkumar21/sdd/internal/roadmap"
)

var (
	reviewStatusRe = regexp.MustCompile(`(?i)^Status:\s*([^\r\n]+)`)
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

	// 3. If task is Done in ROADMAP.md, review status should be Approved
	if normTaskStatus == "done" {
		if reviewStatus != "" && reviewStatus != "approved" {
			c.add(Warn, reviewPath, 0, "review", "task is Done in ROADMAP.md but review.md status is %q", reviewStatus)
		}
	}
}
