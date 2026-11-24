package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStageDiscoveryBundle_Success(t *testing.T) {
	root := setupTestProject(t, "T-053", "discovery-ensemble")
	fixedNow := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)

	findingsContent := "Discovered 4 services, 12 packages, and 3 schema migrations."
	rubricContent := "# Custom Discovery Rubric\nVector 1: Test Vector"

	bundle, err := StageDiscoveryBundle(root, "T-053", findingsContent, rubricContent, fixedNow)
	if err != nil {
		t.Fatalf("StageDiscoveryBundle failed: %v", err)
	}

	if bundle.TaskID != "T-053" {
		t.Errorf("expected TaskID T-053, got %q", bundle.TaskID)
	}
	if bundle.BundleDir != ".sdd/review/T-053/discovery" {
		t.Errorf("expected BundleDir .sdd/review/T-053/discovery, got %q", bundle.BundleDir)
	}
	if !bundle.CreatedAt.Equal(fixedNow) {
		t.Errorf("expected CreatedAt %v, got %v", fixedNow, bundle.CreatedAt)
	}
	if len(bundle.Files) != 3 {
		t.Fatalf("expected 3 files, got %d: %+v", len(bundle.Files), bundle.Files)
	}

	bundleDirAbs := filepath.Join(root, filepath.FromSlash(bundle.BundleDir))

	// Verify findings.md
	findingsData, err := os.ReadFile(filepath.Join(bundleDirAbs, "findings.md"))
	if err != nil {
		t.Fatalf("missing findings.md: %v", err)
	}
	findingsStr := string(findingsData)
	if !strings.Contains(findingsStr, findingsContent) {
		t.Errorf("findings.md missing findingsContent: %s", findingsStr)
	}
	if !strings.Contains(findingsStr, "Requirements (requirements.md)") {
		t.Errorf("findings.md missing requirements section: %s", findingsStr)
	}

	// Verify rubric.md
	rubricData, err := os.ReadFile(filepath.Join(bundleDirAbs, "rubric.md"))
	if err != nil {
		t.Fatalf("missing rubric.md: %v", err)
	}
	if !strings.Contains(string(rubricData), rubricContent) {
		t.Errorf("rubric.md missing rubricContent: %s", string(rubricData))
	}

	// Verify manifest.json
	manifestData, err := os.ReadFile(filepath.Join(bundleDirAbs, "manifest.json"))
	if err != nil {
		t.Fatalf("missing manifest.json: %v", err)
	}
	var loadedBundle DiscoveryBundle
	if err := json.Unmarshal(manifestData, &loadedBundle); err != nil {
		t.Fatalf("failed to unmarshal manifest.json: %v", err)
	}
	if loadedBundle.TaskID != "T-053" || len(loadedBundle.Files) < 2 {
		t.Errorf("unexpected loaded manifest: %+v", loadedBundle)
	}

	// Verify .gitignore was updated
	gitIgnoreData, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}
	if !strings.Contains(string(gitIgnoreData), ".sdd/review/") {
		t.Errorf(".gitignore missing .sdd/review/ entry: %s", string(gitIgnoreData))
	}
}

func TestStageDiscoveryBundle_WithContext(t *testing.T) {
	root := setupTestProject(t, "T-053", "discovery-ensemble")
	fixedNow := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)

	// Stage sample context files in .sdd/context/
	ctxDir := filepath.Join(root, ".sdd", "context")
	if err := os.MkdirAll(filepath.Join(ctxDir, "ref"), 0755); err != nil {
		t.Fatalf("failed to create context dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "migration.md"), []byte("# Migration Guide\nPlatform details"), 0644); err != nil {
		t.Fatalf("failed to write context file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "ref", "spec.json"), []byte(`{"version": 2}`), 0644); err != nil {
		t.Fatalf("failed to write nested context file: %v", err)
	}

	bundle, err := StageDiscoveryBundle(root, "T-053", "Findings report", "# Rubric", fixedNow)
	if err != nil {
		t.Fatalf("StageDiscoveryBundle with context failed: %v", err)
	}

	// Expect 5 files: findings.md, rubric.md, context/migration.md, context/ref/spec.json, manifest.json
	if len(bundle.Files) != 5 {
		t.Fatalf("expected 5 staged files, got %d: %+v", len(bundle.Files), bundle.Files)
	}

	bundleDirAbs := filepath.Join(root, filepath.FromSlash(bundle.BundleDir))
	stagedMigPath := filepath.Join(bundleDirAbs, "context", "migration.md")
	migData, err := os.ReadFile(stagedMigPath)
	if err != nil {
		t.Fatalf("missing staged context migration.md: %v", err)
	}
	if !strings.Contains(string(migData), "Platform details") {
		t.Errorf("unexpected content in staged migration.md: %s", string(migData))
	}

	stagedSpecPath := filepath.Join(bundleDirAbs, "context", "ref", "spec.json")
	specData, err := os.ReadFile(stagedSpecPath)
	if err != nil {
		t.Fatalf("missing staged nested context ref/spec.json: %v", err)
	}
	if !strings.Contains(string(specData), `"version": 2`) {
		t.Errorf("unexpected content in staged ref/spec.json: %s", string(specData))
	}
}

func TestStageDiscoveryBundle_DefaultFallback(t *testing.T) {
	root := setupTestProject(t, "T-053", "discovery-ensemble")
	fixedNow := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)

	// Empty findingsContent and rubricContent -> should read task docs and resolve discovery-challenger skill
	bundle, err := StageDiscoveryBundle(root, "t-053", "", "", fixedNow)
	if err != nil {
		t.Fatalf("StageDiscoveryBundle default fallback failed: %v", err)
	}
	if !bundle.CreatedAt.Equal(fixedNow) {
		t.Errorf("expected CreatedAt %v, got %v", fixedNow, bundle.CreatedAt)
	}

	bundleDirAbs := filepath.Join(root, filepath.FromSlash(bundle.BundleDir))

	findingsData, err := os.ReadFile(filepath.Join(bundleDirAbs, "findings.md"))
	if err != nil {
		t.Fatalf("missing findings.md: %v", err)
	}
	if !strings.Contains(string(findingsData), "Test goal for T-053") {
		t.Errorf("expected findings.md to contain task requirements goal: %s", string(findingsData))
	}

	rubricData, err := os.ReadFile(filepath.Join(bundleDirAbs, "rubric.md"))
	if err != nil {
		t.Fatalf("missing rubric.md: %v", err)
	}
	if !strings.Contains(string(rubricData), "name: discovery-challenger") {
		t.Errorf("expected rubric.md to resolve discovery-challenger role skill: %s", string(rubricData))
	}
}

func TestStageDiscoveryBundle_InvalidTaskID(t *testing.T) {
	root := t.TempDir()
	_, err := StageDiscoveryBundle(root, "INVALID", "", "")
	if err == nil {
		t.Fatalf("expected error for invalid task ID, got nil")
	}
	if !strings.Contains(err.Error(), "invalid task ID format") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestStageDiscoveryBundle_TaskNotFound(t *testing.T) {
	root := t.TempDir()
	_, err := StageDiscoveryBundle(root, "T-999", "", "")
	if err == nil {
		t.Fatalf("expected error for non-existent task, got nil")
	}
}

func TestStageDiscoveryBundle_SymlinkEscape(t *testing.T) {
	// 1. Test symlinked bundle directory escaping root
	root1 := setupTestProject(t, "T-053", "discovery-ensemble")
	outsideDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root1, ".sdd"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideDir, filepath.Join(root1, ".sdd", "review")); err != nil {
		t.Skip("symlinks not supported on this platform")
	}
	_, err := StageDiscoveryBundle(root1, "T-053", "", "")
	if err == nil {
		t.Fatalf("expected error when .sdd/review is a symlink escaping root, got nil")
	}

	// 2. Test symlinked task directory escaping root
	root2 := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root2, "tasks", "features"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideDir, filepath.Join(root2, "tasks", "features", "T-053-symlink-task")); err != nil {
		t.Skip("symlinks not supported on this platform")
	}
	_, err = StageDiscoveryBundle(root2, "T-053", "", "")
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink rejection for task directory, got: %v", err)
	}
}

func TestExtractRequirements(t *testing.T) {
	mdContent := `# Requirements — T-057
Status: Active

## 4. Functional requirements
| ID | Requirement (SHALL …) | Priority | Verification |
|---|---|---|---|
| R-001 | The system SHALL define ledger schema. | MUST | test / inspection |
| R-002 | The system SHALL extract requirements and stage ledger.md. | MUST | test |
| R-003 | The system SHALL enforce omission-first review protocol. | MUST | inspection |
| R-010 | The system SHALL handle edge cases gracefully. | SHOULD | test |
`

	reqs, err := ExtractRequirements(mdContent)
	if err != nil {
		t.Fatalf("ExtractRequirements failed: %v", err)
	}
	if len(reqs) != 4 {
		t.Fatalf("expected 4 requirements, got %d", len(reqs))
	}

	expectedIDs := []string{"R-001", "R-002", "R-003", "R-010"}
	for i, expID := range expectedIDs {
		if reqs[i].ID != expID {
			t.Errorf("reqs[%d].ID = %q, want %q", i, reqs[i].ID, expID)
		}
	}
	if reqs[0].Statement != "The system SHALL define ledger schema." {
		t.Errorf("reqs[0].Statement = %q", reqs[0].Statement)
	}
	if reqs[0].Priority != "MUST" {
		t.Errorf("reqs[0].Priority = %q", reqs[0].Priority)
	}
	if reqs[0].Verification != "test / inspection" {
		t.Errorf("reqs[0].Verification = %q", reqs[0].Verification)
	}
}

func TestExtractRequirements_EmptyAndMalformed(t *testing.T) {
	// Empty content
	reqs, err := ExtractRequirements("")
	if err != nil {
		t.Fatalf("expected nil error on empty content, got: %v", err)
	}
	if len(reqs) != 0 {
		t.Errorf("expected 0 reqs on empty content, got %d", len(reqs))
	}

	// Content without R-xxx
	noReqContent := `# Notes
| Feature | Status |
|---|---|
| Core | Active |
`
	reqs, err = ExtractRequirements(noReqContent)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if len(reqs) != 0 {
		t.Errorf("expected 0 reqs, got %d", len(reqs))
	}

	// Duplicate requirements & ordering
	dupContent := `
| ID | Statement | Priority | Verification |
|---|---|---|---|
| R-003 | Third requirement | MUST | test |
| R-001 | First requirement | MUST | test |
| R-001 | First requirement duplicate | MUST | test |
| R-002 | Second requirement | MUST | test |
`
	reqs, err = ExtractRequirements(dupContent)
	if err != nil {
		t.Fatalf("ExtractRequirements failed: %v", err)
	}
	if len(reqs) != 3 {
		t.Fatalf("expected 3 unique requirements, got %d", len(reqs))
	}
	if reqs[0].ID != "R-001" || reqs[1].ID != "R-002" || reqs[2].ID != "R-003" {
		t.Errorf("expected sorted unique requirements, got: %+v", reqs)
	}
}

func TestFormatTraceabilityLedger(t *testing.T) {
	reqs := []RequirementItem{
		{ID: "R-001", Statement: "System SHALL do A."},
		{ID: "R-002", Statement: "System SHALL support B | C."},
	}

	ledger := FormatTraceabilityLedger(reqs)
	if !strings.Contains(ledger, "# Requirements Traceability Ledger") {
		t.Errorf("ledger missing header:\n%s", ledger)
	}
	if !strings.Contains(ledger, "| Req ID | Requirement Statement | Code Symbol & File | Positive Test Evidence | Negative / Boundary Test Evidence | Status |") {
		t.Errorf("ledger missing table header:\n%s", ledger)
	}
	if !strings.Contains(ledger, "| R-001 | System SHALL do A. | | | | Open |") {
		t.Errorf("ledger missing R-001 row:\n%s", ledger)
	}
	// Pipe should be escaped
	if !strings.Contains(ledger, `| R-002 | System SHALL support B \| C. | | | | Open |`) {
		t.Errorf("ledger missing escaped pipe in R-002 row:\n%s", ledger)
	}
}

func TestStageReviewBundle_LedgerStaged(t *testing.T) {
	root := setupTestProject(t, "T-099", "sample-task")
	fixedTime := time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)

	bundle, err := StageReviewBundle(root, "T-099", "spec", fixedTime)
	if err != nil {
		t.Fatalf("StageReviewBundle failed: %v", err)
	}

	ledgerPath := filepath.Join(root, bundle.BundleDir, "ledger.md")
	data, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatalf("missing staged ledger.md: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "# Requirements Traceability Ledger") {
		t.Errorf("staged ledger.md missing header: %s", content)
	}
	if !strings.Contains(content, "| R-001 | System SHALL pass | | | | Open |") {
		t.Errorf("staged ledger.md missing R-001 row: %s", content)
	}

	// Verify manifest includes ledger.md
	manifestPath := filepath.Join(root, bundle.BundleDir, "manifest.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("reading manifest.json: %v", err)
	}
	if !strings.Contains(string(manifestBytes), "ledger.md") {
		t.Errorf("manifest.json missing ledger.md: %s", string(manifestBytes))
	}

	// Verify prompt mentions ledger.md and unanchored step
	prompt, err := FormatReviewPrompt(bundle, root)
	if err != nil {
		t.Fatalf("FormatReviewPrompt failed: %v", err)
	}
	if !strings.Contains(prompt, "Traceability Ledger") || !strings.Contains(prompt, "ledger.md") {
		t.Errorf("prompt missing ledger reference:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Unanchored Traceability Scan (Step 1)") {
		t.Errorf("prompt missing Unanchored Traceability Scan step:\n%s", prompt)
	}
}
