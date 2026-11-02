package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/context"
)

func TestIngestText(t *testing.T) {
	root := t.TempDir()
	var out, errBuf bytes.Buffer
	env := IO{
		Out:    &out,
		Err:    &errBuf,
		Now:    func() time.Time { return time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC) },
		Getenv: func(string) string { return "" },
	}

	code := Run([]string{"ingest", "--dir", root, "--text", "meeting transcript content", "--name", "kickoff"}, env)
	if code != 0 {
		t.Fatalf("runIngest failed with %d: %s", code, errBuf.String())
	}

	m, ok, err := context.ReadManifest(root)
	if err != nil || !ok {
		t.Fatalf("ReadManifest failed: ok=%v, err=%v", ok, err)
	}
	if len(m.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(m.Entries))
	}
	if !strings.Contains(m.Entries[0].Path, "kickoff") {
		t.Errorf("expected path to contain kickoff, got %s", m.Entries[0].Path)
	}
}

func TestIngestFiles(t *testing.T) {
	root := t.TempDir()
	srcFile := filepath.Join(t.TempDir(), "spec.md")
	if err := os.WriteFile(srcFile, []byte("# Project Spec\nDetails here"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out, errBuf bytes.Buffer
	env := IO{
		Out: &out,
		Err: &errBuf,
		Now: func() time.Time { return time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC) },
	}

	code := Run([]string{"ingest", "--dir", root, "--file", srcFile}, env)
	if code != 0 {
		t.Fatalf("runIngest failed: %s", errBuf.String())
	}

	m, ok, err := context.ReadManifest(root)
	if err != nil || !ok {
		t.Fatal(err)
	}
	if len(m.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(m.Entries))
	}
	if m.Entries[0].Format != "markdown" {
		t.Errorf("expected markdown format, got %s", m.Entries[0].Format)
	}
}

func TestIngestStdin(t *testing.T) {
	root := t.TempDir()
	var out, errBuf bytes.Buffer
	stdin := strings.NewReader("piped input context")
	env := IO{
		In:          stdin,
		Out:         &out,
		Err:         &errBuf,
		Interactive: false,
		Now:         func() time.Time { return time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC) },
	}

	code := Run([]string{"ingest", "--dir", root, "--name", "stdin-notes"}, env)
	if code != 0 {
		t.Fatalf("runIngest stdin failed: %s", errBuf.String())
	}

	m, ok, err := context.ReadManifest(root)
	if err != nil || !ok {
		t.Fatal(err)
	}
	if len(m.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(m.Entries))
	}
	if m.Entries[0].Source != "stdin" {
		t.Errorf("expected source stdin, got %s", m.Entries[0].Source)
	}
}

func TestIngestTaskAssociation(t *testing.T) {
	root := t.TempDir()
	var out, errBuf bytes.Buffer
	env := IO{
		Out: &out,
		Err: &errBuf,
		Now: func() time.Time { return time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC) },
	}

	code := Run([]string{"ingest", "--dir", root, "--text", "task notes", "--task", "T-002"}, env)
	if code != 0 {
		t.Fatalf("runIngest task failed: %s", errBuf.String())
	}

	m, ok, err := context.ReadManifest(root)
	if err != nil || !ok {
		t.Fatal(err)
	}
	if len(m.Entries) != 1 || m.Entries[0].Task != "T-002" {
		t.Fatalf("expected entry with task T-002: %+v", m.Entries)
	}
	if !strings.Contains(m.Entries[0].Path, ".sdd/context/T-002/") {
		t.Errorf("expected path inside .sdd/context/T-002/: %s", m.Entries[0].Path)
	}
}

func TestCLIIngestJira(t *testing.T) {
	root := t.TempDir()
	jiraJSON := `{
		"key": "PAY-100",
		"fields": {
			"summary": "Stripe webhook idempotency",
			"description": "Store processed webhook event IDs in Redis.",
			"issuetype": {"name": "Bug"}
		}
	}`
	jiraFile := filepath.Join(t.TempDir(), "pay-100.json")
	if err := os.WriteFile(jiraFile, []byte(jiraJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	var out, errBuf bytes.Buffer
	env := IO{
		Out: &out,
		Err: &errBuf,
		Now: func() time.Time { return time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC) },
	}

	code := Run([]string{"ingest", "--dir", root, "--jira", "--file", jiraFile}, env)
	if code != 0 {
		t.Fatalf("runIngest --jira failed with %d: %s", code, errBuf.String())
	}

	m, ok, err := context.ReadManifest(root)
	if err != nil || !ok {
		t.Fatal(err)
	}
	if len(m.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(m.Entries))
	}
	if m.Entries[0].Format != "jira" {
		t.Errorf("expected format 'jira', got %q", m.Entries[0].Format)
	}
	if !strings.Contains(m.Entries[0].Path, "pay-100") {
		t.Errorf("expected path to contain key pay-100: %s", m.Entries[0].Path)
	}
}

func TestCLIIngestJiraInvalid(t *testing.T) {
	root := t.TempDir()
	var out, errBuf bytes.Buffer
	env := IO{
		Out: &out,
		Err: &errBuf,
		Now: func() time.Time { return time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC) },
	}

	// Malformed Jira input
	code := Run([]string{"ingest", "--dir", root, "--jira", "--text", "random text without key or summary"}, env)
	if code != 1 {
		t.Errorf("expected exit code 1 for invalid Jira issue, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "invalid jira issue") {
		t.Errorf("expected 'invalid jira issue' error, got %q", errBuf.String())
	}
}

func TestCLIFeatureNewFromJira(t *testing.T) {
	root := t.TempDir()
	var out, errBuf bytes.Buffer
	env := IO{
		Out: &out,
		Err: &errBuf,
		Now: func() time.Time { return time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC) },
	}

	// Initialize kit
	initCode := Run([]string{"init", "--yes", "--dir", root, "--name", "demo"}, env)
	if initCode != 0 {
		t.Fatalf("init failed: %s", errBuf.String())
	}

	jiraMD := `# PROJ-777: User Password Reset Flow
Type: Story

## Description
Provide secure self-service password reset via signed email link.

## Acceptance Criteria
- Send password reset email containing HMAC token with 15min expiry
- Invalidate token immediately upon successful password update
`
	jiraFile := filepath.Join(t.TempDir(), "proj-777.md")
	if err := os.WriteFile(jiraFile, []byte(jiraMD), 0o644); err != nil {
		t.Fatal(err)
	}

	out.Reset()
	errBuf.Reset()
	code := Run([]string{"feature", "new", "T-001", "pwd-reset", "--from-jira", jiraFile, "--dir", root}, env)
	if code != 0 {
		t.Fatalf("feature new --from-jira failed with %d: %s", code, errBuf.String())
	}

	if !strings.Contains(out.String(), "Staged Jira issue:") {
		t.Errorf("expected output to mention staged jira issue: %s", out.String())
	}

	reqBytes, err := os.ReadFile(filepath.Join(root, "features", "T-001-pwd-reset", "requirements.md"))
	if err != nil {
		t.Fatalf("failed reading requirements.md: %v", err)
	}
	req := string(reqBytes)
	if !strings.Contains(req, "## 1. Goal\nUser Password Reset Flow") {
		t.Errorf("expected Goal to match Jira summary: %s", req)
	}
	if !strings.Contains(req, "- Jira issue: PROJ-777 (Story)") {
		t.Errorf("expected Context to mention Jira issue: %s", req)
	}
	if !strings.Contains(req, "Send password reset email containing HMAC token with 15min expiry") {
		t.Errorf("expected Acceptance criteria in requirements table: %s", req)
	}

	// Test non-existent file fails
	out.Reset()
	errBuf.Reset()
	failCode := Run([]string{"feature", "new", "T-002", "fail", "--from-jira", "does-not-exist.json", "--dir", root}, env)
	if failCode != 1 {
		t.Errorf("expected exit code 1 for non-existent jira file, got %d", failCode)
	}
}
