package upgrade

import (
	"encoding/json"
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
	reviewPath := filepath.Join(root, "tasks", "features", "_templates", "review.md")
	_ = os.Remove(reviewPath)

	plan, err := BuildPlan(root, Options{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}

	var reviewItem *Item
	for i := range plan.Items {
		if plan.Items[i].Path == "tasks/features/_templates/review.md" {
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

func TestUpgradeTypologies(t *testing.T) {
	root := generateKit(t)

	// Remove bugfixes, hotfixes, spikes templates to simulate an older kit
	_ = os.RemoveAll(filepath.Join(root, "tasks", "bugfixes"))
	_ = os.RemoveAll(filepath.Join(root, "tasks", "hotfixes"))
	_ = os.RemoveAll(filepath.Join(root, "tasks", "spikes"))

	plan, err := BuildPlan(root, Options{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}

	createdMap := map[string]bool{}
	for _, it := range plan.Items {
		if it.Action == ActionCreate {
			createdMap[it.Path] = true
		}
	}

	expectedTemplates := []string{
		"tasks/bugfixes/_templates/reproduction.md",
		"tasks/bugfixes/_templates/rca.md",
		"tasks/bugfixes/_templates/plan.md",
		"tasks/bugfixes/_templates/validation.md",
		"tasks/bugfixes/_templates/review.md",
		"tasks/hotfixes/_templates/incident.md",
		"tasks/hotfixes/_templates/patch.md",
		"tasks/hotfixes/_templates/validation.md",
		"tasks/hotfixes/_templates/post_mortem.md",
		"tasks/spikes/_templates/hypotheses.md",
		"tasks/spikes/_templates/findings.md",
		"tasks/spikes/_templates/plan.md",
	}

	for _, tmpl := range expectedTemplates {
		if !createdMap[tmpl] {
			t.Errorf("expected plan to create %s, but it was not in create items", tmpl)
		}
	}

	rep, err := Apply(plan, fixedNow.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	for _, tmpl := range expectedTemplates {
		abs := filepath.Join(root, filepath.FromSlash(tmpl))
		if _, err := os.Stat(abs); err != nil {
			t.Errorf("expected %s to be created on disk, but stat failed: %v", tmpl, err)
		}
	}

	if len(rep.Created) < len(expectedTemplates) {
		t.Errorf("expected at least %d created templates, got %d", len(expectedTemplates), len(rep.Created))
	}
}

func TestUpgradeLegacyFolderMigration(t *testing.T) {
	root := generateKit(t)

	// Transform kit to legacy layout: move tasks/features to features, etc.
	for _, topo := range []string{"features", "bugfixes", "hotfixes", "spikes"} {
		src := filepath.Join(root, "tasks", topo)
		dst := filepath.Join(root, topo)
		if err := os.Rename(src, dst); err != nil {
			t.Fatal(err)
		}
	}
	_ = os.Remove(filepath.Join(root, "tasks"))

	// Create a dummy task in legacy features/
	taskDir := filepath.Join(root, "features", "T-001-legacy-task")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	taskReq := filepath.Join(taskDir, "requirements.md")
	if err := os.WriteFile(taskReq, []byte("# Requirements"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Update manifest to match legacy paths
	manifestPath := filepath.Join(root, ".sdd", "manifest.json")
	mBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	updatedManifest := strings.ReplaceAll(string(mBytes), "tasks/features/", "features/")
	updatedManifest = strings.ReplaceAll(updatedManifest, "tasks/bugfixes/", "bugfixes/")
	updatedManifest = strings.ReplaceAll(updatedManifest, "tasks/hotfixes/", "hotfixes/")
	updatedManifest = strings.ReplaceAll(updatedManifest, "tasks/spikes/", "spikes/")
	if err := os.WriteFile(manifestPath, []byte(updatedManifest), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(root, Options{Now: fixedNow})
	if err != nil {
		t.Fatalf("BuildPlan failed: %v", err)
	}

	if plan.UpToDate {
		t.Errorf("expected plan not to be up-to-date due to legacy folders")
	}

	if len(plan.Migrations) != 4 {
		t.Fatalf("expected 4 folder migrations, got %d: %+v", len(plan.Migrations), plan.Migrations)
	}

	rep, err := Apply(plan, fixedNow.Add(time.Hour))
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if len(rep.Migrated) != 4 {
		t.Errorf("expected 4 migrated folders in report, got %d: %+v", len(rep.Migrated), rep.Migrated)
	}

	// Verify legacy folders no longer exist at root
	for _, topo := range []string{"features", "bugfixes", "hotfixes", "spikes"} {
		if _, err := os.Stat(filepath.Join(root, topo)); !os.IsNotExist(err) {
			t.Errorf("expected legacy folder %s to be moved, but it still exists", topo)
		}
	}

	// Verify canonical folders exist under tasks/
	for _, topo := range []string{"features", "bugfixes", "hotfixes", "spikes"} {
		if info, err := os.Stat(filepath.Join(root, "tasks", topo)); err != nil || !info.IsDir() {
			t.Errorf("expected tasks/%s to exist", topo)
		}
	}

	// Verify task folder moved
	migratedTaskReq := filepath.Join(root, "tasks", "features", "T-001-legacy-task", "requirements.md")
	if _, err := os.Stat(migratedTaskReq); err != nil {
		t.Errorf("expected migrated task requirements.md to exist: %v", err)
	}

	// Verify manifest paths updated
	man, ok, err := scaffold.ReadManifest(root)
	if err != nil || !ok {
		t.Fatalf("failed to read updated manifest: %v", err)
	}
	for _, f := range man.Files {
		if strings.HasPrefix(f.Path, "features/") || strings.HasPrefix(f.Path, "bugfixes/") {
			t.Errorf("found unmigrated manifest path: %s", f.Path)
		}
	}
}

func TestUpgradeDryRunMigration(t *testing.T) {
	root := generateKit(t)

	// Move tasks/features to root features
	src := filepath.Join(root, "tasks", "features")
	dst := filepath.Join(root, "features")
	if err := os.Rename(src, dst); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(filepath.Join(root, "tasks"))

	plan, err := BuildPlan(root, Options{Now: fixedNow})
	if err != nil {
		t.Fatalf("BuildPlan failed: %v", err)
	}

	hasFeatureMigration := false
	for _, mig := range plan.Migrations {
		if mig.From == "features" && mig.To == "tasks/features" {
			hasFeatureMigration = true
			break
		}
	}
	if !hasFeatureMigration {
		t.Errorf("expected migration from features to tasks/features in plan")
	}

	// In a dry run, Apply is never called. Verify root features still exists.
	if _, err := os.Stat(filepath.Join(root, "features")); err != nil {
		t.Errorf("features directory should still exist in dry-run mode: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "tasks", "features")); !os.IsNotExist(err) {
		t.Errorf("tasks/features directory should not exist in dry-run mode")
	}
}

func TestUpgradeMigrationIsDeterministic(t *testing.T) {
	createLegacyKit := func() string {
		root := generateKit(t)
		for _, topo := range []string{"features", "bugfixes", "hotfixes", "spikes"} {
			src := filepath.Join(root, "tasks", topo)
			dst := filepath.Join(root, topo)
			if err := os.Rename(src, dst); err != nil {
				t.Fatal(err)
			}
		}
		_ = os.Remove(filepath.Join(root, "tasks"))

		manifestPath := filepath.Join(root, ".sdd", "manifest.json")
		mBytes, err := os.ReadFile(manifestPath)
		if err != nil {
			t.Fatal(err)
		}
		updatedManifest := strings.ReplaceAll(string(mBytes), "tasks/features/", "features/")
		updatedManifest = strings.ReplaceAll(updatedManifest, "tasks/bugfixes/", "bugfixes/")
		updatedManifest = strings.ReplaceAll(updatedManifest, "tasks/hotfixes/", "hotfixes/")
		updatedManifest = strings.ReplaceAll(updatedManifest, "tasks/spikes/", "spikes/")
		if err := os.WriteFile(manifestPath, []byte(updatedManifest), 0o644); err != nil {
			t.Fatal(err)
		}
		return root
	}

	root1 := createLegacyKit()
	root2 := createLegacyKit()

	plan1, err := BuildPlan(root1, Options{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Apply(plan1, fixedNow.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	plan2, err := BuildPlan(root2, Options{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Apply(plan2, fixedNow.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	m1, err := os.ReadFile(filepath.Join(root1, ".sdd", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	m2, err := os.ReadFile(filepath.Join(root2, ".sdd", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}

	if string(m1) != string(m2) {
		t.Errorf("manifests after upgrade are not identical across runs:\n--- run 1 ---\n%s\n--- run 2 ---\n%s", string(m1), string(m2))
	}
}

func TestUpgradeInstallsMissingRAGIgnore(t *testing.T) {
	root := generateKit(t)

	// Remove .ragignore if present
	ragignorePath := filepath.Join(root, ".ragignore")
	_ = os.Remove(ragignorePath)

	// Also remove .ragignore entry from manifest to simulate kit created before .ragignore
	manifestPath := filepath.Join(root, ".sdd", "manifest.json")
	m, ok, err := scaffold.ReadManifest(root)
	if err != nil || !ok {
		t.Fatal(err)
	}
	var filtered []scaffold.FileEntry
	for _, f := range m.Files {
		if f.Path != ".ragignore" {
			filtered = append(filtered, f)
		}
	}
	m.Files = filtered
	mBytes, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, append(mBytes, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(root, Options{Now: fixedNow})
	if err != nil {
		t.Fatalf("BuildPlan failed: %v", err)
	}

	var ragItem *Item
	for i := range plan.Items {
		if plan.Items[i].Path == ".ragignore" {
			ragItem = &plan.Items[i]
			break
		}
	}
	if ragItem == nil {
		t.Fatal("expected .ragignore in plan items")
	}
	if ragItem.Action != ActionCreate {
		t.Fatalf("expected ActionCreate for missing .ragignore, got %v", ragItem.Action)
	}

	rep, err := Apply(plan, fixedNow.Add(time.Hour))
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	found := false
	for _, c := range rep.Created {
		if c == ".ragignore" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected .ragignore in rep.Created, got: %+v", rep.Created)
	}

	content, err := os.ReadFile(ragignorePath)
	if err != nil {
		t.Fatalf("missing .ragignore file on disk: %v", err)
	}
	if !strings.Contains(string(content), ".sdd/") {
		t.Errorf("expected .sdd/ in .ragignore content, got:\n%s", string(content))
	}
}
