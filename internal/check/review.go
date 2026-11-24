package check

import (
	"bufio"
	"regexp"
	"strings"

	"github.com/vikashkumar21/sdd/internal/audit"
	"github.com/vikashkumar21/sdd/internal/roadmap"
)

var (
	reviewStatusRe   = regexp.MustCompile(`(?i)^Status:\s*([^\r\n]+)`)
	reviewReviewerRe = regexp.MustCompile(`(?i)^Reviewer:\s*([^\r\n]+)`)
	reqIDPattern     = regexp.MustCompile(`^R-\d{3,}$`)
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
	var currentHeading string
	inFrontmatter := false

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip YAML frontmatter if present
		if lineNum == 1 && trimmed == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter {
			if trimmed == "---" {
				inFrontmatter = false
			}
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			currentHeading = strings.ToLower(trimmed)
		}

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
			if strings.Contains(currentHeading, "traceability") {
				continue
			}
			parts := splitTableRow(trimmed)
			if len(parts) >= 7 && strings.EqualFold(strings.TrimSpace(parts[1]), "req id") {
				continue
			}
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

	// 6. Verify Requirements Traceability Ledger
	c.checkReviewTraceabilityLedger(rel, content, taskStatus, reviewStatus)
}

type ledgerRowInfo struct {
	line        int
	id          string
	statement   string
	codeSymbol  string
	posEvidence string
	negEvidence string
	status      string
}

func (c *checker) checkReviewTraceabilityLedger(rel, content, taskStatus, reviewStatus string) {
	reviewPath := rel + "/review.md"
	normTaskStatus := roadmap.Kind(taskStatus)
	hasLedger := strings.Contains(content, "Requirements Traceability Ledger")

	isApproved := reviewStatus == "approved"
	if !hasLedger {
		if normTaskStatus == "awaiting review" && isApproved {
			c.add(Error, reviewPath, 0, "traceability-ledger", "task is awaiting review with Approved review.md but is missing ## Requirements Traceability Ledger")
		}
		return
	}

	// Load requirements.md if present
	reqPath := rel + "/requirements.md"
	reqContent, hasReqs := c.load(reqPath)
	var expectedReqs []audit.RequirementItem
	if hasReqs {
		expectedReqs, _ = audit.ExtractRequirements(reqContent)
	}

	ledgerRows := make(map[string]ledgerRowInfo)
	var headerLine int
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0
	inLedgerSection := false

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "#") {
			if strings.Contains(trimmed, "Requirements Traceability Ledger") {
				inLedgerSection = true
				continue
			} else if inLedgerSection {
				inLedgerSection = false
			}
		}

		if !inLedgerSection {
			continue
		}

		if strings.HasPrefix(trimmed, "|") {
			parts := splitTableRow(trimmed)
			if len(parts) >= 2 && strings.Contains(strings.ToLower(parts[1]), "req id") {
				headerLine = lineNum
				continue
			}
			if strings.Contains(trimmed, "---") {
				continue
			}

			if len(parts) >= 7 {
				reqID := strings.TrimSpace(parts[1])
				if !reqIDPattern.MatchString(reqID) {
					continue
				}
				statement := strings.TrimSpace(parts[2])
				codeSymbol := strings.TrimSpace(parts[3])
				posEvidence := strings.TrimSpace(parts[4])
				negEvidence := strings.TrimSpace(parts[5])
				status := strings.TrimSpace(parts[6])

				ledgerRows[reqID] = ledgerRowInfo{
					line:        lineNum,
					id:          reqID,
					statement:   statement,
					codeSymbol:  codeSymbol,
					posEvidence: posEvidence,
					negEvidence: negEvidence,
					status:      status,
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		c.add(Error, reviewPath, 0, "traceability-ledger", "reading review.md: %v", err)
	}

	if headerLine == 0 {
		c.add(Error, reviewPath, 0, "traceability-ledger", "review.md contains Requirements Traceability Ledger section but table header is missing or malformed")
		return
	}

	// If review is approved, perform strict completeness & omission checks
	if isApproved || normTaskStatus == "awaiting review" {
		checked := make(map[string]bool)
		for _, expReq := range expectedReqs {
			checked[expReq.ID] = true
			row, exists := ledgerRows[expReq.ID]
			if !exists {
				c.add(Error, reviewPath, headerLine, "traceability-ledger", "requirement %s (%s) from requirements.md is omitted from Requirements Traceability Ledger", expReq.ID, expReq.Statement)
				continue
			}
			c.validateLedgerRow(reviewPath, row)
		}
		for id, row := range ledgerRows {
			if !checked[id] {
				c.validateLedgerRow(reviewPath, row)
			}
		}
	}
}

func (c *checker) validateLedgerRow(reviewPath string, row ledgerRowInfo) {
	if isPlaceholderEvidence(row.codeSymbol) {
		c.add(Error, reviewPath, row.line, "traceability-ledger", "row %s: Code Symbol & File is missing or placeholder (%q)", row.id, row.codeSymbol)
	}
	if isPlaceholderEvidence(row.posEvidence) {
		c.add(Error, reviewPath, row.line, "traceability-ledger", "row %s: Positive Test Evidence is missing or placeholder (%q)", row.id, row.posEvidence)
	}
	if isPlaceholderEvidence(row.negEvidence) {
		c.add(Error, reviewPath, row.line, "traceability-ledger", "row %s: Negative / Boundary Test Evidence is missing or placeholder (%q)", row.id, row.negEvidence)
	}
	lowerStatus := strings.ToLower(row.status)
	if strings.Contains(lowerStatus, "omission") || strings.Contains(lowerStatus, "missing") {
		c.add(Error, reviewPath, row.line, "traceability-ledger", "row %s: requirement marked %q in Requirements Traceability Ledger", row.id, row.status)
	} else if !strings.EqualFold(row.status, "verified") {
		c.add(Error, reviewPath, row.line, "traceability-ledger", "row %s: status is %q (expected 'Verified')", row.id, row.status)
	}
}

func isPlaceholderEvidence(val string) bool {
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return true
	}
	lower := strings.ToLower(trimmed)
	if lower == "pending" || lower == "todo" || lower == "n/a" || lower == "none" || lower == "-" || lower == "tbd" {
		return true
	}
	if strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">") {
		return true
	}
	if strings.Contains(lower, "pending implementation") || strings.Contains(lower, "<paste") {
		return true
	}
	return false
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
