package feature

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/detect"
	"github.com/vikashkumar21/sdd/internal/roadmap"
	"github.com/vikashkumar21/sdd/internal/scaffold"
	"github.com/vikashkumar21/sdd/internal/templates"
)

func kit(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	set, err := templates.New("")
	if err != nil {
		t.Fatal(err)
	}
	det, _ := detect.Detect(root)
	now := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)
	p, err := scaffold.BuildPlan(root, scaffold.Answers{Project: "demo", Mode: scaffold.ModeGreenfield, Languages: []string{"go"}}, det, set, now, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scaffold.Apply(p, now, ""); err != nil {
		t.Fatal(err)
	}
	return root
}

func read(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestNewCreatesFolderFromProjectTemplates(t *testing.T) {
	root := kit(t)
	res, err := New(root, "T-001", "define-scope", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Dir != "features/T-001-define-scope" || len(res.Files) != 4 || !res.StatusUpdated {
		t.Fatalf("unexpected result: %+v", res)
	}
	req := read(t, root, "features/T-001-define-scope/requirements.md")
	if !strings.HasPrefix(req, "# Requirements — T-001 Define mission and scope (docs only)") {
		t.Errorf("title not taken from the roadmap row: %q", strings.SplitN(req, "\n", 2)[0])
	}
	rev := read(t, root, "features/T-001-define-scope/review.md")
	if !strings.HasPrefix(rev, "# Code Review — T-001 Define mission and scope (docs only)") {
		t.Errorf("review title not taken from roadmap row: %q", strings.SplitN(rev, "\n", 2)[0])
	}
	if strings.Contains(req, "T-xxx") || strings.Contains(read(t, root, "features/T-001-define-scope/plan.md"), "T-xxx") || strings.Contains(rev, "T-xxx") {
		t.Error("T-xxx tokens were not replaced")
	}
	if !strings.Contains(read(t, root, "features/T-001-define-scope/plan.md"), "`APPROVED T-001`") {
		t.Error("approval phrase should name the real task")
	}
	for _, task := range roadmap.Parse(read(t, root, "ROADMAP.md")) {
		if task.ID == "T-001" && roadmap.Kind(task.Status) != "specifying" {
			t.Errorf("T-001 status = %q, want Specifying", task.Status)
		}
	}
}

func TestExplicitTitleWins(t *testing.T) {
	root := kit(t)
	if _, err := New(root, "T-001", "scope", "My custom title"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(read(t, root, "features/T-001-scope/validation.md"), "# Validation — T-001 My custom title") {
		t.Error("explicit title not used")
	}
}

func TestNewRefusesBadInput(t *testing.T) {
	root := kit(t)
	tests := []struct{ name, id, slug string }{
		{"bad id", "T-1", "x"},
		{"bootstrap task", "T-000", "x"},
		{"bad slug upper", "T-001", "Bad Slug"},
		{"bad slug underscore", "T-001", "bad_slug"},
		{"unknown task", "T-042", "x"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New(root, tc.id, tc.slug, ""); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
	if entries, _ := os.ReadDir(filepath.Join(root, "features")); len(entries) != 1 {
		t.Errorf("failed attempts must not create folders, found %d entries", len(entries))
	}
}

func TestNewRefusesSecondFolderForSameTask(t *testing.T) {
	root := kit(t)
	if _, err := New(root, "T-001", "first", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := New(root, "T-001", "second", ""); err == nil {
		t.Fatal("expected an error for a second folder")
	}
}

func TestStatusOnlyChangesFromNotStarted(t *testing.T) {
	root := kit(t)
	p := filepath.Join(root, "ROADMAP.md")
	data, _ := os.ReadFile(p)
	edited := strings.Replace(string(data), "| Not started |", "| Blocked (waiting on a decision) |", 1)
	if err := os.WriteFile(p, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := New(root, "T-001", "scope", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusUpdated {
		t.Error("a Blocked task's status must not be changed")
	}
	if !strings.Contains(read(t, root, "ROADMAP.md"), "Blocked (waiting on a decision)") {
		t.Error("ROADMAP.md was modified")
	}
}

func TestMissingTemplatesIsAnError(t *testing.T) {
	root := kit(t)
	if err := os.Remove(filepath.Join(root, "features", "_templates", "plan.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := New(root, "T-001", "scope", ""); err == nil {
		t.Fatal("expected an error")
	}
	if _, err := os.Stat(filepath.Join(root, "features", "T-001-scope")); err == nil {
		t.Error("nothing may be created when a template is missing")
	}
}

func TestNewConfinedRoot(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "does-not-exist")
	if _, err := New(nonExistent, "T-001", "scope", ""); err == nil {
		t.Fatal("expected error opening non-existent root, got nil")
	}
}

func TestNewFromJira(t *testing.T) {
	root := kit(t)
	jiraJSON := `{
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

	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	opts := Options{
		FromJiraData: []byte(jiraJSON),
		JiraOrigin:   "issue-101.json",
		Now:          now,
	}

	res, err := NewWithOptions(root, "T-001", "oauth-refresh", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Dir != "features/T-001-oauth-refresh" {
		t.Errorf("unexpected dir: %q", res.Dir)
	}

	req := read(t, root, "features/T-001-oauth-refresh/requirements.md")

	// 1. Goal replaced with Jira summary
	if !strings.Contains(req, "## 1. Goal\nImplement OAuth2 token refresh endpoint") {
		t.Errorf("Goal not replaced with Jira summary in requirements.md:\n%s", req)
	}

	// 2. Context reference added
	if !strings.Contains(req, "- Jira issue: PROJ-101 (Story)") {
		t.Errorf("Context missing Jira issue reference in requirements.md:\n%s", req)
	}

	// 3. Functional requirements table injected
	if !strings.Contains(req, "| R-001 | The system SHALL Reject expired refresh tokens with 401 Unauthorized | MUST | test |") {
		t.Errorf("R-001 row missing or malformed in requirements.md:\n%s", req)
	}
	if !strings.Contains(req, `| R-002 | The system SHALL Issue new access token with 1 hour TTL \| include token type Bearer | MUST | test |`) {
		t.Errorf("R-002 row with escaped pipe missing in requirements.md:\n%s", req)
	}

	// 4. Acceptance criteria section injected
	if !strings.Contains(req, "- Reject expired refresh tokens with 401 Unauthorized\n- Issue new access token with 1 hour TTL | include token type Bearer") {
		t.Errorf("Acceptance criteria section missing injected bullets:\n%s", req)
	}
}

func TestNewFromJiraStagedContext(t *testing.T) {
	root := kit(t)
	jiraMD := `# PROJ-202: Add Healthcheck Endpoint
Type: Task

## Description
Provide /healthz for Kubernetes probes.

## Acceptance Criteria
- Return 200 OK when database ping succeeds
- Return 503 Service Unavailable when database ping fails
`

	now := time.Date(2026, 11, 2, 14, 30, 0, 0, time.UTC)
	opts := Options{
		FromJiraData: []byte(jiraMD),
		JiraOrigin:   "task-202.md",
		Now:          now,
	}

	res, err := NewWithOptions(root, "T-001", "healthcheck", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.StagedContext == "" {
		t.Fatal("expected StagedContext to be non-empty")
	}

	// Verify staged file exists
	stagedPath := filepath.Join(root, filepath.FromSlash(res.StagedContext))
	data, err := os.ReadFile(stagedPath)
	if err != nil {
		t.Fatalf("reading staged file %s: %v", stagedPath, err)
	}
	if string(data) != jiraMD {
		t.Errorf("staged file content mismatch: got %q, want %q", string(data), jiraMD)
	}

	// Verify .sdd/context/manifest.json contains the entry
	manifestPath := filepath.Join(root, ".sdd", "context", "manifest.json")
	mdata, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("reading context manifest: %v", err)
	}
	if !strings.Contains(string(mdata), `"format": "jira"`) {
		t.Errorf("context manifest missing jira format: %s", string(mdata))
	}
	if !strings.Contains(string(mdata), `"task": "T-001"`) {
		t.Errorf("context manifest missing task T-001: %s", string(mdata))
	}
}

func TestNewFromDocPDF(t *testing.T) {
	root := kit(t)
	pdfData := []byte("%PDF-1.4\n1 0 obj\n<<>>\nendobj\n%%EOF")
	now := time.Date(2026, 11, 2, 15, 0, 0, 0, time.UTC)
	opts := Options{
		FromDocData: pdfData,
		FromDocPath: "architecture.pdf",
		Now:         now,
	}

	res, err := NewWithOptions(root, "T-001", "arch-design", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.StagedContext == "" {
		t.Fatal("expected StagedContext to be set")
	}

	req := read(t, root, "features/T-001-arch-design/requirements.md")
	if !strings.Contains(req, "- Reference document: `"+res.StagedContext+"` (Format: pdf, Source: architecture.pdf)") {
		t.Errorf("expected reference line in requirements.md:\n%s", req)
	}

	// Verify staged file in context
	mdata := read(t, root, ".sdd/context/manifest.json")
	if !strings.Contains(mdata, `"format": "pdf"`) {
		t.Errorf("expected format pdf in manifest: %s", mdata)
	}
}

func TestNewFromDocInvalidPDF(t *testing.T) {
	root := kit(t)
	opts := Options{
		FromDocData: []byte("corrupted non-pdf file"),
		FromDocPath: "bad.pdf",
	}

	_, err := NewWithOptions(root, "T-001", "bad-doc", opts)
	if err == nil {
		t.Fatal("expected error for invalid PDF signature, got nil")
	}
	if !strings.Contains(err.Error(), "missing %PDF- header signature") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewWithLegacy3TemplatesOnly(t *testing.T) {
	root := kit(t)
	// Remove review.md template to simulate legacy kit
	reviewTpl := filepath.Join(root, "features", "_templates", "review.md")
	if err := os.Remove(reviewTpl); err != nil {
		t.Fatal(err)
	}

	res, err := New(root, "T-001", "legacy-task", "")
	if err != nil {
		t.Fatalf("unexpected failure on legacy kit without review.md: %v", err)
	}
	if len(res.Files) != 3 {
		t.Fatalf("expected 3 files on legacy kit, got %d: %+v", len(res.Files), res.Files)
	}
}

func TestNewAllTypologies(t *testing.T) {
	root := kit(t)

	// 1. Bugfix
	resBug, err := NewWithOptions(root, "T-001", "fix-null-pointer", Options{Type: TypologyBugfix})
	if err != nil {
		t.Fatalf("NewWithOptions(bugfix) failed: %v", err)
	}
	if resBug.Dir != "bugfixes/T-001-fix-null-pointer" {
		t.Errorf("expected bugfixes dir, got %s", resBug.Dir)
	}
	repro := read(t, root, "bugfixes/T-001-fix-null-pointer/reproduction.md")
	if !strings.Contains(repro, "Mandatory Step 0: Minimal Failing Test Case") {
		t.Errorf("expected reproduction.md to have Step 0 failing test section: %s", repro)
	}
	if len(resBug.Files) != 5 {
		t.Errorf("expected 5 files for bugfix, got %d: %+v", len(resBug.Files), resBug.Files)
	}

	// 2. Cross-typology duplicate refusal
	_, err = NewWithOptions(root, "T-001", "duplicate-in-hotfixes", Options{Type: TypologyHotfix})
	if err == nil {
		t.Fatal("expected error when task already exists in bugfixes, got nil")
	}
	if !strings.Contains(err.Error(), "already has a bugfixes folder") {
		t.Errorf("unexpected error: %v", err)
	}

	// Add T-002 and T-003 to ROADMAP.md
	rmPath := filepath.Join(root, "ROADMAP.md")
	rmContent := read(t, root, "ROADMAP.md")
	rmContent += `
| T-002 | Outage fix | S | none | Hotfix | none | Not started |
| T-003 | Architecture spike | S | none | Spike | none | Not started |
`
	if err := os.WriteFile(rmPath, []byte(rmContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// 3. Hotfix
	resHot, err := NewWithOptions(root, "T-002", "emergency-patch", Options{Type: TypologyHotfix})
	if err != nil {
		t.Fatalf("NewWithOptions(hotfix) failed: %v", err)
	}
	if resHot.Dir != "hotfixes/T-002-emergency-patch" {
		t.Errorf("expected hotfixes dir, got %s", resHot.Dir)
	}
	if len(resHot.Files) != 4 {
		t.Errorf("expected 4 files for hotfix, got %d: %+v", len(resHot.Files), resHot.Files)
	}
	incident := read(t, root, "hotfixes/T-002-emergency-patch/incident.md")
	if !strings.Contains(incident, "Incident Overview") {
		t.Errorf("expected incident.md content, got: %s", incident)
	}

	// 4. Spike
	resSpike, err := NewWithOptions(root, "T-003", "investigate-mcp", Options{Type: TypologySpike})
	if err != nil {
		t.Fatalf("NewWithOptions(spike) failed: %v", err)
	}
	if resSpike.Dir != "spikes/T-003-investigate-mcp" {
		t.Errorf("expected spikes dir, got %s", resSpike.Dir)
	}
	if len(resSpike.Files) != 3 {
		t.Errorf("expected 3 files for spike, got %d: %+v", len(resSpike.Files), resSpike.Files)
	}
	hyp := read(t, root, "spikes/T-003-investigate-mcp/hypotheses.md")
	if !strings.Contains(hyp, "Disposable Prototype Boundary") {
		t.Errorf("expected hypotheses.md to have Disposable Prototype Boundary: %s", hyp)
	}
}
