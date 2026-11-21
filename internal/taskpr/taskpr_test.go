package taskpr

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/state"
)

func setupTestTask(t *testing.T, typology, taskID, slug string, reviewStatus string, blockers int) string {
	t.Helper()
	root := t.TempDir()

	var relDir string
	switch typology {
	case "bugfix":
		relDir = filepath.Join("tasks", "bugfixes", taskID+"-"+slug)
	case "hotfix":
		relDir = filepath.Join("tasks", "hotfixes", taskID+"-"+slug)
	case "spike":
		relDir = filepath.Join("tasks", "spikes", taskID+"-"+slug)
	default:
		relDir = filepath.Join("tasks", "features", taskID+"-"+slug)
	}

	taskDir := filepath.Join(root, relDir)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create requirements / reproduction / hypotheses
	switch typology {
	case "bugfix":
		repro := "# Reproduction — " + taskID + " Fix null pointer\n\n## 1. Defect\nNull pointer crash when processing empty input.\n"
		_ = os.WriteFile(filepath.Join(taskDir, "reproduction.md"), []byte(repro), 0o644)
	case "spike":
		hyp := "# Hypotheses — " + taskID + " Evaluate vector db\n\n## 1. Hypotheses\nVector databases improve retrieval recall.\n"
		_ = os.WriteFile(filepath.Join(taskDir, "hypotheses.md"), []byte(hyp), 0o644)
	case "hotfix":
		inc := "# Incident — " + taskID + " Memory leak patch\n\n## 1. Incident\nServer memory exhaustion in production worker.\n"
		_ = os.WriteFile(filepath.Join(taskDir, "incident.md"), []byte(inc), 0o644)
	default:
		req := "# Requirements — " + taskID + " Universal CI workflows\n\n## 1. Goal\nProvide adaptable CI workflow templates.\n"
		_ = os.WriteFile(filepath.Join(taskDir, "requirements.md"), []byte(req), 0o644)
	}

	// Create plan.md (or patch.md)
	plan := "# Plan — " + taskID + "\n\n## 5. Files to touch\n\n| File | Action | Purpose |\n|---|---|---|\n| `cmd/sdd/main.go` | modify | update |\n| `internal/ci/ci.go` | create | ci engine |\n"
	if typology == "hotfix" {
		_ = os.WriteFile(filepath.Join(taskDir, "patch.md"), []byte(plan), 0o644)
	} else {
		_ = os.WriteFile(filepath.Join(taskDir, "plan.md"), []byte(plan), 0o644)
	}

	// Create validation.md
	val := "# Validation — " + taskID + "\n\n## 1. Traceability matrix\n\n| ID | Method | Command | Expected | Actual | Status |\n|---|---|---|---|---|---|\n| R-001 | test | `make test` | pass | pass | PASS |\n"
	_ = os.WriteFile(filepath.Join(taskDir, "validation.md"), []byte(val), 0o644)

	// Create review.md if not spike or hotfix without review
	if typology != "spike" && reviewStatus != "" {
		rev := "# Code Review — " + taskID + "\nStatus: " + reviewStatus + "\n\n## 2. Risk & Findings Register\n\n| ID | Category | Severity | Finding | Suggested Action | Status |\n|---|---|---|---|---|---|\n"
		for i := 0; i < blockers; i++ {
			rev += "| B-001 | Security | 🔴 Blocker | critical flaw | fix it | Open |\n"
		}
		_ = os.WriteFile(filepath.Join(taskDir, "review.md"), []byte(rev), 0o644)
	}

	return root
}

func TestPackageTaskPR_QualifiedFeature(t *testing.T) {
	root := setupTestTask(t, "feature", "T-046", "ci-workflows", "Approved", 0)

	bundle, err := PackageTaskPR(root, PROptions{
		TaskID: "T-046",
	})
	if err != nil {
		t.Fatalf("PackageTaskPR failed: %v", err)
	}

	if bundle.TaskID != "T-046" {
		t.Errorf("taskID = %q, want T-046", bundle.TaskID)
	}
	if bundle.Typology != "feature" {
		t.Errorf("typology = %q, want feature", bundle.Typology)
	}
	if bundle.Title != "feat(T-046): Universal CI workflows" {
		t.Errorf("title = %q, want feat(T-046): Universal CI workflows", bundle.Title)
	}
	if !strings.Contains(bundle.Summary, "Provide adaptable CI workflow templates") {
		t.Errorf("summary = %q", bundle.Summary)
	}
	if len(bundle.ChangedFiles) != 2 {
		t.Fatalf("expected 2 changed files, got %d", len(bundle.ChangedFiles))
	}
	if !bundle.QualifiesForPR {
		t.Errorf("expected QualifiesForPR=true, got false (warnings: %v)", bundle.Warnings)
	}
	if bundle.ReviewStatus != "Approved" {
		t.Errorf("reviewStatus = %q, want Approved", bundle.ReviewStatus)
	}

	// Verify formatted markdown body
	if !strings.Contains(bundle.Body, "## Summary") {
		t.Errorf("markdown body missing Summary header")
	}
	if !strings.Contains(bundle.Body, "PR Qualification:** `QUALIFIED`") {
		t.Errorf("markdown body missing QUALIFIED badge")
	}
}

func TestPackageTaskPR_UnqualifiedWithBlockers(t *testing.T) {
	root := setupTestTask(t, "feature", "T-046", "ci-workflows", "Pending", 2)

	bundle, err := PackageTaskPR(root, PROptions{
		TaskID: "T-046",
	})
	if err != nil {
		t.Fatalf("PackageTaskPR failed: %v", err)
	}

	if bundle.QualifiesForPR {
		t.Errorf("expected QualifiesForPR=false when review has blockers and Pending status")
	}
	if bundle.ReviewBlockers != 2 {
		t.Errorf("expected 2 blockers, got %d", bundle.ReviewBlockers)
	}
	if len(bundle.Warnings) == 0 {
		t.Errorf("expected warnings list to contain blocker details")
	}

	// Now test --allow-unreviewed
	bundleOverridden, err := PackageTaskPR(root, PROptions{
		TaskID:          "T-046",
		AllowUnreviewed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bundleOverridden.QualifiesForPR {
		t.Errorf("expected QualifiesForPR=true when AllowUnreviewed=true")
	}
}

func TestPackageTaskPR_BugfixTypology(t *testing.T) {
	root := setupTestTask(t, "bugfix", "T-099", "null-pointer", "Approved", 0)

	bundle, err := PackageTaskPR(root, PROptions{
		TaskID: "T-099",
	})
	if err != nil {
		t.Fatalf("PackageTaskPR failed: %v", err)
	}

	if bundle.Title != "fix(T-099): Fix null pointer" {
		t.Errorf("title = %q, want fix(T-099): Fix null pointer", bundle.Title)
	}
	if !strings.Contains(bundle.Summary, "Null pointer crash") {
		t.Errorf("expected summary from reproduction.md, got: %s", bundle.Summary)
	}
	if !bundle.QualifiesForPR {
		t.Errorf("expected QualifiesForPR=true")
	}
}

func TestPackageTaskPR_HotfixTypology(t *testing.T) {
	root := setupTestTask(t, "hotfix", "T-100", "memory-leak", "", 0)

	bundle, err := PackageTaskPR(root, PROptions{
		TaskID: "T-100",
	})
	if err != nil {
		t.Fatalf("PackageTaskPR failed: %v", err)
	}

	if bundle.Title != "hotfix(T-100): Memory leak patch" {
		t.Errorf("title = %q, want hotfix(T-100): Memory leak patch", bundle.Title)
	}
	if !bundle.QualifiesForPR {
		t.Errorf("expected QualifiesForPR=true for hotfix with patch.md")
	}
}

func TestPackageTaskPR_SpikeTypology(t *testing.T) {
	root := setupTestTask(t, "spike", "T-101", "vector-db", "", 0)

	bundle, err := PackageTaskPR(root, PROptions{
		TaskID: "T-101",
	})
	if err != nil {
		t.Fatalf("PackageTaskPR failed: %v", err)
	}

	if bundle.Title != "spike(T-101): Evaluate vector db" {
		t.Errorf("title = %q, want spike(T-101): Evaluate vector db", bundle.Title)
	}
	if bundle.ReviewStatus != "N/A (spike prototype)" {
		t.Errorf("reviewStatus = %q, want N/A", bundle.ReviewStatus)
	}
	if !bundle.QualifiesForPR {
		t.Errorf("expected QualifiesForPR=true for spike prototype")
	}
}

func TestPackageTaskPR_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := PackageTaskPR(tmpDir, PROptions{
		TaskID: "T-999",
	})
	if err == nil {
		t.Errorf("expected error for non-existent task, got nil")
	}
}

func TestPackageTaskPR_ResolvedBlockerQualifies(t *testing.T) {
	root := setupTestTask(t, "feature", "T-050", "resolved-task", "Approved", 0)
	taskDir := filepath.Join(root, "tasks", "features", "T-050-resolved-task")

	// Overwrite review.md with resolved and closed blockers
	revContent := `# Code Review — T-050
Status: Approved

## 2. Risk & Findings Register

| ID | Category | Severity | Finding | Suggested Action | Status |
|---|---|---|---|---|---|
| SEC-01 | AppSec | 🔴 Blocker | directory traversal risk | sanitize paths | Resolved |
| PAT-01 | Pattern | 🔴 Blocker | unbuffered channel | buffer channel | Closed |
| PAT-02 | Pattern | 🟡 Warning | missing doc comment | add comment | Resolved |
`
	if err := os.WriteFile(filepath.Join(taskDir, "review.md"), []byte(revContent), 0o644); err != nil {
		t.Fatal(err)
	}

	bundle, err := PackageTaskPR(root, PROptions{
		TaskID: "T-050",
	})
	if err != nil {
		t.Fatalf("PackageTaskPR failed: %v", err)
	}

	if bundle.ReviewBlockers != 0 {
		t.Errorf("ReviewBlockers = %d, want 0 for resolved blockers", bundle.ReviewBlockers)
	}
	if !bundle.QualifiesForPR {
		t.Errorf("expected QualifiesForPR=true for task with only resolved blockers, warnings=%v", bundle.Warnings)
	}
}

func TestPackageTaskPR_BranchSanitization(t *testing.T) {
	root := setupTestTask(t, "feature", "T-051", "sanitize-test", "Approved", 0)

	bundle, err := PackageTaskPR(root, PROptions{
		TaskID:     "T-051",
		BaseBranch: "main\r\n`injection`|pipe",
		HeadBranch: "feature/T-051\n",
	})
	if err != nil {
		t.Fatalf("PackageTaskPR failed: %v", err)
	}

	if strings.ContainsAny(bundle.BaseBranch, "\r\n`|") {
		t.Errorf("BaseBranch contains unescaped characters: %q", bundle.BaseBranch)
	}
	if bundle.BaseBranch != "maininjectionpipe" {
		t.Errorf("BaseBranch = %q, want maininjectionpipe", bundle.BaseBranch)
	}
	if bundle.HeadBranch != "feature/T-051" {
		t.Errorf("HeadBranch = %q, want feature/T-051", bundle.HeadBranch)
	}
}

func TestFormatPRMarkdown_BypassedQualification(t *testing.T) {
	bundle := &PRBundle{
		TaskID:         "T-052",
		Typology:       "feature",
		BaseBranch:     "main",
		HeadBranch:     "feature/T-052",
		ReviewStatus:   "Changes requested",
		ReviewBlockers: 1,
		QualifiesForPR: true,
		Warnings:       []string{"1 unresolved 🔴 Blocker(s) found in review.md"},
	}

	md := FormatPRMarkdown(bundle, "Evidence")
	if !strings.Contains(md, "QUALIFIED (Bypassed via --allow-unreviewed)") {
		t.Errorf("expected markdown to indicate bypassed qualification, got:\n%s", md)
	}
}

func TestPackageTaskPR_TokenTelemetry(t *testing.T) {
	root := setupTestTask(t, "feature", "T-055", "token-pr-test", "Approved", 0)

	// Record token telemetry in task state store
	now := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
	_, _, err := state.RecordTokens(root, "T-055", 1200, 5000, 300, now)
	if err != nil {
		t.Fatalf("failed to record tokens: %v", err)
	}

	bundle, err := PackageTaskPR(root, PROptions{
		TaskID: "T-055",
	})
	if err != nil {
		t.Fatalf("PackageTaskPR failed: %v", err)
	}

	if bundle.Tokens == nil {
		t.Fatalf("expected bundle.Tokens to be populated, got nil")
	}
	if bundle.Tokens.InputTokens != 1200 || bundle.Tokens.CacheReadTokens != 5000 || bundle.Tokens.OutputTokens != 300 || bundle.Tokens.TotalTokens != 6500 {
		t.Errorf("unexpected bundle.Tokens: %+v", bundle.Tokens)
	}

	if !strings.Contains(bundle.Body, "## Token Telemetry") {
		t.Errorf("expected body to contain ## Token Telemetry, got:\n%s", bundle.Body)
	}
	if !strings.Contains(bundle.Body, "- **Input Tokens:** 1200") || !strings.Contains(bundle.Body, "- **Total Task Tokens:** 6500") {
		t.Errorf("expected formatted token metrics, got:\n%s", bundle.Body)
	}
}

func TestPackageTaskPR_WithoutTokenTelemetry(t *testing.T) {
	root := setupTestTask(t, "feature", "T-055", "no-token-test", "Approved", 0)

	bundle, err := PackageTaskPR(root, PROptions{
		TaskID: "T-055",
	})
	if err != nil {
		t.Fatalf("PackageTaskPR failed: %v", err)
	}

	if bundle.Tokens != nil {
		t.Errorf("expected bundle.Tokens to be nil, got: %+v", bundle.Tokens)
	}
	if strings.Contains(bundle.Body, "## Token Telemetry") {
		t.Errorf("expected body to NOT contain ## Token Telemetry when absent, got:\n%s", bundle.Body)
	}
}
