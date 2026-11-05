package coordinator

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupQualificationRepo(t *testing.T, reviewStatus string, blockerCount int) (string, string, func(...string)) {
	t.Helper()
	dir, run := setupTestGitRepo(t)

	initialBranch, _ := runGit(dir, "branch", "--show-current")
	if initialBranch == "" {
		initialBranch = "main"
	}

	// Setup go.mod
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test.mod\n\ngo 1.27\n"), 0o644)
	run("add", "go.mod")
	run("commit", "-m", "add go.mod")

	// Setup ROADMAP.md
	roadmapContent := `# ROADMAP — sdd

## 4. Tasks

### M1 — Tasks

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-001 | Feature | S | none | outcome | none | Implementing |
`
	_ = os.WriteFile(filepath.Join(dir, "ROADMAP.md"), []byte(roadmapContent), 0o644)
	run("add", "ROADMAP.md")
	run("commit", "-m", "add roadmap")

	// Create task folder
	taskDir := filepath.Join(dir, "features", "T-001-feature")
	_ = os.MkdirAll(taskDir, 0o755)

	planContent := `# Plan — T-001

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| feature.go | create | code |
`
	_ = os.WriteFile(filepath.Join(taskDir, "plan.md"), []byte(planContent), 0o644)

	reviewContent := "# Code Review — T-001\nStatus: " + reviewStatus + "\n\n## 3. Risk & Findings Register\n\n| ID | Category | Severity | Finding | Suggested Action | Status |\n|---|---|---|---|---|---|\n"
	for i := 0; i < blockerCount; i++ {
		reviewContent += "| F-001 | Bug | 🔴 Blocker | critical bug | fix it | Open |\n"
	}
	_ = os.WriteFile(filepath.Join(taskDir, "review.md"), []byte(reviewContent), 0o644)

	// Create branch feature/T-001-feature
	run("checkout", "-b", "feature/T-001-feature")
	_ = os.WriteFile(filepath.Join(dir, "feature.go"), []byte("package main\n"), 0o644)
	run("add", "feature.go")
	run("commit", "-m", "add feature.go")

	return dir, initialBranch, run
}

func TestQualifyReviewBlockers(t *testing.T) {
	// Setup with 1 blocker and status Approved
	dir, targetBranch, _ := setupQualificationRepo(t, "Approved", 1)

	mockRunner := func(dir string, pkgs []string) (bool, string, error) {
		return true, "ok", nil
	}

	opts := QualifyOptions{
		TargetBranch: targetBranch,
		RunTestFn:    mockRunner,
	}

	rep, err := Qualify(dir, "T-001", opts)
	if err != nil {
		t.Fatalf("Qualify failed: %v", err)
	}

	if rep.ReadyToMerge {
		t.Errorf("expected ReadyToMerge=false when review has blockers")
	}
	if rep.ReviewBlockers != 1 {
		t.Errorf("expected 1 review blocker, got %d", rep.ReviewBlockers)
	}
	foundBlockerMsg := false
	for _, e := range rep.Errors {
		if strings.Contains(e, "1 unresolved 🔴 Blocker(s)") {
			foundBlockerMsg = true
			break
		}
	}
	if !foundBlockerMsg {
		t.Errorf("expected blocker error message in errors list: %+v", rep.Errors)
	}
}

func TestFlakeGuardDetection(t *testing.T) {
	// Setup with 0 blockers and Approved review
	dir, targetBranch, _ := setupQualificationRepo(t, "Approved", 0)

	callCount := 0
	flakyRunner := func(dir string, pkgs []string) (bool, string, error) {
		callCount++
		if callCount == 1 {
			return true, "PASS", nil
		}
		// Second run fails (flake)
		return false, "FAIL: TestFlake timeout", errors.New("exit 1")
	}

	opts := QualifyOptions{
		TargetBranch: targetBranch,
		RunTestFn:    flakyRunner,
	}

	rep, err := Qualify(dir, "T-001", opts)
	if err != nil {
		t.Fatalf("Qualify failed: %v", err)
	}

	if rep.ReadyToMerge {
		t.Errorf("expected ReadyToMerge=false when flake detected")
	}
	if !rep.TestRun1Pass {
		t.Errorf("expected Run 1 to pass")
	}
	if rep.TestRun2Pass {
		t.Errorf("expected Run 2 to fail")
	}
	if !rep.FlakeDetected {
		t.Errorf("expected FlakeDetected to be true")
	}
}

func TestQualifyClean(t *testing.T) {
	dir, targetBranch, _ := setupQualificationRepo(t, "Approved", 0)

	mockRunner := func(dir string, pkgs []string) (bool, string, error) {
		return true, "PASS", nil
	}

	opts := QualifyOptions{
		TargetBranch: targetBranch,
		RunTestFn:    mockRunner,
	}

	rep, err := Qualify(dir, "T-001", opts)
	if err != nil {
		t.Fatalf("Qualify failed: %v", err)
	}

	if !rep.ReadyToMerge {
		t.Fatalf("expected ReadyToMerge=true, got errors: %+v", rep.Errors)
	}
	if rep.ReviewBlockers != 0 {
		t.Errorf("expected 0 blockers, got %d", rep.ReviewBlockers)
	}
	if !rep.MergeClean {
		t.Errorf("expected clean merge simulation")
	}

	reportText := FormatQualificationReport(rep)
	if !strings.Contains(reportText, "READY TO MERGE") {
		t.Errorf("expected READY TO MERGE in report output:\n%s", reportText)
	}

	presentation := FormatGate2Presentation(rep)
	if !strings.Contains(presentation, "APPROVED MERGE T-001") {
		t.Errorf("expected APPROVED MERGE in presentation:\n%s", presentation)
	}
}

func TestRegressionTestSweep(t *testing.T) {
	dir, targetBranch, _ := setupQualificationRepo(t, "Approved", 0)

	var testedPkgs []string
	mockRunner := func(dir string, pkgs []string) (bool, string, error) {
		testedPkgs = pkgs
		return true, "PASS", nil
	}

	opts := QualifyOptions{
		TargetBranch: targetBranch,
		RunTestFn:    mockRunner,
	}

	rep, err := Qualify(dir, "T-001", opts)
	if err != nil {
		t.Fatalf("Qualify failed: %v", err)
	}

	if !rep.TestRun1Pass {
		t.Errorf("expected test run 1 to pass")
	}
	if len(testedPkgs) == 0 {
		t.Errorf("expected tested packages to be non-empty")
	}
}

func TestGate2PresentationFormatting(t *testing.T) {
	rep := &QualificationReport{
		TaskID:         "T-001",
		Typology:       "feature",
		Branch:         "feature/T-001-demo",
		TargetBranch:   "main",
		ReviewStatus:   "Approved",
		ReviewBlockers: 0,
		ModifiedPkgs:   []string{"internal/coordinator"},
		BlastRadius:    []string{"internal/cli"},
		TestRun1Pass:   true,
		TestRun2Pass:   true,
		MergeClean:     true,
		ReadyToMerge:   true,
	}

	text := FormatGate2Presentation(rep)
	if !strings.Contains(text, "Task Coordinator GATE 2 Presentation — T-001") {
		t.Errorf("expected title in presentation:\n%s", text)
	}
	if !strings.Contains(text, "APPROVED MERGE T-001") {
		t.Errorf("expected approval token in presentation:\n%s", text)
	}
	if !strings.Contains(text, "feat(feature): merge feature/T-001-demo into main (T-001)") {
		t.Errorf("expected commit message in presentation:\n%s", text)
	}
}

func TestDeterminism(t *testing.T) {
	dir, targetBranch, _ := setupQualificationRepo(t, "Approved", 0)

	mockRunner := func(dir string, pkgs []string) (bool, string, error) {
		return true, "PASS", nil
	}

	opts := QualifyOptions{
		TargetBranch: targetBranch,
		RunTestFn:    mockRunner,
	}

	rep1, err := Qualify(dir, "T-001", opts)
	if err != nil {
		t.Fatal(err)
	}
	rep2, err := Qualify(dir, "T-001", opts)
	if err != nil {
		t.Fatal(err)
	}

	text1 := FormatQualificationReport(rep1)
	text2 := FormatQualificationReport(rep2)
	if text1 != text2 {
		t.Fatalf("expected deterministic report text output:\n--- 1 ---\n%s\n--- 2 ---\n%s", text1, text2)
	}
}
