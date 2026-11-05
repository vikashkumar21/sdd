package status_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vikashkumar21/sdd/internal/status"
)

const sampleRoadmap = `# ROADMAP — sdd

## 3. Milestones

| Milestone | Goal | Tasks | Est. hours | Exit criteria |
|---|---|---|---|---|
| M0 | Scaffolding | T-000 | 1 | Done |
| M1 | Core CLI | T-001 | 4 | Done |
| M2 | Ergonomics | T-002, T-003 | 5 | In progress |

## 4. Tasks

### M0 — Scaffolding

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-000 | Project setup | S | none | Scaffolding reviewed | D-001 | Done |

### M1 — Core CLI

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-001 | Basic commands | M | T-000 | Commands implemented | D-002 | Done |

### M2 — Ergonomics

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-002 | Status command | S | T-001 | Fast CLI summary | none | Specifying |
| T-003 | Autocomplete | S | T-002 | Shell completion | none | Not started |
`

func setupSampleProject(t *testing.T, roadmapContent string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ROADMAP.md"), []byte(roadmapContent), 0o644); err != nil {
		t.Fatalf("writing ROADMAP.md: %v", err)
	}
	return dir
}

func TestScanActiveTasksAndGates(t *testing.T) {
	dir := setupSampleProject(t, sampleRoadmap)

	ps, err := status.Scan(dir)
	if err != nil {
		t.Fatalf("unexpected Scan error: %v", err)
	}

	if ps.TaskSummary.Total != 4 {
		t.Errorf("expected 4 total tasks, got %d", ps.TaskSummary.Total)
	}
	if ps.TaskSummary.Done != 2 {
		t.Errorf("expected 2 done tasks, got %d", ps.TaskSummary.Done)
	}
	if ps.TaskSummary.InProgress != 1 {
		t.Errorf("expected 1 in progress task, got %d", ps.TaskSummary.InProgress)
	}
	if ps.TaskSummary.NotStarted != 1 {
		t.Errorf("expected 1 not started task, got %d", ps.TaskSummary.NotStarted)
	}

	if len(ps.ActiveTasks) != 1 {
		t.Fatalf("expected 1 active task, got %d", len(ps.ActiveTasks))
	}

	active := ps.ActiveTasks[0]
	if active.ID != "T-002" {
		t.Errorf("expected active task ID T-002, got %s", active.ID)
	}
	if active.Phase != "Phase A (Specify)" {
		t.Errorf("expected Phase A (Specify), got %s", active.Phase)
	}
	if active.Gate == nil || active.Gate.Gate != "GATE 1" {
		t.Errorf("expected GATE 1 pending, got %#v", active.Gate)
	}
	if !strings.Contains(active.Gate.Action, "APPROVED T-002") {
		t.Errorf("expected GATE 1 action to mention APPROVED T-002, got %q", active.Gate.Action)
	}

	if ps.NextGate == nil || ps.NextGate.Gate != "GATE 1" {
		t.Errorf("expected NextGate to be GATE 1, got %#v", ps.NextGate)
	}

	// Verify Milestones
	if len(ps.Milestones) != 3 {
		t.Fatalf("expected 3 milestones, got %d", len(ps.Milestones))
	}
	if ps.Milestones[0].ID != "M0" || ps.Milestones[0].Status != "Done" {
		t.Errorf("expected M0 Done, got %#v", ps.Milestones[0])
	}
	if ps.Milestones[2].ID != "M2" || ps.Milestones[2].Status != "In progress" {
		t.Errorf("expected M2 In progress, got %#v", ps.Milestones[2])
	}
	if ps.ActiveMilestone == nil || ps.ActiveMilestone.ID != "M2" {
		t.Errorf("expected ActiveMilestone M2, got %#v", ps.ActiveMilestone)
	}
}

func TestScanGate2WhenAwaitingReview(t *testing.T) {
	roadmap := strings.Replace(sampleRoadmap, "Specifying", "Awaiting review", 1)
	dir := setupSampleProject(t, roadmap)

	ps, err := status.Scan(dir)
	if err != nil {
		t.Fatalf("unexpected Scan error: %v", err)
	}

	if len(ps.ActiveTasks) != 1 {
		t.Fatalf("expected 1 active task, got %d", len(ps.ActiveTasks))
	}
	active := ps.ActiveTasks[0]
	if active.Phase != "Phase D (Close)" {
		t.Errorf("expected Phase D (Close), got %s", active.Phase)
	}
	if active.Gate == nil || active.Gate.Gate != "GATE 2" {
		t.Errorf("expected GATE 2, got %#v", active.Gate)
	}
	if !strings.Contains(active.Gate.Action, "COMMITTED T-002") {
		t.Errorf("expected action to mention COMMITTED T-002, got %q", active.Gate.Action)
	}
}

func TestScanIdleRepository(t *testing.T) {
	roadmap := strings.Replace(sampleRoadmap, "Specifying", "Done", 1)
	dir := setupSampleProject(t, roadmap)

	ps, err := status.Scan(dir)
	if err != nil {
		t.Fatalf("unexpected Scan error: %v", err)
	}

	if len(ps.ActiveTasks) != 0 {
		t.Errorf("expected 0 active tasks, got %d", len(ps.ActiveTasks))
	}
	if ps.NextTask == nil || ps.NextTask.ID != "T-003" {
		t.Errorf("expected NextTask to be T-003, got %#v", ps.NextTask)
	}
}

func TestScanMissingRoadmap(t *testing.T) {
	dir := t.TempDir()
	_, err := status.Scan(dir)
	if err == nil {
		t.Fatal("expected error when ROADMAP.md is missing, got nil")
	}
	if !strings.Contains(err.Error(), "ROADMAP.md not found") {
		t.Errorf("expected 'ROADMAP.md not found' error message, got: %v", err)
	}
}

func TestScanReviewBlockersWarning(t *testing.T) {
	dir := setupSampleProject(t, sampleRoadmap)

	// Create feature folder and review.md with an unresolved 🔴 Blocker
	featDir := filepath.Join(dir, "features", "T-002-status-command")
	if err := os.MkdirAll(featDir, 0o755); err != nil {
		t.Fatalf("mkdir feature dir: %v", err)
	}
	reviewContent := `# Review
| ID | Category | Severity | Finding | Suggested Action | Status |
|---|---|---|---|---|---|
| F-001 | Correctness | 🔴 Blocker | Missing edge case test | Add negative test | Open |
`
	if err := os.WriteFile(filepath.Join(featDir, "review.md"), []byte(reviewContent), 0o644); err != nil {
		t.Fatalf("write review.md: %v", err)
	}

	ps, err := status.Scan(dir)
	if err != nil {
		t.Fatalf("unexpected Scan error: %v", err)
	}

	if len(ps.ActiveTasks) != 1 {
		t.Fatalf("expected 1 active task, got %d", len(ps.ActiveTasks))
	}
	if ps.ActiveTasks[0].ReviewBlockers != 1 {
		t.Errorf("expected 1 review blocker, got %d", ps.ActiveTasks[0].ReviewBlockers)
	}

	foundWarning := false
	for _, w := range ps.Warnings {
		if strings.Contains(w, "unresolved 🔴 Blocker") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Errorf("expected review blocker warning in warnings list, got: %v", ps.Warnings)
	}
}

func TestScanFactExpiryWarning(t *testing.T) {
	dir := setupSampleProject(t, sampleRoadmap)

	// In sampleRoadmap, M0 and M1 are Done. Add TECH_STACK.md with an expired fact referencing M0.
	techStackContent := `# TECH_STACK

## 7. Verified facts log

| Date | Fact | Source | Re-verify at |
|---|---|---|---|
| 2026-10-01 | Old fact that needs reverification | test | M0 |
`
	if err := os.WriteFile(filepath.Join(dir, "TECH_STACK.md"), []byte(techStackContent), 0o644); err != nil {
		t.Fatalf("write TECH_STACK.md: %v", err)
	}

	ps, err := status.Scan(dir)
	if err != nil {
		t.Fatalf("unexpected Scan error: %v", err)
	}

	foundFactWarning := false
	for _, w := range ps.Warnings {
		if strings.Contains(w, "past due for re-verification") && strings.Contains(w, "milestone M0 is completed") {
			foundFactWarning = true
			break
		}
	}
	if !foundFactWarning {
		t.Errorf("expected fact expiry warning, got: %v", ps.Warnings)
	}
}

func TestMultipleActiveTasksWarning(t *testing.T) {
	// Set both T-002 and T-003 to Specifying
	roadmap := strings.Replace(sampleRoadmap, "Not started", "Specifying", 1)
	dir := setupSampleProject(t, roadmap)

	ps, err := status.Scan(dir)
	if err != nil {
		t.Fatalf("unexpected Scan error: %v", err)
	}

	if len(ps.ActiveTasks) != 2 {
		t.Fatalf("expected 2 active tasks, got %d", len(ps.ActiveTasks))
	}

	foundMultiWarning := false
	for _, w := range ps.Warnings {
		if strings.Contains(w, "Multiple tasks are active simultaneously") {
			foundMultiWarning = true
			break
		}
	}
	if !foundMultiWarning {
		t.Errorf("expected multiple active tasks warning, got: %v", ps.Warnings)
	}
}

func TestRenderTextAndJSON(t *testing.T) {
	dir := setupSampleProject(t, sampleRoadmap)

	ps, err := status.Scan(dir)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	text := status.RenderText(ps)
	if !strings.Contains(text, "Project:") {
		t.Errorf("expected RenderText to contain 'Project:', got:\n%s", text)
	}
	if !strings.Contains(text, "Active Task:") {
		t.Errorf("expected RenderText to contain 'Active Task:', got:\n%s", text)
	}
	if !strings.Contains(text, "GATE 1") {
		t.Errorf("expected RenderText to contain 'GATE 1', got:\n%s", text)
	}
	if !strings.Contains(text, "Roadmap Milestones:") {
		t.Errorf("expected RenderText to contain 'Roadmap Milestones:', got:\n%s", text)
	}

	jsonData, err := status.RenderJSON(ps)
	if err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}

	var parsed status.ProjectStatus
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if parsed.TaskSummary.Total != 4 {
		t.Errorf("expected parsed total 4, got %d", parsed.TaskSummary.Total)
	}
}

func TestDeterminism(t *testing.T) {
	dir := setupSampleProject(t, sampleRoadmap)

	ps1, err := status.Scan(dir)
	if err != nil {
		t.Fatalf("Scan 1 error: %v", err)
	}
	json1, err := status.RenderJSON(ps1)
	if err != nil {
		t.Fatalf("JSON 1 error: %v", err)
	}

	ps2, err := status.Scan(dir)
	if err != nil {
		t.Fatalf("Scan 2 error: %v", err)
	}
	json2, err := status.RenderJSON(ps2)
	if err != nil {
		t.Fatalf("JSON 2 error: %v", err)
	}

	if string(json1) != string(json2) {
		t.Errorf("non-deterministic output:\nRun 1: %s\nRun 2: %s", string(json1), string(json2))
	}
}

func TestStatusTypologies(t *testing.T) {
	roadmapContent := `# ROADMAP — sdd

## 3. Milestones

| Milestone | Goal | Tasks | Est. hours | Exit criteria |
|---|---|---|---|---|
| M1 | Multi-Typology | T-001, T-002, T-003, T-004 | 4 | In progress |

## 4. Tasks

### M1 — Multi-Typology

| ID | Title | Size | Depends on | Outcome | Decisions | Status |
|---|---|---|---|---|---|---|
| T-001 | Normal feature | S | none | outcome | none | Specifying |
| T-002 | Bugfix leak | S | none | outcome | none | Specifying |
| T-003 | Hotfix crash | S | none | outcome | none | Implementing |
| T-004 | Spike arch | S | none | outcome | none | Specifying |
`
	dir := setupSampleProject(t, roadmapContent)

	// Create task folders across all 4 typologies
	if err := os.MkdirAll(filepath.Join(dir, "features", "T-001-normal-feature"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "bugfixes", "T-002-bugfix-leak"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "hotfixes", "T-003-hotfix-crash"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "spikes", "T-004-spike-arch"), 0o755); err != nil {
		t.Fatal(err)
	}

	ps, err := status.Scan(dir)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	taskMap := make(map[string]status.TaskInfo)
	for _, at := range ps.ActiveTasks {
		taskMap[at.ID] = at
	}

	// 1. T-001 feature
	t1, ok := taskMap["T-001"]
	if !ok {
		t.Fatal("T-001 not found in active tasks")
	}
	if t1.Typology != "feature" {
		t.Errorf("expected T-001 typology 'feature', got %q", t1.Typology)
	}
	if !strings.HasPrefix(filepath.ToSlash(t1.Folder), "features/T-001-normal-feature") {
		t.Errorf("expected T-001 folder in features/, got %q", t1.Folder)
	}
	if !strings.Contains(t1.Gate.Action, "requirements.md") {
		t.Errorf("expected T-001 Gate.Action to mention requirements.md, got %q", t1.Gate.Action)
	}

	// 2. T-002 bugfix
	t2, ok := taskMap["T-002"]
	if !ok {
		t.Fatal("T-002 not found in active tasks")
	}
	if t2.Typology != "bugfix" {
		t.Errorf("expected T-002 typology 'bugfix', got %q", t2.Typology)
	}
	if !strings.HasPrefix(filepath.ToSlash(t2.Folder), "bugfixes/T-002-bugfix-leak") {
		t.Errorf("expected T-002 folder in bugfixes/, got %q", t2.Folder)
	}
	if !strings.Contains(t2.Gate.Action, "reproduction.md") {
		t.Errorf("expected T-002 Gate.Action to mention reproduction.md, got %q", t2.Gate.Action)
	}

	// 3. T-003 hotfix
	t3, ok := taskMap["T-003"]
	if !ok {
		t.Fatal("T-003 not found in active tasks")
	}
	if t3.Typology != "hotfix" {
		t.Errorf("expected T-003 typology 'hotfix', got %q", t3.Typology)
	}
	if !strings.HasPrefix(filepath.ToSlash(t3.Folder), "hotfixes/T-003-hotfix-crash") {
		t.Errorf("expected T-003 folder in hotfixes/, got %q", t3.Folder)
	}
	if !strings.Contains(t3.Gate.Action, "patch.md") {
		t.Errorf("expected T-003 Gate.Action to mention patch.md, got %q", t3.Gate.Action)
	}

	// 4. T-004 spike
	t4, ok := taskMap["T-004"]
	if !ok {
		t.Fatal("T-004 not found in active tasks")
	}
	if t4.Typology != "spike" {
		t.Errorf("expected T-004 typology 'spike', got %q", t4.Typology)
	}
	if !strings.HasPrefix(filepath.ToSlash(t4.Folder), "spikes/T-004-spike-arch") {
		t.Errorf("expected T-004 folder in spikes/, got %q", t4.Folder)
	}
	if !strings.Contains(t4.Gate.Action, "hypotheses.md") {
		t.Errorf("expected T-004 Gate.Action to mention hypotheses.md, got %q", t4.Gate.Action)
	}

	// Verify text render includes typology
	rendered := status.RenderText(ps)
	if !strings.Contains(rendered, "Typology:  bugfix") {
		t.Errorf("expected RenderText to include 'Typology:  bugfix', got:\n%s", rendered)
	}
}
