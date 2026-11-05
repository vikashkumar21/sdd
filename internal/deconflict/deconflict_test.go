package deconflict

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractPlanFiles(t *testing.T) {
	planContent := `# Plan — T-030 Static task deconfliction

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| ` + "`" + `internal/deconflict/deconflict.go` + "`" + ` | create | Core analyzer |
| ` + "`" + `internal/cli/task.go` + "`" + ` | modify | Add CLI command |
| internal/cli/cli.go | modify | Update usage |
|  ` + "`" + `./README.md` + "`" + `  | modify | Documentation |
| (none) | none | nothing |

## 6. Dependencies
Standard library only.
`

	files := ExtractPlanFiles(planContent)
	expected := []string{
		"README.md",
		"internal/cli/cli.go",
		"internal/cli/task.go",
		"internal/deconflict/deconflict.go",
	}

	if len(files) != len(expected) {
		t.Fatalf("expected %d files, got %d: %+v", len(expected), len(files), files)
	}

	for i := range expected {
		if files[i] != expected[i] {
			t.Errorf("at index %d: expected %s, got %s", i, expected[i], files[i])
		}
	}
}

func TestDisjointTasksParallel(t *testing.T) {
	t1 := TaskPlan{
		TaskID: "T-001",
		Files:  []string{"internal/a.go", "internal/b.go"},
	}
	t2 := TaskPlan{
		TaskID: "T-002",
		Files:  []string{"internal/c.go", "internal/d.go"},
	}

	rep := Analyze([]TaskPlan{t1, t2})
	if rep.HasConflict {
		t.Errorf("expected no conflicts, got %+v", rep.Conflicts)
	}
	if len(rep.SafePairs) != 1 {
		t.Errorf("expected 1 safe pair, got %d", len(rep.SafePairs))
	}
	if len(rep.Batches) != 1 {
		t.Fatalf("expected 1 batch for disjoint tasks, got %d", len(rep.Batches))
	}
	if len(rep.Batches[0]) != 2 {
		t.Errorf("expected both tasks in Batch 1, got %+v", rep.Batches[0])
	}
}

func TestOverlappingTasksSequential(t *testing.T) {
	t1 := TaskPlan{
		TaskID: "T-001",
		Files:  []string{"internal/cli/cli.go", "internal/a.go"},
	}
	t2 := TaskPlan{
		TaskID: "T-002",
		Files:  []string{"internal/cli/cli.go", "internal/b.go"},
	}

	rep := Analyze([]TaskPlan{t1, t2})
	if !rep.HasConflict {
		t.Errorf("expected collision hazard, but none detected")
	}
	if len(rep.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(rep.Conflicts))
	}
	c := rep.Conflicts[0]
	if c.TaskA != "T-001" || c.TaskB != "T-002" {
		t.Errorf("unexpected conflict pair: %s <-> %s", c.TaskA, c.TaskB)
	}
	if len(c.SharedFiles) != 1 || c.SharedFiles[0] != "internal/cli/cli.go" {
		t.Errorf("unexpected shared files: %+v", c.SharedFiles)
	}

	// Should be scheduled in 2 separate sequential waves
	if len(rep.Batches) != 2 {
		t.Fatalf("expected 2 batches for overlapping tasks, got %d: %+v", len(rep.Batches), rep.Batches)
	}
	if rep.Batches[0][0] != "T-001" || rep.Batches[1][0] != "T-002" {
		t.Errorf("unexpected batch scheduling: %+v", rep.Batches)
	}
}

func TestConcurrencyBatchScheduling(t *testing.T) {
	// T1 conflicts with T2
	// T3 is completely disjoint (can run in parallel with T1)
	// T4 conflicts with T3 (must wait for T3)
	t1 := TaskPlan{TaskID: "T-001", Files: []string{"file1.go"}}
	t2 := TaskPlan{TaskID: "T-002", Files: []string{"file1.go", "file2.go"}}
	t3 := TaskPlan{TaskID: "T-003", Files: []string{"file3.go"}}
	t4 := TaskPlan{TaskID: "T-004", Files: []string{"file3.go"}}

	rep := Analyze([]TaskPlan{t1, t2, t3, t4})

	// Batch 1: T-001 and T-003 can run concurrently
	// Batch 2: T-002 and T-004 can run concurrently once Wave 1 completes
	if len(rep.Batches) != 2 {
		t.Fatalf("expected 2 waves, got %d: %+v", len(rep.Batches), rep.Batches)
	}
	if len(rep.Batches[0]) != 2 || rep.Batches[0][0] != "T-001" || rep.Batches[0][1] != "T-003" {
		t.Errorf("expected Batch 1 to have [T-001, T-003], got %+v", rep.Batches[0])
	}
	if len(rep.Batches[1]) != 2 || rep.Batches[1][0] != "T-002" || rep.Batches[1][1] != "T-004" {
		t.Errorf("expected Batch 2 to have [T-002, T-004], got %+v", rep.Batches[1])
	}
}

func TestLoadTaskPlan(t *testing.T) {
	root := t.TempDir()

	// Create feature task folder
	taskDir := filepath.Join(root, "features", "T-001-plan-demo")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	planContent := `# Plan — T-001

## 5. Files to create or modify

| Path | Action | Reason |
|---|---|---|
| ` + "`" + `cmd/main.go` + "`" + ` | modify | entry |
`
	if err := os.WriteFile(filepath.Join(taskDir, "plan.md"), []byte(planContent), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := LoadTaskPlan(root, "T-001")
	if err != nil {
		t.Fatalf("LoadTaskPlan failed: %v", err)
	}
	if plan.TaskID != "T-001" {
		t.Errorf("expected T-001, got %s", plan.TaskID)
	}
	if len(plan.Files) != 1 || plan.Files[0] != "cmd/main.go" {
		t.Errorf("unexpected files: %+v", plan.Files)
	}
}

func TestFormatText(t *testing.T) {
	t1 := TaskPlan{TaskID: "T-001", Files: []string{"a.go"}}
	t2 := TaskPlan{TaskID: "T-002", Files: []string{"a.go"}}

	rep := Analyze([]TaskPlan{t1, t2})
	text := FormatText(rep)

	if !strings.Contains(text, "COLLISION HAZARD") {
		t.Errorf("expected COLLISION HAZARD in text:\n%s", text)
	}
	if !strings.Contains(text, "T-001 ↔ T-002 share 1 file(s):") {
		t.Errorf("expected share count in text:\n%s", text)
	}
	if !strings.Contains(text, "Wave 1") || !strings.Contains(text, "Wave 2") {
		t.Errorf("expected waves in schedule:\n%s", text)
	}
}

func TestEdgeCases(t *testing.T) {
	root := t.TempDir()

	// 1. Task folder with no plan or patch file
	noPlanDir := filepath.Join(root, "features", "T-010-no-plan")
	if err := os.MkdirAll(noPlanDir, 0o755); err != nil {
		t.Fatal(err)
	}
	plan, err := LoadTaskPlan(root, "T-010")
	if err != nil {
		t.Fatalf("expected no error for missing plan, got: %v", err)
	}
	if len(plan.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(plan.Files))
	}

	// 2. Empty table / no files in Section 5
	emptyPlanDir := filepath.Join(root, "features", "T-011-empty-plan")
	if err := os.MkdirAll(emptyPlanDir, 0o755); err != nil {
		t.Fatal(err)
	}
	emptyContent := "# Plan — T-011\n\n## 5. Files to create or modify\n\n| Path | Action | Reason |\n|---|---|---|\n\n## 6. Dependencies\n"
	if err := os.WriteFile(filepath.Join(emptyPlanDir, "plan.md"), []byte(emptyContent), 0o644); err != nil {
		t.Fatal(err)
	}
	planEmpty, err := LoadTaskPlan(root, "T-011")
	if err != nil {
		t.Fatalf("LoadTaskPlan empty: %v", err)
	}
	if len(planEmpty.Files) != 0 {
		t.Errorf("expected 0 files, got %+v", planEmpty.Files)
	}

	// 3. Single task passed to Analyze
	rep := Analyze([]TaskPlan{*planEmpty})
	if rep.HasConflict {
		t.Errorf("single task should have no conflicts")
	}
	if len(rep.Batches) != 1 || len(rep.Batches[0]) != 1 || rep.Batches[0][0] != "T-011" {
		t.Errorf("unexpected batches for single task: %+v", rep.Batches)
	}
}

func TestDeterminism(t *testing.T) {
	t1 := TaskPlan{TaskID: "T-002", Files: []string{"pkg/b.go", "pkg/a.go"}}
	t2 := TaskPlan{TaskID: "T-001", Files: []string{"pkg/a.go", "pkg/c.go"}}

	rep1 := Analyze([]TaskPlan{t1, t2})
	rep2 := Analyze([]TaskPlan{t2, t1})

	text1 := FormatText(rep1)
	text2 := FormatText(rep2)

	if text1 != text2 {
		t.Fatalf("expected deterministic output regardless of input order:\n--- 1 ---\n%s\n--- 2 ---\n%s", text1, text2)
	}
}
