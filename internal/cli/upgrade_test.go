package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIUpgradeMissingManifest(t *testing.T) {
	emptyDir := t.TempDir()
	r := exec(t, "", false, "", "upgrade", "--dir", emptyDir)
	if r.code != 1 || !strings.Contains(r.err, "has no manifest") {
		t.Fatalf("expected missing manifest error, got: %+v", r)
	}
}

func TestCLIUpgradeAlreadyUpToDate(t *testing.T) {
	root := t.TempDir()
	rInit := exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "demo")
	if rInit.code != 0 {
		t.Fatalf("init failed: %+v", rInit)
	}

	rUp := exec(t, "", false, "", "upgrade", "--dir", root)
	if rUp.code != 0 {
		t.Fatalf("upgrade failed: %+v", rUp)
	}
	if !strings.Contains(rUp.out, "already up-to-date") {
		t.Errorf("expected 'already up-to-date' in output, got:\n%s", rUp.out)
	}
}

func TestCLIUpgradeDryRun(t *testing.T) {
	root := t.TempDir()
	rInit := exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "demo")
	if rInit.code != 0 {
		t.Fatalf("init failed: %+v", rInit)
	}

	// Create template override
	tplDir := t.TempDir()
	baseDir := filepath.Join(tplDir, "base")
	_ = os.MkdirAll(baseDir, 0o755)
	_ = os.WriteFile(filepath.Join(baseDir, "MISSION.md.tmpl"), []byte("# Newer Mission\n"), 0o644)

	rDry := exec(t, "", false, "", "upgrade", "--dir", root, "--templates", tplDir, "--dry-run")
	if rDry.code != 0 {
		t.Fatalf("dry-run failed: %+v", rDry)
	}
	if !strings.Contains(rDry.out, "Dry run: nothing was written") {
		t.Errorf("expected dry-run note in output, got:\n%s", rDry.out)
	}
	// Verify MISSION.md was NOT updated
	curr := read(t, root, "MISSION.md")
	if strings.Contains(curr, "Newer Mission") {
		t.Errorf("file was updated during dry-run:\n%s", curr)
	}
}

func TestCLIUpgradeUntouchedAndEdited(t *testing.T) {
	root := t.TempDir()
	rInit := exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "demo")
	if rInit.code != 0 {
		t.Fatalf("init failed: %+v", rInit)
	}

	// Simulate human edit in MISSION.md
	put(t, root, "MISSION.md", "# Human Mission\nCustom content by user.\n")

	// Create template override modifying both MISSION.md and ROADMAP.md
	tplDir := t.TempDir()
	baseDir := filepath.Join(tplDir, "base")
	_ = os.MkdirAll(baseDir, 0o755)
	_ = os.WriteFile(filepath.Join(baseDir, "MISSION.md.tmpl"), []byte("# Upgraded Mission\n"), 0o644)
	_ = os.WriteFile(filepath.Join(baseDir, "ROADMAP.md.tmpl"), []byte("# Upgraded Roadmap\n"), 0o644)

	rUp := exec(t, "", false, "", "upgrade", "--yes", "--dir", root, "--templates", tplDir)
	if rUp.code != 0 {
		t.Fatalf("upgrade failed: %+v", rUp)
	}

	// 1. Untouched ROADMAP.md should be updated
	roadmapContent := read(t, root, "ROADMAP.md")
	if !strings.Contains(roadmapContent, "Upgraded Roadmap") {
		t.Errorf("ROADMAP.md was not updated:\n%s", roadmapContent)
	}

	// 2. Edited MISSION.md should NOT be overwritten; proposal should exist
	missionContent := read(t, root, "MISSION.md")
	if !strings.Contains(missionContent, "Custom content by user") {
		t.Errorf("human edits in MISSION.md were overwritten! Got:\n%s", missionContent)
	}
	proposalContent := read(t, root, "MISSION.md.sdd-new")
	if !strings.Contains(proposalContent, "Upgraded Mission") {
		t.Errorf("proposal content missing in MISSION.md.sdd-new:\n%s", proposalContent)
	}

	// 3. Backup directory should be reported and contain original ROADMAP.md
	if !strings.Contains(rUp.out, "Backup:") {
		t.Errorf("expected Backup line in output, got:\n%s", rUp.out)
	}
}

func TestCLIUpgradeForce(t *testing.T) {
	root := t.TempDir()
	rInit := exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "demo")
	if rInit.code != 0 {
		t.Fatalf("init failed: %+v", rInit)
	}

	// Simulate human edit in MISSION.md
	put(t, root, "MISSION.md", "# Human Mission\nCustom content.\n")

	tplDir := t.TempDir()
	baseDir := filepath.Join(tplDir, "base")
	_ = os.MkdirAll(baseDir, 0o755)
	_ = os.WriteFile(filepath.Join(baseDir, "MISSION.md.tmpl"), []byte("# Force Upgraded Mission\n"), 0o644)

	rForce := exec(t, "", false, "", "upgrade", "--yes", "--dir", root, "--templates", tplDir, "--force")
	if rForce.code != 0 {
		t.Fatalf("upgrade --force failed: %+v", rForce)
	}

	missionContent := read(t, root, "MISSION.md")
	if !strings.Contains(missionContent, "Force Upgraded Mission") {
		t.Errorf("expected forced replacement in MISSION.md, got:\n%s", missionContent)
	}
	if exists(root, "MISSION.md.sdd-new") {
		t.Error("proposal file should not be created under --force")
	}
}

func TestCLIUpgradeInteractiveAbort(t *testing.T) {
	root := t.TempDir()
	rInit := exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "demo")
	if rInit.code != 0 {
		t.Fatalf("init failed: %+v", rInit)
	}

	tplDir := t.TempDir()
	baseDir := filepath.Join(tplDir, "base")
	_ = os.MkdirAll(baseDir, 0o755)
	_ = os.WriteFile(filepath.Join(baseDir, "ROADMAP.md.tmpl"), []byte("# Upgraded Roadmap\n"), 0o644)

	var out, errb bytes.Buffer
	env := IO{
		In:          strings.NewReader("n\n"),
		Out:         &out,
		Err:         &errb,
		Interactive: true,
		Now:         fixedNow,
	}
	code := Run([]string{"upgrade", "--dir", root, "--templates", tplDir}, env)
	if code != 1 {
		t.Fatalf("expected exit 1 on abort, got %d", code)
	}
	if !strings.Contains(out.String(), "Aborted") {
		t.Errorf("expected 'Aborted' in output, got:\n%s", out.String())
	}
	// Verify ROADMAP.md was NOT modified
	curr := read(t, root, "ROADMAP.md")
	if strings.Contains(curr, "Upgraded Roadmap") {
		t.Errorf("file modified despite abort:\n%s", curr)
	}
}
