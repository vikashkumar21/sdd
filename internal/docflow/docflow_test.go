package docflow_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/docflow"
	"github.com/vikashkumar21/sdd/internal/markdown"
	"github.com/vikashkumar21/sdd/internal/scaffold"
)

func TestBuildInitPlan_Fresh(t *testing.T) {
	tmp := t.TempDir()

	opts := docflow.InitOptions{
		Dir:         tmp,
		ProjectName: "TestApp",
		Now:         time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC),
	}

	plan, err := docflow.BuildInitPlan(opts)
	if err != nil {
		t.Fatalf("BuildInitPlan failed: %v", err)
	}

	if len(plan) != 4 {
		t.Fatalf("expected 4 planned files, got %d", len(plan))
	}

	expectedPaths := map[string]bool{
		"docs/architecture/overview.md":      false,
		"docs/architecture/data_model.md":    false,
		"docs/business_flows/overview.md":    false,
		"docs/business_flows/sample_flow.md": false,
	}

	for _, p := range plan {
		if _, ok := expectedPaths[p.RelPath]; !ok {
			t.Errorf("unexpected planned path %s", p.RelPath)
		}
		expectedPaths[p.RelPath] = true
		if p.Action != "create" {
			t.Errorf("expected action 'create' for %s, got %s", p.RelPath, p.Action)
		}
		// Verify Mermaid syntax in rendered content
		findings := markdown.CheckMermaid(string(p.Content))
		if len(findings) > 0 {
			t.Errorf("Mermaid syntax error in %s: %+v", p.RelPath, findings)
		}
	}

	for p, seen := range expectedPaths {
		if !seen {
			t.Errorf("missing planned path %s", p)
		}
	}
}

func TestApply_CreateAndSkip(t *testing.T) {
	tmp := t.TempDir()

	opts := docflow.InitOptions{
		Dir:         tmp,
		ProjectName: "TestApp",
		Now:         time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC),
	}

	plan, err := docflow.BuildInitPlan(opts)
	if err != nil {
		t.Fatalf("BuildInitPlan failed: %v", err)
	}

	res, err := docflow.Apply(tmp, plan, opts.Now)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if len(res.Created) != 4 {
		t.Fatalf("expected 4 created files, got %d", len(res.Created))
	}

	// Verify files exist on disk
	for _, p := range res.Created {
		full := filepath.Join(tmp, filepath.FromSlash(p))
		if _, err := os.Stat(full); err != nil {
			t.Errorf("file %s not created: %v", full, err)
		}
	}

	// Running build plan again without force should produce skip actions
	plan2, err := docflow.BuildInitPlan(opts)
	if err != nil {
		t.Fatalf("BuildInitPlan second run failed: %v", err)
	}

	for _, p := range plan2 {
		if p.Action != "skip" {
			t.Errorf("expected action 'skip' for %s without force, got %s", p.RelPath, p.Action)
		}
	}

	res2, err := docflow.Apply(tmp, plan2, opts.Now)
	if err != nil {
		t.Fatalf("Apply second run failed: %v", err)
	}
	if len(res2.Skipped) != 4 {
		t.Fatalf("expected 4 skipped files, got %d", len(res2.Skipped))
	}
}

func TestApply_ForceAndBackup(t *testing.T) {
	tmp := t.TempDir()

	opts := docflow.InitOptions{
		Dir:         tmp,
		ProjectName: "TestApp",
		Now:         time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC),
	}

	// 1. Initial write
	plan, _ := docflow.BuildInitPlan(opts)
	_, _ = docflow.Apply(tmp, plan, opts.Now)

	// 2. Modify one file manually
	targetFile := filepath.Join(tmp, "docs", "architecture", "overview.md")
	_ = os.WriteFile(targetFile, []byte("# Custom human content\n"), 0644)

	// 3. Re-run with Force
	optsForce := docflow.InitOptions{
		Dir:         tmp,
		Force:       true,
		ProjectName: "TestApp",
		Now:         time.Date(2026, 11, 22, 11, 0, 0, 0, time.UTC),
	}

	planForce, err := docflow.BuildInitPlan(optsForce)
	if err != nil {
		t.Fatalf("BuildInitPlan with force failed: %v", err)
	}

	for _, p := range planForce {
		if p.Action != "replace" {
			t.Errorf("expected action 'replace' for %s with force, got %s", p.RelPath, p.Action)
		}
	}

	resForce, err := docflow.Apply(tmp, planForce, optsForce.Now)
	if err != nil {
		t.Fatalf("Apply with force failed: %v", err)
	}

	if len(resForce.Backups) != 4 {
		t.Fatalf("expected 4 backup files, got %d", len(resForce.Backups))
	}

	// Verify backup file exists and contains the custom human content
	var customBackupFound bool
	for _, bRel := range resForce.Backups {
		if strings.Contains(bRel, "overview.md") {
			bPath := filepath.Join(tmp, filepath.FromSlash(bRel))
			content, err := os.ReadFile(bPath)
			if err == nil && string(content) == "# Custom human content\n" {
				customBackupFound = true
			}
		}
	}
	if !customBackupFound {
		t.Errorf("custom content backup not found in backups: %+v", resForce.Backups)
	}
}

func TestBuildFlowPlan_ValidAndInvalid(t *testing.T) {
	tmp := t.TempDir()

	// Valid flow
	flowOpts := docflow.FlowOptions{
		Dir:         tmp,
		Name:        "user-login",
		Title:       "User Authentication Flow",
		Now:         time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC),
		ProjectName: "AuthApp",
	}

	plan, err := docflow.BuildFlowPlan(flowOpts)
	if err != nil {
		t.Fatalf("BuildFlowPlan failed: %v", err)
	}

	if len(plan) != 1 {
		t.Fatalf("expected 1 planned entry, got %d", len(plan))
	}
	if plan[0].RelPath != "docs/business_flows/user-login.md" {
		t.Errorf("expected path docs/business_flows/user-login.md, got %s", plan[0].RelPath)
	}

	// Verify Mermaid syntax in flow document
	findings := markdown.CheckMermaid(string(plan[0].Content))
	if len(findings) > 0 {
		t.Errorf("Mermaid syntax error in flow doc: %+v", findings)
	}

	// Apply flow doc
	res, err := docflow.Apply(tmp, plan, flowOpts.Now)
	if err != nil {
		t.Fatalf("Apply flow doc failed: %v", err)
	}
	if len(res.Created) != 1 {
		t.Errorf("expected 1 created file, got %d", len(res.Created))
	}

	// Invalid flow names
	invalidNames := []string{
		"",
		"user/login",
		"../escape",
		"user login",
		"user$name",
		strings.Repeat("a", 65), // > 64 chars
	}

	for _, inv := range invalidNames {
		badOpts := docflow.FlowOptions{
			Dir:  tmp,
			Name: inv,
		}
		if _, err := docflow.BuildFlowPlan(badOpts); err == nil {
			t.Errorf("expected error for invalid flow name %q, got nil", inv)
		}
	}

	// Invalid flow titles
	invalidTitles := []string{
		strings.Repeat("T", 121), // > 120 chars
		"Title with\nnewline",
		"Title with\rcarriage return",
		"Title with `backticks`",
	}

	for _, invTitle := range invalidTitles {
		badOpts := docflow.FlowOptions{
			Dir:   tmp,
			Name:  "valid-name",
			Title: invTitle,
		}
		if _, err := docflow.BuildFlowPlan(badOpts); err == nil {
			t.Errorf("expected error for invalid flow title %q, got nil", invTitle)
		}
	}
}

func TestApply_ManifestUpdate(t *testing.T) {
	tmp := t.TempDir()

	// Create fake manifest
	manifestDir := filepath.Join(tmp, ".sdd")
	_ = os.MkdirAll(manifestDir, 0755)
	initialManifest := scaffold.Manifest{
		Schema: 1,
		Files: []scaffold.FileEntry{
			{Path: "README.md", Kind: "file", State: "existing"},
		},
	}
	mData, _ := json.Marshal(initialManifest)
	_ = os.WriteFile(filepath.Join(manifestDir, "manifest.json"), mData, 0644)

	opts := docflow.InitOptions{
		Dir:         tmp,
		ProjectName: "ManifestApp",
		Now:         time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC),
	}

	plan, err := docflow.BuildInitPlan(opts)
	if err != nil {
		t.Fatalf("BuildInitPlan failed: %v", err)
	}

	_, err = docflow.Apply(tmp, plan, opts.Now)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Read manifest and verify new files were recorded
	updated, ok, err := scaffold.ReadManifest(tmp)
	if err != nil || !ok {
		t.Fatalf("ReadManifest failed: err=%v, ok=%v", err, ok)
	}

	if len(updated.Files) != 5 {
		t.Fatalf("expected 5 files in manifest (1 initial + 4 docs), got %d", len(updated.Files))
	}

	foundDoc := false
	for _, f := range updated.Files {
		if f.Path == "docs/architecture/overview.md" {
			foundDoc = true
			if f.SHA256 == "" {
				t.Errorf("expected non-empty SHA256 for %s", f.Path)
			}
			if f.State != "created" {
				t.Errorf("expected state 'created', got %s", f.State)
			}
		}
	}
	if !foundDoc {
		t.Errorf("docs/architecture/overview.md not recorded in manifest: %+v", updated.Files)
	}
}
