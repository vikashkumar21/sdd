package upgrade

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/detect"
	"github.com/vikashkumar21/sdd/internal/scaffold"
	"github.com/vikashkumar21/sdd/internal/templates"
)

var fixedNow = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)

func generateKit(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	set, err := templates.New("")
	if err != nil {
		t.Fatal(err)
	}
	det, _ := detect.Detect(root)
	a := scaffold.Answers{
		Project:   "upgradedemo",
		Mode:      scaffold.ModeGreenfield,
		Languages: []string{"go"},
		Agents:    []string{"claude", "gemini", "agents"},
	}
	plan, err := scaffold.BuildPlan(root, a, det, set, fixedNow, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scaffold.Apply(plan, fixedNow, ""); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestUpgradeMissingManifest(t *testing.T) {
	emptyDir := t.TempDir()
	_, err := BuildPlan(emptyDir, Options{})
	if err == nil || !strings.Contains(err.Error(), "project has no manifest") {
		t.Fatalf("expected missing manifest error, got: %v", err)
	}
}

func TestUpgradeUnsupportedSchema(t *testing.T) {
	root := generateKit(t)
	manifestPath := filepath.Join(root, ".sdd", "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	corrupted := strings.Replace(string(data), `"schema": 1`, `"schema": 99`, 1)
	if err := os.WriteFile(manifestPath, []byte(corrupted), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = BuildPlan(root, Options{})
	if err == nil || !strings.Contains(err.Error(), "unsupported manifest schema") {
		t.Fatalf("expected unsupported schema error, got: %v", err)
	}
}

func TestUpgradeAlreadyUpToDate(t *testing.T) {
	root := generateKit(t)
	plan, err := BuildPlan(root, Options{Now: fixedNow})
	if err != nil {
		t.Fatalf("BuildPlan failed: %v", err)
	}
	if !plan.UpToDate {
		t.Errorf("expected clean kit to be UpToDate, but got false")
	}

	rep, err := Apply(plan, fixedNow)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if !rep.UpToDate {
		t.Errorf("expected report to be UpToDate, got false")
	}
	if len(rep.Updated) != 0 || len(rep.Proposals) != 0 {
		t.Errorf("expected 0 updates and 0 proposals, got: %+v", rep)
	}
}

func TestUpgradeUntouchedFiles(t *testing.T) {
	root := generateKit(t)

	// Create an override template with updated mission template
	overrideDir := t.TempDir()
	baseDir := filepath.Join(overrideDir, "base")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	newMissionTmpl := "# MISSION — {{.Project}}\n\nUpdated mission statement in newer template.\n"
	if err := os.WriteFile(filepath.Join(baseDir, "MISSION.md.tmpl"), []byte(newMissionTmpl), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(root, Options{
		TemplatesDir: overrideDir,
		Now:          fixedNow,
	})
	if err != nil {
		t.Fatalf("BuildPlan failed: %v", err)
	}
	if plan.UpToDate {
		t.Fatal("expected plan not to be up-to-date with new template override")
	}

	var missionItem *Item
	for i := range plan.Items {
		if plan.Items[i].Path == "MISSION.md" {
			missionItem = &plan.Items[i]
			break
		}
	}
	if missionItem == nil {
		t.Fatal("expected MISSION.md in plan items")
	}
	if missionItem.Action != ActionUpdate {
		t.Fatalf("expected ActionUpdate for untouched MISSION.md, got %v", missionItem.Action)
	}

	rep, err := Apply(plan, fixedNow.Add(time.Hour))
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if len(rep.Updated) == 0 {
		t.Fatal("expected updated files, got none")
	}

	// Verify MISSION.md was updated on disk
	content, err := os.ReadFile(filepath.Join(root, "MISSION.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "Updated mission statement in newer template.") {
		t.Errorf("expected updated content in MISSION.md, got:\n%s", string(content))
	}

	// Verify backup was created
	if rep.BackupDir == "" {
		t.Fatal("expected BackupDir to be set in report")
	}
	backupContent, err := os.ReadFile(filepath.Join(root, rep.BackupDir, "MISSION.md"))
	if err != nil {
		t.Fatalf("backup file missing: %v", err)
	}
	if strings.Contains(string(backupContent), "Updated mission statement") {
		t.Errorf("backup contains new content instead of original: %s", string(backupContent))
	}

	// Verify manifest was updated
	m, ok, err := scaffold.ReadManifest(root)
	if err != nil || !ok {
		t.Fatalf("ReadManifest failed: %v", err)
	}
	var missionEntry *scaffold.FileEntry
	for i := range m.Files {
		if m.Files[i].Path == "MISSION.md" {
			missionEntry = &m.Files[i]
			break
		}
	}
	if missionEntry == nil || missionEntry.State != "replaced" {
		t.Errorf("expected manifest entry for MISSION.md to be 'replaced', got: %+v", missionEntry)
	}
}

func TestUpgradePreservesEditedFiles(t *testing.T) {
	root := generateKit(t)

	// Simulate human edit in MISSION.md
	missionPath := filepath.Join(root, "MISSION.md")
	humanText := "# MISSION — customized\n\nHuman wrote specific custom goals here.\n"
	if err := os.WriteFile(missionPath, []byte(humanText), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create an override template with updated mission template
	overrideDir := t.TempDir()
	baseDir := filepath.Join(overrideDir, "base")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	newMissionTmpl := "# MISSION — {{.Project}}\n\nUpdated mission statement in newer template.\n"
	if err := os.WriteFile(filepath.Join(baseDir, "MISSION.md.tmpl"), []byte(newMissionTmpl), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(root, Options{
		TemplatesDir: overrideDir,
		Now:          fixedNow,
	})
	if err != nil {
		t.Fatalf("BuildPlan failed: %v", err)
	}

	var missionItem *Item
	for i := range plan.Items {
		if plan.Items[i].Path == "MISSION.md" {
			missionItem = &plan.Items[i]
			break
		}
	}
	if missionItem == nil {
		t.Fatal("expected MISSION.md in plan items")
	}
	if missionItem.Action != ActionProposal {
		t.Fatalf("expected ActionProposal for edited MISSION.md, got %v", missionItem.Action)
	}
	if missionItem.Diff == "" {
		t.Fatal("expected unified diff preview for proposal")
	}

	rep, err := Apply(plan, fixedNow.Add(time.Hour))
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify original MISSION.md was NOT overwritten
	content, err := os.ReadFile(missionPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != humanText {
		t.Errorf("original MISSION.md was overwritten! Got:\n%s", string(content))
	}

	// Verify MISSION.md.sdd-new proposal was written
	proposalPath := filepath.Join(root, "MISSION.md.sdd-new")
	propContent, err := os.ReadFile(proposalPath)
	if err != nil {
		t.Fatalf("proposal file missing: %v", err)
	}
	if !strings.Contains(string(propContent), "Updated mission statement in newer template.") {
		t.Errorf("expected new template content in proposal, got:\n%s", string(propContent))
	}
	if len(rep.Proposals) != 1 || rep.Proposals[0] != "MISSION.md.sdd-new" {
		t.Errorf("expected rep.Proposals to list MISSION.md.sdd-new, got: %+v", rep.Proposals)
	}
}

func TestUpgradeManagedBlocks(t *testing.T) {
	root := generateKit(t)

	// Append human custom rules outside the managed block in CLAUDE.md
	claudePath := filepath.Join(root, "CLAUDE.md")
	existing, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}
	humanAppendix := "\n## My Custom Project Rules\n\n1. Always write unit tests.\n2. Never push on Friday.\n"
	if err := os.WriteFile(claudePath, append(existing, []byte(humanAppendix)...), 0o644); err != nil {
		t.Fatal(err)
	}

	// Override base/contract.md.tmpl
	overrideDir := t.TempDir()
	baseDir := filepath.Join(overrideDir, "base")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	newContractTmpl := "> ANTHEM: UPDATED ANTHEM FOR V2.\n\n## 1. Updated workflow\n..."
	if err := os.WriteFile(filepath.Join(baseDir, "contract.md.tmpl"), []byte(newContractTmpl), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(root, Options{
		TemplatesDir: overrideDir,
		Now:          fixedNow,
	})
	if err != nil {
		t.Fatalf("BuildPlan failed: %v", err)
	}

	var claudeItem *Item
	for i := range plan.Items {
		if plan.Items[i].Path == "CLAUDE.md" {
			claudeItem = &plan.Items[i]
			break
		}
	}
	if claudeItem == nil {
		t.Fatal("expected CLAUDE.md in plan items")
	}
	if claudeItem.Action != ActionUpdate {
		t.Fatalf("expected ActionUpdate for untouched managed block with outside text, got %v", claudeItem.Action)
	}

	rep, err := Apply(plan, fixedNow.Add(time.Hour))
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if len(rep.Updated) == 0 {
		t.Fatal("expected CLAUDE.md to be updated")
	}

	// Verify updated CLAUDE.md contains both the new contract AND human custom rules
	updatedData, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}
	updatedStr := string(updatedData)
	if !strings.Contains(updatedStr, "UPDATED ANTHEM FOR V2.") {
		t.Error("managed block was not updated with new template")
	}
	if !strings.Contains(updatedStr, "## My Custom Project Rules") || !strings.Contains(updatedStr, "Never push on Friday.") {
		t.Error("human custom rules outside managed block were deleted or lost!")
	}
}

func TestUpgradeForce(t *testing.T) {
	root := generateKit(t)

	// Human edits MISSION.md
	missionPath := filepath.Join(root, "MISSION.md")
	humanText := "# Human mission\n"
	if err := os.WriteFile(missionPath, []byte(humanText), 0o644); err != nil {
		t.Fatal(err)
	}

	overrideDir := t.TempDir()
	baseDir := filepath.Join(overrideDir, "base")
	_ = os.MkdirAll(baseDir, 0o755)
	_ = os.WriteFile(filepath.Join(baseDir, "MISSION.md.tmpl"), []byte("# Force updated mission\n"), 0o644)

	plan, err := BuildPlan(root, Options{
		TemplatesDir: overrideDir,
		Force:        true,
		Now:          fixedNow,
	})
	if err != nil {
		t.Fatal(err)
	}

	var missionItem *Item
	for i := range plan.Items {
		if plan.Items[i].Path == "MISSION.md" {
			missionItem = &plan.Items[i]
			break
		}
	}
	if missionItem == nil || missionItem.Action != ActionForce {
		t.Fatalf("expected ActionForce under --force, got: %+v", missionItem)
	}

	rep, err := Apply(plan, fixedNow.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(missionPath)
	if !strings.Contains(string(content), "Force updated mission") {
		t.Errorf("expected force updated content, got: %s", string(content))
	}
	if rep.BackupDir == "" {
		t.Error("expected backup under --force")
	}
}

func TestUpgradeCreatesMissingReviewTemplate(t *testing.T) {
	root := generateKit(t)
	reviewPath := filepath.Join(root, "features", "_templates", "review.md")
	_ = os.Remove(reviewPath)

	plan, err := BuildPlan(root, Options{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}

	var reviewItem *Item
	for i := range plan.Items {
		if plan.Items[i].Path == "features/_templates/review.md" {
			reviewItem = &plan.Items[i]
			break
		}
	}
	if reviewItem == nil || reviewItem.Action != ActionCreate {
		t.Fatalf("expected ActionCreate for missing review.md template, got: %+v", reviewItem)
	}

	rep, err := Apply(plan, fixedNow.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Created) == 0 {
		t.Errorf("expected review.md to be created in report")
	}

	content, err := os.ReadFile(reviewPath)
	if err != nil {
		t.Fatalf("failed to read created review.md: %v", err)
	}
	if !strings.Contains(string(content), "# Code Review — T-xxx <title>") {
		t.Errorf("unexpected content in created review.md: %s", string(content))
	}
}
