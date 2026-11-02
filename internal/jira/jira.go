// Package jira parses Jira issues from JSON, Markdown, and plain text formats
// and injects extracted requirements into spec-driven task templates.
package jira

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	keyRegex = regexp.MustCompile(`\b([A-Z][A-Z0-9]+-\d+)\b`)
)

// Issue represents a parsed Jira issue specification.
type Issue struct {
	Key                string   `json:"key"`
	Summary            string   `json:"summary"`
	Description        string   `json:"description"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	Type               string   `json:"type,omitempty"`
	Priority           string   `json:"priority,omitempty"`
	RawFormat          string   `json:"raw_format"` // "json", "markdown", "text"
}

// Parse inspects data, detects format (JSON, Markdown, or plain text), and extracts issue fields.
func Parse(data []byte) (Issue, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return Issue{}, errors.New("jira: input is empty")
	}

	// 1. Try parsing as JSON first
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		issue, err := parseJSON([]byte(trimmed))
		if err != nil {
			return Issue{}, fmt.Errorf("jira: %w", err)
		}
		issue.RawFormat = "json"
		return issue, nil
	}

	// 2. Parse as Markdown / structured text
	issue, err := parseTextOrMarkdown(trimmed)
	if err != nil {
		return Issue{}, fmt.Errorf("jira: %w", err)
	}
	return issue, nil
}

type jiraJSONPayload struct {
	Key      string `json:"key"`
	Summary  string `json:"summary"`
	Desc     any    `json:"description"`
	Type     any    `json:"type"`
	Priority any    `json:"priority"`
	AC       any    `json:"acceptance_criteria"`
	Criteria any    `json:"criteria"`
	Fields   *struct {
		Summary      string         `json:"summary"`
		Desc         any            `json:"description"`
		IssueType    any            `json:"issuetype"`
		Priority     any            `json:"priority"`
		CustomFields map[string]any `json:"-"`
	} `json:"fields"`
}

func parseJSON(data []byte) (Issue, error) {
	var payload jiraJSONPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return Issue{}, err
	}

	var issue Issue
	issue.Key = strings.TrimSpace(payload.Key)
	issue.Summary = strings.TrimSpace(payload.Summary)
	issue.Description = extractText(payload.Desc)
	issue.Type = extractNameOrString(payload.Type)
	issue.Priority = extractNameOrString(payload.Priority)
	issue.AcceptanceCriteria = extractStringSlice(payload.AC)
	if len(issue.AcceptanceCriteria) == 0 {
		issue.AcceptanceCriteria = extractStringSlice(payload.Criteria)
	}

	if payload.Fields != nil {
		if issue.Summary == "" {
			issue.Summary = strings.TrimSpace(payload.Fields.Summary)
		}
		if issue.Description == "" {
			issue.Description = extractText(payload.Fields.Desc)
		}
		if issue.Type == "" {
			issue.Type = extractNameOrString(payload.Fields.IssueType)
		}
		if issue.Priority == "" {
			issue.Priority = extractNameOrString(payload.Fields.Priority)
		}

		// Also check standard dynamic fields in fields object
		var dynamic map[string]any
		if err := json.Unmarshal(data, &struct {
			Fields *map[string]any `json:"fields"`
		}{Fields: &dynamic}); err == nil && dynamic != nil {
			for k, v := range dynamic {
				if strings.Contains(strings.ToLower(k), "acceptance") {
					ac := extractStringSlice(v)
					if len(ac) > 0 {
						issue.AcceptanceCriteria = append(issue.AcceptanceCriteria, ac...)
					}
				}
			}
		}
	}

	// If acceptance criteria wasn't an explicit JSON field, extract from Description
	if len(issue.AcceptanceCriteria) == 0 && issue.Description != "" {
		issue.AcceptanceCriteria = extractACFromText(issue.Description)
	}

	if issue.Key == "" {
		if m := keyRegex.FindStringSubmatch(issue.Summary); len(m) > 1 {
			issue.Key = m[1]
		}
	}

	if issue.Summary == "" && issue.Key == "" {
		return Issue{}, errors.New("jira: JSON lacks both issue key and summary")
	}

	return issue, nil
}

func extractNameOrString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val)
	case map[string]any:
		if n, ok := val["name"].(string); ok {
			return strings.TrimSpace(n)
		}
		if vStr, ok := val["value"].(string); ok {
			return strings.TrimSpace(vStr)
		}
	}
	return ""
}

func extractStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []any:
		var out []string
		for _, item := range val {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	case []string:
		var out []string
		for _, s := range val {
			if strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	case string:
		return extractACFromText(val)
	}
	return nil
}

func extractText(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val)
	case map[string]any:
		// Atlassian Document Format (ADF) handling: collect text values
		var sb strings.Builder
		var walk func(any)
		walk = func(node any) {
			if m, ok := node.(map[string]any); ok {
				if m["type"] == "text" {
					if t, ok := m["text"].(string); ok {
						sb.WriteString(t)
					}
				}
				if content, ok := m["content"].([]any); ok {
					for _, c := range content {
						walk(c)
					}
				}
			} else if arr, ok := node.([]any); ok {
				for _, it := range arr {
					walk(it)
				}
			}
		}
		walk(val)
		return strings.TrimSpace(sb.String())
	}
	return ""
}

func parseTextOrMarkdown(text string) (Issue, error) {
	lines := strings.Split(text, "\n")
	var issue Issue
	issue.RawFormat = "markdown"

	// Find Key
	if m := keyRegex.FindStringSubmatch(text); len(m) > 1 {
		issue.Key = m[1]
	}

	var descriptionLines []string
	var acLines []string
	currentSection := "" // "desc", "ac", ""
	hasExplicitSummary := false

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r\t ")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if currentSection == "desc" {
				descriptionLines = append(descriptionLines, "")
			}
			continue
		}

		lower := strings.ToLower(trimmed)

		// Check section headers
		switch {
		case strings.HasPrefix(lower, "## description") || strings.HasPrefix(lower, "description:") || strings.HasPrefix(lower, "*description:*") || strings.HasPrefix(lower, "h2. description"):
			currentSection = "desc"
			continue
		case strings.HasPrefix(lower, "## acceptance criteria") || strings.HasPrefix(lower, "acceptance criteria:") || strings.HasPrefix(lower, "*acceptance criteria:*") || strings.HasPrefix(lower, "h2. acceptance criteria") || lower == "ac:" || lower == "## ac":
			currentSection = "ac"
			continue
		case strings.HasPrefix(lower, "## ") || strings.HasPrefix(lower, "h2. "):
			currentSection = ""
		}

		// Detect Summary if not set
		if issue.Summary == "" {
			if strings.HasPrefix(lower, "summary:") {
				issue.Summary = strings.TrimSpace(line[len("summary:"):])
				hasExplicitSummary = true
				continue
			}
			if strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "h1. ") {
				header := strings.TrimPrefix(trimmed, "# ")
				header = strings.TrimPrefix(header, "h1. ")
				// Strip [KEY] or KEY: prefix if present
				if issue.Key != "" {
					header = strings.TrimPrefix(header, "["+issue.Key+"]")
					header = strings.TrimPrefix(header, issue.Key+":")
					header = strings.TrimPrefix(header, issue.Key+" -")
					header = strings.TrimPrefix(header, issue.Key)
					header = strings.TrimSpace(header)
				}
				issue.Summary = header
				hasExplicitSummary = true
				continue
			}
		}

		// Detect Issue Type
		if issue.Type == "" && strings.HasPrefix(lower, "type:") {
			issue.Type = strings.TrimSpace(line[len("type:"):])
			continue
		}

		// Collect section content
		switch currentSection {
		case "desc":
			descriptionLines = append(descriptionLines, line)
		case "ac":
			acLines = append(acLines, line)
		}
	}

	issue.Description = strings.TrimSpace(strings.Join(descriptionLines, "\n"))
	issue.AcceptanceCriteria = parseBulletPoints(acLines)

	// Fallback for summary only if Key was found or Jira structure was detected
	if issue.Summary == "" && len(lines) > 0 && issue.Key != "" {
		first := strings.TrimSpace(lines[0])
		first = strings.TrimPrefix(first, issue.Key+":")
		first = strings.TrimPrefix(first, "["+issue.Key+"]")
		first = strings.TrimPrefix(first, issue.Key+" -")
		first = strings.TrimPrefix(first, issue.Key)
		issue.Summary = strings.TrimSpace(first)
	}

	if issue.Key == "" && !hasExplicitSummary {
		return Issue{}, errors.New("unable to detect Jira issue key or summary")
	}

	if issue.Summary == "" && issue.Key == "" {
		return Issue{}, errors.New("unable to detect Jira issue key or summary")
	}

	return issue, nil
}

func parseBulletPoints(lines []string) []string {
	var points []string
	bulletRe := regexp.MustCompile(`^(?:[-*•]|\d+\.|\([a-z0-9]+\)|\[[ xX]?\])\s+(.+)`)
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" {
			continue
		}
		if m := bulletRe.FindStringSubmatch(trimmed); len(m) > 1 {
			points = append(points, strings.TrimSpace(m[1]))
		} else if strings.HasPrefix(strings.ToLower(trimmed), "given ") {
			points = append(points, trimmed)
		} else if len(points) > 0 {
			// Continuation of previous bullet point
			points[len(points)-1] += " " + trimmed
		} else {
			points = append(points, trimmed)
		}
	}
	return points
}

func extractACFromText(text string) []string {
	lines := strings.Split(text, "\n")
	var acLines []string
	inAC := false
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		lower := strings.ToLower(trimmed)
		if strings.Contains(lower, "acceptance criteria") || strings.HasPrefix(lower, "ac:") {
			inAC = true
			continue
		}
		if inAC {
			if strings.HasPrefix(trimmed, "#") || (strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(lower, "given")) {
				break
			}
			acLines = append(acLines, l)
		}
	}
	return parseBulletPoints(acLines)
}

// InjectRequirements merges the extracted Jira issue details into standard requirements.md template content.
func InjectRequirements(templateText string, issue Issue) string {
	res := templateText

	// 1. Inject Goal
	if issue.Summary != "" {
		res = strings.Replace(res, "<One sentence: what outcome does this task deliver?>", escapePipes(issue.Summary), 1)
	}

	// 2. Inject Context & References
	if issue.Key != "" {
		contextAddition := fmt.Sprintf("- Jira issue: %s", issue.Key)
		if issue.Type != "" {
			contextAddition += fmt.Sprintf(" (%s)", issue.Type)
		}
		if strings.Contains(res, "## 2. Context and references") {
			parts := strings.SplitN(res, "## 2. Context and references\n", 2)
			if len(parts) == 2 {
				res = parts[0] + "## 2. Context and references\n" + contextAddition + "\n" + parts[1]
			}
		}
	}

	// 3. Inject Functional requirements table
	if len(issue.AcceptanceCriteria) > 0 {
		var reqRows []string
		for i, ac := range issue.AcceptanceCriteria {
			id := fmt.Sprintf("R-%03d", i+1)
			cleanAC := escapePipes(ac)
			if !strings.HasPrefix(strings.ToLower(cleanAC), "the system shall") {
				cleanAC = "The system SHALL " + cleanAC
			}
			reqRows = append(reqRows, fmt.Sprintf("| %s | %s | MUST | test |", id, cleanAC))
		}
		replacementTable := strings.Join(reqRows, "\n")
		oldRow := "| R-001 | The system SHALL … | MUST | test / inspection / demo |"
		res = strings.Replace(res, oldRow, replacementTable, 1)
	}

	// 4. Inject Acceptance Criteria
	if len(issue.AcceptanceCriteria) > 0 {
		var criteriaBullets []string
		for _, ac := range issue.AcceptanceCriteria {
			criteriaBullets = append(criteriaBullets, "- "+ac)
		}
		oldCriteria := "- Given <…>, when <…>, then <…>."
		res = strings.Replace(res, oldCriteria, strings.Join(criteriaBullets, "\n"), 1)
	}

	return res
}

func escapePipes(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}
