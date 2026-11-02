package jira_test

import (
	"strings"
	"testing"

	"github.com/vikashkumar21/sdd/internal/jira"
)

func TestParseJiraJSON(t *testing.T) {
	t.Parallel()

	// 1. Jira REST API export format
	restJSON := `{
		"key": "PROJ-101",
		"fields": {
			"summary": "Implement OAuth2 token refresh endpoint",
			"description": "Endpoint must refresh expired access tokens using valid refresh tokens.",
			"issuetype": {"name": "Story"},
			"priority": {"name": "High"},
			"customfield_10020_acceptance": [
				"Reject expired refresh tokens with 401 Unauthorized",
				"Issue new access token with 1 hour TTL | include token type Bearer"
			]
		}
	}`

	issue, err := jira.Parse([]byte(restJSON))
	if err != nil {
		t.Fatalf("unexpected error parsing REST JSON: %v", err)
	}
	if issue.Key != "PROJ-101" {
		t.Errorf("expected key PROJ-101, got %q", issue.Key)
	}
	if issue.Summary != "Implement OAuth2 token refresh endpoint" {
		t.Errorf("expected summary %q, got %q", "Implement OAuth2 token refresh endpoint", issue.Summary)
	}
	if issue.Type != "Story" {
		t.Errorf("expected type Story, got %q", issue.Type)
	}
	if issue.Priority != "High" {
		t.Errorf("expected priority High, got %q", issue.Priority)
	}
	if len(issue.AcceptanceCriteria) != 2 {
		t.Fatalf("expected 2 acceptance criteria, got %d", len(issue.AcceptanceCriteria))
	}
	if issue.AcceptanceCriteria[0] != "Reject expired refresh tokens with 401 Unauthorized" {
		t.Errorf("unexpected AC[0]: %q", issue.AcceptanceCriteria[0])
	}
	if issue.RawFormat != "json" {
		t.Errorf("expected raw_format json, got %q", issue.RawFormat)
	}

	// 2. Simplified JSON schema
	simpleJSON := `{
		"key": "CORE-42",
		"summary": "Add health check endpoint",
		"description": "Expose /healthz returning 200 OK",
		"type": "Task",
		"priority": "Medium",
		"acceptance_criteria": [
			"GET /healthz returns 200 with status UP",
			"GET /healthz responds in under 10ms"
		]
	}`

	issue2, err := jira.Parse([]byte(simpleJSON))
	if err != nil {
		t.Fatalf("unexpected error parsing simplified JSON: %v", err)
	}
	if issue2.Key != "CORE-42" || issue2.Summary != "Add health check endpoint" {
		t.Errorf("unexpected issue2 values: %+v", issue2)
	}
	if len(issue2.AcceptanceCriteria) != 2 {
		t.Fatalf("expected 2 criteria in issue2, got %d", len(issue2.AcceptanceCriteria))
	}

	// 3. ADF (Atlassian Document Format) Description
	adfJSON := `{
		"key": "ADF-1",
		"fields": {
			"summary": "Rich text issue",
			"description": {
				"type": "doc",
				"version": 1,
				"content": [
					{
						"type": "paragraph",
						"content": [
							{"type": "text", "text": "This is rich description text."}
						]
					}
				]
			}
		}
	}`
	issue3, err := jira.Parse([]byte(adfJSON))
	if err != nil {
		t.Fatalf("unexpected error parsing ADF JSON: %v", err)
	}
	if issue3.Description != "This is rich description text." {
		t.Errorf("expected extracted ADF text %q, got %q", "This is rich description text.", issue3.Description)
	}
}

func TestParseJiraMarkdownAndText(t *testing.T) {
	t.Parallel()

	// 1. Markdown export
	mdText := `# [PROJ-500] Database Connection Pool Sizing
Type: Enhancement

## Description
Tune connection pool parameters based on available host CPUs and DB max_connections.

## Acceptance Criteria
- Set default max open connections to 25
- Validate max idle connections is less than or equal to max open
* Log warning when pool reaches 80% saturation
`

	issue, err := jira.Parse([]byte(mdText))
	if err != nil {
		t.Fatalf("unexpected error parsing Markdown: %v", err)
	}
	if issue.Key != "PROJ-500" {
		t.Errorf("expected key PROJ-500, got %q", issue.Key)
	}
	if issue.Summary != "Database Connection Pool Sizing" {
		t.Errorf("expected summary 'Database Connection Pool Sizing', got %q", issue.Summary)
	}
	if issue.Type != "Enhancement" {
		t.Errorf("expected type Enhancement, got %q", issue.Type)
	}
	if !strings.Contains(issue.Description, "Tune connection pool parameters") {
		t.Errorf("unexpected description: %q", issue.Description)
	}
	if len(issue.AcceptanceCriteria) != 3 {
		t.Fatalf("expected 3 ACs, got %d", len(issue.AcceptanceCriteria))
	}
	if issue.AcceptanceCriteria[0] != "Set default max open connections to 25" {
		t.Errorf("unexpected AC 0: %q", issue.AcceptanceCriteria[0])
	}
	if issue.AcceptanceCriteria[2] != "Log warning when pool reaches 80% saturation" {
		t.Errorf("unexpected AC 2: %q", issue.AcceptanceCriteria[2])
	}

	// 2. Structured plain text
	plainText := `PROJ-789: Fix memory leak in background worker
Description: Worker routines fail to close context channels upon job completion.

Acceptance Criteria:
1. Ensure all context cancels are called via defer
2. Memory usage remains stable over 24-hour load test
`
	issue2, err := jira.Parse([]byte(plainText))
	if err != nil {
		t.Fatalf("unexpected error parsing plain text: %v", err)
	}
	if issue2.Key != "PROJ-789" {
		t.Errorf("expected PROJ-789, got %q", issue2.Key)
	}
	if issue2.Summary != "Fix memory leak in background worker" {
		t.Errorf("unexpected summary: %q", issue2.Summary)
	}
	if len(issue2.AcceptanceCriteria) != 2 {
		t.Fatalf("expected 2 ACs, got %d", len(issue2.AcceptanceCriteria))
	}
}

func TestParseJiraMalformed(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"whitespace only", "   \n\t  "},
		{"empty JSON", "{}"},
		{"unrelated JSON", `{"foo": "bar"}`},
		{"unstructured text without key or summary", "Just some random text with nothing else"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := jira.Parse([]byte(tc.input))
			if err == nil {
				t.Errorf("expected error for input %q, got nil", tc.input)
			}
		})
	}
}

func TestInjectRequirements(t *testing.T) {
	t.Parallel()

	template := `# Requirements — T-002 token-refresh

> ALWAYS BE DETERMINISTIC. Human approval resolves every decision.

Status: Draft
Roadmap ref: ROADMAP.md T-002

## 1. Goal
<One sentence: what outcome does this task deliver?>

## 2. Context and references
- Mission / principles: Principle 1 (Deterministic)

## 3. Scope
**In scope**

## 4. Functional requirements

| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL … | MUST | test / inspection / demo |

## 5. Non-functional requirements

## 6. Constraints and assumptions

## 7. Open questions

## 8. Acceptance criteria
- Given <…>, when <…>, then <…>.
`

	issue := jira.Issue{
		Key:     "AUTH-99",
		Summary: "Support OAuth refresh token rotation",
		Type:    "Story",
		AcceptanceCriteria: []string{
			"Rotate refresh token on every refresh call",
			"Revoke old token family if revoked token is reused | security measure",
		},
	}

	result := jira.InjectRequirements(template, issue)

	// Goal replaced
	if !strings.Contains(result, "## 1. Goal\nSupport OAuth refresh token rotation") {
		t.Errorf("Goal not injected correctly:\n%s", result)
	}

	// Context and references added
	if !strings.Contains(result, "- Jira issue: AUTH-99 (Story)") {
		t.Errorf("Jira issue reference not injected in Context:\n%s", result)
	}

	// Functional requirements table injected with R-001, R-002 and escaped pipe
	if !strings.Contains(result, "| R-001 | The system SHALL Rotate refresh token on every refresh call | MUST | test |") {
		t.Errorf("R-001 row missing or malformed:\n%s", result)
	}
	if !strings.Contains(result, "| R-002 | The system SHALL Revoke old token family if revoked token is reused \\| security measure | MUST | test |") {
		t.Errorf("R-002 row with escaped pipe missing or malformed:\n%s", result)
	}

	// Acceptance criteria injected
	if !strings.Contains(result, "- Rotate refresh token on every refresh call\n- Revoke old token family if revoked token is reused | security measure") {
		t.Errorf("Acceptance criteria section not injected properly:\n%s", result)
	}
}
