package audit

import (
	"strings"
	"testing"
)

const sampleGitDiff = `diff --git a/cmd/sdd/main.go b/cmd/sdd/main.go
index 1111111..2222222 100644
--- a/cmd/sdd/main.go
+++ b/cmd/sdd/main.go
@@ -10,3 +10,4 @@ func main() {
+	// new command dispatch
 }
diff --git a/ROADMAP.md b/ROADMAP.md
index 3333333..4444444 100644
--- a/ROADMAP.md
+++ b/ROADMAP.md
@@ -200,3 +200,15 @@ Change log
+| 2026-11-25 | T-061 update: lens-scoped review bundles and diff filtering |
+| 2026-11-25 | Milestone 15 token consumption optimization and caching |
+| 2026-11-25 | Subagent model tier routing and calibrated reasoning dispatch |
+| 2026-11-25 | Heterogeneous role routing in roles.json |
diff --git a/internal/audit/bundle.go b/internal/audit/bundle.go
index 5555555..6666666 100644
--- a/internal/audit/bundle.go
+++ b/internal/audit/bundle.go
@@ -50,3 +50,5 @@ func StageReviewBundle() {
+	// lens-scoped directory
 }
diff --git a/tasks/features/T-061-sample/validation.md b/tasks/features/T-061-sample/validation.md
index 7777777..8888888 100644
--- a/tasks/features/T-061-sample/validation.md
+++ b/tasks/features/T-061-sample/validation.md
@@ -10,3 +10,4 @@ Results
+| R-001 | PASS |
`

func TestFilterGitDiffByLens_Security(t *testing.T) {
	filtered, rawBytes, filteredBytes, rawFiles, filteredFiles, savings := FilterGitDiffByLens(sampleGitDiff, "security")

	if rawFiles != 4 {
		t.Fatalf("expected 4 raw files, got %d", rawFiles)
	}
	if filteredFiles != 2 {
		t.Fatalf("expected 2 filtered files for security lens, got %d", filteredFiles)
	}
	if !strings.Contains(filtered, "cmd/sdd/main.go") {
		t.Errorf("security diff missing cmd/sdd/main.go:\n%s", filtered)
	}
	if !strings.Contains(filtered, "internal/audit/bundle.go") {
		t.Errorf("security diff missing internal/audit/bundle.go:\n%s", filtered)
	}
	if strings.Contains(filtered, "ROADMAP.md") {
		t.Errorf("security diff must NOT contain ROADMAP.md:\n%s", filtered)
	}
	if strings.Contains(filtered, "tasks/features/T-061-sample/validation.md") {
		t.Errorf("security diff must NOT contain task validation doc:\n%s", filtered)
	}
	if filteredBytes >= rawBytes {
		t.Errorf("expected filteredBytes (%d) < rawBytes (%d)", filteredBytes, rawBytes)
	}
	if savings <= 0.0 {
		t.Errorf("expected positive savings percent, got %f", savings)
	}
}

func TestFilterGitDiffByLens_AppsecAlias(t *testing.T) {
	filtered, _, _, _, filteredFiles, _ := FilterGitDiffByLens(sampleGitDiff, "appsec")
	if filteredFiles != 2 {
		t.Fatalf("expected 2 filtered files for appsec alias, got %d", filteredFiles)
	}
	if !strings.Contains(filtered, "cmd/sdd/main.go") {
		t.Errorf("appsec diff missing cmd/sdd/main.go:\n%s", filtered)
	}
}

func TestFilterGitDiffByLens_Pattern(t *testing.T) {
	filtered, _, _, rawFiles, filteredFiles, savings := FilterGitDiffByLens(sampleGitDiff, "pattern")

	if rawFiles != 4 {
		t.Fatalf("expected 4 raw files, got %d", rawFiles)
	}
	if filteredFiles != 2 {
		t.Fatalf("expected 2 filtered files for pattern lens, got %d", filteredFiles)
	}
	if !strings.Contains(filtered, "cmd/sdd/main.go") || !strings.Contains(filtered, "internal/audit/bundle.go") {
		t.Errorf("pattern diff missing core Go source files:\n%s", filtered)
	}
	if strings.Contains(filtered, "tasks/features/T-061-sample/validation.md") {
		t.Errorf("pattern diff must NOT contain task validation logs:\n%s", filtered)
	}
	if savings <= 0.0 {
		t.Errorf("expected positive savings percent, got %f", savings)
	}
}

func TestFilterGitDiffByLens_Spec(t *testing.T) {
	filtered, _, _, rawFiles, filteredFiles, savings := FilterGitDiffByLens(sampleGitDiff, "spec")

	if rawFiles != 4 {
		t.Fatalf("expected 4 raw files, got %d", rawFiles)
	}
	if filteredFiles != 4 {
		t.Fatalf("expected 4 files processed for spec boundary check, got %d", filteredFiles)
	}
	// Verify ROADMAP.md header is present for boundary verification, but diff body is elided
	if !strings.Contains(filtered, "diff --git a/ROADMAP.md b/ROADMAP.md") {
		t.Errorf("spec diff must retain file header for boundary verification:\n%s", filtered)
	}
	if strings.Contains(filtered, "T-061 update: lens-scoped review bundles") {
		t.Errorf("spec diff must elide non-code diff body:\n%s", filtered)
	}
	if !strings.Contains(filtered, "spec lens: non-code diff body elided") {
		t.Errorf("spec diff missing elision notice:\n%s", filtered)
	}
	if savings <= 0.0 {
		t.Errorf("expected positive savings percent from eliding doc diff body, got %f", savings)
	}
}

func TestFilterGitDiffByLens_All(t *testing.T) {
	filtered, rawBytes, filteredBytes, rawFiles, filteredFiles, savings := FilterGitDiffByLens(sampleGitDiff, "all")
	if filtered != sampleGitDiff {
		t.Errorf("lens 'all' must preserve byte-identical diff")
	}
	if filteredBytes != rawBytes || filteredFiles != rawFiles || savings != 0.0 {
		t.Errorf("lens 'all' must yield 0 savings: bytes=%d/%d, files=%d/%d, savings=%f", filteredBytes, rawBytes, filteredFiles, rawFiles, savings)
	}
}

func TestFilterGitDiffByLens_ZeroDivision(t *testing.T) {
	filtered, rawBytes, filteredBytes, rawFiles, filteredFiles, savings := FilterGitDiffByLens("", "security")
	if rawBytes != 0 || filteredBytes != 0 || rawFiles != 0 || filteredFiles != 0 || savings != 0.0 {
		t.Errorf("empty diff should evaluate safely to 0: rawBytes=%d, savings=%f", rawBytes, savings)
	}
	if filtered != "" {
		t.Errorf("expected empty string for empty input, got %q", filtered)
	}
}

func TestFilterGitDiffByLens_100PercentOutOfScope(t *testing.T) {
	docOnlyDiff := `diff --git a/README.md b/README.md
index 1111111..2222222 100644
--- a/README.md
+++ b/README.md
@@ -1,3 +1,4 @@
+# SDD Documentation update
`
	filtered, _, _, rawFiles, filteredFiles, savings := FilterGitDiffByLens(docOnlyDiff, "security")
	if rawFiles != 1 {
		t.Fatalf("expected 1 raw file, got %d", rawFiles)
	}
	if filteredFiles != 0 {
		t.Fatalf("expected 0 filtered files for security lens, got %d", filteredFiles)
	}
	if !strings.Contains(filtered, "all changes in this task are outside security lens focus") {
		t.Errorf("expected fail-safe out-of-scope note, got:\n%s", filtered)
	}
	_ = savings
}

func TestFilterFindingsForLens(t *testing.T) {
	findings := []FindingItem{
		{ID: "SPEC-001", Category: "Spec", Finding: "Missing R-002 check"},
		{ID: "SEC-001", Category: "Security", Finding: "Unbounded buffer read"},
		{ID: "PAT-001", Category: "Pattern", Finding: "Missing context first"},
		{ID: "F-001", Category: "General", Finding: "Resolve all findings"},
	}

	secFindings := FilterFindingsForLens(findings, "security")
	if len(secFindings) != 2 {
		t.Fatalf("expected 2 findings for security (SEC-001 and F-001), got %d", len(secFindings))
	}
	if secFindings[0].ID != "SEC-001" || secFindings[1].ID != "F-001" {
		t.Errorf("unexpected security findings: %+v", secFindings)
	}

	specFindings := FilterFindingsForLens(findings, "spec")
	if len(specFindings) != 2 {
		t.Fatalf("expected 2 findings for spec (SPEC-001 and F-001), got %d", len(specFindings))
	}
	if specFindings[0].ID != "SPEC-001" || specFindings[1].ID != "F-001" {
		t.Errorf("unexpected spec findings: %+v", specFindings)
	}

	patFindings := FilterFindingsForLens(findings, "pattern")
	if len(patFindings) != 2 {
		t.Fatalf("expected 2 findings for pattern (PAT-001 and F-001), got %d", len(patFindings))
	}
	if patFindings[0].ID != "PAT-001" || patFindings[1].ID != "F-001" {
		t.Errorf("unexpected pattern findings: %+v", patFindings)
	}

	allFindings := FilterFindingsForLens(findings, "all")
	if len(allFindings) != 4 {
		t.Fatalf("expected 4 findings for 'all', got %d", len(allFindings))
	}
}

func TestFilterGitDiffByLens_OutOfScopeFallback(t *testing.T) {
	TestFilterGitDiffByLens_100PercentOutOfScope(t)
}

func TestFilterGitDiffByLens_MalformedFallback(t *testing.T) {
	malformedDiff := "this is not a valid diff header\njust some random text without diff git prefix\n"
	filtered, rawBytes, filteredBytes, rawFiles, filteredFiles, savings := FilterGitDiffByLens(malformedDiff, "security")
	if filtered != malformedDiff {
		t.Errorf("malformed diff must safely fallback to rawDiff, got:\n%s", filtered)
	}
	if rawFiles != 1 || filteredFiles != 1 || rawBytes != filteredBytes || savings != 0.0 {
		t.Errorf("unexpected fallback metrics: rawFiles=%d, filteredFiles=%d, savings=%f", rawFiles, filteredFiles, savings)
	}
}

func TestFilterGitDiffByLens_UnknownLensFallback(t *testing.T) {
	filtered, _, _, rawFiles, filteredFiles, _ := FilterGitDiffByLens(sampleGitDiff, "unrecognized_custom_lens")
	if rawFiles != 4 || filteredFiles != 4 {
		t.Errorf("unrecognized lens should retain all hunks: raw=%d, filtered=%d", rawFiles, filteredFiles)
	}
	if filtered != sampleGitDiff {
		t.Errorf("unrecognized lens must preserve full diff without error")
	}
}

func TestFilterGitDiffByLens_DeletedFile(t *testing.T) {
	deletedDiff := `diff --git a/cmd/sdd/old_cmd.go b/dev/null
deleted file mode 100644
index 1111111..0000000
--- a/cmd/sdd/old_cmd.go
+++ /dev/null
@@ -1,5 +0,0 @@
-package main
`
	filtered, _, _, rawFiles, filteredFiles, _ := FilterGitDiffByLens(deletedDiff, "security")
	if rawFiles != 1 || filteredFiles != 1 {
		t.Fatalf("expected deleted file to be evaluated via OldPath: raw=%d, filtered=%d", rawFiles, filteredFiles)
	}
	if !strings.Contains(filtered, "diff --git a/cmd/sdd/old_cmd.go") {
		t.Errorf("expected deleted file diff to be retained for security lens:\n%s", filtered)
	}
}

func TestFilterGitDiffByLens_SpacesInPath(t *testing.T) {
	spaceDiff := `diff --git "a/internal/audit/path with space.go" "b/internal/audit/path with space.go"
index 1111111..2222222 100644
--- "a/internal/audit/path with space.go"
+++ "b/internal/audit/path with space.go"
@@ -1,2 +1,3 @@
+// added line
`
	filtered, _, _, rawFiles, filteredFiles, _ := FilterGitDiffByLens(spaceDiff, "security")
	if rawFiles != 1 || filteredFiles != 1 {
		t.Fatalf("expected space path to be parsed correctly: raw=%d, filtered=%d", rawFiles, filteredFiles)
	}
	if !strings.Contains(filtered, "path with space.go") {
		t.Errorf("expected space path diff to be retained:\n%s", filtered)
	}
}

func TestFilterFindingsForLens_Unassigned(t *testing.T) {
	findings := []FindingItem{
		{ID: "F-001", Category: "", Finding: "Missing category"},
		{ID: "GEN-001", Category: "General", Finding: "General feedback"},
		{ID: "MISC-002", Category: "n/a", Finding: "Not applicable"},
		{ID: "SPEC-003", Category: "Spec", Finding: "Spec finding"},
	}
	secFindings := FilterFindingsForLens(findings, "security")
	if len(secFindings) != 3 {
		t.Fatalf("expected 3 unassigned/general findings for security, got %d", len(secFindings))
	}
	for _, f := range secFindings {
		if f.ID == "SPEC-003" {
			t.Errorf("expected SPEC-003 to be excluded from security lens")
		}
	}
}
