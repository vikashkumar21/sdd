package repair

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/block"
)

func writeTestFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, root, rel string) string {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestDiff(t *testing.T) {
	orig := "line 1\nline 2\nline 3\n"
	mod := "line 1\nline 2 modified\nline 3\n"
	diff := Diff(orig, mod, "test.md")
	if !strings.Contains(diff, "--- a/test.md") || !strings.Contains(diff, "+++ b/test.md") {
		t.Fatalf("diff missing header:\n%s", diff)
	}
	if !strings.Contains(diff, "-line 2") || !strings.Contains(diff, "+line 2 modified") {
		t.Fatalf("diff missing changes:\n%s", diff)
	}

	sameDiff := Diff(orig, orig, "test.md")
	if sameDiff != "" {
		t.Fatalf("expected empty diff for identical inputs, got %q", sameDiff)
	}
}

func TestRepairContractParity(t *testing.T) {
	root := t.TempDir()

	canonicalBlock := "CONTRACT RULES v2\n- rule 1\n- rule 2"
	agentsContent := "# AGENTS\n\n" + block.Begin("1.0.0") + "\n" + canonicalBlock + "\n<!-- sdd:end contract -->\n\nHuman footer"
	writeTestFile(t, root, "AGENTS.md", agentsContent)

	geminiContent := "# GEMINI\n\n" + block.Begin("1.0.0") + "\nOLD RULES\n<!-- sdd:end contract -->\n\nHuman notes"
	writeTestFile(t, root, "GEMINI.md", geminiContent)

	claudeContent := "# CLAUDE\n\nSome custom human instructions."
	writeTestFile(t, root, "CLAUDE.md", claudeContent)

	plan, err := PlanRepairs(root, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(plan.Patches) != 2 {
		t.Fatalf("expected 2 patches, got %d", len(plan.Patches))
	}

	// Verify patches
	var geminiPatch, claudePatch *Patch
	for i := range plan.Patches {
		p := &plan.Patches[i]
		switch p.Path {
		case "GEMINI.md":
			geminiPatch = p
		case "CLAUDE.md":
			claudePatch = p
		}
	}
	if geminiPatch == nil || claudePatch == nil {
		t.Fatalf("expected patches for both GEMINI.md and CLAUDE.md, got %+v", plan.Patches)
	}

	// Apply repairs
	now := time.Date(2026, 11, 2, 16, 0, 0, 0, time.UTC)
	res, err := Apply(root, plan, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Repaired) != 2 {
		t.Errorf("repaired count = %d, want 2", len(res.Repaired))
	}
	if res.BackupDir != ".sdd/backup/20261102-160000" {
		t.Errorf("backupDir = %s, want .sdd/backup/20261102-160000", res.BackupDir)
	}

	// Verify GEMINI.md has canonical block and preserved footer
	geminiAfter := readTestFile(t, root, "GEMINI.md")
	if !strings.Contains(geminiAfter, canonicalBlock) {
		t.Errorf("GEMINI.md missing canonical block:\n%s", geminiAfter)
	}
	if !strings.Contains(geminiAfter, "Human notes") {
		t.Errorf("GEMINI.md lost human notes outside block:\n%s", geminiAfter)
	}

	// Verify backup was created
	geminiBackup := readTestFile(t, root, ".sdd/backup/20261102-160000/GEMINI.md")
	if geminiBackup != geminiContent {
		t.Errorf("backup content mismatch:\ngot: %s\nwant: %s", geminiBackup, geminiContent)
	}
}

func TestRepairTableDelimiters(t *testing.T) {
	root := t.TempDir()

	brokenTable := `# Document

| Col 1 | Col 2 | Col 3 |
|---|---|
| A | B | C |
`
	writeTestFile(t, root, "doc.md", brokenTable)

	plan, err := PlanRepairs(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Patches) != 1 {
		t.Fatalf("expected 1 table patch, got %d", len(plan.Patches))
	}
	p := plan.Patches[0]
	if p.Path != "doc.md" {
		t.Errorf("patch path = %s, want doc.md", p.Path)
	}
	if !strings.Contains(p.Modified, "|---|---|---|") {
		t.Errorf("expected 3-column delimiter row, got:\n%s", p.Modified)
	}

	now := time.Now()
	res, err := Apply(root, plan, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Repaired) != 1 {
		t.Errorf("repaired = %v", res.Repaired)
	}
	after := readTestFile(t, root, "doc.md")
	if !strings.Contains(after, "|---|---|---|") {
		t.Errorf("file after repair wrong:\n%s", after)
	}
}

func TestRepairFilterFiles(t *testing.T) {
	root := t.TempDir()
	canonicalBlock := "CONTRACT RULES"
	writeTestFile(t, root, "AGENTS.md", block.Begin("1.0.0")+"\n"+canonicalBlock+"\n<!-- sdd:end contract -->")
	writeTestFile(t, root, "GEMINI.md", block.Begin("1.0.0")+"\nOLD\n<!-- sdd:end contract -->")
	writeTestFile(t, root, "CLAUDE.md", block.Begin("1.0.0")+"\nOLD\n<!-- sdd:end contract -->")

	// Filter to only GEMINI.md
	plan, err := PlanRepairs(root, []string{"GEMINI.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Patches) != 1 || plan.Patches[0].Path != "GEMINI.md" {
		t.Fatalf("expected only GEMINI.md patch, got %+v", plan.Patches)
	}
}
