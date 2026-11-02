package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vikashkumar21/sdd/internal/block"
)

func writeCheckFixFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readCheckFixFile(t *testing.T, root, rel string) string {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func makeKitWithDrift(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	canonicalBlock := "CONTRACT RULES v2\n- rule A\n- rule B"
	agentsContent := "# AGENTS\n\n" + block.Begin("1.0.0") + "\n" + canonicalBlock + "\n<!-- sdd:end contract -->\n"
	writeCheckFixFile(t, root, "AGENTS.md", agentsContent)

	geminiContent := "# GEMINI\n\n" + block.Begin("1.0.0") + "\nOLD RULES\n<!-- sdd:end contract -->\n\nNotes"
	writeCheckFixFile(t, root, "GEMINI.md", geminiContent)

	claudeContent := "# CLAUDE\n\n" + block.Begin("1.0.0") + "\nOLD RULES\n<!-- sdd:end contract -->\n"
	writeCheckFixFile(t, root, "CLAUDE.md", claudeContent)

	return root
}

func TestCheckFixNoRepairsNeeded(t *testing.T) {
	root := t.TempDir()
	canonicalBlock := "RULES"
	contract := block.Begin("1.0.0") + "\n" + canonicalBlock + "\n<!-- sdd:end contract -->"
	writeCheckFixFile(t, root, "AGENTS.md", "# AGENTS\n\n"+contract)
	writeCheckFixFile(t, root, "GEMINI.md", "# GEMINI\n\n"+contract)
	writeCheckFixFile(t, root, "CLAUDE.md", "# CLAUDE\n\n"+contract)

	var stdout, stderr bytes.Buffer
	env := IO{
		In:          bytes.NewReader(nil),
		Out:         &stdout,
		Err:         &stderr,
		Now:         time.Now,
		Interactive: true,
	}

	code := Run([]string{"check", "--dir", root, "--fix"}, env)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "No mechanical repairs needed.") {
		t.Fatalf("stdout missing expected message:\n%s", stdout.String())
	}
}

func TestCheckFixDryRun(t *testing.T) {
	root := makeKitWithDrift(t)

	var stdout, stderr bytes.Buffer
	env := IO{
		In:          bytes.NewReader(nil),
		Out:         &stdout,
		Err:         &stderr,
		Now:         time.Now,
		Interactive: true,
	}

	code := Run([]string{"check", "--dir", root, "--fix", "--dry-run"}, env)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Dry run: 2 file(s) would be repaired. No changes written.") {
		t.Fatalf("stdout missing expected dry run message:\n%s", stdout.String())
	}

	// Verify files on disk were NOT modified
	gemini := readCheckFixFile(t, root, "GEMINI.md")
	if !strings.Contains(gemini, "OLD RULES") {
		t.Fatalf("dry run unexpectedly modified GEMINI.md")
	}
}

func TestCheckFixYes(t *testing.T) {
	root := makeKitWithDrift(t)

	var stdout, stderr bytes.Buffer
	fixedTime := time.Date(2026, 11, 2, 16, 30, 0, 0, time.UTC)
	env := IO{
		In:          bytes.NewReader(nil),
		Out:         &stdout,
		Err:         &stderr,
		Now:         func() time.Time { return fixedTime },
		Interactive: true,
	}

	code := Run([]string{"check", "--dir", root, "--fix", "--yes"}, env)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Repaired 2 file(s). Original(s) backed up in .sdd/backup/20261102-163000/") {
		t.Fatalf("stdout missing success message:\n%s", stdout.String())
	}

	// Verify GEMINI.md modified
	gemini := readCheckFixFile(t, root, "GEMINI.md")
	if !strings.Contains(gemini, "CONTRACT RULES v2") {
		t.Fatalf("GEMINI.md not updated with contract:\n%s", gemini)
	}
	if !strings.Contains(gemini, "Notes") {
		t.Fatalf("GEMINI.md lost notes outside block:\n%s", gemini)
	}

	// Verify backup
	backup := readCheckFixFile(t, root, ".sdd/backup/20261102-163000/GEMINI.md")
	if !strings.Contains(backup, "OLD RULES") {
		t.Fatalf("backup does not contain original content:\n%s", backup)
	}
}

func TestCheckFixPromptAbort(t *testing.T) {
	root := makeKitWithDrift(t)

	var stdout, stderr bytes.Buffer
	env := IO{
		In:          strings.NewReader("n\n"),
		Out:         &stdout,
		Err:         &stderr,
		Now:         time.Now,
		Interactive: true,
	}

	code := Run([]string{"check", "--dir", root, "--fix"}, env)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Repairs aborted. No changes written.") {
		t.Fatalf("stdout missing abort message:\n%s", stdout.String())
	}

	// Verify file was NOT modified
	gemini := readCheckFixFile(t, root, "GEMINI.md")
	if !strings.Contains(gemini, "OLD RULES") {
		t.Fatalf("aborted run modified GEMINI.md")
	}
}

func TestCheckFixPerFilePrompt(t *testing.T) {
	root := makeKitWithDrift(t)

	// User selects 'p' (per-file), then 'y' for CLAUDE.md, 'n' for GEMINI.md
	// Note: Patches order in Plan: agentFiles order is AGENTS.md, CLAUDE.md, GEMINI.md.
	// So CLAUDE.md is prompted first, then GEMINI.md.
	var stdout, stderr bytes.Buffer
	env := IO{
		In:          strings.NewReader("p\ny\nn\n"),
		Out:         &stdout,
		Err:         &stderr,
		Now:         time.Now,
		Interactive: true,
	}

	code := Run([]string{"check", "--dir", root, "--fix"}, env)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Repaired 1 file(s).") {
		t.Fatalf("stdout missing repaired 1 file message:\n%s", stdout.String())
	}

	// CLAUDE.md should be updated
	claude := readCheckFixFile(t, root, "CLAUDE.md")
	if !strings.Contains(claude, "CONTRACT RULES v2") {
		t.Fatalf("CLAUDE.md should have been repaired:\n%s", claude)
	}

	// GEMINI.md should NOT be updated
	gemini := readCheckFixFile(t, root, "GEMINI.md")
	if !strings.Contains(gemini, "OLD RULES") {
		t.Fatalf("GEMINI.md should NOT have been repaired")
	}
}

func TestCheckFixFileFlag(t *testing.T) {
	root := makeKitWithDrift(t)

	var stdout, stderr bytes.Buffer
	env := IO{
		In:          bytes.NewReader(nil),
		Out:         &stdout,
		Err:         &stderr,
		Now:         time.Now,
		Interactive: true,
	}

	code := Run([]string{"check", "--dir", root, "--fix", "--yes", "--file", "GEMINI.md"}, env)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Repaired 1 file(s).") {
		t.Fatalf("stdout missing repaired 1 file message:\n%s", stdout.String())
	}

	// GEMINI.md should be repaired
	gemini := readCheckFixFile(t, root, "GEMINI.md")
	if !strings.Contains(gemini, "CONTRACT RULES v2") {
		t.Fatalf("GEMINI.md should have been repaired")
	}

	// CLAUDE.md should NOT be repaired
	claude := readCheckFixFile(t, root, "CLAUDE.md")
	if !strings.Contains(claude, "OLD RULES") {
		t.Fatalf("CLAUDE.md should NOT have been repaired")
	}
}

func TestCheckFixDisplaysSuppressedBadge(t *testing.T) {
	root := makeKitWithDrift(t)

	// Add suppression for GEMINI.md
	suppJSON := `{
  "schema": 1,
  "suppressions": [
    {
      "file": "GEMINI.md",
      "rule": "contract-parity",
      "reason": "waived drift for spike",
      "date": "2026-11-02"
    }
  ]
}`
	writeCheckFixFile(t, root, ".sdd/suppressions.json", suppJSON)

	var stdout, stderr bytes.Buffer
	env := IO{
		In:          bytes.NewReader(nil),
		Out:         &stdout,
		Err:         &stderr,
		Now:         time.Now,
		Interactive: true,
	}

	// Run dry-run to inspect proposed repairs
	code := Run([]string{"check", "--dir", root, "--fix", "--dry-run"}, env)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "GEMINI.md") || !strings.Contains(out, "[SUPPRESSED: waived drift for spike]") {
		t.Fatalf("expected [SUPPRESSED: waived drift for spike] in output:\n%s", out)
	}
}

func TestCheckFixInteractiveSuppressOption(t *testing.T) {
	root := makeKitWithDrift(t)

	// Interactive input:
	// For GEMINI.md only: answer 's' to suppress, then reason 'spike reason'
	input := "s\nspike reason\n"
	var stdout, stderr bytes.Buffer
	env := IO{
		In:          bytes.NewBufferString(input),
		Out:         &stdout,
		Err:         &stderr,
		Now:         time.Now,
		Interactive: true,
	}

	code := Run([]string{"check", "--dir", root, "--fix", "--file", "GEMINI.md"}, env)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Recorded waiver for GEMINI.md (contract-parity)") {
		t.Fatalf("expected waiver confirmation in output:\n%s", stdout.String())
	}

	// File was untouched
	gemini := readCheckFixFile(t, root, "GEMINI.md")
	if !strings.Contains(gemini, "OLD RULES") {
		t.Fatalf("expected GEMINI.md to remain untouched, got:\n%s", gemini)
	}

	// Verify .sdd/suppressions.json has the waiver
	suppContent := readCheckFixFile(t, root, ".sdd/suppressions.json")
	if !strings.Contains(suppContent, "spike reason") || !strings.Contains(suppContent, "GEMINI.md") {
		t.Fatalf("expected waiver recorded in suppressions.json:\n%s", suppContent)
	}
}

func TestCheckFixClearsSuppressionOnRepair(t *testing.T) {
	root := makeKitWithDrift(t)

	// Add existing suppression for GEMINI.md
	suppJSON := `{
  "schema": 1,
  "suppressions": [
    {
      "file": "GEMINI.md",
      "rule": "contract-parity",
      "reason": "previously waived",
      "date": "2026-11-02"
    }
  ]
}`
	writeCheckFixFile(t, root, ".sdd/suppressions.json", suppJSON)

	var stdout, stderr bytes.Buffer
	env := IO{
		In:          bytes.NewReader(nil),
		Out:         &stdout,
		Err:         &stderr,
		Now:         time.Now,
		Interactive: true,
	}

	// Repair GEMINI.md with --yes
	code := Run([]string{"check", "--dir", root, "--fix", "--yes", "--file", "GEMINI.md"}, env)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", code, stderr.String())
	}

	// GEMINI.md should be repaired
	gemini := readCheckFixFile(t, root, "GEMINI.md")
	if !strings.Contains(gemini, "CONTRACT RULES v2") {
		t.Fatalf("GEMINI.md should have been repaired")
	}

	// .sdd/suppressions.json should no longer contain GEMINI.md
	suppContent := readCheckFixFile(t, root, ".sdd/suppressions.json")
	if strings.Contains(suppContent, "GEMINI.md") {
		t.Fatalf("expected suppression to be cleared upon repair, got:\n%s", suppContent)
	}
}

func TestCheckStaleSuppressionDetectionAndPrune(t *testing.T) {
	root := t.TempDir()
	exec(t, "", false, "", "init", "--yes", "--dir", root, "--name", "stalepruning")

	// Stale waiver for already identical contract
	suppJSON := `{
  "schema": 1,
  "suppressions": [
    {
      "file": "GEMINI.md",
      "rule": "contract-parity",
      "reason": "obsolete waiver",
      "date": "2026-11-02"
    }
  ]
}`
	writeCheckFixFile(t, root, ".sdd/suppressions.json", suppJSON)

	// 1. Standard check should report an INFO note
	var checkOut, checkErr bytes.Buffer
	envCheck := IO{
		In:          bytes.NewReader(nil),
		Out:         &checkOut,
		Err:         &checkErr,
		Now:         time.Now,
		Interactive: true,
	}
	code := Run([]string{"check", "--dir", root}, envCheck)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", code, checkErr.String())
	}
	if !strings.Contains(checkOut.String(), "stale waiver for GEMINI.md (contract-parity) is no longer triggered") {
		t.Fatalf("expected stale waiver INFO in check output:\n%s", checkOut.String())
	}

	// 2. check --fix --yes should detect and prune stale waiver
	var fixOut, fixErr bytes.Buffer
	envFix := IO{
		In:          bytes.NewReader(nil),
		Out:         &fixOut,
		Err:         &fixErr,
		Now:         time.Now,
		Interactive: true,
	}
	codeFix := Run([]string{"check", "--dir", root, "--fix", "--yes"}, envFix)
	if codeFix != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", codeFix, fixErr.String())
	}
	if !strings.Contains(fixOut.String(), "Pruned 1 stale waiver(s)") {
		t.Fatalf("expected prune confirmation in output:\n%s", fixOut.String())
	}

	// suppressions.json should now have 0 entries
	suppAfter := readCheckFixFile(t, root, ".sdd/suppressions.json")
	if strings.Contains(suppAfter, "GEMINI.md") {
		t.Fatalf("expected GEMINI.md to be pruned, got:\n%s", suppAfter)
	}
}
